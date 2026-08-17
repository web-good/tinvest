package backtest

import "testing"

// ivatGrid читает файл сеток IVAT через общий хелпер.
func ivatGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "ivat", file)
}

// TestIVATGridsPinTheirMeasuredAxes сторожит оси каталога ivat/. Каталог собран 2026-08-17
// копированием формы nvtk/ с пересадкой каждой оси на замеры самого IVAT (26 979 30-минутных
// баров, 18 772 будних, 26.0 месяца с 2024-06-18). Две особенности инструмента, из-за которых
// чужие обоснования сюда не переносятся: истории всего 26 месяцев (схема прогонов адаптирована
// до train 9 / OOS 4) и трендовый фильтр открыт 29.4-36.0% времени — самый закрытый в каталоге.
func TestIVATGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := ivatGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках", field, got)
		}
	}

	entry := ivatGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению", v)
		}
	}
	// Уровень 10 обязан остаться: на IVAT он живой — RSI(4)@10 даёт 178 будних кроссов за 26
	// месяцев, слабейший угол RSI(7)@10 — 31, на уровне LSNGP (29) и RENI (23).
	if !containsValue(entry["RSILower"], 10) {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на IVAT это живой угол (178 кроссов RSI(4)@10)", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
	}

	trend := ivatGrid(t, "cal_trend.json")
	// Ось тренда режется только вместе с этим тестом: на IVAT доля баров с EMAFast > EMASlow
	// укладывается в 29.4-36.0% на всех двадцати замеренных парах, и зависимость монотонна —
	// чем медленнее EMASlow, тем уже допуск. Мёртвых пар в сетке нет, сужать её не за что.
	if !containsValue(trend["EMASlow"], 200) {
		t.Errorf("cal_trend.json: EMASlow = %v, не содержит 200 — медленный край оси меряет, во что обходится самый узкий допуск (29.4%%)", trend["EMASlow"])
	}
	for _, fast := range trend["EMAFast"] {
		for _, slow := range trend["EMASlow"] {
			if fast >= slow {
				t.Errorf("cal_trend.json: пара EMAFast=%v EMASlow=%v вырождена — быстрая обязана быть быстрее медленной", fast, slow)
			}
		}
	}

	risk := ivatGrid(t, "cal_risk.json")
	// Строка 0.3 сохранена намеренно: при дневном ATR 3.59% круг издержек 0.1% съедает 9.4%
	// риска — заметно ниже черты 17%, по которой её вырезали из domrf/.
	if !containsValue(risk["StopDailyATR"], 0.3) {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на IVAT комиссия съедает там 9.4%% риска, строка живая", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v <= 0 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: стоп обязан быть положительным", v)
		}
	}

	day := ivatGrid(t, "cal_day.json")
	// Ветка «свежий день» ловит 7.5% будних баров при пороге 0.3 и 23.5% при 0.5; ноль в оси —
	// это выключенная ветка, и она обязана остаться, потому что на всех прод-тикерах каталога
	// победил именно ноль.
	if !containsValue(day["FreshDayATR"], 0) {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — выключенная ветка обязана быть в сетке", day["FreshDayATR"])
	}

	exit := ivatGrid(t, "cal_exit.json")
	// Полоса выхода рабочая на всём диапазоне: кроссов вверх RSI(4) от 2299 (55) до 614 (80).
	for _, v := range exit["RSIUpper"] {
		if v <= 50 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: выход обязан стоять выше средней линии", v)
		}
	}
}

// containsValue сообщает, есть ли значение в оси.
func containsValue(axis []float64, want float64) bool {
	for _, v := range axis {
		if v == want {
			return true
		}
	}
	return false
}
