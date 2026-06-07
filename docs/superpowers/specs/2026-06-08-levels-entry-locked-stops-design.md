# Levels: Entry-Locked Protective Stops — Design

**Date:** 2026-06-08
**Branch:** `feat/levels-volume-profile-strategy`
**Status:** Approved (design), pending spec review

## Problem

The levels strategy holds losing positions for months (up to 5878 bars ≈ 10
months) and never exits on a hard stop. Across every backtest run the trade
journal shows **zero SL exits** — all exits are `TRAIL`. Max drawdown reaches
33–34 % with exposure 58–65 %.

### Root cause (proven, not hypothesised)

Two independent defects in `internal/service/trading_strategy/levels/strategy/core/core.go`,
both stemming from recomputing protective levels every bar from a window that
includes the current bar.

**Defect A — the hard stop can never fire.**

In the position-management branch:

```go
hardSL := in.recentLow - s.p.SLMult*in.atr   // core.go
...
case in.barLow <= hardSL:                     // SL exit condition
```

`in.recentLow = recentLow(md.Lows, SwingLowWindow)` is the minimum low over a
window that **includes the current bar** (the engine builds the window as
`candles[i-l+1 : i+1]` in `engine.go:55`, so the current bar's low is the last
element, and `recentLow` scans it). Therefore the current bar's low `barLow` is
itself a member of that window, so `recentLow ≤ barLow` always. With `SLMult > 0`
and `ATR > 0`:

```
hardSL = recentLow − SLMult·ATR  <  barLow      (always)
```

So `barLow <= hardSL` is **never true**. The hard stop is dead code — this is
why every report shows zero SL exits.

Concrete check (trade #10 in `reports/level_new2/...001100_best.md`):
`SwingLowWindow=5, SLMult=1, ATR=0.2326`. For the stop to fire the current low
must be ≥ 0.23 below the minimum of the last 5 lows *including itself* —
impossible.

**Defect B — the chandelier trail disarms on a pullback.**

```go
armed := s.p.TrailArmATR <= 0 || in.price >= entry+s.p.TrailArmATR*in.atr
```

`armed` is recomputed every bar and is **non-monotonic**: once price falls back
below `entry + TrailArmATR·ATR`, the trail switches off. During a drawdown — when
protection is most needed — the trail disarms, and since the hard stop is dead
(Defect A), the position has no working protective exit at all.

Together these explain the observed behaviour: zero SL exits, multi-month holds,
33–34 % drawdown, 58–65 % exposure. The attractive PF/win-rate are a side effect
of never cutting losers — they are pressed-and-held instead.

## Goal

Make both protective stops actually protect: a hard stop that fires and bounds
downside, and a trail that stays armed once triggered. Fix the backtest path so
calibration reflects a strategy with working stops.

## Scope

**In scope (this iteration):** correctness in the backtest path. The fix anchors
both mechanisms at entry and carries them through `Position`, populated by the
backtest engine/portfolio (which already remembers entry-time state).

**Out of scope (deliberately deferred):**

- **Live wiring.** See the Known Gap section. The levels strategy is not deployed
  live yet (unmerged branch, calibration phase). Live persistence of entry state
  is a separate iteration.
- **Chandelier ratchet** (trail never moving down). With a frozen hard stop now
  bounding downside, a loosening chandelier is bounded from below, so a ratchet is
  YAGNI here.

## Approach

Move both protective mechanisms from "recompute every bar from a self-including
window" to "fixed at entry, carried through `Position`."

### Approaches considered

- **Stateless, live-reconstructable** (`hardSL = entry − SLMult·ATR`, arm from
  `max(highs in window)`): works live with no new state, but the stop is too
  tight (the original entry−ATR stop that bled −17.65 % via intrabar stop-hunts),
  it still drifts with ATR, and arming is not truly monotonic over the whole
  trade. Rejected.
- **Full entry-state persistence** shared by backtest and live: most correct and
  portable, but a new subsystem (per-position store). Rejected for this iteration
  as out of scope (levels not live yet).
- **Entry-anchored state in the backtest engine/portfolio** (chosen): the engine
  already remembers `entryPrice/entryLevel/entryATR`; extend it with the frozen
  stop and a running favourable-price maximum, and carry them through `Position`.
  Correct in backtest now; live deferred.

For the arm latch specifically, storing a running `MaxFavorablePrice` in the
portfolio is preferred over storing a boolean `armed` latch: the portfolio then
tracks a dumb running maximum and holds no strategy knowledge, while the strategy
keeps ownership of the arm rule (`>= entry + TrailArmATR·ATR`). A boolean latch
would push `TrailArmATR` into the engine — unwanted coupling.

## Components and data flow

### 1. `Position` contract (`scalping/strategy/strategy.go`)

Add an entry-context bundle:

```go
type Position struct {
    PurchasePrice     float64
    Quantity          int64
    StopLoss          float64 // hard SL frozen at entry
    EntryATR          float64 // ATR at entry (arm threshold)
    MaxFavorablePrice float64 // max close since entry (monotonic non-decreasing)
}
```

Scalping does not set the new fields → they are zero and its path is unaffected
(same convention as the existing `Signal.RSI` that levels leaves at zero).

### 2. Portfolio (`domain/backtest/portfolio.go`)

- New fields: `entryStop float64`, `maxFavorable float64`.
- `open(price, t, level, target, atr, stop float64)` — additional `stop`
  parameter; sets `entryStop = stop`, `maxFavorable = price`.
- New method `mark(price float64)`: `if p.qty != 0 && price > p.maxFavorable { p.maxFavorable = price }`.
- `close()` resets `entryStop = 0` and `maxFavorable = 0` alongside the existing
  resets.
- `strategyPosition()` returns the new fields:
  `StopLoss: p.entryStop, EntryATR: p.entryATR, MaxFavorablePrice: p.maxFavorable`.

### 3. Engine (`domain/backtest/engine.go`)

In the per-bar loop, while in a position, update the running maximum with the
current bar's close **before** building `Position`, so arming can occur on the
current bar:

```go
c := candles[i]
if p.qty != 0 {
    p.mark(c.Close)
}
md.Position = p.strategyPosition()
```

The Buy branch passes the signal's stop into `open`:

```go
p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR, sig.StopLoss)
```

### 4. Core decision (`levels/strategy/core/core.go`) — management branch only

The entry branch is unchanged: it still computes
`stop := recentLow − SLMult·ATR` as the structural stop at the moment of entry
and emits it via `sig.StopLoss`; the engine freezes it.

The management branch stops recomputing from the window and reads the frozen
entry context:

```go
if in.pos != nil {
    entry      := in.pos.PurchasePrice
    hardSL     := in.pos.StopLoss                        // frozen at entry
    chandelier := in.recentHigh - s.p.TrailMult*in.atr   // trail still moves up
    armed      := s.p.TrailArmATR <= 0 ||
                  in.pos.MaxFavorablePrice >= entry+s.p.TrailArmATR*in.pos.EntryATR
    sig.StopLoss = hardSL
    switch {
    case in.barLow <= hardSL:
        sig.Kind, sig.Reason = model.SignalSell, "SL"
    case armed && in.barLow <= chandelier:
        sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
    }
    return sig
}
```

`in.recentLow` is still computed in `Decide` and consumed by the entry branch;
the management branch no longer uses it.

### Why this fixes both defects

- **Hard SL frozen at entry** does not slide down with price, so `barLow <= hardSL`
  is reachable → the stop fires and bounds downside. The level is still the
  entry-time swing-low stop (below structure), so it is not so tight as to be
  hunted on intrabar noise.
- **`armed` via monotonic `MaxFavorablePrice`** latches: once price has reached
  `entry + TrailArmATR·EntryATR` the trail stays armed even if price later falls
  back, so it can no longer disarm during a pullback.

## Known gap: live trading (deferred, explicit)

In live trading (`scalping/trade.go`), `Position` is built from the broker
portfolio with only `PurchasePrice` and `Quantity`; the broker does not store
entry ATR, the entry stop, or the favourable-price maximum. With the new fields
left zero, a levels strategy run live would have `StopLoss = 0` (hard stop never
fires) and `MaxFavorablePrice = 0` (never arms) — both protective exits disabled.

This is **deliberately deferred**, not overlooked: levels is not deployed live in
this iteration. Closing the gap requires persisting per-position entry state
across runs (the live runner is stateless today). This must be done before levels
goes live. The code will carry a clear precondition comment so the gap cannot
surface silently at deploy time.

## Testing

**Core (`core_test.go`):**

- Hard SL fires when `barLow <= in.pos.StopLoss` (using the frozen level).
- Hard SL does not slide: with the same `in.pos.StopLoss` but a higher
  `in.recentLow`, the SL level used is still the frozen one.
- Arm is monotonic: with `MaxFavorablePrice` already past the threshold but
  current `price` back below it, `armed` is still true (trail can fire).
- Existing management-branch tests updated to populate the new `Position` fields.

**Portfolio (`portfolio_test.go`):**

- `open` records `entryStop` and initialises `maxFavorable` to the entry price.
- `mark` raises `maxFavorable` only upward (monotonic), no-op when flat.
- `close` resets `entryStop` and `maxFavorable`.

**Engine (`engine_test.go`):**

- Integration: a position that drifts down now exits via `SL` rather than being
  held to the end of data.

All existing tests must remain green (`go test ./...`).
