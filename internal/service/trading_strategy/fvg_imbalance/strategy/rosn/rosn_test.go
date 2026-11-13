package rosn

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestParams(t *testing.T) {
	want := core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.1,
		MaxWaitBars:               30,
		StochPeriod:               10,
		UseStochExit:              1,
		StopATR:                   1,
		TakeProfitATR:             0,
		HTFTrendEMA:               50,
		RequireEntryAboveFVGLower: 1,
	}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
