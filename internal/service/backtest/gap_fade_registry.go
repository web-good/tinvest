package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// GapFadeLookupOrGeneric returns the gap_fade binding for a ticker. gap_fade parameters are
// shared across the universe (cmd/gapwf), so there are no ticker packages: every ticker starts
// from core.DefaultParams and partial JSON overrides layer over them.
func GapFadeLookupOrGeneric(ticker string) Binding {
	return Binding{
		DefaultParams: func() any { return core.DefaultParams() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := core.DefaultParams()
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse gap_fade params: %w", err)
			}
			return p, nil
		},
	}
}
