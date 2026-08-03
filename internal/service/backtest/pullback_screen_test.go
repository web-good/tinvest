package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

func TestPullbackGridHas24UniqueConfigs(t *testing.T) {
	grid := PullbackGrid()
	if len(grid) != 24 {
		t.Fatalf("grid size = %d, want 24 (2 RSIPeriod x 3 RSILower x 2 EMASlow x 2 TPDailyATR)", len(grid))
	}
	seen := make(map[core.Params]bool, len(grid))
	for _, p := range grid {
		if seen[p] {
			t.Fatalf("duplicate config in grid: %+v", p)
		}
		seen[p] = true
	}
}

func TestPullbackGridSweepsOnlyTheFourAxes(t *testing.T) {
	rsiPeriods := map[int]bool{}
	rsiLowers := map[float64]bool{}
	emaSlows := map[int]bool{}
	tps := map[float64]bool{}
	for _, p := range PullbackGrid() {
		rsiPeriods[p.RSIPeriod] = true
		rsiLowers[p.RSILower] = true
		emaSlows[p.EMASlow] = true
		tps[p.TPDailyATR] = true

		// Everything else is pinned: the screener compares tickers, not configurations.
		if p.EMAFast != 20 {
			t.Fatalf("EMAFast = %d, want pinned 20", p.EMAFast)
		}
		if p.StopDailyATR != 0.5 {
			t.Fatalf("StopDailyATR = %v, want pinned 0.5", p.StopDailyATR)
		}
		if p.DailyATRPeriod != 14 {
			t.Fatalf("DailyATRPeriod = %d, want pinned 14", p.DailyATRPeriod)
		}
		if p.UseDayATRGate != 1 || p.FreshDayATR != 0.3 || p.SpentDayATR != 0.8 {
			t.Fatalf("day gate = %d/%v/%v, want pinned 1/0.3/0.8", p.UseDayATRGate, p.FreshDayATR, p.SpentDayATR)
		}
		if p.UseVolume != 0 {
			t.Fatalf("UseVolume = %d, want pinned 0 (the volume gate starves an already thin sample)", p.UseVolume)
		}
		if p.UseRSIExit != 1 || p.RSIUpper != 60 {
			t.Fatalf("RSI exit = %d/%v, want pinned 1/60", p.UseRSIExit, p.RSIUpper)
		}
		if p.UseTrail != 0 || p.TrailDailyATR != 0 {
			t.Fatalf("trail = %d/%v, want pinned off (trailing is tuning, not ticker fitness)", p.UseTrail, p.TrailDailyATR)
		}
	}
	if len(rsiPeriods) != 2 || len(rsiLowers) != 3 || len(emaSlows) != 2 || len(tps) != 2 {
		t.Fatalf("axes = %d/%d/%d/%d, want 2/3/2/2", len(rsiPeriods), len(rsiLowers), len(emaSlows), len(tps))
	}
}

func TestPullbackGridNeverDisablesTheStop(t *testing.T) {
	// Same invariant TestRSIPullbackCalFilesValid enforces on the JSON grids: a
	// multi-day long without a stop is not a configuration the screener may pick.
	for _, p := range PullbackGrid() {
		if p.StopDailyATR <= 0 {
			t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
		}
	}
}
