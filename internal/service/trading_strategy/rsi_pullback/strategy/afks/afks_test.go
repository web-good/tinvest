package afks

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsMatchTheCalibratedSnapshot закрепляет точку единой калибровки 2026-09-30 целиком (все
// поля литералами, а не вычислены из core.DefaultParams()): режим 0, дефолты ядра + EMASlow 50.
// Любое поле, изменённое мимо калибровки, обязано валить тест. Разбор — doc-comment пакета и
// data/params/rsi_pullback/afks/plateau_point.json (_comment).
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
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
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
		UseZoneEntry:    0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал AFKS разошёлся со снимком калибровки:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsAFKS(t *testing.T) {
	if Ticker != "AFKS" {
		t.Fatalf("Ticker = %q, want AFKS", Ticker)
	}
}
