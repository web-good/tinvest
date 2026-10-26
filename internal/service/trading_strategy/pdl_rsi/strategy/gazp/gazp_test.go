package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GAZP на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         30,
		RSIFloor:       22,
		StochK:         14,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        65,
		RSITrailFrom:   60,
		RSITrailTo:     80,
		RSITrailStep:   10,
		VolumeMult:     1.2,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        1,
		MaxStopPct:     1.5,
		TargetR:        2,
		TrendEMA:       150,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
