package backtest

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestFVGImbalanceGrid(t *testing.T) {
	for _, tc := range []struct {
		path   string
		phases []Phase
		counts []int
	}{
		{
			path: "gazp/grid.json",
			phases: []Phase{
				{Name: "fvg", KeepTop: 8, Grid: Grid{"MinGapATR": {.25, .5, .8, 1}, "MaxRetestBars": {4, 8, 12}, "MinRetestDepthPct": {0, 50, 100}, "TakeProfitR": {1, 1.5, 2}}},
				{Name: "formation", KeepTop: 8, Grid: Grid{"RequireThirdCandleHigherHigh": {0, 1}}},
				{Name: "retest_mode", KeepTop: 8, Grid: Grid{"RetestDepthByBody": {0, 1}}},
				{Name: "context", KeepTop: 8, Grid: Grid{"HTFTrendEMA": {0, 20, 50, 100, 150, 200}}},
				{Name: "execution", KeepTop: 8, Grid: Grid{"StopBufferATR": {0, .1, .25, .5}}},
				{Name: "stochastic", KeepTop: 8, Grid: Grid{"UseStochFilter": {0, 1}}},
			},
			counts: []int{108, 2, 2, 6, 4, 2},
		},
		{
			path: "gazp/ordered_grid.json",
			phases: []Phase{
				{Name: "gap", KeepTop: 4, Grid: Grid{"MinGapATR": {.25, .5, .8, 1}}},
				{Name: "retest_window", KeepTop: 4, Grid: Grid{"MaxRetestBars": {8, 12, 16}}},
				{Name: "retest_depth", KeepTop: 4, Grid: Grid{"MinRetestDepthPct": {0, 50, 100}}},
				{Name: "formation", KeepTop: 4, Grid: Grid{"RequireThirdCandleHigherHigh": {0, 1}}},
				{Name: "retest_mode", KeepTop: 4, Grid: Grid{"RetestDepthByBody": {0, 1}}},
				{Name: "h4_trend", KeepTop: 4, Grid: Grid{"HTFTrendEMA": {0, 20, 50, 100, 150, 200}}},
				{Name: "stop", KeepTop: 4, Grid: Grid{"StopBufferATR": {0, .1, .25, .5, 1}}},
				{Name: "target", KeepTop: 4, Grid: Grid{"TakeProfitR": {1, 1.5, 2}}},
				{Name: "stochastic", KeepTop: 4, Grid: Grid{"UseStochFilter": {0, 1}}},
			},
			counts: []int{4, 3, 3, 2, 2, 6, 5, 3, 2},
		},
		{
			path: "domrf/grid.json",
			phases: []Phase{
				{Name: "gap", KeepTop: 8, Grid: Grid{"MinGapATR": {.25, .5, .8, 1}}},
				{Name: "formation", KeepTop: 8, Grid: Grid{"RequireThirdCandleHigherHigh": {0, 1}}},
				{Name: "retest_mode", KeepTop: 8, Grid: Grid{"RetestDepthByBody": {0, 1}}},
				{Name: "h4_trend", KeepTop: 8, Grid: Grid{"HTFTrendEMA": {0, 20, 50, 100, 150, 200}}},
				{Name: "stop", KeepTop: 8, Grid: Grid{"StopBufferATR": {0, .1, .25, .5}}},
				{Name: "target", KeepTop: 8, Grid: Grid{"TakeProfitR": {1, 1.5, 2}}},
				{Name: "stochastic", KeepTop: 8, Grid: Grid{"UseStochFilter": {0, 1}}},
			},
			counts: []int{4, 2, 2, 6, 4, 3, 2},
		},
	} {
		t.Run(tc.path, func(t *testing.T) {
			raw, err := os.ReadFile("../../../data/params/fvg_imbalance/" + tc.path)
			if err != nil {
				t.Fatal(err)
			}
			var grid PhasedGrid
			if err := json.Unmarshal(raw, &grid); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(grid.Phases, tc.phases) {
				t.Fatalf("phases = %+v, want %+v", grid.Phases, tc.phases)
			}
			for i, phase := range grid.Phases {
				base := core.DefaultParams()
				combos, err := expandGrid(base, phase.Grid)
				if err != nil {
					t.Fatalf("%s: %v", phase.Name, err)
				}
				if len(combos) != tc.counts[i] {
					t.Fatalf("%s: got %d combinations, want %d", phase.Name, len(combos), tc.counts[i])
				}
				for j, combo := range combos {
					p := combo.(core.Params)
					if err := core.ValidateParams(p); err != nil {
						t.Fatalf("%s combination %d: %v", phase.Name, j, err)
					}
					if phase.Name == "stochastic" {
						want := base
						want.UseStochFilter = j // ordered [0,1]; all other defaults survive.
						if p != want {
							t.Fatalf("stochastic combination %d = %+v, want %+v", j, p, want)
						}
					}
				}
			}
		})
	}
}
