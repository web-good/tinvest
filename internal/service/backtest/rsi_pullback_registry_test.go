package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
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

// TestRSIPullbackTBankStartsFromGAZPConfig pins a DELIBERATE hypothesis, not a fact: T starts
// from GAZP's post-grid literal to test whether parameters transfer between liquid names. The
// equality is pinned so the link cannot dissolve unnoticed — once T is calibrated on its own
// data, this test must be rewritten to pin T's own literal, and the rewrite is the moment the
// hypothesis gets consciously retired.
func TestRSIPullbackTBankStartsFromGAZPConfig(t *testing.T) {
	tbRaw := RSIPullbackLookupOrGeneric("T").DefaultParams()
	tb, ok := tbRaw.(core.Params)
	if !ok {
		t.Fatalf("T: DefaultParams() returned %T, want core.Params", tbRaw)
	}
	gzRaw := RSIPullbackLookupOrGeneric("GAZP").DefaultParams()
	gz, ok := gzRaw.(core.Params)
	if !ok {
		t.Fatalf("GAZP: DefaultParams() returned %T, want core.Params", gzRaw)
	}
	if tb != gz {
		t.Fatalf("T params = %+v, want GAZP's %+v — T is seeded from the GAZP config on purpose", tb, gz)
	}
	if tb == core.DefaultParams() {
		t.Fatal("T returns the baseline: the GAZP seed was lost")
	}
	if got := RSIPullbackLookupOrGeneric("T").Build(tb).Ticker(); got != "T" {
		t.Fatalf("Ticker() = %q, want T", got)
	}
}
