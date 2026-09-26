package ragr

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsMatchTheCalibratedSnapshot закрепляет принятую точку второго круга целиком (все
// восемнадцать полей литералами, а не вычислены из core.DefaultParams()): параметры этого пакета
// уходят в боевую вселенную живого раннера, и любое поле, изменённое мимо калибровки, обязано
// валить тест. Разбор — data/params/rsi_pullback/ragr/plateau_point2.json (_comment) и
// docs/superpowers/plans/task-12-report-ragr.md.
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        35,
		EMAFast:         3,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.8,
		TPDailyATR:      0.6,
		UseVolume:       1,
		VolBaseDays:     14,
		VolLookbackBars: 1,
		VolMult:         1.6,
		UseRSIExit:      1,
		UseTrail:        1,
		TrailDailyATR:   0.5,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал RAGR разошёлся со снимком калибровки:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsRAGR(t *testing.T) {
	if Ticker != "RAGR" {
		t.Fatalf("Ticker = %q, want RAGR", Ticker)
	}
}
