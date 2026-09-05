package backtest

import "testing"

// TestTRNFPGridsStayWide держит НИЖНЮЮ границу ширины сеток TRNFP. Владелец потребовал максимально
// широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие
// обязательных значений и не запрещает лишних. Четыре отступления от канона (спека
// docs/superpowers/specs/2026-09-05-trnfp-rsi-pullback-prep-design.md) посажены на замеры полного
// окна 2026-09-05 (36 мес, in-sample) и пиньятся здесь, чтобы редактор сеток не «починил» их
// обратно к канону:
//
//   - RSIUpper (cal_exit.json) РАСШИРЕНА ВВЕРХ ДО 95 (канон — вниз до 35). Точечный замер:
//     65 -> 1.176/157 и 75 -> 1.163/150 — максимум оси стоит выше дефолта (70 -> 1.075/155), а
//     ВСЯ ЗОНА НИЖЕ 60 УБЫТОЧНА (32 -> 0.594/209). Это прямая противоположность каталогу (у RTKMP
//     максимум стоял на 45), поэтому ось расширяется вверх, а не вниз.
//   - EMASlow (cal_trend_low.json) РАСШИРЕНА ВНИЗ ДО 12 (канон [20,30,40]). Точечный замер:
//     12 -> 1.372/129 — МАКСИМУМ ВСЕЙ ОСИ EMASlow стоит именно там; узел 10 при EMAFast 10
//     вырожден (равные периоды сигнала не дают ни одной сделки).
//   - TPDailyATR (cal_risk.json) НЕ РАСШИРЯЕТСЯ ВНИЗ НИЖЕ 0.2 (в отличие от RTKMP и SPBE, где
//     нижний край расширяли). Точечный замер: ось инертна СВЕРХУ (размах 0.04 PF на всём
//     диапазоне 0.4…2.5 — далёкая цель просто не достигается) и монотонно валится СНИЗУ
//     (0.1 -> 0.721/189, 0.15 -> 0.828/176). Зонды за нижним краем доказали отсутствие сигнала, а
//     не спрятанный максимум, поэтому ось начинается с 0.2.
//   - StopDailyATR (cal_risk.json) ОСТАВЛЕНА ШИРОКОЙ ДО 2.0 НЕСМОТРЯ НА КАПКАН ШИРОКОГО СТОПА:
//     с 0.5 и выше пул сделок практически стоит (155 -> 151), а PF растёт до 1.757 на узле 2.0 —
//     рост куплен отодвинутым убытком, а не отбором входов. Верхнюю границу здесь режет риск-гейт A
//     (потолок 1.0 при выживаемости 44.2% будних дней), а не сетка, поэтому узлы 1.2, 1.3, 1.5 и
//     2.0 остаются в файле — тема обязана показать цену капкана, а не скрыть её обрезкой оси.
func TestTRNFPGridsStayWide(t *testing.T) {
	cases := []struct {
		file   string
		field  string
		values []float64
	}{
		{"cal_screen.json", "UseDayATRGate", []float64{0, 1}},
		{"cal_screen.json", "UseVolume", []float64{0, 1}},
		{"cal_entry.json", "RSIPeriod", []float64{2, 3, 4, 5, 6, 7, 8, 10}},
		{"cal_entry.json", "RSILower", []float64{10, 15, 20, 25, 30, 35, 40, 45, 50}},
		{"cal_entry.json", "RSIUpper", []float64{55, 60, 65, 70, 75, 80, 85}},
		{"cal_trend.json", "EMAFast", []float64{3, 5, 8, 10, 15, 20, 30, 40}},
		{"cal_trend.json", "EMASlow", []float64{50, 75, 100, 150, 200, 250}},
		{"cal_trend_low.json", "EMAFast", []float64{3, 5, 8, 10}},
		{"cal_trend_low.json", "EMASlow", []float64{12, 15, 20, 30, 40}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_day_spent.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5, 2.0}},
		{"cal_volume.json", "VolMult", []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0}},
		{"cal_volume.json", "VolBaseDays", []float64{3, 5, 10, 14, 20}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{1, 2, 3, 5, 8, 12, 16, 24, 32}},
		{"cal_vol_window.json", "VolMult", []float64{1.0, 1.2, 2.0}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.2, 1.3, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.2, 0.25, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.5}},
		{"cal_exit.json", "RSIUpper", []float64{40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "trnfp", c.file)
		got := grid[c.field]
		for _, v := range c.values {
			var found bool
			for _, g := range got {
				if g == v {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("trnfp/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestTRNFPEntryGridKeepsRSIUpperAboveRSILower сторожит инвариант, который расширение осей входа
// может нарушить незаметно: одна лишняя точка в любой из осей даёт пару, где выход стоит НИЖЕ
// входа. Такая пара не ошибка запуска — ядро её честно посчитает и вернёт мусорную конфигурацию,
// которая войдёт в ранжирование темы. Поле RSIUpper на TRNFP меряет тема exit, а не entry.
func TestTRNFPEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "trnfp", "cal_entry.json")
	for _, lower := range grid["RSILower"] {
		for _, upper := range grid["RSIUpper"] {
			if upper <= lower {
				t.Errorf("trnfp/cal_entry.json: пара RSILower=%v, RSIUpper=%v нарушает RSIUpper > RSILower", lower, upper)
			}
		}
	}
}

// TestTRNFPTrendGridsKeepFastBelowSlow проверяет ОБА файла тренда (cal_trend.json,
// cal_trend_low.json). Разложение темы на каноническую сетку и нижний угол существует ровно
// потому, что инвариант EMAFast < EMASlow не позволяет мерить EMASlow 12..40 в одной сетке с
// EMAFast до 40; если кто-нибудь сольёт файлы обратно, тест это поймает.
func TestTRNFPTrendGridsKeepFastBelowSlow(t *testing.T) {
	for _, file := range []string{"cal_trend.json", "cal_trend_low.json"} {
		grid := rsiPullbackTickerGrid(t, "trnfp", file)
		for _, fast := range grid["EMAFast"] {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("trnfp/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}
}
