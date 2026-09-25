package rtkm

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated держит состояние «калибровка не проводилась»: до
// вердикта по спеке docs/superpowers/specs/2026-09-25-rtkm-rsi-pullback-prep-design.md пакет
// отдаёт ровно дефолты ядра. Тест заменяется снимком литерала (Task 14) или остаётся как протокол
// отказа (Task 13).
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки RTKM обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsRTKM(t *testing.T) {
	if Ticker != "RTKM" {
		t.Fatalf("Ticker = %q, want RTKM", Ticker)
	}
}
