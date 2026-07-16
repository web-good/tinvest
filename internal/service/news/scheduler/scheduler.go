// Package scheduler запускает новостной дайджест по cron-расписанию.
package scheduler

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

// Runner — одна итерация дайджеста (news.Service.Run).
type Runner interface {
	Run(ctx context.Context) error
}

type SchedulerService struct {
	sh      scheduler.Scheduler
	service Runner
}

// NewSchedulerService оборачивает Runner: Run регистрирует cron-job и
// блокируется до отмены контекста.
func NewSchedulerService(service Runner) *SchedulerService {
	return &SchedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}

func (s *SchedulerService) Run(ctx context.Context, cronExpr string) error {
	jobTicker := time.NewTicker(time.Hour)
	defer jobTicker.Stop()

	err := s.sh.AddJob(cronExpr, func() {
		// Паника в итерации не должна ронять процесс (прецедент —
		// telegram_commands.runExclusive).
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(ctx, "паника в воркере News", fmt.Sprintf("%v\n%s", r, debug.Stack()))
			}
		}()
		logger.InfoContext(ctx, "Воркер News начал работу")
		if err := s.service.Run(ctx); err != nil {
			logger.ErrorContext(ctx, "Ошибка в ходе работы job News", err.Error())
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
			logger.InfoContext(ctx, "Worker News is running")
		}
	}
}
