package backtest

import "testing"

// ydexGrid читает файл сеток YDEX через общий хелпер.
func ydexGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "ydex", file)
}

// TestYDEXGridsPinTheirMeasuredAxes сторожит оси всех десяти тем каталога ydex/. Каталог собран
// 2026-08-25 по замерам самого YDEX на расчётном окне 2024-07-25 … 2026-08-25 (27 991 получасовой
// бар, из них 18 864 будних; дневная серия 872 свечи, в окне 530 будних).
//
// Окно короче каталожного намеренно: торги YNDX остановлены 2024-06-14, торги YDEX начались
// 2024-07-24, и в получасовой серии это дыра 40 дней с прыжком цены 4007 -> 4542. Вся YNDX-эпоха
// и сам разрыв выброшены, схема прогонов адаптирована до 25/9/4 (прецедент IVAT).
//
// Шесть решений отличают этот каталог от образца bspb/, и каждое опирается на точечный замер
// сделками поверх дефолтов ядра (baseline: 108 сделок, PF 1.778 — второй по силе из записанных,
// после DIAS 153/1.822):
//
//   - RSIPeriod РАСШИРЕН ВНИЗ ДО 3 — второй случай после BSPB. Кроссов у тройки вдвое больше, чем
//     у четвёрки (484 против 230 на уровне 10), то есть правило каталога формально применимо; но
//     RSI(3)@25 даёт 140 сделок при PF 2.050 против 87 при 1.762 у RSI(4)@25, и вся строка тройки
//     живая (1.53–2.05 на семи уровнях из семи).
//   - RSIPeriod ОБОРВАН НА 6: RSI(7)@10 — 1 сделка, RSI(8)@10 — НОЛЬ, RSI(8)@20 — 9 сделок при
//     PF 0.701. Ось периода на YDEX меряется сделками, а не кроссами.
//   - RSIUpper НЕ РАСШИРЕН до 85, в отличие от BSPB: замер даёт максимум на дефолте 70 (1.778),
//     75 рядом (1.769), 80 уже 1.581, а 85 мёртв (1.204). Разворот стоит ВНУТРИ каталожной оси.
//   - Ось EMA НЕ СДВИНУТА вниз, в отличие от BSPB: максимум оси (10/50 -> 1.802) стоит внутри
//     сетки, а расширение вниз замерено и отвергнуто (10/30 -> 1.393, 10/40 -> 1.434,
//     5/30 -> 1.295). Верх оси жив (20/200 -> 1.542, 40/200 -> 1.466).
//   - SpentDayATR ОБРЕЗАН СВЕРХУ ДО 1.3: при 1.5 остаётся 5.2% баров и 14 сделок за 25 месяцев =
//     5 на девятимесячное обучающее окно, вчетверо ниже -min-trades 20 (прецедент SIBN).
//   - TPDailyATR РАСШИРЕН ВНИЗ ДО 0.4 и ОБРЕЗАН СВЕРХУ ДО 1.5: максимум каждой строки риска стоит
//     на 0.5, а колонки 1.5, 2.0 и 2.5 совпадают ПОБАЙТОВО — цель шире 1.5 дневного ATR на YDEX
//     недостижима, RSI-выход или стоп срабатывают раньше.
func TestYDEXGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := ydexGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := ydexGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: RSI(3)@10 даёт 53 сделки при PF 1.811 — второй результат строки.
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на YDEX это 53 сделки при PF 1.811", entry["RSILower"])
	}
	// Тройка — сознательное отступление от правила каталога, держится замером сделками.
	if !containsValue(entry["RSIPeriod"], 3) {
		t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит 3 — на YDEX тройка бьёт четвёрку и по сделкам (140 против 87), и по PF (2.050 против 1.762)", entry["RSIPeriod"])
	}
	for _, v := range entry["RSIPeriod"] {
		if v > 6 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: на YDEX медленный RSI не оставляет сделок — RSI(7)@10 даёт 1, RSI(8)@10 ноль", v)
		}
		if v < 3 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче тройки — уже не откат, а тик", v)
		}
	}
	// Полоса выхода НЕ расширяется до 85: там PF 1.204 против 1.778 на дефолте 70.
	for _, v := range entry["RSIUpper"] {
		if v > 80 {
			t.Errorf("cal_entry.json свипует RSIUpper=%v: на YDEX полоса выше 80 мертва (85 даёт PF 1.204 против 1.778 на 70)", v)
		}
	}

	trend := ydexGrid(t, "cal_trend.json")
	// Ось НЕ расширяется вниз: пары со EMASlow < 50 замерены хуже любой пары внутри оси.
	for _, v := range trend["EMASlow"] {
		if v < 50 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: на YDEX низ оси хуже (10/30 -> 1.393, 10/40 -> 1.434) любой пары со EMASlow >= 50", v)
		}
	}
	// 50 обязан остаться: максимум оси — пара 10/50 (PF 1.802).
	if !containsValue(trend["EMASlow"], 50) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 50 — максимум замера (пара 10/50, PF 1.802) стоял бы вне сетки", trend["EMASlow"])
	}
	// Верх оси жив (20/200 -> 1.542, 30/200 -> 1.518), обрезать его нечем.
	if !containsValue(trend["EMASlow"], 200) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 200 — верх оси на YDEX жив (20/200 даёт PF 1.542)", trend["EMASlow"])
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

	day := ydexGrid(t, "cal_day.json")
	// Ноль в ветке «свежий день» обязателен: на всех прод-тикерах каталога победил именно он, а на
	// YDEX свежая ветка замерена разбавляющей (0.3 -> PF 1.203 против 1.778 без неё).
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — на всех прод-тикерах каталога победил ноль", day["FreshDayATR"])
	}
	for _, name := range []string{"cal_day.json", "cal_day_spent.json"} {
		spent := ydexGrid(t, name)["SpentDayATR"]
		for _, v := range spent {
			if v > 1.3 {
				t.Errorf("%s свипует SpentDayATR=%v: при 1.5 остаётся 5.2%% баров и 14 сделок за 25 месяцев = 5 на обучающее окно при пороге 20", name, v)
			}
		}
		if !containsValue(spent, 1.3) {
			t.Errorf("%s: SpentDayATR = %v, не содержит 1.3 — верхний живой край оси (9.4%% баров, 25 сделок, PF 4.420)", name, spent)
		}
	}
	// Ось «дня исчерпанного» уплотнена живыми уровнями 0.7 (43.5% баров) и 1.1 (15.3%): между ними
	// PF растёт с 1.588 до 2.909.
	spentOnly := ydexGrid(t, "cal_day_spent.json")["SpentDayATR"]
	for _, v := range []float64{0.7, 1.1} {
		if !containsValue(spentOnly, v) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v — уровень живой и разворот кривой лежит рядом", spentOnly, v)
		}
	}

	volume := ydexGrid(t, "cal_volume.json")
	for _, v := range volume["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: при 3.0 остаётся 27 сделок (10 на обучающее окно) при PF 0.981", v)
		}
	}

	window := ydexGrid(t, "cal_vol_window.json")
	// Максимум оси стоит на 1 (41 сделка, PF 2.191), дефолт ядра 3 хуже на 0.204 PF.
	if !containsValue(window["VolLookbackBars"], 1) {
		t.Errorf("cal_vol_window.json: VolLookbackBars = %v, не содержит 1 — максимум замера стоял бы вне сетки", window["VolLookbackBars"])
	}
	for _, v := range window["VolLookbackBars"] {
		if v > 12 {
			t.Errorf("cal_vol_window.json свипует VolLookbackBars=%v: кривая там уже плоская и ниже дефолта (16 -> PF 1.682)", v)
		}
	}

	risk := ydexGrid(t, "cal_risk.json")
	// Стоп 0.3 ATR это 0.85% цены; реальный круг издержек 0.125% съедает 14.6% риска — под чертой
	// 17%, по которой строку вырезали из domrf/ и elfv/.
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на YDEX круг издержек съедает там 14.6%% риска, под чертой 17%%", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: при 1.5 стоп достаёт лишь 11.7%% дней и вытесняет убыток в RSI-выход", v)
		}
	}
	if !containsValue(risk["TPDailyATR"], 0.4) {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит 0.4 — максимум каждой строки стоит на 0.5, край оси обязан её накрыть", risk["TPDailyATR"])
	}
	for _, v := range risk["TPDailyATR"] {
		if v > 1.5 {
			t.Errorf("cal_risk.json свипует TPDailyATR=%v: колонки 1.5, 2.0 и 2.5 совпадают побайтово — цель шире 1.5 ATR на YDEX недостижима", v)
		}
	}

	exit := ydexGrid(t, "cal_exit.json")
	for _, v := range exit["RSIUpper"] {
		if v > 80 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: разворот полосы стоит внутри оси (70 -> 1.778, 75 -> 1.769, 80 -> 1.581, 85 -> 1.204)", v)
		}
	}
}
