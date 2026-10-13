package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/enpg"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// dropBounceBindingFor builds a drop_bounce binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
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

// dropBounceRegistry maps tickers with their own package to their Hour1 literal. The packages
// calibrated on the volume-spike rules (15m) were deleted on 2026-10-13 with the rewrite to the
// 1h RSI-zone rules; a ticker comes back here with a calibration on the new rules
// (skill drop-bounce-calibrate).
var dropBounceRegistry = map[string]func() core.Params{
	"ENPG": enpg.Params,
}

// DropBounceLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. drop_bounce runs on Hour1 only; other timeframes are rejected.
func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Hour1 {
		return Binding{}, fmt.Errorf("drop_bounce runs on Hour1 only, not %s", interval)
	}
	defaults, ok := dropBounceRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return dropBounceBindingFor(ticker, interval, defaults), nil
}
