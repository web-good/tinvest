package backtest

import (
	"encoding/json"
	"fmt"

	rsizoneafks "tinvest/internal/service/trading_strategy/rsi_zone/strategy/afks"
	rsizoneaflt "tinvest/internal/service/trading_strategy/rsi_zone/strategy/aflt"
	rsizoneastr "tinvest/internal/service/trading_strategy/rsi_zone/strategy/astr"
	rsizonebane "tinvest/internal/service/trading_strategy/rsi_zone/strategy/bane"
	rsizonebaza "tinvest/internal/service/trading_strategy/rsi_zone/strategy/baza"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	rsizonedias "tinvest/internal/service/trading_strategy/rsi_zone/strategy/dias"
	rsizonedomrf "tinvest/internal/service/trading_strategy/rsi_zone/strategy/domrf"
	rsizonefixr "tinvest/internal/service/trading_strategy/rsi_zone/strategy/fixr"
	rsizoneivat "tinvest/internal/service/trading_strategy/rsi_zone/strategy/ivat"
	rsizonelent "tinvest/internal/service/trading_strategy/rsi_zone/strategy/lent"
	rsizonemdmg "tinvest/internal/service/trading_strategy/rsi_zone/strategy/mdmg"
	rsizonemoex "tinvest/internal/service/trading_strategy/rsi_zone/strategy/moex"
	rsizonemsng "tinvest/internal/service/trading_strategy/rsi_zone/strategy/msng"
	rsizonemtss "tinvest/internal/service/trading_strategy/rsi_zone/strategy/mtss"
	rsizoneragr "tinvest/internal/service/trading_strategy/rsi_zone/strategy/ragr"
	rsizonereni "tinvest/internal/service/trading_strategy/rsi_zone/strategy/reni"
	rsizonesmlt "tinvest/internal/service/trading_strategy/rsi_zone/strategy/smlt"
	rsizonesvav "tinvest/internal/service/trading_strategy/rsi_zone/strategy/svav"
	rsizonetatnp "tinvest/internal/service/trading_strategy/rsi_zone/strategy/tatnp"
	rsizonetrnfp "tinvest/internal/service/trading_strategy/rsi_zone/strategy/trnfp"
	rsizoneupro "tinvest/internal/service/trading_strategy/rsi_zone/strategy/upro"
	rsizonevsmo "tinvest/internal/service/trading_strategy/rsi_zone/strategy/vsmo"
	rsizonewush "tinvest/internal/service/trading_strategy/rsi_zone/strategy/wush"
	rsizoneydex "tinvest/internal/service/trading_strategy/rsi_zone/strategy/ydex"
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
	rsizoneragr.Ticker:  rsiZoneBindingFor(rsizoneragr.Ticker, rsizoneragr.DefaultParams),
	rsizoneupro.Ticker:  rsiZoneBindingFor(rsizoneupro.Ticker, rsizoneupro.DefaultParams),
	rsizonemtss.Ticker:  rsiZoneBindingFor(rsizonemtss.Ticker, rsizonemtss.DefaultParams),
	rsizoneaflt.Ticker:  rsiZoneBindingFor(rsizoneaflt.Ticker, rsizoneaflt.DefaultParams),
	rsizonemsng.Ticker:  rsiZoneBindingFor(rsizonemsng.Ticker, rsizonemsng.DefaultParams),
	rsizonebane.Ticker:  rsiZoneBindingFor(rsizonebane.Ticker, rsizonebane.DefaultParams),
	rsizonevsmo.Ticker:  rsiZoneBindingFor(rsizonevsmo.Ticker, rsizonevsmo.DefaultParams),
	rsizoneastr.Ticker:  rsiZoneBindingFor(rsizoneastr.Ticker, rsizoneastr.DefaultParams),
	rsizonesvav.Ticker:  rsiZoneBindingFor(rsizonesvav.Ticker, rsizonesvav.DefaultParams),
	rsizonewush.Ticker:  rsiZoneBindingFor(rsizonewush.Ticker, rsizonewush.DefaultParams),
	rsizonetrnfp.Ticker: rsiZoneBindingFor(rsizonetrnfp.Ticker, rsizonetrnfp.DefaultParams),
	rsizoneydex.Ticker:  rsiZoneBindingFor(rsizoneydex.Ticker, rsizoneydex.DefaultParams),
	rsizonesmlt.Ticker:  rsiZoneBindingFor(rsizonesmlt.Ticker, rsizonesmlt.DefaultParams),
	rsizonetatnp.Ticker: rsiZoneBindingFor(rsizonetatnp.Ticker, rsizonetatnp.DefaultParams),
	rsizonereni.Ticker:  rsiZoneBindingFor(rsizonereni.Ticker, rsizonereni.DefaultParams),
	rsizonefixr.Ticker:  rsiZoneBindingFor(rsizonefixr.Ticker, rsizonefixr.DefaultParams),
	rsizonemoex.Ticker:  rsiZoneBindingFor(rsizonemoex.Ticker, rsizonemoex.DefaultParams),
}

// RSIZoneLookupOrGeneric returns the registered rsi_zone binding for a ticker, or a generic
// binding bound to that ticker (with core.DefaultParams) when none is registered.
func RSIZoneLookupOrGeneric(ticker string) Binding {
	if b, ok := rsiZoneRegistry[ticker]; ok {
		return b
	}
	return rsiZoneBindingFor(ticker, core.DefaultParams)
}
