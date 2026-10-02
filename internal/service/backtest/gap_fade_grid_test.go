package backtest

import (
	"os"
	"reflect"
	"testing"

	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
)

// TestGapFadeGridMatchesTheSpec pins the shipped grid: one phase, the spec's four axes, 180
// combos, every value applicable to core.Params, the core defaults inside the grid, and no
// combination that switches the stop off.
func TestGapFadeGridMatchesTheSpec(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/gap_fade/grid.json")
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := Grid{
		"GapATR":       {0.3, 0.4, 0.5, 0.7, 1.0},
		"EntryBar":     {0, 1, 2},
		"TargetFill":   {0.5, 0.75, 1.0},
		"StopDailyATR": {0.3, 0.5, 0.8, 1.0},
	}
	if len(phases) != 1 || !reflect.DeepEqual(phases[0].Grid, want) {
		t.Fatalf("phases = %+v, want one phase %v", phases, want)
	}
	combos, err := expandGrid(core.DefaultParams(), phases[0].Grid)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(combos) != 180 {
		t.Fatalf("combos = %d, want 180", len(combos))
	}
	found := false
	for _, c := range combos {
		p := c.(core.Params)
		if p.StopDailyATR <= 0 {
			t.Fatalf("combo %+v switches the stop off", p)
		}
		if p == core.DefaultParams() {
			found = true
		}
	}
	if !found {
		t.Fatal("core.DefaultParams() is not a grid point")
	}
}
