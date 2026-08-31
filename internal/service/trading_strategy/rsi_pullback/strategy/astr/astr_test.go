package astr

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestCalibratedLiteralIsPinned фиксирует принятую точку ASTR (data/params/rsi_pullback/astr/
// plateau_point.json, собрана 2026-08-31 из пофолдовых победителей десяти тем, схема 34/9/6).
// Любое изменение литерала обязано пройти через ту же процедуру и обновить доку пакета.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        70,
		EMAFast:         5,
		EMASlow:         30,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    1.3,
		TPDailyATR:      0.6,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал ASTR изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("ASTR вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

func TestStopStaysAboveTheCostFloor(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR < 0.3 {
		t.Fatalf("StopDailyATR = %v: на стопе 0.3 ATR реальный круг издержек 0.124%% съедает 13.1%% риска, уже — ещё больше", p.StopDailyATR)
	}
}

func TestTargetIsArmed(t *testing.T) {
	if p := DefaultParams(); p.TPDailyATR <= 0 {
		t.Fatalf("TPDailyATR = %v, want > 0", p.TPDailyATR)
	}
}

func TestTargetStaysReachable(t *testing.T) {
	if p := DefaultParams(); p.TPDailyATR > 1.5 {
		t.Fatalf("TPDailyATR = %v: на ASTR цель шире 1.5 дневного ATR недостижима — колонки 2.0 и 2.5 замера совпадают побайтово", p.TPDailyATR)
	}
}

func TestTrendPairIsValid(t *testing.T) {
	if p := DefaultParams(); p.EMAFast >= p.EMASlow {
		t.Fatalf("EMAFast = %d >= EMASlow = %d: трендовый фильтр вырожден или инвертирован", p.EMAFast, p.EMASlow)
	}
}

func TestEntryBandIsBelowTheExitBand(t *testing.T) {
	if p := DefaultParams(); p.RSILower >= p.RSIUpper {
		t.Fatalf("RSILower = %v >= RSIUpper = %v: вход и выход перепутаны", p.RSILower, p.RSIUpper)
	}
}

func TestTickerIsASTR(t *testing.T) {
	if Ticker != "ASTR" {
		t.Fatalf("Ticker = %q, want ASTR", Ticker)
	}
}
