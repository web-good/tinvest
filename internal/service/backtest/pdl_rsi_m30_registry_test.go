package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_rsi_m30/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_rsi_m30/strategy/reni"
)

func TestPDLRSIM30RENIUsesItsPackage(t *testing.T) {
	b, err := PDLRSIM30LookupOrGeneric("RENI", enum.Minutes30)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := b.DefaultParams().(core.Params); got != reni.Params() {
		t.Fatalf("defaults = %+v, want RENI literal", got)
	}
}

func TestPDLRSIM30GenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := PDLRSIM30LookupOrGeneric("ZZZZ", enum.Minutes30)
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

func TestPDLRSIM30RunsOnlyOnMinutes30(t *testing.T) {
	if _, err := PDLRSIM30LookupOrGeneric("RENI", enum.Minutes30); err != nil {
		t.Fatalf("Minutes30: %v", err)
	}
	for _, interval := range []enum.Interval{
		enum.Minutes5,
		enum.Minutes15,
		enum.Hour1,
		enum.Hour4,
		enum.Day1,
		enum.Week1,
	} {
		if _, err := PDLRSIM30LookupOrGeneric("RENI", interval); err == nil {
			t.Fatalf("%s: want error", interval)
		}
	}
}

func TestPDLRSIM30ParseParamsLayersOverDefaults(t *testing.T) {
	b, err := PDLRSIM30LookupOrGeneric("RENI", enum.Minutes30)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"StochK": 0, "MinRoomATR": 0.5}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	want := reni.Params()
	want.StochK, want.MinRoomATR = 0, 0.5
	if got := parsed.(core.Params); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
