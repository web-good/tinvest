package backtest

import "testing"

// TestAFKSGridsStayWide держит НИЖНЮЮ границу ширины сеток AFKS. Владелец потребовал 2026-09-02
// максимально широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет
// наличие обязательных значений и не запрещает лишних. Четыре отступления от канона §8.1 посажены
// на замеры (reports/_analysis/afks_pullback_prep_measurements.md) и пиньятся здесь, чтобы
// редактор сеток не «починил» их обратно к канону:
//
//   - RSIPeriod ВВЕРХ до 10 и ВНИЗ до 2 (канон 3..8). Зона 7-8 — лучшая связная на оси
//     (1.403/66 и 1.388/56 против 1.100/126 у дефолта 4), десятка ещё держит 38 сделок и
//     отделяет «максимум внутри» от «максимум на краю». Нижний край стоит ради симметрии
//     проверки: двойка убыточна (0.984 на 328 сделках), её победа означала бы переоптимизацию.
//   - RSIUpper в cal_exit ВНИЗ до 35 при сохранённом верхе 85. Замер говорит, что низ оси мёртв
//     (35 -> 0.759/155, 40 -> 0.806/150), а максимум стоит ровно на дефолте 70 (1.100). Ось
//     расширена ради доказуемости середины: на NKHP тот же приём дал обратный ответ, там
//     максимум и оказался на 40.
//   - VolMult ВВЕРХ до 4.0: максимум множителя стоит на крае (4.0 -> 1.348/39 против
//     1.2 -> 1.194/103). Предупреждение о крае: 39 сделок за окно — это около 13 на обучающее
//     окно, ниже -min-trades 20, и порог, скорее всего, отсечёт край внутри фолдов.
//   - EMASlow в cal_trend ВНИЗ до 20 при EMAFast, обрезанной сверху до 10. Оба края оси EMASlow
//     выше середины (20 -> 1.633/108, 250 -> 1.214/138 против 100 -> 1.100/126), но полная
//     канонической ширины пара осей породила бы пары EMAFast >= EMASlow — перевёрнутый трендовый
//     фильтр, способный на падающем на 60% инструменте выиграть тему in-sample. Инвариант не
//     разрешает держать обе оси широкими одновременно.
func TestAFKSGridsStayWide(t *testing.T) {
	type want struct {
		file   string
		field  string
		values []float64
	}
	cases := []want{
		{"cal_screen.json", "UseDayATRGate", []float64{0, 1}},
		{"cal_screen.json", "UseVolume", []float64{0, 1}},
		{"cal_entry.json", "RSILower", []float64{10, 15, 20, 25, 30, 35, 40, 45, 50}},
		{"cal_entry.json", "RSIPeriod", []float64{2, 3, 4, 5, 6, 7, 8, 10}},
		{"cal_entry.json", "RSIUpper", []float64{55, 60, 65, 70, 75, 80, 85}},
		{"cal_trend.json", "EMAFast", []float64{3, 5, 10}},
		{"cal_trend.json", "EMASlow", []float64{20, 30, 50, 70, 100, 150, 200, 250}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.2, 0.3, 0.4, 0.5, 0.6}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_day_spent.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_volume.json", "VolMult", []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0, 4.0}},
		{"cal_volume.json", "VolBaseDays", []float64{3, 5, 10, 14, 20}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{1, 2, 3, 5, 8, 12, 16, 24}},
		{"cal_vol_window.json", "VolMult", []float64{1.2, 2.5, 4.0}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.3, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5}},
		{"cal_exit.json", "RSIUpper", []float64{35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "afks", c.file)
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
				t.Errorf("afks/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestAFKSEntryGridKeepsRSIUpperAboveRSILower пинит жёсткий инвариант связки входа и выхода: в
// теме entry обе оси свипуются вместе, и пара, где уровень выхода не выше уровня входа, описывает
// сделку, которая закрывается в момент открытия. Инвариант держится конструкцией осей
// (RSILower <= 50 < 55 <= RSIUpper), и тест ловит его нарушение при любой правке файла.
func TestAFKSEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "afks", "cal_entry.json")
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

// TestAFKSTrendGridKeepsFastBelowSlow пинит вторую половину решения об осях тренда: EMAFast
// обрезана сверху до 10 ИМЕННО ПОТОМУ, что EMASlow расширена вниз до 20. Любая пара, где быстрая
// EMA не медленнее медленной, превращает трендовый фильтр в перевёрнутый и на падающем
// инструменте способна выиграть тему in-sample.
func TestAFKSTrendGridKeepsFastBelowSlow(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "afks", "cal_trend.json")
	for _, fast := range grid["EMAFast"] {
		for _, slow := range grid["EMASlow"] {
			if fast >= slow {
				t.Fatalf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: перевёрнутый трендовый фильтр", fast, slow)
			}
		}
	}
}
