package msng

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест MSNG. Замеры,
// обосновывающие каждое поле, — в доке пакета и в _comment гридов data/params/rsi_zone/msng/.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:        4,
		RSILower:         25,
		RSIUpper:         75,
		EMAPeriod:        50,
		DailyATRPeriod:   14,
		StopDailyATR:     1.0,
		ProfitExitBars:   0,
		ProfitExitPct:    0,
		EntryBlockFrom:   1000,
		EntryBlockTo:     1400,
		EntryBlockMonThu: 1,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал MSNG изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Откалиброванный тикер не должен отслеживать baseline: иначе смена дефолтов ядра молча
// поменяет и его.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("MSNG вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR отключает стоп в ядре.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
