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
		{Name: "fvg", KeepTop: 8, Grid: Grid{"MinGapATR": {.25, .5, .8, 1}, "MaxRetestBars": {4, 8, 12}, "MinRetestDepthPct": {0, 50, 100}, "TakeProfitR": {1, 1.5, 2}}},
		{Name: "formation", KeepTop: 8, Grid: Grid{"RequireThirdCandleHigherHigh": {0, 1}}},
		{Name: "retest_mode", KeepTop: 8, Grid: Grid{"RetestDepthByBody": {0, 1}}},
		{Name: "context", KeepTop: 8, Grid: Grid{"HTFTrendEMA": {0, 20, 50, 100, 150, 200}}},
		{Name: "execution", KeepTop: 8, Grid: Grid{"StopBufferATR": {0, .1, .25, .5}}},
		{Name: "stochastic", KeepTop: 8, Grid: Grid{"UseStochFilter": {0, 1}}},
	}
	if !reflect.DeepEqual(grid.Phases, want) {
		t.Fatalf("phases = %+v, want %+v", grid.Phases, want)
	}
	for i, phase := range grid.Phases {
		combos, err := expandGrid(core.DefaultParams(), phase.Grid)
		if err != nil {
			t.Fatalf("%s: %v", phase.Name, err)
		}
		counts := []int{108, 2, 2, 6, 4, 2}
		if len(combos) != counts[i] {
			t.Errorf("%s: got %d combinations, want %d", phase.Name, len(combos), counts[i])
		}
	}
}
