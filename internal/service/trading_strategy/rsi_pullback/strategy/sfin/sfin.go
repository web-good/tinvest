// Package sfin supplies the ticker and rsi_pullback Params for SFIN (ПАО «ЭсЭфАй», обыкновенные
// акции, лот 1).
//
// СОСТОЯНИЕ: КАЛИБРОВКА НЕ ПРОВОДИЛАСЬ, параметры следуют дефолтам ядра — см.
// docs/superpowers/specs/2026-09-15-sfin-rsi-pullback-prep-design.md.
package sfin

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package parameterises.
const Ticker = "SFIN"

// DefaultParams returns the baseline until calibration replaces it with a literal.
func DefaultParams() core.Params {
	return core.DefaultParams()
}
