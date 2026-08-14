package live

import (
	"testing"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
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

// Обратная сторона предыдущего теста: быть в реестре — не то же самое, что быть готовым к
// торговле. NVTK зарегистрирован, но возвращает baseline, то есть параметры, которые никогда не
// проверялись на этом инструменте. Попадание такого тикера в дефолтную вселенную
// означало бы живые сделки по неоткалиброванной конфигурации, и заметить это по коду трудно:
// внешне запись в карте выглядит так же, как у откалиброванного соседа.
func TestBaselineTrackingTickersStayOutOfTheDefaultUniverse(t *testing.T) {
	baseline := core.DefaultParams()
	for _, ticker := range config.NewRSIPullbackConfig().Tickers {
		p, ok := ParamsFor(ticker)
		if !ok {
			continue // отсутствие в реестре ловит тест выше
		}
		if p == baseline {
			t.Errorf("%s стоит в дефолтной вселенной, но возвращает baseline: торговля пошла бы по неоткалиброванным параметрам", ticker)
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
