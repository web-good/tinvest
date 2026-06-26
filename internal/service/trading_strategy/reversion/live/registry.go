package live

import (
	"tinvest/internal/service/trading_strategy/reversion/strategy/astr"
	"tinvest/internal/service/trading_strategy/reversion/strategy/core"
	"tinvest/internal/service/trading_strategy/reversion/strategy/eutr"
	"tinvest/internal/service/trading_strategy/reversion/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/reversion/strategy/sfin"
	"tinvest/internal/service/trading_strategy/reversion/strategy/ugld"
)

// paramsByTicker maps every reversion ticker the runner knows to its calibrated
// params. The configured universe (env) selects which of these actually trade; SFIN
// is registered for completeness but is "DO NOT TRADE" and must not be in the universe.
var paramsByTicker = map[string]core.Params{
	ugld.Ticker: ugld.DefaultParams(),
	eutr.Ticker: eutr.DefaultParams(),
	nvtk.Ticker: nvtk.DefaultParams(),
	astr.Ticker: astr.DefaultParams(),
	sfin.Ticker: sfin.DefaultParams(),
}

// StrategyFor returns the calibrated strategy for a known ticker, ok=false otherwise.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := paramsByTicker[ticker]
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}

// MaxHTFTrendEMA returns the largest HTFTrendEMA across the given tickers (0 = no 4H
// filter needed by any). Unknown tickers contribute 0.
func MaxHTFTrendEMA(tickers []string) int {
	m := 0
	for _, t := range tickers {
		if p, ok := paramsByTicker[t]; ok && p.HTFTrendEMA > m {
			m = p.HTFTrendEMA
		}
	}
	return m
}
