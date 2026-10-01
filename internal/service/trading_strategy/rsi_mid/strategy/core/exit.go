package core

import (
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// manage handles an open long. Implemented in full in the exit task.
func (s *Strategy) manage(_ strategy.MarketData, sig model.Signal) model.Signal {
	return sig
}
