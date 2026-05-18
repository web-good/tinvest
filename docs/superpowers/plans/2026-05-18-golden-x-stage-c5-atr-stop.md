# Golden X Stage C5 — ATR Stop Suggestion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Each task is self-contained and runs in its own context window. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an ATR-based stop suggestion line to every buy-tier row in Golden X Telegram alerts. Format: `  Stop: <price> (−<pct>%)`. Stop level is `lastClose − k × ATR(14)` where `k = 2.0` for Dividend (Gold) and `k = 1.5` for Growth. Both 🟡 and 🟢 buy tiers. Buy side only. Dedup, sell side, RSIList unchanged.

**Architecture:** One new pure helper `ATR` in the existing `pkg/indicators` package (joins `VolumeConfirmed`). Two thin Golden-X policy wrappers (`kForKind`, `stopFromATR`) in a new file `golden_x/stop.go`. A new carrier struct `dto.Stop` in the Golden-X dto package. Same parallel-map orchestration pattern used by `trends`, `thresholds`, `sellThresholds`, `divergences`, `volumesConfirmed`: add `stops map[string]dto.Stop` keyed by `share.ID`, passed as the 9th argument to `notif.Trade(...)`. The renderer concatenates a new `stopLine(stops[id])` after the RSI line. Empty `dto.Stop{}` renders as nothing.

**Tech Stack:** Go 1.25, stdlib only. No new dependencies.

**Reference spec:** `docs/superpowers/specs/2026-05-18-golden-x-stage-c5-atr-stop-design.md`.

**Working directory:** `/home/oleg/GolandProjects/tinvest`. Branch: `feature/golden-x-stage-a` (continues from C4 at commit `13d4c86`).

---

## File Structure

**New files (5):**

- `pkg/indicators/atr.go` — single exported function `ATR(highs, lows, closes []float64, period int) float64`. Pure, no side effects. Wilder's smoothing. Returns the last ATR value, or `0` on insufficient/inconsistent input.
- `pkg/indicators/atr_test.go` — table-driven tests for `ATR`.
- `internal/service/trading_strategy/golden_x/dto/stop.go` — `Stop` struct carrying the absolute stop price and the distance % from the last close.
- `internal/service/trading_strategy/golden_x/stop.go` — two pure policy wrappers used by `trade.go`: `kForKind(dto.StrategyKind) float64` and `stopFromATR(lastClose, atr, k float64) dto.Stop`. Empty `dto.Stop{}` on degenerate inputs.
- `internal/service/trading_strategy/golden_x/stop_test.go` — table-driven tests for the two wrappers.

**Modified files (3):**

- `internal/service/trading_strategy/golden_x/trade.go` — add three constants (`goldenXATRPeriod`, `goldenXATRMultiplierGold`, `goldenXATRMultiplierGrowth`); add `stops := make(map[string]dto.Stop)` to the per-tick state; inside the existing `if buyTier != tierNone { ... }` block, after the volume check, extract `highs/lows/closes` from `closed`, call `indicators.ATR`, populate `stops[share.ID]` via `stopFromATR(...)` and `kForKind(in.Kind)`; pass `stops` as the 9th argument to `notif.Trade(...)`.
- `internal/service/trading_strategy/golden_x/notification/notifications.go` — `Trade(...)` signature grows by one trailing parameter `stops map[string]dto.Stop`; new helper `stopLine(dto.Stop) string` (sibling to `divergenceBadge`, `volumeBadge`); buy-loop calls `b.WriteString(stopLine(stops[id]))` immediately after the RSI line.
- `internal/service/trading_strategy/golden_x/notification/notifications_test.go` — update the 11 existing `Trade(...)` call sites to add a trailing `nil` for `stops`; append 3 new render tests (stop-line renders with correct format, stop-line absent when `dto.Stop{}` empty, sell row has no stop line).

**Untouched (explicitly verified in the spec):** `internal/service/instrument/atr/*` (intraday-shaped legacy service; not relevant to weekly Golden X), `internal/service/trading_strategy/golden_x/dto/trade.go` (public API unchanged — multiplier is picked from `in.Kind`), `internal/app/app.go` (no wiring change), `dedup.go`, `divergence.go`, `notification/rsi_by_shares.go`, `internal/service_provider/`, `internal/domain/info.go`.

---

## Task 1: `ATR` pure helper (TDD)

**Files:**
- Create: `pkg/indicators/atr.go`
- Create: `pkg/indicators/atr_test.go`

The `pkg/indicators` package already exists from Stage C4 (it currently contains `volume.go` and `volume_test.go`). `ATR` is the second exported function in the same package.

- [ ] **Step 1: Write the failing tests**

Create `pkg/indicators/atr_test.go`:

```go
package indicators

import (
	"math"
	"testing"
)

func TestATR(t *testing.T) {
	tests := []struct {
		name    string
		highs   []float64
		lows    []float64
		closes  []float64
		period  int
		want    float64
		tol     float64
	}{
		{
			name:   "constant TR=2: ATR equals 2 regardless of close drift",
			highs:  []float64{12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12},
			lows:   []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11},
			period: 14,
			want:   2.0,
			tol:    1e-9,
		},
		{
			name:   "len equals period — insufficient (need period+1)",
			highs:  []float64{12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12},
			lows:   []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11},
			period: 14,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "len equals period+1 — returns the seed (single TR)",
			highs:  []float64{12, 12, 12, 12},
			lows:   []float64{10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11},
			period: 3,
			want:   2.0,
			tol:    1e-9,
		},
		{
			name:   "Wilder smoothing nudges ATR toward larger TR",
			// Period 3 so the example is small enough to hand-check.
			// TRs (i=1..6): 2, 2, 2, 6, 2, 2.
			//   Seed ATR_3 = mean(2,2,2) = 2.
			//   ATR_4 = (2*2 + 6) / 3 = 10/3 ≈ 3.3333.
			//   ATR_5 = (3.3333*2 + 2) / 3 ≈ 2.8889.
			//   ATR_6 = (2.8889*2 + 2) / 3 ≈ 2.5926.
			highs:  []float64{12, 12, 12, 12, 16, 12, 12},
			lows:   []float64{10, 10, 10, 10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11, 11, 11, 11},
			period: 3,
			want:   2.5925925925925926,
			tol:    1e-9,
		},
		{
			name:   "period <= 0 is silent zero",
			highs:  []float64{12, 12, 12},
			lows:   []float64{10, 10, 10},
			closes: []float64{11, 11, 11},
			period: 0,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "negative period is silent zero",
			highs:  []float64{12, 12, 12},
			lows:   []float64{10, 10, 10},
			closes: []float64{11, 11, 11},
			period: -3,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "length mismatch (highs shorter) is silent zero",
			highs:  []float64{12, 12, 12},
			lows:   []float64{10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11},
			period: 3,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "length mismatch (closes shorter) is silent zero",
			highs:  []float64{12, 12, 12, 12},
			lows:   []float64{10, 10, 10, 10},
			closes: []float64{11, 11, 11},
			period: 3,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "empty input is silent zero",
			highs:  nil,
			lows:   nil,
			closes: nil,
			period: 14,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "gap up beats high-low: TR uses |high - prevClose|",
			// Bar 0: high=10, low=8, close=9.
			// Bar 1: high=12, low=11, close=11.5. TR = max(12-11, |12-9|, |11-9|) = 3.
			// Period 1 means seed ATR_1 = TR_1 = 3.
			highs:  []float64{10, 12},
			lows:   []float64{8, 11},
			closes: []float64{9, 11.5},
			period: 1,
			want:   3.0,
			tol:    1e-9,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ATR(tc.highs, tc.lows, tc.closes, tc.period)
			if math.Abs(got-tc.want) > tc.tol {
				t.Fatalf("ATR = %v, want %v (tol %v)", got, tc.want, tc.tol)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail with "undefined"**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./pkg/indicators/... -run TestATR -v 2>&1 | head -40`

Expected: build error / FAIL — `undefined: ATR`. (The package compiles `volume.go` fine; only `ATR` is missing.)

- [ ] **Step 3: Write the minimal implementation**

Create `pkg/indicators/atr.go`:

```go
package indicators

import "math"

// ATR returns Wilder's Average True Range over the input bar series.
//
// Inputs must be aligned (highs[i], lows[i], closes[i] all describe the same
// bar). Returns 0 when period <= 0, when the three slices are not the same
// length, or when len(closes) < period+1 — the insufficient-history rule is
// silent (no error), mirroring VolumeConfirmed.
//
// Algorithm:
//   - True Range at bar i (i >= 1):
//       TR_i = max(High_i - Low_i, |High_i - Close_{i-1}|, |Low_i - Close_{i-1}|)
//   - Seed ATR_{period} = mean(TR_1 .. TR_{period}).
//   - For i > period: ATR_i = (ATR_{i-1} * (period - 1) + TR_i) / period.
//   - Returns ATR at the last index.
func ATR(highs, lows, closes []float64, period int) float64 {
	if period <= 0 {
		return 0
	}
	n := len(closes)
	if len(highs) != n || len(lows) != n {
		return 0
	}
	if n < period+1 {
		return 0
	}

	trueRange := func(i int) float64 {
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		return math.Max(hl, math.Max(hc, lc))
	}

	// Seed: mean of the first `period` TR values (bars 1..period).
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trueRange(i)
	}
	atr := sum / float64(period)

	// Wilder smoothing for every subsequent bar.
	for i := period + 1; i < n; i++ {
		atr = (atr*float64(period-1) + trueRange(i)) / float64(period)
	}
	return atr
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./pkg/indicators/... -run TestATR -v`

Expected: PASS — all 10 subtests green.

- [ ] **Step 5: Run the full indicators package tests (regression on VolumeConfirmed)**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./pkg/indicators/... -v`

Expected: PASS — both `TestATR` (10 cases) and `TestVolumeConfirmed` (9 cases) green.

- [ ] **Step 6: Build the whole repo (sanity check)**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./...`

Expected: no output, exit code 0. (No call sites yet — `ATR` is isolated.)

- [ ] **Step 7: Commit**

```bash
cd /home/oleg/GolandProjects/tinvest && git add pkg/indicators && git commit -m "$(cat <<'EOF'
feat(indicators): add Wilder ATR helper in pkg/indicators

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: `dto.Stop`, `kForKind`, `stopFromATR` (TDD)

**Files:**
- Create: `internal/service/trading_strategy/golden_x/dto/stop.go`
- Create: `internal/service/trading_strategy/golden_x/stop.go`
- Create: `internal/service/trading_strategy/golden_x/stop_test.go`

This task introduces the policy wrappers but does not yet wire them into `trade.go` or `notifications.go`. The wrappers are pure — fully unit-testable in isolation. The `dto.Stop` struct, `kForKind`, and `stopFromATR` are added together because the wrappers depend on the struct and the test depends on both.

- [ ] **Step 1: Write the failing tests**

Create `internal/service/trading_strategy/golden_x/stop_test.go`:

```go
package golden_x

import (
	"math"
	"testing"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func TestKForKind(t *testing.T) {
	tests := []struct {
		name string
		kind dto.StrategyKind
		want float64
	}{
		{"Gold (Dividend) -> 2.0", dto.StrategyKindDividend, 2.0},
		{"Growth -> 1.5", dto.StrategyKindGrowth, 1.5},
		{"unknown zero-value defaults to Gold's multiplier", dto.StrategyKind(0), 2.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := kForKind(tc.kind)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("kForKind(%v) = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}

func TestStopFromATR(t *testing.T) {
	tests := []struct {
		name      string
		lastClose float64
		atr       float64
		k         float64
		wantPrice float64
		wantPct   float64
		wantEmpty bool
		tol       float64
	}{
		{
			name:      "Gold normal: 2410.5 minus 2.0*30",
			lastClose: 2410.5,
			atr:       30.0,
			k:         2.0,
			wantPrice: 2350.5,
			wantPct:   60.0 / 2410.5 * 100,
			tol:       1e-6,
		},
		{
			name:      "Growth normal: 100 minus 1.5*4",
			lastClose: 100.0,
			atr:       4.0,
			k:         1.5,
			wantPrice: 94.0,
			wantPct:   6.0,
			tol:       1e-9,
		},
		{
			name:      "atr zero -> empty Stop",
			lastClose: 100.0,
			atr:       0.0,
			k:         2.0,
			wantEmpty: true,
		},
		{
			name:      "atr negative -> empty Stop",
			lastClose: 100.0,
			atr:       -1.0,
			k:         2.0,
			wantEmpty: true,
		},
		{
			name:      "lastClose zero -> empty Stop",
			lastClose: 0.0,
			atr:       4.0,
			k:         2.0,
			wantEmpty: true,
		},
		{
			name:      "stop price would be <= 0 (k*atr exceeds close) -> empty Stop",
			lastClose: 5.0,
			atr:       4.0,
			k:         2.0,
			wantEmpty: true,
		},
		{
			name:      "stop price exactly zero -> empty Stop",
			lastClose: 8.0,
			atr:       4.0,
			k:         2.0,
			wantEmpty: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stopFromATR(tc.lastClose, tc.atr, tc.k)
			if tc.wantEmpty {
				if got != (dto.Stop{}) {
					t.Fatalf("expected empty Stop{}, got %+v", got)
				}
				return
			}
			if math.Abs(got.Price-tc.wantPrice) > tc.tol {
				t.Fatalf("Price = %v, want %v", got.Price, tc.wantPrice)
			}
			if math.Abs(got.DistancePct-tc.wantPct) > tc.tol {
				t.Fatalf("DistancePct = %v, want %v", got.DistancePct, tc.wantPct)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/... -run 'TestKForKind|TestStopFromATR' -v 2>&1 | head -40`

Expected: build error / FAIL — `undefined: kForKind`, `undefined: stopFromATR`, `undefined: dto.Stop`.

- [ ] **Step 3: Create the `dto.Stop` carrier**

Create `internal/service/trading_strategy/golden_x/dto/stop.go`:

```go
package dto

// Stop is the ATR-based stop suggestion attached to a buy-tier row.
// The zero value means "not computed" — the notification renderer treats it
// as nothing, the same way it treats empty Thresholds and SellThresholds.
type Stop struct {
	Price       float64
	DistancePct float64
}
```

- [ ] **Step 4: Create the Golden-X policy wrappers**

Create `internal/service/trading_strategy/golden_x/stop.go`:

```go
package golden_x

import "tinvest/internal/service/trading_strategy/golden_x/dto"

// kForKind returns the ATR multiplier appropriate for the given strategy kind.
// Gold (Dividend) holds longer and needs wider stops; Growth exits sooner and
// uses tighter stops. Unknown kinds fall back to Gold — defensive only; the
// production code paths construct the enum at the call site.
func kForKind(kind dto.StrategyKind) float64 {
	if kind == dto.StrategyKindGrowth {
		return goldenXATRMultiplierGrowth
	}
	return goldenXATRMultiplierGold
}

// stopFromATR composes a dto.Stop from the last close, ATR value, and
// multiplier. Returns the zero Stop{} when atr or lastClose are non-positive,
// or when the computed stop level would be <= 0 (degenerate).
func stopFromATR(lastClose, atr, k float64) dto.Stop {
	if atr <= 0 || lastClose <= 0 {
		return dto.Stop{}
	}
	price := lastClose - k*atr
	if price <= 0 {
		return dto.Stop{}
	}
	return dto.Stop{
		Price:       price,
		DistancePct: (lastClose - price) / lastClose * 100,
	}
}
```

Note: this file references `goldenXATRMultiplierGold` and `goldenXATRMultiplierGrowth`, which do not exist yet. They are declared in Task 3 Step 3 (constants alongside `volumeMultiplier`). Until then, `stop.go` will fail to compile **only if you try to build before completing Task 3**. The tests in this task target only `kForKind` and `stopFromATR`, but they invoke functions that reference the missing constants — so the test step (Step 5 below) requires the constants to be present. To keep TDD honest, add the constants up front, here, in `stop.go` itself, then move them to `trade.go` in Task 3. That way each task is independently testable.

**Update**: instead of cross-task coupling, declare the two constants at the top of `stop.go` for now. Task 3 Step 3 will move them to `trade.go` and Task 3 Step 4 will replace these declarations with the moved ones.

Prepend to `stop.go` (above the function definitions, after the import):

```go
// goldenXATRMultiplierGold is the ATR stop multiplier for the Dividend
// (long-hold) strategy: wider stops survive deeper weekly noise.
const goldenXATRMultiplierGold = 2.0

// goldenXATRMultiplierGrowth is the stop multiplier for Growth — tighter,
// since the strategy exits sooner on RSI overheats.
const goldenXATRMultiplierGrowth = 1.5
```

Final file content of `internal/service/trading_strategy/golden_x/stop.go`:

```go
package golden_x

import "tinvest/internal/service/trading_strategy/golden_x/dto"

// goldenXATRMultiplierGold is the ATR stop multiplier for the Dividend
// (long-hold) strategy: wider stops survive deeper weekly noise.
const goldenXATRMultiplierGold = 2.0

// goldenXATRMultiplierGrowth is the stop multiplier for Growth — tighter,
// since the strategy exits sooner on RSI overheats.
const goldenXATRMultiplierGrowth = 1.5

// kForKind returns the ATR multiplier appropriate for the given strategy kind.
// Gold (Dividend) holds longer and needs wider stops; Growth exits sooner and
// uses tighter stops. Unknown kinds fall back to Gold — defensive only; the
// production code paths construct the enum at the call site.
func kForKind(kind dto.StrategyKind) float64 {
	if kind == dto.StrategyKindGrowth {
		return goldenXATRMultiplierGrowth
	}
	return goldenXATRMultiplierGold
}

// stopFromATR composes a dto.Stop from the last close, ATR value, and
// multiplier. Returns the zero Stop{} when atr or lastClose are non-positive,
// or when the computed stop level would be <= 0 (degenerate).
func stopFromATR(lastClose, atr, k float64) dto.Stop {
	if atr <= 0 || lastClose <= 0 {
		return dto.Stop{}
	}
	price := lastClose - k*atr
	if price <= 0 {
		return dto.Stop{}
	}
	return dto.Stop{
		Price:       price,
		DistancePct: (lastClose - price) / lastClose * 100,
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/... -run 'TestKForKind|TestStopFromATR' -v`

Expected: PASS — `TestKForKind` (3 cases) and `TestStopFromATR` (7 cases) green.

- [ ] **Step 6: Run all golden_x package tests (regression)**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/... -v 2>&1 | tail -40`

Expected: PASS — pre-existing tests (dedup, divergence, percentile, rsi, trend_filter, candle, notifications) and the new stop tests all green.

- [ ] **Step 7: Build the whole repo (sanity check)**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./...`

Expected: no output, exit code 0.

- [ ] **Step 8: Commit**

```bash
cd /home/oleg/GolandProjects/tinvest && git add internal/service/trading_strategy/golden_x/dto/stop.go internal/service/trading_strategy/golden_x/stop.go internal/service/trading_strategy/golden_x/stop_test.go && git commit -m "$(cat <<'EOF'
feat(golden_x): add Stop dto and ATR stop policy wrappers

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Wire stop into `Trade()` and render in notifications

This task changes `notif.Trade(...)` signature, which means **both** `notifications.go` and the call site in `trade.go` must move together — the package will not build between Step 3 and Step 6. Commit only at the end (Step 10), as a single logical unit (mirrors the pattern from Stage C2 Task 5+6, Stage C3 Task 2, and Stage C4 Task 2).

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/notification/notifications.go`
- Modify: `internal/service/trading_strategy/golden_x/notification/notifications_test.go`
- Modify: `internal/service/trading_strategy/golden_x/trade.go`
- Modify: `internal/service/trading_strategy/golden_x/stop.go` (move two constants out — see Step 3)

- [ ] **Step 1: Update existing notification test calls to new arity**

Edit `internal/service/trading_strategy/golden_x/notification/notifications_test.go`. Each of the 11 existing `Trade(...)` calls gains one trailing `nil` argument for the new `stops` parameter. The line numbers below reflect the file post-C4.

Line 28 (in `TestTrade_AdaptiveTiersAndThresholdSuffix`):

```go
// BEFORE:
got := Trade(info, nil, dto.StrategyKindGrowth, trends, thresholds, nil, nil, nil)
// AFTER:
got := Trade(info, nil, dto.StrategyKindGrowth, trends, thresholds, nil, nil, nil, nil)
```

Line 48 (in `TestTrade_NoThresholdsRendersNoEmojiOrSuffix`):

```go
// BEFORE:
got := Trade(info, nil, dto.StrategyKindDividend, nil, nil, nil, nil, nil)
// AFTER:
got := Trade(info, nil, dto.StrategyKindDividend, nil, nil, nil, nil, nil, nil)
```

Line 74 (in `TestTrade_RendersSellSectionGold`):

```go
// BEFORE:
got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, nil, sellThresholds, nil, nil)
// AFTER:
got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, nil, sellThresholds, nil, nil, nil)
```

Line 105 (in `TestTrade_RendersSellSectionGrowth`):

```go
// BEFORE:
got := Trade(buyInfo, sellInfo, dto.StrategyKindGrowth, nil, nil, sellThresholds, nil, nil)
// AFTER:
got := Trade(buyInfo, sellInfo, dto.StrategyKindGrowth, nil, nil, sellThresholds, nil, nil, nil)
```

Line 128 (in `TestTrade_BuyAndSellTogether`):

```go
// BEFORE:
got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, thresholds, sellThresholds, nil, nil)
// AFTER:
got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, thresholds, sellThresholds, nil, nil, nil)
```

Line 171 (in `TestTrade_RendersBullishDivergenceBadge`):

```go
// BEFORE:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, nil)
// AFTER:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, nil, nil)
```

Line 186 (in `TestTrade_NoBadgeWhenNotDivergent`):

```go
// BEFORE:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, nil)
// AFTER:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, nil, nil)
```

Line 205 (in `TestTrade_BadgeAfterTrendMark`):

```go
// BEFORE:
got := Trade(buyInfo, nil, dto.StrategyKindGrowth, trends, thresholds, nil, divergences, nil)
// AFTER:
got := Trade(buyInfo, nil, dto.StrategyKindGrowth, trends, thresholds, nil, divergences, nil, nil)
```

Line 223 (in `TestTrade_VolumeBadgeAppendedAfterDivergence`):

```go
// BEFORE:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, volumes)
// AFTER:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, volumes, nil)
```

Line 240 (in `TestTrade_VolumeBadgeAbsentWhenNotConfirmed`):

```go
// BEFORE:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, nil)
// AFTER:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, nil, nil)
```

Line 259 (in `TestTrade_VolumeBadgeWithoutDivergence`):

```go
// BEFORE:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, volumes)
// AFTER:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, volumes, nil)
```

- [ ] **Step 2: Append three new render tests**

Append to `internal/service/trading_strategy/golden_x/notification/notifications_test.go`:

```go
func TestTrade_StopLineRendersAfterRSI(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-stop", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-stop": {P5: 24, P15: 31},
	}
	stops := map[string]dto.Stop{
		"share-stop": {Price: 2410.5, DistancePct: 6.2},
	}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, nil, stops)

	if !strings.Contains(got, "<b>Stop:</b> 2410.50 (−6.2%)") {
		t.Errorf("expected '<b>Stop:</b> 2410.50 (−6.2%%)', got:\n%s", got)
	}
	// Stop line must appear AFTER the RSI line of the same share.
	rsiIdx := strings.Index(got, "RSI Value:20")
	stopIdx := strings.Index(got, "<b>Stop:</b>")
	if rsiIdx < 0 || stopIdx < 0 || stopIdx < rsiIdx {
		t.Errorf("expected Stop line after RSI line, got order RSI=%d Stop=%d in:\n%s", rsiIdx, stopIdx, got)
	}
}

func TestTrade_StopLineAbsentWhenEmpty(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-nostop", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-nostop": {P5: 24, P15: 31},
	}
	// Empty stops map.
	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, nil, map[string]dto.Stop{})

	if strings.Contains(got, "Stop:") {
		t.Errorf("expected no Stop line, got:\n%s", got)
	}
}

func TestTrade_StopLineAbsentOnSellRow(t *testing.T) {
	buyInfo := domain.NewInfo()
	sellInfo := domain.NewInfo()
	sellInfo.WriteToMap("sell-id", domain.Item{InstrumentName: "Phosagro", RSIValue: 85})

	sellThresholds := map[string]dto.SellThresholds{
		"sell-id": {P80: 60, P90: 70, P95: 80},
	}
	// Even if the map is wrongly populated for a sell-side ID, the renderer
	// must not surface a Stop line — stop is buy-only by construction.
	stops := map[string]dto.Stop{
		"sell-id": {Price: 1234.5, DistancePct: 9.9},
	}

	got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, nil, sellThresholds, nil, nil, stops)

	if strings.Contains(got, "Stop:") {
		t.Errorf("expected no Stop line on sell row, got:\n%s", got)
	}
}
```

- [ ] **Step 3: Update `Trade(...)` signature, add `stopLine` helper**

Edit `internal/service/trading_strategy/golden_x/notification/notifications.go`. Two changes: the function signature gains a parameter and the buy-row loop calls `stopLine`; a new helper `stopLine` is appended next to `volumeBadge`.

Replace the existing `Trade(...)` declaration (lines 12-52, the full body that ends with `return b.String()`) with:

```go
func Trade(
	buyInfo *domain.Info,
	sellInfo *domain.Info,
	kind dto.StrategyKind,
	trends map[string]dto.TrendStatus,
	thresholds map[string]dto.Thresholds,
	sellThresholds map[string]dto.SellThresholds,
	divergences map[string]bool,
	volumesConfirmed map[string]bool,
	stops map[string]dto.Stop,
) string {
	b := strings.Builder{}
	if medal := kind.Medal(); medal != "" {
		b.WriteString(medal + "\n\n")
	}

	if buyInfo != nil && len(buyInfo.Items()) > 0 {
		b.WriteString("<u><b>Акции находящиеся в локальных минимумах:</b></u>\n\n\n<code>")
		for id, log := range buyInfo.Items() {
			trendMark := trends[id].Mark()
			if trendMark != "" {
				trendMark = " " + trendMark
			}
			b.WriteString("• <b>Акция:</b> " + log.InstrumentName + tierEmoji(log.RSIValue, thresholds[id]) + trendMark + divergenceBadge(divergences[id]) + volumeBadge(volumesConfirmed[id]) + "\n")
			b.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + thresholdSuffix(thresholds[id]) + "\n")
			b.WriteString(stopLine(stops[id]))
			b.WriteString("\n")
		}
		b.WriteString("</code>\n\n")
	}

	if sellInfo != nil && len(sellInfo.Items()) > 0 {
		b.WriteString("<u><b>Акции находящиеся в локальных максимумах:</b></u>\n\n\n<code>")
		for id, log := range sellInfo.Items() {
			b.WriteString("• <b>Акция:</b> " + log.InstrumentName + sellTierEmoji(log.RSIValue, sellThresholds[id]) + "\n")
			b.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + sellThresholdSuffix(sellThresholds[id]) + "\n")
			b.WriteString("\n")
		}
		b.WriteString("</code>")
	}

	return b.String()
}
```

(Differences from the existing function: one new last parameter `stops map[string]dto.Stop`, and one new line `b.WriteString(stopLine(stops[id]))` placed between the RSI line and the trailing newline. Sell loop and footer are untouched.)

Then, immediately after the existing `volumeBadge` definition (before `tierEmoji`), append:

```go
// stopLine renders the ATR-derived stop suggestion as its own line:
// "  <b>Stop:</b> <price> (−<pct>%)\n". The zero-value dto.Stop{} renders
// "" — same convention as Thresholds and SellThresholds, keeps the indicator
// additive and silent on insufficient history.
func stopLine(s dto.Stop) string {
	if s.Price == 0 && s.DistancePct == 0 {
		return ""
	}
	return fmt.Sprintf("  <b>Stop:</b> %.2f (−%.1f%%)\n", s.Price, s.DistancePct)
}
```

(`fmt` is already imported in `notifications.go:5`. No new import.)

- [ ] **Step 4: Run notification tests — should pass standalone**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/notification/... -v`

Expected: PASS for all tests in the notification package (existing 12 + 3 new `TestTrade_StopLine*`).

The wider repo will not yet build — `trade.go` still calls `notif.Trade(...)` with 8 args. That is fixed in Step 6.

- [ ] **Step 5: Move constants from `stop.go` to `trade.go`**

In Task 2 Step 4 we declared `goldenXATRMultiplierGold` and `goldenXATRMultiplierGrowth` inside `stop.go` so the wrappers were independently testable. Now we move them next to the other Golden-X tuning constants and add the new `goldenXATRPeriod`.

**(5a) Remove the two constants from `stop.go`.** Edit `internal/service/trading_strategy/golden_x/stop.go`. Delete the two const declarations and their comments (the block between `import "tinvest/internal/service/trading_strategy/golden_x/dto"` and `func kForKind`). The file after edit:

```go
package golden_x

import "tinvest/internal/service/trading_strategy/golden_x/dto"

// kForKind returns the ATR multiplier appropriate for the given strategy kind.
// Gold (Dividend) holds longer and needs wider stops; Growth exits sooner and
// uses tighter stops. Unknown kinds fall back to Gold — defensive only; the
// production code paths construct the enum at the call site.
func kForKind(kind dto.StrategyKind) float64 {
	if kind == dto.StrategyKindGrowth {
		return goldenXATRMultiplierGrowth
	}
	return goldenXATRMultiplierGold
}

// stopFromATR composes a dto.Stop from the last close, ATR value, and
// multiplier. Returns the zero Stop{} when atr or lastClose are non-positive,
// or when the computed stop level would be <= 0 (degenerate).
func stopFromATR(lastClose, atr, k float64) dto.Stop {
	if atr <= 0 || lastClose <= 0 {
		return dto.Stop{}
	}
	price := lastClose - k*atr
	if price <= 0 {
		return dto.Stop{}
	}
	return dto.Stop{
		Price:       price,
		DistancePct: (lastClose - price) / lastClose * 100,
	}
}
```

**(5b) Add the three constants to `trade.go`.** Immediately after the existing `volumeMultiplier` declaration block (the spec block that ends with the line `const volumeMultiplier = 1.5`), append:

```go
// goldenXATRPeriod is Wilder's standard ATR period applied on the weekly TF
// closed-candle stream used for buy-side stop suggestions.
const goldenXATRPeriod = 14

// goldenXATRMultiplierGold is the ATR stop multiplier for the Dividend
// (long-hold) strategy: wider stops survive deeper weekly noise.
const goldenXATRMultiplierGold = 2.0

// goldenXATRMultiplierGrowth is the stop multiplier for Growth — tighter,
// since the strategy exits sooner on RSI overheats.
const goldenXATRMultiplierGrowth = 1.5
```

- [ ] **Step 6: Update `trade.go` — map, ATR call, notification arity**

Edit `internal/service/trading_strategy/golden_x/trade.go`. Three localized changes (the imports already contain `tinvest/pkg/indicators` from C4 — no import change).

**(6a) Add the `stops` map declaration.** Find the existing block:

```go
		volumesConfirmed := make(map[string]bool)
```

And insert one line below it:

```go
		volumesConfirmed := make(map[string]bool)
		stops := make(map[string]dto.Stop)
```

**(6b) Extend the `if buyTier != tierNone` block.** Find the existing block (it starts with `if buyTier != tierNone {` and ends with the volume check):

```go
		if buyTier != tierNone {
			lows := lowsAlignedToRSI(closed, share.RSILength, rsiSeries)
			if len(lows) > divergenceLookbackWeeks {
				lows = lows[len(lows)-divergenceLookbackWeeks:]
			}
			rsiTail := rsiSeries
			if len(rsiTail) > divergenceLookbackWeeks {
				rsiTail = rsiTail[len(rsiTail)-divergenceLookbackWeeks:]
			}
			if bullishDivergence(lows, rsiTail, divergenceFractalK) {
				divergences[share.ID] = true
			}

			volumes := make([]int64, len(closed))
			for i, c := range closed {
				volumes[i] = c.Volume
			}
			if indicators.VolumeConfirmed(volumes, volumeSMALookback, volumeMultiplier) {
				volumesConfirmed[share.ID] = true
			}
		}
```

Replace with (only the trailing addition before the closing `}` is new):

```go
		if buyTier != tierNone {
			lows := lowsAlignedToRSI(closed, share.RSILength, rsiSeries)
			if len(lows) > divergenceLookbackWeeks {
				lows = lows[len(lows)-divergenceLookbackWeeks:]
			}
			rsiTail := rsiSeries
			if len(rsiTail) > divergenceLookbackWeeks {
				rsiTail = rsiTail[len(rsiTail)-divergenceLookbackWeeks:]
			}
			if bullishDivergence(lows, rsiTail, divergenceFractalK) {
				divergences[share.ID] = true
			}

			volumes := make([]int64, len(closed))
			for i, c := range closed {
				volumes[i] = c.Volume
			}
			if indicators.VolumeConfirmed(volumes, volumeSMALookback, volumeMultiplier) {
				volumesConfirmed[share.ID] = true
			}

			highs := make([]float64, len(closed))
			lowsF := make([]float64, len(closed))
			closes := make([]float64, len(closed))
			for i, c := range closed {
				highs[i] = utils.CombinePrice(c.High.Units, c.High.Nano)
				lowsF[i] = utils.CombinePrice(c.Low.Units, c.Low.Nano)
				closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
			}
			if atrValue := indicators.ATR(highs, lowsF, closes, goldenXATRPeriod); atrValue > 0 {
				lastClose := closes[len(closes)-1]
				stops[share.ID] = stopFromATR(lastClose, atrValue, kForKind(in.Kind))
			}
		}
```

(Naming note: the inner float-`lows` slice is renamed `lowsF` to avoid shadowing the outer `lows []float64` already used for divergence detection. Both serve different roles — `lows` is trimmed to `divergenceLookbackWeeks`; `lowsF` is the full closed-candle Low series. Keeping them as separate variables is intentional.)

**(6c) Update the `notif.Trade(...)` call.** Find the existing line:

```go
		msg := notif.Trade(buyInfo, sellInfo, in.Kind, trends, thresholds, sellThresholds, divergences, volumesConfirmed)
```

Replace with:

```go
		msg := notif.Trade(buyInfo, sellInfo, in.Kind, trends, thresholds, sellThresholds, divergences, volumesConfirmed, stops)
```

- [ ] **Step 7: Build the package and run all golden_x tests**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./...`

Expected: no output, exit code 0.

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/... ./pkg/indicators/... -v 2>&1 | tail -80`

Expected: PASS for all tests (pre-existing + the 3 stop-policy tests from Task 2 + the 3 new render tests from Step 2 + `TestATR` and `TestVolumeConfirmed`).

- [ ] **Step 8: Run the full repo test pass (sanity)**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./... 2>&1 | grep -E "^(ok|FAIL|---)" | sort -u`

Expected:
- `pkg/client/telegram` continues to fail with the **pre-existing** `non-constant format string` build error (out of scope, same caveat documented in C2, B, C3, and C4 plans).
- All other packages either `ok` or `[no test files]`. No new failures.

- [ ] **Step 9: Vet check**

Run: `cd /home/oleg/GolandProjects/tinvest && go vet ./internal/service/trading_strategy/golden_x/... ./pkg/indicators/...`

Expected: empty output.

- [ ] **Step 10: Commit**

```bash
cd /home/oleg/GolandProjects/tinvest && git add -A && git status && git commit -m "$(cat <<'EOF'
feat(golden_x): ATR stop suggestion line in buy alerts

Per-share weekly ATR-based stop: when a share is in a buy tier (🟡 or 🟢),
compute lastClose − k×ATR(14) on the same closed weekly candles already
used for RSI and percentiles. k = 2.0 for Dividend (Gold), 1.5 for Growth.
Render a new line under the RSI value:

  <b>Stop:</b> 2410.50 (−6.2%)

Both Gold and Growth instances. Buy side only. Dedup is unchanged — the
stop is a pure annotation tied to the existing buy-tier signal.

ATR helper lives in pkg/indicators (joins VolumeConfirmed). The policy
wrappers (kForKind, stopFromATR) and the Stop dto live in the strategy
package.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected output of `git status` after commit: clean tree.

---

## Task 4: End-to-end verification (manual)

This is a manual verification step. It cannot be automated — it requires a live Tinkoff Invest API connection.

- [ ] **Step 1: Run the dev binary**

In a terminal with `env/local.env` configured (Tinkoff token, Telegram chat IDs):

```bash
cd /home/oleg/GolandProjects/tinvest && go run ./cmd/main
```

Expected: app boots, runs the Growth instance once on the `* * * * *` schedule.

To also run the Gold instance once in dev, temporarily uncomment / add a one-off goroutine in `runDev` mirroring the existing Growth block but with `Kind: goldenx.StrategyKindDividend` and `ShareList: *a.collection.GoldInstruments`. Revert before merging.

- [ ] **Step 2: Inspect a Telegram buy-section message**

If at least one share is in a buy tier (🟡 or 🟢), expect each buy-section row to look like:

```
🥈

Акции находящиеся в локальных минимумах:

• Акция: Yandex 🟢 ✅ 📈 🔊
  RSI Value:22  (p5=24.0, p15=31.0)
  Stop: 4587.20 (−7.3%)

• Акция: Polyus 🟡 🚫
  RSI Value:28  (p5=24.0, p15=31.0)
  Stop: 12950.00 (−5.4%)
```

Confirm:
- A new `  Stop: <price> (−<pct>%)` line follows the `RSI Value:` line on **every** buy row (both 🟡 and 🟢).
- `<pct>` for a Growth instance is meaningfully smaller than for the same volatility profile in Gold (k=1.5 vs k=2.0). Easiest sanity check: pick the same instrument (e.g. Yandex) in two runs and compare distances.
- Sell-section rows have no `Stop:` line.
- `RSIList` (informational dump) has no `Stop:` line.
- The stop line uses the en-dash hyphen `−` (U+2212) followed by the percentage, formatted to one decimal place. Price is formatted to two decimal places.

- [ ] **Step 3: Sanity-check log volume**

Search the dev log for any ATR-related panics or errors. There should be none — `ATR` returns `0` on bad input, `stopFromATR` returns `dto.Stop{}` on bad input, and the renderer is silent on the zero value.

- [ ] **Step 4: Finish the branch**

The branch `feature/golden-x-stage-a` now carries Stages A + C1 + C2 + B + C3 + C4 + C5 — seven logical stages, ready for review/merge as one batch. Manual E2E for B, C3, C4, and now C5 can be co-verified in the same dev run.

After E2E passes, invoke `superpowers:finishing-a-development-branch` to decide between local merge and PR.

---

## Self-Review

(Done by the planner — left here so the executor can sanity-check.)

**1. Spec coverage:**

- Stop = `close − k × ATR(14)` formula → Task 3 Step 6b (`closes[len-1] - k*ATR`).
- k = 2.0 Gold, 1.5 Growth → Task 2 Step 4 (`kForKind`).
- Wilder ATR algorithm → Task 1 Step 3 (seed = SMA of first `period` TRs, then Wilder smoothing).
- Same closed-weekly data, no new RPC → Task 3 Step 6b extracts highs/lows/closes from `closed` (already in scope at that point in `trade.go`).
- Both 🟡 and 🟢 tiers → placement inside `if buyTier != tierNone { ... }` (covers both); no per-tier gating.
- Both Gold and Growth → `dto.Trade` untouched, multiplier picks from `in.Kind`.
- Buy side only → block in `trade.go` is inside the `if buyTier != tierNone` branch; sell loop in `notifications.go` is left as-is.
- Insufficient history is silent → `ATR` returns 0 on bad input (Task 1 Step 3), `stopFromATR` returns `dto.Stop{}` on bad input (Task 2 Step 4), `stopLine` returns `""` on the zero value (Task 3 Step 3). Three layers of silence, matching the divergence/volume pattern.
- Notification format `  <b>Stop:</b> <price> (−<pct>%)\n` → Task 3 Step 3 (`fmt.Sprintf("  <b>Stop:</b> %.2f (−%.1f%%)\n", ...)`).
- Helper in `pkg/indicators` → Task 1 creates `pkg/indicators/atr.go`.
- `internal/service/instrument/atr/*` untouched → no task references it.
- Dedup unchanged → no task touches `dedup.go`; `ShouldAlert` keys on tier, not on stop.
- RSIList unchanged → no task touches `rsi_by_shares.go`.

**2. Placeholder scan:** every step contains either complete code blocks or exact shell commands. No TBD / TODO / "similar to …" placeholders. The Task 2 Step 4 design-rationale prose explains *why* constants are temporarily in `stop.go` and confirms Task 3 Step 5 moves them — not a TODO, just a planning note.

**3. Type consistency:**

- `ATR(highs, lows, closes []float64, period int) float64` — declared in Task 1 Step 3, invoked identically in Task 3 Step 6b.
- `dto.Stop{Price, DistancePct}` — declared in Task 2 Step 3, used in Task 2 Step 4 (`stopFromATR`), Task 3 Step 3 (`stopLine`), Task 3 Step 6a (`stops := make(map[string]dto.Stop)`), and Task 3 Step 6b (`stops[share.ID] = stopFromATR(...)`).
- `kForKind(dto.StrategyKind) float64` — declared in Task 2 Step 4, invoked in Task 3 Step 6b.
- `stopFromATR(float64, float64, float64) dto.Stop` — declared in Task 2 Step 4, invoked in Task 3 Step 6b.
- `goldenXATRPeriod = 14`, `goldenXATRMultiplierGold = 2.0`, `goldenXATRMultiplierGrowth = 1.5` — multipliers declared in `stop.go` at Task 2 Step 4, **moved** to `trade.go` at Task 3 Step 5 alongside the new `goldenXATRPeriod`. Used in Task 3 Step 6b.
- Notification call-site arity: 9 arguments — matches the 9-parameter signature in Task 3 Step 3.

**4. Risks for the executor:**

- The package is not buildable between Task 3 Step 3 and Task 3 Step 6. The plan explicitly groups Steps 1–6 into a single commit at Step 10. Do not commit partial state in between.
- The variable name collision (`lows` outer vs `lowsF` inner) in `trade.go` is intentional: the outer `lows` slice is the trimmed `divergenceLookbackWeeks` series for divergence, while the inner `lowsF` is the full closed-candle Low series for ATR. Resist the temptation to "deduplicate" them.
- `pkg/client/telegram` build failure is pre-existing and out of scope.
- If `go test ./...` shows any **new** failure beyond `pkg/client/telegram`, stop and inspect — that's a regression.
- The U+2212 minus sign `−` in the format string and tests is intentional (matches the spec). Do not normalize to ASCII hyphen-minus.
