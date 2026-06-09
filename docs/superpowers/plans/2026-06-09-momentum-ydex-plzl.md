# YDEX + PLZL Momentum Integration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Register YDEX (Яндекс) and PLZL (Полюс) as momentum-strategy tickers, mirroring the existing RUAL/AFKS pattern, so the user can later run `-calibrate` and hardcode winning params.

**Architecture:** Each ticker is a tiny package exposing `const Ticker` and `func DefaultParams() core.Params`. The momentum registry maps each ticker to a `Binding` built from those defaults. New tickers start with values equal to the generic baseline (uncalibrated) plus a doc-comment warning about each instrument's corporate-action data gap. A per-ticker calibration grid lives in `data/params/<ticker>/momentum_grid.json`.

**Tech Stack:** Go 1.25, standard `testing` package.

---

## File Structure

- Create: `internal/service/trading_strategy/momentum/strategy/ydex/ydex.go` — YDEX ticker + uncalibrated defaults.
- Create: `internal/service/trading_strategy/momentum/strategy/plzl/plzl.go` — PLZL ticker + uncalibrated defaults.
- Modify: `internal/service/backtest/momentum_registry.go` — import + register both tickers.
- Modify: `internal/service/backtest/momentum_registry_test.go` — add registration tests for YDEX and PLZL.
- Create: `data/params/plzl/momentum_grid.json` — calibration grid (copy of the standard grid; `ydex` already exists).

The uncalibrated baseline values used in both new packages (identical to `genericMomentumDefaults()` in `momentum_registry.go`):

```go
core.Params{
    EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
    VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
    ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
    MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
    CooldownBars: 0, DailyTrendPeriod: 0,
}
```

---

### Task 1: YDEX ticker package + registration

**Files:**
- Test: `internal/service/backtest/momentum_registry_test.go`
- Create: `internal/service/trading_strategy/momentum/strategy/ydex/ydex.go`
- Modify: `internal/service/backtest/momentum_registry.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/service/backtest/momentum_registry_test.go`:

```go
func TestMomentumLookupRegisteredYDEX(t *testing.T) {
	if _, ok := momentumRegistry["YDEX"]; !ok {
		t.Fatal("YDEX not registered in momentumRegistry")
	}
	b := MomentumLookupOrGeneric("YDEX")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	want := core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
	if got != want {
		t.Fatalf("YDEX defaults = %+v\nwant %+v", got, want)
	}
	if s := b.Build(got); s.Ticker() != "YDEX" {
		t.Fatalf("ticker=%q want YDEX", s.Ticker())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestMomentumLookupRegisteredYDEX -v`
Expected: FAIL at `"YDEX not registered in momentumRegistry"` (key absent).

- [ ] **Step 3: Create the YDEX package**

Create `internal/service/trading_strategy/momentum/strategy/ydex/ydex.go`:

```go
// Package ydex supplies the ticker and momentum Params for YDEX (Яндекс).
// Values currently mirror the generic baseline — UNCALIBRATED. Run the backtest
// with -calibrate and hardcode the winning combination here (MACD periods are
// tuned per ticker).
//
// Data caveat: YDEX is the post-relisting ticker for Yandex (trading resumed
// ~Aug 2024 after the MOEX symbol change from YNDX). Calibrate only on a window
// that STARTS after the relisting — otherwise the price gap is counted as a real
// move and corrupts the metrics. Constrain the window via the backtest -months flag.
package ydex

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "YDEX"

// DefaultParams returns YDEX's momentum parameters (uncalibrated generic baseline).
func DefaultParams() core.Params {
	return core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
}
```

- [ ] **Step 4: Register YDEX in the registry**

In `internal/service/backtest/momentum_registry.go`, add the import (alongside the existing `momentumafks` / `momentumrusal` imports):

```go
	momentumydex "tinvest/internal/service/trading_strategy/momentum/strategy/ydex"
```

And add to the `momentumRegistry` map:

```go
	momentumydex.Ticker: momentumBindingFor(momentumydex.Ticker, momentumydex.DefaultParams),
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/service/backtest/ -run TestMomentumLookupRegisteredYDEX -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/momentum/strategy/ydex/ydex.go \
        internal/service/backtest/momentum_registry.go \
        internal/service/backtest/momentum_registry_test.go
git commit -m "feat(momentum): register YDEX ticker (uncalibrated baseline)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: PLZL ticker package + grid + registration

**Files:**
- Test: `internal/service/backtest/momentum_registry_test.go`
- Create: `internal/service/trading_strategy/momentum/strategy/plzl/plzl.go`
- Create: `data/params/plzl/momentum_grid.json`
- Modify: `internal/service/backtest/momentum_registry.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/service/backtest/momentum_registry_test.go`:

```go
func TestMomentumLookupRegisteredPLZL(t *testing.T) {
	if _, ok := momentumRegistry["PLZL"]; !ok {
		t.Fatal("PLZL not registered in momentumRegistry")
	}
	b := MomentumLookupOrGeneric("PLZL")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	want := core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
	if got != want {
		t.Fatalf("PLZL defaults = %+v\nwant %+v", got, want)
	}
	if s := b.Build(got); s.Ticker() != "PLZL" {
		t.Fatalf("ticker=%q want PLZL", s.Ticker())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestMomentumLookupRegisteredPLZL -v`
Expected: FAIL at `"PLZL not registered in momentumRegistry"` (key absent).

- [ ] **Step 3: Create the PLZL package**

Create `internal/service/trading_strategy/momentum/strategy/plzl/plzl.go`:

```go
// Package plzl supplies the ticker and momentum Params for PLZL (Полюс).
// Values currently mirror the generic baseline — UNCALIBRATED. Run the backtest
// with -calibrate and hardcode the winning combination here (MACD periods are
// tuned per ticker).
//
// Data caveat: Polyus did a 1:10 stock split in 2024, which drops the price by a
// factor of 10 on the split date. Calibrate only on a window that STARTS after
// the split — otherwise the gap is counted as a real move and corrupts the
// metrics. Constrain the window via the backtest -months flag.
package plzl

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "PLZL"

// DefaultParams returns PLZL's momentum parameters (uncalibrated generic baseline).
func DefaultParams() core.Params {
	return core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
}
```

- [ ] **Step 4: Create the PLZL calibration grid**

Create `data/params/plzl/momentum_grid.json`:

```json
{
  "MACDSlow": [21, 26],
  "MACDSignal": [9],
  "MACDBelowZeroOnly": [0, 1],
  "SLMult": [0.5, 1.0],
  "TakeProfitRR": [1.5, 2.0, 3.0],
  "VolMultiplier": [1.0, 1.2, 1.5],
  "MaxDailyATRUsed": [0.5, 0.6, 0.7],
  "EMAPeriod": [200, 150, 100],
  "MACDFast": [12, 10, 8],
  "UseTrail": [0, 1],
  "DailyTrendPeriod": [0, 10, 20]
}
```

- [ ] **Step 5: Register PLZL in the registry**

In `internal/service/backtest/momentum_registry.go`, add the import:

```go
	momentumplzl "tinvest/internal/service/trading_strategy/momentum/strategy/plzl"
```

And add to the `momentumRegistry` map:

```go
	momentumplzl.Ticker: momentumBindingFor(momentumplzl.Ticker, momentumplzl.DefaultParams),
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/service/backtest/ -run TestMomentumLookupRegisteredPLZL -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/momentum/strategy/plzl/plzl.go \
        data/params/plzl/momentum_grid.json \
        internal/service/backtest/momentum_registry.go \
        internal/service/backtest/momentum_registry_test.go
git commit -m "feat(momentum): register PLZL ticker (uncalibrated baseline) + grid

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Full verification + commit ydex grid

**Files:**
- Add: `data/params/ydex/momentum_grid.json` (already on disk, untracked — commit it).

- [ ] **Step 1: Run the full backtest test suite**

Run: `go test ./internal/service/backtest/...`
Expected: PASS (all tests, including the frozen-baseline and partial-override tests).

- [ ] **Step 2: Build and vet the whole module**

Run: `go build ./... && go vet ./...`
Expected: no output, exit 0.

- [ ] **Step 3: Commit the YDEX grid**

The YDEX grid was created earlier and is still untracked; commit it now so both tickers have grids in version control.

```bash
git add data/params/ydex/momentum_grid.json
git commit -m "chore(momentum): track YDEX calibration grid

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Post-plan: user's calibration step (out of scope)

After this plan merges, for each of YDEX and PLZL:
1. Pick a `-months` window that starts AFTER the corporate action (YDEX relisting ~Aug 2024; PLZL 1:10 split 2024). Verify candles with `-refresh` and check there is no price gap inside the window.
2. Run the backtest with `-calibrate` for the ticker.
3. Hardcode the winning combination into the ticker's `DefaultParams()` and update its doc-comment with the grid date / PF / trade count (mirroring `rusal.go` / `afks.go`).
4. Update the matching `want` struct in `momentum_registry_test.go`.
