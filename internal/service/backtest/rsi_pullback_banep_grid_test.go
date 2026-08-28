package backtest

import "testing"

// banepGrid читает файл сеток BANEP через общий хелпер.
func banepGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "banep", file)
}

// TestBANEPGridsStayWide сторожит оси всех десяти тем каталога banep/. Каталог собран 2026-08-28 по
// замерам самого BANEP на расчётном окне 2023-08-28 … 2026-08-28 (34 069 получасовых баров, из них
// 24 039 будних; дневная серия 1129 свечей, в окне 764 будних).
//
// РЕШЕНИЕ ВЛАДЕЛЬЦА 2026-08-28: сетки этого тикера держатся МАКСИМАЛЬНО ШИРОКИМИ. Поэтому тест
// сторожит оси с ДРУГОЙ стороны, чем у остальных тикеров каталога: он проверяет, что край оси не
// урезали, а не что его не расширили. Замеры, которыми обосновывались обрезки первого варианта
// каталога, живут в _comment каждой сетки как предупреждения о том, чего ждать на краях.
//
// Инвариантов, которые остаются жёсткими, всего три, и все три — про смысл, а не про диапазон:
//
//   - RSILower не может быть выше 50: 50 — средняя линия осциллятора, выше неё отката нет по
//     определению, там кросс вниз означает уже не откат, а начало движения.
//   - RSIPeriod не может быть короче 3: двойка — это не откат, а тик.
//   - Ось тренда не может порождать пары EMAFast >= EMASlow: при равенстве фильтр вырождается
//     (допуск 0.0%), при инверсии становится другим фильтром («медленная над быстрой»).
func TestBANEPGridsStayWide(t *testing.T) {
	screen := banepGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := banepGrid(t, "cal_entry.json")
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
	// Ось входа держится широкой по обеим границам: тройка (212 сделок при PF 1.448 на полной
	// истории) и восьмёрка (52 сделки при 0.940) обе обязаны остаться в сетке.
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
	for _, v := range []float64{55, 85} {
		if !containsValue(entry["RSIUpper"], v) {
			t.Errorf("cal_entry.json: RSIUpper = %v, не содержит %v — полоса выхода меряется целиком", entry["RSIUpper"], v)
		}
	}

	trend := banepGrid(t, "cal_trend.json")
	// Ось тренда держится целиком: и низ, где стоит точечный максимум (10/40 -> 1.444, 5/30 ->
	// 1.443), и верх, который на полной истории слаб (10/150 -> 1.152, 40/200 -> 1.129).
	for _, v := range []float64{30, 200} {
		if !containsValue(trend["EMASlow"], v) {
			t.Errorf("cal_trend.json: EMASlow = %v, не содержит %v — ось тренда меряется целиком", trend["EMASlow"], v)
		}
	}
	for _, f := range trend["EMAFast"] {
		for _, s := range trend["EMASlow"] {
			if f >= s {
				t.Errorf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: фильтр вырожден или инвертирован", f, s)
			}
		}
	}

	day := banepGrid(t, "cal_day.json")
	// Ноль в свежей ветке обязателен: на всех прод-тикерах каталога победил именно он.
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — на всех прод-тикерах каталога победил ноль", day["FreshDayATR"])
	}
	for _, name := range []string{"cal_day.json", "cal_day_spent.json"} {
		spent := banepGrid(t, name)["SpentDayATR"]
		// Обе стороны оси дня: 0.6 (57.8% баров) и 1.5 (6.4%) — край, о цене которого предупреждает
		// _comment, но который по решению владельца в сетке остаётся.
		for _, v := range []float64{0.6, 1.5} {
			if !containsValue(spent, v) {
				t.Errorf("%s: SpentDayATR = %v, не содержит %v — ось дня меряется целиком", name, spent, v)
			}
		}
	}

	volume := banepGrid(t, "cal_volume.json")
	// Верх оси множителя остаётся: 3.0 (64 сделки, PF 1.155) и 4.0 (51, 1.012) — предупреждение, а
	// не основание вырезать край.
	for _, v := range []float64{1.0, 4.0} {
		if !containsValue(volume["VolMult"], v) {
			t.Errorf("cal_volume.json: VolMult = %v, не содержит %v — ось объёма меряется целиком", volume["VolMult"], v)
		}
	}

	window := banepGrid(t, "cal_vol_window.json")
	// Кривая окна растёт монотонно к верхнему краю (1 -> 1.222, 12 -> 1.329, 16 -> 1.343), поэтому
	// верх доведён до 24; нижний край 1 нужен, чтобы кривая была видна целиком.
	for _, v := range []float64{1, 24} {
		if !containsValue(window["VolLookbackBars"], v) {
			t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит %v — ось окна меряется целиком", window["VolLookbackBars"], v)
		}
	}

	risk := banepGrid(t, "cal_risk.json")
	// Строка 0.3 ОСТАВЛЕНА, хотя круг издержек съедает там 21.8% риска: решение владельца держать
	// сетку широкой сильнее каталожной черты 17%, а предупреждение живёт в _comment.
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — узкий край оси риска остаётся в сетке по решению владельца", risk["StopDailyATR"])
	}
	if !containsValue(risk["StopDailyATR"], 1.5) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 1.5 — широкий край оси риска остаётся в сетке", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v <= 0 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: сделка без стопа запрещена ядром", v)
		}
	}
	// Цель обязана накрывать самый широкий стоп, иначе контрольная точка каталога не выполняется.
	if !containsValue(risk["TPDailyATR"], 2.5) {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит 2.5 — цель меряется целиком и обязана быть шире самого широкого стопа", risk["TPDailyATR"])
	}

	exit := banepGrid(t, "cal_exit.json")
	for _, v := range []float64{55, 85} {
		if !containsValue(exit["RSIUpper"], v) {
			t.Errorf("cal_exit.json: RSIUpper = %v, не содержит %v — полоса выхода меряется целиком", exit["RSIUpper"], v)
		}
	}

	trail := banepGrid(t, "cal_trail.json")
	if !containsValue(trail["TrailDailyATR"], 1.5) {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит 1.5 — ось трейла меряется целиком", trail["TrailDailyATR"])
	}
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want {0,1} — тема меряет, способен ли трейл заменить RSI-выход", got)
	}
}
