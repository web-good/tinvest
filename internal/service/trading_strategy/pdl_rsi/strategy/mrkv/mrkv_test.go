package mrkv

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест MRKV на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  8,
		RSIArm:         25,
		RSIFloor:       0,
		StochK:         0,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        65,
		RSITrailFrom:   40,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0.5,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        1,
		MaxStopPct:     0,
		TargetR:        2,
		TrendEMA:       170,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал MRKV изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
