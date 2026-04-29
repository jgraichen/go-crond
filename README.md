# go-crond

[![GitHub release](https://img.shields.io/github/release/jgraichen/go-crond.svg)](https://github.com/jgraichen/go-crond/releases)
[![Licensed GPL-2.0](https://img.shields.io/github/license/jgraichen/go-crond.svg)](https://github.com/jgraichen/go-crond/blob/master/LICENSE)

> A cron daemon written in golang
>
> Inspired by <https://github.com/anarcher/go-cron>, using <https://godoc.org/github.com/robfig/cron>.

Fork of [webdevops/go-crond](https://github.com/webdevops/go-crond) to maintain up-to-date binaries and container images for personal use.

## Container images

Only released to [`ghcr.io/jgraichen/go-crond`](https://github.com/jgraichen/go-crond/pkgs/container/go-crond).

### Usage

Use the container image to copy the correct binary into your own application, and run `go-crond` as an entry point, as a s6 service, or from your own entry point script.

```Dockerfile
FROM ghcr.io/jgraichen/go-crond:24.1.0 AS go-crond

FROM your-base-image
COPY --from=go-crond /usr/bin/go-crond /usr/bin/go-crond

# ...
```

## Features

- system crontab (with username inside)
- user crontabs (without username inside)
- run-parts support
- Logging to STDOUT and STDERR (instead of sending mails)
- Keep current environment (e.g. for usage in Docker containers)
- Supports Linux, macOS, ARM/ARM64 (Raspberry Pi and others)
- Timeouts
- Locking (skip or queue)

## CLI Usage

```console
Usage:
  go-crond [OPTIONS] [Crontabs...]

Application Options:
  -V, --version               show version and exit
      --dumpversion           show only version number and exit
  -h, --help                  show this help message
      --default-user=         Default user (default: root)
      --include=              Include files in directory as system crontabs (with user)
      --auto                  Enable automatic system crontab detection
      --run-parts=            Execute files in directory with custom spec (like run-parts; spec-units:ns,us,s,m,h;
                              format:time-spec:path; eg:10s,1m,1h30m)
      --run-parts-1min=       Execute files in directory every beginning minute (like run-parts)
      --run-parts-15min=      Execute files in directory every beginning 15 minutes (like run-parts)
      --run-parts-hourly=     Execute files in directory every beginning hour (like run-parts)
      --run-parts-daily=      Execute files in directory every beginning day (like run-parts)
      --run-parts-weekly=     Execute files in directory every beginning week (like run-parts)
      --run-parts-monthly=    Execute files in directory every beginning month (like run-parts)
      --allow-unprivileged    Allow daemon to run as non root (unprivileged) user
      --working-directory=    Set the working directory for crontab commands (default: /)
  -v, --verbose               verbose mode [$VERBOSE]
      --log.json              Switch log output to json format [$LOG_JSON]
      --server.bind=          Server address, eg. ':8080' (/healthz and /metrics for prometheus) [$SERVER_BIND]
      --server.timeout.read=  Server read timeout (default: 5s) [$SERVER_TIMEOUT_READ]
      --server.timeout.write= Server write timeout (default: 10s) [$SERVER_TIMEOUT_WRITE]
      --server.metrics        Enable prometheus metrics (do not use senstive informations in commands -> use environment
                              variables or files for storing these informations) [$SERVER_METRICS]

Help Options:
  -h, --help                  Show this help message

Arguments:
  Crontabs:                   path to crontab files
```

Crontab files can be added as arguments or automatic included by using eg. `--include=crond-path/`

### Examples

Run crond with a system crontab:

```console
go-crond examples/crontab
```

Run crond with user crontabs (without user in it) under specific users:

```console
go-crond \
    root:examples/crontab-root \
    guest:examples/crontab-guest
```

Run crond with auto include of /etc/cron.d and script execution of hourly, weekly, daily and monthly:

```console
go-crond \
    --include=/etc/cron.d \
    --run-parts-hourly=/etc/cron.hourly \
    --run-parts-weekly=/etc/cron.weekly \
    --run-parts-daily=/etc/cron.daily \
    --run-parts-monthly=/etc/cron.monthly
```

Run crond with run-parts with custom time spec:

```console
go-crond \
    --run-parts=1m:/etc/cron.minute \
    --run-parts=15m:/etc/cron.15min
```

Run crond with run-parts with custom time spec and different user:

```console
go-crond \
    --run-parts=1m:application:/etc/cron.minute \
    --run-parts=15m:admin:/etc/cron.15min
```

## Timeout and Locking

`go-crond` supports timeouts and locking for cronjobs.

You can set a timeout for each cronjob by setting the `GOCROND_TIMEOUT` environment variable to a duration string (e.g. `30s`, `1m`, `1h`).

Two lock modes are supported: `skip` and `queue`. In `skip` mode, if a
cronjob is still running when it's time to run it again, the new
execution will be skipped. In `queue` mode, the new execution will be
queued and executed immediately after the current execution finishes.

```cronjob
GOCROND_TIMEOUT=30s
GOCROND_LOCK=skip

* *   *   *   *  sleep 5 && id >> /tmp/test-1
```

## Metrics

`go-crond` exposes [Prometheus][] metrics on `:8080/metrics` if enabled.

| Metric                      | Description                                     |
| :-------------------------- | :---------------------------------------------- |
| `gocrond_task_info`         | List of all cronjobs                            |
| `gocrond_task_run_count`    | Counter for each executed task                  |
| `gocrond_task_run_result`   | Last status (0=failed, 1=success) for each task |
| `gocrond_task_run_time`     | Last exec time (unix timestamp) for each task   |
| `gocrond_task_run_duration` | Duration of last exec                           |

[Prometheus]: https://prometheus.io/
