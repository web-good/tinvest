package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/afks"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/aflt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/alrs"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/aptk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/aqua"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/banep"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/belu"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/blng"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/bspb"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/cbom"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/data"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/deli"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/dias"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/domrf"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/elfv"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/enpg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/etln"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/fees"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/fesh"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/flot"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gazp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gche"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gemc"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gmkn"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/head"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/hnfg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/hydr"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/irao"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/irkt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/ivat"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lent"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lkoh"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lsngp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lsrg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/magn"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mbnk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mdmg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mgnt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/moex"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrkc"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrkp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrku"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrkv"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrkz"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/msng"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/msrs"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mtlr"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mtlrp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mtss"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mvid"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nkhp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nlmk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nmtp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/ogkb"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/ozon"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/ozph"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/phor"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/plzl"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/posi"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/prmd"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/ragr"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rasp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rbcm"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/reni"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rnft"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rolo"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rosn"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rtkm"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rtkmp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rual"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sber"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sberp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/selg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sfin"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sgzh"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/smlt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sngs"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sngsp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sofl"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/spbe"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/svcb"
	tbank "tinvest/internal/service/trading_strategy/pdl_rsi/strategy/t"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/tatn"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/tatnp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/tgka"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/trmk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/trnfp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/ugld"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/unac"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/upro"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/uwgn"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vkco"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vseh"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vsmo"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vtbr"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/x5"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/ydex"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/zaym"
)

func TestPDLRSITickersUseTheirPackages(t *testing.T) {
	for ticker, want := range map[string]func() core.Params{"AFKS": afks.Params, "AFLT": aflt.Params, "ALRS": alrs.Params, "APTK": aptk.Params, "AQUA": aqua.Params, "BANEP": banep.Params, "BELU": belu.Params, "BLNG": blng.Params, "BSPB": bspb.Params, "CBOM": cbom.Params, "DATA": data.Params, "DELI": deli.Params, "DIAS": dias.Params, "DOMRF": domrf.Params, "ELFV": elfv.Params, "ENPG": enpg.Params, "ETLN": etln.Params, "FEES": fees.Params, "FESH": fesh.Params, "FLOT": flot.Params, "GAZP": gazp.Params, "GCHE": gche.Params, "GEMC": gemc.Params, "GMKN": gmkn.Params, "HEAD": head.Params, "HNFG": hnfg.Params, "HYDR": hydr.Params, "IRAO": irao.Params, "IRKT": irkt.Params, "IVAT": ivat.Params, "LENT": lent.Params, "LKOH": lkoh.Params, "LSNGP": lsngp.Params, "LSRG": lsrg.Params, "MAGN": magn.Params, "MBNK": mbnk.Params, "MDMG": mdmg.Params, "MGNT": mgnt.Params, "MOEX": moex.Params, "MRKC": mrkc.Params, "MRKP": mrkp.Params, "MRKU": mrku.Params, "MRKV": mrkv.Params, "MRKZ": mrkz.Params, "MSNG": msng.Params, "MSRS": msrs.Params, "MTLR": mtlr.Params, "MTLRP": mtlrp.Params, "MTSS": mtss.Params, "MVID": mvid.Params, "NKHP": nkhp.Params, "NLMK": nlmk.Params, "NMTP": nmtp.Params, "NVTK": nvtk.Params, "OGKB": ogkb.Params, "OZON": ozon.Params, "OZPH": ozph.Params, "PHOR": phor.Params, "PLZL": plzl.Params, "POSI": posi.Params, "PRMD": prmd.Params, "RAGR": ragr.Params, "RASP": rasp.Params, "RBCM": rbcm.Params, "RENI": reni.Params, "RNFT": rnft.Params, "ROLO": rolo.Params, "ROSN": rosn.Params, "RTKM": rtkm.Params, "RTKMP": rtkmp.Params, "RUAL": rual.Params, "SBER": sber.Params, "SBERP": sberp.Params, "SELG": selg.Params, "SFIN": sfin.Params, "SGZH": sgzh.Params, "SMLT": smlt.Params, "SNGS": sngs.Params, "SNGSP": sngsp.Params, "SOFL": sofl.Params, "SPBE": spbe.Params, "SVCB": svcb.Params, "T": tbank.Params, "TATN": tatn.Params, "TATNP": tatnp.Params, "TGKA": tgka.Params, "TRMK": trmk.Params, "TRNFP": trnfp.Params, "UGLD": ugld.Params, "UNAC": unac.Params, "UPRO": upro.Params, "UWGN": uwgn.Params, "VKCO": vkco.Params, "VSEH": vseh.Params, "VSMO": vsmo.Params, "VTBR": vtbr.Params, "X5": x5.Params, "YDEX": ydex.Params, "ZAYM": zaym.Params} {
		b, err := PDLRSILookupOrGeneric(ticker, enum.Hour1)
		if err != nil {
			t.Fatalf("%s lookup: %v", ticker, err)
		}
		if got := b.DefaultParams().(core.Params); got != want() {
			t.Fatalf("%s defaults = %+v, want the package literal", ticker, got)
		}
	}
}

func TestPDLRSIGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := PDLRSILookupOrGeneric("ZZZZ", enum.Hour1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	got, ok := b.DefaultParams().(core.Params)
	if !ok || got != core.DefaultParams() {
		t.Fatalf("defaults = %#v, want core.DefaultParams()", b.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "ZZZZ" {
		t.Fatalf("ticker = %q, want ZZZZ", s.Ticker())
	}
}

func TestPDLRSIRunsOnlyOnHour1(t *testing.T) {
	if _, err := PDLRSILookupOrGeneric("NVTK", enum.Hour1); err != nil {
		t.Fatalf("Hour1: %v", err)
	}
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Minutes30, enum.Day1} {
		if _, err := PDLRSILookupOrGeneric("NVTK", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestPDLRSIParseParamsLayersOverDefaults(t *testing.T) {
	b, err := PDLRSILookupOrGeneric("ZZZZ", enum.Hour1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"StochK": 10, "MinRoomATR": 0.5}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	want := core.DefaultParams()
	want.StochK, want.MinRoomATR = 10, 0.5
	if got := parsed.(core.Params); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
