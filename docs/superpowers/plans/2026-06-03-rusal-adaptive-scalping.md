# RUSAL Adaptive Scalping — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the single-regime RUSAL strategy with an ADX-gated adaptive one — mean-reversion in a range, momentum in a trend — keeping the existing `strategy.Strategy` contract and runner untouched.

**Architecture:** Two new pure indicator helpers (`ADX`, `Donchian`) join the existing `ATR`/`RSISeries`/`ema.Compute`. The `rusal` package is rewritten: `Decide` computes all indicators from the candle window and delegates to a pure, table-tested `decide` core that picks a regime from ADX and applies regime-specific entry/exit rules. All tunables live in an external `Params` struct so they can be calibrated later without touching logic.

**Tech Stack:** Go 1.25, standard `testing` (table-driven), no new dependencies.

**Spec:** `docs/superpowers/specs/2026-06-03-rusal-adaptive-scalping-design.md`

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `pkg/indicators/adx.go` | Pure Wilder ADX + DI+/DI− | Create |
| `pkg/indicators/adx_test.go` | ADX fixtures + guard cases | Create |
| `pkg/indicators/donchian.go` | Pure Donchian channel (upper/lower) | Create |
| `pkg/indicators/donchian_test.go` | Donchian fixtures + guard cases | Create |
| `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go` | Params, Strategy, Decide, pure `decide` core, helpers | Rewrite |
| `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go` | helper unit tests + `decide` table tests + e2e `Decide` | Rewrite |

The `strategy.Strategy` contract, `MarketData`, `Position`, `Signal`, `registry.go`, `trade.go` and the service wiring are **unchanged** — `rusal.New()` keeps its zero-arg signature, so the registry still compiles as-is.

---

## Task 1: ADX/DMI pure helper

**Files:**
- Create: `pkg/indicators/adx.go`
- Test: `pkg/indicators/adx_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/indicators/adx_test.go`:

```go
package indicators

import (
	"math"
	"testing"
)

func TestADX(t *testing.T) {
	tests := []struct {
		name           string
		highs          []float64
		lows           []float64
		closes         []float64
		period         int
		wantADX        float64 // checked with tol
		wantADXTol     float64
		assertDirected func(t *testing.T, diPlus, diMinus float64)
	}{
		{
			// Strict uptrend: every -DM is 0, so DX == 100 on every bar => ADX == 100,
			// -DI == 0, +DI > 0. Robust to the exact Wilder seed convention.
			name:       "strict uptrend -> adx 100, diMinus 0",
			highs:      []float64{10, 11, 12, 13, 14, 15},
			lows:       []float64{9, 10, 11, 12, 13, 14},
			closes:     []float64{9.5, 10.5, 11.5, 12.5, 13.5, 14.5},
			period:     2,
			wantADX:    100,
			wantADXTol: 1e-9,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diMinus != 0 {
					t.Errorf("diMinus = %v, want 0", diMinus)
				}
				if diPlus <= 0 {
					t.Errorf("diPlus = %v, want > 0", diPlus)
				}
			},
		},
		{
			// Strict downtrend: mirror image. +DM is 0 => DX == 100 => ADX == 100,
			// +DI == 0, -DI > 0.
			name:       "strict downtrend -> adx 100, diPlus 0",
			highs:      []float64{15, 14, 13, 12, 11, 10},
			lows:       []float64{14, 13, 12, 11, 10, 9},
			closes:     []float64{14.5, 13.5, 12.5, 11.5, 10.5, 9.5},
			period:     2,
			wantADX:    100,
			wantADXTol: 1e-9,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 {
					t.Errorf("diPlus = %v, want 0", diPlus)
				}
				if diMinus <= 0 {
					t.Errorf("diMinus = %v, want > 0", diMinus)
				}
			},
		},
		{
			// Flat bars: no directional movement, +DI == -DI == 0 => DX guarded to 0 => ADX 0.
			name:       "flat -> adx 0",
			highs:      []float64{12, 12, 12, 12, 12, 12},
			lows:       []float64{10, 10, 10, 10, 10, 10},
			closes:     []float64{11, 11, 11, 11, 11, 11},
			period:     2,
			wantADX:    0,
			wantADXTol: 1e-9,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
		{
			name:       "period <= 0 is silent zero",
			highs:      []float64{10, 11, 12, 13, 14},
			lows:       []float64{9, 10, 11, 12, 13},
			closes:     []float64{9.5, 10.5, 11.5, 12.5, 13.5},
			period:     0,
			wantADX:    0,
			wantADXTol: 0,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
		{
			name:       "insufficient history (need 2*period+1) is silent zero",
			highs:      []float64{10, 11, 12, 13}, // n=4 < 2*2+1=5
			lows:       []float64{9, 10, 11, 12},
			closes:     []float64{9.5, 10.5, 11.5, 12.5},
			period:     2,
			wantADX:    0,
			wantADXTol: 0,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
		{
			name:       "length mismatch is silent zero",
			highs:      []float64{10, 11, 12, 13, 14},
			lows:       []float64{9, 10, 11, 12}, // shorter
			closes:     []float64{9.5, 10.5, 11.5, 12.5, 13.5},
			period:     2,
			wantADX:    0,
			wantADXTol: 0,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adx, diPlus, diMinus := ADX(tc.highs, tc.lows, tc.closes, tc.period)
			if math.Abs(adx-tc.wantADX) > tc.wantADXTol {
				t.Fatalf("ADX = %v, want %v (tol %v)", adx, tc.wantADX, tc.wantADXTol)
			}
			tc.assertDirected(t, diPlus, diMinus)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/indicators/ -run TestADX -v`
Expected: FAIL — `undefined: ADX`.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/indicators/adx.go`:

```go
package indicators

import "math"

// ADX returns Wilder's Average Directional Index together with the directional
// indicators +DI and -DI over the input bar series.
//
// Inputs must be aligned (highs[i], lows[i], closes[i] all describe the same bar).
// Returns (0, 0, 0) when period <= 0, when the slices are not the same length, or
// when len(closes) < 2*period+1 — the insufficient-history rule is silent (no error),
// mirroring ATR. The +2*period+1 minimum covers the double smoothing: DI/DX seed over
// the first `period` increments, then ADX seeds over the next `period` DX values.
//
// Returned values are the indicators at the last bar.
func ADX(highs, lows, closes []float64, period int) (adx, diPlus, diMinus float64) {
	if period <= 0 {
		return 0, 0, 0
	}
	n := len(closes)
	if len(highs) != n || len(lows) != n {
		return 0, 0, 0
	}
	if n < 2*period+1 {
		return 0, 0, 0
	}

	plusDM := make([]float64, n)
	minusDM := make([]float64, n)
	tr := make([]float64, n)
	for i := 1; i < n; i++ {
		up := highs[i] - highs[i-1]
		down := lows[i-1] - lows[i]
		if up > down && up > 0 {
			plusDM[i] = up
		}
		if down > up && down > 0 {
			minusDM[i] = down
		}
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(hl, math.Max(hc, lc))
	}

	// Wilder seeds over the first `period` increments (indices 1..period).
	var sTR, sPlus, sMinus float64
	for i := 1; i <= period; i++ {
		sTR += tr[i]
		sPlus += plusDM[i]
		sMinus += minusDM[i]
	}

	p := float64(period)
	dx := func() float64 {
		var pdi, mdi float64
		if sTR != 0 {
			pdi = 100 * sPlus / sTR
			mdi = 100 * sMinus / sTR
		}
		denom := pdi + mdi
		if denom == 0 {
			return 0
		}
		return 100 * math.Abs(pdi-mdi) / denom
	}

	// Seed ADX as the average of the first `period` DX values (indices period..2*period-1).
	dxSum := dx()
	for i := period + 1; i <= 2*period-1; i++ {
		sTR += tr[i] - sTR/p
		sPlus += plusDM[i] - sPlus/p
		sMinus += minusDM[i] - sMinus/p
		dxSum += dx()
	}
	adx = dxSum / p

	// Wilder-smooth ADX (and the underlying DI) to the last bar.
	for i := 2 * period; i < n; i++ {
		sTR += tr[i] - sTR/p
		sPlus += plusDM[i] - sPlus/p
		sMinus += minusDM[i] - sMinus/p
		adx = (adx*(p-1) + dx()) / p
	}

	if sTR != 0 {
		diPlus = 100 * sPlus / sTR
		diMinus = 100 * sMinus / sTR
	}
	return adx, diPlus, diMinus
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/indicators/ -run TestADX -v`
Expected: PASS (all sub-tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/indicators/adx.go pkg/indicators/adx_test.go
git commit -m "feat(indicators): add pure Wilder ADX/DMI helper

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Donchian channel pure helper

**Files:**
- Create: `pkg/indicators/donchian.go`
- Test: `pkg/indicators/donchian_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/indicators/donchian_test.go`:

```go
package indicators

import "testing"

func TestDonchian(t *testing.T) {
	tests := []struct {
		name       string
		highs      []float64
		lows       []float64
		period     int
		wantUpper  float64
		wantLower  float64
	}{
		{
			name:      "last 3 bars window",
			highs:     []float64{5, 7, 3, 9, 4},
			lows:      []float64{2, 1, 0, 4, 3},
			period:    3, // last 3 highs {3,9,4} -> 9 ; last 3 lows {0,4,3} -> 0
			wantUpper: 9,
			wantLower: 0,
		},
		{
			name:      "period equals length spans all bars",
			highs:     []float64{5, 7, 3},
			lows:      []float64{2, 1, 4},
			period:    3,
			wantUpper: 7,
			wantLower: 1,
		},
		{
			name:      "insufficient history is silent zero",
			highs:     []float64{5, 7},
			lows:      []float64{2, 1},
			period:    3,
			wantUpper: 0,
			wantLower: 0,
		},
		{
			name:      "length mismatch is silent zero",
			highs:     []float64{5, 7, 3},
			lows:      []float64{2, 1},
			period:    2,
			wantUpper: 0,
			wantLower: 0,
		},
		{
			name:      "period <= 0 is silent zero",
			highs:     []float64{5, 7, 3},
			lows:      []float64{2, 1, 4},
			period:    0,
			wantUpper: 0,
			wantLower: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			upper, lower := Donchian(tc.highs, tc.lows, tc.period)
			if upper != tc.wantUpper || lower != tc.wantLower {
				t.Fatalf("Donchian = (%v, %v), want (%v, %v)", upper, lower, tc.wantUpper, tc.wantLower)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/indicators/ -run TestDonchian -v`
Expected: FAIL — `undefined: Donchian`.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/indicators/donchian.go`:

```go
package indicators

// Donchian returns the highest high and lowest low over the last `period` bars.
//
// Returns (0, 0) when period <= 0, when highs and lows differ in length, or when
// fewer than `period` bars are available — the insufficient-history rule is silent
// (no error), mirroring ATR. The channel midpoint is the caller's (upper+lower)/2.
func Donchian(highs, lows []float64, period int) (upper, lower float64) {
	if period <= 0 {
		return 0, 0
	}
	n := len(highs)
	if len(lows) != n || n < period {
		return 0, 0
	}
	upper = highs[n-period]
	lower = lows[n-period]
	for i := n - period + 1; i < n; i++ {
		if highs[i] > upper {
			upper = highs[i]
		}
		if lows[i] < lower {
			lower = lows[i]
		}
	}
	return upper, lower
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/indicators/ -run TestDonchian -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/indicators/donchian.go pkg/indicators/donchian_test.go
git commit -m "feat(indicators): add pure Donchian channel helper

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: RUSAL skeleton — Params, wiring, helpers (green commit, `decide` stubbed)

This task replaces both `rusal.go` and `rusal_test.go`. The new `decide` is a **stub returning `SignalNone`** so the package compiles and a first slice of tests (Params, Lookback, Ticker, helper functions, flat-uptrend e2e) passes. Task 4 then adds the decision tests and the real `decide` body.

**Files:**
- Modify (rewrite): `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`
- Modify (rewrite): `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go`

- [ ] **Step 1: Rewrite the strategy file with stubbed `decide`**

Overwrite `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`:

```go
package rusal

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

const ticker = "RUAL"

// Params holds every tunable for the RUSAL adaptive strategy. They are exposed so
// they can be calibrated on real history without touching the decision logic.
type Params struct {
	EMAPeriod        int     // fast EMA for the trend pullback
	ADXPeriod        int     // ADX/DMI period
	ADXTrendLevel    float64 // ADX >= -> trend regime
	ADXRangeLevel    float64 // ADX <= -> range regime (between the two = dead zone)
	RSIPeriod        int     // RSI period
	RSITrendLevel    float64 // RSI reversal threshold in trend (shallow pullbacks)
	RSIRangeLevel    float64 // RSI reversal threshold in range (oversold)
	PullbackWindow   int     // bars back over which an EMA "touch" still counts
	DonchianPeriod   int     // channel period: lower for entry, mid for range exit
	ATRPeriod        int     // ATR period for stops/trailing
	SLMult           float64 // initial stop = entry - SLMult*ATR
	TrailMult        float64 // chandelier = max(High over window) - TrailMult*ATR
	ChandelierWindow int     // window for the chandelier high
	EMATouchTol      float64 // EMA touch tolerance (fraction, e.g. 0.002 = 0.2%)
	BandTol          float64 // lower-band proximity tolerance (fraction)
}

// DefaultParams returns standard, NOT-yet-calibrated starting values.
func DefaultParams() Params {
	return Params{
		EMAPeriod:        21,
		ADXPeriod:        14,
		ADXTrendLevel:    25,
		ADXRangeLevel:    20,
		RSIPeriod:        14,
		RSITrendLevel:    45,
		RSIRangeLevel:    35,
		PullbackWindow:   5,
		DonchianPeriod:   20,
		ATRPeriod:        14,
		SLMult:           1.0,
		TrailMult:        2.5,
		ChandelierWindow: 20,
		EMATouchTol:      0.002,
		BandTol:          0.003,
	}
}

// Strategy trades RUSAL adaptively: it picks a regime from ADX and applies
// mean-reversion in a range or momentum in a trend.
type Strategy struct{ p Params }

// New returns the RUSAL strategy with default (uncalibrated) params.
func New() *Strategy { return &Strategy{p: DefaultParams()} }

// NewWithParams returns the RUSAL strategy with explicit params (for calibration/tests).
func NewWithParams(p Params) *Strategy { return &Strategy{p: p} }

func (s *Strategy) Ticker() string { return ticker }

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
	}

	sig := s.decide(in)
	sig.Ticker = ticker
	return sig
}

// decide is the pure decision core. STUB: filled in Task 4.
func (s *Strategy) decide(in decideInput) model.Signal {
	return model.Signal{Price: in.price, RSI: in.rsiNow}
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
func recentHigh(highs []float64, window int) float64 {
	n := len(highs)
	if n == 0 {
		return 0
	}
	start := n - window
	if start < 0 {
		start = 0
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

- [ ] **Step 2: Rewrite the test file with the green-against-stub slice**

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
}

func TestEMATouched(t *testing.T) {
	ema := []float64{10, 10, 10, 10, 10}
	// A low at index 3 dips to the EMA within tolerance.
	lows := []float64{12, 12, 12, 10.01, 12}
	if !emaTouched(lows, ema, 3, 0.002) { // window covers indices 2,3,4
		t.Error("expected touch within last 3 bars")
	}
	if emaTouched(lows, ema, 1, 0.002) { // window covers only index 4 (low 12, no touch)
		t.Error("did not expect touch within last 1 bar")
	}
	if emaTouched(nil, nil, 3, 0.002) {
		t.Error("empty input must not touch")
	}
}

func TestRecentHigh(t *testing.T) {
	highs := []float64{5, 9, 3, 7, 4}
	if got := recentHigh(highs, 3); got != 7 { // last 3 -> {3,7,4} -> 7
		t.Errorf("recentHigh = %v, want 7", got)
	}
	if got := recentHigh(highs, 10); got != 9 { // window > len -> all -> 9
		t.Errorf("recentHigh = %v, want 9", got)
	}
	if got := recentHigh(nil, 3); got != 0 {
		t.Errorf("recentHigh(nil) = %v, want 0", got)
	}
}

// TestDecide_FlatUptrendIsNone: a monotonic uptrend keeps RSI high (no upward cross)
// and price runs above the EMA, so no entry fires. Holds for the stub and the real core.
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
	md := strategy.MarketData{
		Price:  closes[199],
		Highs:  highs,
		Lows:   lows,
		Closes: closes,
	}
	got := s.Decide(md)
	if got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", got.Kind)
	}
	if got.Ticker != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", got.Ticker)
	}
}
```

- [ ] **Step 3: Run tests to verify the slice passes**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/ -v`
Expected: PASS — `TestTickerAndLookback`, `TestDefaultParams`, `TestEMATouched`, `TestRecentHigh`, `TestDecide_FlatUptrendIsNone`.

- [ ] **Step 4: Confirm the package and dependents still build**

Run: `go build ./...`
Expected: success (registry/service call `rusal.New()` with no args — unchanged).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/rusal/rusal.go internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go
git commit -m "refactor(rusal): adaptive skeleton — Params, regime wiring, helpers

decide core stubbed to SignalNone; real decision logic lands next.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: RUSAL pure `decide` core — regime, entry, exit

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go` (add `decide` table tests + e2e SL)
- Modify: `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go` (replace the stub `decide`)

- [ ] **Step 1: Add the failing decision tests**

Append to `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go`:

```go
// testParams uses clean levels/multipliers so expected stops/targets are exact.
// Indicator periods are irrelevant here: decide() consumes pre-computed scalars.
func testParams() Params {
	return Params{
		EMAPeriod: 3, ADXPeriod: 2, ADXTrendLevel: 25, ADXRangeLevel: 20,
		RSIPeriod: 2, RSITrendLevel: 45, RSIRangeLevel: 35,
		PullbackWindow: 5, DonchianPeriod: 3, ATRPeriod: 2,
		SLMult: 1.0, TrailMult: 2.0, ChandelierWindow: 3,
		EMATouchTol: 0.002, BandTol: 0.003,
	}
}

func TestDecideCore(t *testing.T) {
	s := NewWithParams(testParams())

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
			wantKind: model.SignalBuy, wantTP: 104, wantSL: 98, // 100+2*2 ; 100-1*2
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
			name: "range entry: at lower band + rsi cross",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8, // 100 <= 99.8*1.003 = 100.0994
			},
			wantKind: model.SignalBuy, wantTP: 104.9, wantSL: 98, // mid=(110+99.8)/2 ; 100-2
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
				price: 106, atr: 2, adx: 15, donUpper: 110, donLower: 100, // mid=105
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "TP", wantTP: 105, wantSL: 98,
		},
		{
			name: "range exit: stop loss",
			in: decideInput{
				price: 97, atr: 2, adx: 15, donUpper: 110, donLower: 100, // mid=105
				pos: &strategy.Position{PurchasePrice: 100},
			},
			wantKind: model.SignalSell, wantReason: "SL", wantTP: 105, wantSL: 98,
		},
		{
			name: "trend exit: chandelier trail",
			in: decideInput{
				price: 105, atr: 2, adx: 30, chandelierHigh: 110, // chandelier=110-2*2=106
				pos: &strategy.Position{PurchasePrice: 100}, // hardSL=98, price 105 > 98
			},
			wantKind: model.SignalSell, wantReason: "TRAIL", wantSL: 106,
		},
		{
			name: "trend exit: initial stop wins over trail",
			in: decideInput{
				price: 97.5, atr: 2, adx: 30, chandelierHigh: 101, // chandelier=97, hardSL=98
				pos: &strategy.Position{PurchasePrice: 100}, // 97.5 <= 98 -> SL first
			},
			wantKind: model.SignalSell, wantReason: "SL", wantSL: 98,
		},
		{
			name: "trend hold while rising -> none",
			in: decideInput{
				price: 118, atr: 2, adx: 30, chandelierHigh: 120, // chandelier=116, hardSL=98
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

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/ -run 'TestDecideCore|TestDecide_CrushedPriceIsSL' -v`
Expected: FAIL — the stub returns `SignalNone`, so every buy/sell case mismatches.

- [ ] **Step 3: Replace the stub `decide` with the real core**

In `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`, replace the stub:

```go
// decide is the pure decision core. STUB: filled in Task 4.
func (s *Strategy) decide(in decideInput) model.Signal {
	return model.Signal{Price: in.price, RSI: in.rsiNow}
}
```

with:

```go
// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(in decideInput) model.Signal {
	sig := model.Signal{Price: in.price, RSI: in.rsiNow}
	reg := s.regimeOf(in.adx)

	// Manage an open position (long-only): exits are regime-dependent.
	if in.pos != nil {
		hardSL := in.pos.PurchasePrice - s.p.SLMult*in.atr
		if reg == regimeTrend {
			chandelier := in.chandelierHigh - s.p.TrailMult*in.atr
			switch {
			case in.price <= hardSL:
				sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", hardSL
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
		case in.price >= mid:
			sig.Kind, sig.Reason = model.SignalSell, "TP"
		}
		return sig
	}

	// Flat -> regime-specific entries (long only).
	switch reg {
	case regimeTrend:
		crossedUp := in.rsiPrev < s.p.RSITrendLevel && in.rsiNow >= s.p.RSITrendLevel
		if in.diPlus > in.diMinus && in.emaTouched && crossedUp && in.price > in.emaNow {
			sig.Kind = model.SignalBuy
			sig.StopLoss = in.price - s.p.SLMult*in.atr
			sig.TakeProfit = in.price + s.p.TrailMult*in.atr
		}
	case regimeRange:
		crossedUp := in.rsiPrev < s.p.RSIRangeLevel && in.rsiNow >= s.p.RSIRangeLevel
		if in.price <= in.donLower*(1+s.p.BandTol) && crossedUp {
			sig.Kind = model.SignalBuy
			sig.StopLoss = in.price - s.p.SLMult*in.atr
			sig.TakeProfit = (in.donUpper + in.donLower) / 2
		}
	}
	return sig
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/ -v`
Expected: PASS (all tests, including the Task 3 slice).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/rusal/rusal.go internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go
git commit -m "feat(rusal): ADX-gated adaptive decide core

Range: lower Donchian + RSI reversal entry, mean-reversion exit at mid.
Trend: EMA pullback + RSI reversal entry, ATR chandelier trailing exit.
Hard ATR stop in both regimes.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Full verification

**Files:** none (verification only).

- [ ] **Step 1: Run the whole test suite**

Run: `go test ./...`
Expected: PASS (exit 0).

- [ ] **Step 2: Vet the touched packages**

Run: `go vet ./pkg/indicators/... ./internal/service/trading_strategy/scalping/...`
Expected: no output (clean).

- [ ] **Step 3: Build everything**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Confirm no dead references to the old single-regime API**

Run: `git grep -n "aboveEMA\|rsiReversalLevel\|tpMult" -- internal/service/trading_strategy/scalping/strategy/rusal/`
Expected: no matches (old fields/params fully removed).

- [ ] **Step 5: Finish the branch**

Announce and use **superpowers:finishing-a-development-branch** to verify tests, present options, and execute the choice.

---

## Self-Review Notes

- **Spec coverage:** ADX helper (Task 1), Donchian helper (Task 2), Params externalisation + regime wiring + stateless helpers (Task 3), regime/entry/exit decision logic with all spec branches (Task 4), verification (Task 5). The "honesty note" / calibration follow-up is documentation-only and needs no task.
- **Type consistency:** `decideInput`, `Params`, `regimeOf`, `emaTouched`, `recentHigh`, `decide` signatures are identical across Tasks 3 and 4. `rusal.New()` stays zero-arg, so `registry.go`/service wiring is untouched.
- **Stateless exit:** chandelier uses `recentHigh(Highs, ChandelierWindow)`, not since-entry memory — matches the spec.
