# Golden X Stage C2 — Adaptive RSI Tiers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the hard-coded `31/35/40` RSI thresholds with per-share `p5/p15` percentiles computed from up to 200 closed weekly RSI values for both Gold and Growth instances.

**Architecture:** One `GetCandles` RPC per share per tick. RSI series and EMA200 are both computed locally from the same candle slice. Tiers are decided by `tierFromAdaptive(rsi, p5, p15)`. Shares with `<100` closed-week RSI values are skipped (consistent with C1).

**Tech Stack:** Go 1.25, Tinkoff Invest gRPC, existing `pkg/client/grpc.MarketDataServiceClient`. No new dependencies.

**Reference spec:** `docs/superpowers/specs/2026-05-16-golden-x-stage-c2-adaptive-tiers-design.md`.

**Working directory:** `/home/oleg/GolandProjects/tinvest`. Branch: `feature/golden-x-stage-a` (continues from C1).

---

## Task 1: `dto.Thresholds` struct

**Files:**
- Create: `internal/service/trading_strategy/golden_x/dto/thresholds.go`

- [ ] **Step 1: Create the struct file**

Create `internal/service/trading_strategy/golden_x/dto/thresholds.go`:

```go
package dto

// Thresholds are the per-share adaptive RSI percentile boundaries that drive
// the Golden X buy tiers (replaces the static 31/35/40 buckets).
type Thresholds struct {
	P5  float64
	P15 float64
}
```

- [ ] **Step 2: Build succeeds**

Run: `go build ./...`
Expected: no output, exit code 0.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/golden_x/dto/thresholds.go
git commit -m "$(cat <<'EOF'
feat(golden_x): add dto.Thresholds for adaptive RSI tiers

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: `percentile()` — R-7 linear interpolation

**Files:**
- Create: `internal/service/trading_strategy/golden_x/percentile.go`
- Create: `internal/service/trading_strategy/golden_x/percentile_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/service/trading_strategy/golden_x/percentile_test.go`:

```go
package golden_x

import (
	"math"
	"testing"
)

func TestPercentile_R7(t *testing.T) {
	tests := []struct {
		name   string
		sorted []float64
		p      float64
		want   float64
	}{
		{
			// R-7 reference: percentile([1..100], 5) = 1 + 0.05*99 = 5.95
			name:   "p5 of 1..100",
			sorted: rangeFloat(1, 100),
			p:      5,
			want:   5.95,
		},
		{
			// R-7 reference: percentile([1..100], 15) = 1 + 0.15*99 = 15.85
			name:   "p15 of 1..100",
			sorted: rangeFloat(1, 100),
			p:      15,
			want:   15.85,
		},
		{
			name:   "single element returns itself",
			sorted: []float64{42},
			p:      5,
			want:   42,
		},
		{
			name:   "all equal returns the common value",
			sorted: []float64{5, 5, 5, 5, 5},
			p:      50,
			want:   5,
		},
		{
			name:   "p=0 returns the smallest",
			sorted: []float64{10, 20, 30, 40},
			p:      0,
			want:   10,
		},
		{
			name:   "p=100 returns the largest",
			sorted: []float64{10, 20, 30, 40},
			p:      100,
			want:   40,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := percentile(tc.sorted, tc.p)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("percentile(%v, %v) = %v, want %v", tc.sorted, tc.p, got, tc.want)
			}
		})
	}
}

// rangeFloat returns [from, from+1, …, to] inclusive.
func rangeFloat(from, to int) []float64 {
	out := make([]float64, 0, to-from+1)
	for i := from; i <= to; i++ {
		out = append(out, float64(i))
	}
	return out
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/golden_x/ -run TestPercentile_R7`
Expected: FAIL — `undefined: percentile`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/service/trading_strategy/golden_x/percentile.go`:

```go
package golden_x

import "math"

// percentile returns the R-7 (linear-interpolation) percentile of a sorted
// (ascending) slice. This is the default method in numpy.percentile and Excel.
// p is in [0, 100]. Empty input returns 0.
func percentile(sortedAsc []float64, p float64) float64 {
	n := len(sortedAsc)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sortedAsc[0]
	}
	rank := (p / 100.0) * float64(n-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sortedAsc[lo]
	}
	weight := rank - float64(lo)
	return sortedAsc[lo]*(1-weight) + sortedAsc[hi]*weight
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/trading_strategy/golden_x/ -run TestPercentile_R7 -v`
Expected: PASS for all 6 subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/golden_x/percentile.go internal/service/trading_strategy/golden_x/percentile_test.go
git commit -m "$(cat <<'EOF'
feat(golden_x): add R-7 percentile helper for adaptive tiers

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: `tierFromAdaptive` and `adaptiveThresholds`

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/percentile.go`
- Modify: `internal/service/trading_strategy/golden_x/percentile_test.go`

- [ ] **Step 1: Add failing tests**

Append to `internal/service/trading_strategy/golden_x/percentile_test.go`:

```go
func TestTierFromAdaptive(t *testing.T) {
	tests := []struct {
		name string
		rsi  float64
		p5   float64
		p15  float64
		want alertTier
	}{
		{"rsi strictly below p5 → Green", 20, 24, 31, tierGreen},
		{"rsi == p5 → Yellow (strict <)", 24, 24, 31, tierYellow},
		{"rsi between p5 and p15 → Yellow", 28, 24, 31, tierYellow},
		{"rsi == p15 → None (strict <)", 31, 24, 31, tierNone},
		{"rsi above p15 → None", 40, 24, 31, tierNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tierFromAdaptive(tc.rsi, tc.p5, tc.p15)
			if got != tc.want {
				t.Fatalf("tierFromAdaptive(%v, %v, %v) = %v, want %v", tc.rsi, tc.p5, tc.p15, got, tc.want)
			}
		})
	}
}

func TestAdaptiveThresholds(t *testing.T) {
	// percentile([1..100], 5)  = 5.95
	// percentile([1..100], 15) = 15.85
	rsi := rangeFloat(1, 100)
	got := adaptiveThresholds(rsi)
	if math.Abs(got.P5-5.95) > 1e-9 {
		t.Errorf("P5 = %v, want 5.95", got.P5)
	}
	if math.Abs(got.P15-15.85) > 1e-9 {
		t.Errorf("P15 = %v, want 15.85", got.P15)
	}
}

func TestAdaptiveThresholds_DoesNotMutateInput(t *testing.T) {
	// Input may arrive in any order; helper must sort defensively without
	// scrambling the caller's slice.
	in := []float64{50, 10, 30, 20, 40}
	original := append([]float64(nil), in...)
	_ = adaptiveThresholds(in)
	for i := range in {
		if in[i] != original[i] {
			t.Fatalf("input mutated at %d: got %v, want %v", i, in[i], original[i])
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/golden_x/ -run "TestTierFromAdaptive|TestAdaptiveThresholds" -v`
Expected: FAIL — `undefined: tierFromAdaptive`, `undefined: adaptiveThresholds`.

- [ ] **Step 3: Add implementations**

Append to `internal/service/trading_strategy/golden_x/percentile.go`:

```go
import (
	"sort"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

// tierFromAdaptive maps the last closed-week RSI to a Golden X buy tier using
// the share's own historical percentiles. Comparisons are strict — equality
// at the boundary falls into the looser tier (Yellow at p5, None at p15).
func tierFromAdaptive(rsi, p5, p15 float64) alertTier {
	switch {
	case rsi < p5:
		return tierGreen
	case rsi < p15:
		return tierYellow
	default:
		return tierNone
	}
}

// adaptiveThresholds computes P5 and P15 over an unordered slice of historical
// RSI values. The input is not mutated; a sorted copy is taken internally.
func adaptiveThresholds(rsiSeries []float64) dto.Thresholds {
	sorted := append([]float64(nil), rsiSeries...)
	sort.Float64s(sorted)
	return dto.Thresholds{
		P5:  percentile(sorted, 5),
		P15: percentile(sorted, 15),
	}
}
```

Update the existing `import "math"` line in `percentile.go` to combine with the new `sort` and `dto` imports:

```go
import (
	"math"
	"sort"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/golden_x/ -run "TestTierFromAdaptive|TestAdaptiveThresholds" -v`
Expected: PASS for all subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/golden_x/percentile.go internal/service/trading_strategy/golden_x/percentile_test.go
git commit -m "$(cat <<'EOF'
feat(golden_x): add tierFromAdaptive and adaptiveThresholds helpers

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: `computeRSISeries` (Wilder RSI)

**Files:**
- Create: `internal/service/trading_strategy/golden_x/rsi.go`
- Create: `internal/service/trading_strategy/golden_x/rsi_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/service/trading_strategy/golden_x/rsi_test.go`:

```go
package golden_x

import (
	"math"
	"testing"
)

func TestComputeRSISeries_AlternatingFixture(t *testing.T) {
	// Closes alternating up/down by 1: [10, 11, 10, 11, 10, 11, 10, 11, 10, 11].
	// Period = 3. Hand-traced expected values:
	//
	// Changes: [+1, -1, +1, -1, +1, -1, +1, -1, +1]
	//          (9 changes, indexes 1..9 of the closes array)
	//
	// Seed at index 3 (after 3 changes: +1, -1, +1):
	//   avgGain = (1+0+1)/3 = 0.6667
	//   avgLoss = (0+1+0)/3 = 0.3333
	//   RS = 2.0  → RSI = 100 - 100/3 ≈ 66.67
	//
	// Index 4 (change = -1):  g=0, l=1
	//   avgGain = (0.6667*2 + 0)/3 ≈ 0.4444
	//   avgLoss = (0.3333*2 + 1)/3 ≈ 0.5556
	//   RS ≈ 0.8 → RSI ≈ 44.44
	//
	// Index 5 (change = +1):  g=1, l=0
	//   avgGain = (0.4444*2 + 1)/3 ≈ 0.6296
	//   avgLoss = (0.5556*2 + 0)/3 ≈ 0.3704
	//   RS ≈ 1.7 → RSI ≈ 62.96
	closes := []float64{10, 11, 10, 11, 10, 11, 10, 11, 10, 11}
	got := computeRSISeries(closes, 3)

	if len(got) != len(closes) {
		t.Fatalf("len = %d, want %d", len(got), len(closes))
	}
	// Positions before period have no RSI.
	for i := 0; i < 3; i++ {
		if got[i] != 0 {
			t.Errorf("pos %d: got %v, want 0", i, got[i])
		}
	}
	// Check the seed and the next two Wilder steps within 0.05 (rounding-friendly).
	checkClose := func(t *testing.T, pos int, want float64) {
		t.Helper()
		if math.Abs(got[pos]-want) > 0.05 {
			t.Errorf("pos %d: got %v, want %v ±0.05", pos, got[pos], want)
		}
	}
	checkClose(t, 3, 66.67)
	checkClose(t, 4, 44.44)
	checkClose(t, 5, 62.96)
}

func TestComputeRSISeries_TooFewClosesReturnsZeroes(t *testing.T) {
	got := computeRSISeries([]float64{1, 2, 3}, 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, v := range got {
		if v != 0 {
			t.Errorf("pos %d: got %v, want 0", i, v)
		}
	}
}

func TestComputeRSISeries_NoLossesBoundary(t *testing.T) {
	// Strictly increasing input: avgLoss stays at 0. Match the existing
	// instrument/rsi calculator's edge-case behavior (rs=1 → RSI=50) for
	// behavioral parity. See internal/service/instrument/rsi/calculate.go:83.
	closes := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := computeRSISeries(closes, 3)
	if math.Abs(got[3]-50.0) > 0.01 {
		t.Errorf("pos 3: got %v, want 50.00 (boundary parity with instrument/rsi)", got[3])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/golden_x/ -run TestComputeRSISeries -v`
Expected: FAIL — `undefined: computeRSISeries`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/service/trading_strategy/golden_x/rsi.go`:

```go
package golden_x

import "math"

// computeRSISeries returns the Wilder RSI value for each position in closes.
// Positions before index `period` are zero (no RSI defined yet). The result
// has the same length as the input.
//
// Boundary parity note: when accumulated avgLoss is exactly 0 (a strict run of
// gains across the entire warmup), this implementation matches the existing
// internal/service/instrument/rsi/calculate.go behavior of falling back to
// rs=1 → RSI=50, rather than the mathematically correct RSI=100. This keeps
// signal behavior consistent for any share that briefly hits the boundary.
func computeRSISeries(closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	if len(closes) <= period {
		return out
	}

	p := float64(period)
	var avgGain, avgLoss float64
	// Seed: SMA of the first `period` price changes (closes[1] − closes[0],
	// closes[2] − closes[1], … closes[period] − closes[period-1]).
	for i := 1; i <= period; i++ {
		ch := closes[i] - closes[i-1]
		if ch > 0 {
			avgGain += ch
		} else {
			avgLoss += -ch
		}
	}
	avgGain /= p
	avgLoss /= p
	out[period] = wilderRSI(avgGain, avgLoss)

	for i := period + 1; i < len(closes); i++ {
		ch := closes[i] - closes[i-1]
		var g, l float64
		if ch > 0 {
			g = ch
		} else {
			l = -ch
		}
		avgGain = (avgGain*(p-1) + g) / p
		avgLoss = (avgLoss*(p-1) + l) / p
		out[i] = wilderRSI(avgGain, avgLoss)
	}
	return out
}

// wilderRSI converts smoothed average gain/loss into an RSI value, rounding to
// two decimal places to match the existing calculator's quantization.
func wilderRSI(avgGain, avgLoss float64) float64 {
	rs := 1.0
	if avgLoss != 0 {
		rs = avgGain / avgLoss
	}
	rsi := 100 - 100/(1+rs)
	return math.Round(rsi*100) / 100
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/golden_x/ -run TestComputeRSISeries -v`
Expected: PASS for all 3 subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/golden_x/rsi.go internal/service/trading_strategy/golden_x/rsi_test.go
git commit -m "$(cat <<'EOF'
feat(golden_x): add local Wilder RSI series computation

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Drop the `tierBrown` constant

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/dedup.go`
- Modify: `internal/service/trading_strategy/golden_x/dedup_test.go`

- [ ] **Step 1: Update the test first (red)**

Edit `internal/service/trading_strategy/golden_x/dedup_test.go`. Replace the existing `TestAlertState_ShouldAlert` table with:

```go
func TestAlertState_ShouldAlert(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	tests := []struct {
		name string
		tier alertTier
		want bool
	}{
		{"первый Yellow — алерт", tierYellow, true},
		{"повторный Yellow — нет", tierYellow, false},
		{"переход Yellow→Green — алерт", tierGreen, true},
		{"повторный Green — нет", tierGreen, false},
		{"откат Green→None (RSI выше p15) — нет (молчим)", tierNone, false},
		{"снова Yellow после None — алерт", tierYellow, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.ShouldAlert(id, tc.tier)
			if got != tc.want {
				t.Fatalf("ShouldAlert(%v) = %v, want %v", tc.tier, got, tc.want)
			}
		})
	}
}
```

Also delete the entire `TestTierFromRSI` function — its target (`tierFromRSI`) is being removed. Keep `TestAlertState_IndependentShares` as-is.

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/service/trading_strategy/golden_x/ -run "TestAlertState" -v`
Expected: FAIL — compilation error or test failure, depending on whether `tierFromRSI` is still referenced (it is, by `trade.go`, so build fails first).

Note: if compilation fails because `trade.go` still calls `tierFromRSI`, that's expected — Task 6 fixes the call site. Continue to Step 3.

- [ ] **Step 3: Remove `tierBrown` and `tierFromRSI` from `dedup.go`**

Edit `internal/service/trading_strategy/golden_x/dedup.go` — replace the `alertTier` block and remove `tierFromRSI`:

```go
package golden_x

import "sync"

type alertTier int

const (
	tierNone alertTier = iota
	tierYellow
	tierGreen
)

// alertState tracks the last tier emitted per shareID and decides whether
// a new alert should be sent. An alert fires only when the tier changes
// AND the new tier is not tierNone (RSI above p15 means "silent reset").
type alertState struct {
	mu   sync.Mutex
	last map[string]alertTier
}

func newAlertState() *alertState {
	return &alertState{last: make(map[string]alertTier)}
}

// ShouldAlert returns true if a fresh alert with `tier` should be emitted
// for `shareID`. On a tierNone input it resets the stored state silently.
func (s *alertState) ShouldAlert(shareID string, tier alertTier) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.last[shareID]
	if tier == tierNone {
		s.last[shareID] = tier
		return false
	}
	if prev == tier {
		return false
	}
	s.last[shareID] = tier
	return true
}
```

- [ ] **Step 4: Build (still expected to fail in trade.go)**

Run: `go build ./internal/service/trading_strategy/golden_x/ 2>&1 | head -20`
Expected: errors in `trade.go` referencing `tierFromRSI`. This is fine — Task 6 fixes it.

For now we are NOT committing this task standalone — the package won't build. Skip the commit step.

> **Continue directly to Task 6.** Stage tasks 5 and 6 are committed together as one logical change ("switch to adaptive tiers").

---

## Task 6: Refactor `trade.go`, update notifications, drop `rsiInstrument`

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/trade.go`
- Modify: `internal/service/trading_strategy/golden_x/types.go`
- Modify: `internal/service_provider/service.go`
- Modify: `internal/service/trading_strategy/golden_x/notification/notifications.go`
- Modify: `internal/service/trading_strategy/golden_x/notification/rsi_by_shares.go`
- Modify: `internal/service/trading_strategy/golden_x/notification/notifications_test.go`

- [ ] **Step 1: Update notification — `Trade` to accept thresholds**

Edit `internal/service/trading_strategy/golden_x/notification/notifications.go`. Replace its body with:

```go
package notification

import (
	"fmt"
	"strconv"
	"strings"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func Trade(
	info *domain.Info,
	kind dto.StrategyKind,
	trends map[string]dto.TrendStatus,
	thresholds map[string]dto.Thresholds,
) string {
	notifyMessageBuilder := strings.Builder{}
	if medal := kind.Medal(); medal != "" {
		notifyMessageBuilder.WriteString(medal + "\n\n")
	}

	notifyMessageBuilder.WriteString("<u><b>Акции находящиеся в локальных минимумах:</b></u>\n\n\n<code>")
	for id, log := range info.Items() {
		trendMark := trends[id].Mark()
		if trendMark != "" {
			trendMark = " " + trendMark
		}
		notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + tierEmoji(log.RSIValue, thresholds[id]) + trendMark + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + thresholdSuffix(thresholds[id]) + "\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}

// tierEmoji renders the colored circle based on adaptive thresholds. Empty
// Thresholds (zero value, e.g. for shares filtered out before we could compute
// them) renders no emoji — the row still appears with the raw RSI value.
func tierEmoji(rsi float64, th dto.Thresholds) string {
	if th.P5 == 0 && th.P15 == 0 {
		return ""
	}
	switch {
	case rsi < th.P5:
		return " 🟢"
	case rsi < th.P15:
		return " 🟡"
	default:
		return ""
	}
}

// thresholdSuffix renders the percentile annotation appended to the RSI line,
// e.g. "  (p5=24, p15=31)". Empty Thresholds renders nothing.
func thresholdSuffix(th dto.Thresholds) string {
	if th.P5 == 0 && th.P15 == 0 {
		return ""
	}
	return fmt.Sprintf("  (p5=%.1f, p15=%.1f)", th.P5, th.P15)
}
```

- [ ] **Step 2: Update notification — `RSIList` to accept thresholds**

Edit `internal/service/trading_strategy/golden_x/notification/rsi_by_shares.go`. Replace its body with:

```go
package notification

import (
	"strconv"
	"strings"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func RSIList(info *domain.Info, kind dto.StrategyKind, thresholds map[string]dto.Thresholds) string {
	notifyMessageBuilder := strings.Builder{}
	if medal := kind.Medal(); medal != "" {
		notifyMessageBuilder.WriteString(medal + "\n\n")
	}

	notifyMessageBuilder.WriteString("🧾\n<u><b>Промежуточные значения RSI и % цены к див доход:</b></u>\n\n\n<code>")

	for id, log := range info.Items() {
		notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + tierEmoji(log.RSIValue, thresholds[id]) + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Length:</b>" + strconv.Itoa(log.RSILength) + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + thresholdSuffix(thresholds[id]) + "\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
```

- [ ] **Step 3: Update notification tests**

Replace `internal/service/trading_strategy/golden_x/notification/notifications_test.go` with:

```go
package notification

import (
	"strings"
	"testing"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func TestTrade_AdaptiveTiersAndThresholdSuffix(t *testing.T) {
	info := domain.NewInfo()
	info.WriteToMap("green-id", domain.Item{InstrumentName: "Yandex", RSIValue: 20})
	info.WriteToMap("yellow-id", domain.Item{InstrumentName: "Polyus", RSIValue: 28})
	info.WriteToMap("notrend-id", domain.Item{InstrumentName: "Sber", RSIValue: 20})

	trends := map[string]dto.TrendStatus{
		"green-id":   dto.TrendWith,
		"yellow-id":  dto.TrendAgainst,
		"notrend-id": dto.TrendUnknown,
	}
	thresholds := map[string]dto.Thresholds{
		"green-id":   {P5: 24, P15: 31},
		"yellow-id":  {P5: 24, P15: 31},
		"notrend-id": {P5: 24, P15: 31},
	}

	got := Trade(info, dto.StrategyKindGrowth, trends, thresholds)

	if !strings.Contains(got, "Yandex 🟢 ✅") {
		t.Errorf("expected 'Yandex 🟢 ✅', got:\n%s", got)
	}
	if !strings.Contains(got, "Polyus 🟡 🚫") {
		t.Errorf("expected 'Polyus 🟡 🚫', got:\n%s", got)
	}
	if !strings.Contains(got, "Sber 🟢\n") {
		t.Errorf("expected 'Sber 🟢' without trend mark, got:\n%s", got)
	}
	if !strings.Contains(got, "(p5=24.0, p15=31.0)") {
		t.Errorf("expected threshold suffix '(p5=24.0, p15=31.0)', got:\n%s", got)
	}
}

func TestTrade_NoThresholdsRendersNoEmojiOrSuffix(t *testing.T) {
	info := domain.NewInfo()
	info.WriteToMap("any-id", domain.Item{InstrumentName: "Lukoil", RSIValue: 28})

	got := Trade(info, dto.StrategyKindDividend, nil, nil)

	if !strings.Contains(got, "🥇") {
		t.Errorf("expected gold medal, got:\n%s", got)
	}
	if strings.Contains(got, "🟢") || strings.Contains(got, "🟡") {
		t.Errorf("expected no tier emoji when thresholds missing, got:\n%s", got)
	}
	if strings.Contains(got, "p5=") {
		t.Errorf("expected no threshold suffix, got:\n%s", got)
	}
}

func TestRSIList_RendersAdaptiveTier(t *testing.T) {
	info := domain.NewInfo()
	info.WriteToMap("any-id", domain.Item{InstrumentName: "Lukoil", RSILength: 11, RSIValue: 28})

	thresholds := map[string]dto.Thresholds{
		"any-id": {P5: 24, P15: 31},
	}

	got := RSIList(info, dto.StrategyKindDividend, thresholds)

	if !strings.Contains(got, "Lukoil 🟡") {
		t.Errorf("expected 'Lukoil 🟡' in RSIList, got:\n%s", got)
	}
	if !strings.Contains(got, "(p5=24.0, p15=31.0)") {
		t.Errorf("expected threshold suffix in RSIList, got:\n%s", got)
	}
}
```

- [ ] **Step 4: Update `types.go` — drop `rsiInstrument`**

Replace `internal/service/trading_strategy/golden_x/types.go` with:

```go
package golden_x

import (
	"context"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type GoldenX interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
	state                       *alertState
}

func NewService(
	instrumentsServiceClient grpc.InstrumentsServiceClient,
	marketDataServiceGrpcClient grpc.MarketDataServiceClient,
	tgClient telegram.Client,
) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
		state:                       newAlertState(),
	}
}
```

- [ ] **Step 5: Update `service_provider/service.go` — drop the RSI argument**

Edit `internal/service_provider/service.go`. Inside `GetGoldenXTradingService()` replace the constructor call:

```go
serviceProvider.service.goldenXTradingService = golden_x.NewService(
	grpcClient.InstrumentsServiceClient(),
	grpcClient.MarketDataServiceClient(),
	tgClient,
)
```

(Just drop the `serviceProvider.RSI()` line — the other three arguments remain in the same order.)

- [ ] **Step 6: Refactor `trade.go` — single fetch + adaptive tiers**

Replace `internal/service/trading_strategy/golden_x/trade.go` with:

```go
package golden_x

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	notif "tinvest/internal/service/trading_strategy/golden_x/notification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

// adaptiveWindowMax is the maximum number of historical RSI values we keep for
// percentile computation. Matches the original ~200 from the design spec.
const adaptiveWindowMax = 200

// adaptiveWindowMin is the lower bound; shares with fewer closed-week RSI
// values than this are skipped (consistent with C1's insufficient-history rule).
const adaptiveWindowMin = 100

// candleLookbackWeeks is how many weekly candles we request per share per tick.
// Covers EMA200 warmup (Growth-only) and adaptive-tier RSI history (both
// instances) in one RPC, staying under the Tinkoff weekly cap (300 per call).
const candleLookbackWeeks = 260

// ErrAdaptiveInsufficientHistory is returned when a share has fewer than
// adaptiveWindowMin closed weekly RSI values available.
var ErrAdaptiveInsufficientHistory = errors.New("adaptive tiers: insufficient RSI history")

func (s *service) Trade(ctx context.Context, in dto.Trade) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("panic in golden_x.Trade: %v", r))
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	loc, _ := time.LoadLocation("Europe/Moscow")
	dateNow := time.Now().In(loc)
	info := domain.NewInfo()
	RSIInfo := domain.NewInfo()
	trends := make(map[string]dto.TrendStatus)
	thresholds := make(map[string]dto.Thresholds)

	for _, share := range in.ShareList.All() {
		candles, candleErr := s.fetchWeeklyCandles(ctx, share.ID, in.Interval, dateNow)
		if candleErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("get candles for %s: %w", share.Name, candleErr).Error())
			continue
		}
		closed := closedWeeklyCandles(candles, dateNow, loc)

		lastRSI, sharedThresholds, adaptiveErr := adaptiveRSIForShare(closed, share.RSILength)
		if errors.Is(adaptiveErr, ErrAdaptiveInsufficientHistory) {
			logger.InfoContext(ctx, "adaptive tiers: insufficient history", "share", share.Name)
			continue
		}
		if adaptiveErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("adaptive tiers for %s: %w", share.Name, adaptiveErr).Error())
			continue
		}

		if in.UseTrendFilter {
			status, trendErr := trendStatusFromCandles(candles, trendEMAPeriod, dateNow, loc)
			if errors.Is(trendErr, ErrInsufficientHistory) {
				logger.InfoContext(ctx, "trend filter: insufficient history", "share", share.Name)
				continue
			}
			if trendErr != nil {
				logger.ErrorContext(ctx, fmt.Errorf("trend filter for %s: %w", share.Name, trendErr).Error())
				continue
			}
			trends[share.ID] = status
		}

		thresholds[share.ID] = sharedThresholds
		RSIInfo.WriteToMap(
			share.ID,
			domain.Item{
				InstrumentName: share.Name,
				RSILength:      share.RSILength,
				RSIValue:       lastRSI,
			})

		tier := tierFromAdaptive(lastRSI, sharedThresholds.P5, sharedThresholds.P15)
		if !s.state.ShouldAlert(share.ID, tier) {
			continue
		}

		info.WriteToMap(
			share.ID,
			domain.Item{
				InstrumentName: share.Name,
				RSIValue:       lastRSI,
			})
	}

	if len(info.Items()) > 0 {
		if sendErr := s.tgClient.SendMessage(notif.Trade(info, in.Kind, trends, thresholds)); sendErr != nil {
			logger.ErrorContext(ctx, "message is not sent", sendErr)
			return sendErr
		}
	}

	if len(RSIInfo.Items()) > 0 {
		if sendErr := s.tgClient.SendMessage(notif.RSIList(RSIInfo, in.Kind, thresholds)); sendErr != nil {
			logger.ErrorContext(ctx, "message is not sent", sendErr)
			return sendErr
		}
	}

	return nil
}

// fetchWeeklyCandles pulls up to ~candleLookbackWeeks weekly candles for the
// share — enough for both EMA200 (Growth-only) and adaptive RSI percentiles
// (both instances) in a single RPC.
func (s *service) fetchWeeklyCandles(ctx context.Context, shareID string, interval enum.Interval, dateNow time.Time) ([]*model.CandleItemTechAnalyse, error) {
	limit := int32(candleLookbackWeeks + 20)
	return s.marketDataServiceGrpcClient.GetCandles(
		ctx,
		&shareID,
		interval.ToNumberInvestApi(),
		utils.TimeStampPbGenerator(dateNow, -int64(candleLookbackWeeks), interval),
		timestamppb.New(dateNow),
		&limit,
		true,
	)
}

// adaptiveRSIForShare computes the share's last-closed-week RSI and adaptive
// p5/p15 thresholds over up to adaptiveWindowMax historical RSI values.
// Returns ErrAdaptiveInsufficientHistory if fewer than adaptiveWindowMin RSI
// values are available.
func adaptiveRSIForShare(closedCandles []*model.CandleItemTechAnalyse, rsiPeriod int) (float64, dto.Thresholds, error) {
	closes := make([]float64, len(closedCandles))
	for i, c := range closedCandles {
		closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
	}
	full := computeRSISeries(closes, rsiPeriod)
	// Trim warmup zeros — RSI is defined from index rsiPeriod onward.
	if len(full) <= rsiPeriod {
		return 0, dto.Thresholds{}, ErrAdaptiveInsufficientHistory
	}
	rsi := full[rsiPeriod:]
	if len(rsi) < adaptiveWindowMin {
		return 0, dto.Thresholds{}, ErrAdaptiveInsufficientHistory
	}
	// Keep only the most recent adaptiveWindowMax values for percentile stability.
	if len(rsi) > adaptiveWindowMax {
		rsi = rsi[len(rsi)-adaptiveWindowMax:]
	}
	lastRSI := rsi[len(rsi)-1]
	return lastRSI, adaptiveThresholds(rsi), nil
}
```

- [ ] **Step 7: Build the package and run all golden_x tests**

Run: `go build ./...`
Expected: no output, exit code 0.

Run: `go test ./internal/service/trading_strategy/golden_x/...`
Expected: all PASS.

- [ ] **Step 8: Run the full repo test pass**

Run: `go test ./... 2>&1 | grep -E "^(ok|FAIL|---)"`
Expected:
- `pkg/client/telegram` continues to fail with the **pre-existing** `non-constant format string` build error (out of scope).
- All other packages either `ok` or `[no test files]`.

- [ ] **Step 9: Commit Tasks 5 + 6 together**

```bash
git add -A
git status
git commit -m "$(cat <<'EOF'
feat(golden_x): adaptive p5/p15 RSI tiers replace static 31/35/40 thresholds

Per-share percentile tiers (🟡 RSI < p15, 🟢 RSI < p5) computed on up to 200
closed weekly RSI values per Trade() tick. Shares with fewer than 100 RSI
samples are skipped. Applied to both Gold and Growth instances.

Switches per-share RPC pattern to a single GetCandles call that serves both
the EMA200 trend filter (Growth) and the new adaptive tiers (both flows).
Drops the rsi.Instrument dependency from golden_x — RSI is computed locally
from the same candle slice. Notifications include the share's own p5/p15.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: End-to-end verification

This is a manual verification step. It cannot be automated — it needs a live Tinkoff Invest API connection.

- [ ] **Step 1: Run the dev binary**

In a terminal with `env/local.env` configured (token, chat IDs):

```bash
go run ./cmd/main
```

Expected: app boots, runs the Growth instance once (`Scheduler: "* * * * *"`).

- [ ] **Step 2: Inspect the Telegram message from the Growth instance**

Expected message shape:

```
🥈

Акции находящиеся в локальных минимумах:

• Акция: Yandex 🟢 ✅
  RSI Value:22  (p5=24.0, p15=31.0)

• Акция: Polyus 🟡 🚫
  RSI Value:28  (p5=24.0, p15=31.0)
```

Confirm:
- Tier emoji (`🟢` / `🟡`) is present.
- Trend mark (`✅` / `🚫`) is present (because UseTrendFilter is `true` for Growth).
- `(p5=…, p15=…)` annotation is on the RSI Value line.
- A share known to lack history (e.g. Yandex if its local re-listing is < 100 weeks old) does **not** appear, and the dev log contains `adaptive tiers: insufficient history`.

- [ ] **Step 3: Inspect the Gold instance (run prod-style or temporarily add a Gold goroutine to runDev)**

Expected message shape:

```
🥇

Акции находящиеся в локальных минимумах:

• Акция: Лукойл 🟡
  RSI Value:30  (p5=22.0, p15=33.0)
```

Confirm:
- Tier emoji is present.
- Trend mark is **absent** (UseTrendFilter is `false` for Gold).
- Threshold suffix is present.

- [ ] **Step 4: Verify dedup on consecutive runs**

Re-run within the same week without restarting (e.g. by re-triggering the scheduler if running prod, or running dev twice). Confirm: no new message arrives if no share crossed a tier boundary.

- [ ] **Step 5: Mark the stage complete**

Push the branch or merge per `superpowers:finishing-a-development-branch`.

---

## Self-Review

(Done by the planner — left here so the executor can sanity-check.)

- **Spec coverage:** All sections of the design spec are covered. Two-level tier → Task 3 + Task 6. Window 200/min 100 → Task 6 (`adaptiveWindowMax`, `adaptiveWindowMin`). R-7 percentile → Task 2. Both Gold and Growth → Task 6 (no `UseAdaptiveTiers` flag added). One RPC per share → Task 6 (`fetchWeeklyCandles`). Notifications with thresholds → Task 6 step 1 + 2. Domain.Item unchanged → confirmed (no edit in Task 6).
- **Placeholders:** none.
- **Type consistency:** `dto.Thresholds{P5, P15}` used identically in Tasks 1, 3, 6 and tests. `alertTier` values `{tierNone, tierYellow, tierGreen}` consistent across Tasks 3, 5, 6.
- **Risks for executor:** Task 5 leaves the package non-building until Task 6 ships — explicitly flagged. Commits are tied (Tasks 5 + 6 share a single commit).
