package backtest

import "testing"

// TestNKHPGridsStayWide держит НИЖНЮЮ границу ширины сеток NKHP. Владелец потребовал 2026-09-02
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет
// наличие обязательных значений и не запрещает лишних. Четыре отступления от канона §8.1 посажены на
// замеры (reports/_analysis/nkhp_pullback_prep_measurements.md) и пиньятся здесь, чтобы редактор
// сеток не «починил» их обратно к канону:
//
//   - RSIPeriod вниз до 2: период 2 даёт PF 1.572 на 300 сделках против 1.732 на 112 у дефолта 4;
//     при дневном ATR 4.28% сверхкороткий RSI успевает отработать откат.
//   - RSIUpper в cal_exit вниз до 35: максимум оси стоит на 40 (2.100/145), плато 40-50, а вся
//     каноническая ось 55..85 лежит на затухающей стороне (её лучшая точка 55 -> 1.811 хуже
//     худшей точки живой зоны 50 -> 2.028). Верхний край 85 сохранён: ось расширена в обе
//     стороны, а не сдвинута.
//   - VolMult вверх до 4.0: объёмный гейт на NKHP впервые в каталоге несёт сигнал (весь массив
//     оси выше baseline без гейта 1.732, максимум 2.5 -> 2.562/65), и край нужен, чтобы отличить
//     «максимум внутри» от «максимум на краю».
//   - EMASlow в cal_trend вниз до 20 при EMAFast, обрезанной до 10: та же причина — жёсткий
//     инвариант EMAFast < EMASlow не разрешает держать обе оси широкими одновременно.
func TestNKHPGridsStayWide(t *testing.T) {
	type want struct {
		file   string
		field  string
		values []float64
	}
	cases := []want{
		{"cal_screen.json", "UseDayATRGate", []float64{0, 1}},
		{"cal_screen.json", "UseVolume", []float64{0, 1}},
		{"cal_entry.json", "RSILower", []float64{10, 15, 20, 25, 30, 35, 40, 45, 50}},
		{"cal_entry.json", "RSIPeriod", []float64{2, 3, 4, 5, 6, 7, 8}},
		{"cal_entry.json", "RSIUpper", []float64{55, 60, 65, 70, 75, 80, 85}},
		{"cal_trend.json", "EMAFast", []float64{3, 5, 10}},
		{"cal_trend.json", "EMASlow", []float64{20, 30, 50, 70, 100, 150, 200, 250}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.2, 0.3, 0.4, 0.5, 0.6}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5}},
		{"cal_day_spent.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5}},
		{"cal_volume.json", "VolMult", []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0, 4.0}},
		{"cal_volume.json", "VolBaseDays", []float64{3, 5, 10, 14, 20}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{1, 2, 3, 5, 8, 12, 16, 24}},
		{"cal_vol_window.json", "VolMult", []float64{1.2, 2.5, 4.0}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.3, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.5}},
		{"cal_exit.json", "RSIUpper", []float64{35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "nkhp", c.file)
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
				t.Errorf("nkhp/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestNKHPEntryGridKeepsRSIUpperAboveRSILower пинит жёсткий инвариант связки входа и выхода: в
// теме entry обе оси свипуются вместе, и пара, где уровень выхода не выше уровня входа, описывает
// сделку, которая закрывается в момент открытия. Инвариант держится конструкцией осей
// (RSILower <= 50 < 55 <= RSIUpper), и тест ловит его нарушение при любой правке файла.
func TestNKHPEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "nkhp", "cal_entry.json")
	var maxLower, minUpper float64
	minUpper = 1e9
	for _, v := range grid["RSILower"] {
		if v > maxLower {
			maxLower = v
		}
	}
	for _, v := range grid["RSIUpper"] {
		if v < minUpper {
			minUpper = v
		}
	}
	if minUpper <= maxLower {
		t.Fatalf("nkhp/cal_entry.json: min(RSIUpper)=%v <= max(RSILower)=%v — сетка порождает пары, где выход не выше входа", minUpper, maxLower)
	}
}

// TestNKHPTrendGridKeepsFastBelowSlow пинит жёсткий инвариант оси тренда: сетка не должна
// порождать пар, где период «быстрой» EMA не меньше периода «медленной». На таких парах условие
// fast > slow означает не восходящий тренд, а перевёрнутый фильтр, и на падающем инструменте он
// способен выиграть тему in-sample. Инвариант держится конструкцией осей (EMAFast <= 10 <
// EMASlow >= 20), и тест ловит его нарушение при любой правке файла.
func TestNKHPTrendGridKeepsFastBelowSlow(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "nkhp", "cal_trend.json")
	var maxFast, minSlow float64
	minSlow = 1e9
	for _, v := range grid["EMAFast"] {
		if v > maxFast {
			maxFast = v
		}
	}
	for _, v := range grid["EMASlow"] {
		if v < minSlow {
			minSlow = v
		}
	}
	if maxFast >= minSlow {
		t.Fatalf("nkhp/cal_trend.json: max(EMAFast)=%v >= min(EMASlow)=%v — сетка порождает пары с перевёрнутым трендовым фильтром", maxFast, minSlow)
	}
}
