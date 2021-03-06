package logr_test

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mattermost/logr"
	"github.com/mattermost/logr/format"
	"github.com/mattermost/logr/target"
	"github.com/mattermost/logr/test"
)

var (
	LoginLevel  = logr.Level{ID: 100, Name: "login ", Stacktrace: false}
	LogoutLevel = logr.Level{ID: 101, Name: "logout", Stacktrace: false}
	BadLevel    = logr.Level{ID: logr.MaxLevelID + 1, Name: "invalid", Stacktrace: false}
)

func TestCustomLevel(t *testing.T) {
	lgr := &logr.Logr{}
	buf := &test.Buffer{}

	// create a custom filter with custom levels.
	filter := &logr.CustomFilter{}
	filter.Add(LoginLevel, LogoutLevel)

	formatter := &format.Plain{Delim: " | "}
	tgr := target.NewWriterTarget(filter, formatter, buf, 1000)
	err := lgr.AddTarget(tgr)
	if err != nil {
		t.Error(err)
	}

	logger := lgr.NewLogger().WithFields(logr.Fields{"user": "Bob", "role": "admin"})

	logger.Log(LoginLevel, "this item will get logged")
	logger.Log(logr.Error, "XXX - won't be logged as Error was not added to custom filter.")
	logger.Debug("XXX - won't be logged")
	logger.Log(LogoutLevel, "will get logged")

	err = lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}

	output := buf.String()
	fmt.Println(output)

	if !strings.Contains(output, "will get logged") {
		t.Error("missing levels")
	}

	if strings.Contains(output, "XXX") {
		t.Error("wrong level(s) output")
	}

}

func TestLevelIDTooLarge(t *testing.T) {
	lgr := &logr.Logr{}
	buf := &test.Buffer{}
	var count int32

	lgr.OnLoggerError = func(err error) {
		atomic.AddInt32(&count, 1)
	}

	// create a custom filter with custom level.
	filter := &logr.CustomFilter{}
	filter.Add(BadLevel)

	formatter := &format.Plain{Delim: " | "}
	tgr := target.NewWriterTarget(filter, formatter, buf, 1000)
	err := lgr.AddTarget(tgr)
	if err != nil {
		t.Error(err)
	}

	logger := lgr.NewLogger().WithFields(logr.Fields{"user": "Bob", "role": "admin"})

	logger.Log(BadLevel, "this item will trigger OnLoggerError")

	err = lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}

	if atomic.LoadInt32(&count) != 1 {
		t.Error("OnLoggerError should be called once")
	}
}
