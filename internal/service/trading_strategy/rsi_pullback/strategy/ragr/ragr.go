// Package ragr supplies the ticker and rsi_pullback Params for RAGR (ПАО «Русагро»,
// обыкновенные акции после редомициляции; до неё — расписки AGRO).
//
// СОСТОЯНИЕ: калибровка не проводилась, параметры следуют дефолтам ядра. Процедура и критерии —
// docs/superpowers/specs/2026-09-26-ragr-rsi-pullback-prep-design.md.
package ragr

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "RAGR"

// DefaultParams returns the rsi_pullback parameters for RAGR.
func DefaultParams() core.Params { return core.DefaultParams() }
