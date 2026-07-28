package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

func TestRSIPullbackBindingBuildsForTicker(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("DefaultParams() = %+v, want %+v", p, core.DefaultParams())
	}
	s := b.Build(p)
	if s.Ticker() != "GAZP" {
		t.Fatalf("Ticker() = %q, want GAZP", s.Ticker())
	}
	if s.Lookback() < 220 {
		t.Fatalf("Lookback() = %d, want >= 220", s.Lookback())
	}
}

func TestRSIPullbackParseParamsLayersOverDefaults(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	got, err := b.ParseParams([]byte(`{"RSILower": 10}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.RSILower != 10 {
		t.Fatalf("RSILower = %v, want the JSON value 10", p.RSILower)
	}
	if p.RSIUpper != core.DefaultParams().RSIUpper {
		t.Fatalf("RSIUpper = %v, want the default %v (partial JSON must not zero other fields)",
			p.RSIUpper, core.DefaultParams().RSIUpper)
	}
}

func TestRSIPullbackParseParamsRejectsGarbage(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	if _, err := b.ParseParams([]byte(`{"RSILower":`)); err == nil {
		t.Fatal("ParseParams accepted malformed JSON, want an error")
	}
}
