package magn

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит СОСТОЯНИЕ ДО КАЛИБРОВКИ: MAGN ещё не прошёл ни
// одного круга подбора параметров. Walk-forward дефолтов ядра даёт ровный ноль — pooled OOS PF
// 1.005 на 94 сделках схемы 36/12/6 при двух убыточных фолдах из четырёх (0.915/25, 1.050/24,
// 0.961/22, 1.122/23), поэтому маршрут «завести на дефолтах ядра» по прецеденту TGKA закрыт (см.
// doc-комментарий пакета). До появления литерала пакет обязан отдавать ровно дефолты ядра; тест
// сломается, как только калибровка подберёт точку и DefaultParams() перестанет совпадать с
// core.DefaultParams().
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("до калибровки MAGN обязан отдавать baseline ядра:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsMAGN(t *testing.T) {
	if Ticker != "MAGN" {
		t.Fatalf("Ticker = %q, want MAGN", Ticker)
	}
}
