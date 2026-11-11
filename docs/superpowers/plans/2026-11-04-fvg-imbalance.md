# FVG Imbalance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an independently calibrated, long-only M30 bullish-FVG backtest strategy with a GAZP literal and grid.

**Architecture:** `fvg_imbalance/strategy/core` will implement a pure `scalping/strategy.Strategy`, reconstructing eligible FVG zones from the supplied candle window. Its binding will mirror the existing M30-only strategy registries, while a GAZP literal locks the initial defaults. The existing backtest engine supplies next-open entry and intrabar fills; the core emits `FillBy` and derives actual-fill exits from `Position`.

**Tech Stack:** Go 1.25, existing internal backtest engine, Go standard-library JSON, existing Wilder ATR indicator.

**Spec:** `docs/superpowers/specs/2026-11-04-fvg-imbalance-design.md`

## Global Constraints

- Run only on `enum.Minutes30`; reject all other intervals.
- Keep decisions pure and use completed candle data only; no live adapter, I/O, short side, scanner, trailing stop, trend filter, or EOD exit.
- Preserve the user's existing `pdl_rsi_m30` deletions and unrelated dirty changes.
- Grid-facing fields are numeric: `ATRPeriod`, `MinGapATR`, `MaxRetestBars`, `RequireCloseAboveZone`, and `TakeProfitR`.
- Default GAZP values are `14`, `0.25`, `8`, `1`, and `1.5` respectively.

## Review Focus

- A FVG whose body is not fully inside the two outer wick levels must not create a zone — Task 1 formation rejection test.
- ATR changes after formation must not turn an earlier invalid/valid zone into the opposite result — Task 1 formation-time ATR test.
- A retest after a missing M30 slot must never fill at a later session or weekend price — Task 2 `FillBy` engine test.
- A next-open fill at or below the FVG lower boundary must exit on that same bar at the open — Task 2 gap-through-stop engine test.
- More than one active zone must choose the newest qualifying zone deterministically — Task 1 overlapping-zones test.

---

### Task 1: FVG core and decision contract

**Files:**
- Create: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`
- Create: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`
- Create: `internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go`

**Interfaces:**
- Consumes: `strategy.MarketData`, `strategy.Position`, `model.Signal`, `pkg/indicators.ATR`.
- Produces: `type Params`, `func DefaultParams() Params`, `func ValidateParams(Params) error`, `func NewWithParams(ticker string, step time.Duration, p Params) *Strategy`, and methods `Ticker() string`, `Lookback() int`, `Decide(strategy.MarketData) model.Signal`.

- [ ] **Step 1: Write failing formation and parameter-validation tests**

  Add table tests for a valid three-bar green-body FVG and rejection for a red middle candle, touching outer wicks, body outside `[high_a, low_c]`, undersized formation-time ATR, and each invalid parameter domain.

- [ ] **Step 2: Run the core tests to verify they fail**

  Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test(FVG|Validate)' -count=1`

  Expected: FAIL because the core package and exported API do not exist.

- [ ] **Step 3: Implement the minimal FVG discovery API and validated defaults**

  Define `Params` with the five grid fields, defaults from the spec, and `ValidateParams`. Discover only completed triples; freeze ATR at formation and represent zones as lower/upper plus formation index.

- [ ] **Step 4: Run the formation tests to verify they pass**

  Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test(FVG|Validate)' -count=1`

  Expected: PASS.

- [ ] **Step 5: Write failing entry/exit tests**

  Add tests proving: expiry after 8 later bars; lower-boundary invalidation; `RequireCloseAboveZone=1` rejects a close inside the zone while `0` accepts it; the newest of overlapping zones wins; a buy has `FillBy` exactly one M30 step later; and a position returns `SL` before `TP` when both boundaries are crossed.

- [ ] **Step 6: Run the entry/exit tests to verify they fail**

  Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test(Retest|Position|Newest)' -count=1`

  Expected: FAIL because `Decide` does not yet implement retest selection and management.

- [ ] **Step 7: Implement `(*Strategy).Decide`**

  When flat, reconstruct still-live zones from the M30 window and emit a next-open buy for the newest protected retest. Set `StopLoss` to the zone lower boundary and `FillBy` to the next M30 open. When positioned, use `Position.PurchasePrice` and `Position.StopLoss` to derive the actual-fill target; return `SL` before `TP`, including a fill at/below stop.

- [ ] **Step 8: Run the core package tests to verify they pass**

  Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1`

  Expected: PASS.

- [ ] **Step 9: Write the failing engine integration tests**

  Use `backtest.Run` with `EntryAtNextOpen: true` to prove the signal expires across a missing M30 slot, actual entry target is calculated from the opened position rather than the signal close, and a gap-through-stop exits on the fill bar at that bar's open.

- [ ] **Step 10: Run the engine tests to verify they fail**

  Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestEngine -count=1`

  Expected: FAIL until the core manages fill-bar position state and actual-fill target correctly.

- [ ] **Step 11: Make only the core changes required for engine cases**

  Keep all execution behaviour inside the strategy and the existing engine contract; do not change shared engine code.

- [ ] **Step 12: Run all core tests to verify they pass**

  Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1`

  Expected: PASS.

- [ ] **Step 13: Commit the core task**

  ```bash
  git add internal/service/trading_strategy/fvg_imbalance/strategy/core
  git commit -m "feat: add FVG imbalance strategy core"
  ```

### Task 2: GAZP literal and M30 registry

**Files:**
- Create: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go`
- Create: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp_test.go`
- Create: `internal/service/backtest/fvg_imbalance_registry.go`
- Create: `internal/service/backtest/fvg_imbalance_registry_test.go`

**Interfaces:**
- Consumes: Task 1 `core.Params`, `core.DefaultParams`, `core.ValidateParams`, and `core.NewWithParams`.
- Produces: `gazp.Params() core.Params` and `FVGImbalanceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error)`.

- [ ] **Step 1: Write failing literal and registry tests**

  Assert the GAZP literal exactly matches the five specified defaults; assert GAZP and an unregistered ticker bind on `Minutes30`, non-M30 intervals fail, partial JSON override layers over defaults, and invalid JSON parameters are rejected.

- [ ] **Step 2: Run the registry tests to verify they fail**

  Run: `go test ./internal/service/backtest ./internal/service/trading_strategy/fvg_imbalance/strategy/gazp -run 'Test(FVGImbalance|GAZP)' -count=1`

  Expected: FAIL because the literal and lookup function do not exist.

- [ ] **Step 3: Implement the literal and binding**

  Follow `HammerLookupOrGeneric`'s binding shape. Start JSON from the registered/generic defaults, unmarshal overrides, validate `core.Params`, and construct `core.NewWithParams(ticker, interval.ToTimeDuration(), p)`.

- [ ] **Step 4: Run literal and registry tests to verify they pass**

  Run: `go test ./internal/service/backtest ./internal/service/trading_strategy/fvg_imbalance/strategy/gazp -run 'Test(FVGImbalance|GAZP)' -count=1`

  Expected: PASS.

- [ ] **Step 5: Commit the registry task**

  ```bash
  git add internal/service/backtest/fvg_imbalance_registry.go internal/service/backtest/fvg_imbalance_registry_test.go internal/service/trading_strategy/fvg_imbalance/strategy/gazp
  git commit -m "feat: register GAZP FVG imbalance backtest"
  ```

### Task 3: CLI wiring, calibration grid, and mechanics documentation

**Files:**
- Modify: `cmd/backtest/main.go`
- Modify: `cmd/backtest/main_test.go`
- Create: `data/params/fvg_imbalance/gazp/grid.json`
- Create: `docs/fvg_imbalance/strategy.md`

**Interfaces:**
- Consumes: Task 2 `FVGImbalanceLookupOrGeneric`.
- Produces: `cmd/backtest -strategy fvg_imbalance -interval Minutes30` and the GAZP calibration grid.

- [ ] **Step 1: Write the failing CLI-route test**

  Extend the table-driven strategy-selection test so `fvg_imbalance` chooses the new binding; add a negative test that a non-M30 interval returns the registry error.

- [ ] **Step 2: Run the command tests to verify they fail**

  Run: `go test ./cmd/backtest -run 'Test.*FVGImbalance' -count=1`

  Expected: FAIL because the strategy name is not routed.

- [ ] **Step 3: Add the strategy case and strategy list text**

  Add the `fvg_imbalance` switch case with the same error flow as `hammer`; include the name in the unknown-strategy help text.

- [ ] **Step 4: Run the command tests to verify they pass**

  Run: `go test ./cmd/backtest -run 'Test.*FVGImbalance' -count=1`

  Expected: PASS.

- [ ] **Step 5: Add the GAZP grid and mechanics-only document**

  Create a one-phase grid that includes `MinGapATR`, `MaxRetestBars` (including `8`), `RequireCloseAboveZone` (`0` and `1`), and `TakeProfitR` (including `1.5`). Document only algorithm mechanics, defaults, validation, and an M30 backtest invocation; do not add calibration results.

- [ ] **Step 6: Verify JSON and focused packages**

  Run: `jq empty data/params/fvg_imbalance/gazp/grid.json && go test ./cmd/backtest ./internal/service/backtest ./internal/service/trading_strategy/fvg_imbalance/... -count=1`

  Expected: exit 0.

- [ ] **Step 7: Commit the integration task**

  ```bash
  git add cmd/backtest/main.go cmd/backtest/main_test.go data/params/fvg_imbalance/gazp/grid.json docs/fvg_imbalance/strategy.md
  git commit -m "feat: expose FVG imbalance backtest"
  ```

### Task 4: Full verification

**Files:**
- Modify: only files required by failures discovered in this task.

**Interfaces:**
- Consumes: Tasks 1–3.
- Produces: fresh evidence that the full repository test and supported production packages build with the new strategy.

- [ ] **Step 1: Run formatting and focused static checks**

  Run: `gofmt -w internal/service/trading_strategy/fvg_imbalance internal/service/backtest/fvg_imbalance_registry.go internal/service/backtest/fvg_imbalance_registry_test.go && go test ./internal/service/trading_strategy/fvg_imbalance/... ./internal/service/backtest ./cmd/backtest -count=1`

  Expected: exit 0.

- [ ] **Step 2: Run the repository test suite**

  Run: `go test -race ./...`

  Expected: exit 0; report any pre-existing failure by test name rather than hiding it.

- [ ] **Step 3: Build supported production packages**

  Run: `go build ./internal/... ./pkg/... ./cmd/...`

  Expected: exit 0; do not use `go build ./...` because the repository's `magefiles` package is not a main package.

- [ ] **Step 4: Inspect the final diff**

  Run: `git diff --check && git status --short && git diff -- docs/superpowers/specs/2026-11-04-fvg-imbalance-design.md docs/superpowers/plans/2026-11-04-fvg-imbalance.md internal/service/trading_strategy/fvg_imbalance internal/service/backtest/fvg_imbalance_registry.go internal/service/backtest/fvg_imbalance_registry_test.go cmd/backtest/main.go cmd/backtest/main_test.go data/params/fvg_imbalance/gazp/grid.json docs/fvg_imbalance/strategy.md`

  Expected: no whitespace errors and no changes outside the feature plus the pre-existing user changes.

- [ ] **Step 5: Commit any verification-only corrections**

  ```bash
  git add <only-files-fixed-in-this-task>
  git commit -m "test: verify FVG imbalance strategy"
  ```
