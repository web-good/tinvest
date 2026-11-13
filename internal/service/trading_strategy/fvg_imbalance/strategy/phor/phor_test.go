package phor

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestParams(t *testing.T) {
	want := core.Params{ATRPeriod: 14, MinGapATR: 0.25, MaxWaitBars: 30, StochPeriod: 14, UseStochExit: 1, StopATR: 0.5, TakeProfitATR: 1.5, HTFTrendEMA: 0, RequireEntryAboveFVGLower: 0}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
