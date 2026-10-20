package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GAZP на 15m.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		EntryPDL:       1,
		EntryPDH:       0,
		HoldBars:       3,
		StopDailyATR:   1.0,
		DailyATRPeriod: 14,
		VolumeMult:     1.5,
		VolBars:        20,
		MaxDayRangeATR: 0.8,
		IndexTrendEMA:  200,
		PDLEngulf:      0,
		PDLHammer:      0,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
