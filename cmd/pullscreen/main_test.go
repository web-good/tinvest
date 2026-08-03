package main

import "testing"

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
