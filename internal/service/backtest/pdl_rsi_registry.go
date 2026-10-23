package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// pdlRSIBindingFor builds a pdl_rsi binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
func pdlRSIBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse pdl_rsi params: %w", err)
			}
			return p, nil
		},
	}
}

// pdlRSIRegistry maps tickers with their own package to their Minutes15 literal.
var pdlRSIRegistry = map[string]func() core.Params{
	"NVTK": nvtk.Params,
}

// PDLRSILookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. pdl_rsi runs on Minutes15 only; other timeframes are rejected.
func PDLRSILookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes15 {
		return Binding{}, fmt.Errorf("pdl_rsi runs on Minutes15, not %s", interval)
	}
	defaults, ok := pdlRSIRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return pdlRSIBindingFor(ticker, interval, defaults), nil
}
