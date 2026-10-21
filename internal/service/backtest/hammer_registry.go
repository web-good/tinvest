package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/hammer/strategy/core"
	"tinvest/internal/service/trading_strategy/hammer/strategy/gazp"
	"tinvest/internal/service/trading_strategy/hammer/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/hammer/strategy/sber"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// hammerBindingFor builds a hammer binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
func hammerBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse hammer params: %w", err)
			}
			return p, nil
		},
	}
}

// hammerRegistry maps tickers with their own package to their literal (the same on Hour1 and
// Minutes30).
var hammerRegistry = map[string]func() core.Params{
	"GAZP": gazp.Params,
	"SBER": sber.Params,
	"NVTK": nvtk.Params,
}

// HammerLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. hammer runs on Hour1 and Minutes30; other timeframes are rejected.
func HammerLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Hour1 && interval != enum.Minutes30 {
		return Binding{}, fmt.Errorf("hammer runs on Hour1 or Minutes30, not %s", interval)
	}
	defaults, ok := hammerRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return hammerBindingFor(ticker, interval, defaults), nil
}
