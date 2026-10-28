package alrs

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест ALRS на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         30,
		RSIFloor:       0,
		StochK:         11,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        65,
		RSITrailFrom:   40,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.3,
		MaxStopPct:     0,
		TargetR:        1.5,
		TrendEMA:       150,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал ALRS изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
