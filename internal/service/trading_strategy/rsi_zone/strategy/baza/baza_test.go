package baza

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест BAZA. Замеры,
// обосновывающие решение, — в доке пакета и в _comment гридов data/params/rsi_zone/baza/.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      4,
		RSILower:       25,
		RSIUpper:       75,
		EMAPeriod:      200,
		DailyATRPeriod: 14,
		StopDailyATR:   1.0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал BAZA изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestParamsAreTheCoreBaselineByDesign заменяет у BAZA проверку «литерал не равен дефолтам»:
// калибровка 2026-09-29 заводит тикер на дефолтах ядра, потому что точка большинства проиграла им
// на всех трёх схемах. Если дефолты ядра сменятся, тест упадёт — решение BAZA тогда нужно
// перепроверить, а не молча унаследовать новые дефолты.
func TestParamsAreTheCoreBaselineByDesign(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("BAZA торгует дефолты ядра по решению 2026-09-29:\n got: %+v\nwant: %+v", got, want)
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR отключает стоп в ядре.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
