package backtest

import (
	"os"
	"reflect"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// TestRSIZoneGridMatchesTheSpec parses the shipped calibration grid, applies every swept value
// to core.Params (a typo in the JSON would otherwise surface only mid-run) and pins the phases,
// their axes and the total combo count from the spec.
func TestRSIZoneGridMatchesTheSpec(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/rsi_zone/grid.json")
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse phases: %v", err)
	}
	for _, ph := range phases {
		for name, values := range ph.Grid {
			for _, v := range values {
				if _, err := applyField(core.DefaultParams(), name, v); err != nil {
					t.Fatalf("phase %q: apply %s=%v: %v", ph.Name, name, v, err)
				}
			}
		}
	}

	want := []struct {
		name string
		grid Grid
	}{
		{"entry", Grid{"RSIPeriod": {4, 5, 6}, "RSILower": {10, 15, 20, 25, 30}}},
		{"trend", Grid{"EMAPeriod": {50, 100, 150, 200}}},
		{"exit", Grid{"RSIUpper": {60, 65, 70, 75, 80, 85}}},
		{"risk", Grid{"StopDailyATR": {0.5, 0.7, 1.0, 1.5}}},
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

	// A naked multi-day long is not a point calibration may pick.
	for _, v := range phases[len(phases)-1].Grid["StopDailyATR"] {
		if v <= 0 {
			t.Errorf("StopDailyATR=%v in the grid: the stop must never be switched off by calibration", v)
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
	if total != 85 {
		t.Errorf("total combos = %d, want 85", total)
	}
}
