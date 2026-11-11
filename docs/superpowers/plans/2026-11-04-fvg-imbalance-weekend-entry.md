# FVG Imbalance Weekend Entry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent `fvg_imbalance` from opening a new position when its next-M30 `FillBy` is on Saturday or Sunday.

**Architecture:** Keep the rule in the stateless FVG core, immediately before it returns a buy signal. The existing position-management branch remains before this guard and is unchanged. Tests set deterministic candle timestamps so they prove both rejection of a weekend fill and acceptance of the equivalent weekday fill.

**Tech Stack:** Go 1.25; existing scalping strategy interfaces; Go standard-library `time`.

**Spec:** `docs/superpowers/specs/2026-11-04-fvg-imbalance-weekend-entry-design.md`

## Global Constraints

- Determine the prohibited day from the exact `FillBy` timestamp, not the retest candle timestamp.
- Saturday and Sunday suppress only new buy signals; do not defer them to Monday.
- SL/TP handling for an existing position must remain unaffected.
- Preserve the existing long-only M30 behavior, parameters, and fill timestamp.

## Review Focus

- Friday retest with a Saturday `FillBy` returns `SignalNone`; pinned in Task 1.
- Identical formation and retest with a weekday `FillBy` returns `SignalBuy`; pinned in Task 1.
- Sunday `FillBy` is also rejected; pinned in Task 1.
- A position on a weekend timestamp still produces SL/TP according to existing logic; pinned in Task 1.
- A Monday `FillBy` is not accidentally treated as a weekend; pinned in Task 1.

### Task 1: Guard new entries by `FillBy` weekday

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`
- Modify: `docs/fvg_imbalance/strategy.md`

**Interfaces:**
- Consumes: `Strategy.Decide(md strategy.MarketData) model.Signal` and its current `FillBy = md.Times[k].Add(s.step)` contract.
- Produces: the same buy signal on working days, or the zero-value signal when that fill timestamp is Saturday or Sunday.

- [ ] **Step 1: Write failing weekday tests**

Add table-driven cases that use the existing formation and protected-retest fixtures, override their timestamps, and assert:

```go
{"Saturday fill", friday2300, model.SignalNone}
{"Sunday fill", saturday2300, model.SignalNone}
{"Monday fill", sunday2300, model.SignalBuy}
```

Add a weekend-timestamp open-position SL case that still asserts `model.SignalSell`.

- [ ] **Step 2: Run the focused test to verify it fails**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'TestWeekendFill|TestPosition' -count=1`

Expected: weekend buy cases fail because the current implementation emits `SignalBuy`.

- [ ] **Step 3: Add the minimal weekday guard in `Strategy.Decide`**

Calculate the already-defined next-open `FillBy`; return no buy signal when its `Weekday()` is `time.Saturday` or `time.Sunday`. Keep this code after position handling and before populating the buy signal.

- [ ] **Step 4: Run focused tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'TestWeekendFill|TestPosition' -count=1`

Expected: PASS.

- [ ] **Step 5: Document the execution rule**

In `docs/fvg_imbalance/strategy.md`, state that a new entry whose `FillBy` is Saturday or Sunday is discarded and is not carried into Monday; exits are unaffected.

- [ ] **Step 6: Run scoped verification and commit**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/... -count=1`

Expected: PASS.

```bash
git add internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go \
  docs/fvg_imbalance/strategy.md
git commit -m "fix: skip FVG entries with weekend fills"
```
