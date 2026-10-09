package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GAZP.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		MinDropPct:  0.6,
		RSIPeriod:   6,
		RSIEntry:    30,
		RSIExit:     70,
		MACDExit:    1,
		MACDFast:    9,
		MACDSlow:    14,
		MACDSignal:  3,
		StochPeriod: 14,
		StochSmooth: 3,
		StochExit:   80,
		ProfitPct:   1,
		StopPct:     1,
		StartHour:   8,
		EMAPeriod:   200,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
