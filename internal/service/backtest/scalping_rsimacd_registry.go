package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/service/trading_strategy/scalping_rsimacd/strategy/core"
)

// scalpingRSIMACDBindingFor builds a Binding for a ticker on the scalping_rsimacd engine.
// The strategy is ticker-agnostic; only the ticker label differs, so a single generic
// default suffices until calibration proves per-ticker params are needed.
func scalpingRSIMACDBindingFor(ticker string) Binding {
	return Binding{
		DefaultParams: func() any { return core.DefaultParams() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := core.DefaultParams() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse scalping_rsimacd params: %w", err)
			}
			return p, nil
		},
	}
}

// ScalpingRSIMACDLookupOrGeneric returns a scalping_rsimacd binding bound to the ticker.
// There are no per-ticker packages yet (calibration pending), so every ticker gets the
// generic defaults.
func ScalpingRSIMACDLookupOrGeneric(ticker string) Binding {
	return scalpingRSIMACDBindingFor(ticker)
}
