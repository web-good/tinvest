package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	midcore "tinvest/internal/service/trading_strategy/rsi_mid/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// rsiMidBindingFor builds an rsi_mid binding for one ticker whose baseline params come from
// defaults: a calibrated ticker literal, or midcore.DefaultParams for an unregistered pair.
func rsiMidBindingFor(ticker string, defaults func() midcore.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return midcore.NewWithParams(ticker, params.(midcore.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse rsi_mid params: %w", err)
			}
			return p, nil
		},
	}
}

// rsiMidKey addresses one calibrated literal: a ticker on one candle timeframe. Every rsi_mid
// period is counted in bars, so a literal is never reused across intervals.
type rsiMidKey struct {
	ticker   string
	interval enum.Interval
}

// rsiMidRegistry holds every calibrated ticker literal, one binding per timeframe. Empty until
// the first calibration: register a ticker package as `<pkg>.Ticker: <pkg>.Literals()`.
var rsiMidRegistry = buildRSIMidRegistry(map[string]map[enum.Interval]midcore.Params{})

func buildRSIMidRegistry(byTicker map[string]map[enum.Interval]midcore.Params) map[rsiMidKey]Binding {
	reg := make(map[rsiMidKey]Binding)
	for ticker, lits := range byTicker {
		for iv, p := range lits {
			reg[rsiMidKey{ticker: ticker, interval: iv}] = rsiMidBindingFor(ticker, func() midcore.Params { return p })
		}
	}
	return reg
}

// rsiMidLookupIn returns the binding registered in reg for a ticker on a timeframe, or a generic
// binding with midcore.DefaultParams — including a calibrated ticker on a timeframe it has no
// literal for.
func rsiMidLookupIn(reg map[rsiMidKey]Binding, ticker string, interval enum.Interval) Binding {
	if b, ok := reg[rsiMidKey{ticker: ticker, interval: interval}]; ok {
		return b
	}
	return rsiMidBindingFor(ticker, midcore.DefaultParams)
}

// RSIMidLookupOrGeneric returns the rsi_mid binding for a ticker on a timeframe.
func RSIMidLookupOrGeneric(ticker string, interval enum.Interval) Binding {
	return rsiMidLookupIn(rsiMidRegistry, ticker, interval)
}
