package backtest

import (
	"math"
	"testing"
)

// feshGrid читает файл сеток FESH через общий хелпер.
func feshGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "fesh", file)
}

// TestFESHSignalGridsPinTheirMeasuredAxes сторожит оси, обоснованные замерами инструмента, а не
// вкусом. Каталог fesh/ заводится копированием структуры reni/, и типовая ошибка такой копии —
// притащить вместе с формой чужие обоснования. FESH шире всех заведённых тикеров (дневной ATR
// 4.42% против 3.36% у RENI и 1.94% у DOMRF), и опасны обе стороны: и перенос сужений DOMRF,
// сделанных при дефиците сигналов, и перенос оговорок RENI про мёртвые углы — здесь их нет,
// слабейший угол RSI(7)@10 даёт 49 будних кроссов против 23 у RENI.
func TestFESHSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := feshGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках, а любая другая пара не даёт сравнить гейт с его отсутствием", field, got)
		}
	}

	entry := feshGrid(t, "cal_entry.json")
	// Ось расширена владельцем 2026-08-14 до полной шкалы [10 ... 50] с шагом 5 — прежний
	// потолок 25 снят сознательно. Верхние строки исполнимы, но описывают не откат, а дыхание
	// цены: RSI(4) уходит под 30 2000 раз по будням за 36 месяцев, под 45 — 2980. Замер против
	// расширения есть, и он четвёртый подряд: тема entry упала с 1.029 (ось до 25) до 0.895 (ось
	// до 50), а выбор RSILower разъехался на 25/35/45/50. Потолок 50 — край шкалы RSI, за
	// которым порог входа перестаёт быть порогом отката в принципе.
	for _, v := range entry["RSILower"] {
		if v > 50 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 50 «откат» перестаёт быть откатом — это уже верхняя половина шкалы RSI", v)
		}
	}
	// Шаг оси обязан остаться пятёркой: владелец задал именно шкалу с шагом 5, и сгущение
	// узлов на выборке в 60-70 сделок покупает только степени свободы.
	for _, v := range entry["RSILower"] {
		if v != float64(int(v)) || int(v)%5 != 0 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ось задана шкалой с шагом 5", v)
		}
	}
	// Ниже 10 выборка истончается быстрее, чем растёт качество сигнала: у RSI(7) на уровне 10
	// остаётся 49 будних кроссов за всю историю, и более глубокий порог режет и их.
	for _, v := range entry["RSILower"] {
		if v < 10 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 10 сигналов почти не остаётся (у RSI(7)@10 их 49)", v)
		}
	}
	// Уровень 10 обязан остаться: скринер выбрал для FESH лучшей конфигурацией RSI 6/10, и
	// 81 будний кросс RSI(6)@10 эту точку выдерживает. На DOMRF таких кроссов было 18, и там
	// уровень 10 вырезали — при копировании сеток это сужение легко притащить по ошибке.
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
	// RSIUpper здесь не свипуется: 4x4x6 = 96 комбинаций на одной теме — переобучение по
	// построению. Полоса выхода меряется отдельно, файлом cal_exit.json.
	if got := entry["RSIUpper"]; len(got) != 0 {
		t.Errorf("cal_entry.json свипует RSIUpper=%v: полоса выхода принадлежит cal_exit.json", got)
	}

	trend := feshGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 25321 будних
	// в кэше, то есть окно прогрева занимает 1.7% истории.
	for _, v := range trend["EMASlow"] {
		if v > 200 {
			t.Errorf("cal_trend.json свипует EMASlow=%v: окно прогрева растёт быстрее, чем польза от сглаживания", v)
		}
	}
	// Быстрая EMA обязана быть быстрее самой быстрой из медленных, иначе строка сетки означает
	// фильтр, который никогда не пропускает вход. Порог берётся из фактического минимума оси
	// EMASlow, а не зашивается константой: понижение оси EMASlow иначе тихо перестало бы
	// совпадать с тем, что здесь проверяется.
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

// TestFESHRiskGridsPinTheirMeasuredAxes сторожит оси стопа, цели и обоих гейтов. Каждое число
// ниже опирается на замер по кэшу FESH, а не на перенос с соседнего тикера: дневной ATR 4.42%,
// круг издержек 0.023 ATR, медианный дневной размах 0.85 ATR.
func TestFESHRiskGridsPinTheirMeasuredAxes(t *testing.T) {
	day := feshGrid(t, "cal_day.json")
	if got := day["UseDayATRGate"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_day.json: UseDayATRGate = %v, want ровно [1] — цена отключения гейта меряется в cal_screen.json", got)
	}
	// Медианный день проходит 0.27 ATR уже ко второму бару, 0.32 к третьему. Пороги 0.1-0.2 из
	// ugld/ оставляют ветке «день только начался» 2.9-6.3% будних баров: она почти мертва.
	for _, v := range day["FreshDayATR"] {
		if v > 0 && v < 0.25 {
			t.Errorf("cal_day.json свипует FreshDayATR=%v: ко второму бару медианный день прошёл 0.27 ATR, ветке остаётся меньше 7%% баров", v)
		}
	}
	// Порог 0.6 проходят 54.3% будних баров — на этом инструменте он перестаёт быть гейтом.
	for _, v := range day["SpentDayATR"] {
		if v < 0.8 {
			t.Errorf("cal_day.json свипует SpentDayATR=%v: порог достижим для 54%% баров, это не гейт", v)
		}
	}
	// Соотношение двух веток гейта: положительный максимум FreshDayATR обязан быть строго
	// меньше минимума SpentDayATR. dayStateOK пропускает бар, когда день ещё не раскрылся
	// (used <= fresh*ATR) ИЛИ когда он уже исчерпан (used >= spent*ATR); если верх ветки
	// «свежий» дотягивается до низа ветки «исчерпан», обе полосы дают true почти на каждом
	// баре, и UseDayATRGate=1 в лидерборде продолжит формально числиться включённым, хотя
	// фактически не отсекает ничего.
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

	spent := feshGrid(t, "cal_day_spent.json")
	if got := spent["FreshDayATR"]; len(got) != 1 || got[0] != 0 {
		t.Errorf("cal_day_spent.json: FreshDayATR = %v, want ровно [0] — ветка «день начался» выключена целиком", got)
	}
	// Левый край 0.6 стоит контрольной строкой «гейт почти выключен» (54.3% баров). Точки
	// 0.4-0.5 из ugld/ на FESH не гейтят вовсе и заняли бы прогоны впустую.
	for _, v := range spent["SpentDayATR"] {
		if v < 0.6 {
			t.Errorf("cal_day_spent.json свипует SpentDayATR=%v: ниже 0.6 порог не гейтит (0.6 проходят 54%% баров)", v)
		}
	}

	vol := feshGrid(t, "cal_volume.json")
	if got := vol["UseVolume"]; len(got) != 1 || got[0] != 1 {
		t.Errorf("cal_volume.json: UseVolume = %v, want ровно [1] — точка «гейт выключен» принадлежит cal_screen.json", got)
	}
	// Все четыре точки живые: при базе 5 дней гейт проходят 29.9% баров при 1.2, 24.2% при 1.5,
	// 17.9% при 2.0, 14.0% при 2.5. Выше 2.5 остаётся меньше седьмой части баров, и объёмный
	// гейт начинает резать выборку сильнее, чем несёт информации.
	for _, v := range vol["VolMult"] {
		if v > 2.5 {
			t.Errorf("cal_volume.json свипует VolMult=%v: выше 2.5 гейт проходит меньше 14%% баров", v)
		}
	}
	// Только две базы, 5 и 10: короткая быстрее реагирует на смену активности, длинная
	// устойчивее к одиночному всплеску. База 3 из ugld/ ловит один выброс объёма, база 14 —
	// размывает; на вторичном гейте лишние степени свободы не окупаются.
	for _, v := range vol["VolBaseDays"] {
		if v != 5 && v != 10 {
			t.Errorf("cal_volume.json свипует VolBaseDays=%v: обоснованы только 5 и 10 (3 ловит одиночный всплеск, 14 размывает)", v)
		}
	}

	risk := feshGrid(t, "cal_risk.json")
	// Круг издержек стоит 0.023 дневного ATR — самый дешёвый из четырёх заведённых тикеров: на
	// стопе 0.3 ATR (= 1.33% цены) комиссия съедает 8% риска. На DOMRF та же строка стоила 17%
	// и была оттуда вырезана — при копировании сеток это сужение легко притащить по ошибке,
	// поэтому присутствие строки проверяется явно.
	hasTightStop := false
	for _, v := range risk["StopDailyATR"] {
		if v == 0.3 {
			hasTightStop = true
		}
	}
	if !hasTightStop {
		t.Errorf("cal_risk.json: StopDailyATR = %v, не содержит 0.3 (издержки 0.023 ATR за круг эту строку лицензируют)", risk["StopDailyATR"])
	}
	// Нижняя граница оси: тот же круг издержек, который лицензирует строку 0.3 (8% риска),
	// запрещает идти уже. На 0.15 доля выросла бы до 15%, на 0.1 — до 23%: это уже та черта,
	// по которой DOMRF отверг свою строку 0.3 при 17%. «Попробуем стоп потуже» не должно
	// суметь добавить такую строку молча.
	for _, v := range risk["StopDailyATR"] {
		if v > 0 && v < 0.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: издержки съели бы %.0f%% риска (круг 0.023 ATR) против 8%% на разрешённой строке 0.3; для сравнения, DOMRF отверг свою строку 0.3 при 17%%", v, 0.023/v*100)
		}
	}
	// Верх оси 1.3: медианный день покрывает 0.85 ATR, такой стоп переживает целиком 79.7%
	// дней. Шире — это уже не стоп, а отсутствие стопа, оплаченное размером позиции.
	for _, v := range risk["StopDailyATR"] {
		if v > 1.3 {
			t.Errorf("cal_risk.json свипует StopDailyATR=%v: шире 1.3 ATR стоп перестаёт быть стопом (медианный день 0.85 ATR)", v)
		}
	}

	exit := feshGrid(t, "cal_exit.json")
	// Это единственное место, где меряется полоса выхода: cal_entry.json её намеренно не свипует.
	// Ось расширена владельцем 2026-08-14 до [50 ... 90] с шагом 5. Все девять точек живые по
	// кроссам RSI вверх (будние, 36 мес): RSI(5) даёт 2690 на уровне 50 и 141 на уровне 90.
	// Принятый литерал стоит на 55, то есть в левой половине новой оси — расширение вправо
	// оправдало себя как проверка гипотезы, а не как источник литерала.
	if got := exit["RSIUpper"]; !sameSet(got, 50, 55, 60, 65, 70, 75, 80, 85, 90) {
		t.Errorf("cal_exit.json: RSIUpper = %v, want ровно {50,55,...,90} — cal_entry.json полосу выхода не свипует, а пропуск внутри шкалы сужает единственное место, где эта полоса измеряется", got)
	}

	trail := feshGrid(t, "cal_trail.json")
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
