// Package gazp supplies the ticker and starting rsi_pullback Params for GAZP (Gazprom).
//
// Calibration HAS been run for this ticker. The values below are GAZP's own post-grid result,
// copied in as an explicit literal rather than left as core.DefaultParams() — the two consequences
// of that are deliberate. First, this package does NOT track the baseline: a future change to
// core.DefaultParams() must not reach GAZP silently, because these values came from GAZP's own
// calibration, not from the generic starting point. Second, once a fresh -calibrate run picks a
// different winning combination for GAZP, this literal must be REPLACED — leaving the old numbers
// in place after a newer run would ship a config the latest data never endorsed.
//
// Как и T, GAZP уходит в прод БЕЗ walk-forward подтверждения (решение владельца от
// 2026-08-05): есть только пост-грид литерал, OOS-прогона не было.
package gazp

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "GAZP"

// DefaultParams returns GAZP's calibrated rsi_pullback parameters: its own post-grid result,
// not core.DefaultParams().
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       4,
		RSILower:        25,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         70,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    0.5,
		TPDailyATR:      0.7,
		UseVolume:       0,
		VolBaseDays:     7,
		VolLookbackBars: 2,
		VolMult:         1,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
}
