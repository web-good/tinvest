package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GAZP на 1h. Замеры,
// обосновывающие каждое поле, — в доке пакета и в _comment гридов data/params/drop_bounce/gazp/.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      6,
		RSIZone:        30,
		RSICap:         50,
		EMAPeriod:      100,
		DaySpentATR:    0.7,
		StopDailyATR:   0.7,
		DailyATRPeriod: 14,
		RSIExit:        70,
		StartHour:      7,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		StaleBars:      10,
		StaleATR:       0.3,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
