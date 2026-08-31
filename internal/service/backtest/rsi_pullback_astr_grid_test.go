package backtest

import "testing"

// astrGrid читает файл сеток ASTR через общий хелпер.
func astrGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "astr", file)
}

// TestASTRGridsStayWide сторожит оси всех десяти тем каталога astr/. Каталог собран 2026-08-30 по
// замерам самого ASTR на расчётном окне 2023-10-29 … 2026-08-29 (34 551 получасовой бар в кэше, в
// окне 34 353, из них 24 287 будних; дневная серия 848 свечей, в окне 719 будних).
//
// РЕШЕНИЕ ВЛАДЕЛЬЦА 2026-08-30: сетки этого тикера держатся МАКСИМАЛЬНО ШИРОКИМИ, как у BANEP.
// Поэтому тест сторожит оси с ДРУГОЙ стороны, чем у большинства тикеров каталога: он проверяет,
// что край оси не урезали, а не что его не расширили. Замеры, которые в узком каталоге были бы
// основанием обрезать край, живут в _comment каждой сетки как предупреждения о том, чего ждать.
//
// Инвариантов, которые остаются жёсткими, всего три, и все три — про смысл, а не про диапазон:
//
//   - RSILower не может быть выше 50: 50 — средняя линия осциллятора, выше неё отката нет по
//     определению, там кросс вниз означает уже не откат, а начало движения.
//   - RSIPeriod не может быть короче 3: двойка — это не откат, а тик.
//   - Ось тренда не может порождать пары EMAFast >= EMASlow: при равенстве фильтр вырождается,
//     при инверсии становится другим фильтром («медленная над быстрой»).
//
// ОКНО КОРОЧЕ КАТАЛОЖНОГО: первый бар получасовой серии 2023-10-13 (IPO Группы Астра), доступно
// 34.5 месяца, канон 36/12/6 не помещается. Схема адаптирована до 34/10/6 — четыре фолда встык,
// жертвуется длина обучающего окна, а не число фолдов, потому что правило сборки точки «≥3 фолда
// из 4» при трёх фолдах вырождается. Прецеденты: IVAT 25/9/4, DIAS 30/10/5, YDEX 25/9/4.
func TestASTRGridsStayWide(t *testing.T) {
	screen := astrGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := astrGrid(t, "cal_entry.json")
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
	// Ось входа держится широкой по обеим границам. На ASTR рельеф уровня монотонный и сильный
	// (RSI(4): 10 -> 2.252, 20 -> 1.714, 30 -> 1.093, 45 -> 0.987), поэтому глубокий край 10
	// обязан остаться в сетке, а мелкий край 50 — чтобы монотонность была видна целиком.
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

	trend := astrGrid(t, "cal_trend.json")
	// Ось тренда на ASTR шумит без монотонности (5/30 -> 1.182, 10/30 -> 0.952, 20/50 -> 1.135,
	// 20/100 -> 0.938, 40/200 -> 1.069), края не выделяются — тем более она меряется целиком.
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

	day := astrGrid(t, "cal_day.json")
	// Ноль в свежей ветке обязателен: на всех прод-тикерах каталога победил именно он, а на ASTR
	// свежая ветка разбавляет (0.2 -> 185 сделок/1.053 против 158/1.093 при нуле).
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — на всех прод-тикерах каталога победил ноль", day["FreshDayATR"])
	}
	for _, name := range []string{"cal_day.json", "cal_day_spent.json"} {
		spent := astrGrid(t, name)["SpentDayATR"]
		// Обе стороны оси дня: 0.6 (57.6% баров) и 1.5 (7.2% баров, 23 сделки за 34 месяца =
		// 7 на обучающее окно при пороге 20) — край, о цене которого предупреждает _comment,
		// но который по решению владельца в сетке остаётся и утонет под порогом сам.
		for _, v := range []float64{0.6, 1.5} {
			if !containsValue(spent, v) {
				t.Errorf("%s: SpentDayATR = %v, не содержит %v — ось дня меряется целиком", name, spent, v)
			}
		}
	}

	volume := astrGrid(t, "cal_volume.json")
	// Верх оси множителя остаётся, хотя гейт платный по всей оси (1.0 -> 0.919, 2.0 -> 1.058,
	// 4.0 -> 0.830 против 1.093 у выключенного): это предупреждение, а не основание резать край.
	for _, v := range []float64{1.0, 4.0} {
		if !containsValue(volume["VolMult"], v) {
			t.Errorf("cal_volume.json: VolMult = %v, не содержит %v — ось объёма меряется целиком", volume["VolMult"], v)
		}
	}
	// Ось базы на ASTR ЗНАЧИМА, а не шумит: при VolMult 1.5 база 3 дня -> 124 сделки/1.343,
	// 5 -> 118/1.263, 10 -> 117/1.132, 14 -> 110/0.982. Разброс 0.36 PF, оба края обязательны.
	for _, v := range []float64{3, 20} {
		if !containsValue(volume["VolBaseDays"], v) {
			t.Errorf("cal_volume.json: VolBaseDays = %v, не содержит %v — ось базы на ASTR значима, разброс 0.36 PF", volume["VolBaseDays"], v)
		}
	}

	window := astrGrid(t, "cal_vol_window.json")
	// Максимум кривой окна стоит у УЗКОГО края (1 -> 1.066, 2 -> 1.073, 3 -> 1.058, 12 -> 1.049,
	// 16 -> 1.021), как у YDEX. Верх доведён до 24 всё равно: ось меряется целиком.
	for _, v := range []float64{1, 24} {
		if !containsValue(window["VolLookbackBars"], v) {
			t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит %v — ось окна меряется целиком", window["VolLookbackBars"], v)
		}
	}

	risk := astrGrid(t, "cal_risk.json")
	// Строка 0.3 остаётся, и на ASTR у неё есть отдельное основание помимо решения о ширине:
	// реальный круг издержек 0.124% съедает там 13.1% риска — НИЖЕ каталожной черты 17%, по
	// которой её вырезали из domrf/ и elfv/. Дешёвая бумага (шаг 0.05 ₽ = 0.0122% цены).
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — узкий край жив: круг 0.124%% съедает там 13.1%% риска, ниже черты 17%%", risk["StopDailyATR"])
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

	exit := astrGrid(t, "cal_exit.json")
	for _, v := range []float64{55, 85} {
		if !containsValue(exit["RSIUpper"], v) {
			t.Errorf("cal_exit.json: RSIUpper = %v, не содержит %v — полоса выхода меряется целиком", exit["RSIUpper"], v)
		}
	}

	trail := astrGrid(t, "cal_trail.json")
	if !containsValue(trail["TrailDailyATR"], 1.5) {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит 1.5 — ось трейла меряется целиком", trail["TrailDailyATR"])
	}
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want {0,1} — тема меряет, способен ли трейл заменить RSI-выход", got)
	}
}
