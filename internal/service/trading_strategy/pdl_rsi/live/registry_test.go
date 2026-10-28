package live

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/aflt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/belu"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/blng"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/bspb"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/data"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/deli"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/enpg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/fesh"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/flot"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gazp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gemc"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/hnfg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lent"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lsngp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mdmg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mgnt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/moex"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrkc"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrku"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mrkv"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/msng"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/msrs"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mtlr"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mtss"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/phor"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rbcm"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/reni"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rnft"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/rosn"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sber"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sberp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/sgzh"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/smlt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/spbe"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/svcb"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/tgka"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/unac"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vtbr"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/x5"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/zaym"
)

// Реестр live совпадает с реестром бэктеста (internal/service/backtest/pdl_rsi_registry.go,
// он не экспортирован — сверка по явному списку) и отдаёт литерал пакета тикера как есть.
func TestRegistryMatchesBacktestRegistry(t *testing.T) {
	want := map[string]func() any{
		"AFLT":  func() any { return aflt.Params() },
		"BELU":  func() any { return belu.Params() },
		"BLNG":  func() any { return blng.Params() },
		"BSPB":  func() any { return bspb.Params() },
		"DATA":  func() any { return data.Params() },
		"DELI":  func() any { return deli.Params() },
		"ENPG":  func() any { return enpg.Params() },
		"FESH":  func() any { return fesh.Params() },
		"FLOT":  func() any { return flot.Params() },
		"GAZP":  func() any { return gazp.Params() },
		"GEMC":  func() any { return gemc.Params() },
		"HNFG":  func() any { return hnfg.Params() },
		"LENT":  func() any { return lent.Params() },
		"LSNGP": func() any { return lsngp.Params() },
		"MDMG":  func() any { return mdmg.Params() },
		"MGNT":  func() any { return mgnt.Params() },
		"MOEX":  func() any { return moex.Params() },
		"MRKC":  func() any { return mrkc.Params() },
		"MRKU":  func() any { return mrku.Params() },
		"MRKV":  func() any { return mrkv.Params() },
		"MSNG":  func() any { return msng.Params() },
		"MSRS":  func() any { return msrs.Params() },
		"MTLR":  func() any { return mtlr.Params() },
		"MTSS":  func() any { return mtss.Params() },
		"NVTK":  func() any { return nvtk.Params() },
		"PHOR":  func() any { return phor.Params() },
		"RBCM":  func() any { return rbcm.Params() },
		"RENI":  func() any { return reni.Params() },
		"RNFT":  func() any { return rnft.Params() },
		"ROSN":  func() any { return rosn.Params() },
		"SBER":  func() any { return sber.Params() },
		"SBERP": func() any { return sberp.Params() },
		"SGZH":  func() any { return sgzh.Params() },
		"SMLT":  func() any { return smlt.Params() },
		"SPBE":  func() any { return spbe.Params() },
		"SVCB":  func() any { return svcb.Params() },
		"TGKA":  func() any { return tgka.Params() },
		"UNAC":  func() any { return unac.Params() },
		"VTBR":  func() any { return vtbr.Params() },
		"X5":    func() any { return x5.Params() },
		"ZAYM":  func() any { return zaym.Params() },
	}
	var got []string
	for tk := range paramsByTicker {
		got = append(got, tk)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"AFLT", "BELU", "BLNG", "BSPB", "DATA", "DELI", "ENPG", "FESH", "FLOT", "GAZP", "GEMC", "HNFG", "LENT", "LSNGP", "MDMG", "MGNT", "MOEX", "MRKC", "MRKU", "MRKV", "MSNG", "MSRS", "MTLR", "MTSS", "NVTK", "PHOR", "RBCM", "RENI", "RNFT", "ROSN", "SBER", "SBERP", "SGZH", "SMLT", "SPBE", "SVCB", "TGKA", "UNAC", "VTBR", "X5", "ZAYM"}) {
		t.Fatalf("тикеры реестра = %v", got)
	}
	for tk, lit := range want {
		p, ok := ParamsFor(tk)
		if !ok || !reflect.DeepEqual(any(p), lit()) {
			t.Errorf("%s: ParamsFor = %+v/%v, want литерал пакета %+v", tk, p, ok, lit())
		}
	}
	if _, ok := ParamsFor("NOPE"); ok {
		t.Error("неизвестный тикер NOPE не должен быть в реестре")
	}
}

// Без биржевого стопа гость не защищён между проходами раннера: StopATR > 0 обязателен.
func TestRegisteredLiteralsHaveExchangeStop(t *testing.T) {
	for tk := range paramsByTicker {
		if p, _ := ParamsFor(tk); p.StopATR <= 0 {
			t.Errorf("%s: StopATR = %v, want > 0", tk, p.StopATR)
		}
	}
}

// Ядро не раскрывает шаг баров, поэтому проверяется только реестровость.
func TestStrategyForOnlyRegistered(t *testing.T) {
	st, ok := StrategyFor("SVCB")
	if !ok || st == nil {
		t.Fatalf("StrategyFor(SVCB) = %v/%v", st, ok)
	}
	if _, ok := StrategyFor("НЕТ"); ok {
		t.Fatal("StrategyFor(НЕТ) ok")
	}
}

// Каждый тикер из PDL_RSI_TICKERS в примерах env (local и prod) зарегистрирован: незарегистрированный
// тикер гость молча не торговал бы. Пустое значение допустимо.
func TestPDLRSILiveTickersAreRegistered(t *testing.T) {
	for _, name := range []string{"local.env.example", "prod.env.example"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "env", name))
		if err != nil {
			t.Fatalf("read env/%s: %v", name, err)
		}
		found := false
		for _, line := range strings.Split(string(raw), "\n") {
			key, v, ok := strings.Cut(strings.TrimSpace(line), "=")
			if !ok || key != "PDL_RSI_TICKERS" {
				continue
			}
			found = true
			for _, tk := range strings.Split(v, ",") {
				if tk = strings.TrimSpace(tk); tk == "" {
					continue
				}
				if _, ok := ParamsFor(tk); !ok {
					t.Errorf("%s есть в PDL_RSI_TICKERS env/%s, но не зарегистрирован в реестре pdl_rsi/live", tk, name)
				}
			}
		}
		if !found {
			t.Errorf("PDL_RSI_TICKERS не найден в env/%s", name)
		}
	}
}
