package svcb

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsMatchTheCalibratedSnapshot пинит принятую точку первого круга целиком (все
// восемнадцать полей литералами, а не вычислены из core.DefaultParams()): параметры этого пакета
// уходят в боевую вселенную живого раннера, и любое поле, изменённое мимо калибровки, обязано
// валить тест. Разбор — data/params/rsi_pullback/svcb/plateau_point.json (_comment) и
// docs/superpowers/plans/task-11r-report-svcb.md.
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         20,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.7,
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
		t.Fatalf("литерал SVCB разошёлся со снимком калибровки:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSVCB(t *testing.T) {
	if Ticker != "SVCB" {
		t.Fatalf("Ticker = %q, want SVCB", Ticker)
	}
}
