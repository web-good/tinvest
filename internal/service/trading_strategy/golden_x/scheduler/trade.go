package scheduler

import (
	"context"
	"time"
	"tinvest/internal/service/trading_strategy/golden_x"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh       scheduler.Scheduler
	service  golden_x.GoldenX
	tgClient telegram.Client
}

func (s *schedulerService) Trade(ctx context.Context, in dto.Trade) error {
	jobTicker := time.NewTicker(time.Hour)
	defer jobTicker.Stop()

	err := s.sh.AddJob(in.Scheduler, func() {
		logger.InfoContext(ctx, "Воркер Golden RSI начал работу")
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
			logger.InfoContext(ctx, "Worker Golden Share is running")
		}
	}
}

func NewSchedulerService(service golden_x.GoldenX, tgClient telegram.Client) golden_x.GoldenX {
	return &schedulerService{
		sh:       scheduler.NewScheduler(),
		tgClient: tgClient,
		service:  service,
	}
}
