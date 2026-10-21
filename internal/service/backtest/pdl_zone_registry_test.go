package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_zone/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_zone/strategy/nvtk"
)

func TestPDLZoneGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := PDLZoneLookupOrGeneric("ZZZZ", enum.Minutes5)
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

func TestPDLZoneRunsOnlyOnMinutes5(t *testing.T) {
	if _, err := PDLZoneLookupOrGeneric("NVTK", enum.Minutes5); err != nil {
		t.Fatalf("Minutes5: %v", err)
	}
	for _, iv := range []enum.Interval{enum.Minutes15, enum.Minutes30, enum.Hour1, enum.Day1} {
		if _, err := PDLZoneLookupOrGeneric("NVTK", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestPDLZoneParseParamsLayersOverDefaults(t *testing.T) {
	b, err := PDLZoneLookupOrGeneric("ZZZZ", enum.Minutes5)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"ZoneATR": 0.3, "TargetR": 3}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	want := core.DefaultParams()
	want.ZoneATR, want.TargetR = 0.3, 3
	if got := parsed.(core.Params); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPDLZoneTickersUseTheirPackages(t *testing.T) {
	b, err := PDLZoneLookupOrGeneric("NVTK", enum.Minutes5)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := b.DefaultParams().(core.Params); got != nvtk.Params() {
		t.Fatalf("NVTK defaults = %+v, want nvtk.Params()", got)
	}
}
