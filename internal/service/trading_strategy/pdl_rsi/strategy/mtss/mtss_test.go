package mtss

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест MTSS на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  5,
		RSIArm:         25,
		RSIFloor:       0,
		StochK:         9,
		StochSmooth:    2,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   0,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     1.2,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        1,
		MaxStopPct:     0,
		TargetR:        3,
		TrendEMA:       170,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал MTSS изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
