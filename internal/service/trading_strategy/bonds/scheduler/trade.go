package scheduler

import (
	"context"
	"time"
	"tinvest/internal/service/trading_strategy/bonds"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service bonds.Bonds
}

func (s *schedulerService) Trade(ctx context.Context) error {
	jobTicker := time.NewTicker(time.Hour)
	defer func() {
		jobTicker.Stop()
	}()
	err := s.sh.AddJob("0 */10 * * *", func() {
		logger.InfoContext(ctx, "Воркер Golden RSI начал работу")
		err := s.service.Trade(ctx)

		if err != nil {
			logger.ErrorContext(ctx, "Ошибка в ходе работы job", err)
		}
	})
	s.sh.Start()
	if err != nil {
		return err
	}
	defer s.sh.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-jobTicker.C:
			logger.InfoContext(ctx, "Worker Bonds Report is running")
		default:
			time.Sleep(10 * time.Second)
		}
	}
}

func NewScheduler(service bonds.Bonds) bonds.Bonds {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}
