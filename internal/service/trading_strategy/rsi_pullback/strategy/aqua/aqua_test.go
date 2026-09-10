package aqua

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestDefaultParamsMatchTheCalibratedPoint пинит принятую точку второго круга целиком. Снимок стоит
// здесь, а не в комментарии, потому что любое поле, изменённое мимо калибровки, обязано валить
// тест: параметры этого пакета уходят в боевую вселенную живого раннера. Разбор — doc-комментарий
// пакета и docs/superpowers/plans/task-13-report-aqua.md.
func TestDefaultParamsMatchTheCalibratedPoint(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        45,
		EMAFast:         10,
		EMASlow:         100,
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
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки AQUA разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestPointDiffersFromTheCoreBaseline сторожит единственное поле, которым принятая точка отличается
// от дефолтов ядра: ранний RSI-выход 45 вместо 70. Это поле выиграло единогласно во всех четырёх
// фолдах темы r2_pair второго круга и остаётся единственным рабочим рычагом AQUA — дефолты ядра на
// этой бумаге убыточны (PF 0.931 на расчётном окне). Если литерал когда-нибудь схлопнется обратно в
// baseline, это будет означать потерю калибровки, а не упрощение.
func TestPointDiffersFromTheCoreBaseline(t *testing.T) {
	got, base := DefaultParams(), core.DefaultParams()
	if got == base {
		t.Fatal("литерал AQUA совпал с baseline ядра: калибровка потеряна")
	}
	if got.RSIUpper != 45 || base.RSIUpper != 70 {
		t.Fatalf("ранний RSI-выход: got.RSIUpper = %v, base.RSIUpper = %v, want 45 и 70", got.RSIUpper, base.RSIUpper)
	}
}

func TestTickerIsAQUA(t *testing.T) {
	if Ticker != "AQUA" {
		t.Fatalf("Ticker = %q, want AQUA", Ticker)
	}
}
