package etln

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест ETLN на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  4,
		RSIArm:         25,
		RSIFloor:       0,
		StochK:         0,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   40,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0.8,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.7,
		MaxStopPct:     0,
		TargetR:        1.5,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал ETLN изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
