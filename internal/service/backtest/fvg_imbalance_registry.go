package backtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/bspb"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/domrf"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/gazp"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/lkoh"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sber"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sngs"
	tickerT "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/t"
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
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&p); err != nil {
				return nil, fmt.Errorf("backtest: parse FVG imbalance params: %w", err)
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				if err == nil {
					err = fmt.Errorf("multiple JSON values")
				}
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
	"BSPB":  bspb.Params,
	"DOMRF": domrf.Params,
	"GAZP":  gazp.Params,
	"LKOH":  lkoh.Params,
	"SBER":  sber.Params,
	"SNGS":  sngs.Params,
	"T":     tickerT.Params,
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
