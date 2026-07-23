package backtest

import (
	"os"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping_rsimacd/strategy/core"
)

// TestScalpingRSIMACDGridFieldsExist parses the shipped calibration grid and applies every
// swept value to core.Params. A typo in the JSON would otherwise surface only mid-run.
func TestScalpingRSIMACDGridFieldsExist(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/scalping_rsimacd/grid.json")
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
	wantSwept := []string{"RSIEntryMin", "HTFTrendEMA", "StopATR", "TPATR", "EnableRSIExit"}
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
