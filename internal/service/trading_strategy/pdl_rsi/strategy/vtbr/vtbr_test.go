package vtbr

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест VTBR на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  5,
		RSIArm:         35,
		RSIFloor:       0,
		StochK:         0,
		StochSmooth:    3,
		StochD:         3,
		RSIExit:        70,
		RSITrailFrom:   0,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.5,
		MaxStopPct:     0,
		TargetR:        2,
		TrendEMA:       100,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал VTBR изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
