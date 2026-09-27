// Package mdmg supplies the ticker and rsi_pullback Params for MDMG (МКПАО «МД Медикал Груп»,
// «Мать и дитя», обыкновенные акции после редомициляции; до неё — расписки).
//
// Не путать с пакетом reversion/strategy/mdmg: другая стратегия, другое дерево.
//
// СОСТОЯНИЕ: откалиброван 2026-09-27, разбор ниже. Процедура и критерии —
// docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md.
package mdmg

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "MDMG"

// DefaultParams returns the rsi_pullback parameters for MDMG — принятую точку второго круга. Все
// восемнадцать полей выписаны явно, включая совпавшие с дефолтом ядра: пакет обязан читаться без
// обращения к ядру. Разбор — doc-комментарий пакета; снимок держит mdmg_test.go.
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       6,
		RSILower:        25,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.6,
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
