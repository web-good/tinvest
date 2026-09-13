package irkt

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestDefaultParamsMatchTheCalibratedPoint пинит принятую точку первого круга целиком. Снимок стоит
// здесь, а не в комментарии, потому что любое поле, изменённое мимо калибровки, обязано валить
// тест: параметры этого пакета уходят в боевую вселенную живого раннера (когда IRKT будет в неё
// заведён). Разбор — doc-комментарий пакета и docs/superpowers/plans/task-10-report-irkt.md.
func TestDefaultParamsMatchTheCalibratedPoint(t *testing.T) {
	want := core.Params{
		RSIPeriod:       3,
		RSILower:        15,
		RSIUpper:        60,
		EMAFast:         10,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      0.6,
		UseVolume:       0,
		VolBaseDays:     3,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки IRKT разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsIRKT(t *testing.T) {
	if Ticker != "IRKT" {
		t.Fatalf("Ticker = %q, want IRKT", Ticker)
	}
}
