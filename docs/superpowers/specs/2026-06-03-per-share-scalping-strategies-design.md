# Per-share scalping strategies — design

Date: 2026-06-03
Branch: `feat/per-share-scalping-strategies`

## Problem

The current scalping strategy (`internal/service/trading_strategy/scalping`) applies
one universal rule to a dynamically ranked universe of the top-N most volatile shares.
Every share is evaluated with the same `Decide` logic and the same parameters.

We want the opposite shape: a **fixed, hand-picked set of shares, each with its own
trading logic**, because different shares "move" differently and may warrant entirely
different entry/exit rules. We start with a single share — RUSAL (`RUAL`).

## Decisions (locked in)

- **Variant A — different logic per share** (not just different parameters).
- The fixed per-share strategies **fully replace** the dynamic top-N universe scan.
  The universal RSI-scalping path and the `universe` package are removed.
- **Only the signal logic differs** per share. The runner keeps all I/O: loading
  shares, candles, positions, sending Telegram notifications, scheduling.
- **Approach B** — a strategy declares the data it needs; the runner hands it raw
  candle series; the strategy computes its own indicators from candles using the
  pure helpers in `pkg/indicators` (`ATR`, `RSISeries`, `VolumeConfirmed`) and
  `internal/domain/ema`. No per-strategy API calls.
- **Start with RUSAL only.** No `MaxOpenPositions` limit (single share, irrelevant),
  so `openCount` is dropped from the contract too.

## Architecture

The runner owns all side effects; each share is a small, pure Go unit implementing
one interface and computing its indicators from raw candles.

```
scalping/
  strategy/
    strategy.go     # Strategy interface + MarketData + Position
    registry.go     # DefaultRegistry() — the enabled shares
    rusal/
      rusal.go      # RUSAL strategy
      rusal_test.go # table-driven Decide tests
  model/            # Signal; Settings (trimmed to Interval)
  trade.go          # runner (rewritten)
  notification/     # unchanged
  scheduler/        # unchanged
  dto/              # unchanged
```

## Strategy contract

```go
// MarketData is the raw, per-instrument snapshot the runner hands to a strategy.
// All series are oldest-first and aligned to the same candles.
type MarketData struct {
    Price    float64
    Highs    []float64
    Lows     []float64
    Closes   []float64
    Volumes  []int64
    Position *Position // nil when flat
}

type Position struct {
    PurchasePrice float64
    Quantity      int64
}

type Strategy interface {
    Ticker() string                 // e.g. "RUAL"
    Lookback() int                  // number of candles it needs
    Decide(md MarketData) model.Signal
}
```

`Decide` stays a **pure function** — no ctx, no API. The strategy computes ATR/RSI/EMA
itself from `md` and computes its own take-profit / stop-loss. This is what lets each
share carry genuinely different logic without touching anything shared.

## Runner (rewritten `Trade`)

1. `Shares()` → build a `ticker → Share` map (so instrument UIDs are resolved, not
   hardcoded).
2. `GetPortfolio(accountID)` → `posByID` (unchanged position-reading logic).
3. For each strategy in the registry:
   - resolve its share by `Ticker()`; skip with a log if not tradable/found,
   - fetch `Lookback()` candles at `Settings.Interval` via `marketDataClient.GetCandles`,
   - build `MarketData` (with `Position` if held),
   - call `Decide`, collect non-`None` signals.
4. Send `notification.Trade(signals)` (unchanged). Log "no signals" when empty.

The strategy fills the trading fields of `Signal` (`Kind`, `Reason`, `Price`,
`TakeProfit`, `StopLoss`, `RSI`, `Ticker`). The runner enriches each returned signal
with `InstrumentID` / `InstrumentName` from the resolved share, since the strategy
does not know the instrument UID.

Removed: universe ranking, the in-runner EMA/RSI/ATR API calls (strategies compute
these now), the `universe` package, and the `UniverseSize` field.

## First strategy — RUSAL

Seed `rusal/` with the **current** logic so we have a working reference immediately:
EMA200 trend filter + upward RSI reversal through a level + ATR-based TP/SL. The
existing `decide_test.go` moves into `rusal/` as a table-driven test. From there the
logic is tuned to RUSAL's character.

## Settings

Trimmed to:

```go
type Settings struct {
    Interval enum.Interval
}
```

`DefaultSettings()` keeps `enum.Hour1`. `WithSettings` option preserved.

## What does NOT change

`scheduler/`, `notification/`, `dto/`, and the `service_provider` / `app` wiring. The
runner keeps the same `Scalping.Trade(ctx, dto.Trade)` interface, so nothing outside
the package is affected.

## Testing

- Each strategy's `Decide` is pure → table-driven unit tests (TDD), porting the
  existing RSI/ATR edge cases (sell-at-cap, no-signal) into `rusal/`.
- Runner: the rewritten `Trade` is covered by the existing client interfaces with
  fakes where it already is; the indicator-fetching branches collapse into a single
  candle fetch, simplifying the path.
