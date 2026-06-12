# Reversion → дневной RSI: переработка — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Переработать стратегию `reversion` в чистый RSI mean-reversion на дневном таймфрейме: вход/выход управляются RSI с калибруемыми длиной, зонами и режимами «вход/выход из зоны», опциональный фильтр тренда, ATR-стоп с множителем.

**Architecture:** Вся логика в `internal/service/trading_strategy/reversion/strategy/core/core.go` (чистое ядро `decide()` над предрасчитанными индикаторами). Меняется только набор `Params` и правила входа/выхода; плумбинг дневных свечей уже есть в движке — запуск с `-interval Day1`. Удаляются volume/stochastic/time-stop из логики (пакет `pkg/indicators/Stochastic` остаётся в репо неиспользованным). Сетки и дефолты приводятся к новой форме в 8 тикерных пакетах + generic.

**Tech Stack:** Go 1.25, табличные тесты `testing`, фазовая grid-калибровка из JSON.

**Спецификация:** `docs/superpowers/specs/2026-06-12-reversion-rsi-daily-rework-design.md`

---

## File Structure

- `internal/service/trading_strategy/reversion/strategy/core/core.go` — новый `Params`, `decide`/`manage`/`dipFired`/`exitFired`/`uptrend`/`Explain`/`Lookback`/`buildInput`.
- `internal/service/trading_strategy/reversion/strategy/core/core_test.go` — переписанные табличные тесты.
- `internal/service/trading_strategy/reversion/strategy/{afks,gazp,mdmg,nvtk,plzl,rusal,sber,ydex}/<ticker>.go` — `DefaultParams` под новую форму.
- `internal/service/backtest/reversion_registry.go` — `genericReversionDefaults`.
- `internal/service/backtest/reversion_registry_test.go` — обновить override-тест (`StopLossPct` → `ATRMult`).
- `data/params/{afks,gazp,mdmg,nvtk,plzl,rual,sber,ydex}/reversion_grid.json` — новые фазовые сетки.
- `docs/reversion/strategy.md` — обновить объяснение.

**Важно про порядок:** ядро (`core.go`) и его тесты переписываются в одной задаче, потому что старые тесты ссылаются на удаляемые поля и старая структура `Params` не скомпилируется частично. После этого пакет `core` собирается, но пакеты-потребители (тикеры, registry) временно сломаны — их чиним в следующих задачах до того, как гонять весь `./...`.

---

## Task 1: Переписать ядро `core.go` + тесты под новый `Params`

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

// defaultParams returns valid, entry-capable params for tests: trend on,
// EntryMode "exit oversold zone" (up-cross through 40), ExitMode "enter
// overbought zone" (up-cross through 70), ATR stop = 1x ATR.
func defaultParams() Params {
	return Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 6, RSIOversold: 40, RSIOverbought: 70,
		EntryMode: triggerExitZone, ExitMode: triggerEnterZone,
		ATRPeriod: 14, ATRMult: 1.0,
	}
}

// passingInput returns a flat decideInput that clears every entry gate: uptrend,
// RSI up-cross through 40 (exit of oversold zone), ATR positive.
func passingInput() decideInput {
	return decideInput{
		price:   100,
		atr:     2,
		emaFast: 95,
		emaSlow: 90,
		rsiPrev: 38, // below 40
		rsiNow:  45, // above 40 -> up-cross (exit oversold) fires
		barLow:  100,
	}
}

// openInput returns an input with an open position above its stop (no exit triggers).
func openInput() decideInput {
	in := passingInput()
	in.pos = &strategy.Position{PurchasePrice: 100, StopLoss: 98}
	in.rsiPrev, in.rsiNow = 45, 50 // not crossing overbought
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
	if !strings.Contains(sig.EntryReason, "RSI(6)") {
		t.Fatalf("EntryReason missing RSI detail: %q", sig.EntryReason)
	}
}

func TestTrendFilterToggles(t *testing.T) {
	// UseTrend=1: fast below slow -> blocked.
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.emaFast = 85
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, fast<slow: want no Buy")
	}
	in = passingInput()
	in.price = 89 // below slow EMA (90)
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, price<slowEMA: want no Buy")
	}

	// UseTrend=0: same broken-trend input now passes (trend ignored).
	p := defaultParams()
	p.UseTrend = 0
	s0 := NewWithParams("TEST", p)
	in = passingInput()
	in.emaFast = 85
	if sig := s0.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("UseTrend=0: want Buy regardless of trend, got %v", sig.Kind)
	}
}

func TestEntryModeEnterVsExitZone(t *testing.T) {
	up := passingInput()                       // 38 -> 45: up-cross (exit oversold)
	down := passingInput()                      // build a down-cross into oversold
	down.rsiPrev, down.rsiNow = 45, 38          // 45 -> 38: down-cross (enter oversold)

	// EntryMode = exit zone (default): up-cross fires, down-cross does not.
	s := NewWithParams("TEST", defaultParams())
	if !s.entryFired(up) {
		t.Fatalf("exit-zone: up-cross should fire")
	}
	if s.entryFired(down) {
		t.Fatalf("exit-zone: down-cross should NOT fire")
	}

	// EntryMode = enter zone: down-cross fires, up-cross does not.
	p := defaultParams()
	p.EntryMode = triggerEnterZone
	se := NewWithParams("TEST", p)
	if !se.entryFired(down) {
		t.Fatalf("enter-zone: down-cross should fire")
	}
	if se.entryFired(up) {
		t.Fatalf("enter-zone: up-cross should NOT fire")
	}
}

func TestExitModeEnterVsExitZone(t *testing.T) {
	// ExitMode = enter overbought zone (default): up-cross through 70 sells.
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.rsiPrev, in.rsiNow = 65, 72 // up-cross through 70
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("enter-zone exit: want RSI sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
	// Down-cross through 70 must NOT sell in enter-zone mode.
	in = openInput()
	in.rsiPrev, in.rsiNow = 72, 65
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("enter-zone exit: down-cross should NOT sell")
	}

	// ExitMode = exit overbought zone: down-cross through 70 sells.
	p := defaultParams()
	p.ExitMode = triggerExitZone
	se := NewWithParams("TEST", p)
	in = openInput()
	in.rsiPrev, in.rsiNow = 72, 65 // down-cross through 70
	if sig := se.decide(in); sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("exit-zone exit: want RSI sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestStopSanityBlocksWhenNoStop(t *testing.T) {
	// ATRMult <= 0 -> no protective stop -> no entry.
	p := defaultParams()
	p.ATRMult = 0
	s := NewWithParams("TEST", p)
	if sig := s.decide(passingInput()); sig.Kind == model.SignalBuy {
		t.Fatalf("ATRMult=0: want no Buy (safety mandatory)")
	}
	// atr <= 0 -> cannot size a stop -> no entry.
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
	in.barLow = 97                 // SL hit (stop 98)
	in.rsiPrev, in.rsiNow = 65, 72 // RSI overbought too
	sig := s.decide(in)
	if sig.Reason != "SL" {
		t.Fatalf("protective first: want SL, got %q", sig.Reason)
	}
}

func TestExplainBlocksOnTrend(t *testing.T) {
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
		t.Fatalf("Explain should block on trend: %q", out)
	}
}

func TestExplainTrendOffSkipsGate(t *testing.T) {
	p := defaultParams()
	p.UseTrend = 0
	s := NewWithParams("TEST", p)
	md := strategy.MarketData{
		Price:   1,
		Highs:   []float64{1},
		Lows:    []float64{1},
		Closes:  []float64{1},
		Volumes: []int64{1},
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

- [ ] **Step 2: Запустить тесты — убедиться, что не компилируется/падает**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ 2>&1 | head -20`
Expected: FAIL компиляции (`triggerExitZone`/`entryFired`/`exitFired` не определены, старые поля `Params` отсутствуют).

- [ ] **Step 3: Переписать `core.go` под новый `Params` и логику**

Полностью заменить содержимое `core.go`:

```go
// Package core implements a long-only mean-reversion strategy driven purely by
// RSI on the daily timeframe. It buys when RSI crosses the oversold zone and exits
// when RSI crosses the overbought zone; the exact moment (entering vs exiting each
// zone) is configurable per side. An optional trend filter restricts buys to a
// confirmed uptrend. The protective stop is a daily-ATR stop frozen at entry. The
// decision logic is pure and ticker-agnostic; per-share packages supply ticker +
// Params. Run it with `-interval Day1`.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Trigger modes for EntryMode (oversold zone) and ExitMode (overbought zone).
// The semantics are shared: 0 fires when price enters the zone, 1 when it exits.
const (
	triggerEnterZone = 0 // RSI crosses into the zone
	triggerExitZone  = 1 // RSI crosses out of the zone
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	UseTrend      int     // 1 = require uptrend before buying; 0 = ignore trend
	FastEMA       int     // fast regime EMA (e.g. 50)
	SlowEMA       int     // slow regime EMA + price floor (e.g. 200)
	RSIPeriod     int     // RSI length; required (>0)
	RSIOversold   float64 // oversold zone boundary (entry side)
	RSIOverbought float64 // overbought zone boundary (exit side)
	EntryMode     int     // 0 = buy when RSI enters oversold; 1 = when it exits
	ExitMode      int     // 0 = sell when RSI enters overbought; 1 = when it exits
	ATRPeriod     int     // ATR length for the stop
	ATRMult       float64 // stop = entry - ATRMult*ATR; must be > 0
}

// Strategy trades a single instrument with the mean-reversion rules. Ticker-agnostic
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
		s.p.ATRPeriod + 1,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core.
type decideInput struct {
	price   float64
	atr     float64
	emaFast float64
	emaSlow float64
	rsiNow  float64
	rsiPrev float64
	barLow  float64
	pos     *strategy.Position
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

	var barLow float64
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	return decideInput{
		price:   md.Price,
		atr:     atr,
		emaFast: emaFast,
		emaSlow: emaSlow,
		rsiNow:  rsiNow,
		rsiPrev: rsiPrev,
		barLow:  barLow,
		pos:     md.Position,
	}
}

// crossUp reports an up-cross of level: prev at/below, now above.
func crossUp(prev, now, level float64) bool { return prev <= level && now > level }

// crossDown reports a down-cross of level: prev at/above, now below.
func crossDown(prev, now, level float64) bool { return prev >= level && now < level }

// entryFired reports whether the RSI entry trigger fires, honouring EntryMode.
// enter zone: RSI crosses DOWN through oversold. exit zone: crosses UP through it.
func (s *Strategy) entryFired(in decideInput) bool {
	if s.p.RSIPeriod <= 0 {
		return false
	}
	if s.p.EntryMode == triggerEnterZone {
		return crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold)
	}
	return crossUp(in.rsiPrev, in.rsiNow, s.p.RSIOversold)
}

// exitFired reports whether the RSI exit trigger fires, honouring ExitMode.
// enter zone: RSI crosses UP through overbought. exit zone: crosses DOWN through it.
func (s *Strategy) exitFired(in decideInput) bool {
	if s.p.RSIPeriod <= 0 {
		return false
	}
	if s.p.ExitMode == triggerEnterZone {
		return crossUp(in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
	}
	return crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
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
	// 2. RSI entry trigger.
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

// zoneWord renders the trigger mode for a zone in human terms.
func zoneWord(mode int) string {
	if mode == triggerEnterZone {
		return "вход в зону"
	}
	return "выход из зоны"
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(in decideInput, stop, risk float64) string {
	trend := "выкл"
	if s.p.UseTrend == 1 {
		trend = fmt.Sprintf("EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}
	return fmt.Sprintf(
		"Тренд: %s; RSI(%d) %s перепроданности %.0f (%.2f→%.2f); SL=%.4f (−%.2g×ATR %.4f, риск %.4f)",
		trend,
		s.p.RSIPeriod, zoneWord(s.p.EntryMode), s.p.RSIOversold, in.rsiPrev, in.rsiNow,
		stop, s.p.ATRMult, in.atr, risk,
	)
}

// manage handles an open long: the frozen ATR stop first, then the RSI exit trigger.
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
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: %.2f → %.2f, %s перекупленности %.0f", in.rsiPrev, in.rsiNow, zoneWord(s.p.ExitMode), s.p.RSIOverbought)
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

	// 2. RSI entry trigger.
	if !s.entryFired(in) {
		return block("RSI(%d): нет события (%s перепроданности %.0f), %.2f→%.2f",
			s.p.RSIPeriod, zoneWord(s.p.EntryMode), s.p.RSIOversold, in.rsiPrev, in.rsiNow)
	}
	pass("RSI(%d): %s перепроданности %.0f (%.2f→%.2f)", s.p.RSIPeriod, zoneWord(s.p.EntryMode), s.p.RSIOversold, in.rsiPrev, in.rsiNow)

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

- [ ] **Step 4: Запустить тесты ядра — должны проходить**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -v 2>&1 | tail -30`
Expected: PASS (все тесты `TestEntry*`, `TestTrend*`, `TestExit*`, `TestStop*`, `TestExplain*`).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/
git commit -m "feat(reversion): daily RSI core — trend toggle, dual zone triggers, ATR stop

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Привести 8 тикерных `DefaultParams` к новой форме

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/{afks,gazp,mdmg,nvtk,plzl,rusal,sber,ydex}/<ticker>.go`

- [ ] **Step 1: Обновить тело `DefaultParams` во всех 8 файлах**

В каждом файле заменить блок внутри `core.Params{...}` на единый baseline (имена пакетов/тикеров/комментарии шапки не трогать):

```go
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
		EntryMode: 1, ExitMode: 0,
		ATRPeriod: 14, ATRMult: 1.0,
	}
```

Файлы (значение `Ticker` в каждом своё, его не трогаем):
`afks/afks.go`, `gazp/gazp.go`, `mdmg/mdmg.go`, `nvtk/nvtk.go`, `plzl/plzl.go`, `rusal/rusal.go`, `sber/sber.go`, `ydex/ydex.go`.

- [ ] **Step 2: Собрать тикерные пакеты**

Run: `go build ./internal/service/trading_strategy/reversion/...`
Expected: успешная сборка, без ошибок про неизвестные поля.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/
git commit -m "feat(reversion): per-ticker defaults for daily-RSI params

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Обновить generic-дефолты и registry-тест

**Files:**
- Modify: `internal/service/backtest/reversion_registry.go:54-59` (тело `genericReversionDefaults`)
- Modify: `internal/service/backtest/reversion_registry_test.go`

- [ ] **Step 1: Заменить тело `genericReversionDefaults`**

В `reversion_registry.go` заменить возвращаемый `core.Params{...}` на:

```go
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
		EntryMode: 1, ExitMode: 0,
		ATRPeriod: 14, ATRMult: 1.0,
	}
```

- [ ] **Step 2: Обновить override-тест в `reversion_registry_test.go`**

Заменить `TestReversionLookupGenericFallback` (строки про `StopLossPct`) и `TestReversionDefaultsValid`:

В `TestReversionLookupGenericFallback` заменить блок ParseParams:

```go
	// ParseParams must layer the override on top of genericReversionDefaults.
	got, err := b.ParseParams([]byte(`{"ATRMult": 2.0}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.ATRMult != 2.0 {
		t.Fatalf("ATRMult=%v want 2.0 (override)", p.ATRMult)
	}
	if p.FastEMA != 50 || p.SlowEMA != 200 {
		t.Fatalf("generic defaults not preserved: FastEMA=%d SlowEMA=%d want 50/200", p.FastEMA, p.SlowEMA)
	}
```

Заменить `TestReversionDefaultsValid`:

```go
func TestReversionDefaultsValid(t *testing.T) {
	if p := genericReversionDefaults(); p.ATRMult <= 0 || p.SlowEMA <= p.FastEMA || p.RSIPeriod <= 0 {
		t.Fatalf("invalid generic defaults: %+v", p)
	}
}
```

- [ ] **Step 3: Прогнать тесты пакета backtest**

Run: `go test ./internal/service/backtest/ 2>&1 | tail -20`
Expected: PASS (включая `TestReversionLookup*`, `TestReversionDefaultsValid`).

- [ ] **Step 4: Полная сборка, vet и тесты**

Run: `go build ./... && go vet ./... && go test ./... 2>&1 | tail -30`
Expected: всё зелёное.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/
git commit -m "feat(reversion): generic daily-RSI defaults + registry test

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Новые сетки калибровки для 8 тикеров

**Files:**
- Modify: `data/params/{afks,gazp,mdmg,nvtk,plzl,rual,sber,ydex}/reversion_grid.json`

> Примечание про папки: тикер RUSAL хранит грид в папке `rual/` (известное расхождение `rusal`≠`rual`). Все 8 папок: `afks, gazp, mdmg, nvtk, plzl, rual, sber, ydex`.

- [ ] **Step 1: Записать единую сетку во все 8 файлов**

Каждый файл `data/params/<ticker>/reversion_grid.json` получает идентичное содержимое:

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
        "RSIPeriod": [4, 5, 10, 12, 14],
        "RSIOversold": [15, 20, 35],
        "EntryMode": [0, 1]
      }
    },
    {
      "name": "exit",
      "keepTop": 5,
      "grid": {
        "RSIOverbought": [65, 70, 85],
        "ExitMode": [0, 1],
        "ATRMult": [1.0, 1.5, 2.0],
        "ATRPeriod": [14]
      }
    }
  ]
}
```

- [ ] **Step 2: Проверить валидность JSON**

Run: `for f in data/params/{afks,gazp,mdmg,nvtk,plzl,rual,sber,ydex}/reversion_grid.json; do python3 -m json.tool "$f" >/dev/null && echo "ok $f" || echo "BAD $f"; done`
Expected: `ok` для всех 8 файлов.

- [ ] **Step 3: Дымовой прогон калибровки на одном тикере (короткий период)**

Run: `go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 -calibrate data/params/sber/reversion_grid.json -out ./reports/SBER -months 24 -min-trades 5 -test-months 6 -metric profit_factor 2>&1 | tail -25`
Expected: калибровка отрабатывает обе фазы (`entry`, `exit`) и пишет `*_best.md`/`*_calibration.md` в `./reports/SBER` без паник. (Требуется валидный токен в `env/local.env`; если сети/токена нет — зафиксировать это и пропустить прогон, не считая задачу проваленной.)

- [ ] **Step 4: Commit**

```bash
git add data/params/*/reversion_grid.json
git commit -m "feat(reversion): daily-RSI calibration grids for all 8 tickers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Обновить документацию стратегии

**Files:**
- Modify: `docs/reversion/strategy.md`

- [ ] **Step 1: Прочитать текущий `docs/reversion/strategy.md`**

Run: `cat docs/reversion/strategy.md`
Цель: понять структуру, чтобы переписать под новую модель (дневной ТФ, чистый RSI, тренд-тумблер, ATR-стоп, режимы вход/выход из зоны), убрав упоминания volume-gate/Stochastic-гейта/time-stop как активных фильтров.

- [ ] **Step 2: Переписать объяснение под новую логику**

Привести разделы в соответствие со спекой `docs/superpowers/specs/2026-06-12-reversion-rsi-daily-rework-design.md`: параметры `UseTrend/RSIPeriod/RSIOversold/RSIOverbought/EntryMode/ExitMode/ATRMult/ATRPeriod`, семантика «вход/выход из зоны», ATR-стоп, запуск `-interval Day1`. Явно отметить, что Stochastic в этой стратегии больше не используется (индикатор остаётся в `pkg/indicators`).

- [ ] **Step 3: Commit**

```bash
git add docs/reversion/strategy.md
git commit -m "docs(reversion): rewrite explainer for daily-RSI strategy

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review (выполнено при написании плана)

- **Покрытие спеки:** Params (Task 1), семантика триггеров вход/выход (Task 1 тесты `TestEntryMode*`/`TestExitMode*`), тренд-тумблер (Task 1 `TestTrendFilterToggles`), ATR-стоп + санити (Task 1 `TestStop*`), приоритет защиты (Task 1 `TestProtectiveStopWinsTie`), дефолты тикеров (Task 2), generic + registry (Task 3), сетки (Task 4), docs (Task 5). Удаление volume/stoch/time-stop отражено в переписанном `core.go` (Task 1) и обновлённых дефолтах.
- **Плейсхолдеры:** нет TBD/TODO; весь код приведён целиком.
- **Согласованность типов:** имена `triggerEnterZone`/`triggerExitZone`, `entryFired`/`exitFired`, поля `UseTrend/EntryMode/ExitMode/ATRMult/ATRPeriod` единообразны между `core.go`, тестами, дефолтами и JSON-сетками. Тесты используют только новые поля; старые (`StopLossPct`, `MaxHoldBars`, `UseStoch`, `Vol*`) удалены везде, где грепались.
