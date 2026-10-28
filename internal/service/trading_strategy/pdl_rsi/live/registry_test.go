package live

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/aflt"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/blng"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/data"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/flot"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gazp"
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

// Реестр live совпадает с реестром бэктеста (internal/service/backtest/pdl_rsi_registry.go,
// он не экспортирован — сверка по явному списку) и отдаёт литерал пакета тикера как есть.
func TestRegistryMatchesBacktestRegistry(t *testing.T) {
	want := map[string]func() any{
		"AFLT": func() any { return aflt.Params() },
		"BLNG": func() any { return blng.Params() },
		"DATA": func() any { return data.Params() },
		"FLOT": func() any { return flot.Params() },
		"GAZP": func() any { return gazp.Params() },
		"LENT": func() any { return lent.Params() },
		"MDMG": func() any { return mdmg.Params() },
		"MGNT": func() any { return mgnt.Params() },
		"MOEX": func() any { return moex.Params() },
		"MRKC": func() any { return mrkc.Params() },
		"MRKV": func() any { return mrkv.Params() },
		"MSNG": func() any { return msng.Params() },
		"MSRS": func() any { return msrs.Params() },
		"MTSS": func() any { return mtss.Params() },
		"NVTK": func() any { return nvtk.Params() },
		"PHOR": func() any { return phor.Params() },
		"RENI": func() any { return reni.Params() },
		"ROSN": func() any { return rosn.Params() },
		"SPBE": func() any { return spbe.Params() },
		"SVCB": func() any { return svcb.Params() },
		"TGKA": func() any { return tgka.Params() },
		"VTBR": func() any { return vtbr.Params() },
		"X5":   func() any { return x5.Params() },
	}
	var got []string
	for tk := range paramsByTicker {
		got = append(got, tk)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"AFLT", "BLNG", "DATA", "FLOT", "GAZP", "LENT", "MDMG", "MGNT", "MOEX", "MRKC", "MRKV", "MSNG", "MSRS", "MTSS", "NVTK", "PHOR", "RENI", "ROSN", "SPBE", "SVCB", "TGKA", "VTBR", "X5"}) {
		t.Fatalf("тикеры реестра = %v", got)
	}
	for tk, lit := range want {
		p, ok := ParamsFor(tk)
		if !ok || !reflect.DeepEqual(any(p), lit()) {
			t.Errorf("%s: ParamsFor = %+v/%v, want литерал пакета %+v", tk, p, ok, lit())
		}
	}
	if _, ok := ParamsFor("SBER"); ok {
		t.Error("SBER не должен быть в реестре")
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
