package gmkn

import (
	"testing"

	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест GMKN на 30m. Замеры,
// обосновывающие каждое поле, — в доке пакета и в _comment гридов data/params/drop_bounce/gmkn/.
func TestLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      4,
		RSIZone:        30,
		RSICap:         50,
		ZoneEntry:      1,
		EMAPeriod:      200,
		StopDailyATR:   0.7,
		DailyATRPeriod: 14,
		RSIExitPeriod:  11,
		RSIExit:        70,
		HourMACDFast:   8,
		HourMACDSlow:   26,
		HourMACDSignal: 5,
		HourMACDHours:  20,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		ConfirmBars:    6,
		FridayTo:       11,
	}
	if got := Params(); got != want {
		t.Fatalf("литерал GMKN изменился:\n got: %+v\nwant: %+v", got, want)
	}
}
