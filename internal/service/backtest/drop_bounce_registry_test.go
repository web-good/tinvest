package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/domrf"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/gazp"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/ivat"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/tbank"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/wush"
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

// Each ticker package supplies its own literal on every timeframe.
func TestDropBounceTickerLiteralPerTimeframe(t *testing.T) {
	cases := []struct {
		ticker string
		want   map[enum.Interval]func() core.Params
	}{
		{"GAZP", map[enum.Interval]func() core.Params{
			enum.Minutes5: gazp.ParamsMinutes5, enum.Minutes15: gazp.ParamsMinutes15, enum.Minutes30: gazp.ParamsMinutes30, enum.Hour1: gazp.ParamsHour1,
		}},
		{"DOMRF", map[enum.Interval]func() core.Params{
			enum.Minutes5: domrf.ParamsMinutes5, enum.Minutes15: domrf.ParamsMinutes15, enum.Minutes30: domrf.ParamsMinutes30, enum.Hour1: domrf.ParamsHour1,
		}},
		{"T", map[enum.Interval]func() core.Params{
			enum.Minutes5: tbank.ParamsMinutes5, enum.Minutes15: tbank.ParamsMinutes15, enum.Minutes30: tbank.ParamsMinutes30, enum.Hour1: tbank.ParamsHour1,
		}},
		{"IVAT", map[enum.Interval]func() core.Params{
			enum.Minutes5: ivat.ParamsMinutes5, enum.Minutes15: ivat.ParamsMinutes15, enum.Minutes30: ivat.ParamsMinutes30, enum.Hour1: ivat.ParamsHour1,
		}},
		{"WUSH", map[enum.Interval]func() core.Params{
			enum.Minutes5: wush.ParamsMinutes5, enum.Minutes15: wush.ParamsMinutes15, enum.Minutes30: wush.ParamsMinutes30, enum.Hour1: wush.ParamsHour1,
		}},
	}
	for _, c := range cases {
		for iv, want := range c.want {
			b, err := DropBounceLookupOrGeneric(c.ticker, iv)
			if err != nil {
				t.Fatalf("%s %s: lookup: %v", c.ticker, iv, err)
			}
			if got := b.DefaultParams().(core.Params); got != want() {
				t.Fatalf("%s %s: defaults = %+v, want %+v", c.ticker, iv, got, want())
			}
		}
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
