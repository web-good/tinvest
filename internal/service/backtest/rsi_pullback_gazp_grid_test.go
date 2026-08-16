package backtest

import (
	"math"
	"testing"
)

// gazpGrid читает файл сеток GAZP через общий хелпер.
func gazpGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "gazp", file)
}

// TestGAZPSignalGridsPinTheirMeasuredAxes сторожит оси сигнальных тем. Каталог gazp/ пересобран
// 2026-08-15 копированием формы lent/ и domrf/ с пересадкой каждой оси на замеры самого GAZP
// (36 770 30-минутных баров, 25 744 будних, 36.0 месяца с 2023-08-14). Типовая ошибка такой
// копии — притащить вместе с формой чужие обоснования: у GAZP дневной ATR 2.71% против 3.16% у
// LENT, а кроссов RSI в разы больше, потому что инструмент падает всю историю.
func TestGAZPSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := gazpGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках, а любая другая пара не даёт сравнить гейт с его отсутствием", field, got)
		}
	}

	entry := gazpGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению (50 — средняя линия RSI)", v)
		}
	}
	// Уровень 10 обязан остаться. На GAZP он живой в отличие от DOMRF, где его вырезали: самый
	// слабый угол сетки RSI(7)@10 даёт 54 будних кросса за 36 месяцев (RSI(4)@10 — 340), то есть
	// больше, чем у LENT (29) и RENI (23), где угол объявляли мёртвым. Уровень вернули в ось
	// 2026-08-15 именно по этому замеру, и его молчаливое исчезновение при следующем
	// копировании соседнего каталога обязано валить сборку.
	hasDeep := false
	for _, v := range entry["RSILower"] {
		if v == 10 {
			hasDeep = true
		}
	}
	if !hasDeep {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на GAZP это живой угол (54 кросса RSI(7)@10 за 36 месяцев)", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат: RSI(3) реагирует на любое дрожание цены.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
	}
	// Вырожденная полоса запрещена: вход требует креста вниз через RSILower, выход — креста
	// вверх через RSIUpper. Если верхняя граница не выше нижней, строка сетки означает позицию,
	// которую закрывает любой отскок, а в лидерборде она выглядит обычной строкой с высоким
	// win rate.
	maxLower := 0.0
	for _, v := range entry["RSILower"] {
		if v > maxLower {
			maxLower = v
		}
	}
	for _, v := range entry["RSIUpper"] {
		if v <= maxLower {
			t.Errorf("cal_entry.json свипует RSIUpper=%v при максимуме RSILower=%.0f: полоса вырождена, выход сработал бы на любом отскоке после входа", v, maxLower)
		}
		if v > 100 {
			t.Errorf("cal_entry.json свипует RSIUpper=%v: шкала RSI кончается на 100", v)
		}
	}

	trend := gazpGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 25 744 будних в
	// кэше, окно прогрева занимает 1.6% истории. Шире двигать нечего — прогрев растёт быстрее
	// пользы от сглаживания.
	for _, v := range trend["EMASlow"] {
		if v > 200 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: окно прогрева растёт быстрее, чем польза от сглаживания", v)
		}
	}
	// Быстрая EMA обязана быть быстрее самой быстрой из медленных, иначе строка сетки означает
	// фильтр, который никогда не пропускает вход. Порог берётся из фактического минимума оси.
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

	exit := gazpGrid(t, "cal_exit.json")
	// Единственное место, где меряется полоса выхода однотемно: cal_entry.json свипует RSIUpper
	// только в связке с нижней границей.
	if got := exit["RSIUpper"]; !sameSet(got, 55, 60, 65, 70, 75, 80) {
		t.Errorf("cal_exit.json: RSIUpper = %v, want ровно {55,60,65,70,75,80} — пропуск внутри шкалы сузил бы единственное однотемное измерение полосы выхода", got)
	}
}

// TestGAZPRiskGridsPinTheirMeasuredAxes сторожит оси стопа, цели и обоих гейтов. Каждое число
// ниже опирается на замер по кэшу GAZP: медианный дневной ATR 2.71% цены, круг издержек 0.037
// ATR, медианный дневной размах 0.86 ATR предыдущего дня.
func TestGAZPRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	day := gazpGrid(t, "cal_day.json")
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1] — цена отключения гейта меряется в cal_screen.json", got)
	}
	// День GAZP раскрывается МЕДЛЕННЕЕ соседей: медиана доли ATR ко второму бару 0.23 против
	// 0.35 у LENT и 0.33 у WUSH. Поэтому порог 0.3 оставляет ветке «день только начался» 13.6%
	// будних баров, 0.4 — 23.0%, 0.5 — 35.7%. Ось [0, 0.3, 0.4, 0.5] измерима целиком, но точки
	// ниже 0.25 оставили бы ветке меньше десятой части баров и дали бы ложный вывод «ветка не
	// нужна» — тот же капкан, из-за которого ось FESH нельзя было переносить на LENT.
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: ветке остаётся меньше 10%% будних баров (порог 0.3 даёт 13.6%%)", v)
		}
	}
	// Порог 0.6 проходят 52.8% будних баров — на этом инструменте он перестаёт быть гейтом, и
	// его контрольная роль принадлежит cal_day_spent.json.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим больше чем для половины баров, это не гейт", v)
		}
	}
	// Соотношение двух веток: положительный максимум FreshDayATR обязан быть строго меньше
	// минимума SpentDayATR, иначе dayStateOK даёт true почти на каждом баре и гейт формально
	// включён, фактически не отсекая ничего.
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
		t.Errorf("cal_day.json: max(FreshDayATR)=%.2f >= min(SpentDayATR)=%.2f — ветки «день начался» и «день исчерпан» перекрываются, dayStateOK почти всегда true", maxFresh, minSpent)
	}
	// Глубина отката принадлежит cal_entry.json: тема дня обязана остаться однотемной.
	if got := day["RSILower"]; len(got) != 0 {
		t.Errorf("cal_day.json свипует RSILower=%v: глубина отката принадлежит cal_entry.json", got)
	}

	spent := gazpGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Левый край 0.6 стоит контрольной строкой «гейт почти выключен» (52.8% баров). Ниже
	// порог не гейтит вовсе и занял бы прогоны впустую.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог не гейтит (0.6 проходят 53%% баров)", v)
		}
	}

	vol := gazpGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Все четыре точки живые: при базе 5 дней гейт проходят 30.1% баров при 1.2, 22.3% при 1.5,
	// 14.4% при 2.0, 10.0% при 2.5. Выше 2.5 остаётся меньше десятой части баров.
	for _, v := range vol["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.5 гейт проходит меньше 10%% баров", v)
		}
	}
	for _, v := range vol["VolBaseDays"] {
		if v != 5 && v != 10 {
			t.Errorf("cal_volume.json свипует VolBaseDays=%v: обоснованы только 5 и 10 (3 ловит одиночный всплеск, 14 размывает)", v)
		}
	}

	risk := gazpGrid(t, "cal_risk.json")
	// Круг издержек на GAZP стоит 0.037 дневного ATR — дороже, чем у LENT (0.032), WUSH (0.024)
	// и FESH (0.023), потому что инструмент узкий: медианный дневной ATR 2.71%. На стопе 0.3 ATR
	// комиссия съедает 12.3% риска, и это ближе к черте 17%, по которой строку 0.3 вырезали из
	// domrf/, чем у любого другого каталога, где её оставили. Строка сохранена сознательно, и
	// её удаление обязано быть решением, а не побочным эффектом правки.
	hasTightStop := false
	for _, v := range risk["StopDailyATR"] {
		if v == 0.3 {
			hasTightStop = true
		}
	}
	if !hasTightStop {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — строка дорогая (12.3%% риска на издержки), но замеренная", risk["StopDailyATR"])
	}
	// Нижняя граница оси: на 0.2 издержки съели бы 18.5% риска, на 0.15 — 24.7%. Это уже за
	// чертой, по которой DOMRF отверг свою строку 0.3 при 17%.
	for _, v := range risk["StopDailyATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: издержки съели бы %.0f%% риска (круг 0.037 ATR) против 12%% на разрешённой строке 0.3", v, 0.037/v*100)
		}
	}
	// Верх оси 1.3: медианный день покрывает 0.86 ATR, стоп 1.3 переживает целиком 79.3% дней.
	// Шире — это уже не стоп, а его отсутствие, оплаченное размером позиции.
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: шире 1.3 ATR стоп перестаёт быть стопом (медианный день 0.86 ATR, стоп 1.3 переживает 79%% дней)", v)
		}
	}
	// Цель действующего литерала обязана оставаться в оси: без неё тема не сравнит новую
	// конфигурацию с той, что торгует в проде.
	hasLiveTarget := false
	for _, v := range risk["TPDailyATR"] {
		if v == 0.7 {
			hasLiveTarget = true
		}
	}
	if !hasLiveTarget {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит боевую цель 0.7 — тему нельзя будет сравнить с торгующей конфигурацией", risk["TPDailyATR"])
	}

	trail := gazpGrid(t, "cal_trail.json")
	if got := trail["UseTrail"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_trail.json: UseTrail = %v, want ровно [1] — тема меряет форму трейла, а не факт включения", got)
	}
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — трейл и RSI-выход конкурируют за одну сделку, и посторонняя точка не даёт замерить оба режима", got)
	}
	// Правый край 0.8 (2.2% цены при ATR 2.71%): ось цели в cal_risk.json идёт до 2.5, и трейлу
	// нужно пространство для позднего срабатывания. Шире 0.9 трейл отставал бы от цены дальше,
	// чем движется медианный день (0.86 ATR).
	hasFarTrail := false
	for _, v := range trail["TrailDailyATR"] {
		if v == 0.8 {
			hasFarTrail = true
		}
		if v > 0.9 {
			t.Errorf("cal_trail.json свипует TrailDailyATR=%v: шире медианного дневного размаха (0.86 ATR) трейл не догоняет цену", v)
		}
	}
	if !hasFarTrail {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит правый край 0.8 (цель поднята до 2.5)", trail["TrailDailyATR"])
	}
}
