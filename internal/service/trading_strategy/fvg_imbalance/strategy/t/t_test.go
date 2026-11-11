package t

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestParams(t *testing.T) {
	want := core.Params{ATRPeriod: 14, MinGapATR: 0.35, MaxWaitBars: 40, StochPeriod: 12, UseStochExit: 0, StopATR: 0.6, TakeProfitATR: 2, HTFTrendEMA: 0, RequireEntryAboveFVGLower: 1}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
