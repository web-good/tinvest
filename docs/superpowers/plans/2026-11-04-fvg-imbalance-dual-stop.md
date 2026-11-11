# FVG imbalance dual-stop implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make FVG entries require a deep retest and replace the wick-triggered stop with a close-confirmed primary stop plus an intrabar emergency stop.

**Architecture:** The core keeps the existing `Position.StopLoss` as the primary, sizing and target stop. It derives the emergency level from frozen entry data as `Position.StopLoss - 2 * StopBufferATR * Position.EntryATR`, so no new persisted position field or risk-sizing contract is needed. The engine gives `SL_CLOSE` its completed-candle close fill, while `SL_EMERGENCY` retains normal intrabar gap execution.

**Tech Stack:** Go 1.25, existing scalping strategy interface, internal backtest engine, Go standard-library JSON.

**Spec:** `docs/superpowers/specs/2026-11-04-fvg-imbalance-dual-stop-design.md`

## Global Constraints

- Only completed M30 candles drive decisions; the strategy remains backtest-only and long-only.
- The primary level is unchanged: `lower - StopBufferATR * ATR_formation`.
- Primary exit requires `close <= Position.StopLoss`; intrabar wick-only contact is ignored.
- Emergency level is `Position.StopLoss - 2 * StopBufferATR * Position.EntryATR` and exits when `low <= emergency`, including a gap down.
- Risk sizing and `TakeProfitR` remain based on the primary stop.
- `MinRetestDepthPct` is a finite numeric parameter in `0..100`, default `50`; an entry candle's low must be `<= upper - MinRetestDepthPct/100*(upper-lower)`, irrespective of its close.
- A retest low below `lower` invalidates the zone, while `low == lower` is allowed.
- Remove `RequireCloseAboveZone` from FVG parameters, ticker literals, grids, validation and mechanics documentation; reject it when supplied in FVG JSON overrides.
- Preserve unrelated dirty worktree changes; alter the shared backtest engine only to fill `SL_CLOSE` at the completed candle's close in both Run and Trace.

## Review Focus

- A candle whose low pierces the primary stop but closes above it must hold the position unless it also reaches emergency; covered in Task 1.
- A close exactly at the primary stop must exit, while equality at the emergency low must also exit; covered in Task 1.
- `StopBufferATR=0` makes the two levels equal, so a low touch must remain an immediate emergency exit; covered in Task 1.
- A low exactly at the configured depth threshold must enter, while a higher low must not; the retest close must not change that result; covered in Task 3.
- Depth `100` must accept a low on `lower`; a low below `lower` must reject it; covered in Task 3.
- A legacy JSON override with `RequireCloseAboveZone` must fail rather than silently changing strategy semantics; covered in Task 2.

### Task 1: Core entry and dual-stop rules

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go`

**Interfaces:**
- Consumes: `strategy.Position{PurchasePrice, StopLoss, EntryATR}` and completed `strategy.MarketData`.
- Produces: `SignalBuy` only for a retest close in `[lower, upper - MinRetestDepthPct/100*(upper-lower)]`; `SignalSell` with reason `SL_CLOSE` for a primary close breach and `SL_EMERGENCY` for an intrabar emergency breach. Both reasons must be recognized by `model.IsStopReason`.

- [x] **Step 1: Write failing core tests for deep retest eligibility**

Replace tests that depend on `RequireCloseAboveZone` with `TestRetestRequiresConfiguredDepth`: for `MinRetestDepthPct=50`, a retest with `close=(lower+upper)/2` returns `SignalBuy` and one with `close>midpoint` returns `SignalNone`. Add `MinRetestDepthPct=100` cases: `low==close==lower` enters, while `low<lower` returns `SignalNone`.

- [x] **Step 2: Run the retest test to verify it fails**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestRetestRequiresConfiguredDepth -count=1`

Expected: FAIL because the current core has no depth parameter and permits a close above the desired threshold.

- [x] **Step 3: Implement the minimum entry-rule change in `core.go`**

Add `Params.MinRetestDepthPct float64` with default `50` and validation for finite values in `0..100`; remove `Params.RequireCloseAboveZone` and its validation. In flat-position retest selection, reject `low < z.lower`, `close < z.lower`, or `close > z.upper-s.p.MinRetestDepthPct/100*(z.upper-z.lower)`; preserve the existing zone-touch condition.

- [x] **Step 4: Run the retest tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'TestRetest|TestDefaultParams|TestValidateParams' -count=1`

Expected: PASS.

- [x] **Step 5: Write failing management tests for primary and emergency stops**

Add a table test such as `TestPositionDualStops` using `PurchasePrice=104`, `StopLoss=100`, `EntryATR=2`, `StopBufferATR=.5` (emergency `98`):

```go
// low 99, close 101 => no sell (primary wick only)
// low 101, close 100 => SignalSell / "SL_CLOSE" / StopLoss 100
// low 98, close 101 => SignalSell / "SL_EMERGENCY" / StopLoss 98
```

Add equality and zero-buffer cases: `low==emergency` exits, and with buffer `0`, `low==primary` yields `SL_EMERGENCY`. Assert a target hit on the same candle loses to either applicable stop.

- [x] **Step 6: Run the dual-stop test to verify it fails**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestPositionDualStops -count=1`

Expected: FAIL because the current core exits on `low <= Position.StopLoss` and has only reason `SL`.

- [x] **Step 7: Implement primary-close and emergency-stop management**

In the positioned branch, derive `emergency := pos.StopLoss - 2*s.p.StopBufferATR*pos.EntryATR`. Check `low <= emergency` first and return `SL_EMERGENCY` with that exact stop level; then check `close <= pos.StopLoss` and return `SL_CLOSE` at the primary level; check TP only afterwards. Keep the target formula based on `pos.StopLoss`.

- [x] **Step 8: Extend stop-reason recognition and run focused core tests**

Add the two new reasons to `model.IsStopReason` in its owning model file, preserving existing stop reasons. Run:

`go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core ./internal/service/trading_strategy/scalping/model -count=1`

Expected: PASS.

- [x] **Step 9: Write and run failing engine tests for fill price behavior**

Add engine cases showing an emergency breach fills at `min(emergency, open)`, while a close-confirmed primary stop exits at `min(primary, open)` at the candle close timestamp. Run the new cases first and observe failure, then make only the test-driven core/model changes necessary for them to pass.

- [x] **Step 10: Run the complete FVG core package**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1`

Expected: PASS.

- [x] **Step 11: Commit the core change**

```bash
git add internal/service/trading_strategy/fvg_imbalance/strategy/core internal/service/trading_strategy/scalping/model
git commit -m "feat: add FVG close-confirmed dual stop"
```

### Task 2: Parameter contract, FVG data and documentation

**Files:**
- Modify: `internal/service/backtest/fvg_imbalance_registry.go`
- Modify: `internal/service/backtest/fvg_imbalance_registry_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/domrf/domrf.go`
- Modify: `data/params/fvg_imbalance/gazp/grid.json`
- Modify: `data/params/fvg_imbalance/gazp/ordered_grid.json`
- Modify: `docs/fvg_imbalance/strategy.md`

**Interfaces:**
- Consumes: the revised `core.Params` with `MinRetestDepthPct` and no `RequireCloseAboveZone` field.
- Produces: FVG override parsing that rejects unknown fields, and parameter grids/documentation that no longer refer to the removed switch.

- [ ] **Step 1: Write failing registry and literal tests**

Remove assertions for `RequireCloseAboveZone`. Add literal/default assertions for `MinRetestDepthPct: 50`, validation rejections for `-1`, `101`, `NaN` and infinity, and `TestFVGImbalanceRejectsLegacyCloseAboveZone` that calls `ParseParams([]byte(`{"RequireCloseAboveZone":1}`))` and expects an error naming that field. Keep a partial override using `MinRetestDepthPct` to prove valid overlays still work.

- [ ] **Step 2: Run the registry test to verify it fails**

Run: `go test ./internal/service/backtest ./internal/service/trading_strategy/fvg_imbalance/strategy/gazp -run 'Test(FVGImbalance|GAZP)' -count=1`

Expected: FAIL because `encoding/json.Unmarshal` silently accepts the legacy unknown field.

- [ ] **Step 3: Implement strict FVG JSON decoding**

Replace the FVG registry's direct `json.Unmarshal` with a `json.Decoder` using `DisallowUnknownFields`, while preserving defaults-before-overlay and `core.ValidateParams`. Remove the field from both GAZP and DOMRF literals.

- [ ] **Step 4: Run registry and literal tests to verify they pass**

Run: `go test ./internal/service/backtest ./internal/service/trading_strategy/fvg_imbalance/strategy/gazp -run 'Test(FVGImbalance|GAZP)' -count=1`

Expected: PASS.

- [ ] **Step 5: Update grid and mechanics documentation**

Remove `RequireCloseAboveZone` from both FVG grids and any phase expectations, then add a `MinRetestDepthPct` grid axis spanning `0`, `50` and `100`. In `docs/fvg_imbalance/strategy.md`, state the percentage-depth entry condition, lower-boundary rule, two stop formulas, stop priority and fill behavior; remove the old confirmation-switch text and table row. Keep calibration results out of mechanics docs.

- [ ] **Step 6: Validate data and focused integration**

Run: `jq empty data/params/fvg_imbalance/gazp/grid.json data/params/fvg_imbalance/gazp/ordered_grid.json && go test ./cmd/backtest ./internal/service/backtest ./internal/service/trading_strategy/fvg_imbalance/... -count=1`

Expected: PASS.

- [ ] **Step 7: Commit the contract/documentation change**

```bash
git add internal/service/backtest/fvg_imbalance_registry.go internal/service/backtest/fvg_imbalance_registry_test.go internal/service/trading_strategy/fvg_imbalance/strategy/gazp internal/service/trading_strategy/fvg_imbalance/strategy/domrf data/params/fvg_imbalance/gazp docs/fvg_imbalance/strategy.md
git commit -m "feat: require deep FVG retests"
```

### Task 3: Wick-depth entry and close-confirmed fill

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go`
- Modify: `internal/domain/backtest/engine.go`
- Modify: `internal/domain/backtest/engine_test.go`
- Modify: `internal/service/trading_strategy/scalping/model/signal.go`
- Modify: `internal/service/trading_strategy/scalping/model/signal_test.go`
- Modify: `docs/fvg_imbalance/strategy.md`

**Interfaces:**
- Consumes: the Task 1 `SL_CLOSE` and `SL_EMERGENCY` signals and Task 2 `MinRetestDepthPct` parameter.
- Produces: a buy when the retest **low**, rather than its close, reaches the configured depth threshold; `SL_CLOSE` fills at the completed decision candle's `Close` in `Run` and `Trace`, while the emergency reason continues to use `min(stop, open)`.

- [x] **Step 1: Write failing wick-depth core tests**

Replace the close-based depth assertions with `TestRetestDepthUsesWick`: at `MinRetestDepthPct=50` (`lower=100`, `upper=102`), `low=101` produces `SignalBuy` even if `close=103`; `low=101.01` produces `SignalNone` regardless of close. Keep equality at `low=lower` valid and `low<lower` invalid, including `100%` depth.

- [x] **Step 2: Run the wick-depth test to verify it fails**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestRetestDepthUsesWick -count=1`

Expected: FAIL because the current core applies `MinRetestDepthPct` to the close.

- [x] **Step 3: Implement wick-only depth selection**

In the flat-position FVG retest branch, retain the strict lower-bound invalidation and reject when `md.Lows[k]` is above `z.upper-s.p.MinRetestDepthPct/100*(z.upper-z.lower)`. Remove depth-dependent conditions on `md.Closes[k]`.

- [x] **Step 4: Run focused core tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'TestRetest|TestDefaultParams|TestValidateParams' -count=1`

Expected: PASS.

- [x] **Step 5: Write failing engine tests for `SL_CLOSE` fill semantics**

Add tests where `SL_CLOSE` is emitted on a candle with `Close=98.5`, `Open=99`, primary level `100`: both `backtest.Run` and `backtest.Trace` must use exit price `98.5`, not `min(100,99)`. Keep/add an emergency case proving its fill remains `min(emergency, open)`.

- [x] **Step 6: Run the new engine tests to verify they fail**

Run: `go test ./internal/domain/backtest ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test.*SLClose|TestEngineDualStopFills' -count=1`

Expected: FAIL because the current engine classifies `SL_CLOSE` as a generic stop and fills it at `min(level, open)`.

- [x] **Step 7: Implement distinct close-stop execution in both engine paths**

Add `model.IsCloseConfirmedStopReason(reason string) bool` returning true only for `SL_CLOSE`. In `backtest.Run` and `backtest.Trace`, check this predicate before generic `model.IsStopReason`: leave the default `exitPrice=c.Close` for close-confirmed stops, and preserve the existing `min(stop, open)` branch for emergency and all old stop reasons.

- [x] **Step 8: Update mechanics documentation and run scoped tests**

Document that depth is measured by the candle low/tail and not close; document `SL_CLOSE` at candle close and emergency intrabar fills. Run:

`go test ./internal/domain/backtest ./internal/service/trading_strategy/scalping/model ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1`

Expected: PASS.

- [x] **Step 9: Format and run full verification**

Run: `gofmt -w internal/service/trading_strategy/fvg_imbalance/strategy/core internal/service/trading_strategy/scalping/model internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go && go test -race ./... && go build ./internal/... ./pkg/... ./cmd/...`

Expected: PASS; report any pre-existing failure by exact test/package name.

- [x] **Step 10: Commit the final integration**

```bash
git add internal/service/trading_strategy/fvg_imbalance/strategy/core internal/service/trading_strategy/scalping/model internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go docs/fvg_imbalance/strategy.md
git commit -m "feat: use FVG wick depth and close stop fills"
```
