package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// pdlReclaimBindingFor builds a pdl_reclaim binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
func pdlReclaimBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse pdl_reclaim params: %w", err)
			}
			return p, nil
		},
	}
}

// pdlReclaimRegistry maps tickers with their own package to their Minutes5 literal. A ticker
// comes here with its calibration; GAZP, SBER and NVTK are the first in line.
var pdlReclaimRegistry = map[string]func() core.Params{}

// PDLReclaimLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. pdl_reclaim runs on Minutes5 only; other timeframes are rejected.
func PDLReclaimLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes5 {
		return Binding{}, fmt.Errorf("pdl_reclaim runs on Minutes5 only, not %s", interval)
	}
	defaults, ok := pdlReclaimRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return pdlReclaimBindingFor(ticker, interval, defaults), nil
}
