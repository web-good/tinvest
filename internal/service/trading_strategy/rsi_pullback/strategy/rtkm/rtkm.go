// Package rtkm supplies the ticker and rsi_pullback Params for RTKM (ПАО «Ростелеком»,
// обыкновенные акции, лот 10).
//
// СОСТОЯНИЕ: калибровка не проводилась, параметры следуют дефолтам ядра. Процедура и критерии —
// docs/superpowers/specs/2026-09-25-rtkm-rsi-pullback-prep-design.md.
package rtkm

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "RTKM"

// DefaultParams returns the rsi_pullback parameters for RTKM.
func DefaultParams() core.Params { return core.DefaultParams() }
