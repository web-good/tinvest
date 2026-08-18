package svav

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Пакет заведён 2026-08-18 ДО калибровки: он обязан отслеживать baseline, чтобы правка дефолтов
// доходила до тикера, а не расходилась с ним молча. Тест держит это состояние и подлежит замене
// снимком литерала ровно тогда, когда калибровка закончится (задача 11 плана).
func TestParamsTrackTheBaseline(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("SVAV ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSVAV(t *testing.T) {
	if Ticker != "SVAV" {
		t.Fatalf("Ticker = %q, want SVAV", Ticker)
	}
}
