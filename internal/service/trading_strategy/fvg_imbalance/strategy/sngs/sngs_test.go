package sngs

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestParams(t *testing.T) {
	if got, want := Params(), (core.Params{ATRPeriod: 14, MinGapATR: 0.1, MaxWaitBars: 30, StochPeriod: 21, UseStochExit: 1, StopATR: 1.7, TakeProfitATR: 0, HTFTrendEMA: 120, RequireEntryAboveFVGLower: 1}); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
