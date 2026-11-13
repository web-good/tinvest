// Package mrkc provides default parameters for MRKC in the FVG imbalance backtest.
package mrkc

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for MRKC.
func Params() core.Params { return core.DefaultParams() }
