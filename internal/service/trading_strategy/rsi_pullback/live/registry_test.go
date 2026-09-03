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
// торговле. Тикер, чей пакет возвращает baseline, торговал бы параметрами, которые никогда не
// проверялись на этом инструменте. Сейчас таких в карте нет — последним был NVTK, откалиброванный
// 2026-08-16, — но состояние воспроизводится каждый раз, когда заводят пакет под будущий литерал
// (так было с reni, fesh, wush, lsngp), и тест обязан пережить следующий такой случай.
// Попадание такого тикера в дефолтную вселенную
// означало бы живые сделки по неоткалиброванной конфигурации, и заметить это по коду трудно:
// внешне запись в карте выглядит так же, как у откалиброванного соседа.
func TestBaselineTrackingTickersStayOutOfTheDefaultUniverse(t *testing.T) {
	baseline := core.DefaultParams()
	for _, ticker := range config.NewRSIPullbackConfig().Tickers {
		p, ok := ParamsFor(ticker)
		if !ok {
			continue // отсутствие в реестре ловит тест выше
		}
		if reason, allowed := baselineByDesignTickers[ticker]; allowed {
			if p != baseline {
				t.Errorf("%s стоит в списке исключений «торгуем дефолты осознанно», но его параметры от baseline ОТЛИЧАЮТСЯ: либо тикер откалиброван и исключение надо снять, либо литерал разошёлся с решением владельца (%s)", ticker, reason)
			}
			continue
		}
		if p == baseline {
			t.Errorf("%s стоит в дефолтной вселенной, но возвращает baseline: торговля пошла бы по неоткалиброванным параметрам", ticker)
		}
	}
}

// baselineByDesignTickers — ИМЕННЫЕ исключения из теста выше: тикеры, которые торгуют дефолты ядра
// НЕ ПО ЗАБЫВЧИВОСТИ, А ПО РЕШЕНИЮ ВЛАДЕЛЬЦА, принятому по числам. Список ведётся руками, каждая
// запись обязана называть причину, и попасть сюда можно только после того, как калибровка тикера
// ПРОВЕДЕНА и её результат оказался ХУЖЕ дефолтов вне выборки. «Калибровка ещё не проводилась» —
// не основание: такой тикер в боевой вселенной делать нечего, и общее правило теста его ловит.
//
// Смысл исключения именно такой: дефолты ядра — единственная конфигурация каталога, которая НЕ
// подбиралась под конкретную бумагу, поэтому подгонки в ней нет по построению. Когда три
// независимых среза данных показывают, что подобранная точка проигрывает дефолтам, торговля
// дефолтами — не «неоткалиброванная конфигурация», а лучшая из проверенных.
var baselineByDesignTickers = map[string]string{
	// TGKA: калибровка 2026-09-03 проведена и провалена ТРИЖДЫ, каждый раз против дефолтов ядра —
	// контроль 24/12/3 дал 0.899 против 2.518 у дефолтов, точка окна B выродилась до одной сделки в
	// пуле OOS, а на слепом holdout 2026-03-03..2026-09-03 точка дала PF 0.815 при net -1 070 против
	// 3.709 и +20 223 у дефолтов на том же куске. Сами дефолты на TGKA дают лучший baseline каталога
	// (PF 1.967 на 167 сделках за 36 месяцев, 2.518 на 131 сделке за 24 месяца), и владелец 2026-09-03
	// решил завести тикер именно на них. Полный разбор — docs/superpowers/plans/task-11-report-tgka.md
	// и док-комментарий пакета strategy/tgka.
	"TGKA": "калибровка провалена трижды, дефолты ядра лучше точки на всех трёх срезах (решение владельца 2026-09-03)",
}

// TestBaselineByDesignExceptionsStayDocumented держит вторую половину смысла списка исключений:
// пустая или безымянная запись превратила бы его в тихую дыру в главном сторожевом тесте боевой
// вселенной. Причина обязана быть непустой, а сам тикер — стоять в дефолтной вселенной: исключение,
// пережившее удаление тикера из прода, — это забытая строка, которая однажды пропустит чужой
// baseline в бой.
func TestBaselineByDesignExceptionsStayDocumented(t *testing.T) {
	universe := make(map[string]bool)
	for _, ticker := range config.NewRSIPullbackConfig().Tickers {
		universe[ticker] = true
	}
	for ticker, reason := range baselineByDesignTickers {
		if reason == "" {
			t.Errorf("%s: исключение без причины — список исключений обязан объяснять каждую запись", ticker)
		}
		if !universe[ticker] {
			t.Errorf("%s: исключение есть, а тикера в боевой вселенной нет — забытая строка, снять её", ticker)
		}
	}
}

// Ловушка нулевого значения: тикерные пакеты задают core.Params литералом, и забытое
// поле молча даёт UseRSIExit=0 — то есть выключенный основной выход (67.6% выходов UGLD после
// перекалибровки 2026-08-17; с UseRSIExit=0 та же точка даёт pooled OOS PF 0.973 вместо 3.627).
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
