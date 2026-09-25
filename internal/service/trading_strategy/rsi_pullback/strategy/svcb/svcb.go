// Package svcb supplies the ticker and rsi_pullback Params for SVCB (ПАО «Совкомбанк»,
// обыкновенные акции, лот 100).
//
// СОСТОЯНИЕ: калибровка не проводилась, параметры следуют дефолтам ядра. Процедура и критерии —
// docs/superpowers/specs/2026-09-25-svcb-rsi-pullback-prep-design.md.
package svcb

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "SVCB"

// DefaultParams returns the rsi_pullback parameters for SVCB.
func DefaultParams() core.Params { return core.DefaultParams() }
