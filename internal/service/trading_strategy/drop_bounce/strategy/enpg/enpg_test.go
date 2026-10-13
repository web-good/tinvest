package enpg

import (
	"testing"

	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест ENPG на 1h. Замеры,
// обосновывающие каждое поле, — в доке пакета и в _comment гридов data/params/drop_bounce/enpg/.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      6,
		RSIZone:        30,
		RSICap:         50,
		EMAPeriod:      200,
		DaySpentATR:    0.6,
		StopDailyATR:   1.03,
		DailyATRPeriod: 14,
		RSIExit:        66,
		StartHour:      7,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал ENPG изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
