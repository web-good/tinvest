package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestGAZPParams(t *testing.T) {
	want := core.Params{ATRPeriod: 14, MinGapATR: 0.8, MaxRetestBars: 12, MinRetestDepthPct: 50, TakeProfitR: 1, SwingK: 2, StructureLookbackBars: 50, MaxSweepToFVGBars: 15, HTFTrendEMA: 150, StopBufferATR: 1}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
