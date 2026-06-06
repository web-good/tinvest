package backtest

import (
	"encoding/json"
	"fmt"
	"reflect"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
	"tinvest/internal/service/trading_strategy/scalping/strategy/afks"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

// Binding adapts a concrete strategy's params to the generic engine: it builds
// the strategy from params, supplies defaults, and parses params from JSON.
type Binding struct {
	DefaultParams func() any                         // e.g. rusal.DefaultParams()
	Build         func(params any) strategy.Strategy // e.g. adaptive.NewWithParams(ticker, p)
	ParseParams   func(raw []byte) (any, error)      // JSON -> adaptive.Params
}

// bindingFor builds a Binding for a ticker whose defaults come from defaults().
// All scalping tickers share the adaptive engine; only ticker + defaults differ.
func bindingFor(ticker string, defaults func() adaptive.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return adaptive.NewWithParams(ticker, params.(adaptive.Params))
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

var registry = map[string]Binding{
	rusal.Ticker: bindingFor(rusal.Ticker, rusal.DefaultParams),
	afks.Ticker:  bindingFor(afks.Ticker, afks.DefaultParams),
}

// Lookup returns the binding registered for a ticker.
func Lookup(ticker string) (Binding, bool) {
	b, ok := registry[ticker]
	return b, ok
}

// genericDefaults are neutral baseline params for tickers without a dedicated config,
// used for basket validation. HTF filter on; quality knobs opted in at starting values.
func genericDefaults() adaptive.Params {
	// Intentionally independent of afks.DefaultParams: the generic baseline must not drift when AFKS is calibrated to ticker-specific values.
	return adaptive.Params{
		EMAPeriod: 21, ADXPeriod: 14, ADXTrendLevel: 25, ADXRangeLevel: 20,
		RSIPeriod: 14, RSITrendLevel: 45, RSIRangeLevel: 35,
		PullbackWindow: 5, DonchianPeriod: 20, ATRPeriod: 14,
		SLMult: 1.0, TrailMult: 2.5, ChandelierWindow: 20,
		EMATouchTol: 0.002, BandTol: 0.003, TrendFilterPeriod: 100,
		TrailArmATR: 1.0, ADXMargin: 2.0, MinRR: 1.5, MinATRFrac: 0.003,
	}
}

// LookupOrGeneric returns the registered binding for a ticker, or a generic binding
// bound to that ticker (with genericDefaults) when none is registered. This lets the
// backtest command validate the strategy on any ticker without a dedicated package.
func LookupOrGeneric(ticker string) Binding {
	if b, ok := registry[ticker]; ok {
		return b
	}
	return bindingFor(ticker, genericDefaults)
}

// ParamRows reflects a params struct into report rows (field name -> value).
func ParamRows(params any) []backtest.ParamLine {
	v := reflect.ValueOf(params)
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	rows := make([]backtest.ParamLine, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		rows = append(rows, backtest.ParamLine{
			Name:  t.Field(i).Name,
			Value: fmt.Sprintf("%v", v.Field(i).Interface()),
		})
	}
	return rows
}
