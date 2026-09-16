package sfin

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsMatchTheCalibratedSnapshot пинит принятую точку второго круга целиком (все
// восемнадцать полей литералами, а не вычислены из core.DefaultParams()): параметры этого пакета
// уходят в боевую вселенную живого раннера, и любое поле, изменённое мимо калибровки, обязано
// валить тест. Разбор — data/params/rsi_pullback/sfin/plateau_point2.json (_comment) и
// docs/superpowers/plans/task-12-report-sfin.md.
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       5,
		RSILower:        30,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         100,
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
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки SFIN разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsSFIN(t *testing.T) {
	if Ticker != "SFIN" {
		t.Fatalf("Ticker = %q, want SFIN", Ticker)
	}
}
