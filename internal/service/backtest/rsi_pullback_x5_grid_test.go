package backtest

import (
	"os"
	"path/filepath"
	"testing"
)

// x5GridFiles перечисляет одиннадцать файлов темы X5 ПОИМЁННО, а не обходом каталога
// data/params/rsi_pullback/x5 целиком. Причина решения контроллера та же, что у IRKT и MAGN
// (rsi_pullback_irkt_grid_test.go, rsi_pullback_magn_grid_test.go): будущие задачи плана
// (сборка точки, зонды капкана широкого стопа) могут положить в этот же каталог файлы-точки
// (plateau_*.json, probe_*.json), которые обходом упадут на инвариантах осей ниже — файл-точка
// законно фиксирует ось на одном значении, потому что точка не сетка.
var x5GridFiles = []string{
	"cal_screen.json",
	"cal_entry.json",
	"cal_trend.json",
	"cal_trend_hump.json",
	"cal_day.json",
	"cal_day_fresh.json",
	"cal_volume.json",
	"cal_vol_window.json",
	"cal_risk.json",
	"cal_exit.json",
	"cal_trail.json",
}

// TestX5GridsStayWide читает поимённый каталог сеток X5 (data/params/rsi_pullback/x5) и
// проверяет все инварианты §5.1 спеки docs/superpowers/specs/2026-09-13-x5-rsi-pullback-prep-design.md:
//   - RSILower <= 50 во всех файлах, где эта ось встречается, а ось RSILower в cal_entry.json
//     содержит оба края 5 и 50;
//   - RSIPeriod >= 2 везде, где эта ось встречается;
//   - StopDailyATR != 0 везде, где эта ось встречается — стопless многодневная сделка запрещена;
//   - ни cal_trend.json, ни cal_trend_hump.json не порождают пар EMAFast >= EMASlow (декартово
//     произведение осей внутри каждого файла);
//   - ось EMASlow в cal_trend_hump.json содержит края горба 12 и 45;
//   - ось EMASlow в cal_trend.json содержит верхний край 250;
//   - ось RSIUpper в cal_exit.json содержит оба края 35 и 95;
//   - ось StopDailyATR в cal_risk.json содержит узлы уплотнения 0.35, 0.45, 0.55, 0.65 и верхний
//     край 2.0;
//   - ось TPDailyATR в cal_risk.json содержит верхний край 2.0;
//   - ось VolLookbackBars в cal_vol_window.json содержит узлы 12, 16, 24 и верхний край 32;
//   - ось FreshDayATR в cal_day_fresh.json содержит узлы уплотнения 0.05, 0.15, 0.25;
//   - ось FreshDayATR в cal_day.json содержит края 0 и 0.5;
//   - файла cal_trend_volume.json в каталоге X5 нет — тема не заводится (§5.1: складывающегося
//     сигнала двух рычагов на X5 нет, прецедент MAGN, где OOS её не пережила); инвариант
//     проверяется явным os.Stat, а не отсутствием в поимённом списке файлов выше, чтобы
//     исполнитель будущей правки не завёл файл молча.
//
// Владелец требует максимально широкие оси (§5.1: сетки держатся максимально широкими), поэтому
// тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие обязательных узлов и
// отсутствие запрещённых пар, но не запрещает лишних значений в сетке.
func TestX5GridsStayWide(t *testing.T) {
	// RSILower <= 50 везде, где ось встречается в поимённом каталоге X5.
	for _, file := range x5GridFiles {
		grid := rsiPullbackTickerGrid(t, "x5", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("x5/%s: RSILower=%v нарушает инвариант RSILower <= 50", file, v)
			}
		}
	}

	// Обязательные края оси RSILower в cal_entry.json: 5 и 50.
	{
		grid := rsiPullbackTickerGrid(t, "x5", "cal_entry.json")
		for _, want := range []float64{5, 50} {
			if !containsFloat(grid["RSILower"], want) {
				t.Errorf("x5/cal_entry.json: ось RSILower потеряла обязательный край %v (есть %v)", want, grid["RSILower"])
			}
		}
	}

	// RSIPeriod >= 2 везде, где ось встречается в поимённом каталоге X5.
	for _, file := range x5GridFiles {
		grid := rsiPullbackTickerGrid(t, "x5", file)
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("x5/%s: RSIPeriod=%v нарушает инвариант RSIPeriod >= 2", file, v)
			}
		}
	}

	// StopDailyATR != 0 везде, где ось встречается в поимённом каталоге X5.
	for _, file := range x5GridFiles {
		grid := rsiPullbackTickerGrid(t, "x5", file)
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("x5/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
	}

	// EMAFast < EMASlow во всех парах двух трендовых тем.
	for _, file := range []string{"cal_trend.json", "cal_trend_hump.json"} {
		grid := rsiPullbackTickerGrid(t, "x5", file)
		fastValues := grid["EMAFast"]
		for _, fast := range fastValues {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("x5/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}

	// Обязательные края горба в cal_trend_hump.json: EMASlow 12 и 45.
	{
		grid := rsiPullbackTickerGrid(t, "x5", "cal_trend_hump.json")
		for _, want := range []float64{12, 45} {
			if !containsFloat(grid["EMASlow"], want) {
				t.Errorf("x5/cal_trend_hump.json: ось EMASlow потеряла обязательный край горба %v (есть %v)", want, grid["EMASlow"])
			}
		}
	}

	// Верхний край канонической темы trend: EMASlow содержит 250.
	{
		grid := rsiPullbackTickerGrid(t, "x5", "cal_trend.json")
		if !containsFloat(grid["EMASlow"], 250) {
			t.Errorf("x5/cal_trend.json: ось EMASlow потеряла верхний край 250 (есть %v)", grid["EMASlow"])
		}
	}

	// Оба края оси выхода: RSIUpper содержит 35 и 95.
	{
		grid := rsiPullbackTickerGrid(t, "x5", "cal_exit.json")
		for _, want := range []float64{35, 95} {
			if !containsFloat(grid["RSIUpper"], want) {
				t.Errorf("x5/cal_exit.json: ось RSIUpper потеряла обязательный край %v (есть %v)", want, grid["RSIUpper"])
			}
		}
	}

	// Узлы уплотнения и верхний край оси стопа в cal_risk.json: 0.35, 0.45, 0.55, 0.65, 2.0.
	{
		grid := rsiPullbackTickerGrid(t, "x5", "cal_risk.json")
		for _, want := range []float64{0.35, 0.45, 0.55, 0.65, 2.0} {
			if !containsFloat(grid["StopDailyATR"], want) {
				t.Errorf("x5/cal_risk.json: ось StopDailyATR потеряла обязательный узел %v (есть %v)", want, grid["StopDailyATR"])
			}
		}
	}

	// Верхний край оси TPDailyATR в cal_risk.json: 2.0.
	{
		grid := rsiPullbackTickerGrid(t, "x5", "cal_risk.json")
		if !containsFloat(grid["TPDailyATR"], 2.0) {
			t.Errorf("x5/cal_risk.json: ось TPDailyATR потеряла верхний край 2.0 (есть %v)", grid["TPDailyATR"])
		}
	}

	// Узлы уплотнения и верхний край оси VolLookbackBars в cal_vol_window.json: 12, 16, 24, 32.
	{
		grid := rsiPullbackTickerGrid(t, "x5", "cal_vol_window.json")
		for _, want := range []float64{12, 16, 24, 32} {
			if !containsFloat(grid["VolLookbackBars"], want) {
				t.Errorf("x5/cal_vol_window.json: ось VolLookbackBars потеряла обязательный узел %v (есть %v)", want, grid["VolLookbackBars"])
			}
		}
	}

	// Узлы уплотнения оси FreshDayATR в cal_day_fresh.json: 0.05, 0.15, 0.25.
	{
		grid := rsiPullbackTickerGrid(t, "x5", "cal_day_fresh.json")
		for _, want := range []float64{0.05, 0.15, 0.25} {
			if !containsFloat(grid["FreshDayATR"], want) {
				t.Errorf("x5/cal_day_fresh.json: ось FreshDayATR потеряла обязательный узел %v (есть %v)", want, grid["FreshDayATR"])
			}
		}
	}

	// Обязательные края оси FreshDayATR в cal_day.json: 0 и 0.5.
	{
		grid := rsiPullbackTickerGrid(t, "x5", "cal_day.json")
		for _, want := range []float64{0, 0.5} {
			if !containsFloat(grid["FreshDayATR"], want) {
				t.Errorf("x5/cal_day.json: ось FreshDayATR потеряла обязательный узел %v (есть %v)", want, grid["FreshDayATR"])
			}
		}
	}

	// Тема trend_volume не заводится: файла cal_trend_volume.json в каталоге X5 быть не должно.
	// Проверка явная, через os.Stat, а не через отсутствие в x5GridFiles — иначе исполнитель
	// будущей правки мог бы завести файл молча, просто не дописав его в поимённый список.
	{
		path := filepath.Join(rsiPullbackParamsDir, "x5", "cal_trend_volume.json")
		if _, err := os.Stat(path); err == nil {
			t.Errorf("x5/cal_trend_volume.json существует, но тема trend_volume не должна заводиться (§5.1)")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}
