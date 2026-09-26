package mdmg

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated держит состояние «калибровка не проводилась»: до
// вердикта по спеке docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md пакет
// отдаёт ровно дефолты ядра. Тест заменяется снимком литерала (Task 14, маршрут точки) или
// решением «дефолты по выбору» (Task 14, маршрут дефолтов), либо остаётся протоколом отказа.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки MDMG обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsMDMG(t *testing.T) {
	if Ticker != "MDMG" {
		t.Fatalf("Ticker = %q, want MDMG", Ticker)
	}
}
