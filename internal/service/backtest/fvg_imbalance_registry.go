package backtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/afks"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/aflt"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/alrs"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/astr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/banep"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/baza"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/bspb"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/cbom"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/chmf"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/data"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/domrf"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/enpg"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/etln"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/eutr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/fees"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/fesh"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/fixr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/flot"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/gazp"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/gmkn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/head"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/hydr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/irao"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/ivat"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/leas"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/lent"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/lkoh"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/lsrg"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/magn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mdmg"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mgnt"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/moex"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mrkc"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/msng"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mtlr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mtlrp"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mtss"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mvid"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/nlmk"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/ozon"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/ozph"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/phor"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/plzl"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/posi"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/ragr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/reni"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/rnft"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/rosn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/rtkm"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sber"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sberp"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sfin"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/smlt"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sngs"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sngsp"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/spbe"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/svcb"
	tickerT "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/t"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/tatn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/tatnp"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/ugld"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/unac"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/upro"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/uwgn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/vkco"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/vseh"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/vtbr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/x5"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/ydex"
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
	"ASTR":  astr.Params,
	"UNAC":  unac.Params,
	"DATA":  data.Params,
	"CBOM":  cbom.Params,
	"LENT":  lent.Params,
	"MVID":  mvid.Params,
	"LEAS":  leas.Params,
	"OZPH":  ozph.Params,
	"BANEP": banep.Params,
	"MSNG":  msng.Params,
	"HYDR":  hydr.Params,
	"FESH":  fesh.Params,
	"BAZA":  baza.Params,
	"MTLRP": mtlrp.Params,
	"ETLN":  etln.Params,
	"TATNP": tatnp.Params,
	"UWGN":  uwgn.Params,
	"HEAD":  head.Params,
	"VSEH":  vseh.Params,
	"MRKC":  mrkc.Params,
	"EUTR":  eutr.Params,
	"PLZL":  plzl.Params,
	"RTKM":  rtkm.Params,
	"AFLT":  aflt.Params,
	"MDMG":  mdmg.Params,
	"UPRO":  upro.Params,
	"FLOT":  flot.Params,
	"NLMK":  nlmk.Params,
	"IVAT":  ivat.Params,
	"ENPG":  enpg.Params,
	"MGNT":  mgnt.Params,
	"RENI":  reni.Params,
	"RAGR":  ragr.Params,
	"LSRG":  lsrg.Params,
	"SVCB":  svcb.Params,
	"VKCO":  vkco.Params,
	"UGLD":  ugld.Params,
	"SPBE":  spbe.Params,
	"MTLR":  mtlr.Params,
	"TATN":  tatn.Params,
	"RNFT":  rnft.Params,
	"X5":    x5.Params,
	"MOEX":  moex.Params,
	"MTSS":  mtss.Params,
	"SMLT":  smlt.Params,
	"FIXR":  fixr.Params,
	"MAGN":  magn.Params,
	"GMKN":  gmkn.Params,
	"SFIN":  sfin.Params,
	"FEES":  fees.Params,
	"PHOR":  phor.Params,
	"SNGSP": sngsp.Params,
	"POSI":  posi.Params,
	"IRAO":  irao.Params,
	"NVTK":  nvtk.Params,
	"SBERP": sberp.Params,
	"AFKS":  afks.Params,
	"CHMF":  chmf.Params,
	"VTBR":  vtbr.Params,
	"ALRS":  alrs.Params,
	"ROSN":  rosn.Params,
	"YDEX":  ydex.Params,
	"OZON":  ozon.Params,
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
