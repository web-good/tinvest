package scheduler

import (
	"context"
	"time"
	"tinvest/internal/service/trading_strategy/super_trend"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service super_trend.SuperTrend
}

func NewSchedulerService(service super_trend.SuperTrend) super_trend.SuperTrend {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}

func (s *schedulerService) Trade(ctx context.Context) error {
	jobTicker := time.NewTicker(time.Hour)
	defer func() {
		s.sh.Stop()
		jobTicker.Stop()
	}()
	//7-18 * * 1-5"
	err := s.sh.AddJob("*/35 * * * *", func() {
		logger.InfoContext(ctx, "Start worker Super trend")
		err := s.service.Trade(ctx)

		if err != nil {
			logger.ErrorContext(ctx, "Error in job", err)
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
			logger.InfoContext(ctx, "Worker Super trend is running")
		default:
			time.Sleep(10 * time.Second)
		}
	}
}
