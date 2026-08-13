// Package scheduler runs the live rsi_pullback service on a cron schedule.
package scheduler

import (
	"context"
	"time"

	"tinvest/internal/service/trading_strategy/rsi_pullback/live"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
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

// Announce проходит насквозь: приложение держит раннер через эту обёртку, а стартовое
// сообщение обязано уйти из той же точки, где решается, поднимать ли воркер.
func (s *schedulerService) Announce() { s.service.Announce() }

func (s *schedulerService) Run(ctx context.Context, in dto.Run) error {
	jobTicker := time.NewTicker(time.Hour)
	defer jobTicker.Stop()

	err := s.sh.AddJob(in.Scheduler, func() {
		logger.InfoContext(ctx, "Воркер RSI Pullback начал работу")
		if err := s.service.Run(ctx, in); err != nil {
			logger.ErrorContext(ctx, "Ошибка в ходе работы job RSI Pullback", err)
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
			logger.InfoContext(ctx, "Worker RSI Pullback is running")
		}
	}
}
