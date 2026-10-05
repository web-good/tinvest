// Package live подключает rsi_zone к живому раннеру счёта rsi_pullback
// (rsi_pullback/live) адаптером livecore/adapter.Strategy. Механика — docs/rsi_zone/live.md.
package live

import (
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/afks"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/baza"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/dias"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/ivat"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/lent"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/mdmg"
)

// paramsByTicker — тикеры, которые раннер знает. SBER сюда не заводится: калибровка планку
// не взяла. Торгуют только перечисленные в RSI_ZONE_TICKERS.
var paramsByTicker = map[string]core.Params{
	afks.Ticker:  afks.DefaultParams(),
	baza.Ticker:  baza.DefaultParams(),
	dias.Ticker:  dias.DefaultParams(),
	domrf.Ticker: domrf.DefaultParams(),
	ivat.Ticker:  ivat.DefaultParams(),
	lent.Ticker:  lent.DefaultParams(),
	mdmg.Ticker:  mdmg.DefaultParams(),
}

// ParamsFor возвращает параметры тикера; false — тикер не зарегистрирован.
func ParamsFor(ticker string) (core.Params, bool) {
	p, ok := paramsByTicker[ticker]
	return p, ok
}

// StrategyFor строит ядро стратегии для тикера; false — тикер не зарегистрирован.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := paramsByTicker[ticker]
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}
