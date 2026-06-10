package backtest

import (
	"encoding/json"
	"fmt"

	momentumafks "tinvest/internal/service/trading_strategy/momentum/strategy/afks"
	"tinvest/internal/service/trading_strategy/momentum/strategy/core"
	momentumgazp "tinvest/internal/service/trading_strategy/momentum/strategy/gazp"
	momentummdmg "tinvest/internal/service/trading_strategy/momentum/strategy/mdmg"
	momentumnvtk "tinvest/internal/service/trading_strategy/momentum/strategy/nvtk"
	momentumplzl "tinvest/internal/service/trading_strategy/momentum/strategy/plzl"
	momentumrusal "tinvest/internal/service/trading_strategy/momentum/strategy/rusal"
	momentumsber "tinvest/internal/service/trading_strategy/momentum/strategy/sber"
	momentumydex "tinvest/internal/service/trading_strategy/momentum/strategy/ydex"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// momentumBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All momentum tickers share the core engine; only ticker + defaults differ.
func momentumBindingFor(ticker string, defaults func() core.Params) Binding {
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

var momentumRegistry = map[string]Binding{
	momentumrusal.Ticker: momentumBindingFor(momentumrusal.Ticker, momentumrusal.DefaultParams),
	momentumafks.Ticker:  momentumBindingFor(momentumafks.Ticker, momentumafks.DefaultParams),
	momentumydex.Ticker:  momentumBindingFor(momentumydex.Ticker, momentumydex.DefaultParams),
	momentumplzl.Ticker:  momentumBindingFor(momentumplzl.Ticker, momentumplzl.DefaultParams),
	momentumsber.Ticker:  momentumBindingFor(momentumsber.Ticker, momentumsber.DefaultParams),
	momentumgazp.Ticker:  momentumBindingFor(momentumgazp.Ticker, momentumgazp.DefaultParams),
	momentumnvtk.Ticker:  momentumBindingFor(momentumnvtk.Ticker, momentumnvtk.DefaultParams),
	momentummdmg.Ticker:  momentumBindingFor(momentummdmg.Ticker, momentummdmg.DefaultParams),
}

// genericMomentumDefaults are neutral baseline params for tickers without a dedicated
// momentum config. Intentionally independent of rusal.DefaultParams so calibrating RUAL
// never drifts the generic baseline — mirroring genericLevelsDefaults.
func genericMomentumDefaults() core.Params {
	return core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
}

// MomentumLookupOrGeneric returns the registered momentum binding for a ticker, or a
// generic binding bound to that ticker (with genericMomentumDefaults) when none is
// registered. This lets the backtest command validate the momentum strategy on any
// ticker without a dedicated package.
func MomentumLookupOrGeneric(ticker string) Binding {
	if b, ok := momentumRegistry[ticker]; ok {
		return b
	}
	return momentumBindingFor(ticker, genericMomentumDefaults)
}
