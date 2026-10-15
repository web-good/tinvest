package lkoh

import (
	"testing"

	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест LKOH на 30m. Замеры,
// обосновывающие каждое поле, — в доке пакета и в _comment гридов data/params/drop_bounce/lkoh/.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      5,
		RSIZone:        30,
		RSICap:         50,
		ZoneEntry:      1,
		EMAPeriod:      200,
		StopDailyATR:   1,
		BreakevenATR:   0.5,
		TrailATR:       0,
		DailyATRPeriod: 14,
		RSIExitPeriod:  8,
		RSIExit:        70,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		ConfirmBars:    3,
		FridayTo:       11,
		StaleBars:      0,
		StaleATR:       0.3,
		PrevCloseGate:  0,
		PrevCloseATR:   0,
		RSIFloor:       0,
		SkipMonday:     0,
		EMAGapATR:      0,
		MACDGreenOnly:  0,
		BreakevenPct:   0.1,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал LKOH изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
