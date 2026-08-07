package scheduler

import (
	"context"
	"fmt"
	"runtime/debug"

	"tinvest/pkg/closer"
	"tinvest/pkg/logger"

	r "github.com/robfig/cron/v3"
)

type job func()

type Scheduler interface {
	Start()
	Stop()
	AddJob(schedule string, cmd job) error
}

type cron struct {
	sh *r.Cron
}

func NewScheduler() Scheduler {
	return &cron{
		sh: r.New(),
	}
}

func (c *cron) Start() {
	c.sh.Start()
	closer.Add(func() error {
		c.sh.Stop()

		return nil
	})
}

func (c *cron) Stop() {
	c.sh.Stop()
}

func (c *cron) AddJob(schedule string, cmd job) error {
	_, err := c.sh.AddFunc(schedule, withRecover(schedule, cmd))

	if err != nil {
		return err
	}

	return nil
}

// withRecover keeps one panicking job from taking down the process. cron runs every job in
// its own goroutine, and an unrecovered panic there is fatal for the whole binary — so a nil
// map in the news digest would kill the live trading workers that are carrying open
// positions, and they would stay dead until something restarts the container. Recovering
// per job costs one skipped run: the schedule keeps firing and the next tick retries.
func withRecover(schedule string, cmd job) job {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(context.Background(), fmt.Sprintf("scheduler: job %q panicked: %v\n%s", schedule, r, debug.Stack()))
			}
		}()
		cmd()
	}
}
