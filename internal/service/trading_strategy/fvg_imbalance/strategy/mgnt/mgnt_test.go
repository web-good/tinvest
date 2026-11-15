package mgnt

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestParams(t *testing.T) {
	want := core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               40,
		StochPeriod:               9,
		UseStochExit:              0,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
