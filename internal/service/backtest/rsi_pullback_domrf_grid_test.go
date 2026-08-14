package backtest

import "testing"

// domrfGrid читает файл сеток DOMRF. Общий хелпер живёт в rsi_pullback_grid_test.go: у него
// появился второй потребитель (reni/), и копия разъехалась бы при первой же правке.
func domrfGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "domrf", file)
}

// TestDOMRFSignalGridsPinTheirMeasuredAxes сторожит решения, которые обоснованы замерами
// инструмента, а не вкусом. Сетки расширены владельцем 2026-08-14 по образцу lent/, и типовая
// ошибка такого копирования — притащить вместе с формой чужие числа: у LENT 36 месяцев истории и
// дневной ATR 3.16%, у DOMRF — 8.8 месяца и 2.02%. Все пороги ниже пересчитаны по кэшу DOMRF на
// 2026-08-14 (8515 30-минутных баров, 6221 будний; 173 дневных ATR).
func TestDOMRFSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	entry := domrfGrid(t, "cal_entry.json")

	// Уровень 10 мертвее, чем у reni (23 кросса), где угол был объявлен мёртвым заранее: за всю
	// историю DOMRF RSI(7) уходит под 10 девять раз по будням, RSI(6) — девятнадцать. После шести
	// гейтов входа и многодневного удержания выборки на такой точке не остаётся вовсе.
	for _, v := range entry["RSILower"] {
		if v < 15 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 15 сигналов не остаётся (RSI(7)@10 = 9 будних кроссов за 8.8 мес)", v)
		}
	}
	// RSI(3) на инструменте с ATR 2.02% покупает шум, а не откат.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум при ATR 2.02%%", v)
		}
	}
	// Ось RSIUpper внутри entry — сознательное отступление от однотемности, повторяющее
	// lent/cal_entry.json. Запрета на неё больше нет, но замер против неё есть, и он третий
	// подряд: широкая редакция дала pooled OOS PF 1.112 против 1.254-1.394 у узкой 4x4 (wush
	// 2.000 -> 1.674, lent 1.935 -> 1.355). Если ось всё же свипуется, она обязана покрывать
	// полосу целиком, а не пару соседних значений — иначе тема меряет ни то ни другое.
	if up, ok := entry["RSIUpper"]; ok && len(up) < 5 {
		t.Errorf("cal_entry.json свипует RSIUpper = %v: полосу выхода меряют либо целиком (>= 5 значений), либо в cal_exit.json", up)
	}

	if got := len(domrfGrid(t, "cal_exit.json")["RSIUpper"]); got != 6 {
		t.Errorf("cal_exit.json: RSIUpper имеет %d значений, want 6", got)
	}

	// Файл существует, чтобы оценить каждый гейт против его собственного отсутствия — свип
	// должен быть именно {0,1}, а не просто парой значений: [0,0] или [1,1] пройдёт проверку
	// длины, но убьёт единственный смысл файла.
	gates := domrfGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		got := gates[field]
		if len(got) != 2 || !((got[0] == 0 && got[1] == 1) || (got[0] == 1 && got[1] == 0)) {
			t.Errorf("cal_screen.json: %s должен свипуть ровно точки {0,1}, got %v", field, got)
		}
	}

	trend := domrfGrid(t, "cal_trend.json")
	if len(trend["EMAFast"]) != 5 || len(trend["EMASlow"]) != 7 {
		t.Errorf("cal_trend.json: сетка %vx%v, want 5x7", len(trend["EMAFast"]), len(trend["EMASlow"]))
	}
	// EMASlow=200 при train-окне 3 месяца съедает прогревом Lookback=420 баров около 26 будних
	// дней — примерно 40% обучающего окна протокола DOMRF. Ось это переживает, но нижний край
	// обязан остаться: без него у калибратора нет ни одной пары, успевающей прогреться.
	hasFastSlow := false
	for _, v := range trend["EMASlow"] {
		if v <= 70 {
			hasFastSlow = true
		}
	}
	if !hasFastSlow {
		t.Errorf("cal_trend.json: EMASlow = %v, нет ни одной пары короче 70 — при train 3 мес прогрев съест окно", trend["EMASlow"])
	}
}

// TestDOMRFRiskGridsPinTheirMeasuredAxes сторожит оси, выраженные в долях дневного ATR. Здесь
// копия чужого каталога опаснее всего: дневной ATR DOMRF (2.02% медианой) — самый узкий во
// вселенной, поэтому одна и та же цифра означает меньшее движение и большую долю издержек, чем
// у любого соседа.
func TestDOMRFRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	risk := domrfGrid(t, "cal_risk.json")

	// Круг издержек (0.05% за сторону) стоит 0.049 дневного ATR — дороже, чем у lent (0.032) и
	// fesh (0.023). На стопе 0.3 ATR комиссия съедает 16.3% риска, на 0.5 — 9.8%, на 0.7 — 7.0%.
	// Ось 0.3 оставлена владельцем сознательно как нижняя контрольная строка (калибровка
	// 2026-08-14 не выбрала её ни в одном фолде), но всё, что ниже, — уже не риск-параметр, а
	// комиссия. Отдельно: StopDailyATR=0 запрещён по всему каталогу тестом
	// TestRSIPullbackCalFilesValid — стратегия держит позицию через ночи и выходные.
	for _, v := range risk["StopDailyATR"] {
		if v < 0.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: круг издержек 0.049 ATR, ниже 0.3 стоп меряет комиссию", v)
		}
	}
	// Верхний край обязан оставаться в пределах правдоподобного: стоп 1.3 ATR переживается
	// целиком в 83.7% дней и выходом уже почти не является, всё что выше — фикция стопа.
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: уже 1.3 переживается целиком в 83.7%% дней", v)
		}
	}
	if len(risk["TPDailyATR"]) != 5 {
		t.Errorf("cal_risk.json: TPDailyATR имеет %d значений, want 5", len(risk["TPDailyATR"]))
	}

	day := domrfGrid(t, "cal_day.json")
	// К 07:00 MSK медианный день уже прошёл 0.32 ATR, к 09:00 — 0.44, к 10:00 — 0.55. Порог 0.2
	// оставляет ветке «день только начался» 4.3% будних баров и делает её почти мёртвой; 0.3 —
	// 9.5%, 0.4 — 17.2%. Принятый литерал стоит на 0.3.
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: к 07:00 медианный день прошёл 0.32 ATR, порог мёртв", v)
		}
	}
	// Медианный день DOMRF покрывает 0.99 ATR предыдущего дня, и порога 0.6 достигают 84.9%
	// дней — на этом инструменте он перестаёт быть гейтом.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для 85%% дней, это не гейт", v)
		}
	}
	// Фаза day всегда идёт со включённым гейтом: цена его отключения меряется cal_screen.json.
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1]", got)
	}

	spent := domrfGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Ось начинается с 0.6 как контрольная строка «гейт почти выключен» (84.9% дней) и уходит
	// до 1.75 (12.2% дней). Ниже 0.6 гейта не остаётся вовсе.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог почти не гейтит (0.6 достигают 85%% дней)", v)
		}
	}

	vol := domrfGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Потолок поднят с 2.0 до 2.5 вместе с расширением каталога 2026-08-14. Это исполнимо там,
	// где прежняя редакция сетки входа была дефицитной: на принятом уровне RSILower 35 инструмент
	// даёт 570 будних кроссов RSI(4) за историю против 155 на уровне 15. Выше 2.5 гейт срезает
	// выборку в ноль на любом уровне входа.
	for _, v := range vol["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.5 гейт срезает выборку в ноль", v)
		}
	}
	// Только две базы, 5 и 10: короткая быстрее реагирует на смену активности, длинная
	// устойчивее к одиночному всплеску. База 3 из ugld/ на растущем обороте DOMRF
	// систематически завышает отношение объёма к базе, потому что база отстаёт от тренда.
	for _, v := range vol["VolBaseDays"] {
		if v != 5 && v != 10 {
			t.Errorf("cal_volume.json свипует VolBaseDays=%v: обоснованы только 5 и 10 (короткая база отстаёт от растущего оборота DOMRF)", v)
		}
	}

	trail := domrfGrid(t, "cal_trail.json")
	if len(trail["UseRSIExit"]) != 2 {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want обе точки [0,1]", trail["UseRSIExit"])
	}
	// Правый край 0.8, а не 0.6 как у ugld/: там потолок задавала цель TPDailyATR=0.6, выше
	// которой трейл не успевал взвестись до закрытия сделки; здесь ось цели поднята до 2.5,
	// и трейл получает пространство для по-настоящему позднего срабатывания.
	hasFarTrail := false
	for _, v := range trail["TrailDailyATR"] {
		if v == 0.8 {
			hasFarTrail = true
		}
	}
	if !hasFarTrail {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит правый край 0.8 (цель поднята до 2.5, трейл должен получить пространство)", trail["TrailDailyATR"])
	}
}
