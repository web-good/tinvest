// Package nvtk supplies the pdl_zone Params for NVTK (Новатэк) on Minutes5.
//
// # 2026-10-21 — литерал до калибровки
//
// Числа — правила владельца (core.DefaultParams), кроме цели: TargetR 1.7 владелец поставил сам.
// Записаны литералом, чтобы правка дефолтов ядра не меняла тикер молча. Калибровка — скользящий walk-forward, сетка
// data/params/pdl_zone/nvtk/wf_rolling.json.
package nvtk

import "tinvest/internal/service/trading_strategy/pdl_zone/strategy/core"

// Params returns the NVTK literal.
func Params() core.Params {
	return core.Params{
		EntryZone:      1,
		EntryBreak:     1,
		ZoneATR:        0.2,
		BreakReclaim:   0,
		MACDFast:       8,
		MACDSlow:       21,
		MACDSignal:     9,
		VolumeMult:     1.5,
		VolBars:        15,
		StopBars:       5,
		MinStopATR:     0.2,
		TargetR:        1.7,
		DailyATRPeriod: 14,
	}
}
