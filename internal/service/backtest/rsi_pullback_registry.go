package backtest

import (
	"encoding/json"
	"fmt"

	rsipullbackafks "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/afks"
	rsipullbackaqua "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/aqua"
	rsipullbackastr "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/astr"
	rsipullbackbanep "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/banep"
	rsipullbackbspb "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/bspb"
	rsipullbackcnru "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/cnru"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	rsipullbackdias "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/dias"
	rsipullbackdomrf "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	rsipullbackelfv "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/elfv"
	rsipullbackfesh "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
	rsipullbackgazp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	rsipullbackhead "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/head"
	rsipullbackirkt "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/irkt"
	rsipullbackivat "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ivat"
	rsipullbacklent "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lent"
	rsipullbacklsngp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lsngp"
	rsipullbackmagn "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/magn"
	rsipullbackmdmg "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mdmg"
	rsipullbackmvid "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mvid"
	rsipullbacknkhp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nkhp"
	rsipullbacknvtk "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	rsipullbackragr "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ragr"
	rsipullbackreni "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/reni"
	rsipullbackrtkm "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/rtkm"
	rsipullbackrtkmp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/rtkmp"
	rsipullbacksfin "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sfin"
	rsipullbacksibn "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sibn"
	rsipullbacksngsp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sngsp"
	rsipullbacksofl "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sofl"
	rsipullbackspbe "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/spbe"
	rsipullbacksvav "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svav"
	rsipullbacksvcb "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svcb"
	rsipullbacktbank "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tbank"
	rsipullbacktgka "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tgka"
	rsipullbacktrnfp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/trnfp"
	rsipullbackugld "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ugld"
	rsipullbackuwgn "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/uwgn"
	rsipullbackvsmo "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/vsmo"
	rsipullbackwush "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"
	rsipullbackx5 "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/x5"
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
	rsipullbackaqua.Ticker:  rsiPullbackBindingFor(rsipullbackaqua.Ticker, rsipullbackaqua.DefaultParams),
	rsipullbackastr.Ticker:  rsiPullbackBindingFor(rsipullbackastr.Ticker, rsipullbackastr.DefaultParams),
	rsipullbackbanep.Ticker: rsiPullbackBindingFor(rsipullbackbanep.Ticker, rsipullbackbanep.DefaultParams),
	rsipullbackbspb.Ticker:  rsiPullbackBindingFor(rsipullbackbspb.Ticker, rsipullbackbspb.DefaultParams),
	rsipullbackcnru.Ticker:  rsiPullbackBindingFor(rsipullbackcnru.Ticker, rsipullbackcnru.DefaultParams),
	rsipullbackdias.Ticker:  rsiPullbackBindingFor(rsipullbackdias.Ticker, rsipullbackdias.DefaultParams),
	rsipullbackdomrf.Ticker: rsiPullbackBindingFor(rsipullbackdomrf.Ticker, rsipullbackdomrf.DefaultParams),
	rsipullbackelfv.Ticker:  rsiPullbackBindingFor(rsipullbackelfv.Ticker, rsipullbackelfv.DefaultParams),
	rsipullbackfesh.Ticker:  rsiPullbackBindingFor(rsipullbackfesh.Ticker, rsipullbackfesh.DefaultParams),
	rsipullbackgazp.Ticker:  rsiPullbackBindingFor(rsipullbackgazp.Ticker, rsipullbackgazp.DefaultParams),
	rsipullbackhead.Ticker:  rsiPullbackBindingFor(rsipullbackhead.Ticker, rsipullbackhead.DefaultParams),
	rsipullbackirkt.Ticker:  rsiPullbackBindingFor(rsipullbackirkt.Ticker, rsipullbackirkt.DefaultParams),
	rsipullbackivat.Ticker:  rsiPullbackBindingFor(rsipullbackivat.Ticker, rsipullbackivat.DefaultParams),
	rsipullbacklent.Ticker:  rsiPullbackBindingFor(rsipullbacklent.Ticker, rsipullbacklent.DefaultParams),
	rsipullbacklsngp.Ticker: rsiPullbackBindingFor(rsipullbacklsngp.Ticker, rsipullbacklsngp.DefaultParams),
	rsipullbackmagn.Ticker:  rsiPullbackBindingFor(rsipullbackmagn.Ticker, rsipullbackmagn.DefaultParams),
	rsipullbackmdmg.Ticker:  rsiPullbackBindingFor(rsipullbackmdmg.Ticker, rsipullbackmdmg.DefaultParams),
	rsipullbackmvid.Ticker:  rsiPullbackBindingFor(rsipullbackmvid.Ticker, rsipullbackmvid.DefaultParams),
	rsipullbacknkhp.Ticker:  rsiPullbackBindingFor(rsipullbacknkhp.Ticker, rsipullbacknkhp.DefaultParams),
	rsipullbacknvtk.Ticker:  rsiPullbackBindingFor(rsipullbacknvtk.Ticker, rsipullbacknvtk.DefaultParams),
	rsipullbackragr.Ticker:  rsiPullbackBindingFor(rsipullbackragr.Ticker, rsipullbackragr.DefaultParams),
	rsipullbackreni.Ticker:  rsiPullbackBindingFor(rsipullbackreni.Ticker, rsipullbackreni.DefaultParams),
	rsipullbackrtkm.Ticker:  rsiPullbackBindingFor(rsipullbackrtkm.Ticker, rsipullbackrtkm.DefaultParams),
	rsipullbackrtkmp.Ticker: rsiPullbackBindingFor(rsipullbackrtkmp.Ticker, rsipullbackrtkmp.DefaultParams),
	rsipullbacksfin.Ticker:  rsiPullbackBindingFor(rsipullbacksfin.Ticker, rsipullbacksfin.DefaultParams),
	rsipullbacksibn.Ticker:  rsiPullbackBindingFor(rsipullbacksibn.Ticker, rsipullbacksibn.DefaultParams),
	rsipullbacksngsp.Ticker: rsiPullbackBindingFor(rsipullbacksngsp.Ticker, rsipullbacksngsp.DefaultParams),
	rsipullbacksofl.Ticker:  rsiPullbackBindingFor(rsipullbacksofl.Ticker, rsipullbacksofl.DefaultParams),
	rsipullbackspbe.Ticker:  rsiPullbackBindingFor(rsipullbackspbe.Ticker, rsipullbackspbe.DefaultParams),
	rsipullbacksvav.Ticker:  rsiPullbackBindingFor(rsipullbacksvav.Ticker, rsipullbacksvav.DefaultParams),
	rsipullbacksvcb.Ticker:  rsiPullbackBindingFor(rsipullbacksvcb.Ticker, rsipullbacksvcb.DefaultParams),
	rsipullbacktbank.Ticker: rsiPullbackBindingFor(rsipullbacktbank.Ticker, rsipullbacktbank.DefaultParams),
	rsipullbacktgka.Ticker:  rsiPullbackBindingFor(rsipullbacktgka.Ticker, rsipullbacktgka.DefaultParams),
	rsipullbacktrnfp.Ticker: rsiPullbackBindingFor(rsipullbacktrnfp.Ticker, rsipullbacktrnfp.DefaultParams),
	rsipullbackugld.Ticker:  rsiPullbackBindingFor(rsipullbackugld.Ticker, rsipullbackugld.DefaultParams),
	rsipullbackuwgn.Ticker:  rsiPullbackBindingFor(rsipullbackuwgn.Ticker, rsipullbackuwgn.DefaultParams),
	rsipullbackvsmo.Ticker:  rsiPullbackBindingFor(rsipullbackvsmo.Ticker, rsipullbackvsmo.DefaultParams),
	rsipullbackwush.Ticker:  rsiPullbackBindingFor(rsipullbackwush.Ticker, rsipullbackwush.DefaultParams),
	rsipullbackx5.Ticker:    rsiPullbackBindingFor(rsipullbackx5.Ticker, rsipullbackx5.DefaultParams),
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
