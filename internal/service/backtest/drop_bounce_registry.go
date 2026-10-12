package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// dropBounceBindingFor builds a drop_bounce binding for one ticker on one timeframe whose baseline
// params come from defaults: a ticker package literal, or core.DefaultParams when the ticker has
// none for that timeframe.
func dropBounceBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
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

// dropBounceLiterals is a ticker package's literal for each supported timeframe. Registry entries
// use unkeyed composite literals, so a ticker cannot be registered without all four.
type dropBounceLiterals struct {
	minutes5, minutes15, minutes30, hour1 func() core.Params
}

// forInterval returns the literal for a timeframe; ok is false for an unsupported one.
func (l dropBounceLiterals) forInterval(interval enum.Interval) (func() core.Params, bool) {
	switch interval {
	case enum.Minutes5:
		return l.minutes5, true
	case enum.Minutes15:
		return l.minutes15, true
	case enum.Minutes30:
		return l.minutes30, true
	case enum.Hour1:
		return l.hour1, true
	}
	return nil, false
}

// genericDropBounce runs every timeframe on core.DefaultParams: tickers without a package.
var genericDropBounce = dropBounceLiterals{core.DefaultParams, core.DefaultParams, core.DefaultParams, core.DefaultParams}

// dropBounceRegistry maps tickers with their own package to their literals, one per timeframe:
// a literal is tuned for its own timeframe and never leaks into another. Empty since the
// 2026-10-12 volume-spike rework: the packages calibrated on the old rules were deleted, and a
// ticker comes back here only after a calibration on the new ones.
var dropBounceRegistry = map[string]dropBounceLiterals{}

// DropBounceLookupOrGeneric returns the binding for a ticker on a timeframe: the ticker package's
// literal for that timeframe, or core.DefaultParams when the ticker has no package. Partial JSON
// overrides layer over whichever defaults apply. drop_bounce runs on Minutes5, Minutes15,
// Minutes30 and Hour1; other timeframes are rejected.
func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	literals, ok := dropBounceRegistry[ticker]
	if !ok {
		literals = genericDropBounce
	}
	defaults, ok := literals.forInterval(interval)
	if !ok {
		return Binding{}, fmt.Errorf("drop_bounce runs on Minutes5, Minutes15, Minutes30 or Hour1, not %s", interval)
	}
	return dropBounceBindingFor(ticker, interval, defaults), nil
}
