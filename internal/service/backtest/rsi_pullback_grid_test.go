package backtest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// rsiPullbackParamsDir holds the full phased grid plus the single-concern cal_*.json files.
const rsiPullbackParamsDir = "../../../data/params/rsi_pullback"

// rsiPullbackGrid loads the shipped grid file.
func rsiPullbackGrid(t *testing.T) []Phase {
	t.Helper()
	return rsiPullbackPhases(t, filepath.Join(rsiPullbackParamsDir, "grid.json"))
}

// rsiPullbackPhases parses one grid file and fails the test if it is empty or malformed.
func rsiPullbackPhases(t *testing.T, path string) []Phase {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // fixed test fixture path
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if len(phases) == 0 {
		t.Fatalf("%s has no phases", path)
	}
	return phases
}

// rsiPullbackGridFiles lists every grid file shipped for the strategy.
func rsiPullbackGridFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(rsiPullbackParamsDir, "*.json"))
	if err != nil {
		t.Fatalf("glob grids: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("expected the phased grid plus the cal_*.json files, found %d", len(files))
	}
	return files
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

// TestRSIPullbackCalFilesValid guards the single-concern cal_*.json files with the same rules the
// full grid gets: every swept field must resolve through applyField, no phase may be empty, and
// StopDailyATR=0 must not appear anywhere — a stopless multi-day hold is not a configuration any
// grid file may offer. For the cal_*.json files it also pins that the _comment names that same
// file, because those comments carry the run command and are written by copying a sibling.
func TestRSIPullbackCalFilesValid(t *testing.T) {
	for _, path := range rsiPullbackGridFiles(t) {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			for _, ph := range rsiPullbackPhases(t, path) {
				for field, values := range ph.Grid {
					if len(values) == 0 {
						t.Fatalf("phase %q: field %q has no values", ph.Name, field)
					}
					for _, v := range values {
						if _, err := applyField(core.DefaultParams(), field, v); err != nil {
							t.Fatalf("phase %q: applyField(%s=%v): %v", ph.Name, field, v, err)
						}
						if field == "StopDailyATR" && v == 0 {
							t.Fatalf("phase %q sweeps StopDailyATR=0: calibration must not be able to disable the stop", ph.Name)
						}
					}
				}
			}

			if !strings.HasPrefix(name, "cal_") {
				return // only the single-concern files carry a run command in their comment
			}
			raw, err := os.ReadFile(path) //nolint:gosec // fixed test fixture path
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			var doc struct {
				Comment string `json:"_comment"`
			}
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("unmarshal %s: %v", path, err)
			}
			if !strings.Contains(doc.Comment, name) {
				t.Fatalf("_comment of %s does not name the file itself — the run command was copied from a sibling", name)
			}
		})
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
	if total != 243 {
		t.Fatalf("phased calibration costs %d evaluations, want the documented 243", total)
	}
}
