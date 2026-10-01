package backtest

import (
	"os"
	"reflect"
	"testing"

	midcore "tinvest/internal/service/trading_strategy/rsi_mid/strategy/core"
)

// TestRSIMidGridMatchesTheSpec parses the shipped calibration grid, applies every swept value to
// midcore.Params (a typo in the JSON would otherwise surface only mid-run) and pins the phases,
// their axes and the total combo count from the spec.
func TestRSIMidGridMatchesTheSpec(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/rsi_mid/grid.json")
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse phases: %v", err)
	}
	for _, ph := range phases {
		for name, values := range ph.Grid {
			if name == "DailyATRPeriod" {
				t.Errorf("phase %q sweeps DailyATRPeriod: it is fixed by the spec", ph.Name)
			}
			for _, v := range values {
				if _, err := applyField(midcore.DefaultParams(), name, v); err != nil {
					t.Fatalf("phase %q: apply %s=%v: %v", ph.Name, name, v, err)
				}
			}
		}
	}

	want := []struct {
		name string
		grid Grid
	}{
		{"entry", Grid{"RSIPeriod": {5, 7, 9, 14, 21}}},
		{"trend", Grid{"EMAPeriod": {20, 50, 100, 150, 200}}},
		{"exit", Grid{"RSIUpper": {60, 65, 70, 75, 80, 85}, "BelowMidBars": {1, 2, 3, 4, 6}}},
		{"risk", Grid{"StopDailyATR": {0.2, 0.3, 0.5, 0.7, 1.0}}},
	}
	if len(phases) != len(want) {
		t.Fatalf("phases = %d, want %d", len(phases), len(want))
	}
	for i, w := range want {
		if phases[i].Name != w.name {
			t.Errorf("phase %d name = %q, want %q", i, phases[i].Name, w.name)
		}
		if !reflect.DeepEqual(phases[i].Grid, w.grid) {
			t.Errorf("phase %q grid = %v, want %v", w.name, phases[i].Grid, w.grid)
		}
	}

	// Neither the stop nor the trend filter may be switched off by calibration.
	for _, ph := range phases {
		for _, field := range []string{"StopDailyATR", "EMAPeriod"} {
			for _, v := range ph.Grid[field] {
				if v <= 0 {
					t.Errorf("%s=%v in phase %q: must never be switched off by calibration", field, v, ph.Name)
				}
			}
		}
	}

	// Phase 1 runs over the single default seed; every later phase over the previous phase's
	// keepTop survivors (defaultKeepTop when unset).
	total, seeds := 0, 1
	for _, ph := range phases {
		size := 1
		for _, values := range ph.Grid {
			size *= len(values)
		}
		total += seeds * size
		seeds = ph.KeepTop
		if seeds <= 0 {
			seeds = defaultKeepTop
		}
	}
	if total != 205 {
		t.Errorf("total combos = %d, want 205", total)
	}
}
