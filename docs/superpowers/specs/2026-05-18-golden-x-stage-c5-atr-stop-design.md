# Golden X Stage C5 — ATR-Based Stop Suggestion

## Context

Branch `feature/golden-x-stage-a` already carries Stage A (basic fixes), Stage C1 (EMA200 W trend filter for Growth), Stage C2 (adaptive p5/p15 buy tiers), Stage B (adaptive sell tiers — p80/p90/p95 Gold + p90 Growth), Stage C3 (bullish RSI divergence badge), and Stage C4 (volume confirmation badge). The master plan (`~/.claude/plans/internal-service-trading-strategy-golde-woolly-quill.md`) lists **C5 — ATR-based stop & position sizing recommendation** next.

**Why now.** Entry quality stages (C1–C4) tell the user *when* and *whether* to take a buy signal. C5 closes the loop with *where to invalidate* it — a concrete stop level derived from recent weekly volatility. Without it, every alert leaves the exit decision to ad-hoc judgment, which is the most common source of letting losers run.

**Outcome (this iteration).** Each buy-tier row in the Telegram alert gains a new line `  Stop: <price> (−<pct>%)` showing the ATR-derived stop level and its distance from the current close. Position sizing (deposit × risk%) is **out of scope** for this iteration — it would require a new deposit/risk config surface that the project does not have yet, and the user explicitly chose the "stop + risk-distance %" presentation. Everything else (tiers, trend, thresholds, dedup, sell side, RSIList) is unchanged.

## Locked decisions

| Question | Decision |
|---|---|
| What appears in the alert | **Stop price + distance %.** No lot count, no ruble risk. |
| Stop formula | `stop = lastClose − k × ATR(period=14)`. Long-only (strategy buys on oversold). |
| Multiplier k | **2.0 for Gold (Dividend), 1.5 for Growth.** Gold holds longer → wider stop survives weekly noise; Growth exits sooner → tighter. |
| ATR period | **14** (Wilder standard). Computed on the same closed weekly candles already used for RSI. |
| ATR algorithm | **Wilder ATR**: seed = SMA of first 14 True Range values, then `ATR_i = (ATR_{i-1}×13 + TR_i) / 14`. |
| Data source | **No new RPC.** Reuse the `closed` slice already in `trade.go`. |
| Scope across tiers | **Both 🟡 (Yellow) and 🟢 (Green) buy rows.** User wants to see risk distance before entering as well as at entry. |
| Scope across instances | **Both Gold and Growth.** Universal TA primitive. |
| Scope across sides | **Buy side only.** Sell rows and RSIList unchanged. |
| Minimum history | **15 closed weekly candles** (period + 1 for the first TR). Fewer → silent empty `Stop{}`, no error log. In practice `adaptiveWindowMin=100` already enforces this upstream. |
| Helper location | **`pkg/indicators/atr.go`** — second reusable TA primitive after `VolumeConfirmed`. Mirrors C4 pattern; does NOT modify `internal/service/instrument/atr` (intraday-shaped legacy). |
| Stop carrier type | New `dto.Stop{Price float64; DistancePct float64}`. Zero value = "no stop computed", rendered as empty. |
| Notification line | `  <b>Stop:</b> 2410.50 (−6.2%)\n` placed immediately after the RSI line. |
| Dedup | **Unchanged.** Tier-only. A change in stop level between ticks without a tier change does **not** re-alert. |

## Architecture overview

```
trade.go                          stop.go                          pkg/indicators/atr.go
─────────                         ───────                          ──────────────────────
for share in ShareList:           kForKind(kind) -> float64        ATR(highs, lows, closes,
  closed := closedWeeklyCandles   stopFromATR(close, atr, k)            period) -> float64
  rsi, ...                            -> dto.Stop                  // Wilder ATR, last value.
  if buyTier != tierNone:
    if VolumeConfirmed(...): ...
    atr := indicators.ATR(...)
    stops[id] = stopFromATR(...)
  ...

notif.Trade(..., stops)
  for id, log in buyInfo:
    write "• Акция: ... <badges>"
    write "  RSI Value: ..."
    write stopLine(stops[id])
    write "\n"
```

The ATR computation is a pure function. The Golden X wrapper file (`stop.go`) holds two small policy helpers — `kForKind` (StrategyKind → multiplier) and `stopFromATR` (close, atr, k → `dto.Stop`) — so `trade.go` stays a thin orchestrator. The notification renderer gains one extra map argument and one extra helper.

## Component details

### `pkg/indicators/atr.go`

```go
// ATR returns Wilder's Average True Range computed on the input series.
//
// Returns 0 when period <= 0 or len(closes) < period+1 — insufficient history
// is silent (no error), mirroring VolumeConfirmed. The three input slices must
// be aligned (highs[i], lows[i], closes[i] all describe the same bar). Length
// mismatches are treated as insufficient history and return 0.
//
// Algorithm:
//   - True Range at bar i (i >= 1):
//       TR_i = max(High_i - Low_i, |High_i - Close_{i-1}|, |Low_i - Close_{i-1}|)
//   - Seed ATR_{period} = SMA of TR_1 .. TR_{period}.
//   - For i > period: ATR_i = (ATR_{i-1} * (period - 1) + TR_i) / period.
//   - Return ATR_{last index}.
func ATR(highs, lows, closes []float64, period int) float64
```

Tests (`pkg/indicators/atr_test.go`), table-driven:
- Constant TR series (each bar high-low = 2, no gaps): ATR == 2 regardless of period (within ±1e-9).
- Known textbook example with hand-computed expected value.
- `period <= 0` → 0.
- `len(closes) == period` → 0 (need period+1).
- `len(closes) == period + 1` → returns the seed (single TR).
- Length-mismatched slices → 0.
- Empty / nil → 0.

### `internal/service/trading_strategy/golden_x/dto/stop.go`

```go
package dto

// Stop is the ATR-based stop suggestion attached to a buy-tier row.
// Zero value means "not computed" and renders as nothing in the notification.
type Stop struct {
    Price       float64 // absolute stop level in instrument's currency
    DistancePct float64 // (lastClose - Price) / lastClose * 100; always positive when populated
}
```

No `IsEmpty()` method — the renderer checks `Price == 0 && DistancePct == 0` inline (same convention as `Thresholds` and `SellThresholds`).

### `internal/service/trading_strategy/golden_x/stop.go`

```go
package golden_x

import "tinvest/internal/service/trading_strategy/golden_x/dto"

// kForKind returns the ATR multiplier for the given StrategyKind.
// Defaults to Gold's multiplier for unknown kinds (defensive — Trade enforces
// the enum at the call site).
func kForKind(kind dto.StrategyKind) float64 {
    if kind == dto.StrategyKindGrowth {
        return goldenXATRMultiplierGrowth
    }
    return goldenXATRMultiplierGold
}

// stopFromATR composes a dto.Stop from the last close, ATR value, and
// multiplier. Returns the zero Stop{} when atr or lastClose are non-positive,
// or when the computed price is <= 0 (would never make sense as a stop).
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

Tests (`stop_test.go`):
- `kForKind`: Gold → 2.0; Growth → 1.5; unknown (zero value) → 2.0.
- `stopFromATR`: normal positive inputs return non-empty Stop with correct math; `atr=0`, `lastClose=0`, oversized `k*atr` (price <= 0) all return `dto.Stop{}`.

### `internal/service/trading_strategy/golden_x/trade.go`

New constants colocated with `volumeMultiplier`, `divergenceFractalK`, etc.:

```go
const goldenXATRPeriod = 14
const goldenXATRMultiplierGold = 2.0
const goldenXATRMultiplierGrowth = 1.5
```

New map declared alongside `volumesConfirmed`:

```go
stops := make(map[string]dto.Stop)
```

New computation inside the existing `if buyTier != tierNone { ... }` block, after the volume check:

```go
highs := make([]float64, len(closed))
lows := make([]float64, len(closed))
closes := make([]float64, len(closed))
for i, c := range closed {
    highs[i]  = utils.CombinePrice(c.High.Units, c.High.Nano)
    lows[i]   = utils.CombinePrice(c.Low.Units, c.Low.Nano)
    closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
}
if atrValue := indicators.ATR(highs, lows, closes, goldenXATRPeriod); atrValue > 0 {
    lastClose := closes[len(closes)-1]
    stops[share.ID] = stopFromATR(lastClose, atrValue, kForKind(in.Kind))
}
```

Final `notif.Trade(...)` call gains `stops` as the last argument.

### `internal/service/trading_strategy/golden_x/notification/notifications.go`

Signature update:

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
) string
```

Inside the buy-section loop, immediately after the `RSI Value:` line:

```go
b.WriteString(stopLine(stops[id]))
```

New helper next to `divergenceBadge` and `volumeBadge`:

```go
// stopLine renders the ATR-derived stop suggestion as its own line, e.g.
// "  <b>Stop:</b> 2410.50 (−6.2%)\n". Empty Stop{} renders "" — keeps the
// indicator additive and silent on insufficient history.
func stopLine(s dto.Stop) string {
    if s.Price == 0 && s.DistancePct == 0 {
        return ""
    }
    return fmt.Sprintf("  <b>Stop:</b> %.2f (−%.1f%%)\n", s.Price, s.DistancePct)
}
```

Tests (`notifications_test.go`) add three cases:
- Buy row with a populated `Stop` renders a line containing `Stop:` and the formatted price.
- Buy row with empty `Stop{}` renders no `Stop:` substring for that row.
- Sell-section row never renders `Stop:` even if `stops` is non-nil for that share ID (defensive — stop ID space is buy-only by construction, but the test pins the renderer to the buy section).

## Files touched

Creates:
- `pkg/indicators/atr.go`
- `pkg/indicators/atr_test.go`
- `internal/service/trading_strategy/golden_x/dto/stop.go`
- `internal/service/trading_strategy/golden_x/stop.go`
- `internal/service/trading_strategy/golden_x/stop_test.go`

Edits:
- `internal/service/trading_strategy/golden_x/trade.go` — three constants, `stops` map, buy-block computation, extended `notif.Trade(...)` call.
- `internal/service/trading_strategy/golden_x/notification/notifications.go` — extra parameter, `stopLine` helper, buy-loop writer call.
- `internal/service/trading_strategy/golden_x/notification/notifications_test.go` — three new test cases.

Not touched:
- `internal/service/instrument/atr/*` — intraday-shaped legacy service, irrelevant to weekly Golden X.
- `dto.Trade` — public API of the strategy is unchanged; the multiplier picks itself from `Kind`.
- `internal/app/app.go` — no change to instance wiring.
- Sell section, RSIList, dedup (`alertState`), `domain.Info`.

## Verification

1. `go build ./...` — package compiles.
2. `go test ./pkg/indicators/...` — Wilder ATR table-driven tests pass.
3. `go test ./internal/service/trading_strategy/golden_x/...` — `stop_test.go`, existing tests, and updated `notifications_test.go` all pass.
4. Manual E2E (only the user, requires live Tinkoff): with `APP_ENV=dev`, trigger a buy alert on at least one Gold and one Growth instrument and confirm:
   - Buy row contains `Stop: <price> (−<pct>%)` immediately after the RSI line.
   - Same share, when run as Growth vs Gold, has the smaller stop distance for Growth (k=1.5 vs k=2.0).
   - Shares without enough history (none expected) render no stop line.

## Out of scope (deferred)

- **Position sizing in lots/rubles.** Requires a deposit + risk% config surface (`GOLDEN_X_DEPOSIT_RUB`, `GOLDEN_X_RISK_PCT`) which does not exist yet. Once introduced, the `dto.Stop` struct can grow `Lots int` and `RiskRub float64` fields with no breaking change to the renderer signature.
- **Dynamic / trailing stops** (Chandelier exit with running maximum). C5 is entry-time only.
- **Stops on the sell side.** Sell rows already have their own three-tier alert — overlaying stops there is a separate design question.
- **Per-share k tuning.** Single Gold-vs-Growth split is sufficient at this stage.
