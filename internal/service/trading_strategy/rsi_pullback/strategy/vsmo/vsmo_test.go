package vsmo

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsAreTheAcceptedPoint пинит принятую точку целиком. Снимок стоит здесь, а не в
// комментарии, потому что любое поле, изменённое мимо калибровки, обязано валить тест: параметры
// этого пакета уходят в боевую вселенную живого раннера.
func TestParamsAreTheAcceptedPoint(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        70,
		EMAFast:         5,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      0.3,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки VSMO разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestPointDiffersFromTheCoreBaseline сторожит ровно два поля, которыми принятая точка отличается
// от дефолтов ядра: быструю EMA (тема trend_low, три фолда из четырёх, при pooled OOS 2.561 против
// 1.684 у канонической темы тренда) и цель (тема risk, три фолда из четырёх за 0.3).
//
// Шестнадцать остальных полей упали на дефолты ядра не по недосмотру, а потому, что ни одна тема не
// дала за них большинства: вход разошёлся 10/15/15/30, тренд по медленной оси — четырьмя разными
// значениями, стоп — ничьёй 1.5/1.5/1.3/1.3, выход — четырьмя разными значениями. Если литерал
// когда-нибудь схлопнется обратно в baseline целиком, это будет означать потерю калибровки, а не
// упрощение: цель 0.3 вместо 0.6 снижает просадку с 7.24% до 4.92% на полном окне.
func TestPointDiffersFromTheCoreBaseline(t *testing.T) {
	got, base := DefaultParams(), core.DefaultParams()
	if got == base {
		t.Fatal("литерал VSMO совпал с baseline ядра: калибровка потеряна")
	}
	if got.EMAFast != 5 || base.EMAFast != 10 {
		t.Fatalf("быстрая EMA: got.EMAFast = %d, base.EMAFast = %d, want 5 и 10", got.EMAFast, base.EMAFast)
	}
	if got.TPDailyATR != 0.3 || base.TPDailyATR != 0.6 {
		t.Fatalf("цель: got.TPDailyATR = %v, base.TPDailyATR = %v, want 0.3 и 0.6", got.TPDailyATR, base.TPDailyATR)
	}
}

func TestTickerIsVSMO(t *testing.T) {
	if Ticker != "VSMO" {
		t.Fatalf("Ticker = %q, want VSMO", Ticker)
	}
}
