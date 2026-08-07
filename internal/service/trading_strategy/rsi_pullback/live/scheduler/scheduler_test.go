package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/mocks"
)

// Отмена контекста обязана завершать Run без ошибки: раннер живёт в горутине приложения,
// и Run, не реагирующий на Done, задержал бы весь shutdown до таймаута.
func TestSchedulerServiceReturnsOnContextCancel(t *testing.T) {
	inner := mocks.NewMockService(t)
	inner.EXPECT().Run(mock.Anything, mock.Anything).Return(nil).Maybe()
	sch := NewSchedulerService(inner)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sch.Run(ctx, dto.Run{Scheduler: "1,31 6-23 * * *"}) }()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

// Битое cron-выражение обязано возвращать ошибку, а не тихо работающий воркер без единого
// запуска: раннер, который никогда не проснётся, выглядит живым и не торгует.
func TestSchedulerServiceFailsOnBadCronExpression(t *testing.T) {
	inner := mocks.NewMockService(t)
	sch := NewSchedulerService(inner)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sch.Run(ctx, dto.Run{Scheduler: "не-крон"}); err == nil {
		t.Fatal("Run: want error on a malformed cron expression, got nil")
	}
}
