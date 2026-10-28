package enpg

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест ENPG на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  6,
		RSIArm:         25,
		RSIFloor:       0,
		StochK:         12,
		StochSmooth:    3,
		StochD:         3,
		RSIExit:        75,
		RSITrailFrom:   50,
		RSITrailTo:     70,
		RSITrailStep:   20,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        1,
		MaxStopPct:     0,
		TargetR:        1.5,
		TrendEMA:       150,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал ENPG изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
