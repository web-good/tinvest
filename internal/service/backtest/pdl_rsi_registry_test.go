package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nvtk"
)

func TestPDLRSITickersUseTheirPackages(t *testing.T) {
	b, err := PDLRSILookupOrGeneric("NVTK", enum.Minutes15)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := b.DefaultParams().(core.Params); got != nvtk.Params() {
		t.Fatalf("NVTK defaults = %+v, want nvtk.Params()", got)
	}
}

func TestPDLRSIGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := PDLRSILookupOrGeneric("ZZZZ", enum.Minutes15)
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

func TestPDLRSIRunsOnlyOnMinutes15(t *testing.T) {
	if _, err := PDLRSILookupOrGeneric("NVTK", enum.Minutes15); err != nil {
		t.Fatalf("Minutes15: %v", err)
	}
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes30, enum.Hour1, enum.Day1} {
		if _, err := PDLRSILookupOrGeneric("NVTK", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestPDLRSIParseParamsLayersOverDefaults(t *testing.T) {
	b, err := PDLRSILookupOrGeneric("ZZZZ", enum.Minutes15)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"ZoneATR": 0.3, "MinRoomATR": 0}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	want := core.DefaultParams()
	want.ZoneATR, want.MinRoomATR = 0.3, 0
	if got := parsed.(core.Params); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
