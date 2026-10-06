package trmk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест TRMK. Замеры,
// обосновывающие каждое поле, — в доке пакета и в _comment гридов data/params/rsi_zone/trmk/.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      4,
		RSILower:       25,
		RSIUpper:       75,
		EMAPeriod:      200,
		DailyATRPeriod: 14,
		StopDailyATR:   1.0,
		UseStoch:       1,
		StochKPeriod:   9,
		StochDSmooth:   3,
		StochLower:     10,
		ZoneWindowBars: 1,
		StuckExitBars:  0,
		ProfitExitBars: 0,
		ProfitExitPct:  0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал TRMK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Откалиброванный тикер не должен отслеживать baseline: иначе смена дефолтов ядра молча
// поменяет и его.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("TRMK вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Гейт стохастика прошёл §4.7 на точке (WF 1.718 против 1.102 без гейта): без него литерал
// молча вернётся к дефолтам ядра, которые проваливают стоп-условие.
func TestStochGateIsOn(t *testing.T) {
	if p := DefaultParams(); p.UseStoch == 0 {
		t.Fatal("UseStoch = 0 — гейт стохастика калибровки 2026-10-06 потерян")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR отключает стоп в ядре; на литерале стоп 1.0·ATR 14 срабатывает 3 раза
// из 48 сделок.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
