package scheduler

import (
	"context"
	"time"

	"tinvest/internal/service/trading_strategy/scalping"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service scalping.Scalping
}

func (s *schedulerService) Trade(ctx context.Context, in dto.Trade) error {
	jobTicker := time.NewTicker(time.Hour)
	defer jobTicker.Stop()

	err := s.sh.AddJob(in.Scheduler, func() {
		logger.InfoContext(ctx, "Воркер Scalping начал работу")
		if err := s.service.Trade(ctx, in); err != nil {
			logger.ErrorContext(ctx, "Ошибка в ходе работы job", err)
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
			logger.InfoContext(ctx, "Worker Scalping is running")
		}
	}
}

func NewSchedulerService(service scalping.Scalping) scalping.Scalping {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}
