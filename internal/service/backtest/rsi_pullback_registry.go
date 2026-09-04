package backtest

import (
	"encoding/json"
	"fmt"

	rsipullbackafks "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/afks"
	rsipullbackastr "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/astr"
	rsipullbackbanep "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/banep"
	rsipullbackbspb "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/bspb"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	rsipullbackdias "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/dias"
	rsipullbackdomrf "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	rsipullbackelfv "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/elfv"
	rsipullbackfesh "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
	rsipullbackgazp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	rsipullbackhead "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/head"
	rsipullbackivat "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ivat"
	rsipullbacklent "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lent"
	rsipullbacklsngp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lsngp"
	rsipullbacknkhp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nkhp"
	rsipullbacknvtk "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	rsipullbackreni "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/reni"
	rsipullbacksibn "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sibn"
	rsipullbacksngsp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sngsp"
	rsipullbacksofl "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sofl"
	rsipullbacksvav "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svav"
	rsipullbacktbank "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tbank"
	rsipullbacktgka "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tgka"
	rsipullbackugld "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ugld"
	rsipullbackuwgn "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/uwgn"
	rsipullbackvsmo "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/vsmo"
	rsipullbackwush "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"
	rsipullbackydex "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ydex"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// rsiPullbackBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All rsi_pullback tickers share the core engine; only ticker + defaults differ.
func rsiPullbackBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse rsi_pullback params: %w", err)
			}
			return p, nil
		},
	}
}

// rsiPullbackRegistry gives every tracked ticker its own parameter package. An entry is in one of
// three states, documented on the package itself: baseline-tracking (DefaultParams() as-is,
// calibration pending), calibrated (an explicit literal from that ticker's own walk-forward run,
// e.g. gazp, tbank), or seeded from another ticker's literal as an explicit transferability
// hypothesis, not a claim of being tuned — see docs/rsi_pullback/strategy.md §8.0.1.
var rsiPullbackRegistry = map[string]Binding{
	rsipullbackafks.Ticker:  rsiPullbackBindingFor(rsipullbackafks.Ticker, rsipullbackafks.DefaultParams),
	rsipullbackastr.Ticker:  rsiPullbackBindingFor(rsipullbackastr.Ticker, rsipullbackastr.DefaultParams),
	rsipullbackbanep.Ticker: rsiPullbackBindingFor(rsipullbackbanep.Ticker, rsipullbackbanep.DefaultParams),
	rsipullbackbspb.Ticker:  rsiPullbackBindingFor(rsipullbackbspb.Ticker, rsipullbackbspb.DefaultParams),
	rsipullbackdias.Ticker:  rsiPullbackBindingFor(rsipullbackdias.Ticker, rsipullbackdias.DefaultParams),
	rsipullbackdomrf.Ticker: rsiPullbackBindingFor(rsipullbackdomrf.Ticker, rsipullbackdomrf.DefaultParams),
	rsipullbackelfv.Ticker:  rsiPullbackBindingFor(rsipullbackelfv.Ticker, rsipullbackelfv.DefaultParams),
	rsipullbackfesh.Ticker:  rsiPullbackBindingFor(rsipullbackfesh.Ticker, rsipullbackfesh.DefaultParams),
	rsipullbackgazp.Ticker:  rsiPullbackBindingFor(rsipullbackgazp.Ticker, rsipullbackgazp.DefaultParams),
	rsipullbackhead.Ticker:  rsiPullbackBindingFor(rsipullbackhead.Ticker, rsipullbackhead.DefaultParams),
	rsipullbackivat.Ticker:  rsiPullbackBindingFor(rsipullbackivat.Ticker, rsipullbackivat.DefaultParams),
	rsipullbacklent.Ticker:  rsiPullbackBindingFor(rsipullbacklent.Ticker, rsipullbacklent.DefaultParams),
	rsipullbacklsngp.Ticker: rsiPullbackBindingFor(rsipullbacklsngp.Ticker, rsipullbacklsngp.DefaultParams),
	rsipullbacknkhp.Ticker:  rsiPullbackBindingFor(rsipullbacknkhp.Ticker, rsipullbacknkhp.DefaultParams),
	rsipullbacknvtk.Ticker:  rsiPullbackBindingFor(rsipullbacknvtk.Ticker, rsipullbacknvtk.DefaultParams),
	rsipullbackreni.Ticker:  rsiPullbackBindingFor(rsipullbackreni.Ticker, rsipullbackreni.DefaultParams),
	rsipullbacksibn.Ticker:  rsiPullbackBindingFor(rsipullbacksibn.Ticker, rsipullbacksibn.DefaultParams),
	rsipullbacksngsp.Ticker: rsiPullbackBindingFor(rsipullbacksngsp.Ticker, rsipullbacksngsp.DefaultParams),
	rsipullbacksofl.Ticker:  rsiPullbackBindingFor(rsipullbacksofl.Ticker, rsipullbacksofl.DefaultParams),
	rsipullbacksvav.Ticker:  rsiPullbackBindingFor(rsipullbacksvav.Ticker, rsipullbacksvav.DefaultParams),
	rsipullbacktbank.Ticker: rsiPullbackBindingFor(rsipullbacktbank.Ticker, rsipullbacktbank.DefaultParams),
	rsipullbacktgka.Ticker:  rsiPullbackBindingFor(rsipullbacktgka.Ticker, rsipullbacktgka.DefaultParams),
	rsipullbackugld.Ticker:  rsiPullbackBindingFor(rsipullbackugld.Ticker, rsipullbackugld.DefaultParams),
	rsipullbackuwgn.Ticker:  rsiPullbackBindingFor(rsipullbackuwgn.Ticker, rsipullbackuwgn.DefaultParams),
	rsipullbackvsmo.Ticker:  rsiPullbackBindingFor(rsipullbackvsmo.Ticker, rsipullbackvsmo.DefaultParams),
	rsipullbackwush.Ticker:  rsiPullbackBindingFor(rsipullbackwush.Ticker, rsipullbackwush.DefaultParams),
	rsipullbackydex.Ticker:  rsiPullbackBindingFor(rsipullbackydex.Ticker, rsipullbackydex.DefaultParams),
}

// RSIPullbackLookupOrGeneric returns the registered rsi_pullback binding for a ticker, or a
// generic binding bound to that ticker (with core.DefaultParams) when none is registered.
func RSIPullbackLookupOrGeneric(ticker string) Binding {
	if b, ok := rsiPullbackRegistry[ticker]; ok {
		return b
	}
	return rsiPullbackBindingFor(ticker, core.DefaultParams)
}
