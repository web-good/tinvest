package backtest

import (
	"encoding/json"
	"fmt"
	"time"

	"tinvest/internal/domain/backtest"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/afks"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/gazp"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/sber"
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

// pdlReclaimRegistry maps tickers with their own package to their Minutes15 literal. A ticker
// comes here with its calibration; GAZP is here before it, so its formations and filters can be
// set per ticker.
var pdlReclaimRegistry = map[string]func() core.Params{
	"AFKS": afks.Params,
	"GAZP": gazp.Params,
	"NVTK": nvtk.Params,
	"SBER": sber.Params,
}

// PDLReclaimLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. pdl_reclaim runs on Minutes15 and Minutes5 (the same literal: bar counts such as
// HoldBars and VolBars stay in bars of the run); other timeframes are rejected.
func PDLReclaimLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes15 && interval != enum.Minutes5 {
		return Binding{}, fmt.Errorf("pdl_reclaim runs on Minutes15 or Minutes5, not %s", interval)
	}
	defaults, ok := pdlReclaimRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return pdlReclaimBindingFor(ticker, interval, defaults), nil
}

// PDLReclaimWithIndex makes every strategy b builds read the market index dailies (IMOEX) for the
// IndexTrendEMA filter. The engine hands a strategy only its own dailies, so the index series is
// attached at build time; the strategy reads only the index dailies of dates before the current
// bar's.
func PDLReclaimWithIndex(b Binding, index []backtest.Candle) Binding {
	times := make([]time.Time, len(index))
	closes := make([]float64, len(index))
	for i, c := range index {
		times[i], closes[i] = c.Time, c.Close
	}
	build := b.Build
	b.Build = func(params any) strategy.Strategy {
		return build(params).(*core.Strategy).WithIndex(times, closes)
	}
	return b
}
