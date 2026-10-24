package nvtk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест NVTK на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         30,
		RSIFloor:       0,
		RSITriggerMax:  0,
		RSIExit:        70,
		StochK:         10,
		StochSmooth:    3,
		StochD:         3,
		StochBars:      2,
		StochCrossMax:  40,
		VolumeMult:     0.5,
		VolBars:        20,
		MinRoomATR:     0.6,
		StopATR:        0.7,
		MaxStopPct:     1,
		TargetR:        3,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал NVTK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
