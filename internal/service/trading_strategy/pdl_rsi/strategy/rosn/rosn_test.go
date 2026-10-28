package rosn

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест ROSN на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  8,
		RSIArm:         30,
		RSIFloor:       22,
		StochK:         8,
		StochSmooth:    3,
		StochD:         3,
		RSIExit:        70,
		RSITrailFrom:   50,
		RSITrailTo:     70,
		RSITrailStep:   20,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0.6,
		StopATR:        0.3,
		MaxStopPct:     1.5,
		TargetR:        3,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал ROSN изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
