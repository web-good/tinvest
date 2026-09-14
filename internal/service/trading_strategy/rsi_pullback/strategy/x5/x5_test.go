package x5

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsMatchTheCalibratedSnapshot пинит принятую точку первого круга целиком. Снимок стоит
// здесь, а не в комментарии, потому что любое поле, изменённое мимо калибровки, обязано валить
// тест: параметры этого пакета уходят в боевую вселенную живого раннера. Особо важны два поля.
// UseVolume=1 — несущая конструкция точки: без объёмного гейта PF падает 2.204 -> 1.508, просадка
// растёт 3321 -> 10 942 ₽, и возвращается печать 2026-01-02 (-10 453.45 ₽). FreshDayATR=0 — решение
// владельца по риск-гейту C: на 0.1 точка входила 17 раз на баре 06:30, где медиана объёма 148 лотов
// против 25 812 в основной сессии. Разбор — doc-комментарий пакета и
// docs/superpowers/plans/task-10-report-x5.md.
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        70,
		EMAFast:         5,
		EMASlow:         50,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      0.6,
		UseVolume:       1,
		VolBaseDays:     3,
		VolLookbackBars: 12,
		VolMult:         1.0,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал точки X5 разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestTickerIsX5(t *testing.T) {
	if Ticker != "X5" {
		t.Fatalf("Ticker = %q, want X5", Ticker)
	}
}
