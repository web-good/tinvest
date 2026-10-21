package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/hammer/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GAZP на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      5,
		RSIZone:        30,
		MACDFast:       11,
		MACDSlow:       15,
		MACDSignal:     9,
		TrendEMA:       200,
		VolumeMult:     1.5,
		VolBars:        20,
		StopBufferATR:  0.05,
		TargetR:        2,
		DailyATRPeriod: 14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
