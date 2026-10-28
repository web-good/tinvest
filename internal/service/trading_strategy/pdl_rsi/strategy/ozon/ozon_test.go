package ozon

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест OZON на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  8,
		RSIArm:         25,
		RSIFloor:       0,
		StochK:         14,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   50,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0.5,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.5,
		MaxStopPct:     0,
		TargetR:        3,
		TrendEMA:       150,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал OZON изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
