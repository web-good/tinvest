package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_rsi_m30/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_rsi_m30/strategy/reni"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// pdlRSIM30BindingFor builds an independent pdl_rsi_m30 binding for one ticker.
func pdlRSIM30BindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults()
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse pdl_rsi_m30 params: %w", err)
			}
			return p, nil
		},
	}
}

var pdlRSIM30Registry = map[string]func() core.Params{
	"RENI": reni.Params,
}

// PDLRSIM30LookupOrGeneric returns the M30 binding for a ticker. RENI uses its baseline package;
// other tickers use core defaults. Partial JSON overrides layer over those defaults.
func PDLRSIM30LookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes30 {
		return Binding{}, fmt.Errorf("pdl_rsi_m30 runs on Minutes30, not %s", interval)
	}
	defaults, ok := pdlRSIM30Registry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return pdlRSIM30BindingFor(ticker, interval, defaults), nil
}
