package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// dropBounceBindingFor builds a drop_bounce binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
func dropBounceBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse drop_bounce params: %w", err)
			}
			return p, nil
		},
	}
}

// dropBounceRegistry maps tickers with their own package to their 15m literal. Empty until a
// ticker is calibrated.
var dropBounceRegistry = map[string]func() core.Params{}

// DropBounceLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when there is none. Partial JSON overrides layer over whichever defaults
// apply. drop_bounce runs on Minutes15 only; other timeframes are rejected.
func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes15 {
		return Binding{}, fmt.Errorf("drop_bounce runs on Minutes15 only, not %s", interval)
	}
	defaults := core.DefaultParams
	if d, ok := dropBounceRegistry[ticker]; ok {
		defaults = d
	}
	return dropBounceBindingFor(ticker, defaults), nil
}
