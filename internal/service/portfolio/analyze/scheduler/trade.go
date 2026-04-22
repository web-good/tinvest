package scheduler

import (
	"context"
	"time"
	"tinvest/internal/service/portfolio/analyze"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service analyze.Analyze
}

func (s *schedulerService) BondsPortfolio(ctx context.Context, chatID int64) error {
	jobTicker := time.NewTicker(time.Hour)
	defer func() {
		jobTicker.Stop()
	}()
	err := s.sh.AddJob("0 2 * * *", func() {
		logger.InfoContext(ctx, "Воркер Bonds Portfolio Analyze начал работу")
		err := s.service.BondsPortfolio(ctx, chatID)

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
			logger.InfoContext(ctx, "Worker Bonds Portfolio Analyze is running")
		default:
			time.Sleep(10 * time.Second)
		}
	}
}

func NewScheduler(service analyze.Analyze) analyze.Analyze {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}
