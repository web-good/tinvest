package scheduler

import (
	"context"
	"time"

	"tinvest/internal/service/portfolio/yield"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service yield.Yield
}

func (s *schedulerService) PortfolioYieldYTD(ctx context.Context, chatID int64) error {
	jobTicker := time.NewTicker(time.Hour)
	defer func() {
		jobTicker.Stop()
	}()
	err := s.sh.AddJob("0 2 * * *", func() {
		logger.InfoContext(ctx, "Воркер Portfolio Yield YTD начал работу")
		err := s.service.PortfolioYieldYTD(ctx, chatID)

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
			logger.InfoContext(ctx, "Worker Portfolio Yield YTD is running")
		default:
			time.Sleep(10 * time.Second)
		}
	}
}

func NewScheduler(service yield.Yield) yield.Yield {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}
