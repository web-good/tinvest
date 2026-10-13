package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/enpg"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/gazp"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/magn"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/moex"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/nlmk"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/plzl"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/rual"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/tbank"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/vsmo"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// dropBounceBindingFor builds a drop_bounce binding for one ticker on one timeframe whose baseline
// params come from defaults: a ticker package literal, or core.DefaultParams when the ticker has
// none for that timeframe.
func dropBounceBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse drop_bounce params: %w", err)
			}
			return p, nil
		},
	}
}

// dropBounceLiterals is a ticker package's literal for each supported timeframe. Registry entries
// use unkeyed composite literals, so a ticker cannot be registered without all four.
type dropBounceLiterals struct {
	minutes5, minutes15, minutes30, hour1 func() core.Params
}

// forInterval returns the literal for a timeframe; ok is false for an unsupported one.
func (l dropBounceLiterals) forInterval(interval enum.Interval) (func() core.Params, bool) {
	switch interval {
	case enum.Minutes5:
		return l.minutes5, true
	case enum.Minutes15:
		return l.minutes15, true
	case enum.Minutes30:
		return l.minutes30, true
	case enum.Hour1:
		return l.hour1, true
	}
	return nil, false
}

// genericDropBounce runs every timeframe on core.DefaultParams: tickers without a package.
var genericDropBounce = dropBounceLiterals{core.DefaultParams, core.DefaultParams, core.DefaultParams, core.DefaultParams}

// dropBounceRegistry maps tickers with their own package to their literals, one per timeframe:
// a literal is tuned for its own timeframe and never leaks into another. The packages calibrated
// on the pre-2026-10-12 rules were deleted; a ticker comes back here with a calibration on the
// volume-spike rules.
var dropBounceRegistry = map[string]dropBounceLiterals{
	enpg.Ticker:  {enpg.ParamsMinutes5, enpg.ParamsMinutes15, enpg.ParamsMinutes30, enpg.ParamsHour1},
	gazp.Ticker:  {gazp.ParamsMinutes5, gazp.ParamsMinutes15, gazp.ParamsMinutes30, gazp.ParamsHour1},
	magn.Ticker:  {magn.ParamsMinutes5, magn.ParamsMinutes15, magn.ParamsMinutes30, magn.ParamsHour1},
	moex.Ticker:  {moex.ParamsMinutes5, moex.ParamsMinutes15, moex.ParamsMinutes30, moex.ParamsHour1},
	nlmk.Ticker:  {nlmk.ParamsMinutes5, nlmk.ParamsMinutes15, nlmk.ParamsMinutes30, nlmk.ParamsHour1},
	nvtk.Ticker:  {nvtk.ParamsMinutes5, nvtk.ParamsMinutes15, nvtk.ParamsMinutes30, nvtk.ParamsHour1},
	plzl.Ticker:  {plzl.ParamsMinutes5, plzl.ParamsMinutes15, plzl.ParamsMinutes30, plzl.ParamsHour1},
	rual.Ticker:  {rual.ParamsMinutes5, rual.ParamsMinutes15, rual.ParamsMinutes30, rual.ParamsHour1},
	tbank.Ticker: {tbank.ParamsMinutes5, tbank.ParamsMinutes15, tbank.ParamsMinutes30, tbank.ParamsHour1},
	vsmo.Ticker:  {vsmo.ParamsMinutes5, vsmo.ParamsMinutes15, vsmo.ParamsMinutes30, vsmo.ParamsHour1},
}

// DropBounceLookupOrGeneric returns the binding for a ticker on a timeframe: the ticker package's
// literal for that timeframe, or core.DefaultParams when the ticker has no package. Partial JSON
// overrides layer over whichever defaults apply. drop_bounce runs on Minutes5, Minutes15,
// Minutes30 and Hour1; other timeframes are rejected.
func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	literals, ok := dropBounceRegistry[ticker]
	if !ok {
		literals = genericDropBounce
	}
	defaults, ok := literals.forInterval(interval)
	if !ok {
		return Binding{}, fmt.Errorf("drop_bounce runs on Minutes5, Minutes15, Minutes30 or Hour1, not %s", interval)
	}
	return dropBounceBindingFor(ticker, interval, defaults), nil
}
