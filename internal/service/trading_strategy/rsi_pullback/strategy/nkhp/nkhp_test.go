package nkhp

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
		EMAFast:         10,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    1.0,
		TPDailyATR:      0.6,
		UseVolume:       1,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки NKHP разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestPointDiffersFromTheCoreBaseline сторожит ровно два поля, которыми принятая точка отличается
// от дефолтов ядра: объёмный гейт (единогласный выбор всех четырёх фолдов темы screen) и широкий
// стоп (три фолда из четырёх темы risk). Если литерал когда-нибудь схлопнется обратно в baseline,
// это будет означать потерю калибровки, а не упрощение.
func TestPointDiffersFromTheCoreBaseline(t *testing.T) {
	got, base := DefaultParams(), core.DefaultParams()
	if got == base {
		t.Fatal("литерал NKHP совпал с baseline ядра: калибровка потеряна")
	}
	if got.UseVolume != 1 || base.UseVolume != 0 {
		t.Fatalf("объёмный гейт: got.UseVolume = %d, base.UseVolume = %d, want 1 и 0", got.UseVolume, base.UseVolume)
	}
	if got.StopDailyATR != 1.0 || base.StopDailyATR != 0.5 {
		t.Fatalf("стоп: got.StopDailyATR = %v, base.StopDailyATR = %v, want 1.0 и 0.5", got.StopDailyATR, base.StopDailyATR)
	}
}

func TestTickerIsNKHP(t *testing.T) {
	if Ticker != "NKHP" {
		t.Fatalf("Ticker = %q, want NKHP", Ticker)
	}
}
