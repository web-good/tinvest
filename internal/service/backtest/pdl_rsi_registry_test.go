package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gazp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lent"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mdmg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/svcb"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vtbr"
)

func TestPDLRSITickersUseTheirPackages(t *testing.T) {
	for ticker, want := range map[string]func() core.Params{"GAZP": gazp.Params, "LENT": lent.Params, "MDMG": mdmg.Params, "NVTK": nvtk.Params, "SVCB": svcb.Params, "VTBR": vtbr.Params} {
		b, err := PDLRSILookupOrGeneric(ticker, enum.Hour1)
		if err != nil {
			t.Fatalf("%s lookup: %v", ticker, err)
		}
		if got := b.DefaultParams().(core.Params); got != want() {
			t.Fatalf("%s defaults = %+v, want the package literal", ticker, got)
		}
	}
}

func TestPDLRSIGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := PDLRSILookupOrGeneric("ZZZZ", enum.Hour1)
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

func TestPDLRSIRunsOnlyOnHour1(t *testing.T) {
	if _, err := PDLRSILookupOrGeneric("NVTK", enum.Hour1); err != nil {
		t.Fatalf("Hour1: %v", err)
	}
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Minutes30, enum.Day1} {
		if _, err := PDLRSILookupOrGeneric("NVTK", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestPDLRSIParseParamsLayersOverDefaults(t *testing.T) {
	b, err := PDLRSILookupOrGeneric("ZZZZ", enum.Hour1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"StochK": 10, "MinRoomATR": 0.5}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	want := core.DefaultParams()
	want.StochK, want.MinRoomATR = 10, 0.5
	if got := parsed.(core.Params); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
