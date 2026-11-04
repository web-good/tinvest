package backtest

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestFVGImbalanceGrid(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/fvg_imbalance/gazp/grid.json")
	if err != nil {
		t.Fatal(err)
	}
	var grid PhasedGrid
	if err := json.Unmarshal(raw, &grid); err != nil {
		t.Fatal(err)
	}
	want := []Phase{
		{Name: "fvg", KeepTop: 8, Grid: Grid{"MinGapATR": {.25, .5, .8, 1}, "MaxRetestBars": {4, 8, 12}, "RequireCloseAboveZone": {0, 1}, "TakeProfitR": {1, 1.5, 2}}},
		{Name: "context", KeepTop: 8, Grid: Grid{"RequireFirstRetest": {0, 1}, "RequireBullishBOS": {0, 1}, "RequireLiquiditySweep": {0, 1}, "HTFTrendEMA": {0, 20, 50, 100}, "SwingK": {2}, "StructureLookbackBars": {48}, "MaxSweepToFVGBars": {8}}},
		{Name: "execution", KeepTop: 8, Grid: Grid{"MaxEntryDistanceATR": {0, .25, .5, 1}, "StopBufferATR": {0, .1, .25, .5}}},
		{Name: "holding", KeepTop: 8, Grid: Grid{"MaxHoldBars": {0, 8, 16, 32, 48}}},
	}
	if !reflect.DeepEqual(grid.Phases, want) {
		t.Fatalf("phases = %+v, want %+v", grid.Phases, want)
	}
	for i, phase := range grid.Phases {
		combos, err := expandGrid(core.DefaultParams(), phase.Grid)
		if err != nil {
			t.Fatalf("%s: %v", phase.Name, err)
		}
		counts := []int{72, 32, 16, 5}
		if len(combos) != counts[i] {
			t.Errorf("%s: got %d combinations, want %d", phase.Name, len(combos), counts[i])
		}
	}
}
