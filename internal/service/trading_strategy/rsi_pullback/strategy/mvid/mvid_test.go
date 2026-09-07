package mvid

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsAreTheAcceptedPoint пинит принятую точку целиком. Снимок стоит здесь, а не в
// комментарии, потому что любое поле, изменённое мимо калибровки, обязано валить тест: параметры
// этого пакета уходят в боевую вселенную живого раннера.
func TestParamsAreTheAcceptedPoint(t *testing.T) {
	want := core.Params{
		RSIPeriod:       3,
		RSILower:        10,
		RSIUpper:        70,
		EMAFast:         3,
		EMASlow:         50,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      0.2,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        1,
		TrailDailyATR:   0.5,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки MVID разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestPointDiffersFromTheCoreBaseline сторожит семь полей, которыми принятая точка отличается от
// дефолтов ядра (RSIPeriod, RSILower, EMAFast, EMASlow, TPDailyATR, UseTrail, TrailDailyATR). Если
// литерал когда-нибудь схлопнется обратно в baseline, это будет означать потерю калибровки, а не
// упрощение.
func TestPointDiffersFromTheCoreBaseline(t *testing.T) {
	got, base := DefaultParams(), core.DefaultParams()
	if got == base {
		t.Fatal("литерал MVID совпал с baseline ядра: калибровка потеряна")
	}
	if got.RSIPeriod != 3 || base.RSIPeriod != 4 {
		t.Fatalf("период RSI: got.RSIPeriod = %d, base.RSIPeriod = %d, want 3 и 4", got.RSIPeriod, base.RSIPeriod)
	}
	if got.RSILower != 10 || base.RSILower != 30 {
		t.Fatalf("нижняя граница RSI: got.RSILower = %v, base.RSILower = %v, want 10 и 30", got.RSILower, base.RSILower)
	}
	if got.EMAFast != 3 || base.EMAFast != 10 {
		t.Fatalf("быстрая EMA: got.EMAFast = %d, base.EMAFast = %d, want 3 и 10", got.EMAFast, base.EMAFast)
	}
	if got.EMASlow != 50 || base.EMASlow != 100 {
		t.Fatalf("медленная EMA: got.EMASlow = %d, base.EMASlow = %d, want 50 и 100", got.EMASlow, base.EMASlow)
	}
	if got.TPDailyATR != 0.2 || base.TPDailyATR != 0.6 {
		t.Fatalf("цель: got.TPDailyATR = %v, base.TPDailyATR = %v, want 0.2 и 0.6", got.TPDailyATR, base.TPDailyATR)
	}
	if got.UseTrail != 1 || base.UseTrail != 0 {
		t.Fatalf("трейл: got.UseTrail = %d, base.UseTrail = %d, want 1 и 0", got.UseTrail, base.UseTrail)
	}
	if got.TrailDailyATR != 0.5 || base.TrailDailyATR != 0 {
		t.Fatalf("дистанция трейла: got.TrailDailyATR = %v, base.TrailDailyATR = %v, want 0.5 и 0", got.TrailDailyATR, base.TrailDailyATR)
	}
}

func TestTickerIsMVID(t *testing.T) {
	if Ticker != "MVID" {
		t.Fatalf("Ticker = %q, want MVID", Ticker)
	}
}
