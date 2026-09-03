package backtest

import "testing"

// TestTGKAGridsStayWide держит НИЖНЮЮ границу ширины сеток TGKA. Владелец потребовал 2026-09-03
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет
// наличие обязательных значений и не запрещает лишних. Пять отступлений от канона §8.1 посажены на
// замеры (reports/_analysis/tgka_pullback_prep_measurements.md) и пиньятся здесь, чтобы редактор
// сеток не «починил» их обратно к канону:
//
//   - RSIPeriod ВНИЗ до 2 и ВВЕРХ до 10 (канон 3..8). Дефолтная четвёрка — максимум оси по паре
//     «PF + объём выборки»: 4 -> 1.967/167 против 3 -> 1.437/237 и 5 -> 1.760/118. Верхний край
//     нужен, чтобы отделить «максимум внутри оси» от «максимум на краю»: 10 -> 2.094 формально выше
//     всех, но на 38 сделках за три года (~13 на обучающее окно, ниже -min-trades 20), а 12 и 14
//     вырождены (1.034/18 и 1.954/12) и в сетку не идут вовсе. Нижний край держит симметрию
//     проверки: 2 -> 1.303/405 — много сделок и низкий PF.
//   - ТЕМА ТРЕНДА РАЗЛОЖЕНА НА ДВЕ СЕТКИ, и это ЗЕРКАЛО SOFL. На SOFL низ медленной оси был худшим,
//     и приходилось выбирать, какую ось отдать инварианту EMAFast < EMASlow. На TGKA оба максимума
//     лежат ВНИЗУ обеих осей: EMASlow 20 -> 2.221/150 (максимум оси) при EMAFast 10, и
//     EMAFast 3 -> 2.274/142 (максимум оси) при EMASlow 100. Пара 3/20 инвариант не нарушает,
//     поэтому cal_trend остаётся КАНОНИЧЕСКОЙ (EMAFast 3..40 x EMASlow 50..250) — на ней стоит
//     планка и сравнение с каталогом, — а нижний угол (EMASlow 20 и 30), недоступный канонической
//     сетке при EMAFast до 40, мерит отдельная cal_trend_low.
//   - VolLookbackBars ВВЕРХ до 32 (канон 24). Максимум точечного замера стоит на каноническом краю
//     (24 -> 2.068/161), а 32 -> 2.105/162 и 48 -> 2.105/162 равны ПОБАЙТОВО: ось насыщается, и
//     верхний край нужен, чтобы отделить «максимум внутри» от «максимум на краю».
//   - RSIUpper в cal_exit ВВЕРХ до 90 при сохранённом низе 35. Рельеф МОНОТОННЫЙ (в отличие от
//     рваного у SOFL): 35 -> 1.274, 50 -> 1.678, 70 -> 1.967 (дефолт), 85 -> 2.009/159 (максимум,
//     net +113 739). Максимум стоит на верхнем крае канонической полосы, и без зонда за краем край
//     не отличить от обрезанной оси; зонды 90 -> 1.772 и 95 -> 1.700 падают, поэтому 90 в сетке
//     есть, а 95 — нет.
//   - SpentDayATR НЕ расширяется выше 1.5, хотя точечно там максимумы (1.75 -> 3.151/20,
//     2.0 -> 9.829/15). Это единственное место, где ось не расширяется, и основание — вырождение
//     выборки: 15-20 сделок за три года дают 5-7 на обучающее окно, порог -min-trades 20 их
//     отсечёт, а победа такого края означала бы подгонку под единичные дни.
func TestTGKAGridsStayWide(t *testing.T) {
	cases := []struct {
		file   string
		field  string
		values []float64
	}{
		{"cal_screen.json", "UseDayATRGate", []float64{0, 1}},
		{"cal_screen.json", "UseVolume", []float64{0, 1}},
		{"cal_entry.json", "RSILower", []float64{10, 15, 20, 25, 30, 35, 40, 45, 50}},
		{"cal_entry.json", "RSIPeriod", []float64{2, 3, 4, 5, 6, 7, 8, 10}},
		{"cal_entry.json", "RSIUpper", []float64{55, 60, 65, 70, 75, 80, 85}},
		{"cal_trend.json", "EMAFast", []float64{3, 5, 10, 20, 30, 40}},
		{"cal_trend.json", "EMASlow", []float64{50, 70, 100, 150, 200, 250}},
		{"cal_trend_low.json", "EMAFast", []float64{3, 5, 10}},
		{"cal_trend_low.json", "EMASlow", []float64{20, 30}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.2, 0.3, 0.4, 0.5, 0.6}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_day_spent.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_volume.json", "VolMult", []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0, 4.0}},
		{"cal_volume.json", "VolBaseDays", []float64{3, 5, 10, 14, 20}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{1, 2, 3, 5, 8, 12, 16, 24, 32}},
		{"cal_vol_window.json", "VolMult", []float64{1.2, 2.5, 4.0}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.3, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5}},
		{"cal_exit.json", "RSIUpper", []float64{35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "tgka", c.file)
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
				t.Errorf("tgka/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestTGKAEntryGridKeepsRSIUpperAboveRSILower пинит жёсткий инвариант связки входа и выхода: в теме
// entry обе оси свипуются вместе, и пара, где уровень выхода не выше уровня входа, описывает
// сделку, которая закрывается в момент открытия.
func TestTGKAEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "tgka", "cal_entry.json")
	var maxLower float64
	minUpper := 1e9
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
		t.Fatalf("cal_entry.json порождает пару RSIUpper=%v <= RSILower=%v: такая сделка закрывается в момент открытия", minUpper, maxLower)
	}
}

// TestTGKATrendGridsKeepFastBelowSlow пинит вторую половину решения о разложении темы тренда на две
// сетки: каждая из них по отдельности обязана держать EMAFast < EMASlow на ВСЕХ своих парах. Именно
// это разложение и позволило не резать ни одну ось — в отличие от SOFL, где одну пришлось отдать.
// Любая пара, где быстрая EMA не быстрее медленной, превращает трендовый фильтр в перевёрнутый и на
// падающем инструменте способна выиграть тему in-sample.
func TestTGKATrendGridsKeepFastBelowSlow(t *testing.T) {
	for _, file := range []string{"cal_trend.json", "cal_trend_low.json"} {
		grid := rsiPullbackTickerGrid(t, "tgka", file)
		for _, fast := range grid["EMAFast"] {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Fatalf("%s порождает пару EMAFast=%v >= EMASlow=%v: перевёрнутый трендовый фильтр", file, fast, slow)
				}
			}
		}
	}
}
