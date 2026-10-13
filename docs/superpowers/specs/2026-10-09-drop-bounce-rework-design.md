# drop_bounce rework — oversold drop bar, indicator exits (design)

Date: 2026-10-09. Status: approved in conversation, pending spec review.

## Goal

Replace the drop_bounce rules wholesale. The old core (body-share target/stop, drop series,
confirmation bar, trend EMAs, pump filter, time exit, 5m/15m/1h literals for GAZP and T) is
removed. The new strategy is a 15-minute long-only bounce: buy after a large red bar that closes
oversold on a short RSI, hold — overnight if needed — until an indicator exit, the first-bar
profit exit or a percentage stop fires.

## Rules

Timeframe: 15m only (`-strategy drop_bounce -interval Minutes15`); other intervals are rejected.

### Entry

On the last closed bar `i` (the signal bar):

1. Red bar with `(open − close) / open · 100 ≥ MinDropPct` (default 0.6).
2. `RSI(RSIPeriod)` at bar `i`'s close `< RSIEntry` (defaults 6 and 30). Wilder RSI from
   `pkg/indicators.RSISeries`; too little history means no signal.
3. Window: bar `i` starts on Mon–Fri MSK at or after `StartHour:00` (default 8), and the fill bar
   (bar `i` start + 15 minutes) lies on the same MSK date — a signal never carries to the next
   morning's open.
4. No position open (new signals are ignored while holding).

Fill: the open of the next bar (`Config.EntryAtNextOpen`, already wired for drop_bounce in
cmd/backtest). The signal carries no TakeProfit and no StopLoss; all levels are measured from the
actual fill price `entry = Position.PurchasePrice`.

### Exits

Checked on every bar while holding, the fill bar included, in this order; the first that fires
sells:

1. **SL** — `low ≤ entry · (1 − StopPct/100)` (default 1). The sell signal carries that stop
   price; the engine fills at `min(stop, open)`, so an overnight gap below the stop sells at the
   open. `StopPct ≤ 0` disables the stop.
2. **PROFIT** — only on the fill bar (the bar whose time equals `Position.EntryTime`): its close
   `≥ entry · (1 + ProfitPct/100)` (default 1). Sells at the close. Later bars never use this
   rule. `ProfitPct ≤ 0` disables it.
3. **MACD** — histogram `h = MACD − signal` with `MACDFast/MACDSlow/MACDSignal` (defaults
   9/14/3, `pkg/indicators.MACD`). Green = `h > 0`, red = `h < 0`. The colour of bars before the
   entry does not matter. The exit fires on the first closed red bar after a green bar that
   closed while holding (fill bar or later). Until a green bar appears during the trade, red bars
   never exit. Sells at the close. `MACDExit = 0` disables it.
   Stateless check: if the position is still open, no red bar has followed a green one since
   entry (it would have exited). So the exit fires when `h[i] < 0` and the most recent bar
   `j < i` with `h[j] ≠ 0` among bars at or after the entry time has `h[j] > 0`.
4. **RSI** — `RSI(RSIPeriod)` crosses `RSIExit` upward: `rsi[i−1] < RSIExit ≤ rsi[i]` (default
   70). The previous bar may predate the entry. Sells at the close. `RSIExit ≤ 0` disables it.
5. **STOCH** — smoothed %K crosses `StochExit` upward: `k[i−1] < StochExit ≤ k[i]` (default 80).
   %K is `pkg/indicators.StochasticSeries(highs, lows, closes, StochPeriod, StochSmooth)`'s
   `ds` series (raw %K smoothed by SMA(`StochSmooth`), TradingView 14/3); `StochSmooth = 1`
   means the raw %K. Defaults 14 and 3. Sells at the close. `StochExit ≤ 0` disables it.

No end-of-day exit: positions carry overnight and across weekends until one of the rules fires.

Exit reasons: `SL`, `PROFIT`, `MACD`, `RSI`, `STOCH`; each with a Russian `ExitReason` string in
the style of the old core (values that triggered it, entry price).

## Parameters (`core.Params`)

All fields int or float64 for reflection grid calibration.

| Field | Default | Meaning |
|---|---|---|
| `MinDropPct` | 0.6 | minimum red body of the signal bar, % of open |
| `RSIPeriod` | 6 | RSI length, entry and exit |
| `RSIEntry` | 30 | entry requires RSI below this |
| `RSIExit` | 70 | exit when RSI crosses this upward (0 = off) |
| `MACDExit` | 1 | 1 = MACD green→red exit on, 0 = off |
| `MACDFast`, `MACDSlow`, `MACDSignal` | 9, 14, 3 | MACD periods |
| `StochPeriod`, `StochSmooth` | 14, 3 | stochastic %K period and its SMA smoothing |
| `StochExit` | 80 | exit when %K crosses this upward (0 = off) |
| `ProfitPct` | 1 | fill-bar close profit exit, % (0 = off) |
| `StopPct` | 1 | stop below the fill price, % (0 = off) |
| `StartHour` | 8 | signal bars start at/after this MSK hour |

Invalid configurations (`MinDropPct ≤ 0`, `RSIPeriod ≤ 0`, `RSIEntry ≤ 0`) produce no entries.
Invalid MACD periods (`MACDFast ≥ MACDSlow`, any ≤ 0) or stochastic periods make
`pkg/indicators` return nil, which disables that exit rather than panicking.

## Lookback

Enough history for every indicator to converge and for the MACD scan to see the holding bars:
`max(4·RSIPeriod, 3·MACDSlow + MACDSignal, StochPeriod + StochSmooth) + 2`, floored at 64 bars
(16 hours of 15m bars). If the entry predates the window, the MACD scan covers the
whole window: the green→red pair it needs can only lie after the last red exit point, and a
still-open position means none has happened, so missing older bars can only matter when every
in-window holding bar has `h == 0` exactly.

## Code changes

- `internal/service/trading_strategy/drop_bounce/strategy/core/core.go` — rewritten: new
  `Params`, `DefaultParams`, `Strategy` (`NewWithParams`, `Ticker`, `Lookback`, `Decide`). The
  bar step is fixed at 15 minutes; `NewWithStep` goes away.
- `core/core_test.go`, `core/engine_test.go` — rewritten, TDD: one test per entry condition and per
  exit (including MACD "never green → red bars hold", "green before entry does not count",
  "green on the fill bar counts", PROFIT only on the fill bar, SL gap fill at the open, exit
  priority, disabled exits), plus an engine run through `internal/domain/backtest`.
- `strategy/gazp/`, `strategy/tbank/` and `data/params/drop_bounce/{gazp,t}/` — deleted (literals
  calibrated for the old rules). `docs/drop_bounce/2026-10-09-gazp-loss-analysis.md` — deleted
  (analysis of the old rules). The old version is preserved by a snapshot commit before the
  rework.
- `data/params/drop_bounce/grid_example.json` — a starter grid over the new fields.
- `internal/service/backtest/drop_bounce_registry.go` (+ test) — Minutes15 only; registry map
  `ticker → func() core.Params`, empty for now, `core.DefaultParams` fallback; partial JSON
  overrides as before.
- `cmd/backtest/main.go` — keep `EntryAtNextOpen` for drop_bounce, drop `SkipFillOutsideLevels`.
- `internal/domain/backtest` — remove `Config.SkipFillOutsideLevels`, `openOutsideLevels` and
  their tests (only the old drop_bounce used them; the new signal has no levels). Keep the
  `fillPending` next-open behaviour.
- `docs/drop_bounce/strategy.md` — rewritten for the new rules; CLAUDE.md drop_bounce line updated.

## Out of scope

Calibration of any ticker, ticker packages, live trading.
