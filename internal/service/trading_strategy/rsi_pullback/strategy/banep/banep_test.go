package banep

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка BANEP ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// задачи с литералом, где его заменяет снимок. Пока он стоит, ни одна правка не может тихо
// подсунуть в прод «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("BANEP ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsBANEP(t *testing.T) {
	if Ticker != "BANEP" {
		t.Fatalf("Ticker = %q, want BANEP", Ticker)
	}
}
