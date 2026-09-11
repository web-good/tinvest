package irkt

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ЧЕСТНОЕ состояние: пакет strategy/irkt заведён
// 2026-09-11, чтобы прогоны шли через общий реестр, а не через generic-ветку, и до тех пор, пока
// калибровка не проведена (если она проведена вообще — маршрут TGKA на дефолтах уже закрыт по
// пункту 6 стоп-условия, см. doc-комментарий пакета), обязан отдавать ровно дефолты ядра. Литерала
// здесь нет и не будет до отдельной задачи калибровки.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки IRKT обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsIRKT(t *testing.T) {
	if Ticker != "IRKT" {
		t.Fatalf("Ticker = %q, want IRKT", Ticker)
	}
}
