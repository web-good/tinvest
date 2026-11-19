package app

import (
	"context"
	"errors"
	"testing"
	"time"
	"tinvest/internal/config"
)

type fakeDensityWorker struct {
	started chan struct{}
	stopped chan struct{}
}

func TestMalformedDensityLiveSettingsRemainLocal(t *testing.T) {
	for _, value := range []string{"not-a-number", "NaN"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("DENSITY_LIVE_RR", value)
			cfg := &config.Config{DensityLive: config.NewDensityLiveConfig(), RSIPullback: &config.RSIPullbackConfig{AccountID: "shared", Token: "token"}}
			loadDensityLiveConfig(context.Background(), cfg)
			if cfg.DensityLiveError == nil || !cfg.RSIPullback.Ready() {
				t.Fatal("density error lost or existing account config damaged")
			}
		})
	}
}

func TestDensityCleanupFailureIsLoggedEvenWhenAppCanceled(t *testing.T) {
	failure := errors.New("unresolved live position")
	if !shouldLogWorkerError(errors.Join(context.Canceled, failure), context.Canceled) {
		t.Fatal("cleanup failure hidden by cancellation")
	}
	if shouldLogWorkerError(context.Canceled, context.Canceled) {
		t.Fatal("normal cancellation reported as failure")
	}
}

func (f *fakeDensityWorker) Run(ctx context.Context) error {
	close(f.started)
	<-ctx.Done()
	close(f.stopped)
	return ctx.Err()
}
func TestDensityWorkerStopsWithApp(t *testing.T) {
	worker := &fakeDensityWorker{started: make(chan struct{}), stopped: make(chan struct{})}
	stop := startWorker(context.Background(), "density", worker)
	<-worker.started
	stop()
	select {
	case <-worker.stopped:
	case <-time.After(time.Second):
		t.Fatal("worker not stopped")
	}
}
