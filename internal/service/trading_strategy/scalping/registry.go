package scalping

import (
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/service/trading_strategy/scalping/strategy/afks"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

// defaultStrategies is the fixed set of per-share strategies the runner evaluates.
func defaultStrategies() []strategy.Strategy {
	return []strategy.Strategy{
		rusal.New(),
		afks.New(),
	}
}
