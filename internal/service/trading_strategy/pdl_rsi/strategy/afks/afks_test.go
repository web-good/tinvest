package afks

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест AFKS на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  8,
		RSIArm:         30,
		RSIFloor:       0,
		StochK:         14,
		StochSmooth:    1,
		StochD:         3,
		RSIExit:        70,
		RSITrailFrom:   0,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.3,
		MaxStopPct:     1,
		TargetR:        2,
		TrendEMA:       150,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал AFKS изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
