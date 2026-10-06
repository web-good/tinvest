package afks

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест AFKS. Замеры,
// обосновывающие каждое поле, — в доке пакета и в _comment гридов data/params/rsi_zone/afks/.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:        4,
		RSILower:         25,
		RSIUpper:         75,
		EMAPeriod:        50,
		DailyATRPeriod:   14,
		StopDailyATR:     1.0,
		UseStoch:         1,
		StochKPeriod:     14,
		StochDSmooth:     3,
		StochLower:       20,
		ZoneWindowBars:   1,
		ProfitExitBars:   0,
		ProfitExitPct:    0,
		EntryBlockFrom:   1000,
		EntryBlockTo:     1400,
		EntryBlockMonThu: 1,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал AFKS изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Откалиброванный тикер не должен отслеживать baseline: иначе смена дефолтов ядра молча
// поменяет и его.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("AFKS вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Перекалибровка 2026-10-03 включила гейт стохастика: литерал без него — это прежняя точка
// 2026-09-28, и гейт потерян.
func TestStochGateIsOn(t *testing.T) {
	if p := DefaultParams(); p.UseStoch == 0 {
		t.Fatal("UseStoch = 0 — гейт стохастика перекалибровки 2026-10-03 потерян")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR отключает стоп в ядре.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
