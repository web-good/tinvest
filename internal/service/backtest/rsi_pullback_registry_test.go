package backtest

import (
	"testing"

	rsipullbackbanep "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/banep"
	rsipullbackbspb "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/bspb"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	rsipullbackdias "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/dias"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	rsipullbackelfv "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/elfv"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/fesh"
	rsipullbackivat "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ivat"
	rsipullbacklent "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lent"
	rsipullbacklsngp "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/lsngp"
	rsipullbacknvtk "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/reni"
	rsipullbacksibn "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/sibn"
	rsipullbacksvav "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/svav"
	rsipullbackwush "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/wush"
	rsipullbackydex "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ydex"
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
// names. That hypothesis is retired, and so is the single in-sample run that replaced it: on
// 2026-08-16 T went through its first FULL calibration (nine themes plus a point assembly,
// data/params/rsi_pullback/t/plateau_point.json, pooled OOS PF 3.412 on 35 trades with no
// degenerate fold). The snapshot below is that point. Exactly two fields moved against the
// previous literal — UseTrail 0 -> 1 and TrailDailyATR 0 -> 0.5 — and the pairing matters:
// DesiredStop takes the GREATEST active level, so a trail equal to the fixed stop starts the
// position exactly where the old literal did and only tightens after favourable closes. Пин
// живёт здесь, потому что этот тест ловит другое, чем снимок в strategy/tbank: он проверяет
// путь ЧЕРЕЗ РЕЕСТР бэктеста, то есть что калибратор и живой раннер видят один и тот же
// литерал.
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
		UseTrail:        1,
		TrailDailyATR:   0.5,
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

// TestRSIPullbackLENTIsRegisteredAndCalibrated пинует два неотличимых снаружи факта: LENT есть в
// карте (а не проваливается в generic-ветку) И возвращает собственный литерал, а не baseline.
// Пакет strategy/lent заведён 2026-08-14 ДО калибровки, и в тот же день сторожевой тест
// TestRSIPullbackLENTTracksBaseline, державший это состояние, заменён этим снимком.
//
// Отдельно от снимка: тикер НЕ подтверждён штатным протоколом §8 — тематические прогоны дали
// pooled OOS PF 1.355 по входу и 1.777 по тренду при объявленной заранее планке 1.5 на обеих
// темах, причём ведущая ось не устойчива ни там, ни там (RSILower 35/50/20/25, EMASlow
// 150/100/100/50 при требуемых трёх совпадениях из четырёх). Литерал собран по лидербордам и
// перепроверен точечным walk-forward (pooled PF 2.487 на 120 сделках, все четыре фолда выше
// 1.5), но для фиксированной точки это не out-of-sample: конфигурацию выбрал человек, видевший
// всю историю. Наличие литерала здесь означает снятую техническую блокировку, а не право
// ставить LENT в боевую вселенную — это решение владельца, как было с RENI, DOMRF, FESH и WUSH.
// Снимок литерала и обоснование пяти уязвимых полей живут рядом с ним, в strategy/lent/lent_test.go.
func TestRSIPullbackLENTIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["LENT"]
	if !ok {
		t.Fatal("LENT отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("LENT: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("LENT вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
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

// TestRSIPullbackLSNGPIsRegisteredAndCalibrated пинует два неотличимых снаружи факта: LSNGP есть
// в карте (а не проваливается в generic-ветку) И возвращает собственный литерал, а не baseline.
// Пакет strategy/lsngp заведён 2026-08-14 ДО калибровки, и в тот же день сторожевой тест
// TestRSIPullbackLSNGPTracksBaseline, державший это состояние, заменён этим снимком.
//
// Отдельно от снимка: тикер НЕ подтверждён штатным протоколом §8. Все девять тем дали pooled OOS
// PF выше 1.5 на пулах 98–174 сделок — такого не было ни у одного предыдущего тикера, — но
// объявленную заранее планку заваливает УСТОЙЧИВОСТЬ: ведущая ось выбрана двумя разными
// значениями из четырёх на обеих ключевых темах (RSILower 45/20/20/15, EMASlow 170/70/150/70)
// при требуемых трёх совпадениях. Литерал собран по лидербордам и перепроверен точечным
// walk-forward (pooled PF 3.006 на 274 сделках, все четыре фолда выше 2), но для фиксированной
// точки это не out-of-sample: конфигурацию выбрал человек, видевший всю историю. У LSNGP это
// весит больше обычного — растут ОБА окна протокола, падающего рынка в истории нет вовсе.
// Наличие литерала здесь означает снятую техническую блокировку, а не право ставить LSNGP в
// боевую вселенную. Снимок литерала и обоснование пяти уязвимых полей живут рядом с ним, в
// strategy/lsngp/lsngp_test.go.
func TestRSIPullbackLSNGPIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["LSNGP"]
	if !ok {
		t.Fatal("LSNGP отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("LSNGP: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("LSNGP вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbacklsngp.DefaultParams(); p != want {
		t.Fatalf("LSNGP params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "LSNGP" {
		t.Fatalf("Ticker() = %q, want LSNGP", got)
	}
}

// TestRSIPullbackNVTKIsRegisteredAndCalibrated пинует два неотличимых снаружи факта: NVTK есть в
// карте (а не проваливается в generic-ветку) И возвращает собственный литерал, а не baseline.
// До 2026-08-16 пакет strategy/nvtk был единственным в реестре, кто отслеживал baseline: он
// возвращал core.DefaultParams(), и живой раннер прямо запрещал ставить тикер в боевую вселенную.
// Калибровка 2026-08-16 это состояние сняла.
//
// Отдельно от снимка: тикер НЕ подтверждён штатным протоколом §8, причём слабее всех
// предшественников — выше объявленной планки 1.5 поднялась ОДНА тема из девяти (volume
// 1.674/53), ключевые дали entry 1.218/120 и trend 1.044/71, а четыре темы легли на единицу или
// ниже. Литерал собран не по лидербордам тем (они меряют каждую ось поверх дефолтов, а дефолты
// на NVTK стоят там, где стратегии нет), а точечными прогонами, и перепроверен фиксированной
// точкой: pooled PF 2.823 на 93 сделках, все четыре фолда выше 1.89. Для фиксированной точки это
// НЕ out-of-sample — конфигурацию выбрал человек, видевший всю историю. Наличие литерала
// означает снятую техническую блокировку, а не право ставить NVTK в боевую вселенную: это
// решение владельца, как было с domrf, fesh, wush и lent. Снимок литерала и обоснование
// уязвимых полей живут рядом с ним, в strategy/nvtk/nvtk_test.go.
func TestRSIPullbackNVTKIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["NVTK"]
	if !ok {
		t.Fatal("NVTK отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("NVTK: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("NVTK вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbacknvtk.DefaultParams(); p != want {
		t.Fatalf("NVTK params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "NVTK" {
		t.Fatalf("Ticker() = %q, want NVTK", got)
	}
}

// TestRSIPullbackIVATIsRegisteredAndCalibrated пинует два неотличимых снаружи факта: IVAT есть в
// карте (а не проваливается в generic-ветку) И возвращает собственный литерал, а не baseline.
// Пакет strategy/ivat заведён 2026-08-17 ДО калибровки, и 2026-08-18 сторожевой тест
// TestRSIPullbackIVATTracksBaseline, державший это состояние, заменён этим снимком.
//
// Отдельно от снимка: тикер НЕ подтверждён протоколом, причём дважды. Во-первых, у IVAT 26.0
// месяца истории, поэтому штатная схема §8 неисполнима и прогоны шли по адаптированной
// (-months 25 -train-months 9 -test-months 4) — числа не сопоставимы построчно с каталогом.
// Во-вторых, объявленную заранее планку заваливают ОБЕ ключевые темы: entry дала pooled OOS PF
// 0.979 (первый тикер каталога, у которого тема входа ушла ниже единицы), trend — 1.282 при
// устойчивости EMASlow 2 из 4, с разворотом выбора на середине истории. Литерал собран
// точечными прогонами и перепроверен walk-forward принятой точки (pooled 1.988 на 55 сделках),
// но 85% этого результата делает одна неделя июля 2026: на OOS первых трёх фолдов остаётся
// 1.209. Наличие литерала здесь означает снятую техническую блокировку и решение владельца
// «литерал ставится и при непройденной планке», а не подтверждённый протоколом edge. Снимок
// литерала и обоснование каждого поля живут рядом с ним, в strategy/ivat/ivat_test.go.
func TestRSIPullbackIVATIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["IVAT"]
	if !ok {
		t.Fatal("IVAT отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("IVAT: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("IVAT вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbackivat.DefaultParams(); p != want {
		t.Fatalf("IVAT params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "IVAT" {
		t.Fatalf("Ticker() = %q, want IVAT", got)
	}
}

// TestRSIPullbackSVAVIsRegisteredAndCalibrated пинует два неотличимых снаружи факта: SVAV есть в
// карте (а не проваливается в generic-ветку) И возвращает собственный литерал, а не baseline.
// Пакет strategy/svav заведён 2026-08-18 ДО калибровки, и 2026-08-21 сторожевой тест
// TestRSIPullbackSVAVTracksBaseline, державший это состояние, заменён этим снимком.
//
// Отдельно от снимка: SVAV — ПЕРВЫЙ тикер каталога, взявший объявленную заранее планку
// целиком, обеими ключевыми темами. entry дала pooled OOS PF 2.139 на 47 сделках (фолды
// 4.659/6.771/0.568/1.409 при 22/10/4/11), ведущая ось RSILower устойчива 3 из 4 (25/20/20/20);
// оговорка — фолд 3 переобучен (in-sample 7.893 против OOS 0.568, ×13.9), но условие «≥5 сделок
// в фолде» план снял сознательно, поэтому планку это не рушит. trend дала pooled OOS PF 2.343
// на 104 сделках (фолды 2.369/2.666/2.751/1.930 при 33/31/21/19), ведущая ось EMASlow устойчива
// 3 из 4 (170/100/100/100), без переобучения. У девяти предыдущих тикеров каталога (GAZP, NVTK,
// UGLD, FESH, WUSH, LENT, RENI, LSNGP, IVAT) планка не бралась целиком ни разу. Снимок литерала
// и обоснование каждого поля живут рядом с ним, в strategy/svav/svav_test.go и в доке пакета.
func TestRSIPullbackSVAVIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["SVAV"]
	if !ok {
		t.Fatal("SVAV отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("SVAV: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("SVAV вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbacksvav.DefaultParams(); p != want {
		t.Fatalf("SVAV params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "SVAV" {
		t.Fatalf("Ticker() = %q, want SVAV", got)
	}
}

// TestRSIPullbackSIBNIsRegisteredAndCalibrated пинует два неотличимых снаружи факта: SIBN есть в
// карте (а не проваливается в generic-ветку) И возвращает собственный литерал, а не baseline.
// Пакет strategy/sibn заведён 2026-08-21 ДО калибровки, и в тот же день сторожевой тест
// TestRSIPullbackSIBNTracksBaseline, державший это состояние, заменён этим снимком.
//
// ВЕРДИКТ ПО ПЛАНКЕ: НЕ ВЗЯТА, обеими половинами и на обеих ключевых темах. entry дала pooled OOS
// PF 1.378 на 50 сделках при устойчивости ведущей оси RSILower 1 из 4 (по фолдам 35/30/15/20 —
// все четыре значения разные); trend дала pooled OOS PF 0.847 на 94 сделках при устойчивости
// EMASlow 0 из 4 (100/120/50/200) — ни одна другая тема каталога не давала нулевой устойчивости.
// Ни одна из девяти тем SIBN не достала 1.5, три из девяти убыточны. Литерал собран точечными
// свипами поверх принятой зоны риска и перепроверен walk-forward принятой точки (pooled 2.258 на
// 89 сделках, вырожденных фолдов нет, результат размазан по 47 неделям), но наличие литерала
// здесь означает решение владельца «литерал ставится и при непройденной планке», а не
// подтверждённый протоколом edge — ровно как у IVAT. Снимок литерала и обоснование каждого поля
// живут рядом с ним, в strategy/sibn/sibn_test.go и в доке пакета.
func TestRSIPullbackSIBNIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["SIBN"]
	if !ok {
		t.Fatal("SIBN отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("SIBN: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("SIBN вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbacksibn.DefaultParams(); p != want {
		t.Fatalf("SIBN params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "SIBN" {
		t.Fatalf("Ticker() = %q, want SIBN", got)
	}
}

// TestRSIPullbackELFVIsRegisteredAndCalibrated сторожит, что ELFV живёт в rsiPullbackRegistry
// собственной записью (а не проваливается в generic-ветку) И возвращает собственный литерал, а не
// baseline. Пакет strategy/elfv заведён 2026-08-21 ДО калибровки, и 2026-08-22 сторожевой тест
// TestRSIPullbackELFVTracksBaseline, державший это состояние, заменён этим снимком.
//
// ВЕРДИКТ ПО ПЛАНКЕ: НЕ ВЗЯТА — обе ключевые темы прошли по profit factor и обе провалились по
// устойчивости. entry дала pooled OOS PF 1.575 на 106 сделках при устойчивости ведущей оси
// RSILower 1 из 4 (по фолдам 30/20/25/35); trend — pooled OOS PF 1.545 на 147 сделках при
// устойчивости EMASlow 0 из 4 (200/120/150/50). Это первый тикер каталога, где обе ключевые темы
// взяли порог 1.5 по качеству и обе не взяли его по устойчивости: у SIBN обе проваливали и то, и
// другое, у SVAV — брали и то, и другое. Все десять тем ELFV дали pooled выше 1.38, ни одна не
// убыточна, но ни одна ось не устойчива больше чем на 2 фолда из 4. Литерал собран точечными
// свипами поверх принятой зоны риска и перепроверен walk-forward принятой точки (pooled 2.863 на
// 68 сделках, все четыре фолда прибыльные, вырожденных нет, лучшая неделя даёт 28.3% результата),
// но наличие литерала здесь означает решение владельца «литерал ставится и при непройденной
// планке», а не подтверждённый протоколом edge — ровно как у IVAT и SIBN. Снимок литерала и
// обоснование каждого поля живут рядом с ним, в strategy/elfv/elfv_test.go и в доке пакета.
func TestRSIPullbackELFVIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["ELFV"]
	if !ok {
		t.Fatal("ELFV отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("ELFV: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("ELFV вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbackelfv.DefaultParams(); p != want {
		t.Fatalf("ELFV params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "ELFV" {
		t.Fatalf("Ticker() = %q, want ELFV", got)
	}
}

// TestRSIPullbackDIASIsRegisteredAndCalibrated сторожит, что DIAS живёт в rsiPullbackRegistry
// собственной записью (а не проваливается в generic-ветку) И возвращает собственный литерал, а не
// baseline. Пакет strategy/dias заведён 2026-08-22 ДО калибровки, и 2026-08-23 сторожевой тест
// TestRSIPullbackDIASTracksBaseline, державший это состояние, заменён этим снимком.
//
// ВЕРДИКТ ПО ПЛАНКЕ: НЕ ВЗЯТА. Схема прогонов АДАПТИРОВАНА: физическая история DIAS 30.3 месяца
// (IPO 2024-02-13), штатный протокол §8 docs/rsi_pullback/strategy.md (36/12/6) неисполним;
// использована схема -months 30 -train-months 10 -test-months 5 (четыре фолда train 10 + OOS 5,
// прецедент IVAT 25/9/4) — числа DIAS сопоставимы с IVAT, но НЕ построчно с остальным каталогом.
// Тема entry провалила ОБА критерия планки: pooled OOS PF 1.213 на 73 сделках при пороге 1.5, и
// устойчивость ведущей оси RSILower 2 из 4 (по фолдам 25/25/40/40, порог >= 3 из 4). Тема trend
// взяла только критерий PF: pooled OOS PF 1.848 на 116 сделках, но устойчивость ведущей оси
// EMASlow тоже 2 из 4 (150/100/50/50). Обе ключевые темы не набрали устойчивость выбора — планка
// целиком не взята. Литерал собран точечными прогонами поверх принятой зоны риска и перепроверен
// walk-forward принятой точки (pooled OOS PF 2.361 на 98 сделках, все четыре фолда прибыльные,
// вырожденных нет — этого не добилась ни одна из десяти тем), но наличие литерала здесь означает
// решение владельца от 2026-08-22 «литерал ставится и при непройденной планке, если принятая точка
// не упирается в стоп-условие плана», а не подтверждённый протоколом edge — ровно как у IVAT, SIBN
// и ELFV. Снимок литерала и обоснование каждого поля живут рядом с ним, в
// strategy/dias/dias_test.go и в доке пакета.
func TestRSIPullbackDIASIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["DIAS"]
	if !ok {
		t.Fatal("DIAS отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("DIAS: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("DIAS вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbackdias.DefaultParams(); p != want {
		t.Fatalf("DIAS params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "DIAS" {
		t.Fatalf("Ticker() = %q, want DIAS", got)
	}
}

// TestRSIPullbackBSPBIsRegisteredAndCalibrated сторожит, что BSPB живёт в rsiPullbackRegistry
// собственной записью (а не проваливается в generic-ветку) И возвращает собственный литерал, а не
// baseline. Пакет strategy/bspb заведён 2026-08-24 ДО калибровки, и в тот же день сторожевой тест
// TestRSIPullbackBSPBTracksBaseline, державший это состояние, заменён этим снимком.
//
// ВЕРДИКТ ПО ПЛАНКЕ: НЕ ВЗЯТА. Схема прогонов ШТАТНАЯ (36/12/6, история ровно 36 месяцев), поэтому
// числа BSPB сопоставимы построчно со всем каталогом, кроме IVAT и DIAS. Тема entry провалила ОБА
// критерия планки: pooled OOS PF 0.726 на 69 сделках при пороге 1.5, и устойчивость ведущей оси
// RSILower 2 из 4 (по фолдам 15/35/25/15, порог >= 3 из 4). Тема trend взяла только критерий
// устойчивости: ведущая ось EMASlow по фолдам 30/50/30/30 — 3 из 4, но pooled OOS PF 0.980 на 95
// сделках, порог 1.5 НЕ взят. Три критерия из четырёх провалены, и форма провала редкая: обе
// ключевые темы ушли ниже ЕДИНИЦЫ, ранняя тема screen тоже (0.670). Литерал собран гибридной
// процедурой — три ранние темы поверх дефолтов ядра, семь поздних поверх якоря Task 6, выбранного
// на полной истории, — и перепроверен собственным walk-forward принятой точки (pooled OOS PF 2.336
// на 47 сделках, все четыре фолда прибыльные, вырожденных нет; под удвоенными издержками 1.829).
// Четырёх прибыльных фолдов добились и четыре темы BSPB, ближайшая cal_exit даёт 2.261/47 — на 3.3%
// ниже точки, так что разрыва между точкой и лучшей из поздних тем нет. Наличие литерала здесь
// означает решение владельца «литерал ставится и при непройденной планке, если принятая точка не
// упирается в стоп-условие плана» (не сработало ни по одному из трёх пунктов), а не подтверждённый
// протоколом edge — ровно как у IVAT, SIBN, ELFV и DIAS. Снимок литерала и обоснование каждого поля
// живут рядом с ним, в strategy/bspb/bspb_test.go и в доке пакета.
func TestRSIPullbackBSPBIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry["BSPB"]
	if !ok {
		t.Fatal("BSPB отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("BSPB: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("BSPB вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbackbspb.DefaultParams(); p != want {
		t.Fatalf("BSPB params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "BSPB" {
		t.Fatalf("Ticker() = %q, want BSPB", got)
	}
}

// TestRSIPullbackYDEXIsRegisteredAndCalibrated сторожит, что YDEX живёт в rsiPullbackRegistry
// собственной записью (а не проваливается в generic-ветку) И возвращает собственный литерал, а не
// baseline. Пакет strategy/ydex заведён 2026-08-25 ДО калибровки, и в тот же день сторожевой тест
// TestRSIPullbackYDEXTracksBaseline, державший это состояние, заменён этим снимком.
//
// ВЕРДИКТ ПО ПЛАНКЕ: НЕ ВЗЯТА. Схема прогонов АДАПТИРОВАНА (окно укорочено из-за редомициляции
// YNDX->YDEX: -months 25 -train-months 9 -test-months 4). Тема entry взяла критерий PF (pooled
// OOS PF 1.916 на 52 сделках) при устойчивости ведущей оси RSILower 2 из 4 (по фолдам
// 10/25/35/25); тема trend взяла критерий PF (pooled OOS PF 1.642 на 71 сделке) при устойчивости
// ведущей оси EMASlow тоже 2 из 4 (200/50/200/70). ФОРМА ПРОВАЛА ПОВТОРЯЕТ ELFV — это второй такой
// случай в каталоге: обе ключевые темы взяли PF и обе провалили устойчивость. Литерал собран
// точечными прогонами поверх принятой зоны риска и перепроверен собственным walk-forward точки
// (pooled OOS PF 3.014 на 77 сделках, все четыре фолда прибыльные, вырожденных нет; под
// удвоенными издержками 2.167), но наличие литерала здесь означает решение владельца «литерал
// ставится и при непройденной планке, если принятая точка не упирается в стоп-условие плана» (не
// сработало ни по одному из трёх пунктов), а не подтверждённый протоколом edge — ровно как у
// IVAT, SIBN, ELFV, DIAS и BSPB. Снимок литерала и обоснование каждого поля живут рядом с ним, в
// strategy/ydex/ydex_test.go и в доке пакета.
func TestRSIPullbackYDEXIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbackydex.Ticker]
	if !ok {
		t.Fatal("YDEX отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("YDEX: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("YDEX вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbackydex.DefaultParams(); p != want {
		t.Fatalf("YDEX params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "YDEX" {
		t.Fatalf("Ticker() = %q, want YDEX", got)
	}
}

// TestRSIPullbackBANEPIsRegisteredAndCalibrated сторожит, что BANEP живёт в rsiPullbackRegistry
// собственной записью (а не проваливается в generic-ветку) И возвращает собственный литерал, а не
// baseline. Пакет strategy/banep заведён 2026-08-28 до калибровки, и в тот же день сторожевой тест
// TestRSIPullbackBANEPTracksBaseline, державший это состояние, заменён этим снимком.
//
// ВЕРДИКТ ПО ПЛАНКЕ: НЕ ВЗЯТА, обеими ключевыми темами и по обоим критериям — форма провала тяжелее,
// чем у ELFV и YDEX. Тема entry: pooled OOS PF 1.004 на 74 сделках (требуется 1.5), ведущая ось
// RSILower выбрана как 45/25/30/25 — два фолда из четырёх. Тема trend: 1.258 на 81 сделке, ведущая
// ось EMASlow — 50/40/50/70, тоже два из четырёх. Вырожденных фолдов нет. Принятая точка при этом
// перепроверена собственным walk-forward (pooled OOS PF 1.799 на 67 сделках, три фолда из четырёх
// прибыльны, под удвоенными издержками 1.317) и не упирается ни в один из трёх пунктов
// стоп-условия. Литерал стоит по решению владельца «литерал ставится и при непройденной планке,
// если принятая точка не упирается в стоп-условие» — ровно как у IVAT, SIBN, ELFV, DIAS, BSPB и
// YDEX. Сетки этого тикера по отдельному решению владельца держатся МАКСИМАЛЬНО ШИРОКИМИ; что это
// изменило (ровно одну ось — риск), разобрано в доке пакета strategy/banep.
func TestRSIPullbackBANEPIsRegisteredAndCalibrated(t *testing.T) {
	b, ok := rsiPullbackRegistry[rsipullbackbanep.Ticker]
	if !ok {
		t.Fatal("BANEP отсутствует в rsiPullbackRegistry: тикер провалится в generic-ветку")
	}
	p, pok := b.DefaultParams().(core.Params)
	if !pok {
		t.Fatalf("BANEP: DefaultParams() вернул %T, want core.Params", b.DefaultParams())
	}
	if p == core.DefaultParams() {
		t.Fatal("BANEP вернул baseline: откалиброванный тикер обязан иметь собственный литерал")
	}
	if want := rsipullbackbanep.DefaultParams(); p != want {
		t.Fatalf("BANEP params = %+v, want литерал пакета %+v", p, want)
	}
	if got := b.Build(p).Ticker(); got != "BANEP" {
		t.Fatalf("Ticker() = %q, want BANEP", got)
	}
}
