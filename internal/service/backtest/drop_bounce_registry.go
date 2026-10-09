package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	dropbouncegazp "tinvest/internal/service/trading_strategy/drop_bounce/strategy/gazp"
	dropbouncetbank "tinvest/internal/service/trading_strategy/drop_bounce/strategy/tbank"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// dropBounceIntervals are the timeframes drop_bounce runs on.
var dropBounceIntervals = map[enum.Interval]bool{enum.Minutes5: true, enum.Minutes15: true, enum.Hour1: true}

// dropBounceBindingFor builds a drop_bounce binding for one ticker on one timeframe whose baseline
// params come from defaults: a ticker package literal, or core.DefaultParams when the ticker has
// none for that timeframe.
func dropBounceBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithStep(ticker, step, params.(core.Params))
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

// dropBounceRegistry maps tickers with their own package to their literal per timeframe.
var dropBounceRegistry = map[string]map[enum.Interval]func() core.Params{
	dropbouncegazp.Ticker: {
		enum.Minutes5:  dropbouncegazp.DefaultParams5m,
		enum.Minutes15: dropbouncegazp.DefaultParams15m,
		enum.Hour1:     dropbouncegazp.DefaultParams1h,
	},
	dropbouncetbank.Ticker: {
		enum.Minutes5:  dropbouncetbank.DefaultParams5m,
		enum.Minutes15: dropbouncetbank.DefaultParams15m,
		enum.Hour1:     dropbouncetbank.DefaultParams1h,
	},
}

// DropBounceLookupOrGeneric returns the binding for a ticker on a timeframe: the ticker package's
// literal for that timeframe, or core.DefaultParams when there is none. A literal of another
// timeframe is never borrowed — its bar counts and thresholds mean something else. Partial JSON
// overrides layer over whichever defaults apply. Timeframes other than Minutes5, Minutes15 and
// Hour1 are rejected.
func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if !dropBounceIntervals[interval] {
		return Binding{}, fmt.Errorf("drop_bounce runs on Minutes5, Minutes15 or Hour1, not %s", interval)
	}
	if defaults, ok := dropBounceRegistry[ticker][interval]; ok {
		return dropBounceBindingFor(ticker, interval, defaults), nil
	}
	return dropBounceBindingFor(ticker, interval, core.DefaultParams), nil
}
