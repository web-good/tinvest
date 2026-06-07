package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/levels/strategy/core"
	levelsrusal "tinvest/internal/service/trading_strategy/levels/strategy/rusal"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// levelsBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All levels tickers share the core engine; only ticker + defaults differ.
func levelsBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse params: %w", err)
			}
			return p, nil
		},
	}
}

var levelsRegistry = map[string]Binding{
	levelsrusal.Ticker: levelsBindingFor(levelsrusal.Ticker, levelsrusal.DefaultParams),
}

// genericLevelsDefaults are neutral baseline params for tickers without a dedicated
// levels config. Intentionally independent of levelsrusal.DefaultParams so that
// calibrating RUAL never drifts the generic baseline — mirroring the genericDefaults
// rationale in registry.go. Starting values match what RUAL currently has.
func genericLevelsDefaults() core.Params {
	return core.Params{
		ProfileWindow:     120,
		ProfileBins:       40,
		HVNFactor:         1.5,
		ATRPeriod:         14,
		LevelTolATR:       0.5,
		RoomATR:           2.0,
		MaxExtensionATR:   1.0,
		SLMult:            1.0,
		SwingLowWindow:    10,
		TrailMult:         2.5,
		ChandelierWindow:  20,
		TrailArmATR:       1.0,
		BreakoutLookback:  10,
		TrendFilterPeriod: 50,
		TrendSlopeTol:     0.0,
		ConfirmCloseFrac:  0.5,
		MinATRFrac:        0.003,
		MinRR:             1.5,
	}
}

// LevelsLookupOrGeneric returns the registered levels binding for a ticker, or a
// generic binding bound to that ticker (with genericLevelsDefaults) when none is
// registered. This lets the backtest command validate the levels strategy on any
// ticker without a dedicated package.
func LevelsLookupOrGeneric(ticker string) Binding {
	if b, ok := levelsRegistry[ticker]; ok {
		return b
	}
	return levelsBindingFor(ticker, genericLevelsDefaults)
}
