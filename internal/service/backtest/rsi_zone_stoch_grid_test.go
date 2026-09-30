package backtest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// TestRSIZoneStochGrids guards every data/params/rsi_zone/*/cal_stoch.json: each value applies
// to core.Params, the gate is switched on, all four stochastic axes are listed (so no axis is
// silently measured at a zero-resolved default), and the axes carry the minimal values of the
// theme.
func TestRSIZoneStochGrids(t *testing.T) {
	files, err := filepath.Glob("../../../data/params/rsi_zone/*/cal_stoch.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no cal_stoch.json found — the stoch theme has no grid")
	}
	minimal := map[string][]float64{
		"StochKPeriod":   {5, 9, 14},
		"StochDSmooth":   {1, 3},
		"StochLower":     {10, 15, 20, 25, 30},
		"ZoneWindowBars": {1, 2, 3, 5, 8},
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
			if !reflect.DeepEqual(g["UseStoch"], []float64{1}) {
				t.Errorf("UseStoch = %v, want [1]", g["UseStoch"])
			}
			for name, want := range minimal {
				have := map[float64]bool{}
				for _, v := range g[name] {
					have[v] = true
				}
				for _, v := range want {
					if !have[v] {
						t.Errorf("%s misses %v (have %v)", name, v, g[name])
					}
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
