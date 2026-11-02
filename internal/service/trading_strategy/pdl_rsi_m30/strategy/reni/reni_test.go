package reni

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi_m30/strategy/core"
)

func TestParamsAreIndependentM30Baseline(t *testing.T) {
	want := core.Params{
		RSIPeriod:      6,
		RSIArm:         25,
		RSIFloor:       0,
		RSIExit:        70,
		StochK:         13,
		StochSmooth:    3,
		StochD:         3,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.3,
		MaxStopPct:     0,
		TargetR:        3,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
