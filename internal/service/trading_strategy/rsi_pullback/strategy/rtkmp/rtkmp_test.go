package rtkmp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ПРОМЕЖУТОЧНОЕ состояние: пакет заведён под
// калибровку 2026-09-05, литерала ещё нет, и до вердикта он обязан отдавать ровно дефолты ядра.
// Тест снимается вместе с постановкой литерала.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки RTKMP обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsRTKMP(t *testing.T) {
	if Ticker != "RTKMP" {
		t.Fatalf("Ticker = %q, want RTKMP", Ticker)
	}
}
