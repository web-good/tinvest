package nvtk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест NVTK на 15m.
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
		H1TrendEMA:        0,
		PDLEngulf:         0,
		PDLHammer:         1,
		PDLDipVolMult:     0.8,
		PDLReclaimBars:    0,
		PDLPatternTargetR: 1.5,
		PDLExitBelowPDL:   1,
		PDLExitBufferATR:  0.2,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал NVTK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
