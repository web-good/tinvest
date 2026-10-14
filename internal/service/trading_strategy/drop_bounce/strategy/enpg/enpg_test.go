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
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		RSIExitPeriod:  10,
		RSIExit:        70,
		MorningFrom:    7,
		MorningTo:      11,
		EveningFrom:    17,
		EveningTo:      24,
		FridayTo:       11,
		StaleBars:      0,
		StaleATR:       0.3,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал ENPG изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
