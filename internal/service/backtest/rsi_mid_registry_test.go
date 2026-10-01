package backtest

import (
	"testing"

	"tinvest/internal/enum"
	midcore "tinvest/internal/service/trading_strategy/rsi_mid/strategy/core"
)

func TestRSIMidBindingDefaults(t *testing.T) {
	b := RSIMidLookupOrGeneric("ZZZZ", enum.Minutes30)
	got, ok := b.DefaultParams().(midcore.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want midcore.Params", b.DefaultParams())
	}
	if got != midcore.DefaultParams() {
		t.Fatalf("defaults = %+v want %+v", got, midcore.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "ZZZZ" {
		t.Fatalf("ticker = %q want ZZZZ", s.Ticker())
	}
}

func TestRSIMidParseParamsLayersOverDefaults(t *testing.T) {
	b := RSIMidLookupOrGeneric("ZZZZ", enum.Minutes15)
	parsed, err := b.ParseParams([]byte(`{"RSIPeriod": 9, "BelowMidBars": 4}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(midcore.Params)
	if got.RSIPeriod != 9 || got.BelowMidBars != 4 {
		t.Fatalf("overrides not applied: %+v", got)
	}
	if got.EMAPeriod != midcore.DefaultParams().EMAPeriod {
		t.Fatalf("untouched field must keep default: EMAPeriod=%d", got.EMAPeriod)
	}
}

func TestRSIMidParseParamsRejectsGarbage(t *testing.T) {
	if _, err := RSIMidLookupOrGeneric("ZZZZ", enum.Minutes30).ParseParams([]byte(`not json`)); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}

// A literal is bound to its (ticker, interval) pair: the same ticker on another timeframe falls
// back to the core defaults, never to the literal of a different timeframe.
func TestRSIMidRegistryIsKeyedByTickerAndInterval(t *testing.T) {
	lit := midcore.DefaultParams()
	lit.RSIPeriod, lit.EMAPeriod = 7, 50
	reg := buildRSIMidRegistry(map[string]map[enum.Interval]midcore.Params{
		"FAKE": {enum.Minutes30: lit},
	})

	b := rsiMidLookupIn(reg, "FAKE", enum.Minutes30)
	if got := b.DefaultParams().(midcore.Params); got != lit {
		t.Fatalf("FAKE m30 defaults = %+v want the literal %+v", got, lit)
	}
	parsed, err := b.ParseParams([]byte(`{"RSIUpper": 80}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	if got := parsed.(midcore.Params); got.RSIUpper != 80 || got.RSIPeriod != 7 || got.EMAPeriod != 50 {
		t.Fatalf("partial JSON must layer over the literal: %+v", got)
	}
	if s := b.Build(lit); s.Ticker() != "FAKE" {
		t.Fatalf("ticker = %q want FAKE", s.Ticker())
	}

	if got := rsiMidLookupIn(reg, "FAKE", enum.Minutes15).DefaultParams().(midcore.Params); got != midcore.DefaultParams() {
		t.Fatalf("FAKE m15 must fall back to core defaults, got %+v", got)
	}
}
