package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/rsi_ema/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// rsiEMABindingFor builds a Binding for a ticker on the rsi_ema engine. The strategy is
// ticker-agnostic; only the ticker label differs, so a single generic default suffices until
// calibration proves per-ticker params are needed.
func rsiEMABindingFor(ticker string) Binding {
	return Binding{
		DefaultParams: func() any { return core.DefaultParams() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := core.DefaultParams() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse rsi_ema params: %w", err)
			}
			return p, nil
		},
	}
}

// RSIEMALookupOrGeneric returns an rsi_ema binding bound to the ticker. There are no
// per-ticker packages yet (calibration pending), so every ticker gets the generic defaults.
func RSIEMALookupOrGeneric(ticker string) Binding {
	return rsiEMABindingFor(ticker)
}
