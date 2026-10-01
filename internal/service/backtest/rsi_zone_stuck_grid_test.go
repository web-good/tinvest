package backtest

import (
	"os"
	"path/filepath"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// TestRSIZoneStuckGrids guards every data/params/rsi_zone/*/cal_stuck.json: each value applies
// to core.Params, the theme sweeps the minimal StuckExitBars axis including 0 (the exit off, so
// the same run carries its own baseline), never a negative count, and the six base fields are
// all listed so they are pinned to the ticker's literal rather than to the core defaults.
func TestRSIZoneStuckGrids(t *testing.T) {
	files, err := filepath.Glob("../../../data/params/rsi_zone/*/cal_stuck.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no cal_stuck.json found — the stuck theme has no grid")
	}
	base := []string{"RSIPeriod", "RSILower", "RSIUpper", "EMAPeriod", "DailyATRPeriod", "StopDailyATR"}
	for _, f := range files {
		t.Run(filepath.Base(filepath.Dir(f)), func(t *testing.T) {
			raw, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			phases, err := ParsePhases(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(phases) != 1 {
				t.Fatalf("phases = %d, want 1", len(phases))
			}
			g := phases[0].Grid
			for name, values := range g {
				for _, v := range values {
					if _, err := applyField(core.DefaultParams(), name, v); err != nil {
						t.Fatalf("apply %s=%v: %v", name, v, err)
					}
				}
			}
			have := map[float64]bool{}
			for _, v := range g["StuckExitBars"] {
				if v < 0 {
					t.Errorf("StuckExitBars=%v: a bar count is never negative", v)
				}
				have[v] = true
			}
			for _, v := range []float64{0, 2, 3, 4, 6, 8} {
				if !have[v] {
					t.Errorf("StuckExitBars misses %v (have %v)", v, g["StuckExitBars"])
				}
			}
			for _, name := range base {
				if len(g[name]) != 1 {
					t.Errorf("%s = %v, want exactly one value pinned to the ticker literal", name, g[name])
				}
			}
			for _, v := range g["StopDailyATR"] {
				if v <= 0 {
					t.Errorf("StopDailyATR=%v: the stop must never be switched off by calibration", v)
				}
			}
		})
	}
}
