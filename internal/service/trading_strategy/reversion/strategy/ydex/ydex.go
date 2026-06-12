// Package ydex supplies the ticker and starting reversion Params for YDEX (Yandex).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here.
package ydex

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "YDEX"

// DefaultParams returns YDEX's starting reversion parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
	}
}
