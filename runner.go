package main

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/alaingilbert/cron"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type Runner struct {
	cron     *cron.Cron
	cronjobs []*CrontabEntry
}

func NewRunner() *Runner {
	c := cron.New().
		WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)).
		WithLogger(cron.DiscardLogger).
		Build()

	r := &Runner{
		cron:     c,
		cronjobs: make([]*CrontabEntry, 0),
	}
	return r
}

// Add crontab entry
func (r *Runner) Add(cronjob CrontabEntry) error {
	eid, err := r.cron.AddJob(cronjob.Spec, r.cmdFunc(&cronjob, func(execCmd *exec.Cmd) bool {
		// before exec callback
		log.WithFields(LogCronjobToFields(cronjob)).Infof("executing")
		return true
	}))

	if err != nil {
		prometheusMetricTask.With(r.cronjobToPrometheusLabels(cronjob)).Set(0)
		log.WithFields(LogCronjobToFields(cronjob)).Errorf("cronjob failed adding:%v", err)
	} else {
		cronjob.SetEntryId(eid)
		r.cronjobs = append(r.cronjobs, &cronjob)
		prometheusMetricTask.With(r.cronjobToPrometheusLabels(cronjob)).Set(1)
		log.WithFields(LogCronjobToFields(cronjob)).Infof("cronjob added")
	}

	return err
}

// Add crontab entry with user
func (r *Runner) AddWithUser(cronjob CrontabEntry) error {
	eid, err := r.cron.AddJob(cronjob.Spec, r.cmdFunc(&cronjob, func(execCmd *exec.Cmd) bool {
		// before exec callback
		log.WithFields(LogCronjobToFields(cronjob)).Debugf("executing")

		// lookup username
		u, err := user.Lookup(cronjob.User)
		if err != nil {
			log.WithFields(LogCronjobToFields(cronjob)).Errorf("user lookup failed: %v", err)
			return false
		}

		// convert userid to int
		userId, err := strconv.ParseUint(u.Uid, 10, 32)
		if err != nil {
			log.WithFields(LogCronjobToFields(cronjob)).Errorf("Cannot convert user to id:%v", err)
			return false
		}

		// convert groupid to int
		groupId, err := strconv.ParseUint(u.Gid, 10, 32)
		if err != nil {
			log.WithFields(LogCronjobToFields(cronjob)).Errorf("Cannot convert group to id:%v", err)
			return false
		}

		// add process credentials
		execCmd.SysProcAttr = &syscall.SysProcAttr{}
		execCmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(userId), Gid: uint32(groupId)}
		return true
	}))

	if err != nil {
		prometheusMetricTask.With(r.cronjobToPrometheusLabels(cronjob)).Set(0)
		log.WithFields(LogCronjobToFields(cronjob)).Errorf("cronjob failed adding: %v", err)
	} else {
		cronjob.SetEntryId(eid)
		r.cronjobs = append(r.cronjobs, &cronjob)
		prometheusMetricTask.With(r.cronjobToPrometheusLabels(cronjob)).Set(1)
		log.WithFields(LogCronjobToFields(cronjob)).Infof("cronjob added")
	}

	return err
}

// Return number of jobs
func (r *Runner) Len() int {
	return len(r.cron.Entries())
}

// Start runner
func (r *Runner) Start() {
	log.Infof("start runner with %d jobs\n", r.Len())
	r.cron.Start()
	r.initAllCronEntryMetrics()
}

// Stop runner
func (r *Runner) Stop() {
	log.Infof("stop runner")
	ctx := r.cron.Stop()
	<-ctx
	log.Infof("stopped runner")
}

// Execute crontab command
func (r *Runner) cmdFunc(cronjob *CrontabEntry, cmdCallback func(*exec.Cmd) bool) cron.IntoJob {
	cmdFunc := func(ctx context.Context) {
		// fall back to normal shell if not specified
		taskShell := cronjob.Shell
		if taskShell == "" {
			taskShell = DEFAULT_SHELL
		}

		start := time.Now()

		// Init command
		execCmd := exec.CommandContext(ctx, taskShell, "-c", cronjob.Command)
		execCmd.WaitDelay = 100 * time.Millisecond

		// add custom env to cronjob
		if len(cronjob.Env) >= 1 {
			execCmd.Env = append(os.Environ(), cronjob.Env...)
		}

		// exec custom callback
		if cmdCallback(execCmd) {

			// exec job
			cmdOut, err := execCmd.CombinedOutput()

			elapsed := time.Since(start)

			cronjobMetricCommonLables := r.cronjobToPrometheusLabels(*cronjob)
			prometheusMetricTaskRunDuration.With(cronjobMetricCommonLables).Set(elapsed.Seconds())
			prometheusMetricTaskRunTime.With(cronjobMetricCommonLables).SetToCurrentTime()

			logFields := LogCronjobToFields(*cronjob)
			logFields["elapsed_s"] = elapsed.Seconds()
			if execCmd.ProcessState != nil {
				logFields["exitCode"] = execCmd.ProcessState.ExitCode()
			}

			if err != nil {
				prometheusMetricTaskRunCount.With(r.cronjobToPrometheusLabels(*cronjob, prometheus.Labels{"result": "error"})).Inc()
				prometheusMetricTaskRunResult.With(cronjobMetricCommonLables).Set(0)
				logFields["result"] = "error"
				logFields["error"] = err.Error()
				log.WithFields(logFields).Error("failed: " + err.Error())
			} else {
				prometheusMetricTaskRunCount.With(r.cronjobToPrometheusLabels(*cronjob, prometheus.Labels{"result": "success"})).Inc()
				prometheusMetricTaskRunResult.With(cronjobMetricCommonLables).Set(1)
				logFields["result"] = "success"
				log.WithFields(logFields).Info("finished")
			}

			r.updateCronEntryMetrics(cronjob)
			if len(cmdOut) > 0 {
				log.Debugln(string(cmdOut))
			}
		}
	}

	var job cron.IntoJob = cmdFunc

	if cronjob.Timeout > 0 {
		job = cron.WithTimeout(cronjob.Timeout, job)
	}

	switch cronjob.LockMode {
	case LockSkip:
		job = skipIfStillRunning(job, cronjob)
	case LockQueue:
		job = delayIfStillRunning(job, cronjob)
	}

	return job
}

func (r *Runner) cronjobToPrometheusLabels(cronjob CrontabEntry, additionalLabels ...prometheus.Labels) (labels prometheus.Labels) {
	labels = prometheus.Labels{
		"cronSpec":    cronjob.Spec,
		"cronUser":    cronjob.User,
		"cronCommand": cronjob.Command,
	}
	for _, additionalLabelValue := range additionalLabels {
		for labelName, labelValue := range additionalLabelValue {
			labels[labelName] = labelValue
		}
	}
	return
}

func (r *Runner) updateCronEntryMetrics(cronjob *CrontabEntry) {
	cronjobMetricCommonLables := r.cronjobToPrometheusLabels(*cronjob)
	entry, _ := r.cron.Entry(cronjob.EntryId)

	if entry.Next.IsZero() {
		prometheusMetricTaskRunNextTs.With(cronjobMetricCommonLables).Set(0)
	} else {
		prometheusMetricTaskRunNextTs.With(cronjobMetricCommonLables).Set(float64(entry.Next.Unix()))
	}

	if entry.Prev.IsZero() {
		prometheusMetricTaskRunPrevTs.With(cronjobMetricCommonLables).Set(0)
	} else {
		prometheusMetricTaskRunPrevTs.With(cronjobMetricCommonLables).Set(float64(entry.Prev.Unix()))
	}
}

func (r *Runner) initAllCronEntryMetrics() {
	for _, cronjob := range r.cronjobs {
		r.updateCronEntryMetrics(cronjob)
	}
}

func skipIfStillRunning(job cron.IntoJob, cronjob *CrontabEntry) cron.IntoJob {
	var running atomic.Bool

	logger := log.WithFields(LogCronjobToFields(*cronjob))

	return cron.FuncJob(func(ctx context.Context, c *cron.Cron, r cron.JobRun) (err error) {
		if running.CompareAndSwap(false, true) {
			defer running.Store(false)
			err = cron.J(job).Run(ctx, c, r)
		} else {
			logger.WithField("action", "skip").Warn("previous job still running, skipping")
			return cron.ErrJobAlreadyRunning
		}
		return
	})
}

func delayIfStillRunning(job cron.IntoJob, cronjob *CrontabEntry) cron.IntoJob {
	var mu sync.Mutex

	logger := log.WithFields(LogCronjobToFields(*cronjob))

	return func(ctx context.Context, c *cron.Cron, r cron.JobRun) (err error) {
		start := time.Now()

		logger.
			WithField("action", "delay").
			Warn("previous job still running, delaying")

		mu.Lock()
		defer mu.Unlock()
		if delay := time.Since(start); delay > time.Second {
			logger.
				WithField("delay", delay.String()).
				Info("executing queued job")
		}
		return cron.J(job).Run(ctx, c, r)
	}
}
