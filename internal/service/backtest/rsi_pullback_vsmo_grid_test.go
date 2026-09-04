package backtest

import "testing"

// TestVSMOGridsStayWide держит НИЖНЮЮ границу ширины сеток VSMO. Владелец потребовал максимально
// широкие оси, поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие
// обязательных значений и не запрещает лишних. Четыре отступления от канона §8.1 посажены на
// замеры полного окна 2026-09-04 (спека docs/superpowers/specs/2026-09-04-vsmo-rsi-pullback-prep-design.md,
// §3) и пиньятся здесь, чтобы редактор сеток не «починил» их обратно к канону:
//
//   - RSIPeriod ВНИЗ до 2 и ВВЕРХ до 10 (канон 3..8). Ось на VSMO двугорбая: 2 -> 1.534/360,
//     3 -> 1.585/199, 4 -> 1.658/144 (дефолт), 5 -> 1.448/107, 6 -> 1.707/73, 7 -> 1.643/56,
//     8 -> 1.736/49, 10 -> 1.072/30. Длинный RSI даёт чуть более высокий PF на втрое меньшей
//     выборке, поэтому верхний край нужен, чтобы отделить «максимум внутри оси» от «максимум на
//     краю»; 12 (0.745/17) и 14 (0.626/11) убыточны и вырождены — в сетку не идут вовсе.
//   - EMAFast 8 ДОБАВЛЕН в обе сетки тренда (канон 3, 5, 10, 20, 30, 40). Точечно именно восьмёрка
//     держит максимум оси: 3 -> 1.802/130, 5 -> 1.720/137, 8 -> 1.841/141, 10 -> 1.658/144
//     (дефолт), 15 -> 1.462/151, 40 -> 1.389/158. Канон этот узел не содержит, и без него максимум
//     оси был бы недостижим для калибровки.
//   - ТЕМА ТРЕНДА РАЗЛОЖЕНА НА ДВЕ СЕТКИ. Каноническая cal_trend (EMAFast 3..40 x EMASlow 50..250)
//     несёт планку и сравнение с каталогом; нижний угол медленной оси (EMASlow 20 -> 1.699/130,
//     30 -> 1.666/134, 40) канонической сетке при EMAFast до 40 недоступен из-за инварианта
//     EMAFast < EMASlow, и его мерит отдельная cal_trend_low. Обе оси на VSMO пологие — разброс
//     всей оси EMASlow укладывается в 0.35 PF при почти неизменном числе сделок.
//   - StopDailyATR получает узлы 0.4, 0.6 и 0.8 сверх канона (0.3, 0.5, 0.7, 1.0, 1.3, 1.5, 2.0).
//     Узел 0.8 — первая точка КАПКАНА ШИРОКОГО СТОПА, где число сделок перестаёт меняться:
//     0.5 -> 1.658/144, 0.6 -> 1.675/143, 0.7 -> 1.661/139, 0.8 -> 2.034/136, 1.0 -> 2.224/136,
//     1.3 -> 2.167/136, 1.5 -> 2.593/135, 2.0 -> 2.870/135. Рост PF в полтора раза при побайтово
//     неизменном пуле означает, что стоп не отбирает входы, а только отодвигает фиксацию убытка;
//     узлы 0.4 и 0.6 нужны, чтобы граница капкана была видна в самой сетке, а не додумывалась.
//
// Верхняя граница ширины держится не здесь, а риск-гейтом A (потолок StopDailyATR 1.0 при
// выживаемости 38.6% будних дней) и инвариантными тестами ниже.
func TestVSMOGridsStayWide(t *testing.T) {
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
		{"cal_day.json", "FreshDayATR", []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6}},
		{"cal_day.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5}},
		{"cal_day_spent.json", "SpentDayATR", []float64{0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.25, 1.5, 2.0}},
		{"cal_volume.json", "VolMult", []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0, 4.0}},
		{"cal_volume.json", "VolBaseDays", []float64{3, 5, 10, 14, 20}},
		{"cal_vol_window.json", "VolLookbackBars", []float64{1, 2, 3, 5, 8, 12, 16, 24, 32}},
		{"cal_vol_window.json", "VolMult", []float64{1.2, 2.5, 4.0}},
		{"cal_risk.json", "StopDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.3, 1.5, 2.0}},
		{"cal_risk.json", "TPDailyATR", []float64{0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.0, 2.5}},
		{"cal_exit.json", "RSIUpper", []float64{35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90}},
		{"cal_trail.json", "TrailDailyATR", []float64{0.3, 0.5, 0.7, 1.0, 1.5}},
		{"cal_trail.json", "UseRSIExit", []float64{0, 1}},
	}
	for _, c := range cases {
		grid := rsiPullbackTickerGrid(t, "vsmo", c.file)
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
				t.Errorf("vsmo/%s: ось %s потеряла значение %v (есть %v) — ось УРЕЗАНА, а владелец требовал максимально широкую", c.file, c.field, v, got)
			}
		}
	}
}

// TestVSMOEntryGridKeepsRSIUpperAboveRSILower сторожит инвариант, который расширение осей входа
// может нарушить незаметно: нижняя граница поднята до 50, верхняя опущена до 55, и одна лишняя
// точка в любой из осей даёт пару, где выход стоит НИЖЕ входа. Такая пара не ошибка запуска — ядро
// её честно посчитает и вернёт мусорную конфигурацию, которая войдёт в ранжирование темы.
func TestVSMOEntryGridKeepsRSIUpperAboveRSILower(t *testing.T) {
	grid := rsiPullbackTickerGrid(t, "vsmo", "cal_entry.json")
	for _, lower := range grid["RSILower"] {
		for _, upper := range grid["RSIUpper"] {
			if upper <= lower {
				t.Errorf("vsmo/cal_entry.json: пара RSILower=%v, RSIUpper=%v нарушает RSIUpper > RSILower", lower, upper)
			}
		}
	}
}

// TestVSMOTrendGridsKeepFastBelowSlow проверяет ОБА файла тренда. Разложение темы на каноническую
// сетку и нижний угол существует ровно потому, что инвариант EMAFast < EMASlow не позволяет мерить
// EMASlow 20..40 в одной сетке с EMAFast до 40; если кто-нибудь сольёт файлы обратно, тест это
// поймает.
func TestVSMOTrendGridsKeepFastBelowSlow(t *testing.T) {
	for _, file := range []string{"cal_trend.json", "cal_trend_low.json"} {
		grid := rsiPullbackTickerGrid(t, "vsmo", file)
		for _, fast := range grid["EMAFast"] {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("vsmo/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}
}
