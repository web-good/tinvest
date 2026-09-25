package svcb

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ОКОНЧАТЕЛЬНОЕ состояние: калибровка SVCB закрыта
// отказом 2026-09-25 (первый круг — пункты 5, 6 и 7 стоп-условия: 2026 год убыточен, хвост 6 < 1.0,
// гейт D провален сделкой через дату-прокси 2024-07-08; второй круг на узких сетках вокруг зоны
// первого круга — пункты 3, 4, 5 и 6: контроль 24/12/3 0.836 < 1.0, утяжелённые издержки 0.880 <
// 1.0, 2026 год −3 481.59 ₽, оба хвоста < 1.0), тикер в боевую вселенную не заводится, и пакет
// обязан отдавать ровно дефолты ядра. Литерала здесь не будет: точка, не прошедшая стоп-условие, не
// имеет права попасть в живой раннер через реестр. Справочный сосед `plateau_r2_stop_10` (стоп 1.0)
// проходит все семь пунктов на тех же данных — владелец вправе переопределить вердикт (прецедент
// MVID), но по буквальному правилу сборки такая точка не собирается ни в одном из двух кругов.
// Разбор — docs/superpowers/plans/task-11-report-svcb.md (первый круг) и
// docs/superpowers/plans/task-12-report-svcb.md (второй круг).
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки SVCB обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSVCB(t *testing.T) {
	if Ticker != "SVCB" {
		t.Fatalf("Ticker = %q, want SVCB", Ticker)
	}
}
