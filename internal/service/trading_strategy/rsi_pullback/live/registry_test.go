package live

import (
	"testing"

	"tinvest/internal/config"
)

// Каждый тикер дефолтной вселенной обязан находиться в реестре, иначе раннер молча
// не будет торговать его вообще.
func TestEveryDefaultTickerIsRegistered(t *testing.T) {
	for _, ticker := range config.NewRSIPullbackConfig().Tickers {
		if _, ok := ParamsFor(ticker); !ok {
			t.Fatalf("ticker %s from the default universe is missing from the registry", ticker)
		}
	}
}

// Ловушка нулевого значения: тикерные пакеты задают core.Params литералом, и забытое
// поле молча даёт UseRSIExit=0 — то есть выключенный основной выход (61% выходов UGLD).
func TestRegisteredTickersKeepTheRSIExitArmed(t *testing.T) {
	for ticker := range paramsByTicker {
		p, _ := ParamsFor(ticker)
		if p.UseRSIExit != 1 {
			t.Fatalf("%s: UseRSIExit = %d, want 1", ticker, p.UseRSIExit)
		}
	}
}

// Стратегия должна строиться на параметрах тикера, а не на дефолтах ядра.
func TestStrategyForUsesTickerParams(t *testing.T) {
	st, ok := StrategyFor("UGLD")
	if !ok {
		t.Fatal("StrategyFor(UGLD) not ok")
	}
	if st.Ticker() != "UGLD" {
		t.Fatalf("Ticker() = %q, want UGLD", st.Ticker())
	}
	if st.Lookback() < 400 {
		t.Fatalf("Lookback() = %d, want >= 400 (UGLD's volume gate dominates)", st.Lookback())
	}
	if _, ok := StrategyFor("НЕТ-ТАКОГО"); ok {
		t.Fatal("StrategyFor returned ok for an unknown ticker")
	}
}
