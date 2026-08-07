// Package ugld supplies the ticker and calibrated rsi_pullback Params for UGLD
// (Yuzhuralzoloto / ЮГК).
//
// ОТКАЛИБРОВАН — это единственный тикер стратегии, прошедший walk-forward (in-sample PF
// 2.555 на 36 сделках, pooled OOS PF 2.4–3.6 на 12; docs/rsi_pullback/strategy.md §8).
// Тело — явный литерал, и связь с core.DefaultParams() разорвана осознанно: правка baseline
// НЕ должна доходить сюда, иначе прод поедет относительно того, что валидировалось.
// Литерал прибит тестом ugld_test.go.
package ugld

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "UGLD"

// DefaultParams returns UGLD's starting rsi_pullback parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       6,
		RSILower:        15,
		RSIUpper:        60,
		EMAFast:         20,
		EMASlow:         150,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.3,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      1,
		UseVolume:       1,
		VolBaseDays:     5,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        1,
		TrailDailyATR:   0.5,
	}
}
