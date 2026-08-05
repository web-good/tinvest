package backtest

import (
	"path/filepath"
	"testing"
)

// domrfGrid читает файл сеток DOMRF и сливает оси всех его фаз в одну карту. Файлы каталога
// однотемные, поэтому слияние не теряет информации, а тестам не приходится знать имя фазы.
func domrfGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	path := filepath.Join(rsiPullbackParamsDir, "domrf", file)
	out := make(map[string][]float64)
	for _, ph := range rsiPullbackPhases(t, path) {
		for field, values := range ph.Grid {
			out[field] = append(out[field], values...)
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s: сетка пуста", file)
	}
	return out
}

// TestDOMRFSignalGridsPinTheirMeasuredAxes сторожит решения, которые обоснованы замерами
// инструмента, а не вкусом. Каталог domrf/ заведён копированием структуры ugld/, и типовая
// ошибка такой копии — притащить вместе с ней чужие оси: на UGLD дневной ATR 4.28%, на DOMRF
// 1.94%, и половина осей UGLD здесь либо мертва, либо неисполнима.
func TestDOMRFSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	entry := domrfGrid(t, "cal_entry.json")

	// RSI(6) пересекает 10 вниз 18 раз за ВСЮ историю инструмента: после шести гейтов входа
	// и многодневного удержания на этой точке не остаётся выборки вовсе.
	for _, v := range entry["RSILower"] {
		if v < 15 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 15 сигналов не остаётся (RSI(6)@10 = 18 кроссов за 8.4 мес)", v)
		}
	}
	// RSI(3) на инструменте с ATR 1.94% покупает шум, а не откат.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум при ATR 1.94%%", v)
		}
	}
	// Полоса выхода меряется отдельным cal_exit.json. В фазе entry она стоила бы 12x5
	// степеней свободы на выборке в 20-30 сделок — переобучение по построению.
	if _, ok := entry["RSIUpper"]; ok {
		t.Error("cal_entry.json свипует RSIUpper: полоса выхода принадлежит cal_exit.json")
	}

	if got := len(domrfGrid(t, "cal_exit.json")["RSIUpper"]); got != 5 {
		t.Errorf("cal_exit.json: RSIUpper имеет %d значений, want 5", got)
	}

	gates := domrfGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if len(gates[field]) != 2 {
			t.Errorf("cal_screen.json: %s должен свипуть обе точки [0,1], got %v", field, gates[field])
		}
	}

	trend := domrfGrid(t, "cal_trend.json")
	if len(trend["EMAFast"]) != 3 || len(trend["EMASlow"]) != 4 {
		t.Errorf("cal_trend.json: сетка %vx%v, want 3x4", len(trend["EMAFast"]), len(trend["EMASlow"]))
	}
}
