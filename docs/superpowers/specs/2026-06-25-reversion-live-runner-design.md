# Reversion Live Runner — Design

**Date:** 2026-06-25
**Branch:** `feat/reversion-rsi-dip`
**Status:** Approved design, pending implementation plan

## Goal

Run the `reversion` mean-reversion strategy against live Tinkoff Invest market data on a
dedicated account, behaving as close to the backtest as possible. The strategy buys a
full position when an entry signal fires and sells the full position when an exit signal
fires — no partial fills, no averaging, no partial closes. Real order placement and
Telegram notifications are each toggled independently via env. The strategy runs hourly
at the start of the hour over an env-configured ticker universe; per-ticker calibrated
parameters come from the same `DefaultParams()` packages the backtest uses.

## Background

Today `reversion` exists **only** inside the backtest CLI (`cmd/backtest`,
`internal/service/backtest/reversion_registry.go`). The decision engine
`core.Decide` (`internal/service/trading_strategy/reversion/strategy/core/core.go`) is a
pure function over a `MarketData` snapshot. There is no live service, scheduler, app
wiring, or order placement for it. Other strategies (`golden_x`, `scalping`) run live via
a per-strategy `service` + `scheduler` package wired into `internal/app/app.go`, fetch
candles by polling `GetCandles`, and currently only **notify** (no strategy calls
`PostOrder`). This design adds the first order-placing live runner, modelled on the
`scalping` pattern.

Production-accepted tickers at design time: **UGLD**, **EUTR**, **NVTK** (per the
reversion walk-forward results). The universe is env-configurable and will grow.

## Known, Accepted Divergence from Backtest

This is intrinsic and must be documented, not "fixed":

- **Entry fill timing.** The backtest fills an entry at the **close of the same bar** the
  signal fired (`internal/domain/backtest/engine.go:124-130`, `p.open(c.Close, …)`). The
  live runner wakes at the **start of the next hour**, after that bar has closed, and
  places a market order — which fills near the **open of the next bar**. There is an
  inherent ~1-bar offset plus slippage. On liquid names the gap is small, but live will
  be slightly worse than the backtest's (optimistic) close-fill. Same applies to exits.
- **Exit gap logic is not replicable.** The backtest models stop/TP gap fills as
  `min(StopLoss, nextOpen)` / `max(TakeProfit, nextOpen)` (`engine.go:131-146`). Live
  cannot know a future bar's open, so all exits are plain market orders at signal time.

Everything else — indicators, entry/exit gates, `EntryATR`, `MaxFavorablePrice` — is kept
1:1 by reusing `core.Decide` and the **same `MarketData` assembly code** as the backtest.

## Design Decisions (all confirmed with user)

| Topic | Decision |
|---|---|
| Position entry-state (`EntryATR`, `MaxFavorablePrice`) | Local state file is the primary source; reconstruct from API as a fallback when a position exists but state is missing, and alert via Telegram. |
| Real-trade vs notify toggles | Two **independent** env flags: `REVERSION_TRADE_ENABLED`, `REVERSION_NOTIFY_ENABLED`. |
| Candle data / warm-up | Fetch a fresh rolling window each run, sized for full indicator warm-up. No cache. |
| Order type | Market orders (`ORDER_TYPE_MARKET`). |
| Position sizing | `% of full account value` (cash + positions) at buy time; multiple positions allowed, one per ticker; skip + alert if cash insufficient. |
| Ticker universe & params | Universe via env list; parameters from the per-ticker `DefaultParams()` packages (single source shared with backtest). |
| Schedule — buy | Weekdays 08:00–23:00: cron `0 8-23 * * 1-5`. |
| Schedule — sell/manage | Daily 07:00–00:00 (night 01:00–06:00 skipped): cron `0 7-23,0 * * *`. |
| Dedicated account | `REVERSION_ACCOUNT_ID`, separate from `SCALPING_ACCOUNT_ID`. |
| Config mechanism | All via env (confita), consistent with the existing project config. |

## Architecture

Follows the `scalping`/`golden_x` pattern: a new live-runner package, a scheduler with two
cron jobs, and wiring in `app.go`. The engine (`core.Decide`) and `DefaultParams()` are
reused unchanged.

New package `internal/service/trading_strategy/reversion/live/` with small, focused units:

- **service** — orchestrates a single run (buy-pass or manage-pass); holds the ticker
  universe, config, injected clients, and the sub-components below.
- **statestore** — load/save `data/state/reversion_<account>.json`; per-ticker entry-state
  `{entryTime, entryPrice, entryATR, maxFav}`. Atomic write (temp file + rename).
- **sizing** — compute lot quantity from `BUY_PCT`, total account value, share lot size,
  and price. Returns 0 (skip) when the budget buys < 1 lot or cash is insufficient.
- **executor** — build and place a market `PostOrderRequest` (unique `order_id` UUID for
  idempotency) when `TRADE_ENABLED`; otherwise dry-run (no order). Returns the
  effective fill price/quantity for state recording.
- **reconstruct** — fallback that rebuilds entry-state from the API
  (`GetOperationsByCursor` to find the BUY that opened the current lot → entry time/price;
  `GetCandles` to recompute `EntryATR` at the entry bar and `maxFav` since entry), then
  persists and alerts.
- **marketdata** — assemble the `MarketData` snapshot from live candles (hourly with
  warm-up, 4H for the HTF filter, daily for ATR, volume baseline) using the **same code
  path the backtest uses**, so indicator inputs are identical. Warm-up bar counts per
  timeframe are derived from the active params (see Buy-pass step 1).
- **notifier** — Telegram messages for entries, exits, skips, and alerts.

Scheduler `internal/service/trading_strategy/reversion/live/scheduler/` registers two
cron jobs (buy-pass, manage-pass), mirroring `scalping/scheduler`.

### Reuse / refactor

- Extract or share the backtest's `MarketData` assembly so backtest and live build the
  snapshot identically. This is the crux of "max like backtest" and must be verified
  during planning (confirm exactly which `MarketData` fields `core.Decide` consumes:
  candle window, daily closes, 4H series, volume series).
- `EntryATR` for a new entry comes from the signal itself (`sig.ATR`, as the engine uses
  in `p.open`), not a separate recomputation.

## Run Flow

### Buy-pass (per ticker with no open position)

1. Fetch fresh candle windows sized for full indicator warm-up across **all** timeframes
   and build `MarketData` with no position. The warm-up must cover the longest lookback in
   the active params, per timeframe: hourly (max of `SlowEMA`, `RSIPeriod`, `StochKPeriod`,
   `VolAvgPeriod`, `ADXPeriod` with a safety buffer), 4H (`HTFTrendEMA` — e.g. NVTK's 150 ⇒
   ~150 4H bars ≈ 600 hourly), and daily (`ATRPeriod`). Compute the required bar counts from
   the ticker's params rather than hardcoding a single multiple.
2. `core.Decide(md)`; if `SignalBuy`:
   - lots = `floor(BUY_PCT% × accountValue / (price × lot))`; if 0 or cash insufficient →
     skip + alert.
   - place market BUY if `TRADE_ENABLED`, else notify/log only.
   - record state: `entryPrice = actual fill`, `entryATR = sig.ATR`, `maxFav = entryPrice`,
     `entryTime = now`.
   - notify (if `NOTIFY_ENABLED`).

### Manage-pass (per ticker with an open position from portfolio + state)

1. If the portfolio holds a position but state is missing → reconstruct from API + alert.
2. Build `MarketData` with `Position{PurchasePrice, EntryATR, MaxFavorablePrice}` from
   state; update `maxFav` from the latest completed candle; persist.
3. `core.Decide(md)`; if `SignalSell`:
   - place market SELL of the full position if `TRADE_ENABLED`, else notify/log only.
   - delete state entry.
   - notify (if `NOTIFY_ENABLED`).

### Flag matrix

| `TRADE_ENABLED` | `NOTIFY_ENABLED` | Behaviour |
|---|---|---|
| off | on | Paper mode: signals to Telegram only, no orders. |
| on | on | Live trading + Telegram notifications. |
| on | off | Live trading, silent (orders only). |
| off | off | Dry run to log only. |

### Edge cases

- Position in portfolio for a ticker **not** in the configured universe → left untouched
  (not ours).
- Market closed / no fresh completed candle (weekend, off-session in the manage window) →
  no action; only mark-to-market/observe.
- Order rejected by broker (e.g. session closed) → log + alert; state unchanged (no buy
  recorded; sell retried next tick).
- Broker reports a quantity different from requested → record the actual filled quantity.

## Changes to Existing Code

- `internal/config/` — add `ReversionConfig` (account id, tickers, buy pct, trade/notify
  flags) and register it on `Config`.
- `internal/service_provider/` — add a reversion live-service getter, injecting
  Instruments/MarketData/Operations/Orders clients + Telegram.
- `pkg/client/grpc/operations_service_client.go` — extend to expose per-trade fill data
  (date + price) from `GetOperationsByCursor`, needed by `reconstruct`.
- `internal/app/app.go` — start the two workers in `runProd` (optionally `runDev`).
- Refactor: share the `MarketData` assembly between backtest and live.

## Testing

- **Unit (pure, table-driven):**
  - sizing: pct/lot/price → lots, including skip cases (sub-lot budget, insufficient cash).
  - statestore: round-trip load/save, atomic write, missing-file behaviour.
  - executor: builds correct `PostOrderRequest` (direction, market type, lots, order_id)
    for buy and sell; dry-run places no order.
  - flag matrix: trade/notify combinations drive the right side effects (use fakes).
  - reconstruct: given fake operations + candles, rebuilds expected `EntryATR`/`maxFav`.
- **Parity test (key):** feed the same candle series through the backtest `MarketData`
  assembly and the live `marketdata` unit; assert identical snapshots/decisions, proving
  live reuses backtest inputs 1:1.
- Clients (gRPC, Telegram) are faked behind narrow interfaces; no live API calls in tests.

## Out of Scope (YAGNI)

- Partial fills, averaging/DCA, partial closes.
- Limit orders, order replacement, pending-order tracking.
- Persistent on-disk candle cache.
- Emulating the backtest's gap-fill exit math.
- More than one position per ticker.
