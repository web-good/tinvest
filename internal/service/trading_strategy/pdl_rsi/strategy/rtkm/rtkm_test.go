package rtkm

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест RTKM на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  4,
		RSIArm:         25,
		RSIFloor:       22,
		StochK:         0,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        65,
		RSITrailFrom:   40,
		RSITrailTo:     70,
		RSITrailStep:   20,
		VolumeMult:     0.5,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.5,
		MaxStopPct:     1.5,
		TargetR:        3,
		TrendEMA:       100,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал RTKM изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
