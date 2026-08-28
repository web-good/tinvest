package backtest

import "testing"

// banepGrid читает файл сеток BANEP через общий хелпер.
func banepGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "banep", file)
}

// TestBANEPGridsPinTheirMeasuredAxes сторожит оси всех десяти тем каталога banep/. Каталог собран
// 2026-08-28 по замерам самого BANEP на расчётном окне 2023-08-28 … 2026-08-28 (34 069 получасовых
// баров, из них 24 039 будних; дневная серия 1129 свечей, в окне 764 будних).
//
// Четыре решения отличают этот каталог от каталожной формы, и каждое опирается на точечный замер
// сделками поверх дефолтов ядра (baseline: 134 сделки, PF 1.336):
//
//   - RSIPeriod РАСШИРЕН ВНИЗ ДО 3 — третий случай после BSPB и YDEX. Кроссов у тройки вдвое
//     больше, чем у четвёрки (533 против 264 на уровне 10), то есть правило каталога формально
//     применимо; но RSI(3)@30 даёт 212 сделок при PF 1.448 против 134 при 1.336 у RSI(4)@30.
//   - RSIPeriod ОБОРВАН НА 6: RSI(6)@30 — PF 0.706, RSI(6)@10 — пять сделок за 36 месяцев,
//     RSI(7)@30 — 0.776. Медленный RSI на BANEP не оставляет ни выборки, ни знака.
//   - ОСЬ EMASlow СДВИНУТА ВНИЗ до {30…120}: максимум замера стоит на 10/40 (PF 1.444) и 5/30
//     (1.443), то есть НИЖЕ каталожного минимума 50, а верх оси мёртв (10/150 -> 1.152,
//     20/150 -> 1.118, 40/200 -> 1.129). Прецедент — BSPB.
//   - СТРОКА StopDailyATR 0.3 ВЫРЕЗАНА: стоп 0.3 ATR это 0.86% цены, а реальный круг издержек
//     0.187% (моделируемые 0.1% плюс два тика по 0.0436% при шаге 0.5 ₽ и медианной цене 1148 ₽)
//     съедает 21.8% риска — выше черты 17%, по которой строку вырезали из domrf/ и elfv/. Прогон
//     подтверждает: 0.3 даёт PF 1.052 против 1.336 на дефолтном 0.5.
//   - VolLookbackBars ДОВЕДЁН ДО 16: кривая растёт монотонно к верхнему краю (1 -> 1.222,
//     3 -> 1.288, 12 -> 1.329, 16 -> 1.343), и обрезав край, каталог обрезал бы максимум.
func TestBANEPGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := banepGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := banepGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Тройка — сознательное отступление от правила каталога, держится замером сделками.
	if !containsValue(entry["RSIPeriod"], 3) {
		t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит 3 — на BANEP тройка бьёт четвёрку и по сделкам (212 против 134), и по PF (1.448 против 1.336)", entry["RSIPeriod"])
	}
	for _, v := range entry["RSIPeriod"] {
		if v > 6 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: на BANEP медленный RSI мёртв — RSI(6)@30 даёт PF 0.706, RSI(7)@30 — 0.776", v)
		}
		if v < 3 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче тройки — уже не откат, а тик", v)
		}
	}
	// Оба края уровня живые: 10 даёт у четвёрки максимум строки (PF 1.643), 40 — второй максимум.
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на BANEP это максимум строки RSI(4) (PF 1.643)", entry["RSILower"])
	}
	if !containsValue(entry["RSILower"], 40) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 40 — там второй максимум строки RSI(4) (203 сделки, PF 1.423)", entry["RSILower"])
	}
	// Полоса выхода не расширяется выше 80: там PF 0.864 против 1.336 на дефолте 70.
	for _, v := range entry["RSIUpper"] {
		if v > 80 {
			t.Errorf("cal_entry.json свипует RSIUpper=%v: на BANEP полоса выше 75 мертва (80 даёт 0.864, 85 — 0.927)", v)
		}
	}

	trend := banepGrid(t, "cal_trend.json")
	// Ось СДВИНУТА ВНИЗ: максимум замера стоит ниже каталожного минимума 50.
	if !containsValue(trend["EMASlow"], 30) || !containsValue(trend["EMASlow"], 40) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 30 и 40 — там максимум замера (10/40 -> 1.444, 5/30 -> 1.443)", trend["EMASlow"])
	}
	for _, v := range trend["EMASlow"] {
		if v > 120 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: верх оси на BANEP мёртв (10/150 -> 1.152, 20/150 -> 1.118, 40/200 -> 1.129)", v)
		}
	}
	// Пара с EMAFast >= EMASlow либо вырождена (допуск 0.0%), либо инвертирована («медленная над
	// быстрой» — другой фильтр). При этой оси таких пар нет по построению; тест страхует правки.
	for _, f := range trend["EMAFast"] {
		for _, s := range trend["EMASlow"] {
			if f >= s {
				t.Errorf("cal_trend.json порождает пару EMAFast=%v >= EMASlow=%v: фильтр вырожден или инвертирован", f, s)
			}
		}
	}

	day := banepGrid(t, "cal_day.json")
	// Ноль в ветке «свежий день» обязателен: на всех прод-тикерах каталога победил именно он, а на
	// BANEP свежая ветка замерена разбавляющей (0.3 -> PF 1.290 против 1.336 без неё).
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — на всех прод-тикерах каталога победил ноль", day["FreshDayATR"])
	}
	for _, name := range []string{"cal_day.json", "cal_day_spent.json"} {
		spent := banepGrid(t, name)["SpentDayATR"]
		for _, v := range spent {
			if v > 1.3 {
				t.Errorf("%s свипует SpentDayATR=%v: при 1.5 остаётся 6.4%% баров и 16 сделок за 36 месяцев = 5 на обучающее окно при пороге 20", name, v)
			}
		}
		if !containsValue(spent, 1.0) {
			t.Errorf("%s: SpentDayATR = %v, не содержит 1.0 — там максимум замера (78 сделок, PF 1.554)", name, spent)
		}
	}
	// Ось «дня исчерпанного» уплотнена живыми уровнями 0.9 (29.0% баров) и 1.1 (18.5%): между ними
	// лежит максимум кривой.
	spentOnly := banepGrid(t, "cal_day_spent.json")["SpentDayATR"]
	for _, v := range []float64{0.9, 1.1} {
		if !containsValue(spentOnly, v) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v — уровень живой и максимум кривой лежит рядом", spentOnly, v)
		}
	}

	volume := banepGrid(t, "cal_volume.json")
	for _, v := range volume["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: при 3.0 остаётся 64 сделки (21 на обучающее окно) при PF 1.155 — ниже выключенного гейта", v)
		}
	}
	// 1.5 обязан остаться: там максимум оси (97 сделок, PF 1.400 против 1.336 у выключенного гейта).
	if !containsValue(volume["VolMult"], 1.5) {
		t.Errorf("cal_volume.json: VolMult = %v, не содержит 1.5 — там максимум замера (97 сделок, PF 1.400)", volume["VolMult"])
	}

	window := banepGrid(t, "cal_vol_window.json")
	// Верхний край 16 обязателен: кривая растёт монотонно к нему, максимум стоял бы вне сетки.
	if !containsValue(window["VolLookbackBars"], 16) {
		t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит 16 — кривая растёт к верхнему краю (12 -> 1.329, 16 -> 1.343)", window["VolLookbackBars"])
	}
	if !containsValue(window["VolLookbackBars"], 1) {
		t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит 1 — нижний край нужен, чтобы кривая была видна целиком", window["VolLookbackBars"])
	}

	risk := banepGrid(t, "cal_risk.json")
	// Строка 0.3 ВЫРЕЗАНА: реальный круг издержек съедает там 21.8% риска (черта 17%).
	for _, v := range risk["StopDailyATR"] {
		if v < 0.5 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: круг издержек 0.187%% съедает там 21.8%% риска — выше черты 17%%, по которой строку вырезали из domrf/ и elfv/", v)
		}
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: при 1.5 стоп достаёт лишь 11.6%% дней и вытесняет убыток в RSI-выход", v)
		}
	}
	if !containsValue(risk["TPDailyATR"], 0.6) {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит 0.6 — там максимум каждой строки", risk["TPDailyATR"])
	}
	for _, v := range risk["TPDailyATR"] {
		if v > 1.5 {
			t.Errorf("cal_risk.json свипует TPDailyATR=%v: колонки 1.5, 2.0 и 2.5 совпадают побайтово — цель шире 1.5 ATR на BANEP недостижима", v)
		}
	}

	exit := banepGrid(t, "cal_exit.json")
	for _, v := range exit["RSIUpper"] {
		if v > 80 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: разворот полосы стоит внутри оси (70 -> 1.336, 75 -> 1.234, 80 -> 0.864)", v)
		}
	}
}
