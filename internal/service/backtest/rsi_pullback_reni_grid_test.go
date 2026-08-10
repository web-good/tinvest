package backtest

import "testing"

// reniGrid читает файл сеток RENI через общий хелпер.
func reniGrid(t *testing.T, file string) map[string][]float64 {
	t.Helper()
	return rsiPullbackTickerGrid(t, "reni", file)
}

// TestRENISignalGridsPinTheirMeasuredAxes сторожит оси, обоснованные замерами инструмента, а не
// вкусом. Каталог reni/ заводится копированием структуры ugld/, и типовая ошибка такой копии —
// притащить чужие оси целиком. По ширине RENI действительно сосед UGLD (дневной ATR 3.36% против
// 4.28%), поэтому здесь опасна ОБРАТНАЯ ошибка: перенести сужения, сделанные для DOMRF, где
// ATR 1.94% и сигналов дефицит.
func TestRENISignalGridsPinTheirMeasuredAxes(t *testing.T) {
	screen := reniGrid(t, "cal_screen.json")
	for _, field := range []string{"UseDayATRGate", "UseVolume"} {
		if got := screen[field]; len(got) != 2 {
			t.Errorf("cal_screen.json: %s = %v, want обе точки [0,1] — тема меряет цену каждого гейта в сделках", field, got)
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
	// фильтр, который никогда не пропускает вход.
	for _, v := range trend["EMAFast"] {
		if v >= 50 {
			t.Errorf("cal_trend.json свипует EMAFast=%v: минимум оси EMASlow равен 50, такая пара мертва", v)
		}
	}
}
