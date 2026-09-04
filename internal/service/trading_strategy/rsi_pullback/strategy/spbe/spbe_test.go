package spbe

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ПРОМЕЖУТОЧНОЕ состояние: пакет заведён под
// калибровку 2026-09-04, литерала ещё нет, и до вердикта он обязан отдавать ровно дефолты ядра.
// Тест снимается вместе с постановкой литерала (Task 12).
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки SPBE обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSPBE(t *testing.T) {
	if Ticker != "SPBE" {
		t.Fatalf("Ticker = %q, want SPBE", Ticker)
	}
}
