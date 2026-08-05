// Package domrf supplies the ticker and starting rsi_pullback Params for DOMRF (ДОМ.РФ).
//
// Calibration has NOT been run for this ticker: the body returns core.DefaultParams()
// unchanged rather than copying its fields, so a change to the baseline still reaches every
// uncalibrated ticker instead of silently drifting away from it.
//
// Read this before calibrating DOMRF. The instrument IPO'd on 2025-11-20, so its entire history
// is 8.4 months (158 weekday dailies) — the walk-forward protocol of docs/rsi_pullback/strategy.md
// §8 needs a 12-month train window and simply does not fit. Worse, that whole history is a single
// post-IPO uptrend: 1749.8 -> 2273.2 RUB, +29.9%, with no sustained downward regime. A long-only
// pullback strategy shows an inflated profit factor in that regime whatever its parameters, so a
// good number here is evidence of the regime, not of an edge. The daily ATR(14) runs a median
// 1.94% of price — half of UGLD's 4.28% and in the same class as GAZP and T — which is why the
// grids in data/params/rsi_pullback/domrf/ are rescaled rather than copied from a sibling.
//
// Once a walk-forward on 18-24 months of history picks a winning combination, replace the body
// with an explicit literal — from that point the ticker must stop tracking the baseline.
package domrf

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "DOMRF"

// DefaultParams returns DOMRF's starting rsi_pullback parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.DefaultParams()
}
