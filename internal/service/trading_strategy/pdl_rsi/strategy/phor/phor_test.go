package phor

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест PHOR на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         30,
		RSIFloor:       0,
		StochK:         0,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        65,
		RSITrailFrom:   60,
		RSITrailTo:     70,
		RSITrailStep:   20,
		VolumeMult:     1.2,
		VolBars:        20,
		MinRoomATR:     0.6,
		StopATR:        0.7,
		MaxStopPct:     1.5,
		TargetR:        1.5,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал PHOR изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
