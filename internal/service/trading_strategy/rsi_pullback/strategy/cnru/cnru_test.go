package cnru

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsAreTheCoreBaselineByDesign пинит РЕШЕНИЕ, А НЕ ПРОМЕЖУТОЧНОЕ СОСТОЯНИЕ: калибровка CNRU
// проведена целиком 2026-09-10 и не нашла ни одного поля, уходящего от дефолтов ядра, поэтому пакет
// отдаёт ровно core.DefaultParams() и литерала здесь не появится без новой калибровки. Замена этих
// параметров литералом «по аналогии с соседями» была бы подгонкой, не подтверждённой ни одной
// темой: из восемнадцати полей большинство набрали пять, три из них подтвердили дефолт, стоп
// отвергнут риск-гейтом A и анатомией капкана, а VolMult инертен при выключенном объёмном гейте.
// Разбор — doc-комментарий пакета и docs/superpowers/plans/task-11-report-cnru.md.
func TestParamsAreTheCoreBaselineByDesign(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("CNRU торгует дефолты ядра по решению 2026-09-10:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsCNRU(t *testing.T) {
	if Ticker != "CNRU" {
		t.Fatalf("Ticker = %q, want CNRU", Ticker)
	}
}
