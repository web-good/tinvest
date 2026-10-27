package mgnt

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест MGNT на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  8,
		RSIArm:         30,
		RSIFloor:       0,
		StochK:         8,
		StochSmooth:    2,
		StochD:         3,
		RSIExit:        70,
		RSITrailFrom:   0,
		RSITrailTo:     70,
		RSITrailStep:   10,
		VolumeMult:     0.5,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.7,
		MaxStopPct:     0,
		TargetR:        1.5,
		TrendEMA:       170,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал MGNT изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
