package backtest

import (
	"os"
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

func TestDropBounceGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Minutes30)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	got, ok := b.DefaultParams().(core.Params)
	if !ok || got != core.DefaultParams() {
		t.Fatalf("defaults = %#v, want core.DefaultParams()", b.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "ZZZZ" {
		t.Fatalf("ticker = %q, want ZZZZ", s.Ticker())
	}
}

func TestDropBounceRunsOnMinutes30Only(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Hour1, enum.Hour4, enum.Day1, enum.Week1} {
		if _, err := DropBounceLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestDropBounceParseParamsLayersOverDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Minutes30)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"RSIZone": 25, "EMAPeriod": 150}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	want := core.DefaultParams()
	want.RSIZone, want.EMAPeriod = 25, 150
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// json.Unmarshal ignores unknown fields: a -params file written for the volume-spike rules or for
// the day-gate/MACD or buy-window rules (before 2026-10-14) runs on the defaults instead of failing (a -calibrate grid with an old field is rejected by applyField).
// Pinned so nobody assumes the old fields still apply.
func TestDropBounceParseParamsIgnoresRemovedFields(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Minutes30)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"VolDailyMult": 0.15, "VolAvgPeriod": 10, "DaySpentATR": 0.5, "StartHour": 9, "MorningFrom": 3, "EveningTo": 20}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	if got := parsed.(core.Params); got != core.DefaultParams() {
		t.Fatalf("got %+v, want core.DefaultParams()", got)
	}
}

// Every registered ticker resolves to its own literal on Minutes30.
func TestDropBounceTickerLiteral(t *testing.T) {
	// The registry is empty today, so inject a test-only literal: otherwise the loop below asserts nothing.
	lit := core.DefaultParams()
	lit.RSIZone = 25
	dropBounceRegistry["ZZTEST"] = func() core.Params { return lit }
	t.Cleanup(func() { delete(dropBounceRegistry, "ZZTEST") })

	for ticker, literal := range dropBounceRegistry {
		b, err := DropBounceLookupOrGeneric(ticker, enum.Minutes30)
		if err != nil {
			t.Fatalf("%s: lookup: %v", ticker, err)
		}
		if got := b.DefaultParams().(core.Params); got != literal() {
			t.Fatalf("%s: defaults = %+v, want %+v", ticker, got, literal())
		}
	}
}

// The starter grid must expand over the current core.Params: a field the core no longer has
// would fail -calibrate with "unknown grid field".
func TestDropBounceGridExampleExpands(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/drop_bounce/grid_example.json")
	if err != nil {
		t.Fatal(err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("ParsePhases: %v", err)
	}
	total := 1
	for _, ph := range phases {
		combos, err := expandGrid(core.DefaultParams(), ph.Grid)
		if err != nil {
			t.Fatalf("phase %s: %v", ph.Name, err)
		}
		total *= len(combos)
	}
	if total != 45 {
		t.Fatalf("grid_example expands to %d combos, want 45 (3 RSI × 5 exit RSI × 3 stops)", total)
	}
}
