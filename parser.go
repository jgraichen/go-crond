package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/alaingilbert/cron"
)

const (
	ENV_LINE = `^(\S+)=(\S+)\s*$`

	//                     ----spec------------------------------------    --user--  -cmd-
	CRONJOB_SYSTEM = `^\s*([^@\s]+\s+\S+\s+\S+\s+\S+\s+\S+|@every\s+\S+)\s+([^\s]+)\s+(.+)$`

	//                   ----spec------------------------------------   -cmd-
	CRONJOB_USER = `^\s*([^@\s]+\s+\S+\s+\S+\s+\S+\s+\S+|@every\s+\S+)\s+(.+)$`

	DEFAULT_SHELL = "sh"
)

var (
	envLineRegex       = regexp.MustCompile(ENV_LINE)
	cronjobSystemRegex = regexp.MustCompile(CRONJOB_SYSTEM)
	cronjobUserRegex   = regexp.MustCompile(CRONJOB_USER)
)

type LockMode int

const (
	NoLock    LockMode = iota // 0
	LockSkip                  // 1
	LockQueue                 // 2
)

func (l LockMode) String() string {
	switch l {
	case NoLock:
		return "no"
	case LockSkip:
		return "skip"
	case LockQueue:
		return "queue"
	default:
		return "unknown"
	}
}

type CrontabEntry struct {
	Spec        string
	User        string
	Command     string
	Env         []string
	Shell       string
	CrontabPath string
	EntryId     cron.EntryID
	Timeout     time.Duration
	LockMode    LockMode
}

type Parser struct {
	cronLineRegex   *regexp.Regexp
	cronjobUsername string
	path            string
}

// Create new crontab parser (user crontab without user specification)
func NewCronjobUserParser(path string, username string) (*Parser, error) {
	p := &Parser{
		cronLineRegex:   cronjobUserRegex,
		path:            path,
		cronjobUsername: username,
	}

	return p, nil
}

// Create new crontab parser (crontab with user specification)
func NewCronjobSystemParser(path string) (*Parser, error) {
	p := &Parser{
		cronLineRegex:   cronjobSystemRegex,
		path:            path,
		cronjobUsername: CRONTAB_TYPE_SYSTEM,
	}

	return p, nil
}

func (e *CrontabEntry) SetEntryId(eid cron.EntryID) {
	(*e).EntryId = eid
}

// Parse crontab
func (p *Parser) Parse(io io.Reader) []CrontabEntry {
	var (
		entries        []CrontabEntry
		crontabSpec    string
		crontabUser    string
		crontabCommand string
		environment    []string
	)

	shell := DEFAULT_SHELL
	timeout := time.Duration(0)
	lockMode := NoLock

	specCleanupRegexp := regexp.MustCompile(`\s+`)

	scanner := bufio.NewScanner(io)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// comment line
		if strings.HasPrefix(line, "#") {
			continue
		}

		// environment line
		if envLineRegex.MatchString(line) {
			m := envLineRegex.FindStringSubmatch(line)
			envName := strings.TrimSpace(m[1])
			envValue := strings.TrimSpace(m[2])

			if envName == "SHELL" {
				// custom shell for command
				shell = envValue
			} else {
				if envName == "GOCROND_TIMEOUT" {
					timeout, _ = time.ParseDuration(envValue)
				}

				if envName == "GOCROND_LOCK" {
					switch envValue {
					case "skip":
						lockMode = LockSkip
					case "queue":
						lockMode = LockQueue
					default:
						// error
					}
				}

				// normal environment variable
				environment = append(environment, fmt.Sprintf("%s=%s", envName, envValue))
			}
		}

		// cronjob line
		if p.cronLineRegex.MatchString(line) {
			m := p.cronLineRegex.FindStringSubmatch(line)

			if p.cronjobUsername == CRONTAB_TYPE_SYSTEM {
				crontabSpec = strings.TrimSpace(m[1])
				crontabUser = strings.TrimSpace(m[2])
				crontabCommand = strings.TrimSpace(m[3])
			} else {
				crontabSpec = strings.TrimSpace(m[1])
				crontabUser = p.cronjobUsername
				crontabCommand = strings.TrimSpace(m[2])
			}

			// shrink white spaces for better handling
			crontabSpec = specCleanupRegexp.ReplaceAllString(crontabSpec, " ")

			entries = append(
				entries,
				CrontabEntry{
					Spec:        crontabSpec,
					User:        crontabUser,
					Command:     crontabCommand,
					Env:         environment,
					Shell:       shell,
					CrontabPath: p.path,
					Timeout:     timeout,
					LockMode:    lockMode,
				},
			)
		}
	}

	return entries
}
