package mvid

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит СОСТОЯНИЕ ДО КАЛИБРОВКИ: пакет strategy/mvid
// заведён 2026-09-07, чтобы прогоны шли через общий реестр, а не через generic-ветку, и до тех пор,
// пока калибровка не проведена, обязан отдавать ровно дефолты ядра. Литерала здесь нет и не будет
// до отдельной задачи калибровки — см. doc-комментарий пакета.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки MVID обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsMVID(t *testing.T) {
	if Ticker != "MVID" {
		t.Fatalf("Ticker = %q, want MVID", Ticker)
	}
}
