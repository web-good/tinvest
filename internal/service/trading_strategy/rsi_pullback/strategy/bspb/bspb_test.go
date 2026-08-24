package bspb

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка BSPB ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// Task 13, где его заменяет снимок литерала. Пока он стоит, ни одна правка не может тихо
// подсунуть в прод «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("BSPB ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsBSPB(t *testing.T) {
	if Ticker != "BSPB" {
		t.Fatalf("Ticker = %q, want BSPB", Ticker)
	}
}
