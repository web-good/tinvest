package x5

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated фиксирует ЧЕСТНОЕ состояние: калибровка X5 ещё не
// проводилась, поэтому пакет обязан возвращать ровно baseline ядра. Тест держит это состояние до
// задачи калибровки (§5.13 спеки docs/superpowers/specs/2026-09-13-x5-rsi-pullback-prep-design.md),
// где его заменяет снимок литерала. Пока он стоит, ни одна правка не может тихо подсунуть в прод
// «почти откалиброванные» параметры.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("X5 ещё не откалиброван, параметры обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsX5(t *testing.T) {
	if Ticker != "X5" {
		t.Fatalf("Ticker = %q, want X5", Ticker)
	}
}
