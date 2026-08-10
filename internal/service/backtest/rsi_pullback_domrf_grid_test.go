package backtest

import "testing"

// domrfGrid читает файл сеток DOMRF. Общий хелпер живёт в rsi_pullback_grid_test.go: у него
// появился второй потребитель (reni/), и копия разъехалась бы при первой же правке.
func domrfGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "domrf", file)
}

// TestDOMRFSignalGridsPinTheirMeasuredAxes сторожит решения, которые обоснованы замерами
// инструмента, а не вкусом. Каталог domrf/ заведён копированием структуры ugld/, и типовая
// ошибка такой копии — притащить вместе с ней чужие оси: на UGLD дневной ATR 4.28%, на DOMRF
// 1.94%, и половина осей UGLD здесь либо мертва, либо неисполнима.
func TestDOMRFSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	entry := domrfGrid(t, "cal_entry.json")

	// RSI(6) пересекает 10 вниз 18 раз за ВСЮ историю инструмента: после шести гейтов входа
	// и многодневного удержания на этой точке не остаётся выборки вовсе.
	for _, v := range entry["RSILower"] {
		if v < 15 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 15 сигналов не остаётся (RSI(6)@10 = 18 кроссов за 8.4 мес)", v)
		}
	}
	// RSI(3) на инструменте с ATR 1.94% покупает шум, а не откат.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум при ATR 1.94%%", v)
		}
	}
	// Полоса выхода меряется отдельным cal_exit.json. В фазе entry она стоила бы 12x5
	// степеней свободы на выборке в 20-30 сделок — переобучение по построению.
	if _, ok := entry["RSIUpper"]; ok {
		t.Error("cal_entry.json свипует RSIUpper: полоса выхода принадлежит cal_exit.json")
	}

	if got := len(domrfGrid(t, "cal_exit.json")["RSIUpper"]); got != 5 {
		t.Errorf("cal_exit.json: RSIUpper имеет %d значений, want 5", got)
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
	if len(trend["EMAFast"]) != 3 || len(trend["EMASlow"]) != 4 {
		t.Errorf("cal_trend.json: сетка %vx%v, want 3x4", len(trend["EMAFast"]), len(trend["EMASlow"]))
	}
}

// TestDOMRFRiskGridsPinTheirMeasuredAxes сторожит оси, выраженные в долях дневного ATR. Именно
// здесь копия ugld/ опаснее всего: там ATR 4.28% от цены, здесь 1.94%, поэтому одна и та же
// цифра означает вдвое меньшее движение и вдвое большую долю издержек.
func TestDOMRFRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	risk := domrfGrid(t, "cal_risk.json")

	// Круг издержек (0.05% за сторону) стоит 0.052 дневного ATR. Стоп 0.3 ATR = 0.58% цены,
	// из которых 17% съедает комиссия, а медианный день покрывает 0.99 ATR — такой стоп сидит
	// внутри обычного внутридневного шума и будет снят сносом, а не провалом сетапа. Отдельно,
	// StopDailyATR=0 запрещён везде: стратегия держит позицию через ночи и выходные, и
	// многодневное удержание без стопа недопустимо (эту границу v < 0.5 уже ловит, а
	// TestRSIPullbackCalFilesValid проверяет её по всему каталогу независимо от этого теста).
	for _, v := range risk["StopDailyATR"] {
		if v < 0.5 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: ниже 0.5 стоп внутри дневного шума (медианный день 0.99 ATR)", v)
		}
	}
	if len(risk["TPDailyATR"]) != 4 {
		t.Errorf("cal_risk.json: TPDailyATR имеет %d значений, want 4", len(risk["TPDailyATR"]))
	}

	day := domrfGrid(t, "cal_day.json")
	// К 07:00 MSK медианный день уже прошёл 0.31 ATR, к 10:00 — 0.55. Порог 0.2 из ugld/
	// отсекает медианный день на первом же баре: ветка «день только начался» становится мёртвой.
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: к 07:00 медианный день прошёл 0.31 ATR, порог мёртв", v)
		}
	}
	// Медианный день DOMRF покрывает 0.99 ATR против 0.67 у UGLD, и порога 0.6 достигают
	// 88.2% дней — на этом инструменте он перестаёт быть гейтом.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для 88%% дней, это не гейт", v)
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
	// Ось начинается с 0.6 как контрольная строка «гейт почти выключен» и уходит до 1.75
	// (9.0% дней). Точки 0.4-0.5 из ugld/ здесь означают почти отсутствие гейта: 0.6
	// на DOMRF уже достигают 88.2% будних дней, а 0.4-0.5 — и того больше.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог почти не гейтит (0.6 достигают 88%% дней)", v)
		}
	}

	vol := domrfGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Верхняя граница 2.0: гейт проходят 39.2% баров при 1.0, 32.3% при 1.2, 25.4% при 1.5,
	// 17.5% при 2.0. Выборка дефицитная (146 кроссов RSI(4)@15 за всю историю), поэтому
	// строки 2.5 и 3.0 из ugld/ здесь неисполнимы — они срежут и без того тонкую выборку в ноль.
	for _, v := range vol["VolMult"] {
		if v > 2.0 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.0 неисполнимо на дефицитной выборке DOMRF (146 кроссов за всю историю)", v)
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
	// которой трейл не успевал взвестись до закрытия сделки; здесь ось цели поднята до 2.0,
	// и трейл получает пространство для по-настоящему позднего срабатывания.
	hasFarTrail := false
	for _, v := range trail["TrailDailyATR"] {
		if v == 0.8 {
			hasFarTrail = true
		}
	}
	if !hasFarTrail {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит правый край 0.8 (цель поднята до 2.0, трейл должен получить пространство)", trail["TrailDailyATR"])
	}
}
