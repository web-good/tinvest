# Reversion v3 — RSI + Stochastic двойное подтверждение — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Сделать Stochastic постоянной частью ядра `reversion` и заменить одноиндикаторный RSI-триггер на правило согласия двух осцилляторов «один уже в зоне, второй только заходит», с подбором зон/периодов Stochastic через grid.

**Architecture:** Ядро `core.decide()` остаётся чистым над предрасчитанными индикаторами. Добавляется `indicators.StochasticSeries` (срезы %K/%D) — ядру нужны `now`/`prev` рабочей линии %D для детекции кросса. `Params` получает поля Stochastic, теряет `EntryMode`/`ExitMode` (направление кросса зафиксировано). Дефолты 8 тикеров + generic + сетки + docs приводятся к новой форме.

**Tech Stack:** Go 1.25, табличные тесты `testing`, фазовая grid-калибровка из JSON, дневной таймфрейм (`-interval Day1`).

**Спецификация:** `docs/superpowers/specs/2026-06-12-reversion-rsi-stoch-dual-design.md`

---

## File Structure

- `pkg/indicators/stochastic.go` — `+StochasticSeries`; `Stochastic` рефакторится поверх неё (поведение сохранено).
- `pkg/indicators/stochastic_test.go` — `+` тест серии.
- `internal/service/trading_strategy/reversion/strategy/core/core.go` — новый `Params`, двойная логика, Stoch в `buildInput`/`Lookback`/`Explain`.
- `internal/service/trading_strategy/reversion/strategy/core/core_test.go` — переписанные тесты.
- `internal/service/trading_strategy/reversion/strategy/{afks,gazp,mdmg,nvtk,plzl,rusal,sber,ydex}/<ticker>.go` — `DefaultParams`.
- `internal/service/backtest/reversion_registry.go` — `genericReversionDefaults`.
- `internal/service/backtest/reversion_registry_test.go` — override-тест.
- `data/params/{afks,gazp,mdmg,nvtk,plzl,rual,sber,ydex}/reversion_grid.json` — новые сетки.
- `docs/reversion/strategy.md` — объяснение.

**Порядок:** Task 1 (индикатор) самодостаточен. Task 2 (ядро+тесты) переписывается целиком — после него пакет `core` собирается, но потребители (тикеры, registry) временно сломаны до Task 3–4. Полный `./...` гоняем в конце Task 4.

---

## Task 1: `StochasticSeries` в `pkg/indicators`

**Files:**
- Modify: `pkg/indicators/stochastic.go`
- Test: `pkg/indicators/stochastic_test.go`

- [ ] **Step 1: Добавить failing-тест серии**

Добавить в `pkg/indicators/stochastic_test.go` (не трогая существующий `TestStochastic_RampFixture`):

```go
func TestStochasticSeries_RampFixture(t *testing.T) {
	// Same ramp as TestStochastic_RampFixture. Windows of 3:
	//   %K = [50, 100, 100]; %D(smooth 3) only one full value: (50+100+100)/3 = 83.33
	highs := []float64{10, 12, 11, 13, 14}
	lows := []float64{10, 12, 11, 13, 14}
	closes := []float64{10, 12, 11, 13, 14}

	ks, ds := StochasticSeries(highs, lows, closes, 3, 3)
	if len(ks) != 3 {
		t.Fatalf("len(ks)=%d want 3", len(ks))
	}
	wantKs := []float64{50, 100, 100}
	for i, w := range wantKs {
		if math.Abs(ks[i]-w) > 1e-9 {
			t.Fatalf("ks[%d]=%v want %v", i, ks[i], w)
		}
	}
	if len(ds) != 1 {
		t.Fatalf("len(ds)=%d want 1", len(ds))
	}
	if math.Abs(ds[0]-83.333333) > 1e-4 {
		t.Fatalf("ds[0]=%v want ~83.33", ds[0])
	}
}

func TestStochasticSeries_DSmoothTwoYieldsPrev(t *testing.T) {
	// dSmooth=2 over the ramp gives ds of length 2 so a cross has prev+now.
	highs := []float64{10, 12, 11, 13, 14}
	lows := []float64{10, 12, 11, 13, 14}
	closes := []float64{10, 12, 11, 13, 14}
	_, ds := StochasticSeries(highs, lows, closes, 3, 2)
	// ks=[50,100,100]; ds=[(50+100)/2=75, (100+100)/2=100]
	if len(ds) != 2 || math.Abs(ds[0]-75) > 1e-9 || math.Abs(ds[1]-100) > 1e-9 {
		t.Fatalf("ds=%v want [75 100]", ds)
	}
}

func TestStochasticSeries_InsufficientHistory(t *testing.T) {
	ks, ds := StochasticSeries([]float64{1, 2}, []float64{1, 2}, []float64{1, 2}, 5, 3)
	if ks != nil || ds != nil {
		t.Fatalf("want nil,nil for short history; got ks=%v ds=%v", ks, ds)
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что не компилируется**

Run: `go test ./pkg/indicators/ -run TestStochasticSeries 2>&1 | head`
Expected: FAIL компиляции (`StochasticSeries` не определена).

- [ ] **Step 3: Реализовать `StochasticSeries` и переписать `Stochastic` поверх неё**

Заменить содержимое `pkg/indicators/stochastic.go`:

```go
package indicators

// StochasticSeries returns the right-aligned %K and %D series of the stochastic
// oscillator. %K[t] = 100 * (close - lowestLow) / (highestHigh - lowestLow) over the
// trailing kPeriod bars; a window whose high/low range collapses to zero yields 0. %D
// is the simple moving average of %K over dSmooth values, so len(ds) = len(ks)-dSmooth+1
// (every %D value is fully smoothed — there are no warm-up zeros). Both are nil when the
// inputs are malformed or history is shorter than kPeriod (ds also nil when fewer than
// dSmooth %K values exist).
func StochasticSeries(highs, lows, closes []float64, kPeriod, dSmooth int) (ks, ds []float64) {
	n := len(closes)
	if kPeriod <= 0 || dSmooth <= 0 || n < kPeriod || len(highs) < n || len(lows) < n {
		return nil, nil
	}

	ks = make([]float64, 0, n-kPeriod+1)
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

	if len(ks) < dSmooth {
		return ks, nil
	}
	ds = make([]float64, 0, len(ks)-dSmooth+1)
	for j := dSmooth - 1; j < len(ks); j++ {
		var sum float64
		for i := j - dSmooth + 1; i <= j; i++ {
			sum += ks[i]
		}
		ds = append(ds, sum/float64(dSmooth))
	}
	return ks, ds
}

// Stochastic returns the latest %K and %D of the stochastic oscillator. %D is 0 when
// history is insufficient (fewer than dSmooth %K values); both are 0 when there is no %K
// at all. Thin wrapper over StochasticSeries.
func Stochastic(highs, lows, closes []float64, kPeriod, dSmooth int) (k, d float64) {
	ks, ds := StochasticSeries(highs, lows, closes, kPeriod, dSmooth)
	if len(ks) == 0 {
		return 0, 0
	}
	k = ks[len(ks)-1]
	if len(ds) == 0 {
		return k, 0
	}
	return k, ds[len(ds)-1]
}
```

- [ ] **Step 4: Запустить тесты пакета — зелёные**

Run: `go test ./pkg/indicators/ 2>&1 | tail`
Expected: PASS (старый `TestStochastic_RampFixture` + новые `TestStochasticSeries_*`).

- [ ] **Step 5: Commit**

```bash
git add pkg/indicators/stochastic.go pkg/indicators/stochastic_test.go
git commit -m "feat(indicators): StochasticSeries (%K/%D series) for cross detection

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Переписать ядро `core.go` + тесты под двойное подтверждение

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go`
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Переписать тесты `core_test.go` (failing)**

Полностью заменить содержимое файла:

```go
package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// defaultParams returns valid, entry-capable params: trend on, RSI/Stoch oversold 20,
// overbought 70/80, ATR stop = 1x ATR.
func defaultParams() Params {
	return Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20, StochOverbought: 80,
		ATRPeriod: 14, ATRMult: 1.0,
	}
}

// passingInput clears every entry gate: uptrend; RSI crosses DOWN into oversold while
// Stoch %D is already in the oversold zone; ATR positive.
func passingInput() decideInput {
	return decideInput{
		price: 100, atr: 2, emaFast: 95, emaSlow: 90,
		rsiPrev: 25, rsiNow: 15, // crossDown through 20 (RSI enters oversold)
		stochPrev: 10, stochNow: 8, // already < 20 (Stoch already in zone)
		barLow: 100,
	}
}

// openInput is an open position above its stop with neutral oscillators (no exit).
func openInput() decideInput {
	in := passingInput()
	in.pos = &strategy.Position{PurchasePrice: 100, StopLoss: 98}
	in.rsiPrev, in.rsiNow = 50, 55
	in.stochPrev, in.stochNow = 50, 55
	return in
}

func TestEntryAllGatesPass(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	sig := s.decide(passingInput())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got kind=%v", sig.Kind)
	}
	if math.Abs(sig.StopLoss-98) > 1e-9 { // 100 - 1.0*2
		t.Fatalf("StopLoss=%v want 98", sig.StopLoss)
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
	in.rsiPrev, in.rsiNow = 15, 12 // already < 20, no fresh cross
	in.stochPrev, in.stochNow = 25, 15 // crossDown through 20
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("Stoch cross + RSI already in: want Buy, got %v", sig.Kind)
	}
}

func TestSimultaneousEntry(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.rsiPrev, in.rsiNow = 25, 15 // RSI crosses in
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

func TestExitDualOverbought(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// RSI crosses UP through 70 while Stoch already > 80 -> sell.
	in := openInput()
	in.rsiPrev, in.rsiNow = 65, 75
	in.stochPrev, in.stochNow = 85, 85
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "XOVER" {
		t.Fatalf("dual overbought: want XOVER sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}

	// RSI crosses up but Stoch not high -> no sell.
	in = openInput()
	in.rsiPrev, in.rsiNow = 65, 75
	in.stochPrev, in.stochNow = 50, 55
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("RSI high but Stoch low: should NOT sell")
	}

	// Stoch crosses up while RSI already high -> sell (mirror branch).
	in = openInput()
	in.rsiPrev, in.rsiNow = 75, 75 // already > 70, no cross
	in.stochPrev, in.stochNow = 75, 85 // crossUp through 80
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "XOVER" {
		t.Fatalf("Stoch cross + RSI already high: want XOVER sell, got %v/%q", sig.Kind, sig.Reason)
	}
}

func TestStopSanityBlocksWhenNoStop(t *testing.T) {
	p := defaultParams()
	p.ATRMult = 0
	s := NewWithParams("TEST", p)
	if sig := s.decide(passingInput()); sig.Kind == model.SignalBuy {
		t.Fatalf("ATRMult=0: want no Buy (safety mandatory)")
	}
	s2 := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.atr = 0
	if sig := s2.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("atr=0: want no Buy")
	}
}

func TestStopLevelUsesATRMult(t *testing.T) {
	p := defaultParams()
	p.ATRMult = 1.5
	s := NewWithParams("TEST", p)
	in := passingInput()
	in.atr = 2 // stop = 100 - 1.5*2 = 97
	sig := s.decide(in)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got %v", sig.Kind)
	}
	if math.Abs(sig.StopLoss-97) > 1e-9 {
		t.Fatalf("StopLoss=%v want 97", sig.StopLoss)
	}
}

func TestExitSL(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barLow = 97 // <= stop 98
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want SL sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestProtectiveStopWinsTie(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barLow = 97 // SL hit (stop 98)
	in.rsiPrev, in.rsiNow = 65, 75 // overbought cross too
	in.stochPrev, in.stochNow = 85, 85
	sig := s.decide(in)
	if sig.Reason != "SL" {
		t.Fatalf("protective first: want SL, got %q", sig.Reason)
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
		Position: &strategy.Position{PurchasePrice: 100, StopLoss: 98},
	}
	if out := s.Explain(md); !strings.Contains(out, "позиция уже открыта") {
		t.Fatalf("Explain with open position: %q", out)
	}
}
```

- [ ] **Step 2: Запустить тесты — не компилируется**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ 2>&1 | head -20`
Expected: FAIL компиляции (новые поля `Params`/`decideInput` ещё не объявлены).

- [ ] **Step 3: Переписать `core.go`**

Полностью заменить содержимое `core.go`:

```go
// Package core implements a long-only mean-reversion strategy on the daily timeframe,
// driven by the agreement of two oscillators: RSI and the Stochastic %D line. It buys
// when one oscillator is already inside its oversold zone and the other crosses into it,
// and exits (XOVER) when one is already overbought and the other crosses up into the
// overbought zone. The protective ATR stop is frozen at entry and checked first. An
// optional trend filter restricts buys to a confirmed uptrend. The decision logic is pure
// and ticker-agnostic; per-share packages supply ticker + Params. Run with `-interval Day1`.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	UseTrend        int     // 1 = require uptrend before buying; 0 = ignore trend
	FastEMA         int     // fast regime EMA (e.g. 50)
	SlowEMA         int     // slow regime EMA + price floor (e.g. 200)
	RSIPeriod       int     // RSI length; required (>0)
	RSIOversold     float64 // RSI oversold zone (entry side)
	RSIOverbought   float64 // RSI overbought zone (exit side)
	StochKPeriod    int     // Stochastic %K lookback; required (>0)
	StochDSmooth    int     // Stochastic %D smoothing; required (>0); 1 = raw %K
	StochOversold   float64 // Stochastic oversold zone (entry side)
	StochOverbought float64 // Stochastic overbought zone (exit side)
	ATRPeriod       int     // ATR length for the stop
	ATRMult         float64 // stop = entry - ATRMult*ATR; must be > 0
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
		s.p.ATRPeriod + 1,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core. stochNow/Prev
// are the %D line (smoothed %K).
type decideInput struct {
	price     float64
	atr       float64
	emaFast   float64
	emaSlow   float64
	rsiNow    float64
	rsiPrev   float64
	stochNow  float64
	stochPrev float64
	barLow    float64
	pos       *strategy.Position
}

// Decide computes every indicator from md and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := s.decide(s.buildInput(md))
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

	var stochNow, stochPrev float64
	if s.p.StochKPeriod > 0 && s.p.StochDSmooth > 0 {
		if _, d := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochKPeriod, s.p.StochDSmooth); len(d) >= 2 {
			stochNow, stochPrev = d[len(d)-1], d[len(d)-2]
		}
	}

	var barLow float64
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	return decideInput{
		price:     md.Price,
		atr:       atr,
		emaFast:   emaFast,
		emaSlow:   emaSlow,
		rsiNow:    rsiNow,
		rsiPrev:   rsiPrev,
		stochNow:  stochNow,
		stochPrev: stochPrev,
		barLow:    barLow,
		pos:       md.Position,
	}
}

// crossUp reports an up-cross of level: prev at/below, now above.
func crossUp(prev, now, level float64) bool { return prev <= level && now > level }

// crossDown reports a down-cross of level: prev at/above, now below.
func crossDown(prev, now, level float64) bool { return prev >= level && now < level }

// indicatorsReady reports that both oscillators are configured (valid readings possible).
func (s *Strategy) indicatorsReady() bool {
	return s.p.RSIPeriod > 0 && s.p.StochKPeriod > 0 && s.p.StochDSmooth > 0
}

// entryFired reports the dual oversold confirmation: one oscillator crosses DOWN into its
// oversold zone while the other is already inside its oversold zone. Simultaneous entry
// (both cross the same bar) satisfies this because the "already inside" test reads now.
func (s *Strategy) entryFired(in decideInput) bool {
	if !s.indicatorsReady() {
		return false
	}
	rsiCrossIn := crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold)
	stochCrossIn := crossDown(in.stochPrev, in.stochNow, s.p.StochOversold)
	rsiIn := in.rsiNow < s.p.RSIOversold
	stochIn := in.stochNow < s.p.StochOversold
	return (rsiCrossIn && stochIn) || (stochCrossIn && rsiIn)
}

// exitFired reports the dual overbought confirmation: one oscillator crosses UP into its
// overbought zone while the other is already above its overbought zone.
func (s *Strategy) exitFired(in decideInput) bool {
	if !s.indicatorsReady() {
		return false
	}
	rsiCrossUp := crossUp(in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
	stochCrossUp := crossUp(in.stochPrev, in.stochNow, s.p.StochOverbought)
	rsiHigh := in.rsiNow > s.p.RSIOverbought
	stochHigh := in.stochNow > s.p.StochOverbought
	return (rsiCrossUp && stochHigh) || (stochCrossUp && rsiHigh)
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
	// 3. ATR stop is mandatory and must size a positive risk.
	if s.p.ATRMult <= 0 || in.atr <= 0 {
		return sig
	}
	stop := in.price - s.p.ATRMult*in.atr
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
	trend := "выкл"
	if s.p.UseTrend == 1 {
		trend = fmt.Sprintf("EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}
	return fmt.Sprintf(
		"Тренд: %s; двойное подтверждение перепроданности: RSI(%d) %.2f→%.2f (зона <%.0f) + Stoch%%D(%d,%d) %.2f→%.2f (зона <%.0f); SL=%.4f (−%.2g×ATR %.4f, риск %.4f)",
		trend,
		s.p.RSIPeriod, in.rsiPrev, in.rsiNow, s.p.RSIOversold,
		s.p.StochKPeriod, s.p.StochDSmooth, in.stochPrev, in.stochNow, s.p.StochOversold,
		stop, s.p.ATRMult, in.atr, risk,
	)
}

// manage handles an open long: the frozen ATR stop first, then the dual overbought exit.
// Protective stops are checked first so the worst case for the position wins ties.
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	hardSL := in.pos.StopLoss
	sig.StopLoss = hardSL
	sig.RSI = in.rsiNow

	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (зафиксирован на входе)", in.barLow, hardSL)
	case s.exitFired(in):
		sig.Kind, sig.Reason = model.SignalSell, "XOVER"
		sig.ExitReason = fmt.Sprintf(
			"XOVER: RSI %.2f→%.2f (зона >%.0f) + Stoch%%D %.2f→%.2f (зона >%.0f) — двойное подтверждение перекупленности",
			in.rsiPrev, in.rsiNow, s.p.RSIOverbought, in.stochPrev, in.stochNow, s.p.StochOverbought)
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

	// 3. ATR stop.
	if s.p.ATRMult <= 0 {
		return block("Стоп: ATRMult=%.2g ≤ 0 — защита не задана", s.p.ATRMult)
	}
	if in.atr <= 0 {
		return block("Стоп: ATR=%.4f ≤ 0 — нельзя рассчитать стоп", in.atr)
	}
	stop := in.price - s.p.ATRMult*in.atr
	pass("Стоп: SL=%.4f (−%.2g×ATR %.4f)", stop, s.p.ATRMult, in.atr)

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}
```

- [ ] **Step 4: Запустить тесты ядра — зелёные**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -v 2>&1 | tail -40`
Expected: PASS (все `TestEntry*`, `TestSimultaneousEntry`, `TestTrend*`, `TestExit*`, `TestStop*`, `TestProtectiveStopWinsTie`, `TestExplain*`).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/
git commit -m "feat(reversion): dual RSI+Stochastic confirmation core

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Привести 8 тикерных `DefaultParams` к новой форме

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/{afks,gazp,mdmg,nvtk,plzl,rusal,sber,ydex}/<ticker>.go`

- [ ] **Step 1: Заменить тело `core.Params{...}` во всех 8 файлах**

В каждом файле блок внутри `return core.Params{...}` заменить на единый baseline (имена пакетов/тикеров/комментарии шапки не трогать):

```go
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20, StochOverbought: 80,
		ATRPeriod: 14, ATRMult: 1.0,
	}
```

Файлы: `afks/afks.go`, `gazp/gazp.go`, `mdmg/mdmg.go`, `nvtk/nvtk.go`, `plzl/plzl.go`, `rusal/rusal.go`, `sber/sber.go`, `ydex/ydex.go`.

- [ ] **Step 2: Собрать тикерные пакеты + gofmt**

Run: `gofmt -w internal/service/trading_strategy/reversion/strategy/*/*.go && go build ./internal/service/trading_strategy/reversion/...`
Expected: успешная сборка.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/
git commit -m "feat(reversion): per-ticker defaults for RSI+Stochastic params

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Generic-дефолты + registry-тест + полный прогон

**Files:**
- Modify: `internal/service/backtest/reversion_registry.go` (тело `genericReversionDefaults`)
- Modify: `internal/service/backtest/reversion_registry_test.go`

- [ ] **Step 1: Заменить тело `genericReversionDefaults`**

В `reversion_registry.go` заменить возвращаемый `core.Params{...}` на:

```go
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20, StochOverbought: 80,
		ATRPeriod: 14, ATRMult: 1.0,
	}
```

- [ ] **Step 2: Обновить override-тест и валидатор**

В `reversion_registry_test.go` заменить блок ParseParams внутри `TestReversionLookupGenericFallback`:

```go
	// ParseParams must layer the override on top of genericReversionDefaults.
	got, err := b.ParseParams([]byte(`{"StochOversold": 15}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.StochOversold != 15 {
		t.Fatalf("StochOversold=%v want 15 (override)", p.StochOversold)
	}
	if p.FastEMA != 50 || p.SlowEMA != 200 {
		t.Fatalf("generic defaults not preserved: FastEMA=%d SlowEMA=%d want 50/200", p.FastEMA, p.SlowEMA)
	}
```

И заменить `TestReversionDefaultsValid`:

```go
func TestReversionDefaultsValid(t *testing.T) {
	p := genericReversionDefaults()
	if p.ATRMult <= 0 || p.SlowEMA <= p.FastEMA || p.RSIPeriod <= 0 || p.StochKPeriod <= 0 || p.StochDSmooth <= 0 {
		t.Fatalf("invalid generic defaults: %+v", p)
	}
}
```

- [ ] **Step 3: Тесты пакета backtest**

Run: `go test ./internal/service/backtest/ 2>&1 | tail -20`
Expected: PASS (`TestReversionLookup*`, `TestReversionDefaultsValid`).

- [ ] **Step 4: Полная сборка, vet, тесты**

Run: `go build ./... && go vet ./... && go test ./... 2>&1 | tail -30`
Expected: всё зелёное.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/
git commit -m "feat(reversion): generic RSI+Stochastic defaults + registry test

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Новые сетки калибровки для 8 тикеров

**Files:**
- Modify: `data/params/{afks,gazp,mdmg,nvtk,plzl,rual,sber,ydex}/reversion_grid.json`

> Папка RUSAL — `rual/` (известное расхождение `rusal`≠`rual`). Все 8: `afks, gazp, mdmg, nvtk, plzl, rual, sber, ydex`.

- [ ] **Step 1: Записать единую сетку во все 8 файлов**

Каждый `data/params/<ticker>/reversion_grid.json` — идентичное содержимое:

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
    },
    {
      "name": "exit",
      "keepTop": 5,
      "grid": {
        "RSIOverbought": [70, 80],
        "StochOverbought": [75, 80, 85],
        "ATRMult": [1.0, 1.5, 2.0],
        "ATRPeriod": [14]
      }
    }
  ]
}
```

- [ ] **Step 2: Проверить валидность JSON**

Run: `for f in data/params/{afks,gazp,mdmg,nvtk,plzl,rual,sber,ydex}/reversion_grid.json; do python3 -m json.tool "$f" >/dev/null && echo "ok $f" || echo "BAD $f"; done`
Expected: `ok` для всех 8.

- [ ] **Step 3: Дымовой прогон калибровки на одном тикере**

Run: `go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 -calibrate data/params/sber/reversion_grid.json -out ./reports/SBER -months 24 -min-trades 5 -test-months 6 -metric profit_factor 2>&1 | tail -25`
Expected: обе фазы (`entry`, `exit`) отрабатывают, пишется `*_calibration.md` без паник. (Требуется токен в `env/local.env`; если сети/токена нет — зафиксировать и пропустить, не считая провалом.)

- [ ] **Step 4: Commit**

```bash
git add data/params/afks/reversion_grid.json data/params/gazp/reversion_grid.json data/params/mdmg/reversion_grid.json data/params/nvtk/reversion_grid.json data/params/plzl/reversion_grid.json data/params/rual/reversion_grid.json data/params/sber/reversion_grid.json data/params/ydex/reversion_grid.json
git commit -m "feat(reversion): RSI+Stochastic dual calibration grids for all 8 tickers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Обновить документацию стратегии

**Files:**
- Modify: `docs/reversion/strategy.md`

- [ ] **Step 1: Переписать `docs/reversion/strategy.md`**

Заменить содержимое файла:

```markdown
# Reversion strategy (RSI + Stochastic dual confirmation)

Long-only, **daily timeframe** (`-interval Day1`). A mean-reversion core driven by the
agreement of two oscillators — RSI and the Stochastic %D line. It buys when one oscillator
is already inside its oversold zone and the other crosses into it, and exits when one is
already overbought and the other crosses up into the overbought zone. An optional trend
filter restricts buys to a confirmed uptrend. The only protective stop is a daily-ATR stop
frozen at entry.

The Stochastic working line is **%D** (SMA of %K over `StochDSmooth`; `StochDSmooth=1`
gives the raw %K). Volume gating and the time-stop from earlier versions are gone.

## Entry (gates in short-circuit order)

1. **Trend filter (optional, `UseTrend`):** `1` (default) requires
   `EMA(FastEMA) > EMA(SlowEMA)` and `close > EMA(SlowEMA)` (defaults 50/200); `0` ignores
   trend.
2. **Dual oversold confirmation** — at least one of:
   - RSI(`RSIPeriod`) crosses **down** through `RSIOversold` **and** Stoch %D is already
     `< StochOversold`;
   - Stoch %D crosses **down** through `StochOversold` **and** RSI is already
     `< RSIOversold`.
   Both crossing into the zone on the same bar also fires.
3. **Protective stop:** `ATRMult > 0` and `ATR > 0` required; stop =
   `entry − ATRMult × ATR(ATRPeriod)`, frozen at entry.

## Exit (first trigger wins; protective first)

1. **SL:** bar low ≤ the frozen ATR stop.
2. **XOVER** — at least one of:
   - RSI crosses **up** through `RSIOverbought` **and** Stoch %D is already
     `> StochOverbought`;
   - Stoch %D crosses **up** through `StochOverbought` **and** RSI is already
     `> RSIOverbought`.

## Run

```bash
# single run (daily timeframe)
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -months 12 -out ./reports/SBER

# grid calibration with walk-forward OOS (Stochastic zones/periods are swept)
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -calibrate data/params/sber/reversion_grid.json -out ./reports/SBER \
  -months 24 -test-months 6 -min-trades 20 -metric profit_factor

# diagnose one bar
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -explain '2026-03-14 12:00' -months 12
```

## Params

`UseTrend, FastEMA, SlowEMA, RSIPeriod, RSIOversold, RSIOverbought, StochKPeriod,
StochDSmooth, StochOversold, StochOverbought, ATRPeriod, ATRMult`. Flags (`UseTrend`) are
int `0/1`; the rest are int/float64 so the grid calibrator can sweep them — including the
Stochastic zones and periods.

## Not yet supported

`-basket` walk-forward (the basket runner is currently momentum-only).
```

- [ ] **Step 2: Commit**

```bash
git add docs/reversion/strategy.md
git commit -m "docs(reversion): rewrite explainer for RSI+Stochastic dual confirmation

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review (выполнено при написании плана)

- **Покрытие спеки:** `StochasticSeries` (Task 1); новый `Params` + двойная логика входа/выхода + удаление `EntryMode`/`ExitMode` (Task 2 `core.go`); тесты двух веток входа/выхода, одновременного входа, требования обоих индикаторов, приоритета SL, тренд-тумблера, ATR-санити, Explain (Task 2 тесты); `Lookback` со Stoch (Task 2); дефолты тикеров (Task 3); generic + registry (Task 4); сетки со свипом зон/периодов Stochastic (Task 5); docs (Task 6).
- **Плейсхолдеры:** отсутствуют; весь код приведён целиком.
- **Согласованность типов:** поля `StochKPeriod/StochDSmooth/StochOversold/StochOverbought`, методы `entryFired/exitFired/indicatorsReady`, reason `XOVER`, рабочая линия %D — единообразны между `core.go`, тестами, дефолтами, registry и JSON-сетками. Старые `EntryMode/ExitMode` удалены везде, где грепались.
```
