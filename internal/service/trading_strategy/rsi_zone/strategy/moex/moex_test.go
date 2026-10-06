package moex

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Литерал прибит снимком: правка любого числа меняет то, что гоняет бэктест MOEX. Замеры,
// обосновывающие каждое поле, — в доке пакета и в _comment гридов data/params/rsi_zone/moex/.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:      5,
		RSILower:       25,
		RSIUpper:       75,
		EMAPeriod:      200,
		DailyATRPeriod: 5,
		StopDailyATR:   2.0,
		UseStoch:       1,
		StochKPeriod:   14,
		StochDSmooth:   3,
		StochLower:     15,
		ZoneWindowBars: 1,
		StuckExitBars:  0,
		ProfitExitBars: 0,
		ProfitExitPct:  0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал MOEX изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Откалиброванный тикер не должен отслеживать baseline: иначе смена дефолтов ядра молча
// поменяет и его.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("MOEX вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Гейт стохастика прошёл §4.7 на точке (WF 1.469 против 1.399 без гейта): без него литерал
// молча вернётся к другой выборке входов.
func TestStochGateIsOn(t *testing.T) {
	if p := DefaultParams(); p.UseStoch == 0 {
		t.Fatal("UseStoch = 0 — гейт стохастика калибровки 2026-10-06 потерян")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR отключает стоп в ядре; на литерале стоп 2.0·ATR 5 срабатывает 1 раз
// из 88 сделок.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
