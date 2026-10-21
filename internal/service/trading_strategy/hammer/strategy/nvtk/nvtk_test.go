package nvtk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/hammer/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест NVTK на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       5,
		RSIZone:         30,
		RSIExit:         70,
		TrendEMA:        200,
		VolumeMult:      1.5,
		VolBars:         20,
		BelowEMA:        0,
		BelowRSIPeriod:  6,
		BelowRSIZone:    30,
		BelowRSIExit:    50,
		StopDailyATR:    0.5,
		TargetRangeMult: 0.5,
		DailyATRPeriod:  14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал NVTK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
