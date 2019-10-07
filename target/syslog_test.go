// +build !windows,!nacl,!plan9

package target_test

import (
	"fmt"
	"log/syslog"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/target"
	"github.com/wiggin77/logr/test"
)

func ExampleSyslog() {
	lgr := &logr.Logr{}
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &format.Plain{Delim: " | "}
	params := &target.SyslogParams{Network: "", Raddr: "", Priority: syslog.LOG_WARNING | syslog.LOG_DAEMON, Tag: "logrtest"}
	t, err := target.NewSyslogTarget(filter, formatter, params, 1000)
	if err != nil {
		panic(err)
	}
	lgr.AddTarget(t)

	logger := lgr.NewLogger().WithField("name", "wiggin")

	logger.Errorf("the erroneous data is %s", test.StringRnd(10))
	logger.Warnf("strange data: %s", test.StringRnd(5))
	logger.Debug("XXX")
	logger.Trace("XXX")

	err = lgr.Shutdown()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func TestSyslogPlain(t *testing.T) {
	plain := &format.Plain{Delim: " | ", DisableTimestamp: true}
	syslogger(t, plain)
}

func syslogger(t *testing.T, formatter logr.Formatter) {
	lgr := logr.Logr{}

	lgr.OnLoggerError = func(err error) {
		t.Error(err)
	}

	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Panic}
	params := &target.SyslogParams{Network: "", Raddr: "", Priority: syslog.LOG_WARNING | syslog.LOG_DAEMON, Tag: "logrtest"}
	target, err := target.NewSyslogTarget(filter, formatter, params, 1000)
	if err != nil {
		t.Error(err)
	}
	lgr.AddTarget(target)

	wg := sync.WaitGroup{}
	var id int32

	runner := func(loops int) {
		defer wg.Done()
		tid := atomic.AddInt32(&id, 1)
		logger := lgr.NewLogger().WithFields(logr.Fields{"id": tid, "rnd": rand.Intn(100)})

		for i := 0; i < loops; i++ {
			logger.Debug("XXX")
			logger.Trace("XXX")
			logger.Errorf("count:%d -- the erroneous data is %s", i, test.StringRnd(10))
			logger.Warnf("count:%d -- strange data: %s", i, test.StringRnd(5))
			runtime.Gosched()
		}
	}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go runner(5)
	}
	wg.Wait()

	err = lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}
}
