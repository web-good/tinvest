package spbe

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
		EMASlow:         50,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      0.6,
		UseVolume:       1,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        1,
		TrailDailyATR:   0.7,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки SPBE разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestPointDiffersFromTheCoreBaseline сторожит ровно три решения, которыми принятая точка
// отличается от дефолтов ядра: медленная EMA 50 вместо 100 (тема trend, голоса 50/50/250/50 —
// большинство 3 из 4), объёмный гейт (тема screen, голоса 0/1/1/1 — большинство 3 из 4) и трейл
// 0.7 при включённом UseTrail (тема trail, голоса 0.7/0.7/0.7/1.0 — большинство 3 из 4).
// Если литерал когда-нибудь схлопнется обратно в baseline, это будет означать потерю калибровки,
// а не упрощение.
func TestPointDiffersFromTheCoreBaseline(t *testing.T) {
	got, base := DefaultParams(), core.DefaultParams()
	if got == base {
		t.Fatal("литерал SPBE совпал с baseline ядра: калибровка потеряна")
	}
	if got.EMASlow != 50 || base.EMASlow != 100 {
		t.Fatalf("медленная EMA: got.EMASlow = %d, base.EMASlow = %d, want 50 и 100", got.EMASlow, base.EMASlow)
	}
	if got.UseVolume != 1 || base.UseVolume != 0 {
		t.Fatalf("объёмный гейт: got.UseVolume = %d, base.UseVolume = %d, want 1 и 0", got.UseVolume, base.UseVolume)
	}
	if got.UseTrail != 1 || base.UseTrail != 0 {
		t.Fatalf("трейл: got.UseTrail = %d, base.UseTrail = %d, want 1 и 0", got.UseTrail, base.UseTrail)
	}
	if got.TrailDailyATR != 0.7 || base.TrailDailyATR != 0 {
		t.Fatalf("дистанция трейла: got.TrailDailyATR = %v, base.TrailDailyATR = %v, want 0.7 и 0", got.TrailDailyATR, base.TrailDailyATR)
	}
}

// TestEffectiveProtectionPassesRiskGateA сторожит риск-гейт A на литерале: эффективная защита —
// это min(StopDailyATR, TrailDailyATR при UseTrail=1), и она обязана оставаться не шире потолка
// 1.0 дневного ATR (выживаемость 44.7% будних дней окна при пороге 30%). Принятые 0.5 дают 90.5%.
// Тест ловит именно обход гейта через трейл (урок AFKS): расширение любого из двух полей мимо
// калибровки уводит защиту за потолок.
func TestEffectiveProtectionPassesRiskGateA(t *testing.T) {
	p := DefaultParams()
	protection := p.StopDailyATR
	if p.UseTrail == 1 && p.TrailDailyATR < protection {
		protection = p.TrailDailyATR
	}
	if protection > 1.0 {
		t.Fatalf("эффективная защита %v ATR шире потолка риск-гейта A (1.0)", protection)
	}
}

func TestTickerIsSPBE(t *testing.T) {
	if Ticker != "SPBE" {
		t.Fatalf("Ticker = %q, want SPBE", Ticker)
	}
}
