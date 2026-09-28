package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

func TestRSIZoneBindingDefaults(t *testing.T) {
	b := RSIZoneLookupOrGeneric("SBER")
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

func TestRSIZoneParseParamsLayersOverDefaults(t *testing.T) {
	b := RSIZoneLookupOrGeneric("SBER")
	parsed, err := b.ParseParams([]byte(`{"RSIPeriod": 6, "StopDailyATR": 0.7}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.RSIPeriod != 6 || got.StopDailyATR != 0.7 {
		t.Fatalf("overrides not applied: %+v", got)
	}
	if got.EMAPeriod != core.DefaultParams().EMAPeriod {
		t.Fatalf("untouched field must keep default: EMAPeriod=%d", got.EMAPeriod)
	}
}

func TestRSIZoneParseParamsRejectsGarbage(t *testing.T) {
	b := RSIZoneLookupOrGeneric("SBER")
	if _, err := b.ParseParams([]byte(`not json`)); err == nil {
		t.Fatalf("want error on malformed JSON")
	}
}
