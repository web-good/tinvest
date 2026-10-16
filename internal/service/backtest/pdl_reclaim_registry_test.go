package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/gazp"
)

func TestPDLReclaimGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := PDLReclaimLookupOrGeneric("ZZZZ", enum.Minutes5)
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

func TestPDLReclaimRunsOnMinutes5Only(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes15, enum.Minutes30, enum.Hour1, enum.Hour4, enum.Day1, enum.Week1} {
		if _, err := PDLReclaimLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestPDLReclaimParseParamsLayersOverDefaults(t *testing.T) {
	b, err := PDLReclaimLookupOrGeneric("ZZZZ", enum.Minutes5)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"StopDailyATR": 0.7, "RSIExit": 75}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	want := core.DefaultParams()
	want.StopDailyATR, want.RSIExit = 0.7, 75
	if got := parsed.(core.Params); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPDLReclaimGAZPUsesItsPackage(t *testing.T) {
	b, err := PDLReclaimLookupOrGeneric("GAZP", enum.Minutes5)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := b.DefaultParams().(core.Params); got != gazp.Params() {
		t.Fatalf("got %+v, want gazp.Params()", got)
	}
}

// A registered ticker resolves to its own literal; the test injects one of its own.
func TestPDLReclaimTickerLiteral(t *testing.T) {
	lit := core.DefaultParams()
	lit.RSIExit = 77
	pdlReclaimRegistry["TEST"] = func() core.Params { return lit }
	t.Cleanup(func() { delete(pdlReclaimRegistry, "TEST") })
	b, err := PDLReclaimLookupOrGeneric("TEST", enum.Minutes5)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := b.DefaultParams().(core.Params); got != lit {
		t.Fatalf("got %+v, want %+v", got, lit)
	}
}
