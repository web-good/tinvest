package scheduler

import (
	"context"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/reversion/live/dto"
)

type fakeSvc struct{ calls int }

func (f *fakeSvc) Run(context.Context, dto.Run) error { f.calls++; return nil }

func TestSchedulerService_ReturnsOnContextCancel(t *testing.T) {
	inner := &fakeSvc{}
	sch := NewSchedulerService(inner)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sch.Run(ctx, dto.Run{Scheduler: "0 8-23 * * 1-5", Mode: dto.ModeBuy}) }()

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
