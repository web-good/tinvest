package ivat

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест IVAT на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         35,
		RSIFloor:       0,
		StochK:         10,
		StochSmooth:    2,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   50,
		RSITrailTo:     70,
		RSITrailStep:   20,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0.6,
		StopATR:        0.5,
		MaxStopPct:     0,
		TargetR:        1.5,
		TrendEMA:       100,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал IVAT изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
