# Reversion Live Runner — Operator Guide

**Status:** Live trading enabled (branch `feat/reversion-rsi-dip`)  
**Design spec:** `docs/superpowers/specs/2026-06-25-reversion-live-runner-design.md`

## Configuration

All configuration is via environment variables. Defaults are provided; override by setting env vars before running the app.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `REVERSION_ACCOUNT_ID` | *required* | Dedicated brokerage account ID for the reversion strategy. Must be a valid Tinkoff Invest account. |
| `REVERSION_TICKERS` | `UGLD,EUTR,NVTK` | Comma-separated list of tickers in the live universe. See [Ticker Universe](#ticker-universe) for registered tickers and constraints. |
| `REVERSION_BUY_PCT` | `10` | Position sizing as a percentage of total account value (cash + open positions). E.g., `10` means each buy targets 10% of current account equity. |
| `REVERSION_TRADE_ENABLED` | `false` | **Critical:** If `false`, no orders are placed; only logs and notifications fire (paper mode). Set to `true` only in production after full testing. |
| `REVERSION_NOTIFY_ENABLED` | `true` | If `true`, entries, exits, skips, and errors are posted to Telegram. Set to `false` for silent operation. |

**Example `env/local.env`:**

```bash
REVERSION_ACCOUNT_ID=your_account_id_here
REVERSION_TICKERS=UGLD,EUTR,NVTK
REVERSION_BUY_PCT=10
REVERSION_TRADE_ENABLED=false
REVERSION_NOTIFY_ENABLED=true
```

## Flag Matrix

The `TRADE_ENABLED` and `NOTIFY_ENABLED` flags are independent. Choose your mode:

| `TRADE_ENABLED` | `NOTIFY_ENABLED` | Behaviour |
|---|---|---|
| `false` | `true` | **Paper mode:** signals posted to Telegram only; no broker orders placed. Use for development and testing. |
| `true` | `true` | **Live trading + notifications:** orders placed, full Telegram reporting. Production mode. |
| `true` | `false` | **Live trading, silent:** orders placed, no Telegram output. Use if notifications are handled elsewhere. |
| `false` | `false` | **Dry run:** no orders, no notifications; only log output. For debugging and metrics collection. |

### Recommended Flow

1. **Development:** `TRADE_ENABLED=false`, `NOTIFY_ENABLED=true` (paper mode on Telegram).
2. **Testing:** `TRADE_ENABLED=false`, `NOTIFY_ENABLED=false` (dry run to logs only).
3. **Live:** `TRADE_ENABLED=true`, `NOTIFY_ENABLED=true` (full operation with alerts).

## Execution Schedule

The reversion runner executes two independent cron jobs:

### Buy-Pass Schedule

**Cron:** `0 8-23 * * 1-5`  
**Times:** 08:00, 09:00, …, 23:00 Moscow Time, Monday–Friday  
**Action:** Per ticker with no open position, evaluate entry signals and place buy orders if triggered.

### Manage-Pass Schedule

**Cron:** `0 7-23,0 * * *`  
**Times:** 07:00, 08:00, …, 23:00, 00:00 Moscow Time, every day (no gap 01:00–06:00)  
**Action:** Per ticker with an open position, evaluate exit signals and place sell orders if triggered.

Both passes run in production mode (`APP_ENV=prod`). In development mode (`APP_ENV=dev`), the workers run concurrently as goroutines with no schedule delay.

## State Management

### State File

The runner persists per-ticker entry state in a local JSON file:

**Path:** `data/state/reversion_<REVERSION_ACCOUNT_ID>.json`

**Format:** Object with ticker keys, each holding:

```json
{
  "UGLD": {
    "Ticker": "UGLD",
    "EntryTime": "2026-06-26T09:00:00Z",
    "EntryPrice": 2845.5,
    "EntryATR": 12.3,
    "MaxFav": 2857.8,
    "Quantity": 10
  },
  "EUTR": {
    "Ticker": "EUTR",
    "EntryTime": "2026-06-26T10:00:00Z",
    "EntryPrice": 1234.25,
    "EntryATR": 8.5,
    "MaxFav": 1234.25,
    "Quantity": 50
  }
}
```

**Fields:**

- `Ticker` — instrument code (e.g., `UGLD`).
- `EntryTime` — UTC timestamp of the buy order (when the position was opened).
- `EntryPrice` — average fill price of the buy order.
- `EntryATR` — ATR value at the time of entry; used by the exit logic to compute stop-loss and take-profit.
- `MaxFav` — maximum favorable price reached since entry; updated on each manage-pass tick for monitoring.
- `Quantity` — number of shares in the position (in lots for brokers with lot restrictions; see [Position Sizing](#position-sizing)).

### Atomic Writes

State files are written atomically (temp file + rename) to prevent corruption if the process crashes mid-write.

### Reconstruct Fallback

If the portfolio holds a position in the broker account but the state file is missing (e.g., after a state file loss or recovery), the runner:

1. Queries the broker API for past trades (`GetOperationsByCursor`).
2. Locates the most recent BUY order for that ticker in the current account.
3. Reconstructs `EntryTime`, `EntryPrice`, and `Quantity` from the trade.
4. Recomputes `EntryATR` by fetching historical candles and re-running the indicator.
5. Sets `MaxFav` to the current market price.
6. Persists the reconstructed state to disk.
7. Sends a Telegram alert: `"Reconstructed state for UGLD: entry time=..., price=..., ATR=..."`

This ensures the runner continues to function without manual intervention, and the operator is notified of the recovery.

## Position Sizing

Each buy order targets `REVERSION_BUY_PCT`% of the **total account value** (cash + marked-to-market open positions).

**Calculation:**

1. Fetch total account value (cash + position valuation) from broker.
2. Compute target amount: `accountValue × BUY_PCT / 100`.
3. Fetch current share price and lot size (1 for most Tinkoff Invest stocks).
4. Compute quantity in lots: `floor(targetAmount / (price × lotSize))`.
5. If `quantity == 0` or available cash is insufficient → skip order + alert.

**Example:**

- Account value: 1,000,000 rubles.
- `REVERSION_BUY_PCT=10` → target: 100,000 rubles.
- UGLD price: 2,800 rubles, lot size: 1.
- Quantity: `floor(100,000 / 2,800) = 35 lots`.
- Available cash: 150,000 rubles → sufficient, order placed.

If available cash < target amount → order skipped, Telegram alert sent.

## Position Limits

- **One position per ticker:** Multiple entries in the same ticker are not allowed. If a position exists, no new buy is placed until it is sold.
- **Tickers not in the universe:** Positions held for tickers not in `REVERSION_TICKERS` are left untouched (out of scope for the reversion runner).

## Order Placement

### Order Type

Market orders only (`ORDER_TYPE_MARKET`). Orders fill at the best available market price at the time of execution.

### Idempotency

Each order is assigned a unique UUID (`order_id`) for idempotency. If the same order is retried (e.g., due to a network timeout), the broker will recognize the duplicate `order_id` and not double-execute.

### Rejection Handling

If the broker rejects an order (e.g., market closed, insufficient margin, invalid instrument):

1. No state is recorded (no buy entry, no sell confirmation).
2. An error is logged.
3. If `NOTIFY_ENABLED=true`, a Telegram alert is sent.
4. The runner retries on the next scheduled tick.

### Fill Tracking

The runner records the actual fill price and quantity reported by the broker. If the quantity differs from the order request (partial fill in exceptional cases), the state stores the actual filled quantity.

## Known Divergence from Backtest

The live runner produces different fills than the backtest due to inherent real-time constraints:

### Entry Fill Timing

- **Backtest:** Fills an entry at the **close of the bar when the signal fires** (same-bar close).
- **Live:** Wakes at the **start of the next hour** and places a market order, which fills near the **open of the next bar** (~1-bar offset + slippage).

On liquid names (UGLD, EUTR), the gap is typically small. On less liquid tickers, live fills may deviate more noticeably from backtest PnL.

### Exit Fill Logic

- **Backtest:** Models stop-loss and take-profit gap fills as `min(StopLoss, nextOpen)` / `max(TakeProfit, nextOpen)` — it knows the next bar's open and gap-fills realistically.
- **Live:** All exits are plain market orders at signal time. The next bar's open is unknowable, so gap-fill math is not replicable.

**Impact:** Live results will not match backtest P&L exactly, especially on volatile/gapped names. Backtest results are a reference, not a guarantee.

## Ticker Universe

### Production Tickers (Tested and Approved)

| Ticker | Walk-Forward PF | Status |
|--------|-----------------|--------|
| `UGLD` | 3.39 | ✓ Production |
| `EUTR` | 2.529 | ✓ Production (manually tuned) |
| `NVTK` | 4.37 | ✓ Production |

These three are included in the default `REVERSION_TICKERS` and are ready for live trading.

### Registered Tickers (Not in Default Universe)

| Ticker | Walk-Forward PF | Status | Notes |
|--------|-----------------|--------|-------|
| `ASTR` | N/A (baseline) | Tuning | Added for calibration; do not use in production without explicit testing. |
| `SFIN` | Failed | ⛔ DO NOT TRADE | Walk-forward result was negative. Remove from `REVERSION_TICKERS` if present. |

If you add tickers beyond the default three, ensure you have run walk-forward backtests and are confident in the calibrated parameters.

### Adding a Ticker

1. Run a walk-forward backtest with the candidate ticker.
2. Inspect the out-of-sample Profit Factor (PF). Accept only if PF > 1.0 and showing consistent profitability across test windows.
3. Add the ticker to `REVERSION_TICKERS` in your `.env` file.
4. Deploy and monitor live P&L vs. backtest predictions.

## Monitoring and Alerts

### Telegram Notifications

When `NOTIFY_ENABLED=true`, the runner sends messages for:

- **Entry:** `"Entry UGLD at 2845.50, qty=35, ATR=12.3"`
- **Exit:** `"Exit UGLD at 2857.80, qty=35, PnL=+420.5R"`
- **Skip (insufficient cash):** `"Skip EUTR entry: budget=100k, cash=50k (insufficient)"`
- **Skip (signal but no position):** `"No sell signal for NVTK (no open position)"`
- **Reconstruct alert:** `"Reconstructed state for UGLD: entry=2026-06-26T09:00:00Z, price=2845.50, ATR=12.3"`
- **Error:** `"Order rejected for UGLD: market closed"`

### Logs

Detailed logs are output to stdout/stderr. In production, capture these to a file or log aggregator.

**Log levels:**

- `INFO` — normal operations (entries, exits, skips).
- `ERROR` — order rejections, state file I/O errors, API failures.
- `DEBUG` — candle fetches, indicator calculations, decision logic (dev mode only).

## Recovery and Edge Cases

### Market Closed

If no fresh completed candle is available (weekend, off-session), the manage-pass runs but takes no action (only mark-to-market is recorded). The runner does not fail; it waits for the next scheduled tick.

### Position in Portfolio, State Missing

The reconstruct fallback (see [State Management](#state-management)) handles this. An alert is sent via Telegram.

### Broker Quantity Mismatch

If the broker reports a fill quantity different from the requested quantity, the state records the actual quantity. This may indicate a partial fill (rare) or rounding by the broker. Verify in the broker app.

### Manual Position Changes

If you manually buy or sell a ticker in the account outside the runner, the runner may not detect the change immediately. The next manage-pass will check the broker portfolio. If a manual buy creates a position and the state is missing, reconstruct fires. If a manual close removes a position, the next buy-pass will attempt a new entry (if the signal fires).

**Best practice:** Avoid manual trades in the reversion account during live operation.

## Troubleshooting

### No Entries Are Triggering

1. Verify `REVERSION_TRADE_ENABLED=true` (not paper mode).
2. Check Telegram notifications; if disabled, enable `NOTIFY_ENABLED=true` to see skip reasons.
3. Run a backtest on the same timeframe/data to confirm the signal is expected.
4. Check logs for candle fetch errors or indicator calculation issues.

### Entries Trigger But No Orders Placed

1. If `TRADE_ENABLED=false`, orders are intentionally skipped (paper mode).
2. Check available cash. If the sizing calculation results in 0 lots, a Telegram alert is sent.
3. Check logs for broker API errors (e.g., "Order rejected: market closed").
4. Verify the broker account ID in `REVERSION_ACCOUNT_ID` is correct and has trading permissions.

### State File Corruption

Delete the state file (`data/state/reversion_<account>.json`). On the next manage-pass, the reconstruct fallback will rebuild it from the broker API.

### Stale Data / No Candles

If candle fetches fail (API error, network timeout), logs will show the error. The runner retries on the next scheduled tick. There is no persistent candle cache; each tick fetches fresh data.

## See Also

- **Design Spec:** `docs/superpowers/specs/2026-06-25-reversion-live-runner-design.md`
- **Strategy Guide:** `docs/reversion/strategy.md`
- **Screener Notes:** `docs/reversion/screener.md`
