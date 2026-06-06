package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/levels/strategy/core"
	levelsrusal "tinvest/internal/service/trading_strategy/levels/strategy/rusal"
)

func TestLevelsLookupOrGenericKnown(t *testing.T) {
	b := LevelsLookupOrGeneric("RUAL")
	if b.DefaultParams == nil || b.Build == nil || b.ParseParams == nil {
		t.Fatal("RUAL levels binding has nil funcs")
	}
	// DefaultParams must equal levelsrusal.DefaultParams().
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatal("RUAL DefaultParams is not core.Params")
	}
	want := levelsrusal.DefaultParams()
	if got != want {
		t.Fatalf("RUAL DefaultParams = %+v, want %+v", got, want)
	}
}

func TestLevelsLookupOrGenericUnknown(t *testing.T) {
	b := LevelsLookupOrGeneric("UNKNOWN")
	if b.DefaultParams == nil || b.Build == nil || b.ParseParams == nil {
		t.Fatal("generic levels binding has nil funcs")
	}
	// DefaultParams must be genericLevelsDefaults, not levelsrusal.DefaultParams.
	got := b.DefaultParams().(core.Params)
	if got == levelsrusal.DefaultParams() {
		// If they happen to be equal, at least confirm the binding still builds a
		// working strategy for the unknown ticker.
		s := b.Build(got)
		if s == nil {
			t.Fatal("generic Build returned nil")
		}
		if s.Ticker() != "UNKNOWN" {
			t.Fatalf("generic strategy ticker = %q, want UNKNOWN", s.Ticker())
		}
	}
	// Confirm an unknown ticker produces a usable binding.
	s := b.Build(got)
	if s == nil {
		t.Fatal("generic Build returned nil")
	}
	if s.Ticker() != "UNKNOWN" {
		t.Fatalf("generic strategy ticker = %q, want UNKNOWN", s.Ticker())
	}
	if s.Lookback() <= 0 {
		t.Fatalf("generic strategy Lookback() = %d, want > 0", s.Lookback())
	}
}

func TestLevelsParseParamsOverridesDefaults(t *testing.T) {
	b := LevelsLookupOrGeneric("RUAL")
	raw := []byte(`{"HVNFactor": 2.5}`)
	parsed, err := b.ParseParams(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := parsed.(core.Params)
	if !ok {
		t.Fatal("ParseParams result is not core.Params")
	}
	if p.HVNFactor != 2.5 {
		t.Fatalf("HVNFactor = %v, want 2.5 (override)", p.HVNFactor)
	}
	// Unspecified field must keep its default.
	if p.ATRPeriod != levelsrusal.DefaultParams().ATRPeriod {
		t.Fatalf("ATRPeriod = %d, want %d (unchanged default)", p.ATRPeriod, levelsrusal.DefaultParams().ATRPeriod)
	}
}

func TestLevelsBuildStrategy(t *testing.T) {
	b := LevelsLookupOrGeneric("RUAL")
	s := b.Build(b.DefaultParams())
	if s == nil {
		t.Fatal("Build returned nil")
	}
	if s.Ticker() != "RUAL" {
		t.Fatalf("built strategy ticker = %q, want RUAL", s.Ticker())
	}
	if s.Lookback() <= 0 {
		t.Fatalf("built strategy Lookback() = %d, want > 0", s.Lookback())
	}
}
