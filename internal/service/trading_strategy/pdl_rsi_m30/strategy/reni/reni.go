// Package reni supplies the independent pdl_rsi_m30 baseline Params for RENI
// (Ренессанс Страхование) on Minutes30.
//
// The numbers are a mechanical starting point from RENI's Hour1 literal. They are not an M30
// calibration result and must be replaced only after the dedicated rolling walk-forward procedure.
package reni

import "tinvest/internal/service/trading_strategy/pdl_rsi_m30/strategy/core"

// Params returns the RENI M30 baseline literal.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      6,
		RSIArm:         25,
		RSIFloor:       0,
		RSIExit:        70,
		StochK:         13,
		StochSmooth:    3,
		StochD:         3,
		VolumeMult:     0,
		VolBars:        20,
		MinRoomATR:     0,
		StopATR:        0.3,
		MaxStopPct:     0,
		TargetR:        3,
		TrendEMA:       200,
		DailyATRPeriod: 14,
	}
}
