package bybit_trendline

import (
	"testing"

	"tinvest/internal/service/trading_strategy/bybit_trendline/strategy/core"
)

func TestParamsFor(t *testing.T) {
	for symbol, want := range map[string]core.Params{
		"ETHUSDT": {TakeProfitR: 5, InPlayMinRangePct: 0.1, RequirePreviousDayHigh: true},
		"BTCUSDT": {TakeProfitR: 2, InPlayMinRangePct: 1},
	} {
		if got := ParamsFor(symbol); got != want {
			t.Errorf("%s: got %+v, want %+v", symbol, got, want)
		}
	}
}
