package scheduler

import (
	"context"
	"time"
	"tinvest/internal/service/trading_strategy/macd_rsi"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service macd_rsi.MacdRsi
}

func NewSchedulerService(service macd_rsi.MacdRsi) macd_rsi.MacdRsi {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}

func (s *schedulerService) Trade(ctx context.Context, in dto.Trade) error {
	jobTicker := time.NewTicker(time.Hour)
	defer s.sh.Stop()
	defer jobTicker.Stop()
	err := s.sh.AddJob(in.Scheduler, func() {
		logger.InfoContext(ctx, "Воркер MacD Rsi начал работу")
		err := s.service.Trade(ctx, in)

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
