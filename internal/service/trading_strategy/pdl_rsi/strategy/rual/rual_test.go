package rual

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест RUAL на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         30,
		RSIFloor:       22,
		StochK:         0,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   40,
		RSITrailTo:     70,
		RSITrailStep:   20,
		VolumeMult:     1.2,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.5,
		MaxStopPct:     0,
		TargetR:        2,
		TrendEMA:       100,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал RUAL изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
