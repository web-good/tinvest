package backtest

import (
	"os"
	"path/filepath"
	"testing"
)

// irktGridFiles перечисляет одиннадцать файлов темы IRKT ПОИМЁННО, а не обходом каталога
// data/params/rsi_pullback/irkt целиком. Причина решения контроллера та же, что у MAGN
// (rsi_pullback_magn_grid_test.go): будущие задачи 3-10 могут положить в этот же каталог
// зондовые файлы (probe_*.json, plateau_*.json, r2_*.json), которые обходом упадут на
// инвариантах осей ниже — зонд законно фиксирует ось на одном значении, потому что зонд не
// сетка.
var irktGridFiles = []string{
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
}

// TestIRKTGridsStayWide читает поимённый каталог сеток IRKT (data/params/rsi_pullback/irkt) и
// проверяет все инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-11-irkt-rsi-pullback-prep-design.md:
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
//   - ось SpentDayATR в cal_day_spent.json содержит узлы уплотнения 1.05, 1.15, 1.3, 1.4, 1.75 и
//     верхний край 2.0;
//   - ось TrailDailyATR в cal_trail.json содержит узлы уплотнения 0.4 и 0.6;
//   - ось FreshDayATR в cal_day.json содержит края 0 и 0.5;
//   - файла cal_trend_volume.json в каталоге IRKT нет — тема заведена ради полноты каталога, а не
//     по априору (§5.1: объёмный гейт на IRKT в лучшем случае инертен), и связка тренда с объёмом
//     не проверяется вовсе; инвариант проверяется явным os.Stat, а не отсутствием в списке файлов,
//     чтобы исполнитель будущей правки не завёл файл молча, не попав в поимённый список выше.
//
// Владелец требует максимально широкие оси (§5.1: "Сетки держатся максимально широкими"), поэтому
// тест запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие обязательных узлов и
// отсутствие запрещённых пар, но не запрещает лишних значений в сетке.
func TestIRKTGridsStayWide(t *testing.T) {
	// RSILower <= 50 везде, где ось встречается в поимённом каталоге IRKT.
	for _, file := range irktGridFiles {
		grid := rsiPullbackTickerGrid(t, "irkt", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("irkt/%s: RSILower=%v нарушает инвариант RSILower <= 50", file, v)
			}
		}
	}

	// Обязательные края оси RSILower в cal_entry.json: 5 и 50.
	{
		grid := rsiPullbackTickerGrid(t, "irkt", "cal_entry.json")
		for _, want := range []float64{5, 50} {
			if !containsFloat(grid["RSILower"], want) {
				t.Errorf("irkt/cal_entry.json: ось RSILower потеряла обязательный край %v (есть %v)", want, grid["RSILower"])
			}
		}
	}

	// RSIPeriod >= 2 везде, где ось встречается в поимённом каталоге IRKT.
	for _, file := range irktGridFiles {
		grid := rsiPullbackTickerGrid(t, "irkt", file)
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("irkt/%s: RSIPeriod=%v нарушает инвариант RSIPeriod >= 2", file, v)
			}
		}
	}

	// StopDailyATR != 0 везде, где ось встречается в поимённом каталоге IRKT.
	for _, file := range irktGridFiles {
		grid := rsiPullbackTickerGrid(t, "irkt", file)
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("irkt/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
	}

	// EMAFast < EMASlow во всех парах двух трендовых тем.
	for _, file := range []string{"cal_trend.json", "cal_trend_hump.json"} {
		grid := rsiPullbackTickerGrid(t, "irkt", file)
		fastValues := grid["EMAFast"]
		for _, fast := range fastValues {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("irkt/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}

	// Обязательные края горба в cal_trend_hump.json: EMASlow 12 и 45.
	{
		grid := rsiPullbackTickerGrid(t, "irkt", "cal_trend_hump.json")
		for _, want := range []float64{12, 45} {
			if !containsFloat(grid["EMASlow"], want) {
				t.Errorf("irkt/cal_trend_hump.json: ось EMASlow потеряла обязательный край горба %v (есть %v)", want, grid["EMASlow"])
			}
		}
	}

	// Верхний край канонической темы trend: EMASlow содержит 250.
	{
		grid := rsiPullbackTickerGrid(t, "irkt", "cal_trend.json")
		if !containsFloat(grid["EMASlow"], 250) {
			t.Errorf("irkt/cal_trend.json: ось EMASlow потеряла верхний край 250 (есть %v)", grid["EMASlow"])
		}
	}

	// Оба края оси выхода: RSIUpper содержит 35 и 95.
	{
		grid := rsiPullbackTickerGrid(t, "irkt", "cal_exit.json")
		for _, want := range []float64{35, 95} {
			if !containsFloat(grid["RSIUpper"], want) {
				t.Errorf("irkt/cal_exit.json: ось RSIUpper потеряла обязательный край %v (есть %v)", want, grid["RSIUpper"])
			}
		}
	}

	// Узлы уплотнения и верхний край оси стопа в cal_risk.json: 0.35, 0.45, 0.55, 0.65, 2.0.
	{
		grid := rsiPullbackTickerGrid(t, "irkt", "cal_risk.json")
		for _, want := range []float64{0.35, 0.45, 0.55, 0.65, 2.0} {
			if !containsFloat(grid["StopDailyATR"], want) {
				t.Errorf("irkt/cal_risk.json: ось StopDailyATR потеряла обязательный узел %v (есть %v)", want, grid["StopDailyATR"])
			}
		}
	}

	// Узлы уплотнения и верхний край оси SpentDayATR в cal_day_spent.json: 1.05, 1.15, 1.3, 1.4,
	// 1.75, 2.0.
	{
		grid := rsiPullbackTickerGrid(t, "irkt", "cal_day_spent.json")
		for _, want := range []float64{1.05, 1.15, 1.3, 1.4, 1.75, 2.0} {
			if !containsFloat(grid["SpentDayATR"], want) {
				t.Errorf("irkt/cal_day_spent.json: ось SpentDayATR потеряла обязательный узел %v (есть %v)", want, grid["SpentDayATR"])
			}
		}
	}

	// Узлы уплотнения оси TrailDailyATR в cal_trail.json: 0.4 и 0.6.
	{
		grid := rsiPullbackTickerGrid(t, "irkt", "cal_trail.json")
		for _, want := range []float64{0.4, 0.6} {
			if !containsFloat(grid["TrailDailyATR"], want) {
				t.Errorf("irkt/cal_trail.json: ось TrailDailyATR потеряла обязательный узел %v (есть %v)", want, grid["TrailDailyATR"])
			}
		}
	}

	// Обязательные края оси FreshDayATR в cal_day.json: 0 и 0.5.
	{
		grid := rsiPullbackTickerGrid(t, "irkt", "cal_day.json")
		for _, want := range []float64{0, 0.5} {
			if !containsFloat(grid["FreshDayATR"], want) {
				t.Errorf("irkt/cal_day.json: ось FreshDayATR потеряла обязательный узел %v (есть %v)", want, grid["FreshDayATR"])
			}
		}
	}

	// Тема trend_volume не заводится: файла cal_trend_volume.json в каталоге IRKT быть не должно.
	// Проверка явная, через os.Stat, а не через отсутствие в irktGridFiles — иначе исполнитель
	// будущей правки мог бы завести файл молча, просто не дописав его в поимённый список.
	{
		path := filepath.Join(rsiPullbackParamsDir, "irkt", "cal_trend_volume.json")
		if _, err := os.Stat(path); err == nil {
			t.Errorf("irkt/cal_trend_volume.json существует, но тема trend_volume не должна заводиться (§5.1)")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}
