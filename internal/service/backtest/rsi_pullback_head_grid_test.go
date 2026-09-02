package backtest

import "testing"

// headGrid читает файл сеток HEAD через общий хелпер.
func headGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "head", file)
}

// TestHEADGridsStayWide сторожит оси всех десяти тем каталога head/. Каталог собран 2026-09-02 по
// замерам самой бумаги на расчётном окне 2024-09-02 … 2026-09-02 (в окне 25 788 получасовых
// баров, из них 17 507 будних; дневная серия в окне 610 свечей, из них 491 будняя).
//
// ЗАЯВКА БЫЛА НА HHRU. Тикер мёртв: ряд обрывается 2024-08-09, бумага редомицилирована в МКПАО
// «Хэдхантер» и торгуется как HEAD с 2024-09-26. Калибруется HEAD — решение владельца 2026-09-02.
//
// РЕШЕНИЕ ВЛАДЕЛЬЦА 2026-09-02: сетки этого тикера держатся МАКСИМАЛЬНО ШИРОКИМИ, как у BANEP,
// ASTR и SNGSP. Поэтому тест сторожит оси с ДРУГОЙ стороны, чем у большинства тикеров каталога: он
// проверяет, что край оси не урезали, а не что его не расширили. Замеры, которые в узком каталоге
// были бы основанием обрезать край, живут в _comment каждой сетки как предупреждения.
//
// Инвариантов, которые остаются жёсткими, три, и все три — про смысл, а не про диапазон:
//
//   - RSILower не может быть выше 50: 50 — средняя линия осциллятора, выше неё отката нет по
//     определению.
//   - RSIPeriod не может быть короче 3: двойка — это не откат, а тик.
//   - Ось тренда не может порождать пар EMAFast >= EMASlow: при равенстве фильтр вырождается,
//     при инверсии становится другим фильтром.
//
// СХЕМА ПРОГОНОВ АДАПТИРОВАННАЯ 24/12/3: у нынешней бумаги 23.2 месяца истории, штатная 36/12/6
// недоступна. Четыре фолда встык, число проверено контрольным прогоном до раскладки сеток.
func TestHEADGridsStayWide(t *testing.T) {
	screen := headGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := headGrid(t, "cal_entry.json")
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	for _, v := range entry["RSIPeriod"] {
		if v < 3 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче тройки — уже не откат, а тик", v)
		}
	}
	// Рельеф уровня на HEAD немонотонный, максимум живой зоны в середине оси (RSI(4): 10 -> 0.447
	// на 11 сделках, 30 -> 1.311, 35 -> 1.578, 50 -> 1.044). Оба края обязаны остаться в сетке:
	// середина оси доказуема только при видимых краях.
	for _, v := range []float64{3, 8} {
		if !containsValue(entry["RSIPeriod"], v) {
			t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит %v — ось входа держится широкой по решению владельца", entry["RSIPeriod"], v)
		}
	}
	for _, v := range []float64{10, 50} {
		if !containsValue(entry["RSILower"], v) {
			t.Errorf("cal_entry.json: RSILower = %v, не содержит %v — оба края оси обязаны остаться", entry["RSILower"], v)
		}
	}

	// Ось тренда СДВИНУТА ВНИЗ: EMASlow начинается с 20, EMAFast кончается на 10. Максимум всей
	// таблицы EMA стоит на паре 3/20 (PF 2.669 на 57 сделках), рядом 3/30 (1.626/76) и 5/30
	// (1.485/84), тогда как вся половина EMAFast 20..40 лежит в 1.10-1.30, то есть на уровне
	// baseline 1.311 и ниже. Расширить вниз и сохранить быстрый край одновременно нельзя —
	// инвариант EMAFast < EMASlow этого не разрешает. Прецедент ровно такой: ASTR, BANEP и BSPB
	// держат EMAFast [5,10,20] при EMASlow от 30.
	trend := headGrid(t, "cal_trend.json")
	for _, f := range trend["EMAFast"] {
		for _, s := range trend["EMASlow"] {
			if f >= s {
				t.Errorf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: фильтр вырождается", f, s)
			}
		}
	}
	if !containsValue(trend["EMAFast"], 3) || !containsValue(trend["EMASlow"], 20) {
		t.Errorf("cal_trend.json: пара 3/20 обязана остаться в сетке (максимум таблицы, PF 2.669 на 57 сделках), got fast=%v slow=%v", trend["EMAFast"], trend["EMASlow"])
	}
	if !containsValue(trend["EMASlow"], 250) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 250 — верхний край оси не урезается", trend["EMASlow"])
	}

	// Ось выхода РАСШИРЕНА ВНИЗ ДО 50 (канон §8.1 начинается с 55): максимум оси на 80 (1.470), но
	// второе значение стоит на 55 (1.430), у самого края канонической оси, а ось с сильным краем
	// каталог читать не умеет (ошибка, разобранная на WUSH). Уровень 50 живой и по рельефу
	// (1.380/103), и по событиям (2163 кросса вверх у RSI(4)).
	exit := headGrid(t, "cal_exit.json")
	for _, v := range []float64{50, 85} {
		if !containsValue(exit["RSIUpper"], v) {
			t.Errorf("cal_exit.json: RSIUpper = %v, не содержит %v — оба края полосы выхода обязаны остаться", exit["RSIUpper"], v)
		}
	}

	// Ось риска РАСШИРЕНА ВВЕРХ ДО 2.0 (канон кончается на 1.5). Рельеф насыщается уже на 1.3:
	// колонки 1.3, 1.5 и 2.0 совпадают побайтово (PF 2.392 на 88 сделках). Верх оставлен как
	// КОНТРОЛЬ насыщения: 2.0 дневных ATR это ~5.7% цены, такой стоп достаётся размахом дня в
	// 1.9% дней и защитой быть перестаёт. Победитель проверяется долей SL-выходов, удержанием и
	// экспозицией под гэпом (Task 9).
	risk := headGrid(t, "cal_risk.json")
	if !containsValue(risk["StopDailyATR"], 2.0) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 2.0 — контроль насыщения оси", risk["StopDailyATR"])
	}
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — узкий край оси не урезается", risk["StopDailyATR"])
	}
	// Контрольная строка цели: TestRSIPullbackGridControlPoints требует цель ВЫШЕ самого широкого
	// стопа файла. Строка 2.5 заведомо мертва (колонки 1.5, 2.0 и 2.5 совпадают побайтово) и стоит
	// в сетке как контроль асимметрии, а не как кандидат.
	if !containsValue(risk["TPDailyATR"], 2.5) {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит контрольную 2.5", risk["TPDailyATR"])
	}

	day := headGrid(t, "cal_day.json")
	for _, v := range []float64{0, 0.6} {
		if !containsValue(day["FreshDayATR"], v) {
			t.Errorf("cal_day.json: FreshDayATR = %v, не содержит %v", day["FreshDayATR"], v)
		}
	}
	for _, v := range []float64{0.5, 1.5} {
		if !containsValue(day["SpentDayATR"], v) {
			t.Errorf("cal_day.json: SpentDayATR = %v, не содержит %v", day["SpentDayATR"], v)
		}
	}

	spent := headGrid(t, "cal_day_spent.json")
	if !sameSet(spent["FreshDayATR"], 0) {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно {0} — тема мерит исчерпанную ветку при закрытой свежей", spent["FreshDayATR"])
	}
	for _, v := range []float64{0.4, 1.5} {
		if !containsValue(spent["SpentDayATR"], v) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v", spent["SpentDayATR"], v)
		}
	}

	volume := headGrid(t, "cal_volume.json")
	if !sameSet(volume["UseVolume"], 1) {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно {1} — тема мерит форму гейта, а не его наличие", volume["UseVolume"])
	}
	for _, v := range []float64{1.0, 3.0} {
		if !containsValue(volume["VolMult"], v) {
			t.Errorf("cal_volume.json: VolMult = %v, не содержит %v", volume["VolMult"], v)
		}
	}

	window := headGrid(t, "cal_vol_window.json")
	for _, v := range []float64{1, 24} {
		if !containsValue(window["VolLookbackBars"], v) {
			t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит %v — ось окна держится широкой, сходимости на ней замер не показал", window["VolLookbackBars"], v)
		}
	}

	trail := headGrid(t, "cal_trail.json")
	if !sameSet(trail["UseRSIExit"], 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — тема мерит, заменяет ли трейл RSI-выход", trail["UseRSIExit"])
	}
	if !sameSet(trail["UseTrail"], 1) {
		t.Errorf("cal_trail.json: UseTrail = %v, want ровно {1} — иначе дистанция трейла не измеряется вовсе", trail["UseTrail"])
	}
}
