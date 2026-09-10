package aqua

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsTrackTheBaselineUntilCalibrated пинит ЧЕСТНОЕ промежуточное состояние: AQUA заведён в
// реестр до калибровки, DefaultParams() обязан возвращать ровно core.DefaultParams() поле за
// полем. Контрольный прогон дефолтов ядра на расчётном окне (136 сделок, PF 0.931, net
// -5046.29 руб.) убыточен — «маршрут TGKA» здесь закрыт, а не открыт: этот пакет не завершённое
// решение уровня CNRU/TGKA, а состояние до единого прогона калибровки. Подмена этих параметров
// литералом «по аналогии с соседями» без калибровки была бы подгонкой. Разбор — doc-комментарий
// пакета.
func TestParamsTrackTheBaselineUntilCalibrated(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("AQUA ещё не откалиброван, params обязаны совпадать с baseline:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsAQUA(t *testing.T) {
	if Ticker != "AQUA" {
		t.Fatalf("Ticker = %q, want AQUA", Ticker)
	}
}
