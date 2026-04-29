package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseUser(t *testing.T) {
	parser, _ := NewCronjobUserParser("/user/crontab", "user")

	entries := parser.Parse(strings.NewReader("* * * * * command arg1 arg2"))

	if assert.Len(t, entries, 1) {
		entry := entries[0]
		assert.Equal(t, entry.Spec, "* * * * *")
		assert.Equal(t, entry.User, "user")
		assert.Equal(t, entry.Command, "command arg1 arg2")
		assert.Len(t, entry.Env, 0)
		assert.Equal(t, entry.Shell, "sh")
		assert.Equal(t, entry.CrontabPath, "/user/crontab")
		assert.Equal(t, time.Duration(0), entry.Timeout)
		assert.Equal(t, NoLock, entry.LockMode)
	}
}

func TestParseUserTimeoutAndLockMode(t *testing.T) {
	parser, _ := NewCronjobUserParser("/user/crontab", "user")

	entries := parser.Parse(strings.NewReader(`
		GOCROND_TIMEOUT=90s
		GOCROND_LOCK=skip
		* * * * * command-skip
		GOCROND_LOCK=queue
		* * * * * command-queue
	`))

	if assert.Len(t, entries, 2) {
		assert.Equal(t, "command-skip", entries[0].Command)
		assert.Equal(t, 90*time.Second, entries[0].Timeout)
		assert.Equal(t, LockSkip, entries[0].LockMode)

		assert.Equal(t, "command-queue", entries[1].Command)
		assert.Equal(t, 90*time.Second, entries[1].Timeout)
		assert.Equal(t, LockQueue, entries[1].LockMode)
	}
}

func TestParseUserExpressions(t *testing.T) {
	parser, _ := NewCronjobUserParser("/user/crontab", "user")

	entries := parser.Parse(strings.NewReader(`
		* * * * * 		command
		*/5 * * * *  	command
		15,45 * * * * 	command
		* 1,2 * * *   	command
		0 22 * * 1-5   	command
		* 1-5 * * *     command
	`))

	if assert.Len(t, entries, 6) {
		assert.Equal(t, entries[0].Spec, "* * * * *")
		assert.Equal(t, entries[1].Spec, "*/5 * * * *")
		assert.Equal(t, entries[2].Spec, "15,45 * * * *")
		assert.Equal(t, entries[3].Spec, "* 1,2 * * *")
		assert.Equal(t, entries[4].Spec, "0 22 * * 1-5")
		assert.Equal(t, entries[5].Spec, "* 1-5 * * *")
	}
}

func TestParseUserEvery(t *testing.T) {
	parser, _ := NewCronjobUserParser("/user/crontab", "user")

	entries := parser.Parse(strings.NewReader(`
		* * * * *		command
		@every 1m		command
	`))

	if assert.Len(t, entries, 2) {
		assert.Equal(t, entries[0].Spec, "* * * * *")
		assert.Equal(t, entries[1].Spec, "@every 1m")
	}
}

func TestParseSystem(t *testing.T) {
	parser, _ := NewCronjobSystemParser("/user/crontab")

	entries := parser.Parse(strings.NewReader("* * * * * username command arg1 arg2"))

	if assert.Len(t, entries, 1) {
		entry := entries[0]
		assert.Equal(t, entry.Spec, "* * * * *")
		assert.Equal(t, entry.User, "username")
		assert.Equal(t, entry.Command, "command arg1 arg2")
		assert.Len(t, entry.Env, 0)
		assert.Equal(t, entry.Shell, "sh")
		assert.Equal(t, entry.CrontabPath, "/user/crontab")
		assert.Equal(t, time.Duration(0), entry.Timeout)
		assert.Equal(t, NoLock, entry.LockMode)
	}
}
