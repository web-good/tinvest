package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/hammer/strategy/core"
	"tinvest/internal/service/trading_strategy/hammer/strategy/gazp"
	"tinvest/internal/service/trading_strategy/hammer/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/hammer/strategy/sber"
)

func TestHammerGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := HammerLookupOrGeneric("ZZZZ", enum.Hour1)
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

func TestHammerRunsOnHour1AndMinutes30(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Hour1, enum.Minutes30} {
		if _, err := HammerLookupOrGeneric("GAZP", iv); err != nil {
			t.Fatalf("%s: %v", iv, err)
		}
	}
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Hour4, enum.Day1} {
		if _, err := HammerLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestHammerParseParamsLayersOverDefaults(t *testing.T) {
	b, err := HammerLookupOrGeneric("ZZZZ", enum.Hour1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"StopBufferATR": 0.1, "MACDSlow": 26}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	want := core.DefaultParams()
	want.StopBufferATR, want.MACDSlow = 0.1, 26
	if got := parsed.(core.Params); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestHammerTickersUseTheirPackages(t *testing.T) {
	for ticker, params := range map[string]func() core.Params{"GAZP": gazp.Params, "SBER": sber.Params, "NVTK": nvtk.Params} {
		b, err := HammerLookupOrGeneric(ticker, enum.Hour1)
		if err != nil {
			t.Fatalf("%s lookup: %v", ticker, err)
		}
		if got := b.DefaultParams().(core.Params); got != params() {
			t.Fatalf("%s defaults = %+v, want its package literal", ticker, got)
		}
	}
}
