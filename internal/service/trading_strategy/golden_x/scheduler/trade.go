package scheduler

import (
	"context"
	"time"
	"tinvest/internal/service/trading_strategy/golden_x"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/service/trading_strategy/golden_x/notification"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/heartbeat"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh       scheduler.Scheduler
	service  golden_x.GoldenX
	tgClient telegram.Client
}

func (s *schedulerService) Trade(ctx context.Context, in dto.Trade) error {
	heartbeat := heartbeat.New()
	heartbeatCh := heartbeat.Beating(4 * time.Hour)
	err := s.sh.AddJob(in.Scheduler, func() {
		logger.InfoContext(ctx, "Воркер Golden RSI начал работу")
		err := s.service.Trade(ctx, in)

		if err != nil {
			heartbeat.Stop()
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
		case _, ok := <-heartbeatCh:
			if !ok {
				return nil
			}

			s.tgClient.SendMessage(notification.Heartbeat(&in))
		default:
			time.Sleep(10 * time.Second)
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
