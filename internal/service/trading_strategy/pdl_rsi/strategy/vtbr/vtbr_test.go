package vtbr

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест VTBR на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		HourRSIPeriod:  8,
		RSIArm:         35,
		RSIFloor:       0,
		RSITriggerMax:  0,
		RSIExit:        70,
		StochK:         14,
		StochSmooth:    3,
		StochD:         3,
		StochBars:      2,
		StochCrossMax:  50,
		VolumeMult:     1.2,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.3,
		MaxStopPct:     1,
		TargetR:        3,
		TrendEMA:       0,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал VTBR изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
