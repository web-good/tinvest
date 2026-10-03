package lent

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест LENT. Замеры,
// обосновывающие решение, — в доке пакета и в _comment гридов data/params/rsi_zone/lent/.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      4,
		RSILower:       25,
		RSIUpper:       75,
		EMAPeriod:      200,
		DailyATRPeriod: 7,
		StopDailyATR:   1.0,
		UseStoch:       1,
		StochKPeriod:   14,
		StochDSmooth:   3,
		StochLower:     15,
		ZoneWindowBars: 1,
		ProfitExitBars: 0,
		ProfitExitPct:  0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал LENT изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Перекалибровка 2026-10-01 увела LENT с дефолтов ядра (гейт стохастика, DailyATRPeriod 7):
// литерал, совпавший с дефолтами, значит, что точка потерялась.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("литерал LENT совпал с core.DefaultParams() — точка перекалибровки 2026-10-01 потеряна")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR отключает стоп в ядре.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
