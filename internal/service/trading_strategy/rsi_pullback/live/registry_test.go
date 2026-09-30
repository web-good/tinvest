package live

import (
	"testing"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/aqua"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/irkt"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/magn"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/mdmg"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ragr"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sfin"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svcb"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/x5"
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
// ПРОВЕДЕНА и её результат оказался ХУЖЕ дефолтов вне выборки — либо НЕ ОТЛИЧИМ от них, см. второе
// основание ниже. «Калибровка ещё не проводилась» — не основание: такой тикер в боевой вселенной
// делать нечего, и общее правило теста его ловит.
//
// Смысл исключения именно такой: дефолты ядра — единственная конфигурация каталога, которая НЕ
// подбиралась под конкретную бумагу, поэтому подгонки в ней нет по построению. Когда три
// независимых среза данных показывают, что подобранная точка проигрывает дефолтам, торговля
// дефолтами — не «неоткалиброванная конфигурация», а лучшая из проверенных.
//
// ВТОРОЕ ОСНОВАНИЕ, добавленное 2026-09-10 по CNRU: калибровка проведена целиком и НЕ НАШЛА поля,
// которое ушло бы от дефолта. Это отличается от случая TGKA, где точки находились и проигрывали:
// здесь точка, собранная правилом большинства, поведенчески СОВПАДАЕТ с дефолтами, и литерала,
// который можно было бы записать, просто не существует. Основание такое же твёрдое — оно опирается
// на прогнанные темы, а не на пропущенную работу, — но требует, чтобы причина в записи прямо
// говорила о тождестве, а не о проигрыше: иначе строка читалась бы как «точка была хуже», чего
// замеры не показывали.
var baselineByDesignTickers = map[string]string{
	// CNRU: калибровка 2026-09-10 проведена целиком — одиннадцать тем, схема 36/12/6 на окне
	// 2023-09-10..2026-09-10. Из восемнадцати полей фолдовое большинство >=3/4 набрали пять:
	// UseDayATRGate (4/4), UseVolume (3/4) и UseRSIExit (3/4) подтвердили дефолт ядра; StopDailyATR
	// (3/4 за 2.0) отвергнут риск-гейтом A (выживаемость 2.8% при пороге 30%), а потолок гейта 1.0 —
	// анатомией капкана широкого стопа (доля SL-выходов падает 21.3% -> 7.4% при росте удержания и
	// ночёвок); VolMult (3/4 за 2.5) взят, но ИНЕРТЕН при UseVolume=0 — зонд-сосед с VolMult 1.2 дал
	// побайтово тот же отчёт, кроме строки самого параметра. Дефолты ядра на CNRU дают pooled OOS
	// 2.001 на 123 сделках при четырёх прибыльных фолдах из четырёх, 1.739 на контроле 24/12/3,
	// 1.715 при удвоенном круге издержек и прибыльность всех четырёх календарных лет окна; точка
	// повторяет эти числа ровно, потому что ей нечем от них отличаться. Планку не взяли обе ключевые
	// темы по критерию устойчивости ведущей оси. Полный разбор —
	// docs/superpowers/plans/task-11-report-cnru.md и док-комментарий пакета strategy/cnru.
	"CNRU": "калибровка проведена целиком и не нашла ни одного поля, уходящего от дефолтов: точка поведенчески тождественна baseline ядра (2026-09-10)",

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

// Ловушка нулевого значения во втором входе: zone-поля в ядре по умолчанию нулевые, и тикер с
// UseZoneEntry=1, забывший поле в литерале, молча не торговал бы zone вовсе. Пока zone не включён
// ни у одного тикера, тест проходит пусто — он для будущих калибровок.
func TestRegisteredTickersArmZoneFieldsWhenEnabled(t *testing.T) {
	for ticker, p := range paramsByTicker {
		if !p.ZoneArmed() {
			continue
		}
		if p.ZoneRSIPeriod <= 0 || p.ZoneRSILower <= 0 || p.ZoneEMAPeriod <= 0 {
			t.Errorf("%s: UseZoneEntry=%d, но ZoneRSIPeriod=%d ZoneRSILower=%v ZoneEMAPeriod=%d — поле забыто в литерале",
				ticker, p.UseZoneEntry, p.ZoneRSIPeriod, p.ZoneRSILower, p.ZoneEMAPeriod)
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

// TestRegistryHasAQUA держит связку «пакет — реестр живого раннера» для AQUA: раннер обязан отдавать
// ровно тот литерал, который пинит снимок в пакете. Отдельный тест нужен потому, что точка AQUA
// отличается от дефолтов ядра ЕДИНСТВЕННЫМ полем — ранним RSI-выходом 45 вместо 70, — и потеря
// этого поля не сделала бы карту заметно другой на глаз, зато отдала бы бумагу дефолтам, которые на
// ней убыточны (PF 0.931 на расчётном окне).
func TestRegistryHasAQUA(t *testing.T) {
	p, ok := ParamsFor(aqua.Ticker)
	if !ok {
		t.Fatal("AQUA нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := aqua.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
	if p.RSIUpper != 45 {
		t.Fatalf("AQUA: RSIUpper = %v, want 45 — единственное поле, которым точка уходит от дефолтов", p.RSIUpper)
	}
}

// TestRegistryHasMAGN держит связку «пакет — реестр живого раннера» для MAGN: раннер обязан отдавать
// ровно тот литерал, который пинит снимок в пакете. Отдельный тест нужен потому, что MAGN — первый
// тикер этой карты с ВКЛЮЧЁННЫМ объёмным гейтом и окном гейта на физическом минимуме оси
// (VolLookbackBars=1), а гейт считает базу по дневным оборотам: потеря UseVolume или окна не
// ослабила бы бумагу постепенно, она сменила бы набор сделок целиком (145 сделок вместо 89 и PF
// 1.659 вместо 2.418 на расчётном окне).
func TestRegistryHasMAGN(t *testing.T) {
	p, ok := ParamsFor(magn.Ticker)
	if !ok {
		t.Fatal("MAGN нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := magn.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
	if p.UseVolume != 1 || p.VolLookbackBars != 1 {
		t.Fatalf("MAGN: UseVolume = %v, VolLookbackBars = %v, want 1 и 1 — объёмный гейт с окном в один бар", p.UseVolume, p.VolLookbackBars)
	}
	if p.EMASlow != 50 || p.StopDailyATR != 0.7 {
		t.Fatalf("MAGN: EMASlow = %v, StopDailyATR = %v, want 50 и 0.7", p.EMASlow, p.StopDailyATR)
	}
}

// TestRegistryHasIRKT держит связку «пакет — реестр живого раннера» для IRKT: раннер обязан
// отдавать ровно тот литерал, который пинит снимок в пакете strategy/irkt.
func TestRegistryHasIRKT(t *testing.T) {
	p, ok := ParamsFor(irkt.Ticker)
	if !ok {
		t.Fatal("IRKT нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := irkt.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
}

// TestRegistryHasX5 держит связку «пакет — реестр живого раннера» для X5: раннер обязан отдавать
// ровно тот литерал, который пинит снимок в пакете strategy/x5. У X5 цена ошибки выше обычного:
// UseVolume=1 — единственное поле точки, отсекающее печать 2026-01-02 02:00 (-10 453.45 ₽ на
// baseline), а FreshDayATR=0 — решение владельца по риск-гейту C, которое держит раннер вне бара
// 06:30 с медианой объёма 148 лотов.
func TestRegistryHasX5(t *testing.T) {
	p, ok := ParamsFor(x5.Ticker)
	if !ok {
		t.Fatal("X5 нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := x5.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
}

// TestRegistryHasSFIN держит связку «пакет — реестр живого раннера» для SFIN: раннер обязан
// отдавать ровно тот литерал, который пинит снимок в пакете strategy/sfin.
func TestRegistryHasSFIN(t *testing.T) {
	p, ok := ParamsFor(sfin.Ticker)
	if !ok {
		t.Fatal("SFIN нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := sfin.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
}

// TestRegistryHasSVCB держит связку «пакет — реестр живого раннера» для SVCB: раннер обязан
// торговать ровно литерал пакета.
func TestRegistryHasSVCB(t *testing.T) {
	p, ok := ParamsFor(svcb.Ticker)
	if !ok {
		t.Fatal("SVCB нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := svcb.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
}

// TestRegistryHasRAGR держит связку «пакет — реестр живого раннера» для RAGR: раннер обязан
// торговать ровно литерал пакета.
func TestRegistryHasRAGR(t *testing.T) {
	p, ok := ParamsFor(ragr.Ticker)
	if !ok {
		t.Fatal("RAGR нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := ragr.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не литерал пакета:\n got: %+v\nwant: %+v", p, want)
	}
}

// TestRegistryHasMDMG держит связку «пакет — реестр живого раннера» для MDMG: раннер обязан
// торговать ровно параметры пакета.
func TestRegistryHasMDMG(t *testing.T) {
	p, ok := ParamsFor(mdmg.Ticker)
	if !ok {
		t.Fatal("MDMG нет в реестре живого раннера: тикер не будет торговать вовсе")
	}
	if want := mdmg.DefaultParams(); p != want {
		t.Fatalf("реестр отдаёт не параметры пакета:\n got: %+v\nwant: %+v", p, want)
	}
}
