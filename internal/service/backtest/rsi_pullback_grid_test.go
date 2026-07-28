package backtest

import (
	"os"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// rsiPullbackGrid loads the shipped grid file.
func rsiPullbackGrid(t *testing.T) []Phase {
	t.Helper()
	raw, err := os.ReadFile("../../../data/params/rsi_pullback/grid.json")
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse grid: %v", err)
	}
	if len(phases) == 0 {
		t.Fatal("grid has no phases")
	}
	return phases
}

// TestRSIPullbackGridFieldsExist drives every swept value through applyField, which errors on an
// unknown field name — so a typo in the grid fails this test.
func TestRSIPullbackGridFieldsExist(t *testing.T) {
	for _, ph := range rsiPullbackGrid(t) {
		for name, values := range ph.Grid {
			if len(values) == 0 {
				t.Fatalf("phase %q: field %q has no values", ph.Name, name)
			}
			for _, v := range values {
				if _, err := applyField(core.DefaultParams(), name, v); err != nil {
					t.Fatalf("phase %q: applyField(%s=%v): %v", ph.Name, name, v, err)
				}
			}
		}
	}
}

// TestRSIPullbackGridHasTimeStopOffPoint pins the one deliberate control point: the time stop
// must be sweepable to "off" (0), while the ATR stop must NOT be — calibration may never choose
// to trade without protection.
func TestRSIPullbackGridHasTimeStopOffPoint(t *testing.T) {
	var sawHoldOff, sawStop bool
	for _, ph := range rsiPullbackGrid(t) {
		for _, v := range ph.Grid["MaxHoldBars"] {
			if v == 0 {
				sawHoldOff = true
			}
		}
		for _, v := range ph.Grid["StopATR"] {
			sawStop = true
			if v == 0 {
				t.Fatal("StopATR=0 is in the grid: calibration must not be able to disable the stop")
			}
		}
	}
	if !sawHoldOff {
		t.Fatal("no MaxHoldBars=0 control point in the grid")
	}
	if !sawStop {
		t.Fatal("the grid never sweeps StopATR")
	}
}

// TestRSIPullbackGridCombos pins the documented size so a silent grid edit is visible.
func TestRSIPullbackGridCombos(t *testing.T) {
	total := 0
	for _, ph := range rsiPullbackGrid(t) {
		n := 1
		for _, values := range ph.Grid {
			n *= len(values)
		}
		total += n
	}
	if total != 32 {
		t.Fatalf("grid has %d combos, want the documented 32", total)
	}
}
