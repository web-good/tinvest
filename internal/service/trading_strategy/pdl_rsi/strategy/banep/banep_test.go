package banep

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест BANEP на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         30,
		RSIFloor:       0,
		StochK:         14,
		StochSmooth:    2,
		StochD:         3,
		RSIExit:        70,
		RSITrailFrom:   40,
		RSITrailTo:     70,
		RSITrailStep:   15,
		VolumeMult:     0.8,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.3,
		MaxStopPct:     1,
		TargetR:        2,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал BANEP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
