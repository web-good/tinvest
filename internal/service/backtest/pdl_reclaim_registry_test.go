package backtest

import (
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/gazp"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/sber"
)

func TestPDLReclaimGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := PDLReclaimLookupOrGeneric("ZZZZ", enum.Minutes15)
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

func TestPDLReclaimRunsOnMinutes15Only(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes30, enum.Hour1, enum.Hour4, enum.Day1, enum.Week1} {
		if _, err := PDLReclaimLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestPDLReclaimParseParamsLayersOverDefaults(t *testing.T) {
	b, err := PDLReclaimLookupOrGeneric("ZZZZ", enum.Minutes15)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"StopDailyATR": 0.7, "HoldBars": 3}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	want := core.DefaultParams()
	want.StopDailyATR, want.HoldBars = 0.7, 3
	if got := parsed.(core.Params); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPDLReclaimTickersUseTheirPackages(t *testing.T) {
	for ticker, params := range map[string]func() core.Params{"GAZP": gazp.Params, "NVTK": nvtk.Params, "SBER": sber.Params} {
		b, err := PDLReclaimLookupOrGeneric(ticker, enum.Minutes15)
		if err != nil {
			t.Fatalf("%s lookup: %v", ticker, err)
		}
		if got := b.DefaultParams().(core.Params); got != params() {
			t.Fatalf("%s: got %+v, want its package literal", ticker, got)
		}
	}
}

// A registered ticker resolves to its own literal; the test injects one of its own.
func TestPDLReclaimTickerLiteral(t *testing.T) {
	lit := core.DefaultParams()
	lit.StopDailyATR = 0.77
	pdlReclaimRegistry["TEST"] = func() core.Params { return lit }
	t.Cleanup(func() { delete(pdlReclaimRegistry, "TEST") })
	b, err := PDLReclaimLookupOrGeneric("TEST", enum.Minutes15)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := b.DefaultParams().(core.Params); got != lit {
		t.Fatalf("got %+v, want %+v", got, lit)
	}
}

// PDLReclaimWithIndex keeps the binding's literal and ticker; it only attaches the index dailies.
func TestPDLReclaimWithIndexKeepsBinding(t *testing.T) {
	b, err := PDLReclaimLookupOrGeneric("SBER", enum.Minutes15)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	b = PDLReclaimWithIndex(b, []backtest.Candle{{Time: time.Date(2026, 9, 11, 7, 0, 0, 0, time.UTC), Close: 3000}})
	if got := b.DefaultParams().(core.Params); got != sber.Params() {
		t.Fatalf("defaults = %+v, want sber.Params()", got)
	}
	if s := b.Build(b.DefaultParams()); s.Ticker() != "SBER" {
		t.Fatalf("ticker = %q, want SBER", s.Ticker())
	}
}
