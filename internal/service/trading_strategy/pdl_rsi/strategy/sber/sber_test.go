package sber

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест SBER на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  5,
		RSIArm:         25,
		RSIFloor:       0,
		StochK:         13,
		StochSmooth:    3,
		StochD:         3,
		RSIExit:        70,
		RSITrailFrom:   50,
		RSITrailTo:     70,
		RSITrailStep:   20,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        1,
		MaxStopPct:     1.5,
		TargetR:        2,
		TrendEMA:       100,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал SBER изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
