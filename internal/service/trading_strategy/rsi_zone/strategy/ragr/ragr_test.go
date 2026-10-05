package ragr

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// TestParamsAreTheCoreBaselineByDesign пинит РЕШЕНИЕ, а не промежуточное состояние: калибровка
// RAGR 2026-10-05 прошла два круга и не нашла ни одного поля с большинством голосов, которое
// пережило бы стоп-условие (точка первого круга с RSIUpper 55 провалила пункты 1, 3–6), а дефолты
// ядра прошли все семь пунктов. Замена литерала «по аналогии с соседями» была бы подгонкой без
// подтверждения темой. Если дефолты ядра изменятся, RAGR должен быть перекалиброван, а не
// молча унаследовать их, поэтому снимок ниже записан числами. Разбор — doc-комментарий пакета.
func TestParamsAreTheCoreBaselineByDesign(t *testing.T) {
	want := core.Params{
		RSIPeriod:      4,
		RSILower:       25,
		RSIUpper:       75,
		EMAPeriod:      200,
		DailyATRPeriod: 14,
		StopDailyATR:   1.0,
		ProfitExitBars: 0,
		ProfitExitPct:  0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал RAGR изменился без калибровки:\n got: %+v\nwant: %+v", got, want)
	}
	if got := core.DefaultParams(); got != want {
		t.Fatalf("дефолты ядра изменились — RAGR заведён на дефолтах 2026-10-05, перекалибруй:\n got: %+v\nwant: %+v", got, want)
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR отключает стоп в ядре.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
