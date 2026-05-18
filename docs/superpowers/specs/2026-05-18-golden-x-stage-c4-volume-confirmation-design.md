# Golden X Stage C4 — Volume Confirmation Badge

## Context

Branch `feature/golden-x-stage-a` already carries Stage A (basic fixes), Stage C1 (EMA200 W trend filter for Growth), Stage C2 (adaptive p5/p15 buy tiers), Stage B (adaptive sell tiers — p80/p90/p95 Gold + p90 Growth), and Stage C3 (bullish RSI divergence badge). The master plan (`~/.claude/plans/internal-service-trading-strategy-golde-woolly-quill.md`) lists **C4 — volume confirmation** next: «средний объём последней недели выше N-недельной средней».

**Why now.** Buy tiers already fire on share-specific oversold conditions and divergence already flags reversal candidates. Volume is the classical third confirmation: heavy volume on the oversold week says "real buyers stepped in", light volume says "the move was a vacuum". Adding it as a badge (not a filter) gives the user one more piece of evidence without changing alert volume or dedup semantics.

**Outcome.** When a buy-tier share's last closed weekly candle has a volume strictly greater than `1.5 × SMA20` of the preceding 20 closed weeks, the buy row gets a 🔊 badge after the divergence mark. Everything else (tier, trend, thresholds, dedup, sell side, RSIList) is unchanged.

## Locked decisions

| Question | Decision |
|---|---|
| Integration model | **Badge** in the Telegram message, no filtering. Buy alerts fire as today; volume only annotates. Dedup is untouched. |
| Confirmation rule | `last_volume > 1.5 × SMA20(previous_20_volumes)`. Strict `>`; equality at the boundary is **not** a confirmation. |
| Lookback window | **20 closed weekly candles before the last closed week.** SMA does NOT include the candle being evaluated. |
| Multiplier | **1.5×** (named constant `volumeMultiplier`). |
| Minimum history | **21 closed weeks** (20 for SMA + 1 current). Fewer → silent `false`, no error log. |
| Scope across instances | **Both Gold and Growth.** Universal TA tool, no `dto.Trade` flag added. |
| Scope across sides | **Buy side only.** Sell rows render no volume badge. Detection is skipped for shares with `buyTier == tierNone`. |
| Badge symbol | **🔊** appended after the divergence mark in the buy-section row (`• Акция: Lukoil 🟢 ✅ 📈 🔊`). Absent when no volume confirmation. |
| Dedup | **Unchanged.** Tier-only dedup; the badge is an annotation. A share toggling badge on/off without changing tier sends no new alert. |
| RSIList (informational dump) | **Unchanged.** No badge there. |
| Helper location | New package `pkg/indicators` — first reusable indicator extracted out of `golden_x`. Future `Percentile`, `WilderRSI`, etc. may follow, but those migrations are out of scope. |

## Architecture overview

The volume check is a single pure function in a new package, sequenced inside the existing share loop in `trade.go`:

1. After `adaptiveRSIForShare` returns `(lastRSI, rsiSeries, thresholds, nil)` and `buyTier` is computed, the existing `if buyTier != tierNone { … }` block already runs divergence detection. Volume confirmation runs in the **same block**, immediately after divergence.
2. Extract `volumes []int64` from `closed []*model.CandleItemTechAnalyse` (the same slice already used for RSI closes and divergence lows).
3. Call `indicators.VolumeConfirmed(volumes, volumeSMALookback, volumeMultiplier)`. The helper:
   - Returns `false` if `len(volumes) < lookback + 1` (silent insufficient history).
   - Otherwise compares `volumes[last] > multiplier * mean(volumes[last-lookback : last])`. Strict `>`.
4. Store the resulting `bool` in `volumesConfirmed[share.ID]`. Pass the map through to `notif.Trade(...)`, which renders the 🔊 next to the share name in the buy-section row when set.

Dedup state is **not** consulted for volume — the badge can appear/disappear without ever generating a new Telegram message on its own. Only a tier change triggers a send (current C1+C2+B behavior).

## Files

**New:**

- `pkg/indicators/volume.go` — `VolumeConfirmed(volumes []int64, lookback int, multiplier float64) bool`.
- `pkg/indicators/volume_test.go` — table-driven unit tests for the helper.

**Modified:**

- `internal/service/trading_strategy/golden_x/trade.go` — two new constants (`volumeSMALookback = 20`, `volumeMultiplier = 1.5`); inside the `if buyTier != tierNone` block, extract `volumes []int64` and call `indicators.VolumeConfirmed`; populate `volumesConfirmed map[string]bool`; pass to `notif.Trade(...)`.
- `internal/service/trading_strategy/golden_x/notification/notifications.go` — `Trade(...)` signature gains `volumesConfirmed map[string]bool`; buy row appends `" 🔊"` when `volumesConfirmed[id]` is true (after the divergence mark, before the line break).
- `internal/service/trading_strategy/golden_x/notification/notifications_test.go` — add render tests for badge presence/absence; update existing `Trade(...)` calls to the new arity.

**Untouched (and explicitly so):**

- `internal/service/trading_strategy/golden_x/dto/trade.go` — no flag added; both instances run volume confirmation.
- `internal/service/trading_strategy/golden_x/dedup.go` — `alertTier` and `ShouldAlert` unchanged.
- `internal/service/trading_strategy/golden_x/divergence.go` — unchanged.
- `internal/service/trading_strategy/golden_x/notification/rsi_by_shares.go` — `RSIList` intentionally unchanged.
- `internal/service_provider/service.go` — constructor signature unchanged (no new deps).
- `internal/domain/info.go` — `Item` schema unchanged; volume state lives in a parallel map by shareID, same pattern as `trends`, `thresholds`, `sellThresholds`, `divergences`.

## Helper signature

```go
// pkg/indicators/volume.go
package indicators

// VolumeConfirmed reports whether the last value of volumes is strictly greater
// than multiplier × SMA of the preceding lookback values. Returns false when
// len(volumes) < lookback+1 (silent insufficient-history rule).
//
// This is the classical TA "volume spike" check used to confirm price moves:
// a true result says the last bar's volume is meaningfully above its recent
// baseline by multiplier ratio.
func VolumeConfirmed(volumes []int64, lookback int, multiplier float64) bool {
    if len(volumes) < lookback+1 {
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

## Constants (in `trade.go`)

Added after `divergenceLookbackWeeks`:

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

## Reused utilities

- `closedWeeklyCandles(candles, dateNow, loc)` — already used by `adaptiveRSIForShare` and `bullishDivergence`; the same `[]*model.CandleItemTechAnalyse` slice is reused for volume extraction. No extra RPC.
- `model.CandleItemTechAnalyse.Volume` (`int64`) — already populated by the Tinkoff candle path.
- The parallel-map pattern (`trends`, `thresholds`, `sellThresholds`, `divergences`) — `volumesConfirmed` slots in by the same shape.

## Notification rendering

`Trade(...)` signature evolves to:

```go
func Trade(
    buyInfo, sellInfo *domain.Info,
    kind dto.StrategyKind,
    trends map[string]dto.TrendStatus,
    thresholds map[string]dto.Thresholds,
    sellThresholds map[string]dto.SellThresholds,
    divergences map[string]bool,
    volumesConfirmed map[string]bool, // <-- new, last position
) string
```

Buy-row construction order (left → right): instrument name, tier emoji, trend mark, divergence badge, volume badge. A tiny helper mirrors `divergenceBadge`:

```go
// volumeBadge returns " 🔊" when confirmed, otherwise "".
func volumeBadge(confirmed bool) string {
    if confirmed {
        return " 🔊"
    }
    return ""
}
```

Call site: appended after the existing divergence concat.

## Edge cases

- `len(volumes) < 21` → helper returns `false`, badge absent, alert still fires. No log.
- All volumes are zero (suspended trading) → SMA = 0; helper returns `float64(last) > 1.5 × 0` ⇔ `last > 0`. If `last == 0` too, returns `false`. Correct.
- `lookback == 0` (would degenerate into SMA over empty slice) → guarded by `len(volumes) < lookback+1` ⇒ `len < 1` ⇒ `false` only when `len == 0`. Caller passes 20, so this path is dead in practice; helper stays correct under it via the standard guard.
- Negative volume → cannot occur in Tinkoff candle data; no defensive check.

## Testing strategy

**`pkg/indicators/volume_test.go`** (table-driven, 5 cases minimum):

| Case | Inputs | Expected |
|---|---|---|
| confirmed (last = 2× SMA) | `[10]*20 ++ [20]`, lookback=20, mul=1.5 | `true` |
| not confirmed (last == SMA) | `[10]*21`, lookback=20, mul=1.5 | `false` |
| not confirmed (last < 1.5× SMA) | `[10]*20 ++ [14]`, lookback=20, mul=1.5 | `false` |
| boundary `last == 1.5× SMA` | `[10]*20 ++ [15]`, lookback=20, mul=1.5 | `false` (strict `>`) |
| insufficient history | `[100]*5`, lookback=20, mul=1.5 | `false` |
| empty input | `nil`, lookback=20, mul=1.5 | `false` |
| different multiplier (sanity) | `[10]*20 ++ [20]`, lookback=20, mul=2.0 | `false` (boundary) |

**`internal/service/trading_strategy/golden_x/notification/notifications_test.go`** (additions):

- `TestTrade_VolumeBadgeAppendedAfterDivergence`: buy row with both `divergences[id]=true` and `volumesConfirmed[id]=true` → row contains `📈 🔊` in that order.
- `TestTrade_VolumeBadgeAbsentWhenNotConfirmed`: buy row with `divergences[id]=true` and `volumesConfirmed=nil` → row contains `📈` and no 🔊.
- `TestTrade_VolumeBadgeWithoutDivergence`: buy row with `divergences=nil` and `volumesConfirmed[id]=true` → row contains 🔊 but no 📈.
- All 5 existing `Trade(...)` call sites in this file gain a trailing `nil` for `volumesConfirmed` (mirrors what dreamy-crafting-scott did for divergences).

**End-to-end (manual, deferred to executor):** run `go run ./cmd/main` with `env/local.env` and confirm a buy row of a heavy-volume oversold share shows 🔊, and a buy row of a quiet share does not.

## Out of scope

- Migrating other helpers (`percentile`, `WilderRSI`, `divergence`) into `pkg/indicators` — separate refactor.
- Volume on sell signals — bullish-only by current scope.
- Volume-based dedup (e.g. "Green + volume" as a distinct tier) — too much logic for C4.
- ATR-based stop/position sizing — Stage C5.
- Postgres signal log — Stage D1.
