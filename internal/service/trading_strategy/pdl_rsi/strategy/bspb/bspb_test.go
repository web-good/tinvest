package bspb

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест BSPB на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  5,
		RSIArm:         35,
		RSIFloor:       0,
		StochK:         9,
		StochSmooth:    3,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   50,
		RSITrailTo:     70,
		RSITrailStep:   20,
		VolumeMult:     0.8,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.5,
		MaxStopPct:     1,
		TargetR:        1.5,
		TrendEMA:       150,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал BSPB изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
