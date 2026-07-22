package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping_rsimacd/strategy/core"
)

func TestScalpingRSIMACDBindingDefaults(t *testing.T) {
	b := ScalpingRSIMACDLookupOrGeneric("SBER")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != core.DefaultParams() {
		t.Fatalf("defaults = %+v\nwant %+v", got, core.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "SBER" {
		t.Fatalf("ticker = %q want SBER", s.Ticker())
	}
}

func TestScalpingRSIMACDParseParamsLayersOverDefaults(t *testing.T) {
	b := ScalpingRSIMACDLookupOrGeneric("SBER")
	raw := []byte(`{"RSIPeriod": 5, "RR": 3.0}`)
	parsed, err := b.ParseParams(raw)
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.RSIPeriod != 5 || got.RR != 3.0 {
		t.Fatalf("overrides not applied: %+v", got)
	}
	if got.MACDSlow != core.DefaultParams().MACDSlow {
		t.Fatalf("untouched field must keep its default: MACDSlow=%d", got.MACDSlow)
	}
}

func TestScalpingRSIMACDParseParamsRejectsGarbage(t *testing.T) {
	b := ScalpingRSIMACDLookupOrGeneric("SBER")
	if _, err := b.ParseParams([]byte(`not json`)); err == nil {
		t.Fatalf("want an error on malformed params JSON")
	}
}
