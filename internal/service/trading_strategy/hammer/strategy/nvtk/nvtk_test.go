package nvtk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/hammer/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест NVTK на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      6,
		RSIZone:        50,
		MACDFast:       11,
		MACDSlow:       15,
		MACDSignal:     9,
		RSIHold:        0,
		TrendEMA:       100,
		VolumeMult:     0,
		VolBars:        20,
		ConfirmBars:    0,
		PDLZoneATR:     0,
		StopBufferATR:  0.05,
		TargetR:        1.5,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал NVTK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
