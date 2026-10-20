package nvtk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест NVTK на 15m.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		EntryPDL:       1,
		EntryPDH:       1,
		HoldBars:       3,
		StopDailyATR:   0.7,
		DailyATRPeriod: 14,
		VolumeMult:     1.5,
		VolBars:        20,
		MaxDayRangeATR: 1.2,
		IndexTrendEMA:  50,
		PDLEngulf:      0,
		PDLHammer:      1,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал NVTK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
