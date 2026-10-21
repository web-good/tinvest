package nvtk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_zone/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест NVTK на 5m.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		EntryZone:      1,
		EntryBreak:     1,
		ZoneATR:        0.2,
		MACDFast:       8,
		MACDSlow:       21,
		MACDSignal:     9,
		VolumeMult:     1.5,
		VolBars:        15,
		StopBars:       5,
		MinStopATR:     0.2,
		TargetR:        2,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал NVTK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
