# AFKS Ticker + Adaptive Core Extraction — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add AFK Sistema (AFKS) as a fully-supported scalping ticker by extracting the generic adaptive strategy core into a shared `adaptive` package, leaving `rusal` and a new `afks` as thin per-share configs, then run a RUAL-vs-AFKS comparison backtest.

**Architecture:** The decision logic currently in package `rusal` is fully ticker-agnostic; only `const ticker` and `DefaultParams()` are share-specific. We move the core (Params, Strategy, Decide, decide, regimeOf, emaTouched, recentHigh, Lookback) into `strategy/adaptive`, parametrizing the ticker via `NewWithParams(ticker, params)`. `rusal` and `afks` shrink to `Ticker` + `DefaultParams()` + `New()`. The backtest registry and live runner register both tickers; the live runner is already multi-ticker. The refactor is behaviour-preserving — existing RUAL tests guard it.

**Tech Stack:** Go 1.25, standard `testing`, existing backtest engine (`internal/domain/backtest`), Tinkoff Invest gRPC for candle fetch.

**Spec:** `docs/superpowers/specs/2026-06-06-afks-ticker-and-adaptive-extraction-design.md`

---

## File Structure

```
internal/service/trading_strategy/scalping/strategy/
  adaptive/
    adaptive.go        # NEW: Params, Strategy{ticker,p}, NewWithParams(ticker,p), Decide,
                       #      decide, regimeOf, regime consts, decideInput, emaTouched, recentHigh
    adaptive_test.go   # NEW: ticker-agnostic core tests (moved from rusal_test.go)
  rusal/
    rusal.go           # SHRINK: const Ticker; DefaultParams() adaptive.Params; New()
    rusal_test.go      # SHRINK: RUAL-config tests only (ticker, Lookback, defaults, smoke Decide)
  afks/
    afks.go            # NEW: const Ticker; DefaultParams() adaptive.Params; New()
    afks_test.go       # NEW: AFKS-config tests
internal/service/backtest/
  registry.go          # MODIFY: rusal binding -> adaptive; add AFKS binding
  registry_test.go     # MODIFY: rusal.Params -> adaptive.Params; add AFKS binding tests
  calibrate_test.go    # MODIFY: rusal.Params -> adaptive.Params type assertions
internal/service/trading_strategy/scalping/
  registry.go          # MODIFY: defaultStrategies() returns rusal.New() + afks.New()
data/params/afks/
  scalp.json           # NEW: starter single param set
  grid.json            # NEW: calibration grid
data/candles/
  AFKS_Hour1.json      # NEW (fetched at run time)
  AFKS_Day1.json       # NEW (fetched at run time)
```

---

## Task 1: Extract the `adaptive` core package

**Files:**
- Create: `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go`
- Create: `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go`

This moves the generic logic out of `rusal` verbatim, with two changes: `Strategy` gains a `ticker string` field set by `NewWithParams(ticker, p)`, and `Decide` sets `sig.Ticker = s.ticker` instead of a hardcoded constant. `DefaultParams` is intentionally NOT in this package (per-share configs own it).

- [ ] **Step 1: Write the core test file**

Create `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go`:

```go
package adaptive

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// testParams uses clean levels/multipliers so expected stops/targets are exact.
// Indicator periods are irrelevant where decide() consumes pre-computed scalars.
func testParams() Params {
	return Params{
		EMAPeriod: 3, ADXPeriod: 2, ADXTrendLevel: 25, ADXRangeLevel: 20,
		RSIPeriod: 2, RSITrendLevel: 45, RSIRangeLevel: 35,
		PullbackWindow: 5, DonchianPeriod: 3, ATRPeriod: 2,
		SLMult: 1.0, TrailMult: 2.0, ChandelierWindow: 3,
		EMATouchTol: 0.002, BandTol: 0.003,
	}
}

func TestNewWithParamsTicker(t *testing.T) {
	if got := NewWithParams("ZZZ", testParams()).Ticker(); got != "ZZZ" {
		t.Errorf("Ticker = %q, want ZZZ", got)
	}
}

func TestEMATouched(t *testing.T) {
	ema := []float64{10, 10, 10, 10, 10}
	lows := []float64{12, 12, 12, 10.01, 12}
	if !emaTouched(lows, ema, 3, 0.002) {
		t.Error("expected touch within last 3 bars")
	}
	if emaTouched(lows, ema, 1, 0.002) {
		t.Error("did not expect touch within last 1 bar")
	}
	if emaTouched(nil, nil, 3, 0.002) {
		t.Error("empty input must not touch")
	}
}

func TestRecentHigh(t *testing.T) {
	highs := []float64{5, 9, 3, 7, 4}
	if got := recentHigh(highs, 3); got != 7 {
		t.Errorf("recentHigh = %v, want 7", got)
	}
	if got := recentHigh(highs, 10); got != 9 {
		t.Errorf("recentHigh = %v, want 9", got)
	}
	if got := recentHigh(nil, 3); got != 0 {
		t.Errorf("recentHigh(nil) = %v, want 0", got)
	}
	if got := recentHigh([]float64{5, 9, 3, 7, 4}, 0); got != 4 {
		t.Errorf("recentHigh(window=0) = %v, want 4", got)
	}
}

func TestRegimeOf(t *testing.T) {
	s := NewWithParams("TST", testParams()) // ADXTrendLevel 25, ADXRangeLevel 20
	cases := []struct {
		adx  float64
		want regime
	}{
		{0, regimeDead},
		{-5, regimeDead},
		{30, regimeTrend},
		{25, regimeTrend},
		{15, regimeRange},
		{20, regimeRange},
		{22, regimeDead},
	}
	for _, c := range cases {
		if got := s.regimeOf(c.adx); got != c.want {
			t.Errorf("regimeOf(%v) = %v, want %v", c.adx, got, c.want)
		}
	}
}

func TestDecideCore(t *testing.T) {
	s := NewWithParams("TST", testParams())

	tests := []struct {
		name       string
		in         decideInput
		wantKind   model.SignalKind
		wantReason string
		wantTP     float64
		wantSL     float64
	}{
		{
			name: "trend entry: pullback + rsi cross + di+",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
			},
			wantKind: model.SignalBuy, wantTP: 104, wantSL: 98,
		},
		{
			name: "trend no pullback -> none",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: false,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend di+ < di- -> none",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 10, diMinus: 25, emaTouched: true,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend rsi did not cross -> none",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 46, rsiNow: 50,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend entry blocked when price not above ema",
			in: decideInput{
				price: 100, atr: 2, emaNow: 101, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "range entry: at lower band + rsi cross",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
			},
			wantKind: model.SignalBuy, wantTP: 104.9, wantSL: 98,
		},
		{
			name: "trend entry blocked by HTF filter (filter on, trend down)",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
				trendFilterOn: true, trendUp: false,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend entry allowed when HTF filter passes",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
				trendFilterOn: true, trendUp: true,
			},
			wantKind: model.SignalBuy, wantTP: 104, wantSL: 98,
		},
		{
			name: "range entry blocked by HTF filter (filter on, trend down)",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
				trendFilterOn: true, trendUp: false,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "range entry allowed when HTF filter passes",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
				trendFilterOn: true, trendUp: true,
			},
			wantKind: model.SignalBuy, wantTP: 104.9, wantSL: 98,
		},
		{
			name: "range mid-channel (not near lower) -> none",
			in: decideInput{
				price: 105, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "dead zone flat -> none",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 30, rsiNow: 46,
				adx: 22, diPlus: 25, diMinus: 10, emaTouched: true,
				donUpper: 110, donLower: 99.8,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "range exit: take profit at mid",
			in: decideInput{
				price: 106, atr: 2, adx: 15, donUpper: 110, donLower: 100,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "TP", wantTP: 105, wantSL: 98,
		},
		{
			name: "range exit: stop loss",
			in: decideInput{
				price: 97, atr: 2, adx: 15, donUpper: 110, donLower: 100,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "SL", wantTP: 105, wantSL: 98,
		},
		{
			name: "range exit: degenerate donchian does not fire spurious TP",
			in: decideInput{
				price: 99, atr: 2, adx: 15, donUpper: 0, donLower: 0,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend exit: chandelier trail",
			in: decideInput{
				price: 105, atr: 2, adx: 30, chandelierHigh: 110,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "TRAIL", wantSL: 106,
		},
		{
			name: "trend exit: initial stop wins over trail",
			in: decideInput{
				price: 97.5, atr: 2, adx: 30, chandelierHigh: 101,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "SL", wantSL: 98,
		},
		{
			name: "trend hold while rising -> none",
			in: decideInput{
				price: 118, atr: 2, adx: 30, chandelierHigh: 120,
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.decide(tt.in)
			if got.Kind != tt.wantKind {
				t.Fatalf("Kind = %v, want %v", got.Kind, tt.wantKind)
			}
			if tt.wantKind == model.SignalNone {
				return
			}
			if got.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
			if tt.wantTP != 0 && got.TakeProfit != tt.wantTP {
				t.Errorf("TakeProfit = %v, want %v", got.TakeProfit, tt.wantTP)
			}
			if tt.wantSL != 0 && got.StopLoss != tt.wantSL {
				t.Errorf("StopLoss = %v, want %v", got.StopLoss, tt.wantSL)
			}
		})
	}
}

func TestDecide_DailyFilterGate(t *testing.T) {
	p := testParams()
	p.TrendFilterPeriod = 3 // tiny period so a short daily series is enough

	highs := make([]float64, 200)
	lows := make([]float64, 200)
	closes := make([]float64, 200)
	for i := 0; i < 200; i++ {
		base := 100.0 + float64(i)
		highs[i] = base + 1
		lows[i] = base - 1
		closes[i] = base
	}

	upDaily := []float64{10, 20, 30, 40}
	downDaily := []float64{40, 30, 20, 10}

	mk := func(daily []float64) strategy.MarketData {
		return strategy.MarketData{
			Price: closes[199], Highs: highs, Lows: lows, Closes: closes,
			DailyCloses: daily,
		}
	}

	s := NewWithParams("TST", p)
	if got := s.Decide(mk(upDaily)); got.Kind == model.SignalBuy {
		t.Fatalf("unexpected Buy on rising-only market (up daily)")
	}
	if got := s.Decide(mk(downDaily)); got.Kind == model.SignalBuy {
		t.Fatalf("unexpected Buy on rising-only market (down daily)")
	}
	if got := s.Decide(mk([]float64{1, 2})); got.Kind == model.SignalBuy {
		t.Fatalf("unexpected Buy on cold-start daily series")
	}

	p0 := testParams()
	p0.TrendFilterPeriod = 0
	if got := NewWithParams("TST", p0).Decide(mk(nil)); got.Kind != model.SignalNone {
		t.Fatalf("filter off changed behavior: got %v, want None", got.Kind)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails to compile**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/adaptive/ 2>&1 | head`
Expected: build failure — `undefined: Params`, `undefined: NewWithParams`, etc. (the package has no implementation yet).

- [ ] **Step 3: Create the `adaptive.go` implementation**

Create `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go`:

```go
package adaptive

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds every tunable for the adaptive ADX-regime strategy. They are
// exposed so a ticker can be calibrated on real history without touching the
// decision logic.
type Params struct {
	EMAPeriod         int     // fast EMA for the trend pullback
	ADXPeriod         int     // ADX/DMI period
	ADXTrendLevel     float64 // ADX >= -> trend regime
	ADXRangeLevel     float64 // ADX <= -> range regime (between the two = dead zone)
	RSIPeriod         int     // RSI period
	RSITrendLevel     float64 // RSI reversal threshold in trend (shallow pullbacks)
	RSIRangeLevel     float64 // RSI reversal threshold in range (oversold)
	PullbackWindow    int     // bars back over which an EMA "touch" still counts
	DonchianPeriod    int     // channel period: lower for entry, mid for range exit
	ATRPeriod         int     // ATR period for stops/trailing
	SLMult            float64 // initial stop = entry - SLMult*ATR
	TrailMult         float64 // chandelier = max(High over window) - TrailMult*ATR
	ChandelierWindow  int     // window for the chandelier high
	EMATouchTol       float64 // EMA touch tolerance (fraction, e.g. 0.002 = 0.2%)
	BandTol           float64 // lower-band proximity tolerance (fraction)
	TrendFilterPeriod int     // daily EMA period for the higher-timeframe long filter; 0 disables
}

// Strategy trades a single instrument adaptively: it picks a regime from ADX and
// applies mean-reversion in a range or momentum in a trend. It is ticker-agnostic;
// per-share packages supply the ticker and calibrated Params.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the adaptive strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window for ADX's double smoothing (the hungriest indicator).
func (s *Strategy) Lookback() int { return 6*s.p.ADXPeriod + s.p.DonchianPeriod + 50 }

// regime classifies the market from ADX.
type regime int

const (
	regimeDead regime = iota
	regimeTrend
	regimeRange
)

func (s *Strategy) regimeOf(adx float64) regime {
	switch {
	case adx <= 0: // ADX returns 0 on insufficient/invalid history — treat as no-signal
		return regimeDead
	case adx >= s.p.ADXTrendLevel:
		return regimeTrend
	case adx <= s.p.ADXRangeLevel:
		return regimeRange
	default:
		return regimeDead
	}
}

// decideInput carries already-computed indicator values into the pure decision core.
type decideInput struct {
	price          float64
	atr            float64
	emaNow         float64
	rsiPrev        float64
	rsiNow         float64
	adx            float64
	diPlus         float64
	diMinus        float64
	donUpper       float64
	donLower       float64
	emaTouched     bool
	chandelierHigh float64
	pos            *strategy.Position
	trendFilterOn  bool
	trendUp        bool
}

// Decide computes every indicator from md, packs them, and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	closes := md.Closes
	n := len(closes)

	emaSeries := ema.Compute(closes, s.p.EMAPeriod)
	rsiSeries := indicators.RSISeries(closes, s.p.RSIPeriod)
	atr := indicators.ATR(md.Highs, md.Lows, closes, s.p.ATRPeriod)
	adx, diPlus, diMinus := indicators.ADX(md.Highs, md.Lows, closes, s.p.ADXPeriod)
	donUpper, donLower := indicators.Donchian(md.Highs, md.Lows, s.p.DonchianPeriod)

	var emaNow, rsiPrev, rsiNow float64
	if n > 0 {
		emaNow = emaSeries[n-1]
	}
	if n >= 2 {
		rsiNow = rsiSeries[n-1]
		rsiPrev = rsiSeries[n-2]
	}

	trendFilterOn := s.p.TrendFilterPeriod > 0
	trendUp := false
	if trendFilterOn && len(md.DailyCloses) >= s.p.TrendFilterPeriod {
		emaD := ema.Compute(md.DailyCloses, s.p.TrendFilterPeriod)
		lastDaily := md.DailyCloses[len(md.DailyCloses)-1]
		trendUp = lastDaily > emaD[len(emaD)-1]
	}

	in := decideInput{
		price:          md.Price,
		atr:            atr,
		emaNow:         emaNow,
		rsiPrev:        rsiPrev,
		rsiNow:         rsiNow,
		adx:            adx,
		diPlus:         diPlus,
		diMinus:        diMinus,
		donUpper:       donUpper,
		donLower:       donLower,
		emaTouched:     emaTouched(md.Lows, emaSeries, s.p.PullbackWindow, s.p.EMATouchTol),
		chandelierHigh: recentHigh(md.Highs, s.p.ChandelierWindow),
		pos:            md.Position,
		trendFilterOn:  trendFilterOn,
		trendUp:        trendUp,
	}

	sig := s.decide(in)
	sig.Ticker = s.ticker
	return sig
}

// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(in decideInput) model.Signal {
	sig := model.Signal{Price: in.price, RSI: in.rsiNow}
	reg := s.regimeOf(in.adx)

	// Manage an open position (long-only): exits are regime-dependent.
	if in.pos != nil {
		hardSL := in.pos.PurchasePrice - s.p.SLMult*in.atr
		if reg == regimeTrend {
			chandelier := in.chandelierHigh - s.p.TrailMult*in.atr
			sig.StopLoss = hardSL
			switch {
			case in.price <= hardSL:
				sig.Kind, sig.Reason = model.SignalSell, "SL"
			case in.price <= chandelier:
				sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
			}
			return sig
		}
		// range or dead zone -> mean-reversion management.
		mid := (in.donUpper + in.donLower) / 2
		sig.StopLoss = hardSL
		sig.TakeProfit = mid
		switch {
		case in.price <= hardSL:
			sig.Kind, sig.Reason = model.SignalSell, "SL"
		case in.price >= mid && mid > 0:
			sig.Kind, sig.Reason = model.SignalSell, "TP"
		}
		return sig
	}

	// Flat -> regime-specific entries (long only).
	switch reg {
	case regimeTrend:
		crossedUp := in.rsiPrev < s.p.RSITrendLevel && in.rsiNow >= s.p.RSITrendLevel
		if in.diPlus > in.diMinus && in.emaTouched && crossedUp && in.price > in.emaNow &&
			(!in.trendFilterOn || in.trendUp) {
			sig.Kind = model.SignalBuy
			sig.StopLoss = in.price - s.p.SLMult*in.atr
			sig.TakeProfit = in.price + s.p.TrailMult*in.atr
		}
	case regimeRange:
		crossedUp := in.rsiPrev < s.p.RSIRangeLevel && in.rsiNow >= s.p.RSIRangeLevel
		if in.price <= in.donLower*(1+s.p.BandTol) && crossedUp &&
			(!in.trendFilterOn || in.trendUp) {
			sig.Kind = model.SignalBuy
			sig.StopLoss = in.price - s.p.SLMult*in.atr
			sig.TakeProfit = (in.donUpper + in.donLower) / 2
		}
	}
	return sig
}

// emaTouched reports whether a low dipped to the EMA (within tol) on any of the last
// `window` bars — the pullback condition for a trend entry.
func emaTouched(lows, ema []float64, window int, tol float64) bool {
	n := len(lows)
	if n == 0 || len(ema) != n {
		return false
	}
	start := n - window
	if start < 0 {
		start = 0
	}
	for i := start; i < n; i++ {
		if ema[i] > 0 && lows[i] <= ema[i]*(1+tol) {
			return true
		}
	}
	return false
}

// recentHigh returns the highest high over the last `window` bars (all bars if fewer).
// A non-positive window is clamped to the last bar so it can never index out of range.
func recentHigh(highs []float64, window int) float64 {
	n := len(highs)
	if n == 0 {
		return 0
	}
	start := n - window
	if start < 0 {
		start = 0
	}
	if start > n-1 {
		start = n - 1
	}
	h := highs[start]
	for i := start + 1; i < n; i++ {
		if highs[i] > h {
			h = highs[i]
		}
	}
	return h
}
```

- [ ] **Step 4: Run the adaptive tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/adaptive/ -v`
Expected: PASS — all tests green (`TestNewWithParamsTicker`, `TestEMATouched`, `TestRecentHigh`, `TestRegimeOf`, `TestDecideCore`, `TestDecide_DailyFilterGate`).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/adaptive/
git commit -m "feat(scalping): extract ticker-agnostic adaptive strategy core

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Shrink `rusal` to a thin per-share config

**Files:**
- Modify (overwrite): `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`
- Modify (overwrite): `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go`

`rusal` keeps the RUAL ticker constant and RUAL-calibrated defaults, delegating all behaviour to `adaptive`. The remaining RUAL tests assert config (ticker, Lookback, defaults) and two end-to-end smoke tests of `Decide`.

- [ ] **Step 1: Rewrite the RUAL config tests**

Overwrite `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go`:

```go
package rusal

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestTickerAndLookback(t *testing.T) {
	s := New()
	if s.Ticker() != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", s.Ticker())
	}
	// 6*14 + 20 + 50 = 154
	if got := s.Lookback(); got != 154 {
		t.Errorf("Lookback = %d, want 154", got)
	}
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	if p.ADXTrendLevel <= p.ADXRangeLevel {
		t.Errorf("ADXTrendLevel (%v) must exceed ADXRangeLevel (%v)", p.ADXTrendLevel, p.ADXRangeLevel)
	}
	if p.EMAPeriod <= 0 || p.ADXPeriod <= 0 || p.RSIPeriod <= 0 || p.DonchianPeriod <= 0 || p.ATRPeriod <= 0 {
		t.Errorf("all periods must be positive: %+v", p)
	}
	if p.TrendFilterPeriod != 100 {
		t.Errorf("TrendFilterPeriod = %d, want 100 (RUAL calibrated)", p.TrendFilterPeriod)
	}
}

// TestDecide_FlatUptrendIsNone: a monotonic uptrend keeps RSI high (no upward cross)
// and price runs above the EMA, so no entry fires.
func TestDecide_FlatUptrendIsNone(t *testing.T) {
	s := New()
	highs := make([]float64, 200)
	lows := make([]float64, 200)
	closes := make([]float64, 200)
	for i := 0; i < 200; i++ {
		base := 100.0 + float64(i)
		highs[i] = base + 1
		lows[i] = base - 1
		closes[i] = base
	}
	md := strategy.MarketData{Price: closes[199], Highs: highs, Lows: lows, Closes: closes}
	got := s.Decide(md)
	if got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", got.Kind)
	}
	if got.Ticker != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", got.Ticker)
	}
}

// TestDecide_CrushedPriceIsSL: with an open position and a collapsed price, the hard
// ATR stop fires regardless of regime — exercises the full Decide wiring end-to-end.
func TestDecide_CrushedPriceIsSL(t *testing.T) {
	s := New()
	highs := make([]float64, 200)
	lows := make([]float64, 200)
	closes := make([]float64, 200)
	for i := 0; i < 200; i++ {
		base := 100.0 + float64(i%5) // choppy, bounded
		highs[i] = base + 1
		lows[i] = base - 1
		closes[i] = base
	}
	md := strategy.MarketData{
		Price:    1, // crushed far below any stop
		Highs:    highs,
		Lows:     lows,
		Closes:   closes,
		Position: &strategy.Position{PurchasePrice: 100, Quantity: 1},
	}
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "SL" {
		t.Fatalf("got Kind=%v Reason=%q, want Sell/SL", got.Kind, got.Reason)
	}
	if got.Ticker != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", got.Ticker)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/ 2>&1 | head`
Expected: build failure — old `rusal.go` still defines the full struct/`decideInput`/etc. and may collide, or `New()` signature differs. (Either compile error or duplicate-symbol; both count as "not yet passing".)

- [ ] **Step 3: Rewrite `rusal.go` as a thin config**

Overwrite `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`:

```go
package rusal

import (
	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
)

// Ticker is the instrument this config targets.
const Ticker = "RUAL"

// DefaultParams returns RUAL-calibrated params for the adaptive strategy.
func DefaultParams() adaptive.Params {
	return adaptive.Params{
		EMAPeriod:         21,
		ADXPeriod:         14,
		ADXTrendLevel:     25,
		ADXRangeLevel:     20,
		RSIPeriod:         14,
		RSITrendLevel:     45,
		RSIRangeLevel:     35,
		PullbackWindow:    5,
		DonchianPeriod:    20,
		ATRPeriod:         14,
		SLMult:            1.0,
		TrailMult:         2.5,
		ChandelierWindow:  20,
		EMATouchTol:       0.002,
		BandTol:           0.003,
		TrendFilterPeriod: 100, // calibrated: beats 0/50/200 across 6/12/18/24mo windows
	}
}

// New returns the adaptive strategy bound to RUAL with its calibrated defaults.
func New() *adaptive.Strategy { return adaptive.NewWithParams(Ticker, DefaultParams()) }
```

- [ ] **Step 4: Run the RUAL tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/ -v`
Expected: PASS — `TestTickerAndLookback`, `TestDefaultParams`, `TestDecide_FlatUptrendIsNone`, `TestDecide_CrushedPriceIsSL`.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/rusal/
git commit -m "refactor(scalping): rusal becomes thin adaptive config

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Add the `afks` per-share config

**Files:**
- Create: `internal/service/trading_strategy/scalping/strategy/afks/afks.go`
- Create: `internal/service/trading_strategy/scalping/strategy/afks/afks_test.go`

AFKS starts from the generic (uncalibrated) baseline. The HTF daily trend filter is OFF by default (`TrendFilterPeriod: 0`) — RUAL's calibrated 100 is not inherited; calibration decides whether AFKS wants a filter.

- [ ] **Step 1: Write the AFKS config tests**

Create `internal/service/trading_strategy/scalping/strategy/afks/afks_test.go`:

```go
package afks

import "testing"

func TestTickerAndDefaults(t *testing.T) {
	s := New()
	if s.Ticker() != "AFKS" {
		t.Errorf("Ticker = %q, want AFKS", s.Ticker())
	}
	p := DefaultParams()
	if p.ADXTrendLevel <= p.ADXRangeLevel {
		t.Errorf("ADXTrendLevel (%v) must exceed ADXRangeLevel (%v)", p.ADXTrendLevel, p.ADXRangeLevel)
	}
	if p.EMAPeriod <= 0 || p.ADXPeriod <= 0 || p.RSIPeriod <= 0 || p.DonchianPeriod <= 0 || p.ATRPeriod <= 0 {
		t.Errorf("all periods must be positive: %+v", p)
	}
	if p.TrendFilterPeriod != 0 {
		t.Errorf("TrendFilterPeriod = %d, want 0 (filter off until calibrated)", p.TrendFilterPeriod)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails to compile**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/afks/ 2>&1 | head`
Expected: build failure — `undefined: New`, `undefined: DefaultParams`.

- [ ] **Step 3: Create `afks.go`**

Create `internal/service/trading_strategy/scalping/strategy/afks/afks.go`:

```go
package afks

import (
	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
)

// Ticker is the instrument this config targets.
const Ticker = "AFKS"

// DefaultParams returns generic, NOT-yet-calibrated starting values for AFKS.
// The HTF daily trend filter is disabled (0) until calibration justifies it.
func DefaultParams() adaptive.Params {
	return adaptive.Params{
		EMAPeriod:         21,
		ADXPeriod:         14,
		ADXTrendLevel:     25,
		ADXRangeLevel:     20,
		RSIPeriod:         14,
		RSITrendLevel:     45,
		RSIRangeLevel:     35,
		PullbackWindow:    5,
		DonchianPeriod:    20,
		ATRPeriod:         14,
		SLMult:            1.0,
		TrailMult:         2.5,
		ChandelierWindow:  20,
		EMATouchTol:       0.002,
		BandTol:           0.003,
		TrendFilterPeriod: 0,
	}
}

// New returns the adaptive strategy bound to AFKS with its baseline defaults.
func New() *adaptive.Strategy { return adaptive.NewWithParams(Ticker, DefaultParams()) }
```

- [ ] **Step 4: Run the AFKS tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/afks/ -v`
Expected: PASS — `TestTickerAndDefaults`.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/afks/
git commit -m "feat(scalping): add AFKS per-share adaptive config

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Register both tickers in the backtest registry

**Files:**
- Modify: `internal/service/backtest/registry.go`
- Modify: `internal/service/backtest/registry_test.go`
- Modify: `internal/service/backtest/calibrate_test.go`

The binding now builds via `adaptive.NewWithParams(ticker, p.(adaptive.Params))`. Both RUAL and AFKS are registered. Tests that asserted `rusal.Params` switch to `adaptive.Params`.

- [ ] **Step 1: Update `registry_test.go` and `calibrate_test.go` first (red)**

Overwrite `internal/service/backtest/registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

func TestLookupKnownAndUnknown(t *testing.T) {
	if _, ok := Lookup("NOPE"); ok {
		t.Fatal("expected unknown ticker to miss")
	}
	for _, tk := range []string{"RUAL", "AFKS"} {
		b, ok := Lookup(tk)
		if !ok {
			t.Fatalf("expected %s binding", tk)
		}
		if b.DefaultParams == nil || b.Build == nil || b.ParseParams == nil {
			t.Fatalf("%s binding has nil funcs", tk)
		}
	}
}

func TestBindingsBuildStrategy(t *testing.T) {
	for _, tk := range []string{"RUAL", "AFKS"} {
		b, _ := Lookup(tk)
		s := b.Build(b.DefaultParams())
		if s.Ticker() != tk {
			t.Fatalf("built strategy ticker = %q, want %s", s.Ticker(), tk)
		}
	}
}

func TestParseParamsOverridesDefaults(t *testing.T) {
	b, _ := Lookup("RUAL")
	raw := []byte(`{"EMAPeriod": 50}`)
	parsed, err := b.ParseParams(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := parsed.(adaptive.Params)
	if p.EMAPeriod != 50 {
		t.Fatalf("EMAPeriod = %d, want 50 (override)", p.EMAPeriod)
	}
	if p.ADXPeriod != rusal.DefaultParams().ADXPeriod {
		t.Fatal("non-overridden field should keep its default")
	}
}

func TestParamRows(t *testing.T) {
	rows := ParamRows(rusal.DefaultParams())
	if len(rows) == 0 {
		t.Fatal("expected param rows")
	}
	var found bool
	for _, r := range rows {
		if r.Name == "EMAPeriod" && r.Value == "21" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected EMAPeriod=21 row")
	}
}
```

In `internal/service/backtest/calibrate_test.go`, replace the import of `.../strategy/rusal` with `.../strategy/adaptive`, and replace every `rusal.` with `adaptive.` (4 sites: line ~26 `rusal.DefaultParams()`, lines ~31/38 `updated.(rusal.Params)` / `updated2.(rusal.Params)`, line ~44 `rusal.DefaultParams()`, line ~100 `rusal.DefaultParams()`). Use:

Run: `sed -i 's#strategy/rusal#strategy/adaptive#; s#rusal\.#adaptive.#g' internal/service/backtest/calibrate_test.go`

Then verify by eye that the import line reads `"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"` and no `rusal.` remains:

Run: `grep -n "rusal\|adaptive" internal/service/backtest/calibrate_test.go`
Expected: only `adaptive` references, no `rusal`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/ 2>&1 | head`
Expected: build failure in `registry.go` (still imports/builds via `rusal.NewWithParams` / `rusal.Params`, which no longer exist) and/or no `AFKS` binding.

- [ ] **Step 3: Update `registry.go`**

Overwrite `internal/service/backtest/registry.go`:

```go
package backtest

import (
	"encoding/json"
	"fmt"
	"reflect"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
	"tinvest/internal/service/trading_strategy/scalping/strategy/afks"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

// Binding adapts a concrete strategy's params to the generic engine: it builds
// the strategy from params, supplies defaults, and parses params from JSON.
type Binding struct {
	DefaultParams func() any                         // e.g. rusal.DefaultParams()
	Build         func(params any) strategy.Strategy // e.g. adaptive.NewWithParams(ticker, p)
	ParseParams   func(raw []byte) (any, error)      // JSON -> adaptive.Params
}

// bindingFor builds a Binding for a ticker whose defaults come from defaults().
// All scalping tickers share the adaptive engine; only ticker + defaults differ.
func bindingFor(ticker string, defaults func() adaptive.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return adaptive.NewWithParams(ticker, params.(adaptive.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse params: %w", err)
			}
			return p, nil
		},
	}
}

var registry = map[string]Binding{
	rusal.Ticker: bindingFor(rusal.Ticker, rusal.DefaultParams),
	afks.Ticker:  bindingFor(afks.Ticker, afks.DefaultParams),
}

// Lookup returns the binding registered for a ticker.
func Lookup(ticker string) (Binding, bool) {
	b, ok := registry[ticker]
	return b, ok
}

// ParamRows reflects a params struct into report rows (field name -> value).
func ParamRows(params any) []backtest.ParamLine {
	v := reflect.ValueOf(params)
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	rows := make([]backtest.ParamLine, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		rows = append(rows, backtest.ParamLine{
			Name:  t.Field(i).Name,
			Value: fmt.Sprintf("%v", v.Field(i).Interface()),
		})
	}
	return rows
}
```

- [ ] **Step 4: Run the backtest package tests to verify they pass**

Run: `go test ./internal/service/backtest/ -v 2>&1 | tail -30`
Expected: PASS — `TestLookupKnownAndUnknown`, `TestBindingsBuildStrategy`, `TestParseParamsOverridesDefaults`, `TestParamRows`, plus existing calibrate tests.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/registry.go internal/service/backtest/registry_test.go internal/service/backtest/calibrate_test.go
git commit -m "feat(backtest): register RUAL+AFKS via shared adaptive binding

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Add AFKS to the live runner

**Files:**
- Modify: `internal/service/trading_strategy/scalping/registry.go`

The live runner (`trade.go`) is already multi-ticker; it just needs AFKS in the strategy slice. Scalping only emits Telegram notifications (no order placement), so this is signal-only.

- [ ] **Step 1: Update `defaultStrategies()`**

Overwrite `internal/service/trading_strategy/scalping/registry.go`:

```go
package scalping

import (
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/service/trading_strategy/scalping/strategy/afks"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

// defaultStrategies is the fixed set of per-share strategies the runner evaluates.
func defaultStrategies() []strategy.Strategy {
	return []strategy.Strategy{
		rusal.New(),
		afks.New(),
	}
}
```

- [ ] **Step 2: Build to verify it compiles**

Run: `go build ./internal/service/trading_strategy/scalping/...`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/scalping/registry.go
git commit -m "feat(scalping): add AFKS to the live runner strategy set

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Full-repo verification

**Files:** none (verification only).

- [ ] **Step 1: Build everything**

Run: `go build ./...`
Expected: no output (success).

- [ ] **Step 2: Run the whole test suite**

Run: `go test ./... 2>&1 | tail -40`
Expected: all packages `ok` or `no test files`; no `FAIL`.

- [ ] **Step 3: Confirm no stale `rusal.Params` / `rusal.NewWithParams` references remain**

Run: `grep -rn "rusal\.Params\|rusal\.NewWithParams" --include=*.go || echo "clean"`
Expected: `clean`.

- [ ] **Step 4: Commit (only if Steps 1-3 produced fixes; otherwise skip)**

```bash
git add -A
git commit -m "chore(scalping): post-refactor verification fixes

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Create AFKS parameter files

**Files:**
- Create: `data/params/afks/scalp.json`
- Create: `data/params/afks/grid.json`

Mirrors the RUAL param-file shape so the same backtest commands work. `scalp.json` is a full single set (baseline); `grid.json` sweeps the same 5 fields RUAL sweeps.

- [ ] **Step 1: Create `data/params/afks/scalp.json`**

```json
{
  "EMAPeriod": 21,
  "ADXPeriod": 14,
  "ADXTrendLevel": 25,
  "ADXRangeLevel": 20,
  "RSIPeriod": 14,
  "RSITrendLevel": 45,
  "RSIRangeLevel": 35,
  "PullbackWindow": 5,
  "DonchianPeriod": 20,
  "ATRPeriod": 14,
  "SLMult": 1.0,
  "TrailMult": 2.5,
  "ChandelierWindow": 20,
  "EMATouchTol": 0.002,
  "BandTol": 0.003,
  "TrendFilterPeriod": 0
}
```

- [ ] **Step 2: Create `data/params/afks/grid.json`**

```json
{
  "ADXTrendLevel": [22, 25, 30],
  "ADXRangeLevel": [18, 20, 22],
  "RSIRangeLevel": [30, 35, 40],
  "SLMult": [1.0, 1.5, 2.0],
  "TrailMult": [2.5, 3.0, 3.5]
}
```

- [ ] **Step 3: Commit**

```bash
git add data/params/afks/
git commit -m "feat(backtest): AFKS starter params and calibration grid

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Fetch AFKS candles and run the RUAL-vs-AFKS comparison

**Files:**
- Created as side effects: `data/candles/AFKS_Hour1.json`, `data/candles/AFKS_Day1.json`
- Created as side effects: `reports/*` (backtest reports)

This requires the Tinkoff token `T_BANK` (in `env/local.env`). The first AFKS backtest run fetches and caches candles via `svc.NewCandleProvider`. **If the executing agent lacks the token / network**, ask the user to run these commands themselves with the `!` prefix; the code/tests above are already complete and committed.

- [ ] **Step 1: Baseline RUAL run (DefaultParams)**

Run: `go run ./cmd/backtest -ticker RUAL -months 24 -cash 100000 -out reports/cmp`
Expected: `report: reports/cmp/RUAL_Hour1_<stamp>.md (trades=N, net=..., PF=...)`. Note the printed trades / net / PF.

- [ ] **Step 2: Baseline AFKS run (same DefaultParams), fetches candles**

Run: `go run ./cmd/backtest -ticker AFKS -months 24 -cash 100000 -out reports/cmp`
Expected: candles fetched + cached (`data/candles/AFKS_Hour1.json`, `AFKS_Day1.json` appear), then `report: reports/cmp/AFKS_Hour1_<stamp>.md (...)`.

- [ ] **Step 2a: Confirm candle cache was written**

Run: `ls -la data/candles/AFKS_*.json`
Expected: both `AFKS_Hour1.json` and `AFKS_Day1.json` exist and are non-empty.

- [ ] **Step 3: AFKS grid calibration**

Run: `go run ./cmd/backtest -ticker AFKS -months 24 -cash 100000 -calibrate data/params/afks/grid.json -out reports/cmp`
Expected: `calibration: reports/cmp/AFKS_Hour1_<stamp>_calibration.md (combos=243)`. A `_best.md` is also written.

- [ ] **Step 4: Read the three reports and extract the metrics table**

Run: `grep -A22 "Сводка метрик" reports/cmp/RUAL_Hour1_*.md reports/cmp/AFKS_Hour1_*.md`
Read the `## Сводка метрик` blocks of: RUAL baseline, AFKS baseline, AFKS `_best`. Build a comparison table: Trades / Win rate / PF / Net% / CAGR / Exposure / Max DD for each.

- [ ] **Step 5: Write the comparison summary**

Create `reports/cmp/SUMMARY.md` with:
- the RUAL-vs-AFKS-vs-AFKS-best metrics table,
- the verdict: does AFKS on identical `DefaultParams` produce materially better metrics (esp. exposure and CAGR) than RUAL? This answers "strategy vs share". State it explicitly.

Then commit the artifacts:

```bash
git add data/candles/AFKS_Hour1.json data/candles/AFKS_Day1.json reports/cmp/
git commit -m "chore(backtest): AFKS candles + RUAL-vs-AFKS comparison reports

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

- [ ] **Step 6: Report findings to the user**

Summarize the comparison table and the verdict in chat. If AFKS metrics are also weak (low exposure, sub-risk-free CAGR), recommend the deferred follow-up (risk-adjusted metrics + exposure-aware calibration objective, and/or entry/exit restructuring) as the next iteration. If AFKS is materially better, recommend proceeding to calibrate AFKS battle params.

---

## Notes for the executor

- **TDD on a refactor:** Tasks 1–2 move already-tested code. The "fail" gate is a compile error (symbol moved/renamed), not an assertion failure — that's expected and sufficient.
- **Intentional intermediate breakage:** From the end of Task 2 until Task 4 Step 3, a repo-wide `go build ./...` is RED — `internal/service/backtest/registry.go` still references the removed `rusal.Params` / `rusal.NewWithParams`. This is expected. Tasks 2 and 3 only run their own package's tests (which compile in isolation); the live runner (`scalping/registry.go`) keeps compiling because it only calls `rusal.New()`. The repo goes green again at Task 4 Step 4, confirmed in Task 6. Do not "fix" backtest early — Task 4 rewrites it wholesale.
- **No behaviour change to RUAL:** if any RUAL test in Task 2/6 fails on an assertion (not a compile error), STOP — the move was not verbatim. Diff against the original `rusal.go` logic before changing test expectations.
- **`gofmt`:** the param structs use aligned fields; run `gofmt -w` on any file you hand-edit before committing.
- **Token:** Task 8 needs `T_BANK`. Everything in Tasks 1–7 is offline and must be green before Task 8.
```