package mrku

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест MRKU на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         30,
		RSIFloor:       0,
		StochK:         14,
		StochSmooth:    3,
		StochD:         3,
		RSIExit:        65,
		RSITrailFrom:   0,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0.8,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.5,
		MaxStopPct:     1,
		TargetR:        3,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал MRKU изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
