package live

import (
	"slices"
	"testing"

	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/gazp"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/lent"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/mdmg"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/svcb"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/vtbr"
)

// Реестр live совпадает с реестром бэктеста (internal/service/backtest/pdl_rsi_registry.go,
// он не экспортирован — сверка по явному списку) и отдаёт литерал пакета тикера как есть.
func TestRegistryMatchesBacktestRegistry(t *testing.T) {
	want := map[string]func() any{
		"GAZP": func() any { return gazp.Params() },
		"LENT": func() any { return lent.Params() },
		"MDMG": func() any { return mdmg.Params() },
		"NVTK": func() any { return nvtk.Params() },
		"SVCB": func() any { return svcb.Params() },
		"VTBR": func() any { return vtbr.Params() },
	}
	var got []string
	for tk := range paramsByTicker {
		got = append(got, tk)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"GAZP", "LENT", "MDMG", "NVTK", "SVCB", "VTBR"}) {
		t.Fatalf("тикеры реестра = %v", got)
	}
	for tk, lit := range want {
		p, ok := ParamsFor(tk)
		if !ok || any(p) != lit() {
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

func TestStrategyForIsHourly(t *testing.T) {
	st, ok := StrategyFor("SVCB")
	if !ok || st == nil {
		t.Fatalf("StrategyFor(SVCB) = %v/%v", st, ok)
	}
	if _, ok := StrategyFor("НЕТ"); ok {
		t.Fatal("StrategyFor(НЕТ) ok")
	}
}
