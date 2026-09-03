package tgka

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsStayAtTheCoreBaselineByDesign пинит РЕШЕНИЕ, а не промежуточное состояние: TGKA
// торгует дефолты ядра осознанно. Калибровка 2026-09-03 проведена и провалена трижды, каждый раз
// против самих дефолтов (0.899 против 2.518 на контроле 24/12/3; точка окна B выродилась до одной
// сделки в пуле OOS; на слепом holdout 0.815 при net -1 070 против 3.709 и +20 223), поэтому в
// боевую вселенную тикер заведён на baseline — единственной конфигурации каталога, которая не
// подбиралась под эту бумагу и потому не может быть переоптимизирована.
//
// Тест падает на любой правке параметров. Это и есть его смысл: изменить их можно только вместе с
// НОВОЙ калибровкой, у которой есть проверка вне выборки, — и тогда же снимается именное исключение
// baselineByDesignTickers в live/registry_test.go. Правка «на глазок» здесь опаснее обычного:
// внешне TGKA выглядит как любой другой боевой тикер.
func TestParamsStayAtTheCoreBaselineByDesign(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("TGKA торгует дефолты ядра по решению владельца 2026-09-03; параметры разошлись с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsTGKA(t *testing.T) {
	if Ticker != "TGKA" {
		t.Fatalf("Ticker = %q, want TGKA", Ticker)
	}
}
