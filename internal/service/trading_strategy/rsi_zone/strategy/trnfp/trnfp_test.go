package trnfp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест TRNFP. Замеры,
// обосновывающие каждое поле, — в доке пакета и в _comment гридов data/params/rsi_zone/trnfp/.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:        4,
		RSILower:         25,
		RSIUpper:         75,
		EMAPeriod:        200,
		DailyATRPeriod:   14,
		StopDailyATR:     1.0,
		UseStoch:         1,
		StochKPeriod:     5,
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
		t.Fatalf("откалиброванный литерал TRNFP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Откалиброванный тикер не должен отслеживать baseline: иначе смена дефолтов ядра молча
// поменяет и его.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("TRNFP вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Без гейта стохастика точка TRNFP теряет сигнал: WF 1.228 против 1.950.
func TestStochGateIsOn(t *testing.T) {
	if p := DefaultParams(); p.UseStoch == 0 {
		t.Fatal("UseStoch = 0 — гейт стохастика калибровки 2026-10-05 потерян")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR отключает стоп в ядре.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
