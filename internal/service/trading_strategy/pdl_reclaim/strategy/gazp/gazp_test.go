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
		HoldBars:       1,
		StopDailyATR:   0.85,
		DailyATRPeriod: 14,
		EntryFromHour:  10,
		EntryToHour:    18,
		VolumeMult:     1.5,
		VolBars:        20,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
