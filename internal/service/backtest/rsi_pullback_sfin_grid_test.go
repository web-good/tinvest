package backtest

import (
	"os"
	"path/filepath"
	"testing"
)

// sfinGridFiles перечисляет одиннадцать файлов темы SFIN ПОИМЁННО, а не обходом каталога
// data/params/rsi_pullback/sfin целиком. Причина решения контроллера та же, что у IRKT, MAGN и X5
// (rsi_pullback_irkt_grid_test.go, rsi_pullback_magn_grid_test.go, rsi_pullback_x5_grid_test.go):
// будущие задачи плана (сборка точки, зонды капкана широкого стопа) могут положить в этот же
// каталог файлы-точки (plateau_*.json, probe_*.json), которые обходом упадут на инвариантах осей
// ниже — файл-точка законно фиксирует ось на одном значении, потому что точка не сетка.
var sfinGridFiles = []string{
	"cal_screen.json",
	"cal_entry.json",
	"cal_trend.json",
	"cal_trend_hump.json",
	"cal_trend_volume.json",
	"cal_day.json",
	"cal_volume.json",
	"cal_vol_window.json",
	"cal_risk.json",
	"cal_exit.json",
	"cal_trail.json",
}

// sfinRound2GridFiles перечисляет шесть УЗКИХ сеток второго круга (Task 12 плана, Step 2). Они
// заводятся только по тем темам, чьё поле участвовало в точке первого круга или соседствует с
// лучшей зоной: entry, trend_hump, exit, risk, day, volume. Темы screen, trend, trend_volume,
// vol_window и trail узких сеток не получают по решению контроллера: screen двоичная и решена
// единогласно, trend проиграла trend_hump, trend_volume — арбитр и уже высказалась, vol_window
// проиграла volume по pooled OOS (1.000 против 1.195), trail инертна (все четыре голоса различны,
// ось UseTrail приколочена одним значением).
//
// Список поимённый по той же причине, что и sfinGridFiles: в каталог приезжают файлы-точки
// второго круга (plateau_point2.json, plateau_r2_*.json), которые обходом упали бы на инвариантах
// осей — точка законно фиксирует ось одним значением.
var sfinRound2GridFiles = []string{
	"cal2_entry.json",
	"cal2_trend_hump.json",
	"cal2_exit.json",
	"cal2_risk.json",
	"cal2_day.json",
	"cal2_volume.json",
}

// sfinAllGridFiles — оба круга вместе. Жёсткие инварианты Global Constraints (RSILower <= 50,
// RSIPeriod >= 2, StopDailyATR != 0, отсутствие пар EMAFast >= EMASlow) узкие сетки второго круга
// обязаны держать ровно так же, как широкие сетки первого: сужение зоны поиска не отменяет запретов
// ядра.
func sfinAllGridFiles() []string {
	out := make([]string, 0, len(sfinGridFiles)+len(sfinRound2GridFiles))
	out = append(out, sfinGridFiles...)
	out = append(out, sfinRound2GridFiles...)
	return out
}

// TestSFINGridsStayWide читает поимённый каталог сеток SFIN (data/params/rsi_pullback/sfin) и
// проверяет все инварианты §5.1 спеки
// docs/superpowers/specs/2026-09-15-sfin-rsi-pullback-prep-design.md:
//   - RSILower <= 50 во всех файлах, где эта ось встречается, а ось RSILower в cal_entry.json
//     содержит оба края 5 и 50;
//   - RSIPeriod >= 2 везде, где эта ось встречается;
//   - StopDailyATR != 0 везде, где эта ось встречается — стопless многодневная сделка запрещена;
//   - ни один файл поимённого каталога обоих кругов не порождает пар EMAFast >= EMASlow
//     (декартово произведение осей внутри каждого файла; проверка идёт по всему каталогу, а не по
//     литеральному списку трендовых тем, чтобы будущая трендовая сетка не оказалась освобождённой
//     от неё молча);
//   - ось EMASlow в cal_trend_hump.json содержит края горба 12 и 45;
//   - ось EMASlow в cal_trend.json содержит верхний край 250;
//   - ось RSIUpper в cal_exit.json содержит оба края 35 и 95;
//   - ось StopDailyATR в cal_risk.json содержит узлы уплотнения 0.35, 0.45, 0.55, 0.65 и верхний
//     край 2.0;
//   - ось TPDailyATR в cal_risk.json содержит верхний край рабочей области 2.0 и контрольный узел
//     асимметрии 2.5 (требование TestRSIPullbackGridControlPoints — прецеденты IRKT, MAGN, AQUA,
//     CNRU, SNGSP, HEAD, X5, SMLT);
//   - ось FreshDayATR в cal_day.json содержит узлы уплотнения 0.05, 0.15 и оба края 0 и 0.5;
//   - ось SpentDayATR в cal_day.json содержит верхний край 2.0;
//   - ось VolLookbackBars в cal_vol_window.json содержит верхний край 32;
//   - четыре жёстких инварианта (RSILower <= 50, RSIPeriod >= 2, StopDailyATR != 0, отсутствие пар
//     EMAFast >= EMASlow) держат ОБА круга — и широкие сетки первого, и узкие сетки второго
//     (sfinRound2GridFiles);
//   - ось TPDailyATR в cal2_risk.json содержит дефолт ядра 0.6, а ось FreshDayATR в cal2_day.json —
//     дефолт ядра 0. Оба узла требует постановка второго круга: первый круг вернул на дефолт
//     ТОЛЬКО UseVolume, связка «точка + TPDailyATR 0.6» не пробовалась ни разу, а любой ненулевой
//     FreshDayATR — кандидат на провал риск-гейта C, который вторым кругом не смягчается. Без этих
//     узлов второй круг повторил бы слепое пятно первого, и проверка стоит здесь, чтобы будущая
//     правка не убрала их молча;
//   - файлов cal_entry_deep.json и cal_day_spent.json в каталоге SFIN нет — обе темы намеренно не
//     заводятся (§5.1 спеки: обе оси входа монотонны без провалов между высшими соседями, максимум
//     SpentDayATR широкий), инвариант проверяется явным os.Stat, а не отсутствием в поимённом
//     списке файлов выше, чтобы исполнитель будущей правки не завёл их молча.
//
// Владелец требует максимально широкие оси (сетки держатся максимально широкими), поэтому тест
// запрещает УРЕЗАТЬ ось, а не расширять её: он проверяет наличие обязательных узлов и отсутствие
// запрещённых пар, но не запрещает лишних значений в сетке.
func TestSFINGridsStayWide(t *testing.T) {
	// RSILower <= 50 везде, где ось встречается в поимённом каталоге SFIN (оба круга).
	for _, file := range sfinAllGridFiles() {
		grid := rsiPullbackTickerGrid(t, "sfin", file)
		for _, v := range grid["RSILower"] {
			if v > 50 {
				t.Errorf("sfin/%s: RSILower=%v нарушает инвариант RSILower <= 50", file, v)
			}
		}
	}

	// Обязательные края оси RSILower в cal_entry.json: 5 и 50.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal_entry.json")
		for _, want := range []float64{5, 50} {
			if !containsFloat(grid["RSILower"], want) {
				t.Errorf("sfin/cal_entry.json: ось RSILower потеряла обязательный край %v (есть %v)", want, grid["RSILower"])
			}
		}
	}

	// RSIPeriod >= 2 везде, где ось встречается в поимённом каталоге SFIN (оба круга).
	for _, file := range sfinAllGridFiles() {
		grid := rsiPullbackTickerGrid(t, "sfin", file)
		for _, v := range grid["RSIPeriod"] {
			if v < 2 {
				t.Errorf("sfin/%s: RSIPeriod=%v нарушает инвариант RSIPeriod >= 2", file, v)
			}
		}
	}

	// StopDailyATR != 0 везде, где ось встречается в поимённом каталоге SFIN (оба круга).
	for _, file := range sfinAllGridFiles() {
		grid := rsiPullbackTickerGrid(t, "sfin", file)
		for _, v := range grid["StopDailyATR"] {
			if v == 0 {
				t.Errorf("sfin/%s: StopDailyATR=0 запрещён — стопless многодневная сделка", file)
			}
		}
	}

	// EMAFast < EMASlow во всех парах ВЕЗДЕ, где файл свипует обе оси сразу — обход идёт по тому же
	// склеенному каталогу обоих кругов, что и три инварианта выше, а не по литеральному списку
	// трендовых тем. Литеральный список освободил бы от проверки будущий файл (например,
	// cal2_trend.json или вторую узкую трендовую тему) молча: его достаточно было бы дописать в
	// sfinRound2GridFiles и забыть про эту строку. Файлы, не свипующие обе оси, проходят вырожденно.
	for _, file := range sfinAllGridFiles() {
		grid := rsiPullbackTickerGrid(t, "sfin", file)
		fastValues := grid["EMAFast"]
		for _, fast := range fastValues {
			for _, slow := range grid["EMASlow"] {
				if fast >= slow {
					t.Errorf("sfin/%s: пара EMAFast=%v, EMASlow=%v нарушает EMAFast < EMASlow", file, fast, slow)
				}
			}
		}
	}

	// Обязательные края горба в cal_trend_hump.json: EMASlow 12 и 45.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal_trend_hump.json")
		for _, want := range []float64{12, 45} {
			if !containsFloat(grid["EMASlow"], want) {
				t.Errorf("sfin/cal_trend_hump.json: ось EMASlow потеряла обязательный край горба %v (есть %v)", want, grid["EMASlow"])
			}
		}
	}

	// Верхний край канонической темы trend: EMASlow содержит 250.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal_trend.json")
		if !containsFloat(grid["EMASlow"], 250) {
			t.Errorf("sfin/cal_trend.json: ось EMASlow потеряла верхний край 250 (есть %v)", grid["EMASlow"])
		}
	}

	// Оба края оси выхода: RSIUpper содержит 35 и 95.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal_exit.json")
		for _, want := range []float64{35, 95} {
			if !containsFloat(grid["RSIUpper"], want) {
				t.Errorf("sfin/cal_exit.json: ось RSIUpper потеряла обязательный край %v (есть %v)", want, grid["RSIUpper"])
			}
		}
	}

	// Узлы уплотнения и верхний край оси стопа в cal_risk.json: 0.35, 0.45, 0.55, 0.65, 2.0.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal_risk.json")
		for _, want := range []float64{0.35, 0.45, 0.55, 0.65, 2.0} {
			if !containsFloat(grid["StopDailyATR"], want) {
				t.Errorf("sfin/cal_risk.json: ось StopDailyATR потеряла обязательный узел %v (есть %v)", want, grid["StopDailyATR"])
			}
		}
	}

	// Верхний край рабочей области оси TPDailyATR в cal_risk.json: 2.0. Отдельно — контрольный
	// узел 2.5 (см. _comment файла): он не кандидат, а строка асимметрии cтоп/цель, которую требует
	// репозиторный TestRSIPullbackCalFilesValid / TestRSIPullbackGridControlPoints (файл свипует
	// StopDailyATR до 2.0 — без узла цели строго выше стоп/цель остаются на равных краях). Проверка
	// наличия 2.5 нужна, чтобы будущая правка не убрала контрольную строку молча.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal_risk.json")
		if !containsFloat(grid["TPDailyATR"], 2.0) {
			t.Errorf("sfin/cal_risk.json: ось TPDailyATR потеряла верхний край рабочей области 2.0 (есть %v)", grid["TPDailyATR"])
		}
		if !containsFloat(grid["TPDailyATR"], 2.5) {
			t.Errorf("sfin/cal_risk.json: ось TPDailyATR потеряла контрольный узел асимметрии 2.5, требуемый TestRSIPullbackGridControlPoints (есть %v)", grid["TPDailyATR"])
		}
	}

	// Узлы уплотнения и оба края оси FreshDayATR в cal_day.json: 0, 0.05, 0.15, 0.5.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal_day.json")
		for _, want := range []float64{0, 0.05, 0.15, 0.5} {
			if !containsFloat(grid["FreshDayATR"], want) {
				t.Errorf("sfin/cal_day.json: ось FreshDayATR потеряла обязательный узел %v (есть %v)", want, grid["FreshDayATR"])
			}
		}
	}

	// Верхний край оси SpentDayATR в cal_day.json: 2.0.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal_day.json")
		if !containsFloat(grid["SpentDayATR"], 2.0) {
			t.Errorf("sfin/cal_day.json: ось SpentDayATR потеряла верхний край 2.0 (есть %v)", grid["SpentDayATR"])
		}
	}

	// Верхний край оси VolLookbackBars в cal_vol_window.json: 32.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal_vol_window.json")
		if !containsFloat(grid["VolLookbackBars"], 32) {
			t.Errorf("sfin/cal_vol_window.json: ось VolLookbackBars потеряла верхний край 32 (есть %v)", grid["VolLookbackBars"])
		}
	}

	// Обязательный дефолтный узел оси цели в узкой сетке риска второго круга: TPDailyATR содержит
	// 0.6. Первый круг вернул на дефолт ТОЛЬКО UseVolume, поэтому связка «точка + TPDailyATR 0.6»
	// не пробовалась ни разу; без этого узла второй круг повторил бы слепое пятно первого.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal2_risk.json")
		if !containsFloat(grid["TPDailyATR"], 0.6) {
			t.Errorf("sfin/cal2_risk.json: ось TPDailyATR потеряла обязательный дефолтный узел 0.6 (есть %v)", grid["TPDailyATR"])
		}
	}

	// Обязательный дефолтный узел оси свежести дня во втором круге: FreshDayATR содержит 0. Любое
	// ненулевое значение — кандидат на провал риск-гейта C (вход на первом баре дня), а гейт вторым
	// кругом не смягчается.
	{
		grid := rsiPullbackTickerGrid(t, "sfin", "cal2_day.json")
		if !containsFloat(grid["FreshDayATR"], 0) {
			t.Errorf("sfin/cal2_day.json: ось FreshDayATR потеряла обязательный дефолтный узел 0 (есть %v)", grid["FreshDayATR"])
		}
	}

	// Темы entry_deep и day_spent не заводятся: файлов cal_entry_deep.json и cal_day_spent.json в
	// каталоге SFIN быть не должно. Проверка явная, через os.Stat, а не через отсутствие в
	// sfinGridFiles — иначе исполнитель будущей правки мог бы завести файл молча, просто не
	// дописав его в поимённый список.
	for _, file := range []string{"cal_entry_deep.json", "cal_day_spent.json"} {
		path := filepath.Join(rsiPullbackParamsDir, "sfin", file)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("sfin/%s существует, но эта тема не должна заводиться (§5.1)", file)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}
