package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestGAZPParams(t *testing.T) {
	want := core.Params{ATRPeriod: 14, MinGapATR: 0.6, MaxRetestBars: 10, MinRetestDepthPct: 80, TakeProfitR: 2, RequireThirdCandleHigherHigh: 0, RetestDepthByBody: 0, UseStochFilter: 0, HTFTrendEMA: 200, StopBufferATR: 1}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
