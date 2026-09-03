package sofl

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка SOFL ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// Task 13, где его заменяет снимок литерала. Пока он стоит, ни одна правка не может тихо
// подсунуть в прод «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("SOFL ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSOFL(t *testing.T) {
	if Ticker != "SOFL" {
		t.Fatalf("Ticker = %q, want SOFL", Ticker)
	}
}
