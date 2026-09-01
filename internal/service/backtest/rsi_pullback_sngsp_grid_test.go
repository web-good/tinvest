package backtest

import "testing"

// sngspGrid читает файл сеток SNGSP через общий хелпер.
func sngspGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "sngsp", file)
}

// TestSNGSPGridsStayWide сторожит оси всех десяти тем каталога sngsp/. Каталог собран 2026-09-01
// по замерам самого SNGSP на расчётном окне 2023-09-01 … 2026-09-01 (36 685 получасовых баров в
// кэше, в окне 35 972, из них 25 052 будних; дневная серия 1156 свечей, в окне 881, из них 761
// будняя).
//
// РЕШЕНИЕ ВЛАДЕЛЬЦА 2026-09-01: сетки этого тикера держатся МАКСИМАЛЬНО ШИРОКИМИ, как у BANEP и
// ASTR. Поэтому тест сторожит оси с ДРУГОЙ стороны, чем у большинства тикеров каталога: он
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
// СХЕМА ПРОГОНОВ ШТАТНАЯ 36/12/6 — обе серии кэша длиннее запрошенного окна, адаптация не нужна.
func TestSNGSPGridsStayWide(t *testing.T) {
	screen := sngspGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := sngspGrid(t, "cal_entry.json")
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
	// Рельеф уровня на SNGSP НЕ монотонный: максимум в середине оси (RSI(4): 10 -> 1.323,
	// 25 -> 1.674, 30 -> 1.535, 45 -> 0.997). Оба края обязаны остаться в сетке именно поэтому:
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
	// Полоса выхода расширена ВНИЗ до 50: максимум оси стоит на 60, у нижнего края канона §8.1.
	for _, v := range []float64{50, 85} {
		if !containsValue(entry["RSIUpper"], v) {
			t.Errorf("cal_entry.json: RSIUpper = %v, не содержит %v — полоса выхода меряется целиком, вниз до 50", entry["RSIUpper"], v)
		}
	}

	trend := sngspGrid(t, "cal_trend.json")
	for _, fast := range trend["EMAFast"] {
		for _, slow := range trend["EMASlow"] {
			if fast >= slow {
				t.Errorf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: фильтр тренда вырождается", fast, slow)
			}
		}
	}
	// Ось медленной EMA расширена ВВЕРХ до 250: 200 бьёт 150 (1.400 против 1.355), максимум
	// верхнего участка стоял на краю канонической оси. Быстрая держится до 40 — на паре 40/50
	// стоит максимум таблицы (1.647), и его нельзя прятать обрезкой.
	for _, v := range []float64{50, 250} {
		if !containsValue(trend["EMASlow"], v) {
			t.Errorf("cal_trend.json: EMASlow = %v, не содержит %v — ось тренда меряется целиком", trend["EMASlow"], v)
		}
	}
	for _, v := range []float64{5, 40} {
		if !containsValue(trend["EMAFast"], v) {
			t.Errorf("cal_trend.json: EMAFast = %v, не содержит %v — оба края быстрой EMA обязаны остаться", trend["EMAFast"], v)
		}
	}

	day := sngspGrid(t, "cal_day.json")
	// Ветка свежего дня на SNGSP монотонно вредит (0 -> 1.535, 0.6 -> 1.033), но ноль и верхний
	// край обязаны остаться: без нуля тема не может выбрать «свежая ветка закрыта», без 0.6 —
	// показать, что вред монотонен.
	for _, v := range []float64{0, 0.6} {
		if !containsValue(day["FreshDayATR"], v) {
			t.Errorf("cal_day.json: FreshDayATR = %v, не содержит %v", day["FreshDayATR"], v)
		}
	}
	for _, v := range []float64{0.5, 1.5} {
		if !containsValue(day["SpentDayATR"], v) {
			t.Errorf("cal_day.json: SpentDayATR = %v, не содержит %v — оба края исчерпанной ветки обязаны остаться", day["SpentDayATR"], v)
		}
	}

	spent := sngspGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; !sameSet(got, 0) {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно {0} — тема меряет ТОЛЬКО исчерпанную ветку", got)
	}
	for _, v := range []float64{0.4, 1.5} {
		if !containsValue(spent["SpentDayATR"], v) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v", spent["SpentDayATR"], v)
		}
	}

	volume := sngspGrid(t, "cal_volume.json")
	for _, v := range []float64{1, 3} {
		if !containsValue(volume["VolMult"], v) {
			t.Errorf("cal_volume.json: VolMult = %v, не содержит %v — ось множителя держится широкой, включая выброс 3.0 на 48 сделках", volume["VolMult"], v)
		}
	}
	for _, v := range []float64{3, 20} {
		if !containsValue(volume["VolBaseDays"], v) {
			t.Errorf("cal_volume.json: VolBaseDays = %v, не содержит %v", volume["VolBaseDays"], v)
		}
	}

	window := sngspGrid(t, "cal_vol_window.json")
	for _, v := range []float64{1, 24} {
		if !containsValue(window["VolLookbackBars"], v) {
			t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит %v — оба края окна обязаны остаться", window["VolLookbackBars"], v)
		}
	}

	risk := sngspGrid(t, "cal_risk.json")
	// Ось стопа расширена ВВЕРХ до 2.0: рельеф не насыщается к 1.5 (1.5 -> 1.865, 2.0 -> 2.122
	// при неизменных 175 сделках). Цена расширения записана в _comment как предупреждение о
	// капкане, а не как обрезка.
	for _, v := range []float64{0.3, 2} {
		if !containsValue(risk["StopDailyATR"], v) {
			t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит %v — оба края оси риска обязаны остаться", risk["StopDailyATR"], v)
		}
	}
	// Цель 2.5 — контрольная строка: TestRSIPullbackGridControlPoints требует от файла, свипующего
	// стоп до 2.0, хотя бы одну цель ВЫШЕ 2.0. Строка заведомо мертва (1.5, 2.0 и 2.5 совпадают
	// побайтово), и это записано в _comment.
	for _, v := range []float64{0.3, 2.5} {
		if !containsValue(risk["TPDailyATR"], v) {
			t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит %v", risk["TPDailyATR"], v)
		}
	}

	exit := sngspGrid(t, "cal_exit.json")
	for _, v := range []float64{50, 85} {
		if !containsValue(exit["RSIUpper"], v) {
			t.Errorf("cal_exit.json: RSIUpper = %v, не содержит %v — ось выхода расширена вниз до 50", exit["RSIUpper"], v)
		}
	}

	trail := sngspGrid(t, "cal_trail.json")
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — тема меряет трейл ПРОТИВ RSI-выхода", got)
	}
	for _, v := range []float64{0.3, 1.5} {
		if !containsValue(trail["TrailDailyATR"], v) {
			t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит %v", trail["TrailDailyATR"], v)
		}
	}
}
