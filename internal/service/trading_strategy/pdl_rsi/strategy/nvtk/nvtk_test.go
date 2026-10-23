package nvtk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест NVTK на 15m.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      6,
		RSIArm:         30,
		RSIEntry:       30,
		RSIGate:        50,
		RSIBreakeven:   50,
		RSIExit:        70,
		ZoneATR:        0.4,
		VolumeMult:     1.5,
		VolBars:        20,
		MinRoomATR:     0.6,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал NVTK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
