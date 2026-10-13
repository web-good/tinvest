package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

func TestDropBounceGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Hour1)
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

func TestDropBounceRunsOnHour1Only(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Minutes30, enum.Hour4, enum.Day1, enum.Week1} {
		if _, err := DropBounceLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestDropBounceParseParamsLayersOverDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Hour1)
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

// json.Unmarshal ignores unknown fields: a grid or params file written for the volume-spike rules
// runs on the defaults instead of failing. Pinned so nobody assumes the old fields still apply.
func TestDropBounceParseParamsIgnoresRemovedFields(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Hour1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"VolDailyMult": 0.15, "VolAvgPeriod": 10}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	if got := parsed.(core.Params); got != core.DefaultParams() {
		t.Fatalf("got %+v, want core.DefaultParams()", got)
	}
}

// Every registered ticker resolves to its own literal on Hour1.
func TestDropBounceTickerLiteral(t *testing.T) {
	// The registry is empty today, so inject a test-only literal: otherwise the loop below asserts nothing.
	lit := core.DefaultParams()
	lit.RSIZone = 25
	dropBounceRegistry["ZZTEST"] = func() core.Params { return lit }
	t.Cleanup(func() { delete(dropBounceRegistry, "ZZTEST") })

	for ticker, literal := range dropBounceRegistry {
		b, err := DropBounceLookupOrGeneric(ticker, enum.Hour1)
		if err != nil {
			t.Fatalf("%s: lookup: %v", ticker, err)
		}
		if got := b.DefaultParams().(core.Params); got != literal() {
			t.Fatalf("%s: defaults = %+v, want %+v", ticker, got, literal())
		}
	}
}
