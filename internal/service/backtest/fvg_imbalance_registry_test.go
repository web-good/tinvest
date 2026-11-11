package backtest

import (
	"testing"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/domrf"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/gazp"
)

func TestFVGImbalanceDefaultsAndOverrides(t *testing.T) {
	for _, ticker := range []string{"GAZP", "DOMRF", "ZZZZ"} {
		b, err := FVGImbalanceLookupOrGeneric(ticker, enum.Minutes30)
		if err != nil {
			t.Fatal(err)
		}
		wantDefaults := core.DefaultParams()
		if ticker == "GAZP" {
			wantDefaults = gazp.Params()
		}
		if ticker == "DOMRF" {
			wantDefaults = domrf.Params()
		}
		if b.DefaultParams() != wantDefaults {
			t.Fatalf("%s defaults: %+v, want %+v", ticker, b.DefaultParams(), wantDefaults)
		}
		got, err := b.ParseParams([]byte(`{"StochPeriod":9,"MaxWaitBars":20,"StopATR":0.5,"TakeProfitATR":3}`))
		if err != nil {
			t.Fatal(err)
		}
		p := got.(core.Params)
		if p.StochPeriod != 9 || p.MaxWaitBars != 20 || p.StopATR != .5 || p.TakeProfitATR != 3 {
			t.Fatalf("%+v", p)
		}
		for _, raw := range []string{`{"MaxWaitBars":0}`, `{"StochPeriod":0}`, `{"StopATR":0}`, `{"TakeProfitATR":-1}`, `{"TakeProfitR":2}`, `{"MinRetestDepthPct":50}`, `{"Unknown":1}`} {
			if _, err := b.ParseParams([]byte(raw)); err == nil {
				t.Errorf("accepted %s", raw)
			}
		}
	}
	if _, err := FVGImbalanceLookupOrGeneric("GAZP", enum.Hour1); err == nil {
		t.Fatal("non-M30 accepted")
	}
}
