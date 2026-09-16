// Package sfin supplies the ticker and rsi_pullback Params for SFIN (ПАО «ЭсЭфАй», обыкновенные
// акции, лот 1).
//
// СОСТОЯНИЕ: ОТКАЛИБРОВАН, точка второго круга принята — см. data/params/rsi_pullback/sfin/
// plateau_point2.json (_comment) и docs/superpowers/plans/task-12-report-sfin.md. Снимок
// литерала держит sfin_test.go (TestParamsMatchTheCalibratedSnapshot).
//
// Два поля ушли от дефолтов ядра: RSIPeriod 4 -> 5 (тема entry второго круга, единственное
// большинство ≥3/4) и UseVolume 0 -> 1 (перенесено из первого круга: тема screen дала 3/4, ось
// двоичная и решена единогласно по UseDayATRGate плюс большинством 3/4 по UseVolume). Остальные
// шестнадцать полей — дефолты ядра: ни одна другая узкая тема второго круга большинства не
// набрала (уплотнение сетки вдвое разрушило узлы первого круга вместо того, чтобы их уточнить).
package sfin

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "SFIN"

// DefaultParams returns the rsi_pullback parameters for SFIN — принятую точку второго круга. Все
// восемнадцать полей выписаны явно, включая совпавшие с дефолтом ядра: пакет обязан читаться без
// обращения к ядру. Разбор — doc-комментарий пакета; снимок держит sfin_test.go.
func DefaultParams() core.Params {
	return core.Params{
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
}
