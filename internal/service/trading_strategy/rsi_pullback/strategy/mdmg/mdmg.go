// Package mdmg supplies the ticker and rsi_pullback Params for MDMG (МКПАО «МД Медикал Груп»,
// «Мать и дитя», обыкновенные акции после редомициляции; до неё — расписки).
//
// Не путать с пакетом reversion/strategy/mdmg: другая стратегия, другое дерево.
//
// СОСТОЯНИЕ: калибровка не проводилась, параметры следуют дефолтам ядра. Процедура и критерии —
// docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md.
package mdmg

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "MDMG"

// DefaultParams returns the rsi_pullback parameters for MDMG.
func DefaultParams() core.Params { return core.DefaultParams() }
