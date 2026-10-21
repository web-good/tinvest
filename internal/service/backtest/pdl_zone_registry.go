package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_zone/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_zone/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// pdlZoneBindingFor builds a pdl_zone binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
func pdlZoneBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse pdl_zone params: %w", err)
			}
			return p, nil
		},
	}
}

// pdlZoneRegistry maps tickers with their own package to their Minutes5 literal.
var pdlZoneRegistry = map[string]func() core.Params{
	"NVTK": nvtk.Params,
}

// PDLZoneLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. pdl_zone runs on Minutes5 only; other timeframes are rejected.
func PDLZoneLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes5 {
		return Binding{}, fmt.Errorf("pdl_zone runs on Minutes5, not %s", interval)
	}
	defaults, ok := pdlZoneRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return pdlZoneBindingFor(ticker, interval, defaults), nil
}
