package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/aflt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/data"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/flot"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gazp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/hnfg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lent"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mdmg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mgnt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/moex"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrkc"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrkv"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/msng"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/msrs"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mtss"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/phor"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rosn"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/spbe"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/svcb"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/tgka"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vtbr"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/x5"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// pdlRSIBindingFor builds a pdl_rsi binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
func pdlRSIBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse pdl_rsi params: %w", err)
			}
			return p, nil
		},
	}
}

// pdlRSIRegistry maps tickers with their own package to their Hour1 literal.
var pdlRSIRegistry = map[string]func() core.Params{
	"AFLT": aflt.Params,
	"DATA": data.Params,
	"FLOT": flot.Params,
	"GAZP": gazp.Params,
	"HNFG": hnfg.Params,
	"LENT": lent.Params,
	"MDMG": mdmg.Params,
	"MGNT": mgnt.Params,
	"MOEX": moex.Params,
	"MRKC": mrkc.Params,
	"MRKV": mrkv.Params,
	"MSNG": msng.Params,
	"MSRS": msrs.Params,
	"MTSS": mtss.Params,
	"NVTK": nvtk.Params,
	"PHOR": phor.Params,
	"ROSN": rosn.Params,
	"SPBE": spbe.Params,
	"SVCB": svcb.Params,
	"TGKA": tgka.Params,
	"VTBR": vtbr.Params,
	"X5":   x5.Params,
}

// PDLRSILookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. pdl_rsi runs on Hour1 only; other timeframes are rejected.
func PDLRSILookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Hour1 {
		return Binding{}, fmt.Errorf("pdl_rsi runs on Hour1, not %s", interval)
	}
	defaults, ok := pdlRSIRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return pdlRSIBindingFor(ticker, interval, defaults), nil
}
