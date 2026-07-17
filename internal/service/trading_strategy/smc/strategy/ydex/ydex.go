// Package ydex supplies the ticker and starting SMC Params for YDEX (Яндекс).
// Starting values mirror the generic defaults; calibrate with -calibrate and
// then hardcode the winning combination here.
package ydex

import "tinvest/internal/service/trading_strategy/smc/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "YDEX"

// DefaultParams returns YDEX's starting SMC parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{SwingK: 3, ReclaimBars: 4, Buffer: 0.5, TPR: 2, MaxHoldDays: 3}
}
