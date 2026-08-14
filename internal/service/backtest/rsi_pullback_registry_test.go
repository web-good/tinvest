package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
	rsipullbacklent "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lent"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/reni"
	rsipullbackwush "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"
)

// TestRSIPullbackBindingBuildsForTicker checks the wiring on a ticker whose package still
// tracks the baseline. SBER is deliberate: pinning this to a CALIBRATED ticker would turn
// every future calibration into a red test, which is exactly how this test broke before.
func TestRSIPullbackBindingBuildsForTicker(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("SBER")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("DefaultParams() = %+v, want the baseline %+v", p, core.DefaultParams())
	}
	s := b.Build(p)
	if s.Ticker() != "SBER" {
		t.Fatalf("Ticker() = %q, want SBER", s.Ticker())
	}
	if s.Lookback() < 220 {
		t.Fatalf("Lookback() = %d, want >= 220", s.Lookback())
	}
}

// TestRSIPullbackCalibratedBindingKeepsItsOwnLiteral pins the opposite direction for a ticker
// that HAS been calibrated: GAZP must not drift back to the baseline. A package whose literal
// silently collapses into core.DefaultParams() would look calibrated while trading generic
// values, and the report would carry the ticker's name either way.
func TestRSIPullbackCalibratedBindingKeepsItsOwnLiteral(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("GAZP returns the baseline: its calibrated literal was lost")
	}
	if got := b.Build(p).Ticker(); got != "GAZP" {
		t.Fatalf("Ticker() = %q, want GAZP", got)
	}
}

// TestRSIPullbackParseParamsLayersOverDefaults pins that partial calibration JSON overrides
// only the fields it names. It compares against the BINDING's own defaults, not the package
// baseline: for a calibrated ticker those differ, and the test is about layering, not baseline.
func TestRSIPullbackParseParamsLayersOverDefaults(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	base, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
	}
	got, err := b.ParseParams([]byte(`{"RSILower": 10}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.RSILower != 10 {
		t.Fatalf("RSILower = %v, want the JSON value 10", p.RSILower)
	}
	if p.RSIUpper != base.RSIUpper {
		t.Fatalf("RSIUpper = %v, want the binding default %v (partial JSON must not zero other fields)",
			p.RSIUpper, base.RSIUpper)
	}
	if p.EMASlow != base.EMASlow {
		t.Fatalf("EMASlow = %v, want the binding default %v", p.EMASlow, base.EMASlow)
	}
}

// TestRSIPullbackRegistryEntriesMatchTheirTicker guards the copy-paste hazard of per-ticker
// packages: a map key must build a strategy labelled with that same ticker. A package cloned
// from a sibling with its Ticker constant left unchanged registers itself under the wrong key,
// and every report for that instrument would then carry the wrong label.
func TestRSIPullbackRegistryEntriesMatchTheirTicker(t *testing.T) {
	if len(rsiPullbackRegistry) == 0 {
		t.Fatal("rsi_pullback registry is empty")
	}
	for ticker, b := range rsiPullbackRegistry {
		t.Run(ticker, func(t *testing.T) {
			p, ok := b.DefaultParams().(core.Params)
			if !ok {
				t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
			}
			if got := b.Build(p).Ticker(); got != ticker {
				t.Fatalf("registered under %q but builds Ticker() = %q", ticker, got)
			}
		})
	}
}

// TestRSIPullbackRegistryKeepsTheStopArmed pins the one parameter no per-ticker package may
// relax: StopDailyATR = 0 disables the stop entirely, and this strategy holds positions across
// nights and weekends. A calibrated package that lands on zero would ship an unprotected long.
func TestRSIPullbackRegistryKeepsTheStopArmed(t *testing.T) {
	for ticker, b := range rsiPullbackRegistry {
		p, ok := b.DefaultParams().(core.Params)
		if !ok {
			t.Fatalf("%s: DefaultParams() returned %T, want core.Params", ticker, b.DefaultParams())
		}
		if p.StopDailyATR <= 0 {
			t.Fatalf("%s: StopDailyATR = %v, want > 0 — a multi-day hold must never run without a stop",
				ticker, p.StopDailyATR)
		}
	}
}

// TestRSIPullbackUnknownTickerFallsBackToGeneric pins that an unregistered ticker still runs,
// on the baseline params, rather than failing or silently borrowing another ticker's config.
func TestRSIPullbackUnknownTickerFallsBackToGeneric(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("NOSUCH")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("DefaultParams() = %+v, want the baseline %+v", p, core.DefaultParams())
	}
	if got := b.Build(p).Ticker(); got != "NOSUCH" {
		t.Fatalf("Ticker() = %q, want NOSUCH", got)
	}
}

// TestRSIPullbackTickersKeepTheRSIExitArmed сторожит ловушку нулевого значения: тикерные
// пакеты, задающие core.Params ЛИТЕРАЛОМ, получают 0 в каждом поле, которое забыли
// перечислить, а UseRSIExit=0 означает выключенный выход. ParseParams стартует именно с этих
// дефолтов, поэтому пропуск молча меняет поведение откалиброванного тикера, не роняя ничего.
func TestRSIPullbackTickersKeepTheRSIExitArmed(t *testing.T) {
	for ticker, b := range rsiPullbackRegistry {
		p, ok := b.DefaultParams().(core.Params)
		if !ok {
			t.Fatalf("%s: DefaultParams вернул %T, want core.Params", ticker, b.DefaultParams())
		}
		if p.UseRSIExit != 1 {
			t.Errorf("%s: UseRSIExit = %d, want 1 — поле забыто в литерале core.Params",
				ticker, p.UseRSIExit)
		}
	}
}

func TestRSIPullbackParseParamsRejectsGarbage(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	if _, err := b.ParseParams([]byte(`{"RSILower":`)); err == nil {
		t.Fatal("ParseParams accepted malformed JSON, want an error")
	}
}

// TestRSIPullbackTBankKeepsItsOwnCalibratedLiteral used to pin a DELIBERATE hypothesis — T
// starting from GAZP's post-grid literal, to test whether parameters transfer between liquid
// names. That hypothesis is now retired: T has been calibrated on its own data (report
// reports/T/T_rsi_pullback_Minutes30_20260731_134407.md, 67 trades, in-sample PF 1.312), and its
// literal no longer equals GAZP's. This test pins T's own values instead, the same shape as
// TestRSIPullbackCalibratedBindingKeepsItsOwnLiteral: catch a silent collapse back to
// core.DefaultParams(), and catch T's literal drifting away from what its report was run on.
func TestRSIPullbackTBankKeepsItsOwnCalibratedLiteral(t *testing.T) {
	want := core.Params{
		RSIPeriod:       5,
		RSILower:        20,
		RSIUpper:        65,
		EMAFast:         20,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    0.5,
		TPDailyATR:      1.5,
		UseVolume:       0,
		VolBaseDays:     5,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	tbRaw := RSIPullbackLookupOrGeneric("T").DefaultParams()
	tb, ok := tbRaw.(core.Params)
	if !ok {
		t.Fatalf("T: DefaultParams() returned %T, want core.Params", tbRaw)
	}
	if tb == core.DefaultParams() {
		t.Fatal("T returns the baseline: its calibrated literal was lost")
	}
	if tb != want {
		t.Fatalf("T params = %+v, want the calibrated literal %+v — it must match the report T was run on", tb, want)
	}
	if got := RSIPullbackLookupOrGeneric("T").Build(tb).Ticker(); got != "T" {
		t.Fatalf("Ticker() = %q, want T", got)
	}
}

// TestRSIPullbackRENIIsRegisteredAndCalibrated пинует два неотличимых снаружи факта: RENI есть
// в карте (а не проваливается в generic-ветку) И возвращает собственный литерал, а не baseline.
// Пакет strategy/reni заведён 2026-08-10 ДО калибровки и до 2026-08-12 обязан был возвращать
// core.DefaultParams(); калибровка прогналась, литерал появился — и теперь бэктест обязан гонять
// ровно его, иначе перепроверка отчёта RENI мерила бы не то, что стоит в коде. Снимок самого
// литерала живёт рядом с ним, в strategy/reni/reni_test.go.
func TestRSIPullbackRENIIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["RENI"]
	if !ok {
		t.Fatal("RENI отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("RENI вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := reni.DefaultParams(); p != want {
		t.Fatalf("RENI params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "RENI" {
		t.Fatalf("Ticker() = %q, want RENI", got)
	}
}

// TestRSIPullbackDOMRFIsRegisteredAndCalibrated пинует ДВА разных факта, которые
// RSIPullbackLookupOrGeneric снаружи неотличимы: DOMRF присутствует в карте (а не проваливается
// в generic-ветку) И возвращает свой литерал, а не baseline. С 2026-08-10 тикер откалиброван и
// принят владельцем в прод, поэтому бэктест обязан гонять ровно те параметры, что стоят в живом
// раннере: иначе любая перепроверка отчёта DOMRF мерила бы не то, чем торгуют. Снимок самого
// литерала живёт рядом с ним, в strategy/domrf/domrf_test.go.
func TestRSIPullbackDOMRFIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["DOMRF"]
	if !ok {
		t.Fatal("DOMRF отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("DOMRF вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := domrf.DefaultParams(); p != want {
		t.Fatalf("DOMRF params = %+v, want литерал живого раннера %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "DOMRF" {
		t.Fatalf("Ticker() = %q, want DOMRF", got)
	}
}

// TestRSIPullbackFESHIsRegisteredAndCalibrated пинует два неотличимых снаружи факта: FESH есть
// в карте (а не проваливается в generic-ветку) И возвращает собственный литерал, а не baseline.
// Пакет strategy/fesh заведён 2026-08-12 ДО калибровки и до 2026-08-13 обязан был возвращать
// core.DefaultParams(); калибровка прогналась, литерал появился — и теперь бэктест обязан гонять
// ровно его, иначе перепроверка замеров FESH мерила бы не то, что стоит в коде. Снимок самого
// литерала живёт рядом с ним, в strategy/fesh/fesh_test.go, вместе с обоснованием двух полей,
// которые чаще всего «улучшают» на глаз (SpentDayATR и FreshDayATR).
//
// Отдельно от снимка: штатный протокол §8 этот тикер НЕ подтверждает (тематические сетки на
// 4 фолдах дают pooled OOS PF 1.029 по входу и 0.953 по тренду), поэтому наличие литерала здесь
// означает лишь снятую техническую блокировку, а не право ставить FESH в боевую вселенную —
// это решение владельца, как было с RENI и DOMRF.
func TestRSIPullbackFESHIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["FESH"]
	if !ok {
		t.Fatal("FESH отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("FESH: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("FESH вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := fesh.DefaultParams(); p != want {
		t.Fatalf("FESH params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "FESH" {
		t.Fatalf("Ticker() = %q, want FESH", got)
	}
}

// TestRSIPullbackLENTTracksBaseline держит состояние «калибровка не проводилась» ВИДИМЫМ. Пакет
// strategy/lent заведён 2026-08-14 до прогонов — как место, куда ляжет литерал, и как носитель
// замеров, — и до тех пор обязан возвращать ровно core.DefaultParams(). Без этого теста разница
// между «тикер откалиброван» и «тикер просто зарегистрирован» снаружи неразличима: и то и другое
// выглядит как строка в реестре, а отчёт несёт имя инструмента в обоих случаях.
//
// Тест умрёт в тот день, когда планка из доки пакета будет взята и появится литерал — тогда его
// место занимает снимок вида TestRSIPullback<Ticker>IsRegisteredAndCalibrated, как это уже
// произошло с RENI, FESH и WUSH. Красный тест здесь означает ровно одно: кто-то поставил LENT
// параметры, не пройдя этот шаг.
func TestRSIPullbackLENTTracksBaseline(t *testing.T) {
	b, ok := rsiPullbackRegistry["LENT"]
	if !ok {
		t.Fatal("LENT отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("LENT: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("LENT params = %+v, want baseline %+v — калибровка не проводилась, литерала быть не должно", p, core.DefaultParams())
	}
	if want := rsipullbacklent.DefaultParams(); p != want {
		t.Fatalf("LENT params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "LENT" {
		t.Fatalf("Ticker() = %q, want LENT", got)
	}
}

// TestRSIPullbackWUSHIsRegisteredAndCalibrated пинует два неотличимых снаружи факта: WUSH есть
// в карте (а не проваливается в generic-ветку) И возвращает собственный литерал, а не baseline.
// Пакет strategy/wush заведён 2026-08-13 ДО калибровки и до 2026-08-14 обязан был возвращать
// core.DefaultParams(); сторожевой тест TestRSIPullbackWUSHTracksBaseline держал это состояние и
// заменён этим снимком, когда владелец распорядился поставить литерал.
//
// Отдельно от снимка: тикер НЕ подтверждён штатным протоколом §8 — тематические прогоны дали
// pooled OOS PF 2.000 по входу и 1.317 по тренду при объявленной заранее планке 1.5 на обеих
// темах. Литерал отобран отдельным сканом с порогом частоты и перепроверен точечным
// walk-forward (pooled PF 2.999 на 81 сделке), но для фиксированной точки это не out-of-sample:
// конфигурацию выбрал скан, видевший всю историю. Наличие литерала здесь означает снятую
// техническую блокировку, а не право ставить WUSH в боевую вселенную — это решение владельца,
// как было с RENI, DOMRF и FESH. Снимок самого литерала и обоснование трёх уязвимых полей живут
// рядом с ним, в strategy/wush/wush_test.go.
func TestRSIPullbackWUSHIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["WUSH"]
	if !ok {
		t.Fatal("WUSH отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("WUSH: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("WUSH вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbackwush.DefaultParams(); p != want {
		t.Fatalf("WUSH params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "WUSH" {
		t.Fatalf("Ticker() = %q, want WUSH", got)
	}
}
