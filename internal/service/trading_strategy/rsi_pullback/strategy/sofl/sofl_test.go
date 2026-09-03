package sofl

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestDefaultParamsIsTheCalibratedSnapshot пинит принятую точку SOFL целиком. Литерал —
// результат процедуры Task 12 (гибридные темы, правило большинства 3 из 4, два риск-гейта), и
// любое его изменение обязано быть осознанным: тест падает на каждом поле.
func TestDefaultParamsIsTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       3,
		RSILower:        10,
		RSIUpper:        65,
		EMAFast:         10,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.2,
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
		t.Fatalf("литерал SOFL разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestDefaultParamsDiffersFromCoreBaseline держит вторую половину смысла: калибровка проведена, и
// точка обязана отличаться от дефолтов ядра хотя бы одним полем.
func TestDefaultParamsDiffersFromCoreBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("литерал SOFL совпал с baseline ядра — значит калибровка не отражена в коде")
	}
}

func TestTickerIsSOFL(t *testing.T) {
	if Ticker != "SOFL" {
		t.Fatalf("Ticker = %q, want SOFL", Ticker)
	}
}
