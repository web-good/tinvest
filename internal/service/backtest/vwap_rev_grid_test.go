package backtest

import (
	"os"
	"testing"

	"tinvest/internal/service/trading_strategy/vwap_rev/strategy/core"
)

// TestVWAPRevGridFieldsExist parses the shipped calibration grid and applies every swept
// value to core.Params. A typo in the JSON would otherwise silently collapse a whole phase.
func TestVWAPRevGridFieldsExist(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/vwap_rev/grid.json")
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse phases: %v", err)
	}
	var names []string
	for _, ph := range phases {
		for name, values := range ph.Grid {
			names = append(names, name)
			for _, v := range values {
				if _, err := applyField(core.DefaultParams(), name, v); err != nil {
					t.Fatalf("phase %q: apply %s=%v: %v", ph.Name, name, v, err)
				}
			}
		}
	}
	wantSwept := []string{
		"EntryK", "MinEdgePct", "UseDailyTrend", "DailyEMAPeriod",
		"MaxDevK", "MinClosePos", "StopATR", "MaxHoldBars", "MinBarsFromOpen",
	}
	for _, want := range wantSwept {
		found := false
		for _, got := range names {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("grid must sweep %s", want)
		}
	}
}

// Every optional filter must carry its own "off" control point so the un-filtered baseline
// competes on the leaderboard instead of being assumed away.
func TestVWAPRevGridKeepsFilterOffControlPoints(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/vwap_rev/grid.json")
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse phases: %v", err)
	}
	needOff := map[string]float64{
		"MaxDevK":       0,
		"MinClosePos":   0,
		"UseDailyTrend": 0,
		"MaxHoldBars":   0,
	}
	for _, ph := range phases {
		for name, values := range ph.Grid {
			off, ok := needOff[name]
			if !ok {
				continue
			}
			for _, v := range values {
				if v == off {
					delete(needOff, name)
					break
				}
			}
		}
	}
	for name := range needOff {
		t.Fatalf("grid must include the filter-off control point for %s", name)
	}
}
