package backtest

import "testing"

// TestSOFLGridsStayWide держит НИЖНЮЮ границу ширины сеток SOFL. Владелец потребовал 2026-09-03
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет
// наличие обязательных значений и не запрещает лишних. Три отступления от канона §8.1 посажены на
// замеры (reports/_analysis/sofl_pullback_prep_measurements.md) и пиньятся здесь, чтобы редактор
// сеток не «починил» их обратно к канону:
//
//   - RSIPeriod ВНИЗ до 2 и ВВЕРХ до 10 (канон 3..8). Максимум оси стоит ВНУТРИ неё: 5 -> 1.260/114
//     против дефолтной четвёрки 0.987/166, при этом 6 -> 0.983, 7 -> 1.059, а 8 и 10 мертвы
//     (0.800/57 и 0.803/43). Верхний край нужен, чтобы отделить «максимум внутри» от «максимум на
//     краю»; нижний — ради симметрии проверки (двойка убыточна: 0.965 на 415 сделках).
//     Четырнадцать в сетку не идут вовсе: 9 сделок за три года.
//   - RSIUpper в cal_exit ВНИЗ до 35 при сохранённом верхе 85. Рельеф рваный, и ВТОРАЯ живая зона
//     лежит целиком ниже канонической полосы: 45 -> 1.058/203 и 50 -> 1.058/193 против провала
//     55 -> 0.915 и 65 -> 0.926. Максимум оси (75 -> 1.115/162) канонической полосе принадлежит,
//     но оба её края (55 и 85 -> 0.803) — две худшие точки замера.
//   - EMASlow НЕ расширяется вниз до 20/30, а EMAFast держится канонически широкой (3..40). Это
//     ЗЕРКАЛО AFKS, где пришлось резать быструю ось: на SOFL 20 -> 0.957 и 30 -> 0.939 — две
//     ХУДШИЕ точки медленной оси, тогда как 20 -> 1.073 и 30 -> 1.070 — две ЛУЧШИЕ точки быстрой.
//     Инвариант EMAFast < EMASlow не разрешает держать обе оси широкими, и замер однозначно
//     говорит, какую отдавать.
func TestSOFLGridsStayWide(t *testing.T) {
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
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "sofl", c.file)
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
				t.Errorf("sofl/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestSOFLEntryGridKeepsRSIUpperAboveRSILower пинит жёсткий инвариант связки входа и выхода: в
// теме entry обе оси свипуются вместе, и пара, где уровень выхода не выше уровня входа, описывает
// сделку, которая закрывается в момент открытия.
func TestSOFLEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "sofl", "cal_entry.json")
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

// TestSOFLTrendGridKeepsFastBelowSlow пинит вторую половину решения об осях тренда: EMASlow
// начинается с 50 ИМЕННО ПОТОМУ, что EMAFast держится широкой до 40. Любая пара, где быстрая EMA
// не быстрее медленной, превращает трендовый фильтр в перевёрнутый и на падающем инструменте
// способна выиграть тему in-sample.
func TestSOFLTrendGridKeepsFastBelowSlow(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "sofl", "cal_trend.json")
	for _, fast := range grid["EMAFast"] {
		for _, slow := range grid["EMASlow"] {
			if fast >= slow {
				t.Fatalf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: перевёрнутый трендовый фильтр", fast, slow)
			}
		}
	}
}
