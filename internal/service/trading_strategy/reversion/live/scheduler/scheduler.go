// Package scheduler runs the live reversion service on a cron schedule.
package scheduler

import (
	"context"
	"time"

	"tinvest/internal/service/trading_strategy/reversion/live"
	"tinvest/internal/service/trading_strategy/reversion/live/dto"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service live.Service
}

// NewSchedulerService wraps a live.Service so Run registers a cron job and blocks.
func NewSchedulerService(service live.Service) live.Service {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}

func (s *schedulerService) Run(ctx context.Context, in dto.Run) error {
	jobTicker := time.NewTicker(time.Hour)
	defer jobTicker.Stop()

	err := s.sh.AddJob(in.Scheduler, func() {
		logger.InfoContext(ctx, "Воркер Reversion начал работу")
		if err := s.service.Run(ctx, in); err != nil {
			logger.ErrorContext(ctx, "Ошибка в ходе работы job Reversion", err)
		}
	})
	if err != nil {
		return err
	}

	s.sh.Start()
	defer s.sh.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-jobTicker.C:
			logger.InfoContext(ctx, "Worker Reversion is running")
		}
	}
}
