package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/hammer/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GAZP на 1h.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       5,
		RSIZone:         30,
		RSIExit:         70,
		TrendEMA:        200,
		StopDailyATR:    0.5,
		TargetRangeMult: 0.5,
		DailyATRPeriod:  14,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
