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
