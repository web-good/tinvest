package dias

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Пакет заведён 2026-08-22 ДО калибровки: он обязан отслеживать baseline, чтобы правка дефолтов
// доходила до тикера, а не расходилась с ним молча. Тест держит это состояние и подлежит замене
// снимком литерала ровно тогда, когда калибровка закончится (задача 12 плана).
func TestParamsTrackTheBaseline(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("DIAS ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsDIAS(t *testing.T) {
	if Ticker != "DIAS" {
		t.Fatalf("Ticker = %q, want DIAS", Ticker)
	}
}
