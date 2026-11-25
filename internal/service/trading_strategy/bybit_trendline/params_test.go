package bybit_trendline

import "testing"

func TestParamsFor(t *testing.T) {
	for symbol, want := range map[string]float64{"ETHUSDT": 0.7, "BTCUSDT": 1} {
		if ParamsFor(symbol).RequirePreviousDayHigh {
			t.Errorf("%s: incorrect previous day high setting", symbol)
		}
		if got := ParamsFor(symbol).InPlayMinRangePct; got != want {
			t.Errorf("%s: got %g, want %g", symbol, got, want)
		}
	}
}
