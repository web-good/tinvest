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
	// Глубже 25 порог перестаёт отбирать откат: RSI(4) уходит под 30 1986 раз по будням за
	// 36 месяцев — это обычный шум, а не сетап.
	for _, v := range entry["RSILower"] {
		if v > 25 {
			t.Errorf("cal_entry.json свипует RSILower=%v: выше 25 порог перестаёт отбирать откат (1986 будних кроссов под 30)", v)
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
