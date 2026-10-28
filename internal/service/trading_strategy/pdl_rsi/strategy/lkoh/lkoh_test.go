package lkoh

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест LKOH на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  4,
		RSIArm:         35,
		RSIFloor:       0,
		StochK:         8,
		StochSmooth:    2,
		StochD:         3,
		RSIExit:        70,
		RSITrailFrom:   60,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        1,
		MaxStopPct:     1.5,
		TargetR:        3,
		TrendEMA:       100,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал LKOH изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
