# Golden X Stage C4 — Volume Confirmation Badge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Each task is self-contained and runs in its own context window. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a 🔊 volume-confirmation badge to Golden X buy alerts when the last closed weekly candle's volume is strictly greater than `1.5 × SMA20` of the preceding 20 closed weeks. Both Gold and Growth instances. Buy side only. Dedup, sell side, and `RSIList` unchanged.

**Architecture:** One new pure helper in a brand-new `pkg/indicators` package (first reusable TA function extracted out of `golden_x`). Volume data is already available via `model.CandleItemTechAnalyse.Volume` on the same candle slice `trade.go` already fetches — no extra RPC. Same parallel-map orchestration pattern used by `trends`, `thresholds`, `sellThresholds`, `divergences`: add `volumesConfirmed map[string]bool` keyed by `share.ID`, pass through to `notif.Trade(...)`.

**Tech Stack:** Go 1.25, stdlib only. No new dependencies.

**Reference spec:** `docs/superpowers/specs/2026-05-18-golden-x-stage-c4-volume-confirmation-design.md`.

**Working directory:** `/home/oleg/GolandProjects/tinvest`. Branch: `feature/golden-x-stage-a` (continues from C3).

---

## File Structure

**New files (2):**

- `pkg/indicators/volume.go` — single exported function `VolumeConfirmed(volumes []int64, lookback int, multiplier float64) bool`. Pure, no side effects. Lives in a new package because the user has explicitly asked for the helper to be reusable across strategies; future migrations of `Percentile`, `WilderRSI`, etc. may follow this path but are out of scope.
- `pkg/indicators/volume_test.go` — table-driven unit tests for `VolumeConfirmed`. Covers the confirmed/not-confirmed/boundary/insufficient-history paths plus a multiplier-sensitivity sanity case.

**Modified files (3):**

- `internal/service/trading_strategy/golden_x/trade.go` — add two constants (`volumeSMALookback`, `volumeMultiplier`); add `volumesConfirmed := make(map[string]bool)` to the per-tick state; inside the existing `if buyTier != tierNone { ... }` block, extract volumes from `closed` and call `indicators.VolumeConfirmed`; pass `volumesConfirmed` as the 8th argument to `notif.Trade(...)`. Adds one new import: `"tinvest/pkg/indicators"`.
- `internal/service/trading_strategy/golden_x/notification/notifications.go` — `Trade(...)` signature grows by one trailing parameter `volumesConfirmed map[string]bool`; new helper `volumeBadge(bool) string` (sibling to `divergenceBadge`); buy-loop concatenates `volumeBadge(volumesConfirmed[id])` after the divergence concat.
- `internal/service/trading_strategy/golden_x/notification/notifications_test.go` — update the 8 existing `Trade(...)` call sites to add a trailing `nil` for `volumesConfirmed`; add 3 new render tests (badge-after-divergence, badge-absent-when-unconfirmed, badge-without-divergence).

**Untouched (explicitly verified in the spec):** `internal/service/trading_strategy/golden_x/dto/`, `dedup.go`, `divergence.go`, `notification/rsi_by_shares.go`, `internal/service_provider/`, `internal/domain/info.go`. No new flag in `dto.Trade` — volume confirmation runs for both Gold and Growth.

---

## Task 1: `VolumeConfirmed` pure helper (TDD)

**Files:**
- Create: `pkg/indicators/volume.go`
- Create: `pkg/indicators/volume_test.go`

The `pkg/indicators/` directory does not exist yet — `pkg/indicators/volume.go` creates it. No `doc.go` is required (single-file package, single function).

- [ ] **Step 1: Write the failing tests**

Create `pkg/indicators/volume_test.go`:

```go
package indicators

import "testing"

func TestVolumeConfirmed(t *testing.T) {
	tests := []struct {
		name       string
		volumes    []int64
		lookback   int
		multiplier float64
		want       bool
	}{
		{
			name:       "confirmed: last is 2x SMA",
			volumes:    append(repeatInt64(10, 20), 20),
			lookback:   20,
			multiplier: 1.5,
			want:       true,
		},
		{
			name:       "not confirmed: last equals SMA",
			volumes:    repeatInt64(10, 21),
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "not confirmed: last below 1.5x SMA",
			volumes:    append(repeatInt64(10, 20), 14),
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "boundary: last equals 1.5x SMA (strict >)",
			volumes:    append(repeatInt64(10, 20), 15),
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "insufficient history",
			volumes:    repeatInt64(100, 5),
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "empty input",
			volumes:    nil,
			lookback:   20,
			multiplier: 1.5,
			want:       false,
		},
		{
			name:       "different multiplier 2.0x boundary",
			volumes:    append(repeatInt64(10, 20), 20),
			lookback:   20,
			multiplier: 2.0,
			want:       false,
		},
		{
			name:       "different multiplier 2.0x pass",
			volumes:    append(repeatInt64(10, 20), 21),
			lookback:   20,
			multiplier: 2.0,
			want:       true,
		},
		{
			name:       "lookback zero is silent false",
			volumes:    repeatInt64(10, 5),
			lookback:   0,
			multiplier: 1.5,
			want:       false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := VolumeConfirmed(tc.volumes, tc.lookback, tc.multiplier)
			if got != tc.want {
				t.Fatalf("VolumeConfirmed = %v, want %v", got, tc.want)
			}
		})
	}
}

// repeatInt64 returns a slice of n copies of v.
func repeatInt64(v int64, n int) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = v
	}
	return out
}
```

- [ ] **Step 2: Run tests to verify they fail with "undefined"**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./pkg/indicators/... -run TestVolumeConfirmed -v 2>&1 | head -40`

Expected: build error / FAIL — `undefined: VolumeConfirmed`. (The package `indicators` doesn't yet have any implementation file.)

- [ ] **Step 3: Write the minimal implementation**

Create `pkg/indicators/volume.go`:

```go
package indicators

// VolumeConfirmed reports whether the last value of volumes is strictly greater
// than multiplier × SMA of the preceding lookback values. Returns false when
// lookback <= 0 or len(volumes) < lookback+1 — the insufficient-history rule
// is silent (no error).
//
// Used by Golden X Stage C4 as a buy-side annotation: a true result says the
// last bar's volume is meaningfully above its recent baseline by the given
// multiplier ratio.
func VolumeConfirmed(volumes []int64, lookback int, multiplier float64) bool {
	if lookback <= 0 || len(volumes) < lookback+1 {
		return false
	}
	last := volumes[len(volumes)-1]
	window := volumes[len(volumes)-1-lookback : len(volumes)-1]
	var sum int64
	for _, v := range window {
		sum += v
	}
	sma := float64(sum) / float64(lookback)
	return float64(last) > multiplier*sma
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./pkg/indicators/... -run TestVolumeConfirmed -v`

Expected: PASS — all 9 subtests green.

- [ ] **Step 5: Build the whole repo (sanity check)**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./...`

Expected: no output, exit code 0. (No call sites yet — `pkg/indicators` is isolated.)

- [ ] **Step 6: Commit**

```bash
cd /home/oleg/GolandProjects/tinvest && git add pkg/indicators && git commit -m "$(cat <<'EOF'
feat(indicators): add VolumeConfirmed helper in new pkg/indicators package

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Wire volume badge into `Trade()` and render in notifications

This task changes `notif.Trade(...)` signature, which means **both** `notifications.go` and the call site in `trade.go` must move together — the package will not build between Step 3 and Step 5. Commit only at the end (Step 9), as a single logical unit (mirrors the pattern from Stage C2 Tasks 5+6 and Stage C3 Task 2).

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/notification/notifications.go`
- Modify: `internal/service/trading_strategy/golden_x/notification/notifications_test.go`
- Modify: `internal/service/trading_strategy/golden_x/trade.go`

- [ ] **Step 1: Update existing notification test calls to new arity**

Edit `internal/service/trading_strategy/golden_x/notification/notifications_test.go`. Add a trailing `nil` argument to each of the 8 existing `Trade(...)` calls. Each line below shows the BEFORE → AFTER for that one call:

Line 28 (in `TestTrade_AdaptiveTiersAndThresholdSuffix`):

```go
// BEFORE:
got := Trade(info, nil, dto.StrategyKindGrowth, trends, thresholds, nil, nil)
// AFTER:
got := Trade(info, nil, dto.StrategyKindGrowth, trends, thresholds, nil, nil, nil)
```

Line 48 (in `TestTrade_NoThresholdsRendersNoEmojiOrSuffix`):

```go
// BEFORE:
got := Trade(info, nil, dto.StrategyKindDividend, nil, nil, nil, nil)
// AFTER:
got := Trade(info, nil, dto.StrategyKindDividend, nil, nil, nil, nil, nil)
```

Line 74 (in `TestTrade_RendersSellSectionGold`):

```go
// BEFORE:
got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, nil, sellThresholds, nil)
// AFTER:
got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, nil, sellThresholds, nil, nil)
```

Line 105 (in `TestTrade_RendersSellSectionGrowth`):

```go
// BEFORE:
got := Trade(buyInfo, sellInfo, dto.StrategyKindGrowth, nil, nil, sellThresholds, nil)
// AFTER:
got := Trade(buyInfo, sellInfo, dto.StrategyKindGrowth, nil, nil, sellThresholds, nil, nil)
```

Line 128 (in `TestTrade_BuyAndSellTogether`):

```go
// BEFORE:
got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, thresholds, sellThresholds, nil)
// AFTER:
got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, thresholds, sellThresholds, nil, nil)
```

Line 171 (in `TestTrade_RendersBullishDivergenceBadge`):

```go
// BEFORE:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences)
// AFTER:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, nil)
```

Line 186 (in `TestTrade_NoBadgeWhenNotDivergent`):

```go
// BEFORE:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil)
// AFTER:
got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, nil)
```

Line 205 (in `TestTrade_BadgeAfterTrendMark`):

```go
// BEFORE:
got := Trade(buyInfo, nil, dto.StrategyKindGrowth, trends, thresholds, nil, divergences)
// AFTER:
got := Trade(buyInfo, nil, dto.StrategyKindGrowth, trends, thresholds, nil, divergences, nil)
```

- [ ] **Step 2: Append three new render tests**

Append to `internal/service/trading_strategy/golden_x/notification/notifications_test.go`:

```go
func TestTrade_VolumeBadgeAppendedAfterDivergence(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-both", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-both": {P5: 24, P15: 31},
	}
	divergences := map[string]bool{"share-both": true}
	volumes := map[string]bool{"share-both": true}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, volumes)

	// Expect order: name, tier emoji, divergence badge, volume badge.
	if !strings.Contains(got, "Lukoil 🟢 📈 🔊") {
		t.Errorf("expected 'Lukoil 🟢 📈 🔊' (ordered), got:\n%s", got)
	}
}

func TestTrade_VolumeBadgeAbsentWhenNotConfirmed(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-no-vol", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-no-vol": {P5: 24, P15: 31},
	}
	divergences := map[string]bool{"share-no-vol": true}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, nil)

	if strings.Contains(got, "🔊") {
		t.Errorf("expected no volume badge, got:\n%s", got)
	}
	if !strings.Contains(got, "Lukoil 🟢 📈") {
		t.Errorf("expected 'Lukoil 🟢 📈' (divergence kept), got:\n%s", got)
	}
}

func TestTrade_VolumeBadgeWithoutDivergence(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-vol-only", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-vol-only": {P5: 24, P15: 31},
	}
	volumes := map[string]bool{"share-vol-only": true}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, volumes)

	if !strings.Contains(got, "Lukoil 🟢 🔊") {
		t.Errorf("expected 'Lukoil 🟢 🔊' (volume only, no divergence), got:\n%s", got)
	}
	if strings.Contains(got, "📈") {
		t.Errorf("expected no divergence badge, got:\n%s", got)
	}
}
```

- [ ] **Step 3: Update `Trade(...)` signature and add `volumeBadge`**

Edit `internal/service/trading_strategy/golden_x/notification/notifications.go`. Replace the entire `Trade` function and append the new helper.

Replace the existing `Trade(...)` declaration (lines 12-51) with:

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

(The only differences from the existing function: one new last parameter `volumesConfirmed`, and the buy-row builder appends `+ volumeBadge(volumesConfirmed[id])` after `divergenceBadge(...)`. Sell loop and footer untouched.)

Then, immediately after the existing `divergenceBadge` definition (before `tierEmoji`), append:

```go
// volumeBadge returns " 🔊" when the share's row should display the volume
// confirmation annotation, "" otherwise. The badge is purely additive — it
// never participates in dedup and never replaces an existing emoji.
func volumeBadge(confirmed bool) string {
	if confirmed {
		return " 🔊"
	}
	return ""
}
```

- [ ] **Step 4: Run notification tests — should pass standalone**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/notification/... -v`

Expected: PASS for all tests in the notification package (including the 3 new ones). The wider repo will not yet build — that is fixed in Step 5.

- [ ] **Step 5: Update `trade.go` — constants, map, helper call, notification arity**

Edit `internal/service/trading_strategy/golden_x/trade.go`. Three localized changes:

**(5a) Add the new import.** Replace the existing import block (lines 3-18) with:

```go
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
	"tinvest/pkg/indicators"
	"tinvest/pkg/logger"
)
```

(The only change: one new line `"tinvest/pkg/indicators"`, sorted alphabetically inside the third group.)

**(5b) Add the two new constants.** Immediately after the existing `divergenceLookbackWeeks` declaration block (it ends at line 42), append:

```go
// volumeSMALookback is the number of closed weekly candles preceding the
// last closed week, used as the SMA baseline for volume confirmation.
const volumeSMALookback = 20

// volumeMultiplier is the strictness factor: the last closed week's volume
// must be > volumeMultiplier × SMA of the previous volumeSMALookback weeks
// for the 🔊 badge to fire. 1.5× is the balance between "barely above
// average" (which would emit the badge for most shares and dilute meaning)
// and a 2× "rare spike" (which would almost never fire).
const volumeMultiplier = 1.5
```

**(5c) Add the parallel map declaration.** Find the existing block (around line 63):

```go
divergences := make(map[string]bool)
```

And replace it with two lines:

```go
divergences := make(map[string]bool)
volumesConfirmed := make(map[string]bool)
```

**(5d) Extend the `if buyTier != tierNone` block.** Find the existing block (around lines 102-114):

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
}
```

Replace with:

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

**(5e) Update the `notif.Trade(...)` call.** Find the existing line (around line 150):

```go
msg := notif.Trade(buyInfo, sellInfo, in.Kind, trends, thresholds, sellThresholds, divergences)
```

Replace with:

```go
msg := notif.Trade(buyInfo, sellInfo, in.Kind, trends, thresholds, sellThresholds, divergences, volumesConfirmed)
```

- [ ] **Step 6: Build the package and run all golden_x tests**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./...`

Expected: no output, exit code 0.

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/... ./pkg/indicators/... -v 2>&1 | tail -60`

Expected: PASS for all tests (existing + 3 new notification tests + 9 `VolumeConfirmed` subtests).

- [ ] **Step 7: Run the full repo test pass (sanity)**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./... 2>&1 | grep -E "^(ok|FAIL|---)" | sort -u`

Expected:
- `pkg/client/telegram` continues to fail with the **pre-existing** `non-constant format string` build error (out of scope, same caveat documented in C2, B, and C3 plans).
- All other packages either `ok` or `[no test files]`. No new failures.

- [ ] **Step 8: Vet check**

Run: `cd /home/oleg/GolandProjects/tinvest && go vet ./internal/service/trading_strategy/golden_x/... ./pkg/indicators/...`

Expected: empty output.

- [ ] **Step 9: Commit**

```bash
cd /home/oleg/GolandProjects/tinvest && git add -A && git status && git commit -m "$(cat <<'EOF'
feat(golden_x): volume confirmation badge in buy alerts

Per-share volume check: when the last closed weekly candle's volume is
strictly greater than 1.5× SMA of the preceding 20 closed weeks, the buy
row gets a 🔊 badge after the divergence badge (e.g. "Lukoil 🟢 ✅ 📈 🔊").
Both Gold and Growth instances. Buy side only. Dedup is unchanged — the
badge is a pure annotation and never participates in alert state.

The volume helper lives in a new pkg/indicators package — the first step
toward extracting reusable TA primitives out of golden_x.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected output of `git status` after commit: clean tree.

---

## Task 3: End-to-end verification (manual)

This is a manual verification step. It cannot be automated — it requires a live Tinkoff Invest API connection.

- [ ] **Step 1: Run the dev binary**

In a terminal with `env/local.env` configured (Tinkoff token, Telegram chat IDs):

```bash
cd /home/oleg/GolandProjects/tinvest && go run ./cmd/main
```

Expected: app boots, runs the Growth instance once on the `* * * * *` schedule. Then run a Gold tick by triggering it (or temporarily wire a one-off goroutine in `runDev` for testing).

- [ ] **Step 2: Inspect a Telegram buy-section message**

If at least one share in the buy zone has elevated weekly volume on the last closed week, expect a row like:

```
🥈

Акции находящиеся в локальных минимумах:

• Акция: Yandex 🟢 ✅ 📈 🔊
  RSI Value:22  (p5=24.0, p15=31.0)

• Акция: Polyus 🟡 🚫
  RSI Value:28  (p5=24.0, p15=31.0)
```

Confirm:
- 🔊 appears **after** 📈 (when both present) and after the trend mark (✅/🚫) when no divergence.
- A buy alert without volume confirmation has no 🔊.
- The sell section never has 🔊 (buy-side only).
- `RSIList` (informational) has no 🔊 in any row.

- [ ] **Step 3: Sanity-check log volume**

Search the dev log for any volume-related panics or errors. There should be none — `VolumeConfirmed` is a pure function returning `false` on bad input, and the surrounding code does not log anything when the badge is absent.

- [ ] **Step 4: Finish the branch**

The branch `feature/golden-x-stage-a` now carries Stages A + C1 + C2 + B + C3 + C4 — six logical stages, ready for review/merge as one batch. Manual E2E for B, C3, and now C4 can be co-verified in the same dev run.

After E2E passes, invoke `superpowers:finishing-a-development-branch` to decide between local merge and PR.

---

## Self-Review

(Done by the planner — left here so the executor can sanity-check.)

**1. Spec coverage:**

- Badge model (no filter, dedup unchanged) → Task 2 Step 3 (helper rendered only as concat, no `ShouldAlert` change) and Task 2 Step 5d (volume map populated independently of dedup).
- Rule `last > 1.5 × SMA20`, strict `>` → Task 1 Step 3 (`float64(last) > multiplier*sma`).
- Lookback 20 closed weeks BEFORE the last → Task 1 Step 3 (window `volumes[len-1-lookback : len-1]` excludes the last index).
- Minimum 21 closed weeks; silent skip below → Task 1 Step 3 (`len(volumes) < lookback+1 → return false`, no log).
- Both Gold and Growth, no flag → Task 2 Step 5 leaves `dto.Trade` untouched; volume check runs unconditionally inside the buy-tier branch.
- Buy side only → Task 2 Step 5d places the check inside `if buyTier != tierNone { ... }`; sell loop in Step 3 is untouched.
- Helper in `pkg/indicators` → Task 1 creates `pkg/indicators/volume.go` from scratch.
- 🔊 after 📈 → Task 2 Step 3 (`divergenceBadge(...) + volumeBadge(...)` order in the buy-row concat) plus regression `TestTrade_VolumeBadgeAppendedAfterDivergence` checking substring `"Lukoil 🟢 📈 🔊"`.
- `RSIList` unchanged → Task 2 does not touch `rsi_by_shares.go`; `TestRSIList_RendersAdaptiveTier` continues to use the original signature.

**2. Placeholder scan:** every step contains either complete code blocks or exact shell commands. No TBD / TODO / "similar to …" placeholders.

**3. Type consistency:**

- `VolumeConfirmed(volumes []int64, lookback int, multiplier float64) bool` — declared in Task 1 Step 3, invoked identically in Task 2 Step 5d.
- `volumesConfirmed map[string]bool` — declared in Task 2 Step 5c, read in Task 2 Step 3 (`volumesConfirmed[id]` inside `Trade(...)`).
- `volumeBadge(bool) string` — declared in Task 2 Step 3, called in the same step from the buy-row concat.
- `volumeSMALookback = 20`, `volumeMultiplier = 1.5` — declared in Task 2 Step 5b, used in Task 2 Step 5d.
- Notification call-site arity: 8 arguments — matches the 8-parameter signature in Task 2 Step 3.

**4. Risks for the executor:**

- The package is not buildable between Task 2 Step 3 and Task 2 Step 5. The plan explicitly groups Steps 1-5 into a single commit at Step 9. Do not commit partial state in between.
- `pkg/client/telegram` build failure is pre-existing and out of scope.
- If `go test ./...` shows any **new** failure beyond `pkg/client/telegram`, stop and inspect — that's a regression.
