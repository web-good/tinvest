package backtest

import (
	"encoding/json"
	"fmt"

	rsipullbackafks "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/afks"
	rsipullbackastr "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/astr"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	rsipullbackeutr "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/eutr"
	rsipullbackgazp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	rsipullbackmdmg "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mdmg"
	rsipullbacknvtk "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	rsipullbackpikk "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/pikk"
	rsipullbackplzl "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/plzl"
	rsipullbackrusal "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/rusal"
	rsipullbacksber "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sber"
	rsipullbacksfin "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sfin"
	rsipullbackugld "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ugld"
	rsipullbackydex "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ydex"
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

// rsiPullbackRegistry gives every tracked ticker its own parameter package. Each entry currently
// returns the generic baseline — calibration is pending — so the map exists to give a ticker
// somewhere to put its winning combination, not to claim the tickers are already tuned.
var rsiPullbackRegistry = map[string]Binding{
	rsipullbackafks.Ticker:  rsiPullbackBindingFor(rsipullbackafks.Ticker, rsipullbackafks.DefaultParams),
	rsipullbackastr.Ticker:  rsiPullbackBindingFor(rsipullbackastr.Ticker, rsipullbackastr.DefaultParams),
	rsipullbackeutr.Ticker:  rsiPullbackBindingFor(rsipullbackeutr.Ticker, rsipullbackeutr.DefaultParams),
	rsipullbackgazp.Ticker:  rsiPullbackBindingFor(rsipullbackgazp.Ticker, rsipullbackgazp.DefaultParams),
	rsipullbackmdmg.Ticker:  rsiPullbackBindingFor(rsipullbackmdmg.Ticker, rsipullbackmdmg.DefaultParams),
	rsipullbacknvtk.Ticker:  rsiPullbackBindingFor(rsipullbacknvtk.Ticker, rsipullbacknvtk.DefaultParams),
	rsipullbackpikk.Ticker:  rsiPullbackBindingFor(rsipullbackpikk.Ticker, rsipullbackpikk.DefaultParams),
	rsipullbackplzl.Ticker:  rsiPullbackBindingFor(rsipullbackplzl.Ticker, rsipullbackplzl.DefaultParams),
	rsipullbackrusal.Ticker: rsiPullbackBindingFor(rsipullbackrusal.Ticker, rsipullbackrusal.DefaultParams),
	rsipullbacksber.Ticker:  rsiPullbackBindingFor(rsipullbacksber.Ticker, rsipullbacksber.DefaultParams),
	rsipullbacksfin.Ticker:  rsiPullbackBindingFor(rsipullbacksfin.Ticker, rsipullbacksfin.DefaultParams),
	rsipullbackugld.Ticker:  rsiPullbackBindingFor(rsipullbackugld.Ticker, rsipullbackugld.DefaultParams),
	rsipullbackydex.Ticker:  rsiPullbackBindingFor(rsipullbackydex.Ticker, rsipullbackydex.DefaultParams),
}

// RSIPullbackLookupOrGeneric returns the registered rsi_pullback binding for a ticker, or a
// generic binding bound to that ticker (with core.DefaultParams) when none is registered.
func RSIPullbackLookupOrGeneric(ticker string) Binding {
	if b, ok := rsiPullbackRegistry[ticker]; ok {
		return b
	}
	return rsiPullbackBindingFor(ticker, core.DefaultParams)
}
