package backtest

import "testing"

// TestRTKMPGridsStayWide держит НИЖНЮЮ границу ширины сеток RTKMP. Владелец потребовал максимально
// широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие
// обязательных значений и не запрещает лишних. Три отступления от канона (спека
// docs/superpowers/specs/2026-09-05-rtkmp-rsi-pullback-prep-design.md) посажены на замеры полного
// окна 2026-09-05 (36 мес, in-sample) и пиньятся здесь, чтобы редактор сеток не «починил» их
// обратно к канону:
//
//   - EMASlow (cal_trend_low.json) РАСШИРЕНА ВНИЗ ДО 12 (канон [20,30,40]). Точечный замер:
//     12 -> 1.314/113, 15 -> 1.419/120, 20 -> 1.329/128, 30 -> 1.438/132 — максимум всей оси
//     EMASlow стоит в нижнем углу канонической темы trend, и без узлов 12 и 15 он совпал бы с
//     краем сетки. Ниже 12 ось вырождена: EMASlow 10 при EMAFast 10 даёт РОВНО НОЛЬ сделок
//     (равные периоды не пересекаются), EMASlow 8 даёт убыточные 0.945/317.
//   - TPDailyATR (cal_risk.json) РАСШИРЕНА ВНИЗ ДО 0.15 (канон начинается с 0.3). Точечный замер:
//     максимум оси 0.2 -> 1.502/165 стоит вплотную к нижнему краю канонической оси, узел 0.15
//     (1.190/169) служит зондом за краем. Узел 0.1 (1.149/182) в сетку НЕ включён — зонд показал,
//     что сигнала там нет.
//   - StopDailyATR (cal_risk.json) ОСТАВЛЕНА ШИРОКОЙ ДО 2.0 НЕСМОТРЯ НА КАПКАН ШИРОКОГО СТОПА:
//     0.8 -> 1.452/138, 1.0 -> 1.800/138, 1.3 -> 1.794/138, 2.0 -> 1.886/138 — пул сделок стоит на
//     месте с узла 0.6 (140 -> 138), то есть рост PF куплен отодвинутым убытком, а не отбором
//     входов. Просадка при этом ПАДАЕТ (12 402 руб при 1.0 против 16 170 у дефолта), поэтому
//     риск-гейт B капкан не ловит; верхнюю границу режет риск-гейт A (потолок 1.0 при
//     выживаемости 47.1% будних дней, 1.3 — уже 25.5%), а не сетка.
func TestRTKMPGridsStayWide(t *testing.T) {
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
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.3, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.15, 0.2, 0.25, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5}},
		{"cal_exit.json", "RSIUpper", []float64{35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "rtkmp", c.file)
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
				t.Errorf("rtkmp/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestRTKMPEntryGridKeepsRSIUpperAboveRSILower сторожит инвариант, который расширение осей входа
// может нарушить незаметно: одна лишняя точка в любой из осей даёт пару, где выход стоит НИЖЕ
// входа. Такая пара не ошибка запуска — ядро её честно посчитает и вернёт мусорную конфигурацию,
// которая войдёт в ранжирование темы. На RTKMP соблазн особенно велик: оптимум выхода стоит на
// RSIUpper 45 (1.760/177 против 1.313/142 у дефолта 70), то есть ниже всей контекстной оси темы
// entry, и «дотянуть» её вниз хочется — но тогда пары с RSILower 50 стали бы мусором. Поле
// RSIUpper меряет тема exit, а не entry.
func TestRTKMPEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "rtkmp", "cal_entry.json")
	for _, lower := range grid["RSILower"] {
		for _, upper := range grid["RSIUpper"] {
			if upper <= lower {
				t.Errorf("rtkmp/cal_entry.json: пара RSILower=%v, RSIUpper=%v нарушает RSIUpper > RSILower", lower, upper)
			}
		}
	}
}

// TestRTKMPTrendGridsKeepFastBelowSlow проверяет ОБА файла тренда (cal_trend.json,
// cal_trend_low.json). Разложение темы на каноническую сетку и нижний угол существует ровно
// потому, что инвариант EMAFast < EMASlow не позволяет мерить EMASlow 12..40 в одной сетке с
// EMAFast до 40; если кто-нибудь сольёт файлы обратно, тест это поймает. Цена нарушения на RTKMP
// измерена: пара EMAFast = EMASlow = 10 даёт ровно ноль сделок, то есть молчащую конфигурацию,
// которая войдёт в медиану темы как ноль.
func TestRTKMPTrendGridsKeepFastBelowSlow(t *testing.T) {
	for _, file := range []string{"cal_trend.json", "cal_trend_low.json"} {
		grid := rsiPullbackTickerGrid(t, "rtkmp", file)
		for _, fast := range grid["EMAFast"] {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("rtkmp/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}
}
