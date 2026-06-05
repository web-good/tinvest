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
