package backtest

import (
	"testing"

	reversionafks "tinvest/internal/service/trading_strategy/reversion/strategy/afks"
	"tinvest/internal/service/trading_strategy/reversion/strategy/core"
	reversionrusal "tinvest/internal/service/trading_strategy/reversion/strategy/rusal"
)

func TestReversionLookupRegisteredRUAL(t *testing.T) {
	b := ReversionLookupOrGeneric("RUAL")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != reversionrusal.DefaultParams() {
		t.Fatalf("RUAL defaults = %+v\nwant %+v", got, reversionrusal.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "RUAL" {
		t.Fatalf("ticker=%q want RUAL", s.Ticker())
	}
}

func TestReversionLookupRegisteredAFKS(t *testing.T) {
	b := ReversionLookupOrGeneric("AFKS")
	got := b.DefaultParams().(core.Params)
	if got != reversionafks.DefaultParams() {
		t.Fatalf("AFKS defaults mismatch")
	}
}

func TestReversionLookupGenericFallback(t *testing.T) {
	b := ReversionLookupOrGeneric("UNKNOWN")
	if s := b.Build(b.DefaultParams()); s.Ticker() != "UNKNOWN" {
		t.Fatalf("ticker=%q want UNKNOWN", s.Ticker())
	}
	// ParseParams must layer the override on top of genericReversionDefaults.
	got, err := b.ParseParams([]byte(`{"ATRMult": 2.0}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.ATRMult != 2.0 {
		t.Fatalf("ATRMult=%v want 2.0 (override)", p.ATRMult)
	}
	if p.FastEMA != 50 || p.SlowEMA != 200 {
		t.Fatalf("generic defaults not preserved: FastEMA=%d SlowEMA=%d want 50/200", p.FastEMA, p.SlowEMA)
	}
}

func TestReversionDefaultsValid(t *testing.T) {
	if p := genericReversionDefaults(); p.ATRMult <= 0 || p.SlowEMA <= p.FastEMA || p.RSIPeriod <= 0 {
		t.Fatalf("invalid generic defaults: %+v", p)
	}
}
