package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_ema/strategy/core"
)

func TestRSIEMABindingDefaults(t *testing.T) {
	b := RSIEMALookupOrGeneric("SBER")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != core.DefaultParams() {
		t.Fatalf("defaults = %+v want %+v", got, core.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "SBER" {
		t.Fatalf("ticker = %q want SBER", s.Ticker())
	}
}

func TestRSIEMAParseParamsLayersOverDefaults(t *testing.T) {
	b := RSIEMALookupOrGeneric("SBER")
	parsed, err := b.ParseParams([]byte(`{"RSIPeriod": 9, "EntryCooldownBars": 3}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.RSIPeriod != 9 || got.EntryCooldownBars != 3 {
		t.Fatalf("overrides not applied: %+v", got)
	}
	if got.EMASlow != core.DefaultParams().EMASlow {
		t.Fatalf("untouched field must keep default: EMASlow=%d", got.EMASlow)
	}
}

func TestRSIEMAParseParamsRejectsGarbage(t *testing.T) {
	b := RSIEMALookupOrGeneric("SBER")
	if _, err := b.ParseParams([]byte(`not json`)); err == nil {
		t.Fatalf("want error on malformed JSON")
	}
}
