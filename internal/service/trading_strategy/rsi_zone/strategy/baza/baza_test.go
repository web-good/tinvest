package baza

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест BAZA. Замеры,
// обосновывающие решение, — в доке пакета и в _comment гридов data/params/rsi_zone/baza/.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:        4,
		RSILower:         25,
		RSIUpper:         75,
		EMAPeriod:        200,
		DailyATRPeriod:   14,
		StopDailyATR:     1.0,
		ProfitExitBars:   3,
		ProfitExitPct:    0.5,
		EntryBlockFrom:   1000,
		EntryBlockTo:     1400,
		EntryBlockMonThu: 1,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал BAZA изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestParamsAreTheCoreBaselineByDesign заменяет у BAZA проверку «литерал не равен дефолтам»:
// калибровка 2026-09-29 заводит тикер на дефолтах ядра, потому что точка большинства проиграла им
// на всех трёх схемах; 2026-10-03 поверх них включён выход PROFIT (3 бара, 0.5%). Если дефолты ядра
// сменятся, тест упадёт — решение BAZA тогда нужно перепроверить, а не молча унаследовать новые
// дефолты.
func TestParamsAreTheCoreBaselineByDesign(t *testing.T) {
	want := core.DefaultParams()
	want.ProfitExitBars, want.ProfitExitPct = 3, 0.5
	want.EntryBlockFrom, want.EntryBlockTo, want.EntryBlockMonThu = 1000, 1400, 1
	if got := DefaultParams(); got != want {
		t.Fatalf("BAZA торгует дефолты ядра + выход PROFIT по решению 2026-10-03:\n got: %+v\nwant: %+v", got, want)
	}
}

// PROFIT включён решением 2026-10-03; ProfitExitPct 0 молча выключил бы выход.
func TestProfitExitIsArmed(t *testing.T) {
	if p := DefaultParams(); p.ProfitExitPct <= 0 {
		t.Fatalf("ProfitExitPct = %v, want > 0", p.ProfitExitPct)
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR отключает стоп в ядре.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
