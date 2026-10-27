package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GAZP на 15m.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		EntryPDL:          1,
		EntryPDH:          0,
		HoldBars:          3,
		StopDailyATR:      0.7,
		DailyATRPeriod:    14,
		VolumeMult:        1.5,
		VolBars:           20,
		MaxDayRangeATR:    1.2,
		IndexTrendEMA:     150,
		PDLEngulf:         0,
		PDLHammer:         1,
		PDLDipVolMult:     0.5,
		PDLReclaimBars:    0,
		PDLHammerEntry:    0,
		PDLSignalVolMult:  0.5,
		PDLPatternTargetR: 1.5,
		PDLMinStopATR:     0.4,
		LockAfterR:        1,
		LockAtR:           0.25,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
