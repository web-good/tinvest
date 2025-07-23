package scheduler

import (
	"context"
	"time"
	"tinvest/pkg/logger"
)

func (s *schedulerService) TakeProfit(ctx context.Context) error {
	jobTicker := time.NewTicker(time.Hour)
	defer func() {
		s.sh.Stop()
		jobTicker.Stop()
	}()
	//7-18 * * 1-5"
	err := s.sh.AddJob("*/5 * * * *", func() {
		logger.InfoContext(ctx, "Start worker Super trend take profit")
		err := s.service.TakeProfit(ctx)

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
			logger.InfoContext(ctx, "Worker Super trend  take profit is running")
		default:
			time.Sleep(10 * time.Second)
		}
	}
}
