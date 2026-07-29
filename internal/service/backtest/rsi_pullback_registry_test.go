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

// TestRSIPullbackRegistryEntriesMatchTheirTicker guards the copy-paste hazard of per-ticker
// packages: a map key must build a strategy labelled with that same ticker. A package cloned
// from a sibling with its Ticker constant left unchanged registers itself under the wrong key,
// and every report for that instrument would then carry the wrong label.
func TestRSIPullbackRegistryEntriesMatchTheirTicker(t *testing.T) {
	if len(rsiPullbackRegistry) == 0 {
		t.Fatal("rsi_pullback registry is empty")
	}
	for ticker, b := range rsiPullbackRegistry {
		t.Run(ticker, func(t *testing.T) {
			p, ok := b.DefaultParams().(core.Params)
			if !ok {
				t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
			}
			if got := b.Build(p).Ticker(); got != ticker {
				t.Fatalf("registered under %q but builds Ticker() = %q", ticker, got)
			}
		})
	}
}

// TestRSIPullbackRegistryKeepsTheStopArmed pins the one parameter no per-ticker package may
// relax: StopDailyATR = 0 disables the stop entirely, and this strategy holds positions across
// nights and weekends. A calibrated package that lands on zero would ship an unprotected long.
func TestRSIPullbackRegistryKeepsTheStopArmed(t *testing.T) {
	for ticker, b := range rsiPullbackRegistry {
		p, ok := b.DefaultParams().(core.Params)
		if !ok {
			t.Fatalf("%s: DefaultParams() returned %T, want core.Params", ticker, b.DefaultParams())
		}
		if p.StopDailyATR <= 0 {
			t.Fatalf("%s: StopDailyATR = %v, want > 0 — a multi-day hold must never run without a stop",
				ticker, p.StopDailyATR)
		}
	}
}

// TestRSIPullbackUnknownTickerFallsBackToGeneric pins that an unregistered ticker still runs,
// on the baseline params, rather than failing or silently borrowing another ticker's config.
func TestRSIPullbackUnknownTickerFallsBackToGeneric(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("NOSUCH")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("DefaultParams() = %+v, want the baseline %+v", p, core.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "NOSUCH" {
		t.Fatalf("Ticker() = %q, want NOSUCH", got)
	}
}

func TestRSIPullbackParseParamsRejectsGarbage(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	if _, err := b.ParseParams([]byte(`{"RSILower":`)); err == nil {
		t.Fatal("ParseParams accepted malformed JSON, want an error")
	}
}
