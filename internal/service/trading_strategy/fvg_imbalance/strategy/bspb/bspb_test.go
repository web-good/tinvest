package bspb

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestParams(t *testing.T) {
	want := core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               10,
		UseStochExit:              1,
		StopATR:                   1.5,
		TakeProfitATR:             1,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
