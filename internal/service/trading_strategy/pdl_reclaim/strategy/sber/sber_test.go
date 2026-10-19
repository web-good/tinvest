package sber

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест SBER на 15m.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		EntryPDL:       1,
		EntryPDH:       1,
		HoldBars:       1,
		StopDailyATR:   1.5,
		DailyATRPeriod: 14,
		EntryFromHour:  10,
		EntryToHour:    16,
		VolumeMult:     1.5,
		VolBars:        20,
		MaxDayRangeATR: 1.2,
		IndexTrendEMA:  200,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал SBER изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
