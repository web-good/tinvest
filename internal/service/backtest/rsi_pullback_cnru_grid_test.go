package backtest

import "testing"

// TestCNRUGridsStayWide читает каталог сеток CNRU (data/params/rsi_pullback/cnru) и проверяет
// все жёсткие инварианты §5.1 спеки docs/superpowers/specs/2026-09-08-cnru-rsi-pullback-prep-design.md:
//   - RSILower <= 50 во всех файлах, где эта ось встречается;
//   - ось RSILower в cal_entry.json содержит узел 5 и все узлы плато 22, 24, 25, 26, 28
//     (плато измерено шагом 1 до написания сеток, §4.1 спеки, и накрыто ВНУТРИ двумерной темы
//     entry, а не отдельной entry_deep, потому что RSIPeriod и RSILower взаимодействуют);
//   - RSIPeriod >= 2 везде, где эта ось встречается;
//   - StopDailyATR != 0 везде, где эта ось встречается (стопless многодневная сделка запрещена);
//   - ни cal_trend.json, ни cal_trend_hump.json не порождают пар EMAFast >= EMASlow (декартово
//     произведение осей внутри каждого файла) — горб 12-45 недоступен канонической сетке без
//     таких пар, поэтому тема разделена на две;
//   - ось EMASlow в cal_trend_hump.json содержит узлы горба 30, 35, 40, 45;
//   - ось EMASlow в cal_trend.json содержит верхний край 250;
//   - ось VolBaseDays в cal_volume.json содержит 50 (расширение вдвое за канонический край 20 —
//     §4.5 спеки, максимум оси стоит на 40 и рост не прекращается до 50);
//   - ось VolMult в cal_volume.json содержит 4.0;
//   - ось RSIUpper в cal_exit.json содержит оба края 35 и 95.
//
// Владелец потребовал максимально широкие оси (§5.1: "Сетки держатся максимально широкими"),
// поэтому тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие обязательных узлов
// и отсутствие запрещённых пар, но не запрещает лишних значений в сетке.
func TestCNRUGridsStayWide(t *testing.T) {
	// RSILower <= 50 везде, где ось встречается в каталоге CNRU (только cal_entry.json).
	for _, file := range []string{"cal_entry.json"} {
		grid := rsiPullbackTickerGrid(t, "cnru", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("cnru/%s: RSILower=%v нарушает инвариант RSILower <= 50", file, v)
			}
		}
	}

	// Обязательные узлы оси RSILower в cal_entry.json: нижний зонд 5 и всё плато 22-28.
	{
		grid := rsiPullbackTickerGrid(t, "cnru", "cal_entry.json")
		for _, want := range []float64{5, 22, 24, 25, 26, 28} {
			if !containsFloat(grid["RSILower"], want) {
				t.Errorf("cnru/cal_entry.json: ось RSILower потеряла обязательный узел %v (есть %v)", want, grid["RSILower"])
			}
		}
	}

	// RSIPeriod >= 2 везде, где ось встречается (только cal_entry.json).
	for _, file := range []string{"cal_entry.json"} {
		grid := rsiPullbackTickerGrid(t, "cnru", file)
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("cnru/%s: RSIPeriod=%v нарушает инвариант RSIPeriod >= 2", file, v)
			}
		}
	}

	// StopDailyATR != 0 везде, где ось встречается (только cal_risk.json).
	for _, file := range []string{"cal_risk.json"} {
		grid := rsiPullbackTickerGrid(t, "cnru", file)
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("cnru/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
	}

	// EMAFast < EMASlow во всех парах ОБЕИХ трендовых тем.
	for _, file := range []string{"cal_trend.json", "cal_trend_hump.json"} {
		grid := rsiPullbackTickerGrid(t, "cnru", file)
		for _, fast := range grid["EMAFast"] {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("cnru/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}

	// Обязательные узлы горба в cal_trend_hump.json: 30, 35, 40, 45.
	{
		grid := rsiPullbackTickerGrid(t, "cnru", "cal_trend_hump.json")
		for _, want := range []float64{30, 35, 40, 45} {
			if !containsFloat(grid["EMASlow"], want) {
				t.Errorf("cnru/cal_trend_hump.json: ось EMASlow потеряла обязательный узел горба %v (есть %v)", want, grid["EMASlow"])
			}
		}
	}

	// Верхний край канонической темы trend: EMASlow содержит 250.
	{
		grid := rsiPullbackTickerGrid(t, "cnru", "cal_trend.json")
		if !containsFloat(grid["EMASlow"], 250) {
			t.Errorf("cnru/cal_trend.json: ось EMASlow потеряла верхний край 250 (есть %v)", grid["EMASlow"])
		}
	}

	// Расширенные края объёмной темы: VolBaseDays содержит 50, VolMult содержит 4.0.
	{
		grid := rsiPullbackTickerGrid(t, "cnru", "cal_volume.json")
		if !containsFloat(grid["VolBaseDays"], 50) {
			t.Errorf("cnru/cal_volume.json: ось VolBaseDays потеряла расширенный край 50 (есть %v)", grid["VolBaseDays"])
		}
		if !containsFloat(grid["VolMult"], 4.0) {
			t.Errorf("cnru/cal_volume.json: ось VolMult потеряла расширенный край 4.0 (есть %v)", grid["VolMult"])
		}
	}

	// Оба края оси выхода: RSIUpper содержит 35 и 95.
	{
		grid := rsiPullbackTickerGrid(t, "cnru", "cal_exit.json")
		for _, want := range []float64{35, 95} {
			if !containsFloat(grid["RSIUpper"], want) {
				t.Errorf("cnru/cal_exit.json: ось RSIUpper потеряла край %v (есть %v)", want, grid["RSIUpper"])
			}
		}
	}
}

// containsFloat сообщает, встречается ли want среди values. Точное сравнение достаточно: все
// узлы сеток каталога заданы литералами без вычислений с плавающей точкой.
func containsFloat(values []float64, want float64) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
