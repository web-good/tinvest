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
	got := b.DefaultParams().(core.Params)
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

// genericLevelsDefaults is the frozen neutral baseline for tickers without a dedicated
// levels config. It is intentionally decoupled from levelsrusal.DefaultParams so that
// calibrating RUAL never drifts the generic baseline. RUAL has since been recalibrated
// away from these starting values, so the two no longer coincide; this test pins the
// neutral baseline explicitly to catch accidental drift of the generic path itself.
func TestGenericLevelsDefaultsAreNeutralBaseline(t *testing.T) {
	want := core.Params{
		ProfileWindow:     120,
		ProfileBins:       40,
		HVNFactor:         1.5,
		ATRPeriod:         14,
		LevelTolATR:       0.5,
		RoomATR:           2.0,
		MaxExtensionATR:   1.0,
		SLMult:            1.0,
		SwingLowWindow:    10,
		TrailMult:         2.5,
		ChandelierWindow:  20,
		TrailArmATR:       1.0,
		BreakoutLookback:  10,
		TrendFilterPeriod: 50,
		TrendSlopeTol:     0.0,
		ConfirmCloseFrac:  0.5,
		MinATRFrac:        0.003,
		MinRR:             1.5,
	}
	if got := genericLevelsDefaults(); got != want {
		t.Fatalf("generic levels defaults = %+v, want neutral baseline %+v", got, want)
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
