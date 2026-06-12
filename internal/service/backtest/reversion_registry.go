package backtest

import (
	"encoding/json"
	"fmt"

	reversionafks "tinvest/internal/service/trading_strategy/reversion/strategy/afks"
	"tinvest/internal/service/trading_strategy/reversion/strategy/core"
	reversiongazp "tinvest/internal/service/trading_strategy/reversion/strategy/gazp"
	reversionmdmg "tinvest/internal/service/trading_strategy/reversion/strategy/mdmg"
	reversionnvtk "tinvest/internal/service/trading_strategy/reversion/strategy/nvtk"
	reversionplzl "tinvest/internal/service/trading_strategy/reversion/strategy/plzl"
	reversionrusal "tinvest/internal/service/trading_strategy/reversion/strategy/rusal"
	reversionsber "tinvest/internal/service/trading_strategy/reversion/strategy/sber"
	reversionydex "tinvest/internal/service/trading_strategy/reversion/strategy/ydex"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// reversionBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All reversion tickers share the core engine; only ticker + defaults differ.
func reversionBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse params: %w", err)
			}
			return p, nil
		},
	}
}

var reversionRegistry = map[string]Binding{
	reversionrusal.Ticker: reversionBindingFor(reversionrusal.Ticker, reversionrusal.DefaultParams),
	reversionafks.Ticker:  reversionBindingFor(reversionafks.Ticker, reversionafks.DefaultParams),
	reversionydex.Ticker:  reversionBindingFor(reversionydex.Ticker, reversionydex.DefaultParams),
	reversionplzl.Ticker:  reversionBindingFor(reversionplzl.Ticker, reversionplzl.DefaultParams),
	reversionsber.Ticker:  reversionBindingFor(reversionsber.Ticker, reversionsber.DefaultParams),
	reversiongazp.Ticker:  reversionBindingFor(reversiongazp.Ticker, reversiongazp.DefaultParams),
	reversionnvtk.Ticker:  reversionBindingFor(reversionnvtk.Ticker, reversionnvtk.DefaultParams),
	reversionmdmg.Ticker:  reversionBindingFor(reversionmdmg.Ticker, reversionmdmg.DefaultParams),
}

// genericReversionDefaults are neutral baseline params for tickers without a dedicated
// reversion config. Intentionally independent of any per-ticker defaults so calibrating
// one ticker never drifts the generic baseline.
func genericReversionDefaults() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20, StochOverbought: 80,
		ATRPeriod: 14, ATRMult: 1.0,
	}
}

// ReversionLookupOrGeneric returns the registered reversion binding for a ticker, or a
// generic binding bound to that ticker (with genericReversionDefaults) when none is
// registered.
func ReversionLookupOrGeneric(ticker string) Binding {
	if b, ok := reversionRegistry[ticker]; ok {
		return b
	}
	return reversionBindingFor(ticker, genericReversionDefaults)
}
