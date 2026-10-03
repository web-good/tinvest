package backtest

import (
	"os"
	"path/filepath"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// TestRSIZoneProfitGrids guards every data/params/rsi_zone/*/cal_profit.json: each value applies
// to core.Params, the theme sweeps the minimal ProfitExitBars × ProfitExitPct axes including
// ProfitExitPct 0 (the exit off, so the same run carries its own baseline), never a negative
// value, and the six base fields are all listed so they are pinned to the ticker's literal rather
// than to the core defaults.
func TestRSIZoneProfitGrids(t *testing.T) {
	files, err := filepath.Glob("../../../data/params/rsi_zone/*/cal_profit.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no cal_profit.json found — the profit theme has no grid")
	}
	base := []string{"RSIPeriod", "RSILower", "RSIUpper", "EMAPeriod", "DailyATRPeriod", "StopDailyATR"}
	axes := map[string][]float64{
		"ProfitExitBars": {0, 1, 2, 3, 4, 6, 8, 12},
		"ProfitExitPct":  {0, 0.3, 0.5, 0.75, 1.0, 1.5, 2.0},
	}
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
			for axis, want := range axes {
				have := map[float64]bool{}
				for _, v := range g[axis] {
					if v < 0 {
						t.Errorf("%s=%v: never negative", axis, v)
					}
					have[v] = true
				}
				for _, v := range want {
					if !have[v] {
						t.Errorf("%s misses %v (have %v)", axis, v, g[axis])
					}
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
