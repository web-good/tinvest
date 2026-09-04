package backtest

import "testing"

// TestUWGNGridsStayWide держит НИЖНЮЮ границу ширины сеток UWGN. Владелец потребовал максимально
// широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие
// обязательных значений и не запрещает лишних. Три отступления от канона (спека
// docs/superpowers/specs/2026-09-04-uwgn-rsi-pullback-prep-design.md) посажены на замеры полного
// окна 2026-09-04 (30 мес, окно 2024-03-05..2026-09-04, Minutes30, in-sample) и пиньятся здесь,
// чтобы редактор сеток не «починил» их обратно к канону:
//
//   - TPDailyATR (cal_risk.json) РАСШИРЕНА ВНИЗ ДО 0.2 (канон начинается с 0.3). Нижний край
//     канонической оси держит максимум на самом краю: 0.3 -> 1.111/134 против дефолта
//     0.6 -> 1.055/123 — значит нужен зонд за краем, чтобы отделить «максимум внутри оси» от
//     «максимум на краю».
//   - StopDailyATR (cal_risk.json) ДЕРЖИТСЯ ДО 2.0 НЕСМОТРЯ НА КАПКАН ШИРОКОГО СТОПА:
//     0.6 -> 1.154/121, 1.0 -> 1.232/117, 2.0 -> 1.527/115 — пул сделок стоит на месте (121 -> 115),
//     то есть рост PF куплен отодвинутым убытком, а не отбором входов. Верхнюю границу здесь режет
//     риск-гейт A (потолок 1.0 при выживаемости 41.4% будних дней), а не сетка, поэтому узлы 1.3,
//     1.5 и 2.0 остаются в файле — тема обязана показать цену капкана, а не скрыть её обрезкой оси.
//   - ИСКЛЮЧЕНЫ ВЫРОЖДЕННЫЕ ТОЧКИ входа (cal_entry.json): RSILower 5 (0.801/5), RSIPeriod 12
//     (1.034/16) и 14 (0.705/12). Это не обрезка рабочей оси — три и пять сделок за 30 месяцев не
//     дают фолду ничего измерить, поэтому узлы в сетку не идут вовсе.
func TestUWGNGridsStayWide(t *testing.T) {
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
		{"cal_trend.json", "EMAFast", []float64{3, 5, 8, 10, 15, 20, 30, 40}},
		{"cal_trend.json", "EMASlow", []float64{50, 75, 100, 150, 200, 250}},
		{"cal_trend_low.json", "EMAFast", []float64{3, 5, 8, 10}},
		{"cal_trend_low.json", "EMASlow", []float64{20, 30, 40}},
		{"cal_day.json", "FreshDayATR", []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5}},
		{"cal_day.json", "SpentDayATR", []float64{0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_day_spent.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5, 2.0}},
		{"cal_volume.json", "VolMult", []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0}},
		{"cal_volume.json", "VolBaseDays", []float64{3, 5, 10, 14, 20}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{1, 2, 3, 5, 8, 12, 16, 24, 32}},
		{"cal_vol_window.json", "VolMult", []float64{1.0, 1.2, 2.0}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.3, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5}},
		{"cal_exit.json", "RSIUpper", []float64{35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "uwgn", c.file)
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
				t.Errorf("uwgn/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestUWGNEntryGridKeepsRSIUpperAboveRSILower сторожит инвариант, который расширение осей входа
// может нарушить незаметно: одна лишняя точка в любой из осей даёт пару, где выход стоит НИЖЕ
// входа. Такая пара не ошибка запуска — ядро её честно посчитает и вернёт мусорную конфигурацию,
// которая войдёт в ранжирование темы. Проверка идёт по обоим кругам файлов (cal_entry.json первого
// круга и cal_w24_entry.json второго): оси второго круга — буквальная копия первого на новом окне,
// поэтому один и тот же цикл, а не параллельный тест с суффиксом Rescue — отдельная таблица давала
// бы только дублирование тех же значений под другим именем.
func TestUWGNEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	for _, file := range []string{"cal_entry.json", "cal_w24_entry.json"} {
		grid := rsiPullbackTickerGrid(t, "uwgn", file)
		for _, lower := range grid["RSILower"] {
			for _, upper := range grid["RSIUpper"] {
				if upper <= lower {
					t.Errorf("uwgn/%s: пара RSILower=%v, RSIUpper=%v нарушает RSIUpper > RSILower", file, lower, upper)
				}
			}
		}
	}
}

// TestUWGNTrendGridsKeepFastBelowSlow проверяет ВСЕ ЧЕТЫРЕ файла тренда обоих кругов
// (cal_trend.json, cal_trend_low.json, cal_w24_trend.json, cal_w24_trend_low.json). Разложение
// темы на каноническую сетку и нижний угол существует ровно потому, что инвариант
// EMAFast < EMASlow не позволяет мерить EMASlow 20..40 в одной сетке с EMAFast до 40; если
// кто-нибудь сольёт файлы обратно, тест это поймает. Второй круг добавлен в ту же таблицу, а не
// вынесен в TestUWGNTrendGridsKeepFastBelowSlowRescue, по той же причине, что и выше: оси
// идентичны первому кругу, отдельный тест не проверял бы ничего нового.
func TestUWGNTrendGridsKeepFastBelowSlow(t *testing.T) {
	for _, file := range []string{"cal_trend.json", "cal_trend_low.json", "cal_w24_trend.json", "cal_w24_trend_low.json"} {
		grid := rsiPullbackTickerGrid(t, "uwgn", file)
		for _, fast := range grid["EMAFast"] {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("uwgn/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}
}

// TestUWGNRescueGridsStayWide — аналог TestUWGNGridsStayWide для девяти файлов второго круга
// (cal_w24_*.json, план docs/superpowers/plans/2026-09-04-uwgn-rescue-round.md, Task R1). Оси —
// буквальная копия первого круга: замеры инструмента не изменились, изменилось расчётное окно
// (24 месяца, 2024-09-04..2026-09-04, вместо 30-месячного). Темы volume и vol_window во втором
// круге не заводятся (см. план), поэтому в таблице их нет.
func TestUWGNRescueGridsStayWide(t *testing.T) {
	cases := []struct {
		file   string
		field  string
		values []float64
	}{
		{"cal_w24_screen.json", "UseDayATRGate", []float64{0, 1}},
		{"cal_w24_screen.json", "UseVolume", []float64{0, 1}},
		{"cal_w24_entry.json", "RSILower", []float64{10, 15, 20, 25, 30, 35, 40, 45, 50}},
		{"cal_w24_entry.json", "RSIPeriod", []float64{2, 3, 4, 5, 6, 7, 8, 10}},
		{"cal_w24_entry.json", "RSIUpper", []float64{55, 60, 65, 70, 75, 80, 85}},
		{"cal_w24_trend.json", "EMAFast", []float64{3, 5, 8, 10, 15, 20, 30, 40}},
		{"cal_w24_trend.json", "EMASlow", []float64{50, 75, 100, 150, 200, 250}},
		{"cal_w24_trend_low.json", "EMAFast", []float64{3, 5, 8, 10}},
		{"cal_w24_trend_low.json", "EMASlow", []float64{20, 30, 40}},
		{"cal_w24_day.json", "FreshDayATR", []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5}},
		{"cal_w24_day.json", "SpentDayATR", []float64{0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_w24_day_spent.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5, 2.0}},
		{"cal_w24_risk.json", "StopDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.3, 1.5, 2.0}},
		{"cal_w24_risk.json", "TPDailyATR", []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5}},
		{"cal_w24_exit.json", "RSIUpper", []float64{35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90}},
		{"cal_w24_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_w24_trail.json", "UseRSIExit", []float64{0, 1}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "uwgn", c.file)
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
				t.Errorf("uwgn/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}
