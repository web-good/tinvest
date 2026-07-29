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

// TestRSIPullbackGridControlPoints pins the deliberate on/off points: both optional gates must
// be sweepable to "off", and the stop must NOT be — calibration may never choose to hold a
// multi-day position without protection.
func TestRSIPullbackGridControlPoints(t *testing.T) {
	var sawDayOff, sawVolumeOff, sawStop, sawTPAboveStop bool
	maxStop := 0.0
	for _, ph := range rsiPullbackGrid(t) {
		for _, v := range ph.Grid["UseDayATRGate"] {
			if v == 0 {
				sawDayOff = true
			}
		}
		for _, v := range ph.Grid["UseVolume"] {
			if v == 0 {
				sawVolumeOff = true
			}
		}
		for _, v := range ph.Grid["StopDailyATR"] {
			sawStop = true
			if v == 0 {
				t.Fatal("StopDailyATR=0 is in the grid: calibration must not be able to disable the stop")
			}
			if v > maxStop {
				maxStop = v
			}
		}
	}
	for _, ph := range rsiPullbackGrid(t) {
		for _, v := range ph.Grid["TPDailyATR"] {
			if v > maxStop {
				sawTPAboveStop = true
			}
		}
	}
	if !sawDayOff {
		t.Fatal("no UseDayATRGate=0 control point in the grid")
	}
	if !sawVolumeOff {
		t.Fatal("no UseVolume=0 control point in the grid")
	}
	if !sawStop {
		t.Fatal("the grid never sweeps StopDailyATR")
	}
	if !sawTPAboveStop {
		t.Fatal("the grid never tests a target above the stop: the 0.6:1 asymmetry stays untested")
	}
}

// TestRSIPullbackGridEvaluationCost pins the real cost of a phased calibration. RunPhases
// expands every phase over the previous phase's keepTop seeds, so the number of backtest runs
// is NOT the sum of the grid sizes — an earlier revision of this file understated it fourfold.
func TestRSIPullbackGridEvaluationCost(t *testing.T) {
	phases := rsiPullbackGrid(t)
	seeds := 1
	total := 0
	for _, ph := range phases {
		n := 1
		for _, values := range ph.Grid {
			n *= len(values)
		}
		total += seeds * n
		if ph.KeepTop > 0 {
			seeds = ph.KeepTop
		}
	}
	if total != 231 {
		t.Fatalf("phased calibration costs %d evaluations, want the documented 231", total)
	}
}
