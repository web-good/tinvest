package app

import (
	"context"
	"testing"
	"time"
)

type fakeDensityWorker struct {
	started chan struct{}
	stopped chan struct{}
}

func (f *fakeDensityWorker) Run(ctx context.Context) error {
	close(f.started)
	<-ctx.Done()
	close(f.stopped)
	return ctx.Err()
}
func TestDensityWorkerStopsWithApp(t *testing.T) {
	worker := &fakeDensityWorker{started: make(chan struct{}), stopped: make(chan struct{})}
	stop := startDensityWorker(context.Background(), worker)
	<-worker.started
	stop()
	select {
	case <-worker.stopped:
	case <-time.After(time.Second):
		t.Fatal("worker not stopped")
	}
}
