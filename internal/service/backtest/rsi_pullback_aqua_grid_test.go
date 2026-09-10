package backtest

import "testing"

// TestAQUAGridsStayWide читает каталог сеток AQUA (data/params/rsi_pullback/aqua) и проверяет
// жёсткие инварианты §5.1 спеки docs/superpowers/specs/2026-09-10-aqua-rsi-pullback-prep-design.md:
//   - RSILower <= 50 во всех файлах, где эта ось встречается (только cal_entry.json);
//   - ось RSILower в cal_entry.json содержит оба края 5 и 50 — рельеф входа плоский (§4.1 спеки,
//     размах 0.22 PF, внутреннего максимума нет), поэтому тема entry_deep не заводится и ось не
//     уплотняется, но оба зонда за краями обязаны остаться в сетке;
//   - RSIPeriod >= 2 везде, где эта ось встречается;
//   - StopDailyATR != 0 везде, где эта ось встречается (cal_risk.json и cal_exit_stop.json —
//     стопless многодневная сделка запрещена в обеих);
//   - ни cal_trend.json, ни cal_trend_hump.json не порождают пар EMAFast >= EMASlow (декартово
//     произведение осей внутри каждого файла) — вершина трендовой оси AQUA стоит на EMASlow=12,
//     ниже нижнего края канонической темы 50 (§4.2 спеки), отсюда разделение на две темы;
//   - ось EMASlow в cal_trend_hump.json содержит узлы горба 12, 15, 20, 25;
//   - ось EMASlow в cal_trend.json содержит верхний край 250;
//   - ось RSIUpper в cal_exit.json содержит оба края 35, 95 и узлы уплотнения 40, 42, 44, 46 —
//     единственный рычаг бумаги с внутренним максимумом (42) и живым пулом (§4.8 спеки);
//   - ось StopDailyATR в cal_risk.json содержит узлы уплотнения 0.65, 0.75, 0.85, 0.95 и верхний
//     край 2.0 — капкан широкого стопа начинается на 0.6 (§4.6 спеки), верхние узлы остаются в
//     сетке ради ширины и отсекаются риск-гейтом A при сборке точки, а не вырезанием оси;
//   - ось FreshDayATR в cal_day.json содержит 0 и 0.5 — ранняя ветка гейта дня у AQUA впервые в
//     каталоге лежит выше baseline на всём протяжении (§4.3 спеки), оба края обязаны остаться;
//   - ось RSIUpper в cal_exit_stop.json содержит 42 и 45, ось StopDailyATR — 0.6 и 1.0 — тема
//     владеет парой двух единственных рычагов бумаги совместно (§4.10, §5.2 спеки).
//
// Владелец требует максимально широкие оси (§5.1: "Сетки держатся максимально широкими"), поэтому
// тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие обязательных узлов и
// отсутствие запрещённых пар, но не запрещает лишних значений в сетке.
func TestAQUAGridsStayWide(t *testing.T) {
	// RSILower <= 50 везде, где ось встречается в каталоге AQUA (только cal_entry.json).
	for _, file := range []string{"cal_entry.json"} {
		grid := rsiPullbackTickerGrid(t, "aqua", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("aqua/%s: RSILower=%v нарушает инвариант RSILower <= 50", file, v)
			}
		}
	}

	// Обязательные края оси RSILower в cal_entry.json: 5 и 50.
	{
		grid := rsiPullbackTickerGrid(t, "aqua", "cal_entry.json")
		for _, want := range []float64{5, 50} {
			if !containsFloat(grid["RSILower"], want) {
				t.Errorf("aqua/cal_entry.json: ось RSILower потеряла обязательный край %v (есть %v)", want, grid["RSILower"])
			}
		}
	}

	// RSIPeriod >= 2 везде, где ось встречается (только cal_entry.json).
	for _, file := range []string{"cal_entry.json"} {
		grid := rsiPullbackTickerGrid(t, "aqua", file)
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("aqua/%s: RSIPeriod=%v нарушает инвариант RSIPeriod >= 2", file, v)
			}
		}
	}

	// StopDailyATR != 0 везде, где ось встречается (cal_risk.json и cal_exit_stop.json).
	for _, file := range []string{"cal_risk.json", "cal_exit_stop.json"} {
		grid := rsiPullbackTickerGrid(t, "aqua", file)
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("aqua/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
	}

	// EMAFast < EMASlow во всех парах обеих трендовых тем.
	for _, file := range []string{"cal_trend.json", "cal_trend_hump.json"} {
		grid := rsiPullbackTickerGrid(t, "aqua", file)
		for _, fast := range grid["EMAFast"] {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("aqua/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}

	// Обязательные узлы горба в cal_trend_hump.json: 12, 15, 20, 25.
	{
		grid := rsiPullbackTickerGrid(t, "aqua", "cal_trend_hump.json")
		for _, want := range []float64{12, 15, 20, 25} {
			if !containsFloat(grid["EMASlow"], want) {
				t.Errorf("aqua/cal_trend_hump.json: ось EMASlow потеряла обязательный узел горба %v (есть %v)", want, grid["EMASlow"])
			}
		}
	}

	// Верхний край канонической темы trend: EMASlow содержит 250.
	{
		grid := rsiPullbackTickerGrid(t, "aqua", "cal_trend.json")
		if !containsFloat(grid["EMASlow"], 250) {
			t.Errorf("aqua/cal_trend.json: ось EMASlow потеряла верхний край 250 (есть %v)", grid["EMASlow"])
		}
	}

	// Оба края оси выхода и узлы уплотнения: RSIUpper содержит 35, 95, 40, 42, 44, 46.
	{
		grid := rsiPullbackTickerGrid(t, "aqua", "cal_exit.json")
		for _, want := range []float64{35, 95, 40, 42, 44, 46} {
			if !containsFloat(grid["RSIUpper"], want) {
				t.Errorf("aqua/cal_exit.json: ось RSIUpper потеряла обязательный узел %v (есть %v)", want, grid["RSIUpper"])
			}
		}
	}

	// Узлы уплотнения и верхний край оси стопа в cal_risk.json: 0.65, 0.75, 0.85, 0.95, 2.0.
	{
		grid := rsiPullbackTickerGrid(t, "aqua", "cal_risk.json")
		for _, want := range []float64{0.65, 0.75, 0.85, 0.95, 2.0} {
			if !containsFloat(grid["StopDailyATR"], want) {
				t.Errorf("aqua/cal_risk.json: ось StopDailyATR потеряла обязательный узел %v (есть %v)", want, grid["StopDailyATR"])
			}
		}
	}

	// Обязательные края оси FreshDayATR в cal_day.json: 0 и 0.5.
	{
		grid := rsiPullbackTickerGrid(t, "aqua", "cal_day.json")
		for _, want := range []float64{0, 0.5} {
			if !containsFloat(grid["FreshDayATR"], want) {
				t.Errorf("aqua/cal_day.json: ось FreshDayATR потеряла обязательный узел %v (есть %v)", want, grid["FreshDayATR"])
			}
		}
	}

	// Обязательные узлы связки в cal_exit_stop.json: RSIUpper 42 и 45, StopDailyATR 0.6 и 1.0.
	{
		grid := rsiPullbackTickerGrid(t, "aqua", "cal_exit_stop.json")
		for _, want := range []float64{42, 45} {
			if !containsFloat(grid["RSIUpper"], want) {
				t.Errorf("aqua/cal_exit_stop.json: ось RSIUpper потеряла обязательный узел %v (есть %v)", want, grid["RSIUpper"])
			}
		}
		for _, want := range []float64{0.6, 1.0} {
			if !containsFloat(grid["StopDailyATR"], want) {
				t.Errorf("aqua/cal_exit_stop.json: ось StopDailyATR потеряла обязательный узел %v (есть %v)", want, grid["StopDailyATR"])
			}
		}
	}
}
