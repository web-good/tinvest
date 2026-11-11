package sber

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestParams(t *testing.T) {
	want := core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               45,
		StochPeriod:               10,
		UseStochExit:              0,
		StopATR:                   2,
		TakeProfitATR:             2,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
