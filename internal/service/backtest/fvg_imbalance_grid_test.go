package backtest

import (
	"encoding/json"
	"os"
	"testing"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestFVGImbalanceGrid(t *testing.T) {
	for _, path := range []string{"gazp/grid.json", "gazp/ordered_grid.json", "domrf/grid.json"} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile("../../../data/params/fvg_imbalance/" + path)
			if err != nil {
				t.Fatal(err)
			}
			var grid PhasedGrid
			if err = json.Unmarshal(raw, &grid); err != nil {
				t.Fatal(err)
			}
			axes := map[string]bool{}
			for _, phase := range grid.Phases {
				for key := range phase.Grid {
					axes[key] = true
				}
				combos, err := expandGrid(core.DefaultParams(), phase.Grid)
				if err != nil {
					t.Fatal(err)
				}
				for _, combo := range combos {
					if err := core.ValidateParams(combo.(core.Params)); err != nil {
						t.Fatal(err)
					}
				}
			}
			keys := []string{"MinGapATR", "StochPeriod", "HTFTrendEMA", "StopATR", "TakeProfitATR", "UseStochExit"}
			if path != "gazp/grid.json" {
				keys = append(keys, "MaxWaitBars")
			}
			for _, key := range keys {
				if !axes[key] {
					t.Errorf("missing %s", key)
				}
			}
		})
	}
}

func TestFVGEntryFloorGrid(t *testing.T) {
	combos, err := expandGrid(core.DefaultParams(), map[string][]float64{"RequireEntryAboveFVGLower": {0, 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(combos) != 2 {
		t.Fatalf("combinations = %d", len(combos))
	}
	for i, combo := range combos {
		p := combo.(core.Params)
		if p.RequireEntryAboveFVGLower != i {
			t.Fatalf("filter = %d, want %d", p.RequireEntryAboveFVGLower, i)
		}
		if err := core.ValidateParams(p); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLKOHJointGridIncludesBaselineAndBothFilters(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/fvg_imbalance/lkoh/joint_grid.json")
	if err != nil {
		t.Fatal(err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(phases) != 1 {
		t.Fatalf("want one exhaustive phase, got %d", len(phases))
	}
	combos, err := expandGrid(core.DefaultParams(), phases[0].Grid)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	baseline := false
	for _, combo := range combos {
		p := combo.(core.Params)
		if err := core.ValidateParams(p); err != nil {
			t.Fatal(err)
		}
		seen[p.RequireEntryAboveFVGLower] = true
		if p.ATRPeriod == 14 && p.MinGapATR == .25 && p.MaxWaitBars == 30 && p.StochPeriod == 12 && p.UseStochExit == 1 && p.StopATR == .5 && p.TakeProfitATR == 1.5 && p.HTFTrendEMA == 0 && p.RequireEntryAboveFVGLower == 1 {
			baseline = true
		}
	}
	if !seen[0] || !seen[1] || !baseline {
		t.Fatalf("missing filter values or baseline: %v, %v", seen, baseline)
	}
}
