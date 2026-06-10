# Momentum Basket Walk-Forward Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 4 new momentum tickers (SBER, GAZP, NVTK, MDMG) as first-class instruments and build a walk-forward "basket" mode that pools each stock's out-of-sample trades into one statistically meaningful sample for validating that per-stock calibration generalizes.

**Architecture:** Each ticker keeps its OWN individually-calibrated params (per-stock packages + grids, the existing pattern). The basket is a validation layer only: for each ticker it calibrates on the early window, runs the winner on the OOS tail, and pools the tail trades across all tickers. I/O (gRPC, files) lives in `cmd/backtest`; pure pooling/metrics/rendering live in `internal/service/backtest`.

**Tech Stack:** Go 1.25, existing backtest engine (`internal/domain/backtest`), phased grid calibration (`internal/service/backtest/calibrate.go`), Tinkoff gRPC candle provider.

---

## File Structure

- Create: `internal/service/trading_strategy/momentum/strategy/sber/sber.go` — SBER ticker + DefaultParams
- Create: `internal/service/trading_strategy/momentum/strategy/gazp/gazp.go` — GAZP ticker + DefaultParams
- Create: `internal/service/trading_strategy/momentum/strategy/nvtk/nvtk.go` — NVTK ticker + DefaultParams
- Create: `internal/service/trading_strategy/momentum/strategy/mdmg/mdmg.go` — MDMG ticker + DefaultParams
- Modify: `internal/service/backtest/momentum_registry.go` — register 4 new bindings
- Modify: `internal/service/backtest/momentum_registry_test.go` — 4 new lookup tests
- Create: `data/params/sber/momentum_grid.json`
- Create: `data/params/gazp/momentum_grid.json`
- Create: `data/params/nvtk/momentum_grid.json`
- Create: `data/params/mdmg/momentum_grid.json`
- Create: `internal/service/backtest/basket.go` — `PooledMetrics`, `BasketEntry`, `BasketSummary`, `RenderBasketMarkdown`
- Create: `internal/service/backtest/basket_test.go` — `PooledMetrics` unit tests
- Modify: `cmd/backtest/main.go` — `-basket` / `-grid-dir` flags, `run()` dispatch, `runBasket`, `splitTickers`

---

## Task 1: Four momentum ticker packages

**Files:**
- Create: `internal/service/trading_strategy/momentum/strategy/sber/sber.go`
- Create: `internal/service/trading_strategy/momentum/strategy/gazp/gazp.go`
- Create: `internal/service/trading_strategy/momentum/strategy/nvtk/nvtk.go`
- Create: `internal/service/trading_strategy/momentum/strategy/mdmg/mdmg.go`

Each mirrors the existing `afks`/`rusal` packages. Starting params equal the generic
baseline (see `genericMomentumDefaults` in `momentum_registry.go`) — a placeholder until
the user calibrates and hardcodes the winner.

- [ ] **Step 1: Create `sber/sber.go`**

```go
// Package sber supplies the ticker and calibrated momentum Params for SBER (Sberbank).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here (MACD periods are tuned per ticker).
package sber

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SBER"

// DefaultParams returns SBER's momentum parameters (uncalibrated baseline).
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

- [ ] **Step 2: Create `gazp/gazp.go`** (identical body, swap package/ticker/comment)

```go
// Package gazp supplies the ticker and calibrated momentum Params for GAZP (Gazprom).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here (MACD periods are tuned per ticker).
package gazp

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "GAZP"

// DefaultParams returns GAZP's momentum parameters (uncalibrated baseline).
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

- [ ] **Step 3: Create `nvtk/nvtk.go`**

```go
// Package nvtk supplies the ticker and calibrated momentum Params for NVTK (Novatek).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here (MACD periods are tuned per ticker).
package nvtk

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "NVTK"

// DefaultParams returns NVTK's momentum parameters (uncalibrated baseline).
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

- [ ] **Step 4: Create `mdmg/mdmg.go`**

```go
// Package mdmg supplies the ticker and calibrated momentum Params for MDMG
// (MD Medical Group / "Мать и дитя"). Trades on MOEX since ~Oct-2024 after
// redomiciliation, so its candle history is shorter than the other tickers.
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here (MACD periods are tuned per ticker).
package mdmg

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "MDMG"

// DefaultParams returns MDMG's momentum parameters (uncalibrated baseline).
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

- [ ] **Step 5: Verify the packages compile**

Run: `go build ./internal/service/trading_strategy/momentum/...`
Expected: no output, exit 0.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/momentum/strategy/sber internal/service/trading_strategy/momentum/strategy/gazp internal/service/trading_strategy/momentum/strategy/nvtk internal/service/trading_strategy/momentum/strategy/mdmg
git commit -m "feat(momentum): add SBER, GAZP, NVTK, MDMG ticker packages"
```

---

## Task 2: Register new tickers + lookup tests (TDD)

**Files:**
- Modify: `internal/service/backtest/momentum_registry.go`
- Test: `internal/service/backtest/momentum_registry_test.go`

The 4 new packages all use the generic baseline params, so each test's `want` equals
`genericMomentumDefaults`. Write the tests first; they fail to compile (imports for
unregistered packages), then registration makes them pass.

- [ ] **Step 1: Append 4 tests to `momentum_registry_test.go`**

Add these imports to the existing import block (the package already imports `core`):

```go
	momentumgazp "tinvest/internal/service/trading_strategy/momentum/strategy/gazp"
	momentummdmg "tinvest/internal/service/trading_strategy/momentum/strategy/mdmg"
	momentumnvtk "tinvest/internal/service/trading_strategy/momentum/strategy/nvtk"
	momentumsber "tinvest/internal/service/trading_strategy/momentum/strategy/sber"
```

Append these test functions:

```go
func TestMomentumLookupRegisteredSBER(t *testing.T) {
	b := MomentumLookupOrGeneric(momentumsber.Ticker)
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != momentumsber.DefaultParams() {
		t.Fatalf("SBER defaults = %+v\nwant %+v", got, momentumsber.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "SBER" {
		t.Fatalf("ticker=%q want SBER", s.Ticker())
	}
}

func TestMomentumLookupRegisteredGAZP(t *testing.T) {
	b := MomentumLookupOrGeneric(momentumgazp.Ticker)
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != momentumgazp.DefaultParams() {
		t.Fatalf("GAZP defaults = %+v\nwant %+v", got, momentumgazp.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "GAZP" {
		t.Fatalf("ticker=%q want GAZP", s.Ticker())
	}
}

func TestMomentumLookupRegisteredNVTK(t *testing.T) {
	b := MomentumLookupOrGeneric(momentumnvtk.Ticker)
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != momentumnvtk.DefaultParams() {
		t.Fatalf("NVTK defaults = %+v\nwant %+v", got, momentumnvtk.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "NVTK" {
		t.Fatalf("ticker=%q want NVTK", s.Ticker())
	}
}

func TestMomentumLookupRegisteredMDMG(t *testing.T) {
	b := MomentumLookupOrGeneric(momentummdmg.Ticker)
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != momentummdmg.DefaultParams() {
		t.Fatalf("MDMG defaults = %+v\nwant %+v", got, momentummdmg.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "MDMG" {
		t.Fatalf("ticker=%q want MDMG", s.Ticker())
	}
}
```

- [ ] **Step 2: Run the new tests — verify they FAIL**

Run: `go test ./internal/service/backtest/ -run 'TestMomentumLookupRegistered(SBER|GAZP|NVTK|MDMG)' -v`
Expected: FAIL — the lookups return generic fallback bindings, so `b.Build(got).Ticker()` already equals the ticker (generic binds to the requested ticker), BUT `got != <pkg>.DefaultParams()` because the generic fallback differs in `DailyTrendPeriod`/etc only if values differ. Since the new packages currently equal `genericMomentumDefaults`, the params check passes via fallback — the meaningful assertion is registration. To make the test prove registration, the test must distinguish registered vs fallback. Use the registry directly instead:

Replace each test's lookup assertion with a direct registry membership check. Update each of the 4 tests to additionally assert membership:

```go
	if _, ok := momentumRegistry[momentumsber.Ticker]; !ok {
		t.Fatalf("SBER not registered in momentumRegistry")
	}
```

(Insert the analogous `momentumRegistry[...]` membership check, with the matching package/ticker, as the first assertion in each of the 4 tests.)

Re-run: `go test ./internal/service/backtest/ -run 'TestMomentumLookupRegistered(SBER|GAZP|NVTK|MDMG)' -v`
Expected: FAIL with "SBER not registered in momentumRegistry" (and analogous for the others).

- [ ] **Step 3: Register the 4 bindings in `momentum_registry.go`**

Add imports:

```go
	momentumgazp "tinvest/internal/service/trading_strategy/momentum/strategy/gazp"
	momentummdmg "tinvest/internal/service/trading_strategy/momentum/strategy/mdmg"
	momentumnvtk "tinvest/internal/service/trading_strategy/momentum/strategy/nvtk"
	momentumsber "tinvest/internal/service/trading_strategy/momentum/strategy/sber"
```

Add entries to `momentumRegistry`:

```go
var momentumRegistry = map[string]Binding{
	momentumrusal.Ticker: momentumBindingFor(momentumrusal.Ticker, momentumrusal.DefaultParams),
	momentumafks.Ticker:  momentumBindingFor(momentumafks.Ticker, momentumafks.DefaultParams),
	momentumydex.Ticker:  momentumBindingFor(momentumydex.Ticker, momentumydex.DefaultParams),
	momentumplzl.Ticker:  momentumBindingFor(momentumplzl.Ticker, momentumplzl.DefaultParams),
	momentumsber.Ticker:  momentumBindingFor(momentumsber.Ticker, momentumsber.DefaultParams),
	momentumgazp.Ticker:  momentumBindingFor(momentumgazp.Ticker, momentumgazp.DefaultParams),
	momentumnvtk.Ticker:  momentumBindingFor(momentumnvtk.Ticker, momentumnvtk.DefaultParams),
	momentummdmg.Ticker:  momentumBindingFor(momentummdmg.Ticker, momentummdmg.DefaultParams),
}
```

- [ ] **Step 4: Run the tests — verify they PASS**

Run: `go test ./internal/service/backtest/ -run 'TestMomentumLookupRegistered' -v`
Expected: PASS (all registered tickers, old and new).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/momentum_registry.go internal/service/backtest/momentum_registry_test.go
git commit -m "feat(momentum): register SBER, GAZP, NVTK, MDMG bindings"
```

---

## Task 3: Calibration grids for the 4 new tickers

**Files:**
- Create: `data/params/sber/momentum_grid.json`
- Create: `data/params/gazp/momentum_grid.json`
- Create: `data/params/nvtk/momentum_grid.json`
- Create: `data/params/mdmg/momentum_grid.json`

Phased format (`core` phase with `keepTop`, then `gates`), mirroring
`data/params/afks/momentum_grid.json`, tuned per the volatility profile in the spec.

- [ ] **Step 1: Create `data/params/sber/momentum_grid.json`** (blue chip, clean trends)

```json
{
  "phases": [
    {
      "name": "core",
      "keepTop": 5,
      "grid": {
        "EMAPeriod": [200, 150, 100],
        "SLMult": [0.8, 1.0, 1.5],
        "TakeProfitRR": [2.0, 3.0],
        "VolMultiplier": [1.0, 1.2, 1.5],
        "MaxDailyATRUsed": [0.5, 0.7],
        "MACDBelowZeroOnly": [0, 1]
      }
    },
    {
      "name": "gates",
      "grid": {
        "MACDFast": [12, 8],
        "MACDSlow": [26, 21],
        "CooldownBars": [0, 12, 24],
        "DailyTrendPeriod": [0, 10, 20],
        "SignalValidBars": [0, 2, 4],
        "MaxDriftATR": [0, 1.0]
      }
    }
  ]
}
```

- [ ] **Step 2: Create `data/params/gazp/momentum_grid.json`** (heavy, dividend gaps — wider stops, stronger trend filter)

```json
{
  "phases": [
    {
      "name": "core",
      "keepTop": 5,
      "grid": {
        "EMAPeriod": [200, 150, 100],
        "SLMult": [1.0, 1.5, 2.0],
        "TakeProfitRR": [2.0, 3.0],
        "VolMultiplier": [1.0, 1.2, 1.5],
        "MaxDailyATRUsed": [0.5, 0.7],
        "MACDBelowZeroOnly": [0, 1]
      }
    },
    {
      "name": "gates",
      "grid": {
        "MACDFast": [12, 8],
        "MACDSlow": [26, 21],
        "CooldownBars": [12, 24],
        "DailyTrendPeriod": [10, 20, 50],
        "SignalValidBars": [0, 2, 4],
        "MaxDriftATR": [0, 1.0]
      }
    }
  ]
}
```

- [ ] **Step 3: Create `data/params/nvtk/momentum_grid.json`** (volatile, sanction shocks — wide stops, tight ATR-room, longer cooldown)

```json
{
  "phases": [
    {
      "name": "core",
      "keepTop": 5,
      "grid": {
        "EMAPeriod": [200, 150, 100],
        "SLMult": [1.0, 1.5, 2.0],
        "TakeProfitRR": [2.0, 3.0],
        "VolMultiplier": [1.0, 1.2, 1.5],
        "MaxDailyATRUsed": [0.5, 0.6],
        "MACDBelowZeroOnly": [0, 1]
      }
    },
    {
      "name": "gates",
      "grid": {
        "MACDFast": [12, 8],
        "MACDSlow": [26, 21],
        "CooldownBars": [12, 24, 48],
        "DailyTrendPeriod": [0, 10, 20],
        "SignalValidBars": [0, 2, 4],
        "MaxDriftATR": [0, 1.0]
      }
    }
  ]
}
```

- [ ] **Step 4: Create `data/params/mdmg/momentum_grid.json`** (small-cap, gappy, short history — conservative, fewer combos)

```json
{
  "phases": [
    {
      "name": "core",
      "keepTop": 5,
      "grid": {
        "EMAPeriod": [200, 150, 100],
        "SLMult": [1.0, 1.5],
        "TakeProfitRR": [1.5, 2.0, 3.0],
        "VolMultiplier": [1.2, 1.5],
        "MaxDailyATRUsed": [0.5, 0.7],
        "MACDBelowZeroOnly": [0, 1]
      }
    },
    {
      "name": "gates",
      "grid": {
        "MACDFast": [12, 8],
        "MACDSlow": [26, 21],
        "CooldownBars": [12, 24],
        "DailyTrendPeriod": [10, 20],
        "SignalValidBars": [0, 2, 4],
        "MaxDriftATR": [0, 1.0]
      }
    }
  ]
}
```

- [ ] **Step 5: Validate JSON parses as phased grids**

Run: `for t in sber gazp nvtk mdmg; do python3 -c "import json,sys; d=json.load(open('data/params/$t/momentum_grid.json')); assert d['phases'], '$t'; print('$t ok', len(d['phases']), 'phases')"; done`
Expected: four lines `<ticker> ok 2 phases`.

- [ ] **Step 6: Commit**

```bash
git add data/params/sber data/params/gazp data/params/nvtk data/params/mdmg
git commit -m "feat(momentum): calibration grids for SBER, GAZP, NVTK, MDMG"
```

---

## Task 4: Pooled metrics + basket types (TDD)

**Files:**
- Create: `internal/service/backtest/basket.go`
- Test: `internal/service/backtest/basket_test.go`

`PooledMetrics` reuses the existing `backtest.Compute` with a synthetic `Result` carrying
only trades — equity-based fields (MaxDrawdown, CAGR, NetPnL, Exposure) come out zero
because there is no equity curve and no capital base, which is exactly what we want for a
cross-instrument pool.

- [ ] **Step 1: Write the failing test `basket_test.go`**

```go
package backtest

import (
	"testing"

	"tinvest/internal/domain/backtest"
)

func TestPooledMetrics(t *testing.T) {
	trades := []backtest.Trade{
		{PnL: 100}, {PnL: 50}, {PnL: -40}, {PnL: -10},
	}
	m := PooledMetrics(trades)
	if m.TotalTrades != 4 {
		t.Fatalf("TotalTrades=%d want 4", m.TotalTrades)
	}
	if m.Wins != 2 || m.Losses != 2 {
		t.Fatalf("Wins/Losses=%d/%d want 2/2", m.Wins, m.Losses)
	}
	if m.GrossProfit != 150 || m.GrossLoss != 50 {
		t.Fatalf("Gross profit/loss=%.0f/%.0f want 150/50", m.GrossProfit, m.GrossLoss)
	}
	if m.ProfitFactor != 3.0 {
		t.Fatalf("ProfitFactor=%.3f want 3.0", m.ProfitFactor)
	}
	if m.Expectancy != 25 {
		t.Fatalf("Expectancy=%.3f want 25", m.Expectancy)
	}
	if m.BestTrade != 100 || m.WorstTrade != -40 {
		t.Fatalf("Best/Worst=%.0f/%.0f want 100/-40", m.BestTrade, m.WorstTrade)
	}
	// Equity-based fields have no meaning across a pool and must stay zero.
	if m.MaxDrawdown != 0 || m.CAGR != 0 || m.NetPnL != 0 {
		t.Fatalf("equity fields must be zero: DD=%.3f CAGR=%.3f Net=%.3f", m.MaxDrawdown, m.CAGR, m.NetPnL)
	}
}

func TestPooledMetricsEmpty(t *testing.T) {
	m := PooledMetrics(nil)
	if m.TotalTrades != 0 || m.ProfitFactor != 0 || m.Expectancy != 0 {
		t.Fatalf("empty pool must be zero metrics: %+v", m)
	}
}

func TestPooledMetricsAllWins(t *testing.T) {
	m := PooledMetrics([]backtest.Trade{{PnL: 10}, {PnL: 20}})
	// GrossLoss==0 with positive GrossProfit -> ProfitFactor == GrossProfit (engine convention).
	if m.ProfitFactor != 30 {
		t.Fatalf("ProfitFactor=%.3f want 30 (GrossProfit when no losses)", m.ProfitFactor)
	}
}
```

- [ ] **Step 2: Run the test — verify it FAILS**

Run: `go test ./internal/service/backtest/ -run TestPooledMetrics -v`
Expected: FAIL — `undefined: PooledMetrics`.

- [ ] **Step 3: Create `basket.go` with the types and `PooledMetrics`**

```go
package backtest

import (
	"fmt"
	"strings"
	"time"

	"tinvest/internal/domain/backtest"
)

// BasketEntry is one ticker's out-of-sample result inside a basket walk-forward run.
type BasketEntry struct {
	Ticker         string
	Trades         int
	ProfitFactor   float64
	NetPnL         float64
	NetPnLPct      float64
	MaxDrawdownPct float64
	WinRate        float64
	Params         []backtest.ParamLine // winning calibrated params for this ticker
	Skipped        bool                 // true when the ticker produced no OOS result
	Note           string               // reason when skipped or no trades
}

// BasketSummary aggregates per-ticker OOS results plus the pooled-trade metrics.
type BasketSummary struct {
	Pooled  backtest.Metrics // metrics over the pooled OOS trades (trade-based fields only)
	Entries []BasketEntry
}

// PooledMetrics computes trade-based metrics over a flat list of trades drawn from
// multiple instruments. It reuses backtest.Compute with a synthetic Result carrying
// only trades; equity-based fields (MaxDrawdown, CAGR, NetPnL, Exposure) come out zero
// because a pool spanning separate capital bases has no single equity curve.
func PooledMetrics(trades []backtest.Trade) backtest.Metrics {
	return backtest.Compute(backtest.Result{Trades: trades}, 0, 0, 0)
}
```

- [ ] **Step 4: Run the test — verify it PASSES**

Run: `go test ./internal/service/backtest/ -run TestPooledMetrics -v`
Expected: PASS (all three tests).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/basket.go internal/service/backtest/basket_test.go
git commit -m "feat(backtest): pooled-trade metrics and basket summary types"
```

---

## Task 5: Basket Markdown renderer

**Files:**
- Modify: `internal/service/backtest/basket.go`

Pure rendering — no test (string formatting is exercised by the e2e run in Task 7;
follows the existing convention where `RenderCalibrationMarkdown` has no unit test).

- [ ] **Step 1: Append `RenderBasketMarkdown` and `paramSummary` to `basket.go`**

```go
// RenderBasketMarkdown renders the pooled-OOS aggregate plus a per-ticker breakdown.
// from/to bound the out-of-sample window common to every ticker.
func RenderBasketMarkdown(metric string, s BasketSummary, from, to time.Time) string {
	var b strings.Builder
	m := s.Pooled
	b.WriteString("# Корзина momentum — walk-forward OOS\n\n")
	fmt.Fprintf(&b, "- OOS-период: %s — %s\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
	fmt.Fprintf(&b, "- Калибровка ранжировалась по: %s\n\n", metric)

	b.WriteString("## Пул сделок (агрегат OOS)\n\n")
	b.WriteString("| Метрика | Значение |\n|---|---|\n")
	fmt.Fprintf(&b, "| Всего сделок | %d |\n", m.TotalTrades)
	fmt.Fprintf(&b, "| Выигрышных / проигрышных | %d / %d |\n", m.Wins, m.Losses)
	fmt.Fprintf(&b, "| Win rate | %.2f%% |\n", m.WinRate*100)
	fmt.Fprintf(&b, "| Profit factor | %.3f |\n", m.ProfitFactor)
	fmt.Fprintf(&b, "| Gross profit / loss | %.2f / %.2f |\n", m.GrossProfit, m.GrossLoss)
	fmt.Fprintf(&b, "| Expectancy | %.2f |\n", m.Expectancy)
	fmt.Fprintf(&b, "| Sortino | %.3f |\n", m.Sortino)
	fmt.Fprintf(&b, "| Лучшая / худшая сделка | %.2f / %.2f |\n\n", m.BestTrade, m.WorstTrade)

	b.WriteString("## Разбивка по бумагам (OOS)\n\n")
	b.WriteString("| Тикер | Сделок | PF | Net PnL % | Max DD % | Win rate | Параметры-победителя |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for _, e := range s.Entries {
		if e.Skipped {
			fmt.Fprintf(&b, "| %s | — | — | — | — | — | %s |\n", e.Ticker, e.Note)
			continue
		}
		note := paramSummary(e.Params)
		if e.Note != "" {
			note = e.Note
		}
		fmt.Fprintf(&b, "| %s | %d | %.3f | %.2f%% | %.2f%% | %.2f%% | %s |\n",
			e.Ticker, e.Trades, e.ProfitFactor, e.NetPnLPct*100, e.MaxDrawdownPct*100, e.WinRate*100, note)
	}
	return b.String()
}

// paramSummary renders the handful of params that matter most for a quick scan.
func paramSummary(rows []backtest.ParamLine) string {
	keys := []string{"EMAPeriod", "SLMult", "TakeProfitRR", "CooldownBars", "DailyTrendPeriod"}
	idx := make(map[string]string, len(rows))
	for _, r := range rows {
		idx[r.Name] = r.Value
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := idx[k]; ok {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return strings.Join(parts, ", ")
}
```

- [ ] **Step 2: Verify the package compiles**

Run: `go build ./internal/service/backtest/`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/service/backtest/basket.go
git commit -m "feat(backtest): render basket walk-forward report"
```

---

## Task 6: Wire `-basket` mode into the CLI

**Files:**
- Modify: `cmd/backtest/main.go`

Add `-basket` and `-grid-dir` flags, dispatch in `run()`, implement `runBasket` and
`splitTickers`. The basket reuses the per-ticker walk-forward exactly like
`runCalibration` does with `-test-months`, looping over tickers and pooling the OOS trades.

- [ ] **Step 1: Add `strings` to the import block**

In `cmd/backtest/main.go`, add `"strings"` to the standard-library imports (alphabetically after `"path/filepath"`):

```go
	"os"
	"path/filepath"
	"strings"
	"time"
```

- [ ] **Step 2: Add the two flags in `main()`**

After the `explain` flag declaration, add:

```go
		basket  = flag.String("basket", "", "basket mode: comma-separated tickers; calibrates each on the early window and pools OOS trades (ignores -ticker)")
		gridDir = flag.String("grid-dir", "data/params", "basket mode: directory holding <lower-ticker>/momentum_grid.json")
```

- [ ] **Step 3: Pass them into `run()`**

Change the `run(...)` call in `main()` to append the two new args (before `*outDir`... keep order consistent with the new signature):

```go
	if err := run(*ticker, *strategyName, interval, *months, *cash, *fraction, *commission,
		*paramsPath, *calibrate, *metric, *minTrades, *testMonths, *outDir, *refresh, *explain,
		*basket, *gridDir); err != nil {
		log.Fatalf("backtest: %v", err)
	}
```

- [ ] **Step 4: Update `run()` signature and add the basket dispatch**

Change the signature:

```go
func run(ticker, strategyName string, interval enum.Interval, months int, cash, fraction, commission float64,
	paramsPath, calibratePath, metric string, minTrades, testMonths int, outDir string, refresh bool, explain string,
	basketCSV, gridDir string,
) error {
```

Replace the existing top of `run()` (the `if ticker == ""` / mutual-exclusion block down to the gRPC client creation) so the client is built first and the basket branch returns early:

```go
	if paramsPath != "" && calibratePath != "" {
		return fmt.Errorf("-params and -calibrate are mutually exclusive")
	}

	token, err := loadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(apiAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}
	ctx := context.Background()

	if basketCSV != "" {
		if strategyName != "momentum" {
			return fmt.Errorf("-basket currently supports -strategy momentum only")
		}
		return runBasket(ctx, client, splitTickers(basketCSV), interval, months,
			cash, fraction, commission, metric, minTrades, testMonths, gridDir, outDir, refresh)
	}

	if ticker == "" {
		return fmt.Errorf("-ticker is required")
	}
```

Then delete the now-duplicated `token`/`client`/`ctx` setup that previously followed the `ticker == ""` check (the binding `switch` keeps using the existing `client` and `ctx`).

- [ ] **Step 5: Add `splitTickers` and `runBasket` at the end of `main.go`**

```go
// splitTickers parses a comma-separated ticker list, trimming blanks.
func splitTickers(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// runBasket calibrates each ticker on the early window, runs the winner on the shared
// OOS tail, and pools the tail trades across tickers into one statistically meaningful
// sample. Each ticker keeps its own calibrated params; the pool is validation only.
func runBasket(ctx context.Context, client grpcclient.GrpcClient, tickers []string, interval enum.Interval,
	months int, cash, fraction, commission float64, metric string, minTrades, testMonths int,
	gridDir, outDir string, refresh bool,
) error {
	if testMonths <= 0 {
		return fmt.Errorf("-basket requires -test-months > 0 (walk-forward OOS window)")
	}
	if len(tickers) == 0 {
		return fmt.Errorf("-basket: no tickers parsed")
	}

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	to := time.Now()
	from := to.AddDate(0, -months, 0)
	boundary := to.AddDate(0, -testMonths, 0)
	dailyFrom := from.AddDate(-1, 0, 0)
	periodDays := to.Sub(from).Hours() / 24
	testDays := to.Sub(boundary).Hours() / 24
	gridDays := periodDays - testDays

	var pooled []domain.Trade
	var summary svc.BasketSummary

	for _, ticker := range tickers {
		entry := svc.BasketEntry{Ticker: ticker}

		gridPath := filepath.Join(gridDir, strings.ToLower(ticker), "momentum_grid.json")
		raw, err := os.ReadFile(gridPath)
		if err != nil {
			entry.Skipped, entry.Note = true, fmt.Sprintf("нет грида (%s)", gridPath)
			summary.Entries = append(summary.Entries, entry)
			continue
		}
		phases, err := svc.ParsePhases(raw)
		if err != nil {
			entry.Skipped, entry.Note = true, fmt.Sprintf("грид невалиден: %v", err)
			summary.Entries = append(summary.Entries, entry)
			continue
		}

		share, err := resolveShare(ctx, client, ticker)
		if err != nil {
			entry.Skipped, entry.Note = true, fmt.Sprintf("инструмент не найден: %v", err)
			summary.Entries = append(summary.Entries, entry)
			continue
		}

		candles, err := provider.Load(ctx, ticker, share.ID, interval, from, to, refresh)
		if err != nil {
			return fmt.Errorf("%s: load candles: %w", ticker, err)
		}
		dailyCandles, err := provider.Load(ctx, ticker, share.ID, enum.Day1, dailyFrom, to, refresh)
		if err != nil {
			return fmt.Errorf("%s: load daily: %w", ticker, err)
		}

		binding := svc.MomentumLookupOrGeneric(ticker)
		cfg := domain.Config{InitialCash: cash, Fraction: fraction, Commission: commission, Lot: share.Lot}
		gridCandles, bestCandles := svc.SplitByTime(candles, boundary)
		gridDaily, bestDaily := svc.SplitByTime(dailyCandles, boundary)

		results, err := svc.RunPhases(binding, phases, gridCandles, gridDaily, cfg, metric, minTrades, gridDays, nil)
		if err != nil {
			return fmt.Errorf("%s: calibrate: %w", ticker, err)
		}
		if len(results) == 0 {
			entry.Skipped, entry.Note = true, "калибровка не дала комбинаций"
			summary.Entries = append(summary.Entries, entry)
			continue
		}

		best := results[0].Params
		res := domain.Run(binding.Build(best), bestCandles, bestDaily, cfg)
		m := domain.Compute(res, res.BarsInMarket, len(res.Equity), testDays)

		entry.Trades = m.TotalTrades
		entry.ProfitFactor = m.ProfitFactor
		entry.NetPnL = m.NetPnL
		entry.NetPnLPct = m.NetPnLPct
		entry.MaxDrawdownPct = m.MaxDrawdownPct
		entry.WinRate = m.WinRate
		entry.Params = svc.ParamRows(best)
		if m.TotalTrades == 0 {
			entry.Note = "нет OOS-сделок"
		}
		summary.Entries = append(summary.Entries, entry)
		pooled = append(pooled, res.Trades...)
		fmt.Printf("basket %s: OOS trades=%d PF=%.3f net=%.2f\n", ticker, m.TotalTrades, m.ProfitFactor, m.NetPnL)
	}

	summary.Pooled = svc.PooledMetrics(pooled)

	basketDir := filepath.Join(outDir, "basket")
	if err := os.MkdirAll(basketDir, 0o755); err != nil {
		return fmt.Errorf("mkdir basket dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	base := filepath.Join(basketDir, fmt.Sprintf("basket_momentum_%s", stamp))
	if err := writeFile(base+".md", svc.RenderBasketMarkdown(metric, summary, boundary, to)); err != nil {
		return err
	}
	if err := writeFile(base+"_trades.csv", domain.RenderTradesCSV(pooled)); err != nil {
		return err
	}
	fmt.Printf("basket report: %s.md (pooled trades=%d, PF=%.3f)\n", base, summary.Pooled.TotalTrades, summary.Pooled.ProfitFactor)
	return nil
}
```

- [ ] **Step 6: Build and vet**

Run: `go build ./cmd/backtest/ && go vet ./cmd/backtest/ ./internal/service/backtest/`
Expected: no output, exit 0.

- [ ] **Step 7: Commit**

```bash
git add cmd/backtest/main.go
git commit -m "feat(backtest): walk-forward basket mode pooling per-stock OOS trades"
```

---

## Task 7: Full test suite + e2e basket run

**Files:** none (verification only)

- [ ] **Step 1: Run the full backtest package tests**

Run: `go test ./internal/service/backtest/... ./internal/service/trading_strategy/momentum/...`
Expected: PASS (`ok` for each package).

- [ ] **Step 2: e2e basket run (hits the network; requires `T_BANK` token)**

Run:
```bash
go run ./cmd/backtest -strategy momentum \
  -basket "AFKS,PLZL,RUAL,YDEX,SBER,GAZP,NVTK,MDMG" \
  -months 24 -test-months 6 -metric profit_factor -min-trades 18 \
  -out ./reports
```
Expected: one `basket <TICKER>: OOS trades=...` line per ticker, then
`basket report: reports/basket/basket_momentum_<stamp>.md (pooled trades=N, PF=...)`
with N > 0. MDMG may show fewer trades (shorter history) — that is expected, not a failure.

- [ ] **Step 3: Eyeball the report**

Run: `cat reports/basket/basket_momentum_*.md`
Expected: a "Пул сделок (агрегат OOS)" table with a non-zero trade count and a
per-ticker breakdown row for each of the 8 tickers (skipped ones show a note).

- [ ] **Step 4: Report results to the user**

Summarize the pooled PF / win-rate / expectancy and the per-ticker breakdown. Do NOT
commit report artifacts unless the user asks (reports are run outputs). Stop here — the
user will decide whether to hardcode each ticker's calibrated winner into its package.

---

## Notes for the executor

- The momentum strategy is long-only; all params live in `core.Params` (see
  `internal/service/trading_strategy/momentum/strategy/core/core.go:33`).
- Grid field names in the JSON must match `core.Params` field names exactly — the
  calibrator sets them by reflection and errors on unknown fields.
- `backtest.Compute(Result{Trades: trades}, 0, 0, 0)` is intentional in `PooledMetrics`:
  zero `barsInMarket`/`totalBars`/`periodDays` zero out exposure and CAGR; an empty
  equity slice zeros drawdown; `FinalEquity==InitialCash==0` zeros NetPnL.
- Do not change the existing single-ticker / `-calibrate` paths; the basket branch is
  additive and returns before them.
