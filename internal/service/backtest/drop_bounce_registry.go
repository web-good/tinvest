package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/enpg"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/gazp"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/gmkn"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/ivat"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/lkoh"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/plzl"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/rosn"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/sber"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/tatn"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// dropBounceBindingFor builds a drop_bounce binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
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

// dropBounceRegistry maps tickers with their own package to their Minutes30 literal. The packages
// calibrated on the volume-spike rules (15m) were deleted on 2026-10-13 with the rewrite to the
// 1h RSI-zone rules; since 2026-10-14 the rules run on 30m (hourly signal, 30m MACD
// confirmation) and a ticker comes back here with a calibration on them (skill
// drop-bounce-calibrate).
var dropBounceRegistry = map[string]func() core.Params{
	"ENPG": enpg.Params,
	"GAZP": gazp.Params,
	"GMKN": gmkn.Params,
	"IVAT": ivat.Params,
	"LKOH": lkoh.Params,
	"NVTK": nvtk.Params,
	"PLZL": plzl.Params,
	"ROSN": rosn.Params,
	"SBER": sber.Params,
	"TATN": tatn.Params,
}

// DropBounceLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. drop_bounce runs on Minutes30 only (hourly signal built from the 30m bars, 30m MACD
// confirmation); other timeframes are rejected.
func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes30 {
		return Binding{}, fmt.Errorf("drop_bounce runs on Minutes30 only, not %s", interval)
	}
	defaults, ok := dropBounceRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return dropBounceBindingFor(ticker, interval, defaults), nil
}
