# Reversion (RSI buy-the-dip) Strategy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a long-only hourly mean-reversion strategy `reversion` that buys sharp RSI drawdowns inside a confirmed uptrend and exits on the bounce, reusing the existing backtest reporting layer unchanged.

**Architecture:** A pure, ticker-agnostic decision core (`reversion/strategy/core`) modeled exactly on the existing momentum core: a pure `decide(decideInput)` over pre-computed indicators, wrapped by an impure `Strategy` shell that computes indicators and carries one mutable `barsInPosition` counter for the time-stop. Per-ticker packages supply `Ticker` + `DefaultParams`. A `Binding` registry (`reversion_registry.go`) and a `case "reversion"` branch in `cmd/backtest/main.go` wire it into the existing engine, metrics, and Markdown/CSV renderers.

**Tech Stack:** Go 1.25, existing `pkg/indicators` (RSI, ATR, volume) plus a new `Stochastic`, `internal/domain/ema`, `internal/domain/backtest` engine, `internal/service/backtest` Binding.

Spec: `docs/superpowers/specs/2026-06-12-reversion-rsi-dip-design.md`

---

### Task 1: Stochastic indicator

**Files:**
- Create: `pkg/indicators/stochastic.go`
- Test: `pkg/indicators/stochastic_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/indicators/stochastic_test.go`:

```go
package indicators

import (
	"math"
	"testing"
)

func TestStochastic_RampFixture(t *testing.T) {
	// Windows of 3 over this series:
	//   [10,12,11] -> hi12 lo10 close11 -> %K = 100*(11-10)/2 = 50
	//   [12,11,13] -> hi13 lo11 close13 -> %K = 100
	//   [11,13,14] -> hi14 lo11 close14 -> %K = 100
	// %D (smooth 3) = (50+100+100)/3 = 83.33
	highs := []float64{10, 12, 11, 13, 14}
	lows := []float64{10, 12, 11, 13, 14}
	closes := []float64{10, 12, 11, 13, 14}

	k, d := Stochastic(highs, lows, closes, 3, 3)
	if math.Abs(k-100) > 0.01 {
		t.Errorf("%%K = %v, want 100", k)
	}
	if math.Abs(d-83.33) > 0.01 {
		t.Errorf("%%D = %v, want 83.33", d)
	}
}

func TestStochastic_TooFewBarsReturnsZero(t *testing.T) {
	k, d := Stochastic([]float64{1, 2}, []float64{1, 2}, []float64{1, 2}, 3, 3)
	if k != 0 || d != 0 {
		t.Errorf("got k=%v d=%v, want 0,0", k, d)
	}
}

func TestStochastic_FlatRangeYieldsZeroK(t *testing.T) {
	// A collapsed high/low range must not divide by zero.
	k, _ := Stochastic([]float64{5, 5, 5}, []float64{5, 5, 5}, []float64{5, 5, 5}, 3, 1)
	if k != 0 {
		t.Errorf("%%K = %v, want 0 on zero range", k)
	}
}

func TestStochastic_FewerThanSmoothReturnsZeroD(t *testing.T) {
	// Exactly one %K value but dSmooth=3 -> %D not computable yet.
	_, d := Stochastic([]float64{10, 12, 11}, []float64{10, 12, 11}, []float64{10, 12, 11}, 3, 3)
	if d != 0 {
		t.Errorf("%%D = %v, want 0 (insufficient %%K history)", d)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/indicators/ -run TestStochastic -v`
Expected: FAIL to compile — `undefined: Stochastic`.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/indicators/stochastic.go`:

```go
package indicators

// Stochastic returns the latest %K and %D of the stochastic oscillator.
// %K = 100 * (close - lowestLow) / (highestHigh - lowestLow) over the last kPeriod
// bars; %D = simple moving average of %K over the last dSmooth %K values. Both are 0
// when history is insufficient (fewer than kPeriod bars, or fewer than dSmooth %K
// values for %D) or when the high/low range collapses to zero.
func Stochastic(highs, lows, closes []float64, kPeriod, dSmooth int) (k, d float64) {
	n := len(closes)
	if kPeriod <= 0 || dSmooth <= 0 || n < kPeriod || len(highs) < n || len(lows) < n {
		return 0, 0
	}

	ks := make([]float64, 0, n-kPeriod+1)
	for end := kPeriod; end <= n; end++ {
		hi, lo := highs[end-kPeriod], lows[end-kPeriod]
		for i := end - kPeriod + 1; i < end; i++ {
			if highs[i] > hi {
				hi = highs[i]
			}
			if lows[i] < lo {
				lo = lows[i]
			}
		}
		rng := hi - lo
		if rng == 0 {
			ks = append(ks, 0)
			continue
		}
		ks = append(ks, 100*(closes[end-1]-lo)/rng)
	}

	k = ks[len(ks)-1]
	if len(ks) < dSmooth {
		return k, 0
	}
	var sum float64
	for i := len(ks) - dSmooth; i < len(ks); i++ {
		sum += ks[i]
	}
	return k, sum / float64(dSmooth)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/indicators/ -run TestStochastic -v`
Expected: PASS (all four subtests).

- [ ] **Step 5: Commit**

```bash
git add pkg/indicators/stochastic.go pkg/indicators/stochastic_test.go
git commit -m "feat(indicators): add Stochastic oscillator (%K/%D)"
```

---

### Task 2: Reversion core — Params, shell, pure decide, entry & exit logic

**Files:**
- Create: `internal/service/trading_strategy/reversion/strategy/core/core.go`
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the core implementation**

Create `internal/service/trading_strategy/reversion/strategy/core/core.go`:

```go
// Package core implements a long-only hourly mean-reversion strategy. It buys sharp
// RSI drawdowns inside a confirmed uptrend (fast EMA above slow EMA, and price above
// the slow EMA) and exits on the bounce (RSI crossing up through an overbought level),
// a hard percentage stop frozen at entry, or a time-stop when the bounce never comes.
// Volume must be above its recent average; an optional Stochastic oversold gate can be
// switched on. The decision logic is pure and ticker-agnostic; per-share packages
// supply ticker + Params.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// EntryMode values for Params.EntryMode.
const (
	entryConfirmed = 0 // RSI crosses up through RSIOversold (bounce confirmed)
	entryKnife     = 1 // RSI crosses down through RSIOversold (catch the falling knife)
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	FastEMA       int     // fast regime EMA (e.g. 50)
	SlowEMA       int     // slow regime EMA + price floor (e.g. 200)
	RSIPeriod     int     // RSI length (e.g. 6); required (>0)
	RSIOversold   float64 // dip-trigger level (e.g. 40)
	RSIOverbought float64 // exit level (e.g. 70)
	EntryMode     int     // 0 = confirmed up-cross, 1 = knife down-cross
	VolLookback   int     // SMA window for the volume baseline
	VolMultiplier float64 // last volume must exceed VolMultiplier*SMA(volume)
	UseStoch      int     // 1 = require Stochastic oversold confirmation; 0 = skip
	StochPeriod   int     // %K lookback
	StochSmooth   int     // %D smoothing of %K
	StochOversold float64 // %K oversold threshold (e.g. 20)
	StopLossPct   float64 // hard stop = entry*(1-StopLossPct); must be > 0
	MaxHoldBars   int     // time-stop bar count; <= 0 disables
	ATRPeriod     int     // ATR length — display only, never gates logic
}

// Strategy trades a single instrument with the mean-reversion rules. Ticker-agnostic.
// It carries barsInPosition as mutable state in the impure shell; the pure decide()
// core stays a function of its input. Not safe for concurrent use; the backtest and
// live runners drive Decide sequentially, one bar at a time.
type Strategy struct {
	ticker         string
	p              Params
	barsInPosition int // bars elapsed since entry; reset to 0 while flat
}

// NewWithParams returns the reversion strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to feed the hungriest consumer.
func (s *Strategy) Lookback() int {
	m := s.p.SlowEMA
	for _, c := range []int{
		s.p.FastEMA,
		s.p.RSIPeriod + 1,
		s.p.VolLookback + 1,
		s.p.ATRPeriod + 1,
		s.p.StochPeriod + s.p.StochSmooth,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core.
type decideInput struct {
	price     float64
	atr       float64 // display only
	emaFast   float64
	emaSlow   float64
	rsiNow    float64
	rsiPrev   float64
	stochK    float64 // computed only when UseStoch == 1
	volumeOK  bool
	barLow    float64
	pos       *strategy.Position
	barsInPos int
}

// Decide computes every indicator from md, advances the position-age counter, and
// delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position != nil {
		s.barsInPosition++
	} else {
		s.barsInPosition = 0
	}
	in := s.buildInput(md)
	in.barsInPos = s.barsInPosition
	sig := s.decide(in)
	sig.Ticker = s.ticker
	return sig
}

// buildInput computes every indicator from md and packs them for the pure core.
func (s *Strategy) buildInput(md strategy.MarketData) decideInput {
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)

	emaFast, emaSlow := 0.0, 0.0
	if e := ema.Compute(md.Closes, s.p.FastEMA); len(e) > 0 {
		emaFast = e[len(e)-1]
	}
	if e := ema.Compute(md.Closes, s.p.SlowEMA); len(e) > 0 {
		emaSlow = e[len(e)-1]
	}

	var rsiNow, rsiPrev float64
	if s.p.RSIPeriod > 0 {
		if r := indicators.RSISeries(md.Closes, s.p.RSIPeriod); len(r) >= 2 {
			rsiNow, rsiPrev = r[len(r)-1], r[len(r)-2]
		}
	}

	stochK := 0.0
	if s.p.UseStoch == 1 {
		stochK, _ = indicators.Stochastic(md.Highs, md.Lows, md.Closes, s.p.StochPeriod, s.p.StochSmooth)
	}

	var barLow float64
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	return decideInput{
		price:    md.Price,
		atr:      atr,
		emaFast:  emaFast,
		emaSlow:  emaSlow,
		rsiNow:   rsiNow,
		rsiPrev:  rsiPrev,
		stochK:   stochK,
		volumeOK: indicators.VolumeConfirmed(md.Volumes, s.p.VolLookback, s.p.VolMultiplier),
		barLow:   barLow,
		pos:      md.Position,
	}
}

// dipFired reports whether the RSI dip trigger fires on this bar, honouring EntryMode.
// confirmed: RSI crosses up through RSIOversold (the up-cross implies a prior dip).
// knife: RSI crosses down through RSIOversold.
func (s *Strategy) dipFired(in decideInput) bool {
	if s.p.RSIPeriod <= 0 {
		return false
	}
	if s.p.EntryMode == entryKnife {
		return in.rsiPrev >= s.p.RSIOversold && in.rsiNow < s.p.RSIOversold
	}
	return in.rsiPrev <= s.p.RSIOversold && in.rsiNow > s.p.RSIOversold
}

// uptrend reports the regime gate: fast EMA above slow EMA and price above the slow EMA.
func uptrend(in decideInput) bool {
	return in.emaFast > in.emaSlow && in.emaSlow > 0 && in.price > in.emaSlow
}

// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(in decideInput) model.Signal {
	sig := model.Signal{Price: in.price}

	if in.pos != nil {
		return s.manage(in, sig)
	}

	// 1. Regime: only buy dips inside a confirmed uptrend.
	if !uptrend(in) {
		return sig
	}
	// 2. Dip trigger.
	if !s.dipFired(in) {
		return sig
	}
	// 3. Volume.
	if !in.volumeOK {
		return sig
	}
	// 4. Optional Stochastic oversold confirmation.
	if s.p.UseStoch == 1 && !(in.stochK < s.p.StochOversold) {
		return sig
	}
	// 5. Protective-stop sanity: a hard stop is mandatory.
	if s.p.StopLossPct <= 0 {
		return sig
	}
	stop := in.price * (1 - s.p.StopLossPct)
	risk := in.price - stop
	if risk <= 0 {
		return sig
	}

	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.ATR = in.atr
	sig.RSI = in.rsiNow
	sig.EntryReason = s.entryReason(in, stop, risk)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(in decideInput, stop, risk float64) string {
	mode := "кросс вверх через"
	if s.p.EntryMode == entryKnife {
		mode = "кросс вниз через"
	}
	stoch := "выкл"
	if s.p.UseStoch == 1 {
		stoch = fmt.Sprintf("%%K %.1f < %.0f", in.stochK, s.p.StochOversold)
	}
	return fmt.Sprintf(
		"Тренд↑ (EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d); RSI(%d) %s %.0f (%.2f→%.2f); объём > %.2g×ср(%d); стохастик: %s; SL=%.4f (−%.2g%%, −%.4f)",
		s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA,
		s.p.RSIPeriod, mode, s.p.RSIOversold, in.rsiPrev, in.rsiNow,
		s.p.VolMultiplier, s.p.VolLookback,
		stoch,
		stop, s.p.StopLossPct*100, risk,
	)
}

// manage handles an open long: frozen hard stop, a time-stop, or an RSI-overbought
// exit. Protective stops are checked first so the worst case for the position wins
// ties on a bar.
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	hardSL := in.pos.StopLoss
	sig.StopLoss = hardSL
	sig.RSI = in.rsiNow

	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (зафиксирован на входе)", in.barLow, hardSL)
	case s.p.MaxHoldBars > 0 && in.barsInPos >= s.p.MaxHoldBars:
		sig.Kind, sig.Reason = model.SignalSell, "TIME"
		sig.ExitReason = fmt.Sprintf("TIME: %d бар(ов) в позиции ≥ %d — отскок не пришёл", in.barsInPos, s.p.MaxHoldBars)
	case s.p.RSIPeriod > 0 && in.rsiPrev < s.p.RSIOverbought && in.rsiNow >= s.p.RSIOverbought:
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: %.2f → %.2f, пересёк %.2g вверх (отскок завершён)", in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
	}
	return sig
}
```

- [ ] **Step 2: Write the tests**

Create `internal/service/trading_strategy/reversion/strategy/core/core_test.go`:

```go
package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// defaultParams returns valid, entry-capable params for tests.
func defaultParams() Params {
	return Params{
		FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 6, RSIOversold: 40, RSIOverbought: 70,
		EntryMode:   entryConfirmed,
		VolLookback: 20, VolMultiplier: 1.2,
		UseStoch: 0, StochPeriod: 14, StochSmooth: 3, StochOversold: 20,
		StopLossPct: 0.03, MaxHoldBars: 24, ATRPeriod: 14,
	}
}

// passingInput returns a flat decideInput that clears every entry gate: uptrend,
// confirmed RSI up-cross through 40, volume OK, stochastic well oversold.
func passingInput() decideInput {
	return decideInput{
		price:    100,
		atr:      1,
		emaFast:  95,
		emaSlow:  90,
		rsiPrev:  38, // below 40
		rsiNow:   45, // above 40 -> confirmed up-cross fires
		stochK:   10,
		volumeOK: true,
		barLow:   100,
	}
}

// openInput returns an input with an open position above its stop (no exit triggers).
func openInput() decideInput {
	in := passingInput()
	in.pos = &strategy.Position{PurchasePrice: 100, StopLoss: 97}
	in.rsiPrev, in.rsiNow = 45, 50 // not crossing overbought
	return in
}

func TestEntryAllGatesPass(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	sig := s.decide(passingInput())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got kind=%v", sig.Kind)
	}
	if math.Abs(sig.StopLoss-97) > 1e-9 { // 100*(1-0.03)
		t.Fatalf("StopLoss=%v want 97", sig.StopLoss)
	}
	if !strings.Contains(sig.EntryReason, "RSI(6)") {
		t.Fatalf("EntryReason missing RSI detail: %q", sig.EntryReason)
	}
}

func TestRegimeBlocks(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	in := passingInput()
	in.emaFast = 85 // fast below slow -> not an uptrend
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("fast<slow: want no Buy")
	}

	in = passingInput()
	in.price = 89 // below slow EMA (90)
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("price<slowEMA: want no Buy")
	}
}

func TestDipConfirmedVsKnife(t *testing.T) {
	// confirmed: up-cross fires, down-cross does not.
	s := NewWithParams("TEST", defaultParams())
	if !s.dipFired(passingInput()) {
		t.Fatalf("confirmed: up-cross should fire")
	}
	down := passingInput()
	down.rsiPrev, down.rsiNow = 45, 38 // crossing down
	if s.dipFired(down) {
		t.Fatalf("confirmed: down-cross should NOT fire")
	}

	// knife: down-cross fires, up-cross does not.
	p := defaultParams()
	p.EntryMode = entryKnife
	sk := NewWithParams("TEST", p)
	if !sk.dipFired(down) {
		t.Fatalf("knife: down-cross should fire")
	}
	if sk.dipFired(passingInput()) {
		t.Fatalf("knife: up-cross should NOT fire")
	}
}

func TestVolumeBlocks(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.volumeOK = false
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("volume low: want no Buy")
	}
}

func TestStochGate(t *testing.T) {
	p := defaultParams()
	p.UseStoch = 1
	s := NewWithParams("TEST", p)

	in := passingInput()
	in.stochK = 50 // above oversold 20 -> blocked
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("stoch high: want no Buy")
	}

	in = passingInput()
	in.stochK = 10 // below 20 -> passes
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("stoch oversold: want Buy, got %v", sig.Kind)
	}
}

func TestStopSanityBlocksWhenNoStop(t *testing.T) {
	p := defaultParams()
	p.StopLossPct = 0
	s := NewWithParams("TEST", p)
	if sig := s.decide(passingInput()); sig.Kind == model.SignalBuy {
		t.Fatalf("StopLossPct=0: want no Buy (safety mandatory)")
	}
}

func TestExitSL(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barLow = 96 // <= 97
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want SL sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestExitTime(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barsInPos = 24 // >= MaxHoldBars
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "TIME" {
		t.Fatalf("want TIME sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestExitRSI(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.rsiPrev, in.rsiNow = 65, 72 // crosses up through 70
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("want RSI sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestProtectiveStopWinsTie(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barLow = 96            // SL hit
	in.rsiPrev, in.rsiNow = 65, 72 // RSI overbought too
	sig := s.decide(in)
	if sig.Reason != "SL" {
		t.Fatalf("protective first: want SL, got %q", sig.Reason)
	}
}

func TestDecideTimeStopCounter(t *testing.T) {
	p := defaultParams()
	p.MaxHoldBars = 2
	s := NewWithParams("TEST", p)
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 90}
	md := strategy.MarketData{
		Price:    100,
		Highs:    []float64{100, 100, 100},
		Lows:     []float64{99, 99, 99}, // above SL 90, no SL exit
		Closes:   []float64{100, 100, 100},
		Volumes:  []int64{1, 1, 1},
		Position: pos,
	}
	if sig := s.Decide(md); sig.Reason == "TIME" { // barsInPosition=1
		t.Fatalf("bar1: unexpected TIME exit")
	}
	sig := s.Decide(md) // barsInPosition=2 >= 2
	if sig.Kind != model.SignalSell || sig.Reason != "TIME" {
		t.Fatalf("bar2: want TIME sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/... -v`
Expected: PASS (all entry/exit/counter tests).

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/
git commit -m "feat(reversion): pure mean-reversion core (entry gates + exits)"
```

---

### Task 3: Reversion Explain (optional diagnostic interface)

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (append method)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go` (append test)

- [ ] **Step 1: Append the Explain method**

Append to `core.go` (after `manage`):

```go
// Explain re-runs the entry gates over md and reports each gate's value and verdict
// (✓ pass / ✗ block) in entry order, stopping at the first blocker — the same
// short-circuit order decide uses. Diagnostic only; never part of the trading path;
// never mutates barsInPosition.
func (s *Strategy) Explain(md strategy.MarketData) string {
	in := s.buildInput(md)
	in.barsInPos = s.barsInPosition

	if in.pos != nil {
		return "позиция уже открыта — вход не рассматривается"
	}

	var b strings.Builder
	pass := func(format string, args ...any) { fmt.Fprintf(&b, "✓ "+format+"\n", args...) }
	block := func(format string, args ...any) string {
		fmt.Fprintf(&b, "✗ "+format+"\n", args...)
		fmt.Fprintf(&b, "→ ВХОДА НЕТ: заблокировал этот фильтр")
		return b.String()
	}

	// 1. Regime.
	if !uptrend(in) {
		return block("Тренд: нужно EMA%d > EMA%d и close > EMA%d (EMA%d=%.4f, EMA%d=%.4f, close=%.4f)",
			s.p.FastEMA, s.p.SlowEMA, s.p.SlowEMA, s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price)
	}
	pass("Тренд↑: EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d", s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)

	// 2. Dip trigger.
	mode := "кросс вверх через"
	if s.p.EntryMode == entryKnife {
		mode = "кросс вниз через"
	}
	if !s.dipFired(in) {
		return block("RSI(%d): нет события (%s %.0f), %.2f→%.2f", s.p.RSIPeriod, mode, s.p.RSIOversold, in.rsiPrev, in.rsiNow)
	}
	pass("RSI(%d): сработал триггер просадки (%s %.0f, %.2f→%.2f)", s.p.RSIPeriod, mode, s.p.RSIOversold, in.rsiPrev, in.rsiNow)

	// 3. Volume.
	if !in.volumeOK {
		return block("Объём: ниже %.2g×ср(%d)", s.p.VolMultiplier, s.p.VolLookback)
	}
	pass("Объём: выше %.2g×ср(%d)", s.p.VolMultiplier, s.p.VolLookback)

	// 4. Optional Stochastic.
	if s.p.UseStoch == 1 {
		if !(in.stochK < s.p.StochOversold) {
			return block("Стохастик: %%K %.1f ≥ %.0f (не в перепроданности)", in.stochK, s.p.StochOversold)
		}
		pass("Стохастик: %%K %.1f < %.0f", in.stochK, s.p.StochOversold)
	}

	// 5. Protective stop.
	if s.p.StopLossPct <= 0 {
		return block("Стоп: StopLossPct=%.2g ≤ 0 — защита не задана", s.p.StopLossPct)
	}
	stop := in.price * (1 - s.p.StopLossPct)
	pass("Стоп: SL=%.4f (−%.2g%%)", stop, s.p.StopLossPct*100)

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}
```

- [ ] **Step 2: Append the Explain test**

Append to `core_test.go`:

```go
func TestExplainBlocksOnRegime(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	md := strategy.MarketData{
		Price:   1,
		Highs:   []float64{1},
		Lows:    []float64{1},
		Closes:  []float64{1}, // no EMA history -> emaSlow 0 -> not uptrend
		Volumes: []int64{1},
	}
	out := s.Explain(md)
	if !strings.Contains(out, "Тренд") || !strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain should block on regime: %q", out)
	}
}

func TestExplainPositionOpen(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	md := strategy.MarketData{
		Price: 100, Highs: []float64{100}, Lows: []float64{100},
		Closes: []float64{100}, Volumes: []int64{1},
		Position: &strategy.Position{PurchasePrice: 100, StopLoss: 97},
	}
	if out := s.Explain(md); !strings.Contains(out, "позиция уже открыта") {
		t.Fatalf("Explain with open position: %q", out)
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/service/trading_strategy/reversion/... -run TestExplain -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/
git commit -m "feat(reversion): Explain diagnostic for entry gates"
```

---

### Task 4: Per-ticker packages (8) + verify build

**Files:**
- Create: `internal/service/trading_strategy/reversion/strategy/{afks,gazp,mdmg,nvtk,plzl,rusal,sber,ydex}/<ticker>.go`

- [ ] **Step 1: Create all eight per-ticker packages**

Each file has the same shape; only the package name, `Ticker` constant, and doc string differ. The starting params mirror the generic baseline and are to be calibrated later (same convention as momentum).

`internal/service/trading_strategy/reversion/strategy/afks/afks.go`:

```go
// Package afks supplies the ticker and starting reversion Params for AFKS (AFK Sistema).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here.
package afks

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "AFKS"

// DefaultParams returns AFKS's starting reversion parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{
		FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 6, RSIOversold: 40, RSIOverbought: 70,
		EntryMode:   0,
		VolLookback: 20, VolMultiplier: 1.2,
		UseStoch: 0, StochPeriod: 14, StochSmooth: 3, StochOversold: 20,
		StopLossPct: 0.03, MaxHoldBars: 24, ATRPeriod: 14,
	}
}
```

Repeat for the other seven, changing only `package`, the doc comment ticker name, and `Ticker`:

| Dir | `package` | `Ticker` |
|---|---|---|
| `gazp` | `gazp` | `"GAZP"` |
| `mdmg` | `mdmg` | `"MDMG"` |
| `nvtk` | `nvtk` | `"NVTK"` |
| `plzl` | `plzl` | `"PLZL"` |
| `rusal` | `rusal` | `"RUAL"` |
| `sber` | `sber` | `"SBER"` |
| `ydex` | `ydex` | `"YDEX"` |

Note: the `rusal` package's `Ticker` is `"RUAL"` (the exchange ticker), matching the momentum convention. Each file's `DefaultParams()` body is identical to the AFKS one above.

- [ ] **Step 2: Verify the packages build**

Run: `go build ./internal/service/trading_strategy/reversion/...`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/
git commit -m "feat(reversion): per-ticker starting params for the 8-ticker basket"
```

---

### Task 5: Reversion registry + Binding wiring

**Files:**
- Create: `internal/service/backtest/reversion_registry.go`
- Test: `internal/service/backtest/reversion_registry_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/service/backtest/reversion_registry_test.go`:

```go
package backtest

import (
	"testing"

	reversionafks "tinvest/internal/service/trading_strategy/reversion/strategy/afks"
	"tinvest/internal/service/trading_strategy/reversion/strategy/core"
	reversionrusal "tinvest/internal/service/trading_strategy/reversion/strategy/rusal"
)

func TestReversionLookupRegisteredRUAL(t *testing.T) {
	b := ReversionLookupOrGeneric("RUAL")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != reversionrusal.DefaultParams() {
		t.Fatalf("RUAL defaults = %+v\nwant %+v", got, reversionrusal.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "RUAL" {
		t.Fatalf("ticker=%q want RUAL", s.Ticker())
	}
}

func TestReversionLookupRegisteredAFKS(t *testing.T) {
	b := ReversionLookupOrGeneric("AFKS")
	got := b.DefaultParams().(core.Params)
	if got != reversionafks.DefaultParams() {
		t.Fatalf("AFKS defaults mismatch")
	}
}

func TestReversionLookupGenericFallback(t *testing.T) {
	b := ReversionLookupOrGeneric("UNKNOWN")
	if s := b.Build(b.DefaultParams()); s.Ticker() != "UNKNOWN" {
		t.Fatalf("ticker=%q want UNKNOWN", s.Ticker())
	}
	// ParseParams must layer the override on top of genericReversionDefaults.
	got, err := b.ParseParams([]byte(`{"StopLossPct": 0.05}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.StopLossPct != 0.05 {
		t.Fatalf("StopLossPct=%v want 0.05 (override)", p.StopLossPct)
	}
	if p.FastEMA != 50 || p.SlowEMA != 200 {
		t.Fatalf("generic defaults not preserved: FastEMA=%d SlowEMA=%d want 50/200", p.FastEMA, p.SlowEMA)
	}
}

func TestReversionDefaultsValid(t *testing.T) {
	if p := genericReversionDefaults(); p.StopLossPct <= 0 || p.SlowEMA <= p.FastEMA {
		t.Fatalf("invalid generic defaults: %+v", p)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestReversion -v`
Expected: FAIL to compile — `undefined: ReversionLookupOrGeneric`, `undefined: genericReversionDefaults`.

- [ ] **Step 3: Write the registry**

Create `internal/service/backtest/reversion_registry.go`:

```go
package backtest

import (
	"encoding/json"
	"fmt"

	reversionafks "tinvest/internal/service/trading_strategy/reversion/strategy/afks"
	"tinvest/internal/service/trading_strategy/reversion/strategy/core"
	reversiongazp "tinvest/internal/service/trading_strategy/reversion/strategy/gazp"
	reversionmdmg "tinvest/internal/service/trading_strategy/reversion/strategy/mdmg"
	reversionnvtk "tinvest/internal/service/trading_strategy/reversion/strategy/nvtk"
	reversionplzl "tinvest/internal/service/trading_strategy/reversion/strategy/plzl"
	reversionrusal "tinvest/internal/service/trading_strategy/reversion/strategy/rusal"
	reversionsber "tinvest/internal/service/trading_strategy/reversion/strategy/sber"
	reversionydex "tinvest/internal/service/trading_strategy/reversion/strategy/ydex"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// reversionBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All reversion tickers share the core engine; only ticker + defaults differ.
func reversionBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
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

var reversionRegistry = map[string]Binding{
	reversionrusal.Ticker: reversionBindingFor(reversionrusal.Ticker, reversionrusal.DefaultParams),
	reversionafks.Ticker:  reversionBindingFor(reversionafks.Ticker, reversionafks.DefaultParams),
	reversionydex.Ticker:  reversionBindingFor(reversionydex.Ticker, reversionydex.DefaultParams),
	reversionplzl.Ticker:  reversionBindingFor(reversionplzl.Ticker, reversionplzl.DefaultParams),
	reversionsber.Ticker:  reversionBindingFor(reversionsber.Ticker, reversionsber.DefaultParams),
	reversiongazp.Ticker:  reversionBindingFor(reversiongazp.Ticker, reversiongazp.DefaultParams),
	reversionnvtk.Ticker:  reversionBindingFor(reversionnvtk.Ticker, reversionnvtk.DefaultParams),
	reversionmdmg.Ticker:  reversionBindingFor(reversionmdmg.Ticker, reversionmdmg.DefaultParams),
}

// genericReversionDefaults are neutral baseline params for tickers without a dedicated
// reversion config. Intentionally independent of any per-ticker defaults so calibrating
// one ticker never drifts the generic baseline.
func genericReversionDefaults() core.Params {
	return core.Params{
		FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 6, RSIOversold: 40, RSIOverbought: 70,
		EntryMode:   0,
		VolLookback: 20, VolMultiplier: 1.2,
		UseStoch: 0, StochPeriod: 14, StochSmooth: 3, StochOversold: 20,
		StopLossPct: 0.03, MaxHoldBars: 24, ATRPeriod: 14,
	}
}

// ReversionLookupOrGeneric returns the registered reversion binding for a ticker, or a
// generic binding bound to that ticker (with genericReversionDefaults) when none is
// registered.
func ReversionLookupOrGeneric(ticker string) Binding {
	if b, ok := reversionRegistry[ticker]; ok {
		return b
	}
	return reversionBindingFor(ticker, genericReversionDefaults)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/backtest/ -run TestReversion -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/reversion_registry.go internal/service/backtest/reversion_registry_test.go
git commit -m "feat(reversion): backtest Binding registry + generic fallback"
```

---

### Task 6: Wire `reversion` into the backtest command

**Files:**
- Modify: `cmd/backtest/main.go` (strategy flag usage, strategy switch, unknown-strategy error)

- [ ] **Step 1: Update the `-strategy` flag usage string**

In `cmd/backtest/main.go`, find:

```go
		strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|levels|momentum")
```

Replace with:

```go
		strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|levels|momentum|reversion")
```

- [ ] **Step 2: Add the strategy switch branch**

Find the switch in `run`:

```go
	var binding svc.Binding
	switch strategyName {
	case "levels":
		binding = svc.LevelsLookupOrGeneric(ticker)
	case "momentum":
		binding = svc.MomentumLookupOrGeneric(ticker)
	case "scalping":
		binding = svc.LookupOrGeneric(ticker)
	default:
		return fmt.Errorf("unknown strategy %q (want scalping|levels|momentum)", strategyName)
	}
```

Replace with:

```go
	var binding svc.Binding
	switch strategyName {
	case "levels":
		binding = svc.LevelsLookupOrGeneric(ticker)
	case "momentum":
		binding = svc.MomentumLookupOrGeneric(ticker)
	case "reversion":
		binding = svc.ReversionLookupOrGeneric(ticker)
	case "scalping":
		binding = svc.LookupOrGeneric(ticker)
	default:
		return fmt.Errorf("unknown strategy %q (want scalping|levels|momentum|reversion)", strategyName)
	}
```

- [ ] **Step 3: Build and run the full test suite**

Run: `go build ./... && go test ./...`
Expected: build succeeds; all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/backtest/main.go
git commit -m "feat(reversion): wire reversion into the backtest command"
```

---

### Task 7: Strategy explainer doc

**Files:**
- Create: `docs/reversion/strategy.md`

- [ ] **Step 1: Write the explainer**

Create `docs/reversion/strategy.md`:

```markdown
# Reversion strategy (RSI buy-the-dip)

Long-only, hourly. Buys sharp RSI drawdowns inside a confirmed uptrend and sells the
bounce. The conceptual mirror of the momentum strategy.

## Entry (all gates must pass, short-circuit order)

1. **Regime:** `EMA(FastEMA) > EMA(SlowEMA)` and `close > EMA(SlowEMA)` (defaults 50/200).
2. **Dip trigger** (`EntryMode`):
   - `0` confirmed (default): RSI(`RSIPeriod`, default 6) crosses **up** through
     `RSIOversold` (40) — the bounce has started.
   - `1` knife: RSI crosses **down** through `RSIOversold`.
3. **Volume:** last volume > `VolMultiplier × SMA(VolLookback)`.
4. **Stochastic (optional, `UseStoch=1`):** %K < `StochOversold` (20).
5. **Protective stop:** `StopLossPct > 0` required; stop = `entry × (1 − StopLossPct)`.

## Exit (first trigger wins; protective first)

1. **SL:** bar low ≤ frozen `entry × (1 − StopLossPct)`.
2. **TIME:** `MaxHoldBars` bars elapsed without a bounce.
3. **RSI:** RSI crosses **up** through `RSIOverbought` (70).

## Run

```bash
# single run
go run ./cmd/backtest -ticker SBER -strategy reversion -months 12 -out ./reports/SBER

# grid calibration with walk-forward OOS
go run ./cmd/backtest -ticker SBER -strategy reversion \
  -calibrate data/params/sber/reversion_grid.json -out ./reports/SBER \
  -months 24 -test-months 6 -min-trades 20 -metric profit_factor

# diagnose one bar
go run ./cmd/backtest -ticker SBER -strategy reversion \
  -explain '2026-03-14 12:00' -months 12
```

## Params

`FastEMA, SlowEMA, RSIPeriod, RSIOversold, RSIOverbought, EntryMode, VolLookback,
VolMultiplier, UseStoch, StochPeriod, StochSmooth, StochOversold, StopLossPct,
MaxHoldBars, ATRPeriod` (ATR is display-only). All are int/float64 so the grid
calibrator can sweep them.

## Not yet supported

`-basket` walk-forward (the basket runner is currently momentum-only).
```

- [ ] **Step 2: Commit**

```bash
git add docs/reversion/strategy.md
git commit -m "docs(reversion): strategy explainer"
```

---

### Task 8: Final verification

- [ ] **Step 1: Full build + vet + test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all succeed, all tests PASS.

- [ ] **Step 2: Optional live smoke (requires T_BANK token + network — user runs this)**

Run: `go run ./cmd/backtest -ticker SBER -strategy reversion -months 12 -out ./reports/SBER`
Expected: prints `report: reports/SBER/SBER_reversion_Hour1_<stamp>.md (trades=…, net=…, PF=…)` and writes the Markdown + CSV reports. Inspect the `.md` to confirm trades show entry/exit reasons.

This step depends on a valid `T_BANK` token and network access; it is not a unit-test gate. Skip in CI; run locally to confirm the end-to-end report path.

---

## Notes for the implementer

- The `model` and `strategy` packages are the scalping ones
  (`tinvest/internal/service/trading_strategy/scalping/{model,strategy}`), reused by
  momentum and levels too — not a typo.
- `Explain` is **not** part of the `strategy.Strategy` interface; the engine's `Trace`
  type-asserts for it. The strategy still satisfies `strategy.Strategy` via
  `Ticker`/`Lookback`/`Decide`.
- Keep `decide` pure: it must read only from `decideInput`. The only mutable state is
  `Strategy.barsInPosition`, updated in `Decide` (and read, never written, by `Explain`).
- Frequent commits: one per task as shown.
```
