# FVG Imbalance Quality Filters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional H4 trend, liquidity-sweep, bullish-BOS, first-retest, entry-distance, buffered-stop, and time-exit controls to the stateless M30 FVG strategy while preserving its disabled-control baseline.

**Architecture:** Keep FVG and market-structure semantics inside the pure `fvg_imbalance/strategy/core` package. Extend shared execution metadata only with a zero-compatible maximum entry price and the portfolio's already-known bars-held count; reuse the existing completed-HTF cursor for H4 data.

**Tech Stack:** Go 1.25, existing scalping strategy interfaces, existing backtest engine/calibrator, Wilder ATR and EMA indicator packages, table-driven Go tests.

**Spec:** `docs/superpowers/specs/2026-11-04-fvg-imbalance-quality-filters-design.md`

## Global Constraints

- Follow TDD: add one focused failing test, run it red for the intended reason, implement only enough to pass, then rerun the owning package.
- Preserve the user's current GAZP baseline: `ATRPeriod=14`, `MinGapATR=0.8`, `MaxRetestBars=8`, `RequireCloseAboveZone=0`, `TakeProfitR=1.5`.
- All new controls default to disabled; partial parameter JSON and all existing strategies remain behavior-compatible.
- Use completed candles only; no helper may infer an unconfirmed fractal or still-forming H4 candle.
- Keep the strategy stateless, M30-only, backtest-only, and long-only.
- Do not modify or stage unrelated user changes, including the existing `pdl_rsi_m30` removals and `cmd/backtest` CPU work.
- Do not promote calibration winners into `gazp.Params()` in this implementation.

## Review Focus

- A fractal visible only because of future right-hand bars must not qualify early; Task 2 pins event-time confirmation.
- A prior FVG touch that failed another gate or never filled must still consume a first-retest-only zone; Task 3 pins consumption before all other entry gates.
- A pending next-open exactly at `MaxEntryPrice` must fill while one tick above must be dropped without affecting zero-valued signals; Task 1 pins all boundaries.
- On the time-exit threshold bar, SL then TP must beat `TIME`; Task 4 pins exit priority and exact `BarsHeld` semantics.
- An enabled H4 gate must not see an unfinished H4 candle and must fail closed on short data; Tasks 3 and 5 pin both core and engine/CLI behavior.

---

### Task 1: Add zero-compatible execution metadata

**Files:**
- Modify: `internal/service/trading_strategy/scalping/model/signal.go:14-33`
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go:9-37`
- Modify: `internal/domain/backtest/portfolio.go:135-149`
- Modify: `internal/domain/backtest/engine.go:288-299`
- Modify: `internal/domain/backtest/entry_next_open_test.go:96-135`
- Modify: `internal/domain/backtest/portfolio_test.go:79-115`

**Interfaces:**
- Produces: `model.Signal.MaxEntryPrice float64`, where zero disables the limit.
- Produces: `strategy.Position.BarsHeld int`, populated as `portfolio.bar-portfolio.entryBar`.
- Changes: `fillPending(*portfolio, *model.Signal, Candle) *model.Signal` drops a pending buy when `MaxEntryPrice>0 && c.Open>MaxEntryPrice`.

- [ ] **Step 1: Write the failing pending-entry boundary test**

Extend `TestEntryAtNextOpenMaxEntryPrice` as a table with `MaxEntryPrice=0`, open below, open equal, and open above. Assert that zero/below/equal open positions and above produces no trade or exposure.

- [ ] **Step 2: Run the entry test and verify RED**

Run: `go test ./internal/domain/backtest -run TestEntryAtNextOpenMaxEntryPrice -count=1`

Expected: FAIL because `model.Signal` has no `MaxEntryPrice` field.

- [ ] **Step 3: Add `Signal.MaxEntryPrice` and enforce it in `fillPending`**

Document zero as unlimited and equality as fillable. Keep `FillBy` and already-open-position handling unchanged.

- [ ] **Step 4: Run the entry test and verify GREEN**

Run: `go test ./internal/domain/backtest -run TestEntryAtNextOpenMaxEntryPrice -count=1`

Expected: PASS.

- [ ] **Step 5: Write the failing position-age test**

Add `TestPortfolioStrategyPositionReportsBarsHeld`: open at `p.bar=3`, assert `BarsHeld=0`, advance to bars 4 and 8, and assert `1` and `5` without changing entry state.

- [ ] **Step 6: Run the age test and verify RED**

Run: `go test ./internal/domain/backtest -run TestPortfolioStrategyPositionReportsBarsHeld -count=1`

Expected: FAIL because `strategy.Position` has no `BarsHeld` field.

- [ ] **Step 7: Add `Position.BarsHeld` and populate it from the portfolio**

The zero value remains valid for live builders. Do not add a second counter to `portfolio`.

- [ ] **Step 8: Run the owning package**

Run: `go test ./internal/domain/backtest -count=1`

Expected: PASS with all pre-existing next-open and portfolio tests unchanged.

- [ ] **Step 9: Commit Task 1**

```bash
git add internal/service/trading_strategy/scalping/model/signal.go \
  internal/service/trading_strategy/scalping/strategy/strategy.go \
  internal/domain/backtest/portfolio.go internal/domain/backtest/engine.go \
  internal/domain/backtest/entry_next_open_test.go internal/domain/backtest/portfolio_test.go
git commit -m "feat: add bounded deferred entries and position age"
```

### Task 2: Add FVG parameters and confirmed structure primitives

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go:15-96`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go:35-148`

**Interfaces:**
- Extends: `core.Params` with the ten fields and exact defaults from the spec.
- Produces: `swingHighAt(highs []float64, i, k int) bool` and `swingLowAt(lows []float64, i, k int) bool`.
- Produces: `(*Strategy).structurePasses(md strategy.MarketData, z zone) bool`, enforcing standalone sweep, standalone BOS, and ordered sweep+BOS rules.
- Changes: `Lookback()` uses the legacy formula when BOS and sweep are both disabled; otherwise its formation-history term is `max(3, 2*ATRPeriod, StructureLookbackBars+2*SwingK+3)` before adding `MaxRetestBars`.

- [ ] **Step 1: Extend validation tests first**

Add default assertions and invalid-domain cases for each `0/1` toggle, `SwingK`, `StructureLookbackBars`, `MaxSweepToFVGBars`, `HTFTrendEMA`, `MaxEntryDistanceATR`, `MaxHoldBars`, and `StopBufferATR`, including NaN/Inf float cases.

- [ ] **Step 2: Run validation tests and verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test(DefaultParams|ValidateParams)' -count=1`

Expected: FAIL because the new fields/defaults do not exist.

- [ ] **Step 3: Add parameter fields, defaults, and validation**

Use the exact names and domains in the spec. Keep every validation error lowercase after the field name and preserve existing validation behavior.

- [ ] **Step 4: Run validation tests and verify GREEN**

Run the Step 2 command; expect PASS.

- [ ] **Step 5: Write failing strict-fractal and lookback tests**

Cover strict highs/lows, equal-neighbor rejection, insufficient left/right bars, a pivot unavailable before `i+K`, legacy lookback with both structure gates off, and the exact enlarged formula with either gate on.

- [ ] **Step 6: Run fractal/lookback tests and verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test(Swing|Lookback)' -count=1`

Expected: FAIL because the primitives and conditional lookback are missing.

- [ ] **Step 7: Implement fractal primitives and conditional lookback**

Keep helpers pure and bounds-safe. Equal values never form a pivot.

- [ ] **Step 8: Write failing structure-sequence tests**

Use completed synthetic M30 bars to cover: sweep wick without reclaim, equality boundaries, valid reclaim, expired sweep, wick-only BOS rejection, close BOS acceptance, standalone sweep, standalone BOS, BOS before sweep rejection, sweep and BOS on one candle, and a pivot whose right confirmation occurs after the event.

- [ ] **Step 9: Run structure tests and verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test(LiquiditySweep|BullishBOS|Structure)' -count=1`

Expected: FAIL because `structurePasses` is missing or permissive.

- [ ] **Step 10: Implement `structurePasses` minimally**

Search only within `StructureLookbackBars`; choose the newest qualifying sweep and most recent event-time-usable reference pivots exactly as the spec defines.

- [ ] **Step 11: Run the core package**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1`

Expected: PASS without changing entry behavior yet because new controls default off.

- [ ] **Step 12: Commit Task 2**

```bash
git add internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go
git commit -m "feat: model FVG structure context"
```

### Task 3: Gate entries and compute bounded risk levels

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go:98-153`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go:153-276`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go:27-86`

**Interfaces:**
- Produces: `(*Strategy).htfTrendOK(md strategy.MarketData) (ok bool, close, value float64)` using completed H4 closes supplied by the engine.
- Changes: buy signals optionally set `MaxEntryPrice=upper+MaxEntryDistanceATR*formationATR`.
- Changes: buy stops use `lower-StopBufferATR*formationATR`; provisional and position targets use the buffered stop.
- Changes: `RequireFirstRetest=1` rejects a zone after any earlier post-formation `low<=upper`.

- [ ] **Step 1: Write failing first-retest tests**

Add cases for first touch, second protected touch, earlier touch that failed close confirmation, earlier weekend/missing-slot first touch, and disabled compatibility. Assert a consumed newer zone does not prevent considering an older still-unconsumed qualifying zone.

- [ ] **Step 2: Run first-retest tests and verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestFirstRetest -count=1`

Expected: FAIL because prior protected touches are currently reusable.

- [ ] **Step 3: Enforce first-touch consumption before other entry gates**

Keep the existing `low<=lower` permanent invalidation and newest-qualifying-zone ordering.

- [ ] **Step 4: Write failing structure-gate integration tests**

Drive `Decide` through the valid/invalid standalone and combined contexts from Task 2; assert toggles off reproduce the existing signal.

- [ ] **Step 5: Run and verify RED, then wire `structurePasses` into entry selection**

Run before and after implementation:
`go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestEntryStructureGates -count=1`

Expected sequence: FAIL, then PASS.

- [ ] **Step 6: Write failing H4 gate tests**

Cover disabled/nil bypass, enabled nil/short/NaN fail-closed, equality/downtrend block, and strict uptrend pass. Assert `EntryReason` mentions the enabled H4 verdict.

- [ ] **Step 7: Run H4 tests and verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestHTFTrend -count=1`

Expected: FAIL because no H4 gate exists.

- [ ] **Step 8: Implement `htfTrendOK` and gate only new entries**

Use `pkg/indicators/ema.Compute`; do not consult H4 data when `HTFTrendEMA=0`, and never gate position exits.

- [ ] **Step 9: Write failing entry-distance and buffered-stop tests**

Assert zero leaves `MaxEntryPrice=0`; positive distance uses frozen formation ATR; buffered stop is below `lower`; provisional target and actual-fill target both use the buffered stop; existing gap-through-stop behavior remains.

- [ ] **Step 10: Run risk-level tests and verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test(EntryDistance|StopBuffer|EngineBuffered)' -count=1`

Expected: FAIL on missing fields/old levels.

- [ ] **Step 11: Emit maximum entry price and buffered levels**

Expand `EntryReason` with enabled gates, buffered stop, and optional maximum entry price without changing disabled-control wording required by existing tests.

- [ ] **Step 12: Run the core package**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1`

Expected: PASS.

- [ ] **Step 13: Commit Task 3**

```bash
git add internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go
git commit -m "feat: gate and bound FVG entries"
```

### Task 4: Add bar-count time exits with conservative priority

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go:105-121`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go:277-305`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go:42-86`

**Interfaces:**
- Changes: an open position emits `SignalSell`, `Reason="TIME"`, and a descriptive `ExitReason` when `MaxHoldBars>0 && Position.BarsHeld>=MaxHoldBars`, after SL and TP checks.

- [ ] **Step 1: Write failing unit tests for time-exit boundaries and priority**

Cover `MaxHoldBars=0`, `BarsHeld=limit-1`, equality, greater-than, threshold-bar SL, and threshold-bar TP. Assert SL > TP > TIME ordering and no dependency on H4/structure gates.

- [ ] **Step 2: Run time-exit tests and verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestPositionTimeExit -count=1`

Expected: FAIL because no `TIME` signal exists.

- [ ] **Step 3: Add the time exit after price exits**

Do not add `TIME` to `model.IsStopReason`; generic engine pricing must therefore use the current bar close.

- [ ] **Step 4: Write and run an engine integration test**

Add a next-open trade whose position reaches the exact engine bar count without SL/TP. Assert report `BarsHeld`, close price, `Reason="TIME"`, and fill/exit timestamps.

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestEngineTimeExit -count=1`

Expected sequence: FAIL before integration wiring, then PASS.

- [ ] **Step 5: Run the core and domain packages**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core ./internal/domain/backtest -count=1`

Expected: PASS. The already-known stale GAZP literal test is deliberately outside this command and is fixed in Task 5 before the full suite.

- [ ] **Step 6: Commit Task 4**

```bash
git add internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go
git commit -m "feat: add FVG time exits"
```

### Task 5: Wire H4 data, literals, validation, and staged calibration

**Files:**
- Modify: `cmd/backtest/main.go:195-235`
- Modify: `cmd/backtest/main_test.go:171-205`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go:6-15`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp_test.go:9-13`
- Modify: `internal/service/backtest/fvg_imbalance_registry_test.go:41-98`
- Create: `internal/service/backtest/fvg_imbalance_grid_test.go`
- Modify: `data/params/fvg_imbalance/gazp/grid.json`

**Interfaces:**
- Produces: `htfSpec(strategyName string) (interval enum.Interval, span time.Duration, ok bool)` in `cmd/backtest`, returning H4/4h for `fvg_imbalance`, existing H4 for `reversion`, existing H1 for `scalping_rsimacd`, and `ok=false` otherwise.
- Changes: GAZP literal explicitly lists every new baseline field while preserving `MinGapATR=0.8` and `RequireCloseAboveZone=0`.
- Produces: the exact four-phase grid from the spec.

- [ ] **Step 1: Write the failing HTF-routing test**

Table-test `reversion`, `scalping_rsimacd`, `fvg_imbalance`, and a no-HTF strategy. Assert interval, duration, and enabled flag.

- [ ] **Step 2: Run routing test and verify RED**

Run: `go test ./cmd/backtest -run TestHTFSpec -count=1`

Expected: FAIL because `htfSpec` does not exist.

- [ ] **Step 3: Extract `htfSpec` and route FVG to H4 loading**

Preserve existing coverage reporting for `scalping_rsimacd`; add equivalent fail-loud coverage reporting for enabled FVG H4 data without relabeling it H1.

- [ ] **Step 4: Add a failing completed-H4 integration test**

Use the existing engine HTF cursor with an FVG setup and two H4 candles: one completed downtrend close and one still-forming uptrend close. Assert the forming candle is invisible and entry remains blocked; move decision time past completion and assert the completed series can pass.

- [ ] **Step 5: Run integration test RED, implement only required routing/alignment, then GREEN**

Run: `go test ./cmd/backtest ./internal/domain/backtest -run 'Test(FVG.*HTF|HTFSpec)' -count=1`

Expected sequence: FAIL, then PASS.

- [ ] **Step 6: Update literal/registry tests before production literals**

Assert the exact GAZP baseline, all partial JSON overlays, every new invalid domain, and generic defaults. Run:
`go test ./internal/service/trading_strategy/fvg_imbalance/strategy/gazp ./internal/service/backtest -run 'Test(GAZP|FVGImbalance)' -count=1`

Expected: FAIL first on stale/missing fields, then PASS after updating `gazp.Params()` and validation plumbing.

- [ ] **Step 7: Write the failing staged-grid contract test**

Read `data/params/fvg_imbalance/gazp/grid.json`; assert four phase names, every `keepTop=8`, exact candidate arrays, phase expansion counts `72/32/16/5`, fixed structural helper values, and inclusion of all disabled baselines.

- [ ] **Step 8: Run grid test and verify RED**

Run: `go test ./internal/service/backtest -run TestFVGImbalanceGrid -count=1`

Expected: FAIL against the current one-phase grid.

- [ ] **Step 9: Replace the grid with the four exact phases**

Pin `SwingK=[2]`, `StructureLookbackBars=[48]`, and `MaxSweepToFVGBars=[8]` in the context phase so survivors carry explicit mechanics.

- [ ] **Step 10: Run CLI, registry, grid, and FVG tests**

Run: `jq empty data/params/fvg_imbalance/gazp/grid.json && go test ./cmd/backtest ./internal/service/backtest ./internal/service/trading_strategy/fvg_imbalance/... -count=1`

Expected: PASS.

- [ ] **Step 11: Commit Task 5**

```bash
git add -p cmd/backtest/main.go
git add cmd/backtest/main_test.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp_test.go \
  internal/service/backtest/fvg_imbalance_registry_test.go \
  internal/service/backtest/fvg_imbalance_grid_test.go \
  data/params/fvg_imbalance/gazp/grid.json
git commit -m "feat: calibrate FVG quality filters"
```

For `cmd/backtest/main.go`, stage only the new HTF-routing hunks. Leave the
pre-existing CPU flag and `pdl_rsi_m30` cleanup hunks unstaged. The approved
GAZP `0.8/0` baseline is part of this task and may be included with the new
literal fields because the design explicitly adopts it.

### Task 6: Document mechanics, verify the repository, and collect evidence

**Files:**
- Modify: `docs/fvg_imbalance/strategy.md`
- Generated/ignored evidence: `reports/GAZP/fvg_imbalance/r20261104/`

**Interfaces:**
- Documents: exact filter ordering, parameter table, H4 completion/fail-closed behavior, first-touch consumption, bounded next-open, buffered stop, time-exit priority, and the approved commands.
- Produces: refreshed baseline and primary 24/12/3 walk-forward reports without changing defaults.

- [ ] **Step 1: Update mechanics documentation**

Reconcile the GAZP baseline to `0.8/0`; add no per-ticker results. Include the exact primary walk-forward command from the spec and explain that `risk-pct=1` normalizes stop-width comparisons.

- [ ] **Step 2: Run formatting and focused verification**

Run:

```bash
gofmt -w internal/service/trading_strategy/fvg_imbalance/strategy/core/*.go \
  internal/service/trading_strategy/fvg_imbalance/strategy/gazp/*.go \
  internal/service/trading_strategy/scalping/model/signal.go \
  internal/service/trading_strategy/scalping/strategy/strategy.go \
  internal/domain/backtest/*.go internal/service/backtest/fvg_imbalance_registry_test.go \
  internal/service/backtest/fvg_imbalance_grid_test.go cmd/backtest/*.go
go test ./internal/service/trading_strategy/fvg_imbalance/... \
  ./internal/domain/backtest ./internal/service/backtest ./cmd/backtest -count=1
```

Expected: PASS.

- [ ] **Step 3: Run the repository gate**

Run: `./bin/mage ci`

Expected: lint, `go test -race ./...`, and mock-drift all PASS. If `./bin/mage` is absent, run `./bin/mage tools` first as documented by the project.

- [ ] **Step 4: Run the supported build and whitespace check**

Run: `go build ./internal/... ./pkg/... ./cmd/... && git diff --check`

Expected: PASS.

- [ ] **Step 5: Commit documentation**

```bash
git add docs/fvg_imbalance/strategy.md
git commit -m "docs: describe FVG quality controls"
```

- [ ] **Step 6: Run the refreshed baseline evidence command**

Run:

```bash
go run ./cmd/backtest \
  -ticker GAZP -strategy fvg_imbalance -interval Minutes30 \
  -out reports/GAZP/fvg_imbalance/r20261104/baseline_full \
  -months 24 -risk-pct 1 -commission 0.0005 -refresh
```

Record the actual common M30/H4 cutoff.

- [ ] **Step 7: Run the primary staged walk-forward command**

Run the exact 24/12/3 command in the spec, outputting to `reports/GAZP/fvg_imbalance/r20261104/quality_wf`. Do not promote winner parameters.

- [ ] **Step 8: Review evidence without changing code**

Report pooled OOS PF/trades/PnL/drawdown, per-fold PF/trades, winner stability, any phase where all train combinations missed 20 trades, baseline comparison, and actual data coverage. Explicitly label the result inconclusive if H4 refresh cannot reach the required cutoff or the pooled sample is inadequate.

- [ ] **Step 9: Final status check**

Run: `git status --short && git log -6 --oneline`

Expected: only pre-existing user changes and ignored/generated reports remain; implementation commits are visible and no unrelated files were staged.
