package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/enpg"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/gazp"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/magn"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/moex"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/nlmk"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/plzl"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/rual"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/tbank"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/vsmo"
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
	parsed, err := b.ParseParams([]byte(`{"VolDailyMult": 0.15}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.VolDailyMult != 0.15 || got.StopDailyATR != core.DefaultParams().StopDailyATR {
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

// Each ticker package supplies its own literal on every timeframe.
func TestDropBounceTickerLiteralPerTimeframe(t *testing.T) {
	cases := []struct {
		ticker string
		want   map[enum.Interval]func() core.Params
	}{
		{"ENPG", map[enum.Interval]func() core.Params{
			enum.Minutes5: enpg.ParamsMinutes5, enum.Minutes15: enpg.ParamsMinutes15, enum.Minutes30: enpg.ParamsMinutes30, enum.Hour1: enpg.ParamsHour1,
		}},
		{"GAZP", map[enum.Interval]func() core.Params{
			enum.Minutes5: gazp.ParamsMinutes5, enum.Minutes15: gazp.ParamsMinutes15, enum.Minutes30: gazp.ParamsMinutes30, enum.Hour1: gazp.ParamsHour1,
		}},
		{"MAGN", map[enum.Interval]func() core.Params{
			enum.Minutes5: magn.ParamsMinutes5, enum.Minutes15: magn.ParamsMinutes15, enum.Minutes30: magn.ParamsMinutes30, enum.Hour1: magn.ParamsHour1,
		}},
		{"MOEX", map[enum.Interval]func() core.Params{
			enum.Minutes5: moex.ParamsMinutes5, enum.Minutes15: moex.ParamsMinutes15, enum.Minutes30: moex.ParamsMinutes30, enum.Hour1: moex.ParamsHour1,
		}},
		{"NLMK", map[enum.Interval]func() core.Params{
			enum.Minutes5: nlmk.ParamsMinutes5, enum.Minutes15: nlmk.ParamsMinutes15, enum.Minutes30: nlmk.ParamsMinutes30, enum.Hour1: nlmk.ParamsHour1,
		}},
		{"NVTK", map[enum.Interval]func() core.Params{
			enum.Minutes5: nvtk.ParamsMinutes5, enum.Minutes15: nvtk.ParamsMinutes15, enum.Minutes30: nvtk.ParamsMinutes30, enum.Hour1: nvtk.ParamsHour1,
		}},
		{"PLZL", map[enum.Interval]func() core.Params{
			enum.Minutes5: plzl.ParamsMinutes5, enum.Minutes15: plzl.ParamsMinutes15, enum.Minutes30: plzl.ParamsMinutes30, enum.Hour1: plzl.ParamsHour1,
		}},
		{"RUAL", map[enum.Interval]func() core.Params{
			enum.Minutes5: rual.ParamsMinutes5, enum.Minutes15: rual.ParamsMinutes15, enum.Minutes30: rual.ParamsMinutes30, enum.Hour1: rual.ParamsHour1,
		}},
		{"T", map[enum.Interval]func() core.Params{
			enum.Minutes5: tbank.ParamsMinutes5, enum.Minutes15: tbank.ParamsMinutes15, enum.Minutes30: tbank.ParamsMinutes30, enum.Hour1: tbank.ParamsHour1,
		}},
		{"VSMO", map[enum.Interval]func() core.Params{
			enum.Minutes5: vsmo.ParamsMinutes5, enum.Minutes15: vsmo.ParamsMinutes15, enum.Minutes30: vsmo.ParamsMinutes30, enum.Hour1: vsmo.ParamsHour1,
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
