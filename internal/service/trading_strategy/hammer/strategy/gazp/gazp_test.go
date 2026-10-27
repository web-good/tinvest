package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/hammer/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GAZP на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      9,
		RSIZone:        40,
		MACDFast:       11,
		MACDSlow:       15,
		MACDSignal:     9,
		RSIHold:        60,
		TrendEMA:       200,
		VolumeMult:     0,
		VolBars:        1,
		ConfirmBars:    5,
		PDLZoneATR:     0,
		StopBufferATR:  0.1,
		TargetR:        1.5,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
