package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestGAZPParams(t *testing.T) {
	want := core.Params{ATRPeriod: 14, MinGapATR: 0.8, MaxRetestBars: 8, RequireCloseAboveZone: 0, TakeProfitR: 1.5, SwingK: 2, StructureLookbackBars: 48, MaxSweepToFVGBars: 8}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
