package backtest

import (
	"testing"

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
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/reni"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rosn"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/spbe"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/svcb"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/tgka"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vtbr"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/x5"
)

func TestPDLRSITickersUseTheirPackages(t *testing.T) {
	for ticker, want := range map[string]func() core.Params{"AFLT": aflt.Params, "DATA": data.Params, "FLOT": flot.Params, "GAZP": gazp.Params, "HNFG": hnfg.Params, "LENT": lent.Params, "MDMG": mdmg.Params, "MGNT": mgnt.Params, "MOEX": moex.Params, "MRKC": mrkc.Params, "MRKV": mrkv.Params, "MSNG": msng.Params, "MSRS": msrs.Params, "MTSS": mtss.Params, "NVTK": nvtk.Params, "PHOR": phor.Params, "RENI": reni.Params, "ROSN": rosn.Params, "SPBE": spbe.Params, "SVCB": svcb.Params, "TGKA": tgka.Params, "VTBR": vtbr.Params, "X5": x5.Params} {
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
