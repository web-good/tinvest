package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// RSIZoneLookupOrGeneric returns an rsi_zone binding bound to the ticker. The strategy is
// ticker-agnostic and there are no per-ticker packages yet (calibration pending), so every
// ticker gets the generic defaults.
func RSIZoneLookupOrGeneric(ticker string) Binding {
	return Binding{
		DefaultParams: func() any { return core.DefaultParams() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := core.DefaultParams() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse rsi_zone params: %w", err)
			}
			return p, nil
		},
	}
}
