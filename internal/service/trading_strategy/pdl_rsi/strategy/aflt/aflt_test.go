package aflt

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест AFLT на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         30,
		RSIFloor:       22,
		StochK:         0,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   60,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0.5,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.7,
		MaxStopPct:     0,
		TargetR:        3,
		TrendEMA:       100,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал AFLT изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
