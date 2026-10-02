package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
)

func TestGapFadeBindingDefaults(t *testing.T) {
	b := GapFadeLookupOrGeneric("ZZZZ")
	got, ok := b.DefaultParams().(core.Params)
	if !ok || got != core.DefaultParams() {
		t.Fatalf("defaults = %#v, want core.DefaultParams()", b.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "ZZZZ" {
		t.Fatalf("ticker = %q, want ZZZZ", s.Ticker())
	}
}

func TestGapFadeParseParamsLayersOverDefaults(t *testing.T) {
	parsed, err := GapFadeLookupOrGeneric("ZZZZ").ParseParams([]byte(`{"GapATR": 0.7, "EntryBar": 1}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.GapATR != 0.7 || got.EntryBar != 1 || got.StopDailyATR != core.DefaultParams().StopDailyATR {
		t.Fatalf("got %+v", got)
	}
}
