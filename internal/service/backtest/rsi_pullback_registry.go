package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	rsipullbackdomrf "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	rsipullbackfesh "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
	rsipullbackgazp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	rsipullbackivat "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ivat"
	rsipullbacklent "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lent"
	rsipullbacklsngp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lsngp"
	rsipullbacknvtk "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	rsipullbackreni "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/reni"
	rsipullbacksibn "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sibn"
	rsipullbacksvav "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svav"
	rsipullbacktbank "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tbank"
	rsipullbackugld "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ugld"
	rsipullbackwush "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// rsiPullbackBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All rsi_pullback tickers share the core engine; only ticker + defaults differ.
func rsiPullbackBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse rsi_pullback params: %w", err)
			}
			return p, nil
		},
	}
}

// rsiPullbackRegistry gives every tracked ticker its own parameter package. An entry is in one of
// three states, documented on the package itself: baseline-tracking (DefaultParams() as-is,
// calibration pending), calibrated (an explicit literal from that ticker's own walk-forward run,
// e.g. gazp, tbank), or seeded from another ticker's literal as an explicit transferability
// hypothesis, not a claim of being tuned — see docs/rsi_pullback/strategy.md §8.0.1.
var rsiPullbackRegistry = map[string]Binding{
	rsipullbackdomrf.Ticker: rsiPullbackBindingFor(rsipullbackdomrf.Ticker, rsipullbackdomrf.DefaultParams),
	rsipullbackfesh.Ticker:  rsiPullbackBindingFor(rsipullbackfesh.Ticker, rsipullbackfesh.DefaultParams),
	rsipullbackgazp.Ticker:  rsiPullbackBindingFor(rsipullbackgazp.Ticker, rsipullbackgazp.DefaultParams),
	rsipullbackivat.Ticker:  rsiPullbackBindingFor(rsipullbackivat.Ticker, rsipullbackivat.DefaultParams),
	rsipullbacklent.Ticker:  rsiPullbackBindingFor(rsipullbacklent.Ticker, rsipullbacklent.DefaultParams),
	rsipullbacklsngp.Ticker: rsiPullbackBindingFor(rsipullbacklsngp.Ticker, rsipullbacklsngp.DefaultParams),
	rsipullbacknvtk.Ticker:  rsiPullbackBindingFor(rsipullbacknvtk.Ticker, rsipullbacknvtk.DefaultParams),
	rsipullbackreni.Ticker:  rsiPullbackBindingFor(rsipullbackreni.Ticker, rsipullbackreni.DefaultParams),
	rsipullbacksibn.Ticker:  rsiPullbackBindingFor(rsipullbacksibn.Ticker, rsipullbacksibn.DefaultParams),
	rsipullbacksvav.Ticker:  rsiPullbackBindingFor(rsipullbacksvav.Ticker, rsipullbacksvav.DefaultParams),
	rsipullbacktbank.Ticker: rsiPullbackBindingFor(rsipullbacktbank.Ticker, rsipullbacktbank.DefaultParams),
	rsipullbackugld.Ticker:  rsiPullbackBindingFor(rsipullbackugld.Ticker, rsipullbackugld.DefaultParams),
	rsipullbackwush.Ticker:  rsiPullbackBindingFor(rsipullbackwush.Ticker, rsipullbackwush.DefaultParams),
}

// RSIPullbackLookupOrGeneric returns the registered rsi_pullback binding for a ticker, or a
// generic binding bound to that ticker (with core.DefaultParams) when none is registered.
func RSIPullbackLookupOrGeneric(ticker string) Binding {
	if b, ok := rsiPullbackRegistry[ticker]; ok {
		return b
	}
	return rsiPullbackBindingFor(ticker, core.DefaultParams)
}
