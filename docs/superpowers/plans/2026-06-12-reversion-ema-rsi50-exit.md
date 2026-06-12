# Reversion EMA-cross / RSI-50 exit, no stop-loss — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the reversion strategy's exit (hard ATR stop + dual-overbought XOVER) with two momentum-fade exits — RSI crossing 50 downward, or a bearish FastEMA/SlowEMA cross — and remove the protective stop entirely.

**Architecture:** The pure decision core (`core.go`) drops the ATR stop and overbought logic from both entry and exit. An open long is now managed by two OR-ed exit signals filled at close. The four now-dead Params fields (`ATRPeriod`, `ATRMult`, `RSIOverbought`, `StochOverbought`) are removed from the struct, all 8 per-ticker defaults, the generic registry default, and the 8 calibration grids.

**Tech Stack:** Go 1.25, existing `pkg/indicators` (RSISeries, StochasticSeries) and `internal/domain/ema` (Compute). Table-style Go tests with `go test`.

---

## File Structure

- `internal/service/trading_strategy/reversion/strategy/core/core.go` — pure core; the heart of the change (Params, buildInput, decide, manage, entryReason, Explain).
- `internal/service/trading_strategy/reversion/strategy/core/core_test.go` — core behavior tests; rewritten for the new exit and trimmed Params.
- `internal/service/trading_strategy/reversion/strategy/{afks,gazp,mdmg,nvtk,plzl,rusal,sber,ydex}/*.go` — 8 per-ticker `DefaultParams()`; drop 4 fields each.
- `internal/service/backtest/reversion_registry.go` — `genericReversionDefaults()`; drop 4 fields.
- `internal/service/backtest/reversion_registry_test.go` — `TestReversionDefaultsValid` references `p.ATRMult`; fix.
- `data/params/{afks,gazp,mdmg,nvtk,plzl,rual,sber,ydex}/reversion_grid.json` — 8 grids; drop removed-field sweeps and the now-empty exit phase.
- `docs/reversion/strategy.md` — strategy explainer; rewrite Exit section and Params list.

**Task 1 is one atomic Go change** (struct-field removal is atomic across the package boundary, so core + per-ticker + registry must compile together in a single commit). Tasks 2–4 are independent (JSON, docs, smoke run).

---

### Task 1: Rewrite the core and its tests

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (full rewrite below)
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core_test.go` (full rewrite below)
- Modify: `internal/service/trading_strategy/reversion/strategy/afks/afks.go` (and gazp, mdmg, nvtk, plzl, rusal, sber, ydex — identical edit)
- Modify: `internal/service/backtest/reversion_registry.go:51-58`
- Modify: `internal/service/backtest/reversion_registry_test.go:52-57`

- [ ] **Step 1: Rewrite `core_test.go` to express the new behavior**

Replace the entire file `internal/service/trading_strategy/reversion/strategy/core/core_test.go` with:

```go
package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// defaultParams returns valid, entry-capable params: trend on, RSI/Stoch oversold 20.
func defaultParams() Params {
	return Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
	}
}

// passingInput clears every entry gate: uptrend; RSI crosses DOWN into oversold while
// Stoch %D is already in the oversold zone. EMA prev == now so no spurious cross.
func passingInput() decideInput {
	return decideInput{
		price: 100,
		emaFast: 95, emaFastPrev: 95, emaSlow: 90, emaSlowPrev: 90,
		rsiPrev: 25, rsiNow: 15, rsiOK: true, // crossDown through 20 (RSI enters oversold)
		stochPrev: 10, stochNow: 8, stochOK: true, // already < 20 (Stoch already in zone)
	}
}

// openInput is an open position holding, with neutral signals (no exit): RSI above 50
// and rising, fast EMA above slow on both bars.
func openInput() decideInput {
	in := passingInput()
	in.pos = &strategy.Position{PurchasePrice: 100}
	in.rsiPrev, in.rsiNow = 60, 62
	in.stochPrev, in.stochNow = 50, 55
	in.emaFast, in.emaFastPrev = 95, 95
	in.emaSlow, in.emaSlowPrev = 90, 90
	return in
}

func TestEntryBlockedWhenOscillatorInvalid(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// Stoch warm-up: no valid %D reading. The sentinel 0 < oversold(20) would otherwise
	// masquerade as "already deep in zone" and degrade the dual gate to RSI-only.
	in := passingInput()
	in.stochOK = false
	in.stochNow, in.stochPrev = 0, 0
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("invalid stoch reading: want no Buy (dual confirmation must require a real reading)")
	}

	// Symmetric: RSI warm-up also blocks.
	in = passingInput()
	in.rsiOK = false
	in.rsiNow, in.rsiPrev = 0, 0
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("invalid rsi reading: want no Buy")
	}
}

func TestEntryAllGatesPass(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	sig := s.decide(passingInput())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got kind=%v", sig.Kind)
	}
	if !strings.Contains(sig.EntryReason, "RSI(14)") || !strings.Contains(sig.EntryReason, "Stoch") {
		t.Fatalf("EntryReason missing dual detail: %q", sig.EntryReason)
	}
}

func TestEntryRequiresBothIndicators(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// RSI crosses in but Stoch NOT in zone -> no buy.
	in := passingInput()
	in.stochPrev, in.stochNow = 50, 55
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("RSI cross but Stoch out of zone: want no Buy")
	}

	// Stoch crosses in while RSI already in zone -> buy (the mirror branch).
	in = passingInput()
	in.rsiPrev, in.rsiNow = 15, 12     // already < 20, no fresh cross
	in.stochPrev, in.stochNow = 25, 15 // crossDown through 20
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("Stoch cross + RSI already in: want Buy, got %v", sig.Kind)
	}
}

func TestSimultaneousEntry(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.rsiPrev, in.rsiNow = 25, 15     // RSI crosses in
	in.stochPrev, in.stochNow = 25, 15 // Stoch crosses in same bar
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("both cross into zone same bar: want Buy, got %v", sig.Kind)
	}
}

func TestTrendFilterToggles(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.emaFast = 85 // fast < slow
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, fast<slow: want no Buy")
	}
	in = passingInput()
	in.price = 89 // below slow EMA (90)
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, price<slowEMA: want no Buy")
	}

	p := defaultParams()
	p.UseTrend = 0
	s0 := NewWithParams("TEST", p)
	in = passingInput()
	in.emaFast = 85
	if sig := s0.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("UseTrend=0: want Buy regardless of trend, got %v", sig.Kind)
	}
}

func TestExitRSI50(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// RSI crosses 50 downward -> sell RSI50.
	in := openInput()
	in.rsiPrev, in.rsiNow = 55, 45
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "RSI50" {
		t.Fatalf("RSI down-cross 50: want RSI50 sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}

	// RSI stays above 50 -> no sell.
	in = openInput()
	in.rsiPrev, in.rsiNow = 55, 52
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("RSI above 50: should NOT sell")
	}
}

func TestExitEMACross(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// Fast EMA drops below slow EMA, RSI neutral (no 50 cross) -> sell EMAX.
	in := openInput()
	in.rsiPrev, in.rsiNow = 60, 58 // no 50 cross
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // fast now below slow
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "EMAX" {
		t.Fatalf("bearish EMA cross: want EMAX sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}

	// Fast stays above slow -> no EMA exit.
	in = openInput()
	in.rsiPrev, in.rsiNow = 60, 58
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 94, 90
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("fast above slow: should NOT sell")
	}
}

func TestExitPrecedenceRSIWhenBoth(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.rsiPrev, in.rsiNow = 55, 45     // RSI50 fires
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // EMAX also fires
	if sig := s.decide(in); sig.Reason != "RSI50" {
		t.Fatalf("both fire: want RSI50 precedence, got %q", sig.Reason)
	}
}

func TestNoExitWhenHolding(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	if sig := s.decide(openInput()); sig.Kind == model.SignalSell {
		t.Fatalf("neutral signals: should hold, got sell %q", sig.Reason)
	}
}

func TestExplainBlocksOnTrend(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	md := strategy.MarketData{
		Price: 1, Highs: []float64{1}, Lows: []float64{1},
		Closes: []float64{1}, Volumes: []int64{1}, // no EMA history -> emaSlow 0 -> not uptrend
	}
	out := s.Explain(md)
	if !strings.Contains(out, "Тренд") || !strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain should block on trend: %q", out)
	}
}

func TestExplainTrendOffSkipsGate(t *testing.T) {
	p := defaultParams()
	p.UseTrend = 0
	s := NewWithParams("TEST", p)
	md := strategy.MarketData{
		Price: 1, Highs: []float64{1}, Lows: []float64{1},
		Closes: []float64{1}, Volumes: []int64{1},
	}
	if out := s.Explain(md); strings.Contains(out, "Тренд:") {
		t.Fatalf("UseTrend=0: Explain should not show trend gate: %q", out)
	}
}

func TestExplainPositionOpen(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	md := strategy.MarketData{
		Price: 100, Highs: []float64{100}, Lows: []float64{100},
		Closes: []float64{100}, Volumes: []int64{1},
		Position: &strategy.Position{PurchasePrice: 100},
	}
	if out := s.Explain(md); !strings.Contains(out, "позиция уже открыта") {
		t.Fatalf("Explain with open position: %q", out)
	}
}
```

- [ ] **Step 2: Run core tests to confirm they fail to compile (RED)**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/`
Expected: build failure — `core.go` still references removed Params fields / old `decideInput` shape and `math` import in test is gone. (Compile error is the expected RED here.)

- [ ] **Step 3: Rewrite `core.go`**

Replace the entire file `internal/service/trading_strategy/reversion/strategy/core/core.go` with:

```go
// Package core implements a long-only mean-reversion strategy on the daily timeframe,
// driven by the agreement of two oscillators: RSI and the Stochastic %D line. It buys
// when one oscillator is already inside its oversold zone and the other crosses into it.
// It exits an open long on either momentum-fade signal: RSI crossing the 50 line downward
// (the primary exit), or a bearish EMA cross (FastEMA dropping below SlowEMA) as a
// regime-break backstop. There is no protective stop. An optional trend filter restricts
// buys to a confirmed uptrend. The decision logic is pure and ticker-agnostic; per-share
// packages supply ticker + Params. Run with `-interval Day1`.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// rsiExitLevel is the fixed RSI midline: an open long exits when RSI crosses it downward.
const rsiExitLevel = 50.0

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	UseTrend      int     // 1 = require uptrend before buying; 0 = ignore trend
	FastEMA       int     // fast regime EMA (e.g. 50); also the bearish-cross exit fast line
	SlowEMA       int     // slow regime EMA + price floor (e.g. 200); bearish-cross exit slow line
	RSIPeriod     int     // RSI length; required (>0)
	RSIOversold   float64 // RSI oversold zone (entry side)
	StochKPeriod  int     // Stochastic %K lookback; required (>0)
	StochDSmooth  int     // Stochastic %D smoothing; required (>0); 1 = raw %K
	StochOversold float64 // Stochastic oversold zone (entry side)
}

// Strategy trades a single instrument with the dual-confirmation rules. Ticker-agnostic
// and pure: decide() is a function of its input. Not safe for concurrent use.
type Strategy struct {
	ticker string
	p      Params
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
		s.p.StochKPeriod + s.p.StochDSmooth + 1,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core. stochNow/Prev
// are the %D line (smoothed %K). emaFastPrev/emaSlowPrev are the previous-bar EMA values,
// needed to detect the bearish cross. rsiOK/stochOK report whether each oscillator produced
// a valid two-bar reading; when false the (now/prev) values are warm-up sentinels (0) and
// must NOT be treated as a real in-zone reading.
type decideInput struct {
	price       float64
	emaFast     float64
	emaFastPrev float64
	emaSlow     float64
	emaSlowPrev float64
	rsiNow      float64
	rsiPrev     float64
	rsiOK       bool
	stochNow    float64
	stochPrev   float64
	stochOK     bool
	pos         *strategy.Position
}

// Decide computes every indicator from md and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := s.decide(s.buildInput(md))
	sig.Ticker = s.ticker
	return sig
}

// buildInput computes every indicator from md and packs them for the pure core.
func (s *Strategy) buildInput(md strategy.MarketData) decideInput {
	emaFast, emaFastPrev := lastTwoEMA(md.Closes, s.p.FastEMA)
	emaSlow, emaSlowPrev := lastTwoEMA(md.Closes, s.p.SlowEMA)

	var rsiNow, rsiPrev float64
	rsiOK := false
	if s.p.RSIPeriod > 0 {
		if r := indicators.RSISeries(md.Closes, s.p.RSIPeriod); len(r) >= 2 {
			rsiNow, rsiPrev = r[len(r)-1], r[len(r)-2]
			rsiOK = true
		}
	}

	var stochNow, stochPrev float64
	stochOK := false
	if s.p.StochKPeriod > 0 && s.p.StochDSmooth > 0 {
		if _, d := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochKPeriod, s.p.StochDSmooth); len(d) >= 2 {
			stochNow, stochPrev = d[len(d)-1], d[len(d)-2]
			stochOK = true
		}
	}

	return decideInput{
		price:       md.Price,
		emaFast:     emaFast,
		emaFastPrev: emaFastPrev,
		emaSlow:     emaSlow,
		emaSlowPrev: emaSlowPrev,
		rsiNow:      rsiNow,
		rsiPrev:     rsiPrev,
		rsiOK:       rsiOK,
		stochNow:    stochNow,
		stochPrev:   stochPrev,
		stochOK:     stochOK,
		pos:         md.Position,
	}
}

// lastTwoEMA returns the latest and previous EMA values for the period. When the series
// has fewer than two points, prev equals now (or both are 0), so no false cross is seen.
func lastTwoEMA(closes []float64, period int) (now, prev float64) {
	e := ema.Compute(closes, period)
	switch {
	case len(e) >= 2:
		return e[len(e)-1], e[len(e)-2]
	case len(e) == 1:
		return e[0], e[0]
	default:
		return 0, 0
	}
}

// crossDown reports a down-cross of level: prev at/above, now below.
func crossDown(prev, now, level float64) bool { return prev >= level && now < level }

// indicatorsReady reports that both oscillators produced valid two-bar readings. A warm-up
// sentinel (rsiOK/stochOK false, values 0) must never count as an in-zone reading, or the
// dual confirmation silently degrades to a single-oscillator gate.
func indicatorsReady(in decideInput) bool {
	return in.rsiOK && in.stochOK
}

// entryFired reports the dual oversold confirmation: one oscillator crosses DOWN into its
// oversold zone while the other is already inside its oversold zone. Simultaneous entry
// (both cross the same bar) satisfies this because the "already inside" test reads now.
func (s *Strategy) entryFired(in decideInput) bool {
	if !indicatorsReady(in) {
		return false
	}
	rsiCrossIn := crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold)
	stochCrossIn := crossDown(in.stochPrev, in.stochNow, s.p.StochOversold)
	rsiIn := in.rsiNow < s.p.RSIOversold
	stochIn := in.stochNow < s.p.StochOversold
	return (rsiCrossIn && stochIn) || (stochCrossIn && rsiIn)
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

	// 1. Optional trend filter.
	if s.p.UseTrend == 1 && !uptrend(in) {
		return sig
	}
	// 2. Dual oversold confirmation.
	if !s.entryFired(in) {
		return sig
	}

	sig.Kind = model.SignalBuy
	sig.RSI = in.rsiNow
	sig.EntryReason = s.entryReason(in)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(in decideInput) string {
	trend := "выкл"
	if s.p.UseTrend == 1 {
		trend = fmt.Sprintf("EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}
	return fmt.Sprintf(
		"Тренд: %s; двойное подтверждение перепроданности: RSI(%d) %.2f→%.2f (зона <%.0f) + Stoch%%D(%d,%d) %.2f→%.2f (зона <%.0f)",
		trend,
		s.p.RSIPeriod, in.rsiPrev, in.rsiNow, s.p.RSIOversold,
		s.p.StochKPeriod, s.p.StochDSmooth, in.stochPrev, in.stochNow, s.p.StochOversold,
	)
}

// manage handles an open long. There is no protective stop. It exits on either a downward
// RSI-50 cross (primary momentum fade) or a bearish EMA cross (FastEMA below SlowEMA). When
// both fire on the same bar RSI50 wins; the fill price (close) is identical either way.
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	sig.RSI = in.rsiNow

	switch {
	case in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, rsiExitLevel):
		sig.Kind, sig.Reason = model.SignalSell, "RSI50"
		sig.ExitReason = fmt.Sprintf("RSI50: RSI %.2f→%.2f пересёк 50 сверху вниз", in.rsiPrev, in.rsiNow)
	case crossDown(in.emaFastPrev-in.emaSlowPrev, in.emaFast-in.emaSlow, 0):
		sig.Kind, sig.Reason = model.SignalSell, "EMAX"
		sig.ExitReason = fmt.Sprintf("EMAX: FastEMA%d %.4f ушла под SlowEMA%d %.4f (медвежий кросс)",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow)
	}
	return sig
}

// Explain re-runs the entry gates over md and reports each gate's value and verdict
// (✓ pass / ✗ block) in entry order, stopping at the first blocker. Diagnostic only.
func (s *Strategy) Explain(md strategy.MarketData) string {
	in := s.buildInput(md)

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

	// 1. Optional trend filter.
	if s.p.UseTrend == 1 {
		if !uptrend(in) {
			return block("Тренд: нужно EMA%d > EMA%d и close > EMA%d (EMA%d=%.4f, EMA%d=%.4f, close=%.4f)",
				s.p.FastEMA, s.p.SlowEMA, s.p.SlowEMA, s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price)
		}
		pass("Тренд↑: EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d", s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}

	// 2. Dual oversold confirmation.
	if !s.entryFired(in) {
		return block("Двойное подтверждение: нет (RSI(%d) %.2f→%.2f зона<%.0f; Stoch%%D %.2f→%.2f зона<%.0f) — нужен кросс одного в зону при другом уже в зоне",
			s.p.RSIPeriod, in.rsiPrev, in.rsiNow, s.p.RSIOversold, in.stochPrev, in.stochNow, s.p.StochOversold)
	}
	pass("Двойное подтверждение: RSI(%d) %.2f→%.2f + Stoch%%D %.2f→%.2f в зоне перепроданности",
		s.p.RSIPeriod, in.rsiPrev, in.rsiNow, in.stochPrev, in.stochNow)

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}
```

- [ ] **Step 4: Update all 8 per-ticker `DefaultParams()`**

In each of these files, replace the three-line `core.Params{...}` body so the `RSIOverbought`, `StochOverbought`, `ATRPeriod`, `ATRMult` fields are gone:

Files: `afks/afks.go`, `gazp/gazp.go`, `mdmg/mdmg.go`, `nvtk/nvtk.go`, `plzl/plzl.go`, `rusal/rusal.go`, `sber/sber.go`, `ydex/ydex.go` (all under `internal/service/trading_strategy/reversion/strategy/`).

Old (identical in all 8):
```go
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20, StochOverbought: 80,
		ATRPeriod: 14, ATRMult: 1.0,
	}
```
New:
```go
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
	}
```

- [ ] **Step 5: Update `genericReversionDefaults()` in the registry**

File `internal/service/backtest/reversion_registry.go`, replace the body of `genericReversionDefaults()`:

Old:
```go
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20, StochOverbought: 80,
		ATRPeriod: 14, ATRMult: 1.0,
	}
```
New:
```go
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
	}
```

- [ ] **Step 6: Fix `TestReversionDefaultsValid` (drops `ATRMult` reference)**

File `internal/service/backtest/reversion_registry_test.go`, replace:
```go
	if p.ATRMult <= 0 || p.SlowEMA <= p.FastEMA || p.RSIPeriod <= 0 || p.StochKPeriod <= 0 || p.StochDSmooth <= 0 {
```
with:
```go
	if p.SlowEMA <= p.FastEMA || p.RSIPeriod <= 0 || p.StochKPeriod <= 0 || p.StochDSmooth <= 0 {
```

- [ ] **Step 7: Build and test the whole repo (GREEN)**

Run: `go build ./... && go test ./internal/service/trading_strategy/reversion/... ./internal/service/backtest/...`
Expected: build succeeds; all listed tests PASS (core exit/entry tests + registry tests).

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy internal/service/backtest/reversion_registry.go internal/service/backtest/reversion_registry_test.go
git commit -m "feat(reversion): EMA-cross/RSI-50 exit, drop ATR stop and overbought params"
```

---

### Task 2: Rewrite the 8 calibration grids

**Files:**
- Modify: `data/params/{afks,gazp,mdmg,nvtk,plzl,sber,ydex}/reversion_grid.json` (7 identical)
- Modify: `data/params/rual/reversion_grid.json` (different)

The exit phase has no tunables left (RSI-50 is a constant; the EMA cross reuses the entry EMAs), so each grid collapses to a single entry/regime phase with only surviving fields.

- [ ] **Step 1: Rewrite the 7 identical grids**

For each of `afks, gazp, mdmg, nvtk, plzl, sber, ydex`, write `data/params/<t>/reversion_grid.json`:
```json
{
  "phases": [
    {
      "name": "entry",
      "keepTop": 5,
      "grid": {
        "UseTrend": [0, 1],
        "FastEMA": [50],
        "SlowEMA": [200],
        "RSIPeriod": [7, 14],
        "RSIOversold": [15, 20, 30],
        "StochKPeriod": [9, 14],
        "StochDSmooth": [1, 3],
        "StochOversold": [15, 20, 25]
      }
    }
  ]
}
```

- [ ] **Step 2: Rewrite the RUAL grid**

Write `data/params/rual/reversion_grid.json` (drops `RSIOverbought`/`StochOverbought`, folds `UseTrend` into the single phase):
```json
{
  "phases": [
    {
      "name": "entry",
      "keepTop": 5,
      "grid": {
        "UseTrend": [0, 1],
        "FastEMA": [10],
        "SlowEMA": [200],
        "RSIPeriod": [10],
        "RSIOversold": [30],
        "StochKPeriod": [14],
        "StochDSmooth": [1],
        "StochOversold": [20, 25]
      }
    }
  ]
}
```

- [ ] **Step 3: Validate JSON well-formedness**

Run: `for f in data/params/*/reversion_grid.json; do python3 -m json.tool "$f" >/dev/null && echo "ok $f"; done`
Expected: `ok` for all 8 files, no parse errors.

- [ ] **Step 4: Smoke-run one calibration to confirm the grid binds (no unknown-field error)**

Run: `go run ./cmd/backtest -ticker AFKS -strategy reversion -interval Day1 -calibrate data/params/afks/reversion_grid.json -out ./reports/AFKS -months 24 -test-months 6 -min-trades 5 -metric profit_factor`
Expected: completes without an "unknown param"/reflection error (a low trade count or weak PF is fine — we only verify the grid maps onto the trimmed `Params`).

- [ ] **Step 5: Commit**

```bash
git add data/params/*/reversion_grid.json
git commit -m "chore(reversion): trim calibration grids to surviving params"
```

---

### Task 3: Update the strategy explainer doc

**Files:**
- Modify: `docs/reversion/strategy.md`

- [ ] **Step 1: Rewrite the intro, Exit section, entry stop bullet, and Params list**

In `docs/reversion/strategy.md`:

Replace the intro sentence (lines ~5-8) "...and exits when one is already overbought ... The only protective stop is a daily-ATR stop frozen at entry." with:
```
is already inside its oversold zone and the other crosses into it. It exits on either of
two momentum-fade signals — RSI crossing 50 downward, or a bearish FastEMA/SlowEMA cross —
and carries no protective stop. An optional trend filter restricts buys to a confirmed uptrend.
```

Remove entry gate item **3 (Protective stop)** entirely (the `ATRMult > 0 ... frozen at entry` bullet).

Replace the whole **Exit** section with:
```
## Exit (first trigger wins)

There is no protective stop. An open long exits on either signal, filled at the bar close:

1. **RSI50:** RSI crosses the 50 line from above (`prev ≥ 50`, `now < 50`) — the primary
   momentum-fade exit.
2. **EMAX:** bearish EMA cross — `EMA(FastEMA)` drops below `EMA(SlowEMA)`. A slow
   regime-break backstop; reuses the same EMAs as the trend filter.

If both fire on the same bar, RSI50 is reported (the fill is identical either way).
```

Replace the **Params** list with:
```
`UseTrend, FastEMA, SlowEMA, RSIPeriod, RSIOversold, StochKPeriod, StochDSmooth,
StochOversold`. Flags (`UseTrend`) are int `0/1`; the rest are int/float64 so the grid
calibrator can sweep them. The RSI-50 exit level is a fixed constant, not a param.
```

- [ ] **Step 2: Commit**

```bash
git add docs/reversion/strategy.md
git commit -m "docs(reversion): document EMA-cross/RSI-50 exit and no stop"
```

---

## Self-Review notes

- **Spec coverage:** new exit (RSI50 + EMAX) → Task 1 core.manage + tests; no stop → Task 1 (entry drops ATR, manage has no SL); dead-param removal → Task 1 Params/per-ticker/registry; grids → Task 2; doc → Task 3. All spec sections covered.
- **Type consistency:** `decideInput` gains `emaFastPrev/emaSlowPrev`, drops `atr/barLow`; `crossDown` signature `(prev, now, level)` reused for both RSI and the EMA-diff cross; `crossUp`/`exitFired`/`entryReason(stop,risk)` removed and never referenced afterward; `model.Signal.StopLoss`/`ATR` left unset (engine treats 0 stop as "no stop"). Reasons `"RSI50"`/`"EMAX"` are not in the engine's `SL|TRAIL|TP` fill switch, so they fill at close — intended.
- **No placeholders:** every code/JSON block is complete.
```
