package backtest

import "testing"

// magnGridFiles перечисляет двенадцать файлов темы MAGN ПОИМЁННО, а не обходом каталога
// data/params/rsi_pullback/magn целиком. Причина решения контроллера: задачи 9, 12 и 13 позже
// положат в этот же каталог зондовые файлы (probe_stop_*.json, plateau_*.json, r2_*.json),
// которые обходом упадут на инвариантах осей ниже (например, зонд может законно зафиксировать
// RSILower на одном значении вне требования "оба края 5 и 50", потому что зонд — не сетка).
var magnGridFiles = []string{
	"cal_screen.json",
	"cal_entry.json",
	"cal_trend.json",
	"cal_trend_hump.json",
	"cal_day.json",
	"cal_day_spent.json",
	"cal_volume.json",
	"cal_vol_window.json",
	"cal_risk.json",
	"cal_exit.json",
	"cal_trail.json",
	"cal_trend_volume.json",
}

// TestMAGNGridsStayWide читает поимённый каталог сеток MAGN (data/params/rsi_pullback/magn) и
// проверяет все инварианты §5.1 спеки docs/superpowers/specs/2026-09-10-magn-rsi-pullback-prep-design.md:
//   - RSILower <= 50 во всех файлах, где эта ось встречается;
//   - ось RSILower в cal_entry.json содержит оба края 5 и 50 — рабочая часть оси лежит в 0.73–1.05
//     без внутреннего максимума за пределами дефолта, узел 5 вырожден (4 сделки), узел 10 на грани
//     (19), но оба обязаны остаться в сетке как зонды за краем;
//   - RSIPeriod >= 2 везде, где эта ось встречается;
//   - StopDailyATR != 0 везде, где эта ось встречается — стопless многодневная сделка запрещена;
//   - ни cal_trend.json, ни cal_trend_hump.json, ни cal_trend_volume.json не порождают пар
//     EMAFast >= EMASlow (декартово произведение осей внутри каждого файла); cal_trend_volume.json
//     не свипует EMAFast вовсе — ядро использует дефолт 10, поэтому проверка берёт его как
//     единственное значение фазы, а ось EMASlow там не содержит значений <= 10;
//   - ось EMASlow в cal_trend_hump.json содержит узлы горба 12, 18 и 45 — вершина трендовой оси
//     MAGN (двумерно 10x18 -> 1.812/130) стоит ниже нижнего края канонической темы 50, отсюда
//     разделение на две темы;
//   - ось EMASlow в cal_trend.json содержит верхний край 250;
//   - ось RSIUpper в cal_exit.json содержит оба края 35, 95 — сигнала на этой оси нет (размах
//     0.13 PF), поэтому уплотнения не требуется, но края обязаны остаться;
//   - ось StopDailyATR в cal_risk.json содержит узлы 0.55, 0.65, 0.75 и верхний край 2.0 — капкан
//     широкого стопа встаёт уже на 0.7 (пул замирает на 137 сделках), узлы выше 0.7 остаются в
//     сетке ради ширины и отсекаются риск-гейтом A при сборке точки, а не вырезаются из оси;
//   - ось VolMult в cal_volume.json содержит 1.8 и верхний край 3.0 — объёмный гейт MAGN монотонно
//     улучшает дефолт к жёстким порогам (второй тикер каталога после NKHP с работающим гейтом), а
//     каталожный верх 2.5 стоял бы ещё на растущем участке, поэтому ось расширена до 3.0;
//   - ось FreshDayATR в cal_day.json содержит 0 и 0.5;
//   - ось EMASlow в cal_trend_volume.json содержит 18 и 20, ось VolMult — 1.5 и 1.8 — тема заведена
//     не ради нового оптимума, а ради честного walk-forward по связке двух единственных рабочих
//     рычагов бумаги (тренд и объём).
//
// Владелец требует максимально широкие оси (§5.1: "Сетки держатся максимально широкими"), поэтому
// тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие обязательных узлов и
// отсутствие запрещённых пар, но не запрещает лишних значений в сетке.
func TestMAGNGridsStayWide(t *testing.T) {
	// RSILower <= 50 везде, где ось встречается в поимённом каталоге MAGN.
	for _, file := range magnGridFiles {
		grid := rsiPullbackTickerGrid(t, "magn", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("magn/%s: RSILower=%v нарушает инвариант RSILower <= 50", file, v)
			}
		}
	}

	// Обязательные края оси RSILower в cal_entry.json: 5 и 50.
	{
		grid := rsiPullbackTickerGrid(t, "magn", "cal_entry.json")
		for _, want := range []float64{5, 50} {
			if !containsFloat(grid["RSILower"], want) {
				t.Errorf("magn/cal_entry.json: ось RSILower потеряла обязательный край %v (есть %v)", want, grid["RSILower"])
			}
		}
	}

	// RSIPeriod >= 2 везде, где ось встречается в поимённом каталоге MAGN.
	for _, file := range magnGridFiles {
		grid := rsiPullbackTickerGrid(t, "magn", file)
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("magn/%s: RSIPeriod=%v нарушает инвариант RSIPeriod >= 2", file, v)
			}
		}
	}

	// StopDailyATR != 0 везде, где ось встречается в поимённом каталоге MAGN.
	for _, file := range magnGridFiles {
		grid := rsiPullbackTickerGrid(t, "magn", file)
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("magn/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
	}

	// EMAFast < EMASlow во всех парах трёх трендовых тем. cal_trend_volume.json не свипует
	// EMAFast — ядро использует дефолт 10, поэтому при пустой оси подставляем именно его.
	for _, file := range []string{"cal_trend.json", "cal_trend_hump.json", "cal_trend_volume.json"} {
		grid := rsiPullbackTickerGrid(t, "magn", file)
		fastValues := grid["EMAFast"]
		if len(fastValues) == 0 {
			fastValues = []float64{10} // дефолт ядра, когда EMAFast в файле не свипуется
		}
		for _, fast := range fastValues {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("magn/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}

	// Обязательные узлы горба в cal_trend_hump.json: 12, 18, 45.
	{
		grid := rsiPullbackTickerGrid(t, "magn", "cal_trend_hump.json")
		for _, want := range []float64{12, 18, 45} {
			if !containsFloat(grid["EMASlow"], want) {
				t.Errorf("magn/cal_trend_hump.json: ось EMASlow потеряла обязательный узел горба %v (есть %v)", want, grid["EMASlow"])
			}
		}
	}

	// Верхний край канонической темы trend: EMASlow содержит 250.
	{
		grid := rsiPullbackTickerGrid(t, "magn", "cal_trend.json")
		if !containsFloat(grid["EMASlow"], 250) {
			t.Errorf("magn/cal_trend.json: ось EMASlow потеряла верхний край 250 (есть %v)", grid["EMASlow"])
		}
	}

	// Оба края оси выхода: RSIUpper содержит 35 и 95.
	{
		grid := rsiPullbackTickerGrid(t, "magn", "cal_exit.json")
		for _, want := range []float64{35, 95} {
			if !containsFloat(grid["RSIUpper"], want) {
				t.Errorf("magn/cal_exit.json: ось RSIUpper потеряла обязательный край %v (есть %v)", want, grid["RSIUpper"])
			}
		}
	}

	// Узлы уплотнения и верхний край оси стопа в cal_risk.json: 0.55, 0.65, 0.75, 2.0.
	{
		grid := rsiPullbackTickerGrid(t, "magn", "cal_risk.json")
		for _, want := range []float64{0.55, 0.65, 0.75, 2.0} {
			if !containsFloat(grid["StopDailyATR"], want) {
				t.Errorf("magn/cal_risk.json: ось StopDailyATR потеряла обязательный узел %v (есть %v)", want, grid["StopDailyATR"])
			}
		}
	}

	// Узел уплотнения и верхний край оси объёма в cal_volume.json: 1.8 и 3.0.
	{
		grid := rsiPullbackTickerGrid(t, "magn", "cal_volume.json")
		for _, want := range []float64{1.8, 3.0} {
			if !containsFloat(grid["VolMult"], want) {
				t.Errorf("magn/cal_volume.json: ось VolMult потеряла обязательный узел %v (есть %v)", want, grid["VolMult"])
			}
		}
	}

	// Обязательные края оси FreshDayATR в cal_day.json: 0 и 0.5.
	{
		grid := rsiPullbackTickerGrid(t, "magn", "cal_day.json")
		for _, want := range []float64{0, 0.5} {
			if !containsFloat(grid["FreshDayATR"], want) {
				t.Errorf("magn/cal_day.json: ось FreshDayATR потеряла обязательный узел %v (есть %v)", want, grid["FreshDayATR"])
			}
		}
	}

	// Обязательные узлы связки в cal_trend_volume.json: EMASlow 18 и 20, VolMult 1.5 и 1.8.
	{
		grid := rsiPullbackTickerGrid(t, "magn", "cal_trend_volume.json")
		for _, want := range []float64{18, 20} {
			if !containsFloat(grid["EMASlow"], want) {
				t.Errorf("magn/cal_trend_volume.json: ось EMASlow потеряла обязательный узел %v (есть %v)", want, grid["EMASlow"])
			}
		}
		for _, want := range []float64{1.5, 1.8} {
			if !containsFloat(grid["VolMult"], want) {
				t.Errorf("magn/cal_trend_volume.json: ось VolMult потеряла обязательный узел %v (есть %v)", want, grid["VolMult"])
			}
		}
	}
}
