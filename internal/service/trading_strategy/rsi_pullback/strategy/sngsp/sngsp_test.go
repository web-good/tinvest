package sngsp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsAreTheCalibratedSnapshot прибивает принятый 2026-09-01 литерал. Тест падает и на
// молчаливом дрейфе полей, и на откате к core.DefaultParams(): связь с baseline разорвана
// осознанно, и любое её восстановление обязано быть видимым в диффе.
func TestParamsAreTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        70,
		EMAFast:         20,
		EMASlow:         50,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.2,
		SpentDayATR:     0.7,
		StopDailyATR:    0.5,
		TPDailyATR:      0.3,
		UseVolume:       1,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         2.5,
		UseRSIExit:      1,
		UseTrail:        1,
		TrailDailyATR:   0.5,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал SNGSP разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestParamsDifferFromTheCoreBaseline сторожит сам факт калибровки: если литерал вернулся к
// дефолтам ядра, значит правка откатила работу целиком, и это не должно проходить молча.
func TestParamsDifferFromTheCoreBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("SNGSP откалиброван — DefaultParams() не может совпадать с core.DefaultParams()")
	}
}

func TestTickerIsSNGSP(t *testing.T) {
	if Ticker != "SNGSP" {
		t.Fatalf("Ticker = %q, want SNGSP", Ticker)
	}
}
