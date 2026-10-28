package uwgn

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест UWGN на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  4,
		RSIArm:         25,
		RSIFloor:       22,
		StochK:         0,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   40,
		RSITrailTo:     70,
		RSITrailStep:   20,
		VolumeMult:     0.5,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        1,
		MaxStopPct:     0,
		TargetR:        3,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал UWGN изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
