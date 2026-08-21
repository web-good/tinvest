package backtest

import "testing"

// sibnGrid читает файл сеток SIBN через общий хелпер.
func sibnGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "sibn", file)
}

// TestSIBNGridsPinTheirMeasuredAxes сторожит оси каталога sibn/. Каталог собран 2026-08-21
// копированием формы svav/ с пересадкой каждой оси на замеры самого SIBN (36 735 30-минутных
// баров в кэше, 36 321 в расчётном окне 36.0 месяца с 2023-08-21, из них 25 609 будних). Три
// оси изменены против образца, и каждое изменение опирается на замер:
//
//   - RSIPeriod расширен до 8. Планка живого угла в каталоге — 29 кроссов (столько дал
//     слабейший угол LSNGP, объявленный мёртвым; у RENI было 23). RSI(8)@10 даёт 34 и проходит
//     её, RSI(9)@10 даёт 23 и не проходит. Направление выбрано по свойству инструмента: у SIBN
//     САМЫЙ НИЗКИЙ дневной ATR каталога (2.52% против 4.38% у SVAV), RSI колеблется мельче, и
//     медленный период правдоподобнее. Ось RSILower при этом НЕ расширяется: каталог дважды
//     получил отрицательный результат именно от неё (WUSH 2.000 -> 1.674 при растяжке до 50).
//   - Верхний край дневного гейта сдвинут с 1.5 на 1.3: ветка «день исчерпан» при 1.5 отбирает
//     лишь 5.3% баров (на SVAV было 7.7% и край держался). Взамен ось уплотнена живыми
//     уровнями 0.7 (46.9%) и 1.1 (16.4%).
//   - VolMult остановлен на 2.5, хотя по доле баров ось не исчерпана (3.0 пропускает ещё
//     14.3%). Решает счёт сделок: точечные прогоны дают 2.5 -> 56 сделок за 36 месяцев,
//     3.0 -> 45, то есть 19 и 15 на 12-месячное обучающее окно. При -min-trades 20 точка 3.0
//     не может быть выбрана никогда — мёртвая точка создавала бы видимость проверенного края.
func TestSIBNGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := sibnGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := sibnGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: на SIBN он живой — RSI(4)@10 даёт 359 будних кроссов за 36
	// месяцев, слабейший угол сетки RSI(8)@10 — 34, выше планки живого угла 29.
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на SIBN это живой угол (359 кроссов RSI(4)@10)", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат: RSI(3)@10 даёт 736 кроссов — вдвое больше
	// RSI(4), и это дыхание цены, а не откаты.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
		// Верхний край 8: RSI(9)@10 даёт 23 кросса за 36 месяцев — ниже планки живого угла 29,
		// по которой угол объявляли мёртвым на LSNGP и RENI.
		if v > 8 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: слабейший угол RSI(9)@10 даёт 23 кросса, ось мертва за 8", v)
		}
	}
	// Расширение оси периода — сознательное и первое в каталоге, поэтому прибито явно.
	if !containsValue(entry["RSIPeriod"], 8) {
		t.Errorf("cal_entry.json: RSIPeriod = %v, не содержит 8 — на SIBN угол живой (34 кросса RSI(8)@10) и ось расширена по замеру", entry["RSIPeriod"])
	}

	trend := sibnGrid(t, "cal_trend.json")
	// Ось тренда режется только вместе с этим тестом. На SIBN доля баров с EMAFast > EMASlow
	// укладывается в 45.1-46.3% на ВСЕХ 35 парах: выбор пары меняет не объём допуска, а то,
	// какие именно бары в него попадают. Значит, ни одна пара не мертва по выборке, и сужать
	// сетку не за что — а разница PF между парами читается как качество фильтра.
	if !containsValue(trend["EMASlow"], 200) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 200 — медленный край оси обязан остаться, допуск у него тот же 45.2%%", trend["EMASlow"])
	}
	for _, fast := range trend["EMAFast"] {
		for _, slow := range trend["EMASlow"] {
			if fast >= slow {
				t.Errorf("cal_trend.json: пара EMAFast=%v EMASlow=%v вырождена — быстрая обязана быть быстрее медленной", fast, slow)
			}
		}
	}

	risk := sibnGrid(t, "cal_risk.json")
	// Строка 0.3 сохранена намеренно, но она ближе к черте, чем у любого другого тикера: при
	// дневном ATR 2.52% круг издержек 0.1% съедает 13.2% риска против черты 17%, по которой её
	// вырезали из domrf/. Если дефолтная комиссия когда-нибудь вырастет, эту строку пересмотреть
	// первой.
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на SIBN комиссия съедает там 13.2%% риска, строка ещё живая", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v <= 0 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: стоп обязан быть положительным", v)
		}
		// Верхний край 1.3: уровня 1.5 ATR достаёт лишь 11.8% дней, такой стоп перестаёт быть
		// защитой и становится способом вытеснить убыток в RSI-выход.
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: до него доходит меньше 12%% дней — это не стоп", v)
		}
	}

	day := sibnGrid(t, "cal_day.json")
	// Ветка «свежий день» ловит 11.0% будних баров при пороге 0.3 и 30.1% при 0.5; ноль в оси —
	// это выключенная ветка, и она обязана остаться, потому что на всех прод-тикерах каталога
	// победил именно ноль.
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — выключенная ветка обязана быть в сетке", day["FreshDayATR"])
	}
	// Верхний край ветки «день исчерпан» сдвинут с 1.5 на 1.3: при 1.5 через ветку проходит
	// 5.3% баров — это уже не гейт, а отбор десятка баров.
	for _, v := range day["SpentDayATR"] {
		if v > 1.3 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: на SIBN порог 1.5 отбирает 5.3%% баров, край оси стоит на 1.3", v)
		}
	}

	daySpent := sibnGrid(t, "cal_day_spent.json")
	for _, v := range daySpent["SpentDayATR"] {
		if v > 1.3 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: край оси стоит на 1.3 по замеру 9.2%% баров", v)
		}
	}
	// Уплотнение оси живыми уровнями — часть решения, а не случайность.
	for _, want := range []float64{0.7, 1.1} {
		if !containsValue(daySpent["SpentDayATR"], want) {
			t.Errorf("cal_day_spent.json: SpentDayATR = %v, не содержит %v — уровень живой (46.9%% и 16.4%% баров)", daySpent["SpentDayATR"], want)
		}
	}

	volume := sibnGrid(t, "cal_volume.json")
	// Край 2.5 унаследован от svav/ и остаётся; 3.0 отвергнут по счёту сделок, а не по доле
	// баров: 45 сделок за 36 месяцев это 15 на обучающее окно при -min-trades 20.
	if !containsValue(volume["VolMult"], 2.5) {
		t.Errorf("cal_volume.json: VolMult = %v, не содержит 2.5 — край оси, живой по счёту сделок (56 за 36 мес.)", volume["VolMult"])
	}
	for _, v := range volume["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: при 3.0 остаётся 45 сделок (15 на train), -min-trades 20 топит точку всегда", v)
		}
	}

	exit := sibnGrid(t, "cal_exit.json")
	// Полоса выхода рабочая на всём диапазоне: кроссов вверх RSI(4) от 2876 (55) до 963 (80).
	for _, v := range exit["RSIUpper"] {
		if v <= 50 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: выход обязан стоять выше средней линии", v)
		}
	}
}
