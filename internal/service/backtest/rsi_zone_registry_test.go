package backtest

import (
	"testing"

	rsizoneafks "tinvest/internal/service/trading_strategy/rsi_zone/strategy/afks"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	rsizonedomrf "tinvest/internal/service/trading_strategy/rsi_zone/strategy/domrf"
	rsizonesber "tinvest/internal/service/trading_strategy/rsi_zone/strategy/sber"
)

func TestRSIZoneBindingDefaults(t *testing.T) {
	b := RSIZoneLookupOrGeneric("ZZZZ")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != core.DefaultParams() {
		t.Fatalf("defaults = %+v want %+v", got, core.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "ZZZZ" {
		t.Fatalf("ticker = %q want ZZZZ", s.Ticker())
	}
}

func TestRSIZoneParseParamsLayersOverDefaults(t *testing.T) {
	b := RSIZoneLookupOrGeneric("ZZZZ")
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
	b := RSIZoneLookupOrGeneric("ZZZZ")
	if _, err := b.ParseParams([]byte(`not json`)); err == nil {
		t.Fatalf("want error on malformed JSON")
	}
}

// A calibrated ticker must get its own literal, not the core baseline — both for plain runs and
// as the base that partial grid/params JSON layers over.
func TestRSIZoneRegisteredTickersUseTheirLiteral(t *testing.T) {
	for ticker, want := range map[string]core.Params{
		rsizoneafks.Ticker:  rsizoneafks.DefaultParams(),
		rsizonesber.Ticker:  rsizonesber.DefaultParams(),
		rsizonedomrf.Ticker: rsizonedomrf.DefaultParams(),
	} {
		b := RSIZoneLookupOrGeneric(ticker)
		if got := b.DefaultParams().(core.Params); got != want {
			t.Fatalf("%s defaults = %+v want %+v", ticker, got, want)
		}
		parsed, err := b.ParseParams([]byte(`{"RSIPeriod": 6}`))
		if err != nil {
			t.Fatalf("%s ParseParams: %v", ticker, err)
		}
		if got := parsed.(core.Params); got.RSIPeriod != 6 || got.EMAPeriod != want.EMAPeriod {
			t.Fatalf("%s partial JSON must layer over the ticker literal: %+v", ticker, got)
		}
		if s := b.Build(want); s.Ticker() != ticker {
			t.Fatalf("ticker = %q want %s", s.Ticker(), ticker)
		}
	}
}
