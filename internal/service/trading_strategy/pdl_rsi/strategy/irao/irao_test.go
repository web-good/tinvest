package irao

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест IRAO на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  4,
		RSIArm:         25,
		RSIFloor:       0,
		StochK:         12,
		StochSmooth:    3,
		StochD:         3,
		RSIExit:        70,
		RSITrailFrom:   50,
		RSITrailTo:     70,
		RSITrailStep:   15,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.3,
		MaxStopPct:     1,
		TargetR:        1.5,
		TrendEMA:       170,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал IRAO изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
