package backtest

import "testing"

// svavGrid читает файл сеток SVAV через общий хелпер.
func svavGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "svav", file)
}

// TestSVAVGridsPinTheirMeasuredAxes сторожит оси каталога svav/. Каталог собран 2026-08-18
// копированием формы ivat/ с пересадкой каждой оси на замеры самого SVAV (33 093 30-минутных
// бара, 23 869 будних, 36.0 месяца с 2023-08-21). Три особенности инструмента, из-за которых
// чужие обоснования сюда не переносятся: истории ровно 36 месяцев, поэтому схема прогонов
// ШТАТНАЯ (-months 36 -train-months 12 -test-months 6) в отличие от ivat/; трендовый допуск
// не зависит от пары EMA (42.4-43.6% на всех 35 парах против 29.4-36.0% с монотонной
// зависимостью у IVAT); объёмный гейт мягкий — на каноническом верхнем крае VolMult=2.0 через
// него проходит ещё 27.3% баров, поэтому ось расширена до 2.5.
func TestSVAVGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := svavGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := svavGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: на SVAV он живой — RSI(4)@10 даёт 264 будних кросса за 36
	// месяцев, слабейший угол сетки RSI(7)@10 — 43, выше LSNGP (29) и RENI (23).
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на SVAV это живой угол (264 кросса RSI(4)@10)", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат: RSI(3)@10 даёт 575 кроссов — вдвое больше
	// RSI(4), и это дыхание цены, а не откаты.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
	}

	trend := svavGrid(t, "cal_trend.json")
	// Ось тренда режется только вместе с этим тестом. На SVAV доля баров с EMAFast > EMASlow
	// укладывается в 42.4-43.6% на ВСЕХ 35 парах: выбор пары меняет не объём допуска, а то,
	// какие именно бары в него попадают. Значит, ни одна пара не мертва по выборке, и сужать
	// сетку не за что — а разница PF между парами читается как качество фильтра.
	if !containsValue(trend["EMASlow"], 200) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 200 — медленный край оси обязан остаться, допуск у него тот же 42.5%%", trend["EMASlow"])
	}
	for _, fast := range trend["EMAFast"] {
		for _, slow := range trend["EMASlow"] {
			if fast >= slow {
				t.Errorf("cal_trend.json: пара EMAFast=%v EMASlow=%v вырождена — быстрая обязана быть быстрее медленной", fast, slow)
			}
		}
	}

	risk := svavGrid(t, "cal_risk.json")
	// Строка 0.3 сохранена намеренно: при дневном ATR 4.38% круг издержек 0.1% съедает 7.6%
	// риска — вдвое ниже черты 17%, по которой её вырезали из domrf/.
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на SVAV комиссия съедает там 7.6%% риска, строка живая", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v <= 0 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: стоп обязан быть положительным", v)
		}
		// Верхний край 1.3: уровня 1.5 ATR достаёт лишь 12.7% дней, такой стоп перестаёт быть
		// защитой и становится способом вытеснить убыток в RSI-выход.
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: до него доходит меньше 13%% дней — это не стоп", v)
		}
	}

	day := svavGrid(t, "cal_day.json")
	// Ветка «свежий день» ловит 12.9% будних баров при пороге 0.3 и 36.0% при 0.5; ноль в оси —
	// это выключенная ветка, и она обязана остаться, потому что на всех прод-тикерах каталога
	// победил именно ноль.
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — выключенная ветка обязана быть в сетке", day["FreshDayATR"])
	}

	volume := svavGrid(t, "cal_volume.json")
	// Ось расширена против образца ivat/ по замеру, а не по желанию перебрать больше: при базе
	// 14 дней канонический верхний край 2.0 пропускает ещё 27.3% баров (у IVAT там было 15.8%),
	// то есть отсекающая способность оси не исчерпана.
	if !containsValue(volume["VolMult"], 2.5) {
		t.Errorf("cal_volume.json: VolMult = %v, не содержит 2.5 — на SVAV порог 2.0 пропускает ещё 27.3%% баров, ось не исчерпана", volume["VolMult"])
	}

	exit := svavGrid(t, "cal_exit.json")
	// Полоса выхода рабочая на всём диапазоне: кроссов вверх RSI(4) от 2873 (55) до 786 (80).
	for _, v := range exit["RSIUpper"] {
		if v <= 50 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: выход обязан стоять выше средней линии", v)
		}
	}
}
