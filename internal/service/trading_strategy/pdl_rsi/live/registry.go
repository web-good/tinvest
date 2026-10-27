// Package live подключает pdl_rsi (Hour1) к живому раннеру счёта rsi_pullback
// (rsi_pullback/live) адаптером livecore/adapter.Strategy. Механика — docs/pdl_rsi/live.md.
package live

import (
	"time"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gazp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lent"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mdmg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/svcb"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vtbr"
)

// paramsByTicker — тикеры, которые раннер знает (литералы пакетов тикеров, как в реестре
// бэктеста). Торгуют только перечисленные в PDL_RSI_TICKERS.
var paramsByTicker = map[string]func() core.Params{
	"GAZP": gazp.Params,
	"LENT": lent.Params,
	"MDMG": mdmg.Params,
	"NVTK": nvtk.Params,
	"SVCB": svcb.Params,
	"VTBR": vtbr.Params,
}

// ParamsFor возвращает параметры тикера; false — тикер не зарегистрирован.
func ParamsFor(ticker string) (core.Params, bool) {
	f, ok := paramsByTicker[ticker]
	if !ok {
		return core.Params{}, false
	}
	return f(), true
}

// StrategyFor строит ядро стратегии для тикера на часовом шаге; false — тикер не зарегистрирован.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := ParamsFor(ticker)
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, time.Hour, p), true
}
