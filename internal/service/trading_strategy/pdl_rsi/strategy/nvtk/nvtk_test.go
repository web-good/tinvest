package nvtk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест NVTK на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  5,
		RSIArm:         30,
		RSIFloor:       0,
		StochK:         14,
		StochSmooth:    2,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   60,
		RSITrailTo:     70,
		RSITrailStep:   15,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.7,
		MaxStopPct:     1.5,
		TargetR:        3,
		TrendEMA:       150,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал NVTK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
