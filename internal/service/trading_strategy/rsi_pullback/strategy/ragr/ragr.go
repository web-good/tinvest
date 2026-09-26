// Package ragr supplies the ticker and rsi_pullback Params for RAGR (ПАО «Русагро»,
// обыкновенные акции после редомициляции; до неё — расписки AGRO).
//
// СОСТОЯНИЕ: ОТКАЛИБРОВАН 2026-09-26, ТОЧКА ВТОРОГО КРУГА ПРИНЯТА, разбор ниже (Task 17). Разбор
// сборки, гейтов и walk-forward — data/params/rsi_pullback/ragr/plateau_point2.json (_comment) и
// docs/superpowers/plans/task-12-report-ragr.md.
package ragr

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "RAGR"

// DefaultParams returns the rsi_pullback parameters for RAGR — принятую точку второго круга
// (Task 12). Все восемнадцать полей выписаны явно, включая совпавшие с дефолтом ядра: пакет
// обязан читаться без обращения к ядру. Снимок держит ragr_test.go.
func DefaultParams() core.Params {
	return core.Params{
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
}
