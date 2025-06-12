package scheduler

import (
	r "github.com/robfig/cron/v3"
	"tinvest/pkg/closer"
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
	_, err := c.sh.AddFunc(schedule, cmd)

	if err != nil {
		return err
	}

	return nil
}
