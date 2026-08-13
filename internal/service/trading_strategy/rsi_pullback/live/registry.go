package live

import (
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/reni"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tbank"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ugld"
)

// paramsByTicker maps every rsi_pullback ticker the runner knows to its params. The
// configured universe (RSI_PULLBACK_TICKERS) selects which of these actually trade;
// NVTK is registered for completeness but has no calibrated literal yet — it returns the
// baseline — and must not be put into the universe. FESH got its literal 2026-08-13, which
// only lifts that mechanical block: the standard §8 protocol does not confirm the ticker
// (see the package doc), so putting it into the universe stays the owner's call.
var paramsByTicker = map[string]core.Params{
	ugld.Ticker:  ugld.DefaultParams(),
	tbank.Ticker: tbank.DefaultParams(),
	gazp.Ticker:  gazp.DefaultParams(),
	nvtk.Ticker:  nvtk.DefaultParams(),
	domrf.Ticker: domrf.DefaultParams(),
	reni.Ticker:  reni.DefaultParams(),
	fesh.Ticker:  fesh.DefaultParams(),
}

// ParamsFor returns the params for a known ticker, ok=false otherwise.
func ParamsFor(ticker string) (core.Params, bool) {
	p, ok := paramsByTicker[ticker]
	return p, ok
}

// StrategyFor returns the strategy for a known ticker, ok=false otherwise.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := paramsByTicker[ticker]
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}
