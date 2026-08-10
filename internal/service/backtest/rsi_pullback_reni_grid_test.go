package backtest

import (
	"math"
	"testing"
)

// reniGrid читает файл сеток RENI через общий хелпер.
func reniGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "reni", file)
}

// sameSet сообщает, содержит ли got ровно значения из want — без учёта порядка, но и без
// пропусков, дублей и лишних точек. Проверка длины одна такую подмену не ловит: набор
// [1,2] или [1,1] пройдёт len(got) == 2, хотя обе точки посторонние.
func sameSet(got []float64, want ...float64) bool {
	if len(got) != len(want) {
		return false
	}
	remaining := make(map[float64]int, len(want))
	for _, w := range want {
		remaining[w]++
	}
	for _, g := range got {
		if remaining[g] == 0 {
			return false
		}
		remaining[g]--
	}
	return true
}

// TestRENISignalGridsPinTheirMeasuredAxes сторожит оси, обоснованные замерами инструмента, а не
// вкусом. Каталог reni/ заводится копированием структуры ugld/, и типовая ошибка такой копии —
// притащить чужие оси целиком. По ширине RENI действительно сосед UGLD (дневной ATR 3.36% против
// 4.28%), поэтому здесь опасна ОБРАТНАЯ ошибка: перенести сужения, сделанные для DOMRF, где
// ATR 1.94% и сигналов дефицит.
func TestRENISignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := reniGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках, а любая другая пара не даёт сравнить гейт с его отсутствием", field, got)
		}
	}

	entry := reniGrid(t, "cal_entry.json")
	// Глубже 25 порог перестаёт отбирать откат: RSI(4) уходит под 30 1765 раз за историю,
	// это обычный шум, а не сетап.
	for _, v := range entry["RSILower"] {
		if v > 25 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 25 порог перестаёт отбирать откат (1765 кроссов под 30)", v)
		}
	}
	// Ниже 10 сигналов не остаётся уже на самом мелком из свипуемых периодов: у RSI(6) на
	// уровне 10 всего 51 будний кросс за 35.9 месяца, а более глубокий порог эту и без того
	// тонкую выборку истончает дальше.
	for _, v := range entry["RSILower"] {
		if v < 10 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 10 сигналов почти не остаётся (у RSI(6)@10 их всего 51)", v)
		}
	}
	// Уровень 10 обязан остаться. Скринер выбрал для RENI лучшей конфигурацией RSI 6/10, и
	// 51 будний кросс RSI(6)@10 эту точку выдерживает. На DOMRF таких кроссов было 18, и там
	// уровень 10 вырезали — при копировании оттуда сужение легко притащить по ошибке.
	hasDeep := false
	for _, v := range entry["RSILower"] {
		if v == 10 {
			hasDeep = true
		}
	}
	if !hasDeep {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — основную гипотезу скринера", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат — тот же порог и то же обоснование, что у
	// DOMRF: RSI(3) реагирует на любое дрожание цены, а не на настоящий откат в тренде.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат (то же обоснование, что у DOMRF)", v)
		}
	}
	// RSIUpper здесь не свипуется: 4x4x5 = 80 комбинаций на выборке около 36 сделок —
	// переобучение по построению. Полоса выхода меряется отдельно, файлом cal_exit.json.
	if got := entry["RSIUpper"]; len(got) != 0 {
		t.Errorf("cal_entry.json свипует RSIUpper=%v: полоса выхода принадлежит cal_exit.json", got)
	}

	trend := reniGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 23071 будних
	// в кэше, то есть окно прогрева занимает 1.8% истории.
	for _, v := range trend["EMASlow"] {
		if v > 200 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: окно прогрева растёт быстрее, чем польза от сглаживания", v)
		}
	}
	// Быстрая EMA обязана быть быстрее самой быстрой из медленных, иначе строка сетки означает
	// фильтр, который никогда не пропускает вход. Порог берётся из фактического минимума оси
	// EMASlow, а не зашивается константой: понижение оси EMASlow (например, до 20) иначе
	// тихо перестало бы совпадать с тем, что здесь проверяется.
	minSlow := math.Inf(1)
	for _, v := range trend["EMASlow"] {
		if v < minSlow {
			minSlow = v
		}
	}
	if !math.IsInf(minSlow, 1) {
		for _, v := range trend["EMAFast"] {
			if v >= minSlow {
				t.Errorf("cal_trend.json свипует EMAFast=%v: минимум оси EMASlow сейчас %.0f, такая пара мертва", v, minSlow)
			}
		}
	}
}

// TestRENIRiskGridsPinTheirMeasuredAxes сторожит оси стопа, цели и обоих гейтов. Каждое число
// ниже опирается на замер по кэшу RENI, а не на перенос с соседнего тикера: дневной ATR 3.36%,
// круг издержек 0.030 ATR, медианный дневной размах 0.88 ATR.
func TestRENIRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	day := reniGrid(t, "cal_day.json")
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1] — цена отключения гейта меряется в cal_screen.json", got)
	}
	// К 07:00 MSK медианный день уже прошёл 0.29 ATR, к 10:00 — 0.49. Пороги 0.1-0.2 из ugld/
	// отсекают медианный день на первом же баре: ветка «день только начался» становится мёртвой.
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: к 07:00 медианный день прошёл 0.29 ATR, порог мёртв", v)
		}
	}
	// Медианный день RENI покрывает 0.88 ATR, и порога 0.6 достигают 79.4% дней — на этом
	// инструменте он перестаёт быть гейтом.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для 79%% дней, это не гейт", v)
		}
	}
	// Соотношение двух веток гейта: положительный максимум FreshDayATR обязан быть строго
	// меньше минимума SpentDayATR. dayStateOK пропускает бар, когда день ещё не раскрылся
	// (used <= fresh*ATR) ИЛИ когда он уже исчерпан (used >= spent*ATR); если верх ветки
	// «свежий» дотягивается до низа ветки «исчерпан» (или выше), обе полосы дают true почти
	// на каждом баре, и UseDayATRGate=1 в лидерборде продолжит формально числиться включённым,
	// хотя фактически не отсекает ничего.
	maxFresh := 0.0
	for _, v := range day["FreshDayATR"] {
		if v > maxFresh {
			maxFresh = v
		}
	}
	minSpent := math.Inf(1)
	for _, v := range day["SpentDayATR"] {
		if v < minSpent {
			minSpent = v
		}
	}
	if maxFresh > 0 && !math.IsInf(minSpent, 1) && maxFresh >= minSpent {
		t.Errorf("cal_day.json: max(FreshDayATR)=%.2f >= min(SpentDayATR)=%.2f — ветки «день начался» и «день исчерпан» перекрываются, dayStateOK почти всегда true, и гейт перестаёт что-либо отсекать несмотря на UseDayATRGate=1", maxFresh, minSpent)
	}
	// RSILower в этой фазе не свипуется: у ugld/ он раздувает тему до 60 прогонов, а глубина
	// отката принадлежит cal_entry.json. Тема обязана остаться однотемной.
	if got := day["RSILower"]; len(got) != 0 {
		t.Errorf("cal_day.json свипует RSILower=%v: глубина отката принадлежит cal_entry.json", got)
	}

	spent := reniGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Левый край 0.6 стоит контрольной строкой «гейт почти выключен» (79.4% дней). Точки
	// 0.4-0.5 из ugld/ на RENI не гейтят вовсе и заняли бы прогоны впустую.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог не гейтит (0.6 достигают 79%% дней)", v)
		}
	}

	vol := reniGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Все четыре точки живые: гейт проходят 31.1% баров при 1.2, 24.7% при 1.5, 17.6% при 2.0,
	// 13.3% при 2.5. Выше 2.5 остаётся меньше восьмой части баров, и объёмный гейт начинает
	// резать выборку сильнее, чем несёт информации.
	for _, v := range vol["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.5 гейт проходит меньше 13%% баров", v)
		}
	}
	// Только две базы, 5 и 10: короткая быстрее реагирует на смену активности, длинная
	// устойчивее к одиночному всплеску. База 3 из ugld/ ловит один выброс объёма, база 14 —
	// размывает его; на вторичном гейте лишние степени свободы не окупаются.
	for _, v := range vol["VolBaseDays"] {
		if v != 5 && v != 10 {
			t.Errorf("cal_volume.json свипует VolBaseDays=%v: обоснованы только 5 и 10 (3 ловит одиночный всплеск, 14 размывает)", v)
		}
	}

	risk := reniGrid(t, "cal_risk.json")
	// Круг издержек стоит 0.030 дневного ATR: на стопе 0.3 ATR (= 1.01% цены) комиссия съедает
	// 10%. На DOMRF та же строка стоила 17% и была оттуда вырезана — при копировании сеток это
	// сужение легко притащить по ошибке, поэтому присутствие строки проверяется явно.
	hasTightStop := false
	for _, v := range risk["StopDailyATR"] {
		if v == 0.3 {
			hasTightStop = true
		}
	}
	if !hasTightStop {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 (издержки 0.030 ATR за круг эту строку лицензируют)", risk["StopDailyATR"])
	}
	// Нижняя граница оси: тот же круг издержек (0.030 ATR), который лицензирует строку 0.3
	// (10% риска), запрещает идти уже. На 0.2 доля выросла бы до 15%, на 0.15 — до 20%: это
	// ровно та черта, по которой DOMRF отверг свою строку 0.3 при 17%. «Попробуем стоп потуже»
	// не должно суметь добавить 0.2 или 0.15 молча.
	for _, v := range risk["StopDailyATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: издержки съели бы %.0f%% риска (круг 0.030 ATR) против 10%% на разрешённой строке 0.3; для сравнения, DOMRF отверг свою строку 0.3 при 17%%", v, 0.030/v*100)
		}
	}
	// Верх оси 1.3: медианный день покрывает 0.88 ATR, такой стоп переживает целиком 80.2% дней.
	// Шире — это уже не стоп, а отсутствие стопа, оплаченное размером позиции.
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: шире 1.3 ATR стоп перестаёт быть стопом (медианный день 0.88 ATR)", v)
		}
	}

	exit := reniGrid(t, "cal_exit.json")
	// Это единственное место, где меряется полоса выхода: cal_entry.json её намеренно не свипует.
	if got := exit["RSIUpper"]; !sameSet(got, 55, 60, 65, 70, 75, 80) {
		t.Errorf("cal_exit.json: RSIUpper = %v, want ровно {55,60,65,70,75,80} — cal_entry.json полосу выхода не свипует, а любая точка вне шкалы RSI или пропуск внутри неё сужает единственное место, где эта полоса измеряется", got)
	}

	trail := reniGrid(t, "cal_trail.json")
	if got := trail["UseTrail"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_trail.json: UseTrail = %v, want ровно [1] — тема меряет форму трейла, а не факт включения", got)
	}
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — трейл и RSI-выход конкурируют за одну сделку, и посторонняя точка не даёт замерить оба режима", got)
	}
	// Правый край 0.8, а не 0.6 как у ugld/: там потолок задавала цель TPDailyATR=0.6, выше
	// которой трейл не успевал взвестись; здесь ось цели поднята до 2.5, и трейлу нужно
	// пространство для по-настоящему позднего срабатывания.
	hasFarTrail := false
	for _, v := range trail["TrailDailyATR"] {
		if v == 0.8 {
			hasFarTrail = true
		}
	}
	if !hasFarTrail {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит правый край 0.8 (цель поднята до 2.5)", trail["TrailDailyATR"])
	}
}
