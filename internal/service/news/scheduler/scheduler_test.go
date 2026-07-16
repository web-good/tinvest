package scheduler

import (
	"context"
	"os"
	"testing"
	"time"

	"tinvest/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

type stubRunner struct{}

func (stubRunner) Run(context.Context) error { return nil }

type panicRunner struct{}

func (panicRunner) Run(context.Context) error { panic("boom") }

// TestSchedulerService_RecoversJobPanic: паника в Runner.Run внутри
// cron-джобы не должна ронять процесс (recover + лог со стеком). "@every 1s"
// поддерживается стандартным парсером robfig/cron/v3 (Descriptor во flags).
func TestSchedulerService_RecoversJobPanic(t *testing.T) {
	sch := NewSchedulerService(panicRunner{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sch.Run(ctx, "@every 1s") }()

	time.Sleep(1500 * time.Millisecond) // даём джобе тикнуть и запаниковать
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel (process likely crashed on panic)")
	}
}

func TestSchedulerService_ReturnsOnContextCancel(t *testing.T) {
	sch := NewSchedulerService(stubRunner{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sch.Run(ctx, "5 * * * *") }()

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

func TestSchedulerService_BadCronExpr(t *testing.T) {
	if err := NewSchedulerService(stubRunner{}).Run(context.Background(), "не крон"); err == nil {
		t.Fatal("ожидалась ошибка на невалидном cron-выражении")
	}
}
