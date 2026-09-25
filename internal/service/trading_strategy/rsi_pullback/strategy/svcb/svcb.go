// Package svcb supplies the ticker and rsi_pullback Params for SVCB (ПАО «Совкомбанк»,
// обыкновенные акции, лот 100).
//
// СОСТОЯНИЕ: ОТКАЛИБРОВАН 2026-09-25, точка первого круга (пересобранная задачей 11R) принята.
// Полный разбор — docs/superpowers/plans/task-11r-report-svcb.md и _comment
// data/params/rsi_pullback/svcb/plateau_point.json.
package svcb

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "SVCB"

// DefaultParams returns the rsi_pullback parameters for SVCB — принятую точку первого круга
// (задача 11R). Все восемнадцать полей выписаны явно, включая совпавшие с дефолтом ядра.
func DefaultParams() core.Params {
	return core.Params{
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
}
