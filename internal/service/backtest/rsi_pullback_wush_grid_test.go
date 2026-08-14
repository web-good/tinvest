package backtest

import (
	"math"
	"testing"
)

// wushGrid читает файл сеток WUSH через общий хелпер.
func wushGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "wush", file)
}

// TestWUSHSignalGridsPinTheirMeasuredAxes сторожит оси, обоснованные замерами инструмента, а не
// вкусом. Каталог wush/ заводится копированием структуры fesh/, и типовая ошибка такой копии —
// притащить вместе с формой чужие обоснования. По ширине WUSH стоит рядом с FESH (медианный
// дневной ATR 4.25% против 4.42%), поэтому сужения DOMRF сюда не переносятся; оговорку RENI про
// мёртвые углы переносить тоже не нужно — слабейший угол RSI(7)@10 даёт 60 будних кроссов против
// 23 у RENI и 49 у FESH.
func TestWUSHSignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := wushGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; !sameSet(got, 0, 1) {
			t.Errorf("cal_screen.json: %s = %v, want ровно {0,1} — тема меряет цену каждого гейта в сделках, а любая другая пара не даёт сравнить гейт с его отсутствием", field, got)
		}
	}

	entry := wushGrid(t, "cal_entry.json")
	// Глубже 25 порог перестаёт отбирать откат: RSI(4) уходит под 30 1929 раз по будням за
	// 36 месяцев — это обычный шум, а не сетап.
	for _, v := range entry["RSILower"] {
		if v > 25 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 25 порог перестаёт отбирать откат (1929 будних кроссов под 30)", v)
		}
	}
	// Ниже 10 выборка истончается быстрее, чем растёт качество сигнала: у RSI(7) на уровне 10
	// остаётся 60 будних кроссов за всю историю, и более глубокий порог режет и их.
	for _, v := range entry["RSILower"] {
		if v < 10 {
			t.Errorf("cal_entry.json свипует RSILower=%v: ниже 10 сигналов почти не остаётся (у RSI(7)@10 их 60)", v)
		}
	}
	// Уровень 10 обязан остаться: скринер выбрал для WUSH лучшей конфигурацией RSI 6/10, и
	// 118 будних кроссов RSI(6)@10 эту точку выдерживают. На DOMRF таких кроссов было 18, и там
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

	trend := wushGrid(t, "cal_trend.json")
	// EMASlow=200 исполнима: Lookback при ней равен 2*200+20 = 420 баров против 23901 будних
	// в кэше, то есть окно прогрева занимает 1.8% истории.
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

	exit := wushGrid(t, "cal_exit.json")
	// Это единственное место, где меряется полоса выхода: cal_entry.json её намеренно не свипует.
	if got := exit["RSIUpper"]; !sameSet(got, 55, 60, 65, 70, 75, 80) {
		t.Errorf("cal_exit.json: RSIUpper = %v, want ровно {55,60,65,70,75,80} — cal_entry.json полосу выхода не свипует, а любая точка вне шкалы RSI или пропуск внутри неё сужает единственное место, где эта полоса измеряется", got)
	}
}
