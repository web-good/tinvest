package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GAZP на 5m.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		EntryA:         1,
		EntryB:         1,
		StopDailyATR:   0.5,
		DailyATRPeriod: 14,
		TakeProfitPDH:  1,
		RSIPeriod:      6,
		RSIExit:        70,
		RSIEntryMax:    50,
		MACDFast:       9,
		MACDSlow:       26,
		MACDSignal:     8,
		EntryFromHour:  7,
		EntryToHour:    18,
		ExitHour:       23,
		ExitMinute:     45,
		SwitchHour:     10,
		Entry5m:        1,
		VolDays:        20,
		HourRSIPeriod:  6,
		HourRSIMax:     30,
		EntryC:         1,
		BodyATR:        0.1,
		LevelsOnly:     1,
		LevelTolATR:    0.1,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
