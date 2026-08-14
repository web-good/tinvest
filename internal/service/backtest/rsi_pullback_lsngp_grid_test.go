package backtest

import (
	"math"
	"testing"
)

// lsngpGrid читает файл сеток LSNGP через общий хелпер.
func lsngpGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "lsngp", file)
}

// TestLSNGPSignalGridsPinTheirMeasuredAxes сторожит оси сигнальных тем. Каталог lsngp/ заведён
// копированием формы каталога lent/ по прямому распоряжению владельца («сетка такая же широкая»),
// и типовая ошибка такой копии — притащить вместе с формой чужие обоснования. Здесь проверяется
// то, что подтверждено замерами САМОГО LSNGP: инструмент УЖЕ всех соседей (медианный дневной ATR
// 2.64% против 3.16% у LENT и 4.25% у WUSH), а кроссы RSI у него, наоборот, вдвое чаще на
// главном уровне скринера (RSI(4)@10 — 215 будних против 212 у LENT при том же слабейшем угле).
func TestLSNGPSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := lsngpGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках, а любая другая пара не даёт сравнить гейт с его отсутствием", field, got)
		}
	}

	entry := lsngpGrid(t, "cal_entry.json")
	// Потолок 50 — средняя линия осциллятора, выше неё отката нет по определению. Ширина оси
	// задана владельцем по образцу lent/ ДО первого прогона; предупреждение о цене этой ширины
	// (WUSH 2.000 → 1.674, LENT 1.935 → 1.355 при разъехавшемся выборе) живёт в _comment файла и
	// запретом не является — это разные вещи: там ось расширяли В ОТВЕТ на увиденный результат,
	// здесь она широка с самого начала.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению (50 — средняя линия RSI)", v)
		}
	}
	// Ниже 10 выборка истончается быстрее, чем растёт качество сигнала: у RSI(7) на уровне 10
	// остаётся 29 будних кроссов за 36 месяцев, и более глубокий порог режет и их.
	for _, v := range entry["RSILower"] {
		if v < 10 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 10 сигналов почти не остаётся (у RSI(7)@10 их 29 за три года)", v)
		}
	}
	// Уровень 10 обязан остаться: скринер выбрал для LSNGP лучшей конфигурацией RSI 4/10, и
	// 215 будних кроссов RSI(4)@10 выдерживают эту точку с запасом — это самая сильная опора
	// каталога, а не пограничный угол. На DOMRF таких кроссов было 18, и там уровень 10
	// вырезали; при копировании сеток это сужение легко притащить по ошибке.
	hasDeep := false
	for _, v := range entry["RSILower"] {
		if v == 10 {
			hasDeep = true
		}
	}
	if !hasDeep {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — основную гипотезу скринера", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат: RSI(3) реагирует на любое дрожание цены.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
	}
	// Вырожденная полоса запрещена: вход требует креста вниз через RSILower, выход — креста
	// вверх через RSIUpper. Если верхняя граница не выше нижней, строка сетки означает позицию,
	// которую закрывает любой отскок, а в лидерборде такой профиль выглядит обычной строкой с
	// высоким win rate.
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

	trend := lsngpGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 23 045 будних
	// в кэше, то есть окно прогрева занимает 1.8% истории.
	for _, v := range trend["EMASlow"] {
		if v > 200 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: окно прогрева растёт быстрее, чем польза от сглаживания", v)
		}
	}
	// Быстрая EMA обязана быть быстрее самой быстрой из медленных, иначе строка сетки означает
	// фильтр, который никогда не пропускает вход. Порог берётся из фактического минимума оси
	// EMASlow, а не зашивается константой.
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

	exit := lsngpGrid(t, "cal_exit.json")
	// Единственное место, где верхняя граница полосы меряется в чистом виде, без 224 комбинаций
	// входа рядом. Пропуск внутри шкалы сузил бы это измерение, точки вне шкалы RSI бессмысленны.
	if got := exit["RSIUpper"]; !sameSet(got, 55, 60, 65, 70, 75, 80) {
		t.Errorf("cal_exit.json: RSIUpper = %v, want ровно {55,60,65,70,75,80}", got)
	}
}

// TestLSNGPRiskGridsPinTheirMeasuredAxes сторожит оси стопа, цели и обоих гейтов. Каждое число
// ниже опирается на замер по кэшу LSNGP: медианный дневной ATR 2.64% цены, круг издержек
// 0.038 ATR — САМЫЙ дорогой из заведённых тикеров, медианный дневной размах 0.83 ATR
// предыдущего дня.
func TestLSNGPRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	day := lsngpGrid(t, "cal_day.json")
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1] — цена отключения гейта меряется в cal_screen.json", got)
	}
	// День LSNGP раскрывается быстро, вровень с LENT: медиана доли ATR ко второму бару 0.35
	// против 0.28 у FESH. Ветка «день только начался» поэтому узкая: порог 0.25 оставляет ей
	// 7.3% будних баров, 0.3 — 10.4%, 0.4 — 17.6%. Ось [0, 0.25, 0.35] из fesh/ подсунула бы
	// калибровке почти мёртвую ветку и дала ложный вывод «ветка не нужна».
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: ко второму бару медианный день LSNGP прошёл 0.35 ATR, ветке остаётся меньше 8%% баров (порог 0.3 даёт 10.4%%)", v)
		}
	}
	// Порог 0.6 проходят 61.8% будних баров — на этом инструменте он перестаёт быть гейтом.
	// Контрольную роль этой строки играет cal_day_spent.json, где она стоит намеренно.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для 62%% баров, это не гейт", v)
		}
	}
	// Соотношение двух веток гейта: положительный максимум FreshDayATR обязан быть строго
	// меньше минимума SpentDayATR. dayStateOK пропускает бар, когда день ещё не раскрылся
	// (used <= fresh*ATR) ИЛИ когда он уже исчерпан (used >= spent*ATR); если верх «свежей»
	// ветки дотягивается до низа «исчерпанной», обе полосы дают true почти на каждом баре, и
	// UseDayATRGate=1 в лидерборде продолжит числиться включённым, ничего не отсекая.
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
	// RSILower в этой фазе не свипуется: глубина отката принадлежит cal_entry.json.
	if got := day["RSILower"]; len(got) != 0 {
		t.Errorf("cal_day.json свипует RSILower=%v: глубина отката принадлежит cal_entry.json", got)
	}

	spent := lsngpGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Левый край 0.6 стоит контрольной строкой «гейт почти выключен» (61.8% баров). Точки
	// ниже на LSNGP не гейтят вовсе и заняли бы прогоны впустую.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог не гейтит (0.6 проходят 62%% баров)", v)
		}
	}

	vol := lsngpGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Все четыре точки живые: при базе 5 дней гейт проходят 30.5% баров при 1.2, 24.3% при 1.5,
	// 17.7% при 2.0, 13.6% при 2.5. Выше 2.5 остаётся меньше седьмой части баров, и объёмный
	// гейт начинает резать выборку сильнее, чем несёт информации.
	for _, v := range vol["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.5 гейт проходит меньше 14%% баров", v)
		}
	}
	for _, v := range vol["VolBaseDays"] {
		if v != 5 && v != 10 {
			t.Errorf("cal_volume.json свипует VolBaseDays=%v: обоснованы только 5 и 10 (3 ловит одиночный всплеск, 14 размывает)", v)
		}
	}

	risk := lsngpGrid(t, "cal_risk.json")
	// Круг издержек на LSNGP стоит 0.038 дневного ATR — ДОРОЖЕ, чем у любого заведённого тикера
	// (LENT 0.032, RENI 0.030, WUSH 0.024), потому что инструмент самый узкий: медианный дневной
	// ATR 2.64% против 4.25% у WUSH. На стопе 0.3 ATR издержки съедают 12.6% риска — между
	// разрешённой строкой LENT (10.6%) и чертой DOMRF (17%), по которой ту же строку вырезали.
	// Поэтому её присутствие проверяется явно: удаление обязано быть решением, а не побочным
	// эффектом «оптимизации» каталога.
	hasTightStop := false
	for _, v := range risk["StopDailyATR"] {
		if v == 0.3 {
			hasTightStop = true
		}
	}
	if !hasTightStop {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — строка дорогая (12.6%% риска на издержки), но замеренная", risk["StopDailyATR"])
	}
	// Нижняя граница оси: тот же круг издержек, который ещё терпит строку 0.3 (12.6% риска),
	// запрещает идти уже. На 0.25 доля вырастает до 15%, на 0.2 — до 19%: это уже за чертой, по
	// которой DOMRF отверг свою строку 0.3 при 17%.
	for _, v := range risk["StopDailyATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: издержки съели бы %.0f%% риска (круг 0.038 ATR) против 13%% на разрешённой строке 0.3; DOMRF отверг свою строку 0.3 при 17%%", v, 0.038/v*100)
		}
	}
	// Верх оси 1.3: медианный день покрывает 0.83 ATR, такой стоп переживает целиком 77.7%
	// дней. Шире — это уже не стоп, а отсутствие стопа, оплаченное размером позиции.
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: шире 1.3 ATR стоп перестаёт быть стопом (медианный день 0.83 ATR, стоп 1.3 переживает 78%% дней)", v)
		}
	}

	trail := lsngpGrid(t, "cal_trail.json")
	if got := trail["UseTrail"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_trail.json: UseTrail = %v, want ровно [1] — тема меряет форму трейла, а не факт включения", got)
	}
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — трейл и RSI-выход конкурируют за одну сделку, и посторонняя точка не даёт замерить оба режима", got)
	}
	// Правый край 0.8, а не 0.6 как у ugld/: там потолок задавала цель TPDailyATR=0.6, выше
	// которой трейл не успевал взвестись; здесь ось цели поднята до 2.5.
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
