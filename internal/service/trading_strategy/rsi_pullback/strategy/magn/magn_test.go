package magn

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestDefaultParamsMatchTheCalibratedPoint пинит принятую точку первого круга целиком. Снимок стоит
// здесь, а не в комментарии, потому что любое поле, изменённое мимо калибровки, обязано валить
// тест: параметры этого пакета уходят в боевую вселенную живого раннера. Разбор — doc-комментарий
// пакета и docs/superpowers/plans/task-12-report-magn.md.
func TestDefaultParamsMatchTheCalibratedPoint(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         50,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.1,
		SpentDayATR:     0.8,
		StopDailyATR:    0.7,
		TPDailyATR:      0.6,
		UseVolume:       1,
		VolBaseDays:     14,
		VolLookbackBars: 1,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки MAGN разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestPointDiffersFromTheCoreBaseline сторожит пять полей, которыми принятая точка отличается от
// дефолтов ядра, и держит их вместе: трендовый фильтр EMASlow=50 (единогласие всех четырёх фолдов
// темы trend), дневная свежесть FreshDayATR=0.1, включённый объёмный гейт UseVolume=1 с окном
// VolLookbackBars=1 и стоп 0.7 — последний пришёл не голосом темы, а решением риск-гейта A, который
// отсёк проголосованные фолдами 1.5 и 2.0 как фикцию защиты. Схлопывание литерала обратно в
// baseline не безобидно: дефолты ядра на MAGN дают ровный ноль (pooled OOS PF 1.005 при двух
// убыточных фолдах из четырёх), поэтому маршрут «завести на дефолтах» прецедента TGKA здесь закрыт.
func TestPointDiffersFromTheCoreBaseline(t *testing.T) {
	got, base := DefaultParams(), core.DefaultParams()
	if got == base {
		t.Fatal("литерал MAGN совпал с baseline ядра: калибровка потеряна, а дефолты на этой бумаге дают ровный ноль")
	}
	if got.EMASlow != 50 || base.EMASlow != 100 {
		t.Fatalf("трендовый фильтр: got.EMASlow = %v, base.EMASlow = %v, want 50 и 100", got.EMASlow, base.EMASlow)
	}
	if got.FreshDayATR != 0.1 || base.FreshDayATR != 0 {
		t.Fatalf("дневная свежесть: got.FreshDayATR = %v, base.FreshDayATR = %v, want 0.1 и 0", got.FreshDayATR, base.FreshDayATR)
	}
	if got.UseVolume != 1 || base.UseVolume != 0 {
		t.Fatalf("объёмный гейт: got.UseVolume = %v, base.UseVolume = %v, want 1 и 0", got.UseVolume, base.UseVolume)
	}
	if got.VolLookbackBars != 1 || base.VolLookbackBars != 3 {
		t.Fatalf("окно объёмного гейта: got.VolLookbackBars = %v, base.VolLookbackBars = %v, want 1 и 3", got.VolLookbackBars, base.VolLookbackBars)
	}
	if got.StopDailyATR != 0.7 || base.StopDailyATR != 0.5 {
		t.Fatalf("стоп риск-гейта A: got.StopDailyATR = %v, base.StopDailyATR = %v, want 0.7 и 0.5", got.StopDailyATR, base.StopDailyATR)
	}
}

func TestTickerIsMAGN(t *testing.T) {
	if Ticker != "MAGN" {
		t.Fatalf("Ticker = %q, want MAGN", Ticker)
	}
}
