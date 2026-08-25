package backtest

import "testing"

// bspbGrid читает файл сеток BSPB через общий хелпер.
func bspbGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "bspb", file)
}

// TestBSPBEarlyGridsPinTheirMeasuredAxes сторожит оси трёх РАННИХ тем каталога bspb/ — тех, что
// идут поверх дефолтов ядра. Каталог собран 2026-08-24 по замерам самого BSPB: 35 261
// получасовой бар в кэше (2023-08-24 … 2026-08-24), из них 24 953 будних; дневная серия — 1130
// свечей с 2022-08-24. Запас истории НУЛЕВОЙ: 30m у поставщика отдаёт ровно три года.
//
// Три решения отличают этот каталог от образца dias/, и каждое опирается на замер сделками, а не
// на счёт кроссов:
//
//   - RSIPeriod РАСШИРЕН ВНИЗ ДО 3 — отмена правила каталога для этой бумаги, решение владельца
//     от 2026-08-24. Правило гласило: тройка не берётся, потому что кроссов у неё вдвое больше,
//     чем у четвёрки (на BSPB 660 против 311 на уровне 10), и это дыхание цены. Проверка
//     сделками правило опровергает: RSI(3)@15 даёт 94 сделки при PF 1.327, RSI(4)@15 — 56 при
//     1.068. Тройка и чаще торгует, и точнее.
//   - RSIPeriod ОБОРВАН НА 6, хотя счёт кроссов позволял бы 7 и 8: RSI(7)@10 даёт 54 будних
//     кросса — выше планки живого угла каталога (29). Обрывает ось замер сделками: RSI(7)@10 —
//     2 сделки, RSI(8)@10 — НОЛЬ, RSI(7)@15 — 10, RSI(8)@15 — 7. Дневной гейт и тренд-фильтр
//     вырезают всё, что оставляет медленный RSI.
//   - Окно оси EMA СДВИНУТО ВНИЗ целиком. Максимум замера стоял на нижнем крае каталожной оси
//     (10/30 → PF 2.055, 5/50 → 1.950), а весь её верх мёртв (10/150 → 1.020, 20/150 → 1.135,
//     40/200 → 1.058) при ровном допуске 48.2–50.4% на всех парах. Побочное следствие: пары с
//     EMAFast >= EMASlow либо вырождены (30/30 и 40/40 дают 0.0% допуска), либо инвертированы
//     (40/30 — «медленная над быстрой», другой фильтр), поэтому EMAFast обрезан до 20.
func TestBSPBEarlyGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := bspbGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := bspbGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: RSI(4)@10 даёт 311 будних кроссов и 32 сделки при PF 1.727 —
	// лучший PF всей оси входа на дефолтных прочих полях.
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на BSPB это самый прибыльный угол оси (32 сделки, PF 1.727)", entry["RSILower"])
	}
	// Тройка — сознательное отступление от правила каталога, и держится она замером сделками.
	if !containsValue(entry["RSIPeriod"], 3) {
		t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит 3 — на BSPB тройка бьёт четвёрку и по числу сделок (94 против 56), и по PF (1.327 против 1.068); решение владельца 2026-08-24", entry["RSIPeriod"])
	}
	// Ось обрывается на 6: медленнее её выборка мертва (RSI(7)@10 — 2 сделки, RSI(8)@10 — ноль).
	for _, v := range entry["RSIPeriod"] {
		if v > 6 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: на BSPB медленный RSI не оставляет сделок — RSI(7)@10 даёт 2, RSI(8)@10 ноль", v)
		}
		if v < 3 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче тройки — уже не откат, а тик", v)
		}
	}
	// Полоса выхода расширена до 85, чтобы разворот (75 → 1.806, 80 → 1.636, 85 → 1.324) попал
	// ВНУТРЬ сетки, а не на её край.
	if !containsValue(entry["RSIUpper"], 85) {
		t.Errorf("cal_entry.json: RSIUpper = %v, не содержит 85 — без него максимум полосы (75) стоит у края оси", entry["RSIUpper"])
	}

	trend := bspbGrid(t, "cal_trend.json")
	// Ось расширена вниз: максимум оси — пара 10/30 (PF 2.055), то есть ЗА нижним краем
	// каталожной оси EMASlow [50…200].
	if !containsValue(trend["EMASlow"], 30) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 30 — максимум замера (пара 10/30, PF 2.055) стоял бы вне сетки", trend["EMASlow"])
	}
	// И обрезана сверху: весь верх мёртв при ровном допуске, значит обрезка не отнимает выборку.
	for _, v := range trend["EMASlow"] {
		if v > 150 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: верх оси на BSPB мёртв — 10/150 даёт 1.020, 40/200 даёт 1.058", v)
		}
	}
	// Главное следствие расширения вниз: ни одной пары EMAFast >= EMASlow. Такая пара либо
	// вырождена (30/30 → 0.0% допуска), либо инвертирована (40/30 → «медленная над быстрой»).
	maxFast := 0.0
	for _, v := range trend["EMAFast"] {
		if v > maxFast {
			maxFast = v
		}
	}
	minSlow := 1e9
	for _, v := range trend["EMASlow"] {
		if v < minSlow {
			minSlow = v
		}
	}
	if maxFast >= minSlow {
		t.Errorf("cal_trend.json: max(EMAFast) = %v >= min(EMASlow) = %v — сетка порождает пары, где фильтр вырожден (EMAFast == EMASlow даёт 0.0%% допуска) или инвертирован", maxFast, minSlow)
	}
}

// TestBSPBLateGridsPinTheirAxesAndAnchor сторожит семь ПОЗДНИХ тем — тех, что идут поверх якоря,
// выбранного темами entry и trend. Гибридная процедура объявлена в спеке 2026-08-24 и вызвана
// тем, что дефолты ядра на BSPB стоят почти в мёртвой зоне (baseline PF 1.114 на 142 сделках; в
// каталоге контрольный baseline записан у трёх тикеров — SIBN 1.027/129, ELFV 1.546/181,
// DIAS 1.822/153, — то есть ниже BSPB стоит только SIBN): тема свипует только свои оси, а
// остальные поля берёт из дефолтов, поэтому восемь тем из десяти мерили бы шум. Ось объёмного
// гейта это показала прямо — вся она лежит в полосе 1.04–1.13 вокруг baseline.
//
// Тест прибивает два свойства. Первое: каждая поздняя тема ДЕЙСТВИТЕЛЬНО несёт якорь — все пять
// якорных полей присутствуют и однозначны (список из одного значения не свипует, а фиксирует).
// Второе: якорь ОДИН И ТОТ ЖЕ во всех семи файлах — иначе темы меряют свои оси из разных точек и
// их лидерборды несравнимы между собой.
func TestBSPBLateGridsPinTheirAxesAndAnchor(t *testing.T) {
	late := []string{
		"cal_day.json", "cal_day_spent.json", "cal_volume.json",
		"cal_vol_window.json", "cal_risk.json", "cal_exit.json", "cal_trail.json",
	}
	// cal_exit свипует RSIUpper — это его тема, поэтому якорным полем он его не считает.
	anchorFields := map[string][]string{
		"cal_exit.json": {"RSIPeriod", "RSILower", "EMAFast", "EMASlow"},
	}
	defaultFields := []string{"RSIPeriod", "RSILower", "RSIUpper", "EMAFast", "EMASlow"}

	anchor := map[string]float64{}
	for _, file := range late {
		g := bspbGrid(t, file)
		fields := defaultFields
		if custom, ok := anchorFields[file]; ok {
			fields = custom
		}
		for _, f := range fields {
			values := g[f]
			if len(values) != 1 {
				t.Errorf("%s: %s = %v, want ровно одно значение — якорь фиксируется, а не свипуется", file, f, values)
				continue
			}
			if seen, ok := anchor[f]; ok && seen != values[0] {
				t.Errorf("%s: %s = %v, а другая поздняя тема несёт %v — якорь обязан быть один и тот же во всех семи файлах", file, f, values[0], seen)
				continue
			}
			anchor[f] = values[0]
		}
	}

	// Ось объёмного гейта сужена до 2.5: при 2.5 остаётся 56 сделок = 19 на двенадцатимесячное
	// обучающее окно при -min-trades 20, и PF там 0.839 — ниже baseline 1.114.
	volume := bspbGrid(t, "cal_volume.json")
	for _, v := range volume["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: на BSPB это меньше 56 сделок за 36 месяцев (19 на обучающее окно) при PF ниже baseline", v)
		}
	}

	// Окно объёмного гейта: максимум замера на 5, край 12 нужен, чтобы плато было видно внутри.
	window := bspbGrid(t, "cal_vol_window.json")
	if !containsValue(window["VolLookbackBars"], 5) {
		t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит 5 — это замеренный максимум оси (PF 1.170 против 1.059 у дефолта ядра)", window["VolLookbackBars"])
	}

	// Стоп: строка 0.3 возвращена, верхний край 1.3 сохранён.
	risk := bspbGrid(t, "cal_risk.json")
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на BSPB круг издержек съедает там 14.0%% риска, под чертой 17%%", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: такой стоп достаёт меньше 17%% дней и вытесняет убыток в RSI-выход", v)
		}
	}
	// Цель: 0.4 добавлена, всё выше 1.5 убрано как почти не срабатывающее.
	if !containsValue(risk["TPDailyATR"], 0.4) {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит 0.4 — весь edge BSPB живёт в коротких целях", risk["TPDailyATR"])
	}
	for _, v := range risk["TPDailyATR"] {
		if v > 1.5 {
			t.Errorf("cal_risk.json свипует TPDailyATR=%v: цель шире дневного ATR на BSPB почти не срабатывает — под настоящим якорем (STEP 4 темы risk) колонки целей 1.0 и 1.5 расходятся всего на 0.013-0.016 PF при одинаковых 60 сделках, а диапазона между ними достигают 2 сделки из 60 (3.3%%)", v)
		}
	}

	// Полоса выхода расширена до 85, чтобы разворот (максимум на 75) стоял внутри оси.
	exit := bspbGrid(t, "cal_exit.json")
	if !containsValue(exit["RSIUpper"], 85) {
		t.Errorf("cal_exit.json: RSIUpper = %v, не содержит 85 — без него максимум полосы (75) стоит у края", exit["RSIUpper"])
	}
}
