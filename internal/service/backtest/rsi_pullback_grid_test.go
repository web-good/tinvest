package backtest

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// rsiPullbackParamsDir holds the per-ticker single-concern cal_*.json files and the plateau
// points. There is deliberately no ticker-agnostic grid.json any more: calibration is run
// theme by theme out of a ticker's own subdirectory, because the useful axis of every phase
// (how deep an RSI pullback is, how many daily ATRs a stop is worth) is instrument-specific.
const rsiPullbackParamsDir = "../../../data/params/rsi_pullback"

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

// rsiPullbackGridFiles lists every grid file shipped for the strategy. They all live in
// per-ticker subdirectories (gazp/, t/, ugld/), so the walk must be recursive: a flat glob
// would match nothing at all and silently exempt the whole set from the checks below.
func rsiPullbackGridFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(rsiPullbackParamsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk grids: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("expected the per-ticker cal_*.json files, found %d", len(files))
	}
	return files
}

// TestRSIPullbackCalFilesValid guards the single-concern cal_*.json files with the same rules the
// full grid gets: every swept field must resolve through applyField, no phase may be empty, and
// StopDailyATR=0 must not appear anywhere — a stopless multi-day hold is not a configuration any
// grid file may offer. For the cal_*.json files it also pins that the _comment names that same
// file's path relative to rsiPullbackParamsDir (e.g. "t/cal_risk.json"), not just its basename,
// because those comments carry the run command and are written by copying a sibling: a
// basename-only check would pass for a per-ticker file (data/params/rsi_pullback/t/cal_risk.json)
// whose comment was copied from the root file of the same name and still says
// "data/params/rsi_pullback/cal_risk.json" — same basename, wrong directory, wrong ticker.
func TestRSIPullbackCalFilesValid(t *testing.T) {
	for _, path := range rsiPullbackGridFiles(t) {
		name := filepath.Base(path)
		rel := filepath.Join(filepath.Base(filepath.Dir(path)), name)
		t.Run(rel, func(t *testing.T) {
			for _, ph := range rsiPullbackPhases(t, path) {
				if len(ph.Grid) == 0 {
					t.Fatalf("%s: phase %q has an empty grid — that silently degenerates to DefaultParams() at runtime instead of the pinned point the file and its report claim", name, ph.Name)
				}
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
			if !strings.Contains(doc.Comment, rel) {
				t.Fatalf("_comment of %s does not name the file's own path (%q) — the run command was copied from a sibling", name, rel)
			}
		})
	}
}

// TestRSIPullbackGridControlPoints pins the deliberate on/off points. The two optional gates
// must be sweepable to "off" SOMEWHERE in the shipped set — that control lives in cal_screen.json
// and pinning it to any one file would forbid the split — while the reward-to-risk asymmetry is
// checked WITHIN each file: a target must clear the widest stop the same file sweeps. Comparing
// across files would be meaningless, since a stop of 1.0 daily ATR buys a different amount of
// room on T than it does on UGLD.
func TestRSIPullbackGridControlPoints(t *testing.T) {
	var sawDayOff, sawVolumeOff, sawStop bool
	for _, path := range rsiPullbackGridFiles(t) {
		maxStop := 0.0
		for _, ph := range rsiPullbackPhases(t, path) {
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
				if v > maxStop {
					maxStop = v
				}
			}
		}
		if maxStop == 0 {
			continue // this file does not sweep the stop, so it cannot state an asymmetry
		}
		var sawTP, sawTPAboveStop bool
		for _, ph := range rsiPullbackPhases(t, path) {
			for _, v := range ph.Grid["TPDailyATR"] {
				sawTP = true
				if v > maxStop {
					sawTPAboveStop = true
				}
			}
		}
		if sawTP && !sawTPAboveStop {
			t.Fatalf("%s sweeps the stop up to %.2f daily ATR but no target above it: the reward-to-risk asymmetry stays untested", path, maxStop)
		}
	}
	if !sawDayOff {
		t.Fatal("no UseDayATRGate=0 control point in any grid file: the day gate can never be measured against off")
	}
	if !sawVolumeOff {
		t.Fatal("no UseVolume=0 control point in any grid file: the volume gate can never be measured against off")
	}
	if !sawStop {
		t.Fatal("no grid file sweeps StopDailyATR")
	}
}

// TestRSIPullbackPlateauFilesArePoints pins what makes a plateau check meaningful: every key
// carries exactly ONE value, so each walk-forward fold has a single combo to rank and the
// calibrator makes no choice at all. The pooled OOS profit factor then belongs to that fixed
// configuration. Let any key carry two values and the number silently becomes the result of a
// selection procedure — which is the very thing a plateau check exists to rule out.
func TestRSIPullbackPlateauFilesArePoints(t *testing.T) {
	var seen int
	for _, path := range rsiPullbackGridFiles(t) {
		name := filepath.Base(path)
		if !strings.HasPrefix(name, "plateau_") {
			continue
		}
		seen++
		rel := filepath.Join(filepath.Base(filepath.Dir(path)), name)
		t.Run(rel, func(t *testing.T) {
			for _, ph := range rsiPullbackPhases(t, path) {
				if len(ph.Grid) == 0 {
					t.Fatalf("%s: phase %q has an empty grid — that silently degenerates to DefaultParams() at runtime instead of the pinned point the file and its report claim", rel, ph.Name)
				}
				for field, values := range ph.Grid {
					if len(values) != 1 {
						t.Fatalf("phase %q pins %s over %d values: a plateau file must carry exactly one value per key",
							ph.Name, field, len(values))
					}
				}
			}
		})
	}
	if seen == 0 {
		t.Fatal("no plateau_*.json files found: the plateau checks are part of the shipped set")
	}
}
