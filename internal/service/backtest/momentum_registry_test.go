package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/momentum/strategy/core"
)

func TestMomentumLookupRegisteredRUAL(t *testing.T) {
	b := MomentumLookupOrGeneric("RUAL")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if p.MACDBelowZeroOnly != 1 {
		t.Fatalf("RUAL MACDBelowZeroOnly=%d want 1", p.MACDBelowZeroOnly)
	}
	if s := b.Build(p); s.Ticker() != "RUAL" {
		t.Fatalf("ticker=%q want RUAL", s.Ticker())
	}
}

func TestMomentumLookupGenericFallback(t *testing.T) {
	b := MomentumLookupOrGeneric("UNKNOWN")
	if s := b.Build(b.DefaultParams()); s.Ticker() != "UNKNOWN" {
		t.Fatalf("ticker=%q want UNKNOWN", s.Ticker())
	}
}

func TestMomentumParseParamsPartialOverride(t *testing.T) {
	b := MomentumLookupOrGeneric("RUAL")
	got, err := b.ParseParams([]byte(`{"TakeProfitRR": 3.0}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.TakeProfitRR != 3.0 {
		t.Fatalf("TakeProfitRR=%f want 3.0 (override)", p.TakeProfitRR)
	}
	if p.MACDSlow != 26 {
		t.Fatalf("MACDSlow=%d want 26 (default kept)", p.MACDSlow)
	}
}
