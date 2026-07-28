package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/vwap_rev/strategy/core"
)

func TestVWAPRevBindingDefaults(t *testing.T) {
	b := VWAPRevLookupOrGeneric("GAZP")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != core.DefaultParams() {
		t.Fatalf("defaults = %+v want %+v", got, core.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "GAZP" {
		t.Fatalf("ticker = %q want GAZP", s.Ticker())
	}
}

func TestVWAPRevParseParamsLayersOverDefaults(t *testing.T) {
	b := VWAPRevLookupOrGeneric("GAZP")
	parsed, err := b.ParseParams([]byte(`{"EntryK": 2.5, "MaxHoldBars": 12}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.EntryK != 2.5 || got.MaxHoldBars != 12 {
		t.Fatalf("overrides not applied: %+v", got)
	}
	if got.MinEdgePct != core.DefaultParams().MinEdgePct {
		t.Fatalf("untouched field must keep default: MinEdgePct=%v", got.MinEdgePct)
	}
}

func TestVWAPRevParseParamsRejectsGarbage(t *testing.T) {
	b := VWAPRevLookupOrGeneric("GAZP")
	if _, err := b.ParseParams([]byte(`not json`)); err == nil {
		t.Fatalf("want error on malformed JSON")
	}
}
