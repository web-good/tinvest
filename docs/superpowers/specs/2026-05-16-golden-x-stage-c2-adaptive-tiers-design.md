/superpowers:using-superpowers используя практики чистой ахитектуры , патерны  фабрика пайплайн декоратор многопаточность и тд проревьюуй эту стратегию                                     
internal/service/trading_strategy/golden_x
# Golden X — Stage C2: Adaptive RSI Tiers

## Context

Stage A (basic fixes) and Stage C1 (EMA200 W trend filter for Growth) are merged on branch `feature/golden-x-stage-a`. The next priority from the long-term plan (`~/.claude/plans/internal-service-trading-strategy-golde-woolly-quill.md`, section "Идеи улучшений → C") is **C2: adaptive RSI thresholds**.

Goal: replace the hard-coded buy thresholds `31/35/40` (which treat Сургутнефтегаз-преф and Yandex as if their RSI behaved the same way) with **per-share percentiles** computed on the fly from the share's own recent RSI history.

Why now: with C1 the strategy already discriminates between trending and counter-trending entries; the next biggest source of false signals is the static thresholds that ignore each share's typical RSI amplitude. Whoever owns this stage should also note that the C2 work creates a foundation for C3/C4/C5 because the new code centralizes per-share candle fetching.

## Decisions (from brainstorming)

| Question | Decision |
|---|---|
| Tier mapping onto percentiles | **Two levels**: 🟡 Yellow if `RSI < p15`, 🟢 Green if `RSI < p5`, otherwise no alert. The current Brown tier (`35 ≤ RSI ≤ 40`) is dropped — under personal percentiles it becomes redundant noise. |
| Window for percentiles | **Up to 200** closed weekly RSI values, minimum 100. Less than 100 → skip the share. |
| Percentile method | **Linear interpolation (R-7)**, the default in `numpy.percentile` / `numpy.quantile`. Stable for our N range. |
| Scope (Gold vs Growth) | **Both instances.** Adaptive thresholds replace static ones everywhere; no `UseAdaptiveTiers` flag in `dto.Trade`. |
| Sell side (p95 / p85) | **Not in this stage.** Sell logic is Stage B. We compute and use only the lower-tail percentiles. |
| Caching of percentiles | **No cache.** One `GetCandles` per share per tick (we already do this for C1). |

## Approach

**One fetch per share, full local computation.**

Today `trade.go` makes two RPCs per share for Growth (RSI service + GetCandles for C1) and one for Gold. After C2 there is exactly **one** RPC per share — a single `GetCandles(week1, ~-260 weeks, now, limit ≈ 280)` call that returns enough closed weekly candles to support:

- The current RSI tier check (last closed-week RSI vs. p5/p15);
- The C1 EMA200 trend filter for Growth (re-uses the same candle slice);
- Percentile computation on the RSI series.

The existing `s.rsi.CalculateRSI` is dropped from `Trade()` because it only returns the last `period` RSI values, while C2 needs up to 200 historical values. The RSI formula is duplicated into a pure local helper (`computeRSISeries`) — same approach we took for EMA in C1.

### Data flow per share

```
GetCandles(weekly, -260w, now)
   │
   ▼
closedWeeklyCandles(...)            ← already exists in trend_filter.go
   │
   ├──► computeRSISeries(closes, share.RSILength)
   │       │
   │       ▼
   │   take last ≤200 RSI values
   │       │
   │       │  if < 100  → ErrInsufficientHistory → skip share
   │       │
   │       ▼
   │   percentile(sortedRSI, 5),  percentile(sortedRSI, 15)
   │       │
   │       ▼
   │   lastRSI = closedRSI[len-1]
   │       │
   │       ▼
   │   tierFromAdaptive(lastRSI, p5, p15)  → tierNone | tierYellow | tierGreen
   │
   └──► (Growth only) trendStatusFromCandles(...)  → ✅ / 🚫 / skip
```

### Tier semantics

```go
func tierFromAdaptive(rsi, p5, p15 float64) alertTier {
    switch {
    case rsi < p5:  return tierGreen
    case rsi < p15: return tierYellow
    default:        return tierNone
    }
}
```

The `tierBrown` constant and its branch in `tierFromRSI` (the old absolute mapping) are removed. The dedup state machine in `dedup.go` is untouched structurally — it still keys on `(shareID, tier)`. The transition table effectively collapses from {None, Brown, Yellow, Green} to {None, Yellow, Green}.

### Insufficient history behavior

If a share has fewer than 100 closed-week RSI values (e.g. an IPO with < ~2 years of weekly trading), `trade.go` logs `info`-level `"adaptive tiers: insufficient history" share=<name>` and skips the share — it appears neither in the alert message nor in the intermediate RSIList. This is consistent with C1 behavior.

Note: for Yandex (re-listed July 2024) the boundary is currently very close. The share may oscillate in and out of "enough history" over the next months. That's acceptable — no special handling.

## File Structure

**New files:**

- `internal/service/trading_strategy/golden_x/rsi.go`
  - `computeRSISeries(closes []float64, period int) []float64` — pure function, returns RSI value at each position from index `period` onward (positions before `period` are 0).
- `internal/service/trading_strategy/golden_x/percentile.go`
  - `percentile(sortedAsc []float64, p float64) float64` — linear-interpolation percentile (R-7).
  - `tierFromAdaptive(rsi, p5, p15 float64) alertTier`.
  - `adaptiveThresholds(rsiSeries []float64) dto.Thresholds` — orchestrator that sorts the slice, computes P5 and P15.
- `internal/service/trading_strategy/golden_x/rsi_test.go` — series correctness on a small fixture; equivalence with the existing `instrument/rsi` calculator on the same input (golden test).
- `internal/service/trading_strategy/golden_x/percentile_test.go` — known values (e.g. `percentile([1..100], 5) ≈ 5.95`), edge cases (single value, all equal), and `tierFromAdaptive` truth table.

**Modified files:**

- `internal/service/trading_strategy/golden_x/trade.go` — replace per-share RSI RPC with a single weekly-candles RPC; compute RSI series + percentiles locally; switch tier mapping to `tierFromAdaptive`.
- `internal/service/trading_strategy/golden_x/dedup.go` — remove `tierBrown` constant and its case from `tierFromRSI` (now dead code; `tierFromAdaptive` is the live mapping). Keep `alertState` / `ShouldAlert` unchanged.
- `internal/service/trading_strategy/golden_x/dedup_test.go` — drop the Brown-related rows; add a `tierFromAdaptive` table test in `percentile_test.go` (cleaner separation).
- `internal/service/trading_strategy/golden_x/types.go` — the `rsiInstrument` interface dependency becomes unused; **remove the field** and the constructor parameter to keep the seam tight.
- `internal/service_provider/service.go` — drop the `serviceProvider.RSI()` argument from `GetGoldenXTradingService()` constructor call. The `rsi.Instrument` itself stays (other strategies use it).
- `internal/service/trading_strategy/golden_x/notification/notifications.go` — render the per-share thresholds inline so a user can verify why an alert fired: `• Акция: Yandex 🟢 ✅` followed by `  RSI: 22  (p5=24, p15=31)`.
- `internal/service/trading_strategy/golden_x/notification/rsi_by_shares.go` — same `(p5, p15)` annotation in the intermediate RSI list; the colored circle in this list is now driven by `tierFromAdaptive(rsi, p5, p15)` for consistency.
- `internal/service/trading_strategy/golden_x/dto/thresholds.go` *(new)* — defines `Thresholds { P5, P15 float64 }`. Placed in `dto` (not in `golden_x`) for the same reason `TrendStatus` lives there: the `notification` sub-package must read it without importing the parent `golden_x` package (cyclic import). `trade.go` builds a `map[shareID]dto.Thresholds` and passes it to `notif.Trade(...)` and `notif.RSIList(...)` alongside the existing parallel trend map.
- `internal/domain/info.go` — **unchanged.** Percentile data flows through a parallel map (same trade-off and rationale as the C1 trend map), keeping `domain.Item` strategy-agnostic.

**Unchanged but worth knowing:**

- `internal/service/instrument/rsi/calculate.go` — kept as-is for other strategies (`bonds`, `scalping_rsi`). Not touched.
- `dto/trade.go` — no new fields. `UseTrendFilter` from C1 stays.
- `app/app.go` — no changes.

## Verification

1. `go build ./...` clean.
2. `go vet ./...` — no new findings (pre-existing telegram warning unrelated).
3. Unit tests:
   - `rsi_test.go`: `computeRSISeries` on the canonical Wilder-RSI fixture matches the existing `instrument/rsi` calculator on overlapping points (period 14, 30 closes). This protects against drift between the duplicate and the original.
   - `percentile_test.go`:
     - `percentile([1, 2, …, 100], 5) ≈ 5.95` (R-7 reference).
     - `percentile([42], 5) == 42` (single value).
     - `percentile([5, 5, 5, 5, 5], 50) == 5` (degenerate).
     - `tierFromAdaptive` truth table covering RSI < p5, RSI == p5, p5 ≤ RSI < p15, RSI == p15, RSI > p15.
   - `dedup_test.go`: rewritten to only exercise Yellow ↔ Green ↔ None transitions.
4. End-to-end `APP_ENV=dev` run, manual (cannot be automated — needs live Tinkoff API):
   - Growth message: each alert has the color + EMA200 mark (`✅`/`🚫`) + RSI value + `(p5=…, p15=…)`.
   - Gold message: each alert has color + RSI + `(p5=…, p15=…)` and no EMA200 mark.
   - For Yandex (or any share with < 100 closed weeks of history): not in either message; log line `"adaptive tiers: insufficient history"` present.
   - Tier-dedup invariant from Stage A is preserved: repeated runs without tier change do not re-send.

## Out of Scope

- Sell-side signal (Stage B): C2 computes only the lower tail.
- Pipeline / dependency injection refactor across all C-stage filters: each C-stage owns its own helpers; the pipeline shape (if any) is decided when implementing C3.
- Externalising the `5` and `15` constants to configuration (Stage D).
- Backtesting the new tier definitions against history.
