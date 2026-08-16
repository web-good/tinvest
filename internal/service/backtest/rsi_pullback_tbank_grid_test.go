package backtest

import (
	"math"
	"testing"
)

// tGrid читает файл сеток T через общий хелпер. Каталог называется t/ по биржевому тикеру,
// пакет параметров — strategy/tbank, потому что однобуквенное имя пакета сталкивается с
// идиомой t *testing.T.
func tGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "t", file)
}

// TestTSignalGridsPinTheirMeasuredAxes сторожит оси сигнальных тем. Каталог t/ переписан в
// расширенную редакцию 2026-08-16 при первой ПОЛНОЙ калибровке тикера — прежний литерал был
// получен одиночным in-sample прогоном 2026-07-31 без walk-forward вовсе, а сетки под него
// никогда не прибивались тестом. Каждая ось пересажена на замеры самого T по кэшу
// T_Minutes30.json: 34 997 30-минутных баров, из них 25 031 будний, 757 будних дневок,
// 36.0 месяца с 2023-08-16 по 2026-08-16.
func TestTSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := tGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках, а любая другая пара не даёт сравнить гейт с его отсутствием", field, got)
		}
	}

	entry := tGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению (50 — средняя линия RSI)", v)
		}
	}
	// Уровень 10 обязан остаться. На T он живее, чем у любого соседа, где его сохраняли: самый
	// слабый угол сетки RSI(7)@10 даёт 89 будних кроссов за 36 месяцев против 54 у GAZP, тогда
	// как у LENT (29) и RENI (23) тот же угол объявляли мёртвым. Молчаливое исчезновение
	// уровня при следующем копировании соседнего каталога обязано валить сборку.
	hasDeep := false
	for _, v := range entry["RSILower"] {
		if v == 10 {
			hasDeep = true
		}
	}
	if !hasDeep {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на T это живой угол (89 кроссов RSI(7)@10 за 36 месяцев)", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат, и на T это замерено прямо: RSI(3) уходит
	// под 50 в 3197 будних барах из 25 031 — каждый восьмой бар. Прежняя редакция этого файла
	// свипувала 3 и возвращаться к ней нельзя.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат (RSI(3) уходит под 50 в каждом восьмом будним баре T)", v)
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

	trend := tGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 25 031 будних в
	// кэше, окно прогрева занимает 1.7% истории. Шире двигать нечего — прогрев растёт быстрее
	// пользы от сглаживания.
	for _, v := range trend["EMASlow"] {
		if v > 200 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: окно прогрева растёт быстрее, чем польза от сглаживания", v)
		}
	}
	// Быстрая EMA обязана быть быстрее самой быстрой из медленных, иначе строка сетки означает
	// фильтр, который никогда не пропускает вход.
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

	exit := tGrid(t, "cal_exit.json")
	// Единственное однотемное измерение полосы выхода: cal_entry.json свипует RSIUpper только в
	// связке с нижней границей. Ось доведена до 85 — восходящих кроссов через него у RSI(5) 528
	// за 36 месяцев (через 55 — 2202), то есть верхний угол живой, а не декоративный.
	if got := exit["RSIUpper"]; !sameSet(got, 55, 60, 65, 70, 75, 80, 85) {
		t.Errorf("cal_exit.json: RSIUpper = %v, want ровно {55,60,65,70,75,80,85} — пропуск внутри шкалы сузил бы единственное однотемное измерение полосы выхода", got)
	}
}

// TestTRiskGridsPinTheirMeasuredAxes сторожит оси стопа, цели и обоих гейтов. Каждое число ниже
// опирается на замер по кэшу T: медианный дневной ATR 2.66% цены, круг издержек 0.038 ATR,
// медианный дневной размах 0.85 ATR предыдущего дня, оборот 4961 млн ₽ медианой будних дней.
func TestTRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	day := tGrid(t, "cal_day.json")
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1] — цена отключения гейта меряется в cal_screen.json", got)
	}
	// День T раскрывается быстрее, чем у GAZP: порог 0.3 оставляет ветке «день только начался»
	// 18.5% будних баров (у GAZP 13.6%), 0.4 — 28.7%, 0.5 — 39.1%. Точки ниже 0.25 оставили бы
	// ветке меньше десятой части баров и дали бы ложный вывод «ветка не нужна».
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: ветке остаётся меньше 10%% будних баров (порог 0.3 даёт 18.5%%)", v)
		}
	}
	// Порог 0.6 проходят 48.8% будних баров — на этом инструменте он перестаёт быть гейтом, и
	// его контрольная роль принадлежит cal_day_spent.json.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим почти для половины баров, это не гейт", v)
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
	// Глубина отката принадлежит cal_entry.json: тема дня обязана остаться однотемной. Прежняя
	// редакция файла свипувала RSILower, RSIPeriod и RSIUpper вместе с порогами дня (1680
	// прогонов) — так тема мерила четыре вещи сразу и не отвечала ни на один вопрос.
	for _, field := range []string{"RSILower", "RSIPeriod", "RSIUpper"} {
		if got := day[field]; len(got) != 0 {
			t.Errorf("cal_day.json свипует %s=%v: форма отката принадлежит cal_entry.json, тема дня обязана быть однотемной", field, got)
		}
	}

	spent := tGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Левый край 0.6 стоит контрольной строкой «гейт почти выключен» (48.8% баров). Ниже порог
	// не гейтит вовсе и занял бы прогоны впустую.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог не гейтит (0.6 проходят 49%% баров)", v)
		}
	}
	// Правый край 1.75 проходят 4.1% баров. Дальше тема мерила бы уже не порог, а размер
	// выборки: на 2.0 остались бы единицы сделок за фолд.
	for _, v := range spent["SpentDayATR"] {
		if v > 1.75 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: порог проходят меньше 4%% баров, строка меряет размер выборки, а не гейт", v)
		}
	}

	vol := tGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Все четыре точки живые: при базе 5 дней гейт проходят 31.3% баров при 1.2, 22.5% при 1.5,
	// 14.1% при 2.0, 9.6% при 2.5. Выше 2.5 остаётся меньше десятой части баров.
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

	risk := tGrid(t, "cal_risk.json")
	// Круг издержек на T стоит 0.038 дневного ATR — столько же, сколько у GAZP (0.037), потому
	// что инструменты одинаково узкие (2.66% против 2.71% медианного дневного ATR). На стопе
	// 0.3 ATR комиссия съедает 12.7% риска: строка дорогая, но замеренная, и её удаление
	// обязано быть решением, а не побочным эффектом правки.
	hasTightStop := false
	for _, v := range risk["StopDailyATR"] {
		if v == 0.3 {
			hasTightStop = true
		}
	}
	if !hasTightStop {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — строка дорогая (12.7%% риска на издержки), но замеренная", risk["StopDailyATR"])
	}
	// Нижняя граница оси: на 0.2 издержки съели бы 19% риска, на 0.15 — 25%. Это за чертой 17%,
	// по которой строку 0.3 вырезали из domrf/.
	for _, v := range risk["StopDailyATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: издержки съели бы %.0f%% риска (круг 0.038 ATR) против 13%% на разрешённой строке 0.3", v, 0.038/v*100)
		}
	}
	// Верх оси 1.3: медианный день покрывает 0.85 ATR, стоп 1.3 переживает целиком 78.7% дней.
	// Шире — это уже не стоп, а его отсутствие, оплаченное размером позиции.
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: шире 1.3 ATR стоп перестаёт быть стопом (медианный день 0.85 ATR, стоп 1.3 переживает 79%% дней)", v)
		}
	}
	// Цель действующего боевого литерала обязана оставаться в оси: без неё тема не сравнит
	// новую конфигурацию с той, что торгует в проде.
	hasLiveTarget := false
	for _, v := range risk["TPDailyATR"] {
		if v == 1.5 {
			hasLiveTarget = true
		}
	}
	if !hasLiveTarget {
		t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит боевую цель 1.5 — тему нельзя будет сравнить с торгующей конфигурацией", risk["TPDailyATR"])
	}

	trail := tGrid(t, "cal_trail.json")
	if got := trail["UseTrail"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_trail.json: UseTrail = %v, want ровно [1] — тема меряет форму трейла, а не факт включения", got)
	}
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — трейл и RSI-выход конкурируют за одну сделку, и посторонняя точка не даёт замерить оба режима", got)
	}
	// Контрольная строка 0 обязана быть в оси: desiredStop охраняет трейл условием
	// TrailDailyATR > 0, поэтому такая строка И ЕСТЬ конфигурация с фиксированным стопом и даёт
	// теме собственный baseline без дублирующих комбинаций.
	hasControl := false
	for _, v := range trail["TrailDailyATR"] {
		if v == 0 {
			hasControl = true
		}
		// Шире медианного дневного размаха (0.85 ATR) трейл не догоняет цену.
		if v > 0.9 {
			t.Errorf("cal_trail.json свипует TrailDailyATR=%v: шире медианного дневного размаха (0.85 ATR) трейл не догоняет цену", v)
		}
	}
	if !hasControl {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит контрольную строку 0 (фиксированный стоп) — теме не с чем сравнивать трейл", trail["TrailDailyATR"])
	}
}
