package rtkm

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ОКОНЧАТЕЛЬНОЕ состояние: калибровка RTKM закрыта
// 2026-09-25 отказом обоих кругов (первый — пункты 3, 5 и 6 стоп-условия, точка функционально
// совпала с дефолтами; второй, узкие сетки вокруг зоны первого круга — те же пункты 3, 5 и 6 у обоих
// вариантов точки), тикер в боевую вселенную не заводится, и пакет обязан отдавать ровно дефолты
// ядра. Литерала здесь не будет: точка, не прошедшая стоп-условие, не имеет права попасть в живой
// раннер через реестр. Разбор — docs/superpowers/plans/task-11-report-rtkm.md (первый круг) и
// docs/superpowers/plans/task-12-report-rtkm.md (второй круг).
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
