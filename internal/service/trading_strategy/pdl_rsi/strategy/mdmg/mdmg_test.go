package mdmg

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест MDMG на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  4,
		RSIArm:         35,
		RSIFloor:       0,
		RSITriggerMax:  50,
		RSIExit:        70,
		StochK:         5,
		StochSmooth:    3,
		StochD:         3,
		StochBars:      4,
		StochCrossMax:  50,
		VolumeMult:     1.2,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        1,
		MaxStopPct:     1.5,
		TargetR:        1.5,
		TrendEMA:       0,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал MDMG изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
