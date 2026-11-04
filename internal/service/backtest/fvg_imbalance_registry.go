package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/gazp"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func fvgImbalanceBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, interval.ToTimeDuration(), params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults()
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse FVG imbalance params: %w", err)
			}
			if err := core.ValidateParams(p); err != nil {
				return nil, fmt.Errorf("backtest: validate FVG imbalance params: %w", err)
			}
			return p, nil
		},
	}
}

var fvgImbalanceRegistry = map[string]func() core.Params{
	"GAZP": gazp.Params,
}

// FVGImbalanceLookupOrGeneric returns a backtest-only M30 binding using the ticker
// literal when registered, or core defaults otherwise. Partial JSON overlays
// those defaults and must pass core parameter validation.
func FVGImbalanceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes30 {
		return Binding{}, fmt.Errorf("FVG imbalance runs on Minutes30, not %s", interval)
	}
	defaults, ok := fvgImbalanceRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return fvgImbalanceBindingFor(ticker, interval, defaults), nil
}
