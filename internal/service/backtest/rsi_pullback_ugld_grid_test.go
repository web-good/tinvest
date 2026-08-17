package backtest

import (
	"math"
	"testing"
)

// ugldGrid читает файл сеток UGLD через общий хелпер.
func ugldGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "ugld", file)
}

// TestUGLDSignalGridsPinTheirMeasuredAxes сторожит оси сигнальных тем. Каталог ugld/ переписан
// 2026-08-17 в расширенную редакцию (вход 252 комбинации против 80, тренд 35 против 16, дневной
// гейт 24 против 60 с посторонней осью RSILower, объём 8 против 20) с пересадкой каждой оси на
// замеры самого UGLD: 29 230 30-минутных баров, из них 21 693 будних, 32.8 месяца с 2023-11-22
// (полных 36 у тикера нет — он вышел на биржу в ноябре 2023). Типовая ошибка такой переписки —
// притащить вместе с формой чужие обоснования: у UGLD дневной ATR 3.83% цены против 3.07% у
// NVTK, кроссов RSI(6)@15 — 197 будних, а трендовая ось почти вырождена (45.6–47.7% баров на
// всех двадцати парах EMA), чего нет ни у одного другого тикера каталога.
func TestUGLDSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := ugldGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках, а любая другая пара не даёт сравнить гейт с его отсутствием", field, got)
		}
	}

	entry := ugldGrid(t, "cal_entry.json")
	// Выше 50 отката нет по определению: 50 — средняя линия осциллятора.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 отката нет по определению (50 — средняя линия RSI)", v)
		}
	}
	// Принятый угол входа обязан остаться в оси, и на UGLD это требование жёстче, чем у соседей:
	// вход RSI(6)@15 — не точка на полке, а ПИК (RSILower 10 даёт pooled 0.343, 20 — 1.580,
	// RSIPeriod 5 — 1.765, 7 — 1.069 при прочих принятых полях). Без обеих его координат в сетке
	// следующий прогон не увидит ни самого угла, ни того, что вокруг него обрыв.
	for field, want := range map[string]float64{"RSILower": 15, "RSIPeriod": 6} {
		found := false
		for _, v := range entry[field] {
			if v == want {
				found = true
			}
		}
		if !found {
			t.Errorf("cal_entry.json: %s = %v, не содержит принятое значение %.0f — вокруг него обрыв, и без него тема не воспроизводима", field, entry[field], want)
		}
	}
	// Угол RSI(7)@10 даёт 42 будних кросса за 32.8 месяца — уровень LENT (29), а не RENI (23),
	// где угол объявляли мёртвым. Строка живая и обязана остаться: она единственная проверка
	// того, что глубже принятого уровня становится хуже.
	hasDeep := false
	for _, v := range entry["RSILower"] {
		if v == 10 {
			hasDeep = true
		}
	}
	if !hasDeep {
		t.Errorf("cal_entry.json: RSILower = %v, не содержит 10 — на UGLD это живой угол (42 кросса RSI(7)@10) и единственная проверка того, что глубокий откат хуже", entry["RSILower"])
	}
	// RSIPeriod короче 4 покупает шум, а не откат.
	for _, v := range entry["RSIPeriod"] {
		if v < 4 {
			t.Errorf("cal_entry.json свипует RSIPeriod=%v: короче 4 это шум, а не откат", v)
		}
	}
	// Вырожденная полоса запрещена: вход требует креста вниз через RSILower, выход — креста вверх
	// через RSIUpper. Если верхняя граница не выше нижней, строка означает позицию, которую
	// закрывает любой отскок, а в лидерборде она выглядит обычной строкой с высоким win rate.
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

	trend := ugldGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 21 693 будних в
	// кэше. Шире двигать нечего — прогрев растёт быстрее пользы от сглаживания, а на UGLD ещё и
	// без пользы: замер даёт 47.7% допущенных баров на 200 против 46.0% на 50.
	for _, v := range trend["EMASlow"] {
		if v > 200 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: окно прогрева растёт быстрее, чем польза от сглаживания", v)
		}
	}
	// Быстрая EMA обязана быть быстрее самой быстрой из медленных, иначе строка означает фильтр,
	// который никогда не пропускает вход.
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
	// Принятая пара 20·150 обязана остаться в оси: полка 100–150 плоская (3.495 / 3.485 / 3.627),
	// и сравнивать новую конфигурацию не с чем, если действующей в сетке нет.
	for field, want := range map[string]float64{"EMAFast": 20, "EMASlow": 150} {
		found := false
		for _, v := range trend[field] {
			if v == want {
				found = true
			}
		}
		if !found {
			t.Errorf("cal_trend.json: %s = %v, не содержит принятое значение %.0f", field, trend[field], want)
		}
	}

	exit := ugldGrid(t, "cal_exit.json")
	// Единственное место, где полоса выхода меряется однотемно: cal_entry.json свипует RSIUpper
	// только в связке с нижней границей. Левый край 55 — принятое значение и одновременно
	// граница смысла: ниже средней линии RSI выход превращает многодневную стратегию в скальп
	// (на 45 доля сделок длиннее дня падает с 29% до 14%).
	if got := exit["RSIUpper"]; !sameSet(got, 55, 60, 65, 70, 75, 80) {
		t.Errorf("cal_exit.json: RSIUpper = %v, want ровно {55,60,65,70,75,80} — пропуск внутри шкалы сузил бы единственное однотемное измерение полосы выхода", got)
	}
	for _, v := range exit["RSIUpper"] {
		if v < 50 {
			t.Errorf("cal_exit.json свипует RSIUpper=%v: ниже средней линии RSI выход закрывает позицию, пока осциллятор в нижней половине — это скальп, а не откат в тренде", v)
		}
	}
}

// TestUGLDRiskGridsPinTheirMeasuredAxes сторожит оси стопа, цели, трейла и обоих гейтов. Каждое
// число ниже опирается на замер по кэшу UGLD: медианный дневной ATR 3.83% цены, круг издержек
// около 0.03 ATR (комиссия 0.0005 на сторону плюс шаг цены 0.0001 ₽ при цене около 0.66 ₽),
// медианный дневной размах 0.79 ATR предыдущего дня.
func TestUGLDRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	day := ugldGrid(t, "cal_day.json")
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1] — цена отключения гейта меряется в cal_screen.json", got)
	}
	// День UGLD раскрывается быстро: медиана доли ATR ко второму бару 0.33. Порог 0.3 оставляет
	// ветке «день только начался» 12.3% будних баров, 0.4 — 21.6%, 0.5 — 32.5%. Точки ниже 0.25
	// оставили бы ветке меньше десятой части баров и дали бы ложный вывод «ветка не нужна» — а на
	// UGLD этот вывод как раз верен (её выключение поднимает pooled PF с 3.535 до 3.627) и обязан
	// оставаться выводом ЗАМЕРА, а не следствием неизмеримой оси.
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: ветке остаётся меньше 10%% будних баров (порог 0.3 даёт 12.3%%)", v)
		}
	}
	// Ноль обязан остаться: это выключатель ранней ветки и принятое значение литерала.
	hasFreshOff := false
	for _, v := range day["FreshDayATR"] {
		if v == 0 {
			hasFreshOff = true
		}
	}
	if !hasFreshOff {
		t.Errorf("cal_day.json: FreshDayATR = %v, не содержит 0 — без выключателя ранней ветки принятая конфигурация не воспроизводима", day["FreshDayATR"])
	}
	// Порог 0.6 достигают 66.5% дней — на этом инструменте он перестаёт быть гейтом, и его
	// контрольная роль принадлежит cal_day_spent.json.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для двух третей дней, это не гейт", v)
		}
	}
	// Соотношение двух веток: положительный максимум FreshDayATR обязан быть строго меньше
	// минимума SpentDayATR, иначе dayStateOK даёт true почти на каждом баре и гейт формально
	// включён, фактически не отсекая ничего. Цена ошибки на UGLD: с выключенным дневным гейтом
	// принятая точка даёт pooled PF 1.353 на 40 сделках вместо 3.627 на 23.
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
	// Глубина отката принадлежит cal_entry.json. В прежней редакции файла она свипевалась здесь
	// (60 прогонов вместо 24) и размывала тему: победитель дневного гейта держался на комбинации
	// с глубиной входа, а не на пороге дня.
	if got := day["RSILower"]; len(got) != 0 {
		t.Errorf("cal_day.json свипует RSILower=%v: глубина отката принадлежит cal_entry.json", got)
	}
	// Принятый порог обязан оставаться в оси.
	hasLiveSpent := false
	for _, v := range day["SpentDayATR"] {
		if v == 0.8 {
			hasLiveSpent = true
		}
	}
	if !hasLiveSpent {
		t.Errorf("cal_day.json: SpentDayATR = %v, не содержит принятый порог 0.8", day["SpentDayATR"])
	}

	spent := ugldGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Левый край 0.6 стоит контрольной строкой «гейт почти выключен» (66.5% дней). Ниже порог не
	// гейтит вовсе и занял бы прогоны впустую.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог не гейтит (0.6 достигают 66%% дней)", v)
		}
	}

	vol := ugldGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Все четыре точки живые: при базе 5 дней гейт проходят 30.4% баров при 1.2, 24.0% при 1.5,
	// 17.0% при 2.0, 12.8% при 2.5. Выше 2.5 остаётся десятая часть баров, и прежняя редакция
	// каталога держала там точку 3.0 (10.2%) — вместе с дневным гейтом это оставляло тему на
	// десятке сделок за фолд.
	for _, v := range vol["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.5 гейт проходит около 10%% баров, а вместе с дневным гейтом тема остаётся без выборки", v)
		}
	}
	for _, v := range vol["VolBaseDays"] {
		if v != 5 && v != 10 {
			t.Errorf("cal_volume.json свипует VolBaseDays=%v: обоснованы только 5 и 10 (3 ловит одиночный всплеск — медиана отношения 0.74 против 0.68, 14 размывает и тянет Lookback до 1008 баров)", v)
		}
	}

	risk := ugldGrid(t, "cal_risk.json")
	// Круг издержек на UGLD стоит около 0.03 дневного ATR — самый дешёвый в каталоге, потому что
	// инструмент широкий (медианный дневной ATR 3.83% цены). На стопе 0.3 ATR издержки съедают 9%
	// риска, внутри черты 17%, по которой строку 0.3 вырезали из domrf/.
	hasTightStop := false
	for _, v := range risk["StopDailyATR"] {
		if v == 0.3 {
			hasTightStop = true
		}
	}
	if !hasTightStop {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 — на UGLD это единственная точка оси стопа, которая вообще что-то меняет при включённом трейле", risk["StopDailyATR"])
	}
	for _, v := range risk["StopDailyATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: издержки съели бы %.0f%% риска (круг 0.03 ATR) против 9%% на разрешённой строке 0.3", v, 0.03/v*100)
		}
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: шире 1.3 ATR стоп перестаёт быть стопом (медианный день 0.79 ATR, порога 1.3 достигают 22%% дней)", v)
		}
	}
	// Обе цели, вокруг которых спорят, обязаны остаться в оси: 1.0 — цель прежнего литерала,
	// 1.5 — принятая. Без первой новую конфигурацию не с чем сравнить, без второй тема не увидит
	// действующую.
	for _, want := range []float64{1.0, 1.5} {
		found := false
		for _, v := range risk["TPDailyATR"] {
			if v == want {
				found = true
			}
		}
		if !found {
			t.Errorf("cal_risk.json: TPDailyATR = %v, не содержит %.1f", risk["TPDailyATR"], want)
		}
	}

	trail := ugldGrid(t, "cal_trail.json")
	if got := trail["UseTrail"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_trail.json: UseTrail = %v, want ровно [1] — тема меряет форму трейла, а не факт включения", got)
	}
	if got := trail["UseRSIExit"]; !sameSet(got, 0, 1) {
		t.Errorf("cal_trail.json: UseRSIExit = %v, want ровно {0,1} — трейл и RSI-выход конкурируют за одну сделку, и посторонняя точка не даёт замерить оба режима", got)
	}
	// Принятое значение 0.5 обязано остаться: на UGLD именно трейл, а не поле стопа, связывает
	// риск (точки стопа 0.5/0.7/1.0/1.3 дают побайтово одинаковый результат), поэтому эта ось —
	// главная ось риска тикера, а не украшение.
	hasLiveTrail := false
	for _, v := range trail["TrailDailyATR"] {
		if v == 0.5 {
			hasLiveTrail = true
		}
		if v > 0.9 {
			t.Errorf("cal_trail.json свипует TrailDailyATR=%v: шире медианного дневного размаха (0.79 ATR) трейл не догоняет цену", v)
		}
	}
	if !hasLiveTrail {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит принятое значение 0.5 — это главная ось риска тикера", trail["TrailDailyATR"])
	}
	// Контрольная строка «трейла нет» обязана остаться: при UseTrail=1 и TrailDailyATR=0
	// desiredStop не трогает трейл вовсе, и строка воспроизводит фиксированный стоп. Без неё
	// фаза лишается собственной базы и сравнивать формы трейла становится не с чем.
	hasTrailControl := false
	for _, v := range trail["TrailDailyATR"] {
		if v == 0 {
			hasTrailControl = true
		}
	}
	if !hasTrailControl {
		t.Errorf("cal_trail.json: TrailDailyATR = %v, не содержит контрольную строку 0 (конфигурация с фиксированным стопом)", trail["TrailDailyATR"])
	}
}
