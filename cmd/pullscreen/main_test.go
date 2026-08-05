package main

import (
	"context"
	"testing"
	"time"
)

func TestPauseAfterTickerSleepsForTheRequestedDelay(t *testing.T) {
	start := time.Now()
	if !pauseAfterTicker(context.Background(), 30*time.Millisecond) {
		t.Fatal("pauseAfterTicker returned false on a live context, want true")
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("slept %v, want at least 30ms — the whole point of -pause is to idle the CPU", elapsed)
	}
}

func TestPauseAfterTickerIsFreeWhenDisabled(t *testing.T) {
	// -pause 0 is the default: no timer, no allocation, no measurable delay.
	start := time.Now()
	if !pauseAfterTicker(context.Background(), 0) {
		t.Fatal("pauseAfterTicker returned false for a zero delay, want true")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Millisecond {
		t.Fatalf("slept %v on a zero delay, want an immediate return", elapsed)
	}
}

func TestPauseAfterTickerAbandonsTheSleepOnCancel(t *testing.T) {
	// Ctrl+C during a paced overnight run must not wait out the pause on every
	// in-flight worker before the pool unwinds.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if pauseAfterTicker(ctx, time.Hour) {
		t.Fatal("pauseAfterTicker returned true on a cancelled context, want false")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("took %v to notice cancellation, want an immediate return", elapsed)
	}
}

func TestEffectiveWorkersCapsOnRefresh(t *testing.T) {
	// 8 workers against the market-data API is roughly 26 req/s, well above what the
	// candle provider's cache writer can survive: Load(refresh=true) logs a failed
	// chunk and keeps going, then writes whatever it got over the existing cache file —
	// silently punching holes in the 540MB local cache the whole screener depends on.
	// -refresh must therefore run gentle regardless of what -workers asked for.
	got, capped := effectiveWorkers(8, true)
	if got != 2 || !capped {
		t.Fatalf("effectiveWorkers(8, true) = %d,%v, want 2,true", got, capped)
	}
}

func TestEffectiveWorkersLeavesLowRequestAlone(t *testing.T) {
	got, capped := effectiveWorkers(1, true)
	if got != 1 || capped {
		t.Fatalf("effectiveWorkers(1, true) = %d,%v, want 1,false — already at or below the refresh cap", got, capped)
	}
}

func TestEffectiveWorkersIgnoresCapWithoutRefresh(t *testing.T) {
	got, capped := effectiveWorkers(8, false)
	if got != 8 || capped {
		t.Fatalf("effectiveWorkers(8, false) = %d,%v, want 8,false — the cache-corruption risk is specific to -refresh", got, capped)
	}
}
