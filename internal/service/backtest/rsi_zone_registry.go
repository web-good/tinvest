package backtest

import (
	"encoding/json"
	"fmt"

	rsizoneafks "tinvest/internal/service/trading_strategy/rsi_zone/strategy/afks"
	rsizonebaza "tinvest/internal/service/trading_strategy/rsi_zone/strategy/baza"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	rsizonedias "tinvest/internal/service/trading_strategy/rsi_zone/strategy/dias"
	rsizonedomrf "tinvest/internal/service/trading_strategy/rsi_zone/strategy/domrf"
	rsizoneivat "tinvest/internal/service/trading_strategy/rsi_zone/strategy/ivat"
	rsizonelent "tinvest/internal/service/trading_strategy/rsi_zone/strategy/lent"
	rsizonemdmg "tinvest/internal/service/trading_strategy/rsi_zone/strategy/mdmg"
	rsizonevsmo "tinvest/internal/service/trading_strategy/rsi_zone/strategy/vsmo"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// rsiZoneBindingFor builds an rsi_zone binding for one ticker whose baseline params come from
// defaults: a calibrated ticker package, or core.DefaultParams for an unregistered ticker.
func rsiZoneBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse rsi_zone params: %w", err)
			}
			return p, nil
		},
	}
}

// rsiZoneRegistry gives every calibrated ticker its own parameter package; the calibration
// write-up lives in the package doc and in data/params/rsi_zone/<ticker>/.
var rsiZoneRegistry = map[string]Binding{
	rsizoneafks.Ticker:  rsiZoneBindingFor(rsizoneafks.Ticker, rsizoneafks.DefaultParams),
	rsizonebaza.Ticker:  rsiZoneBindingFor(rsizonebaza.Ticker, rsizonebaza.DefaultParams),
	rsizonedias.Ticker:  rsiZoneBindingFor(rsizonedias.Ticker, rsizonedias.DefaultParams),
	rsizonedomrf.Ticker: rsiZoneBindingFor(rsizonedomrf.Ticker, rsizonedomrf.DefaultParams),
	rsizoneivat.Ticker:  rsiZoneBindingFor(rsizoneivat.Ticker, rsizoneivat.DefaultParams),
	rsizonelent.Ticker:  rsiZoneBindingFor(rsizonelent.Ticker, rsizonelent.DefaultParams),
	rsizonemdmg.Ticker:  rsiZoneBindingFor(rsizonemdmg.Ticker, rsizonemdmg.DefaultParams),
	rsizonevsmo.Ticker:  rsiZoneBindingFor(rsizonevsmo.Ticker, rsizonevsmo.DefaultParams),
}

// RSIZoneLookupOrGeneric returns the registered rsi_zone binding for a ticker, or a generic
// binding bound to that ticker (with core.DefaultParams) when none is registered.
func RSIZoneLookupOrGeneric(ticker string) Binding {
	if b, ok := rsiZoneRegistry[ticker]; ok {
		return b
	}
	return rsiZoneBindingFor(ticker, core.DefaultParams)
}
