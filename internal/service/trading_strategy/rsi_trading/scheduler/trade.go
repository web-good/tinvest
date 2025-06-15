package scheduler

import (
	"context"
	"time"
	"tinvest/internal/service/trading_strategy/rsi_trading"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service rsi_trading.RsiTrading
}

func NewSchedulerService(service rsi_trading.RsiTrading) rsi_trading.RsiTrading {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}

func (s *schedulerService) Trade(ctx context.Context, interval int) error {
	jobTicker := time.NewTicker(time.Hour)
	defer s.sh.Stop()
	defer jobTicker.Stop()
	err := s.sh.AddJob("5 * * * *", func() {
		logger.InfoContext(ctx, "Воркер MacD Rsi начал работу")
		err := s.service.Trade(ctx, interval)

		if err != nil {
			logger.ErrorContext(ctx, "Ошибка в ходе работы job", err)
		}
	})
	s.sh.Start()

	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-jobTicker.C:
			logger.InfoContext(ctx, "Воркер MacD Rsi успешно работает")
		default:
			time.Sleep(10 * time.Second)
		}
	}
}
