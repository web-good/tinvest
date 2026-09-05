package rtkmp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ОКОНЧАТЕЛЬНОЕ состояние: калибровка RTKMP
// закрыта 2026-09-05 отказом обоих кругов (первый — пункт 3 стоп-условия, контроль 24/12/3 дал
// 0.662; второй, спасательный на однородном 24-месячном окне — условия 1 и 5 вердикта, 0.492 и
// 0.298), тикер в боевую вселенную не заводится, и пакет обязан отдавать ровно дефолты ядра.
// Литерала здесь не будет: точка, не прошедшая стоп-условие, не имеет права попасть в живой
// раннер через реестр. Разбор — docs/superpowers/plans/task-11-report-rtkmp.md и
// docs/superpowers/plans/2026-09-05-rtkmp-rescue-round.md.
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
