package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

func TestDropBounceGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Minutes15)
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

func TestDropBounceRejectsOtherTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Hour4, enum.Day1, enum.Week1} {
		if _, err := DropBounceLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestDropBounceParseParamsLayersOverDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Minutes15)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"MinDropPct": 2.5}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.MinDropPct != 2.5 || got.StopDailyATR != core.DefaultParams().StopDailyATR {
		t.Fatalf("got %+v", got)
	}
}

// A ticker without a package runs core defaults on every timeframe.
func TestDropBounceGenericOnLongerTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes30, enum.Hour1} {
		b, err := DropBounceLookupOrGeneric("ZZZZ", iv)
		if err != nil {
			t.Fatalf("%s: lookup: %v", iv, err)
		}
		if got := b.DefaultParams().(core.Params); got != core.DefaultParams() {
			t.Fatalf("%s: defaults = %+v, want core.DefaultParams()", iv, got)
		}
	}
}

// No ticker has a package since the 2026-10-12 rework: every ticker on every supported timeframe
// runs on the core defaults.
func TestDropBounceEveryTimeframeUsesCoreDefaults(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Minutes30, enum.Hour1} {
		b, err := DropBounceLookupOrGeneric("GAZP", iv)
		if err != nil {
			t.Fatalf("%s: %v", iv, err)
		}
		if got, ok := b.DefaultParams().(core.Params); !ok || got != core.DefaultParams() {
			t.Fatalf("%s: defaults = %#v, want core.DefaultParams()", iv, b.DefaultParams())
		}
	}
}
