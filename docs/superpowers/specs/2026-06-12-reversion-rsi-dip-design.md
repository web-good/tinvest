# Reversion (RSI buy-the-dip) strategy — design

Date: 2026-06-12
Branch: `feat/reversion-rsi-dip`
Strategy name / `-strategy` flag / package: `reversion`

## 1. Purpose

A long-only hourly **mean-reversion** strategy: in a confirmed uptrend, buy sharp
RSI drawdowns and exit on the bounce. It is the conceptual mirror of `momentum`
(which buys breakouts/confluence); this one fades sharp dips back toward the mean.

It reuses the **entire** existing reporting layer with no engine or renderer
changes: `domain.Run` / `domain.Compute` / `RenderMarkdown`, the trades/equity CSV
writers, the grid calibration with walk-forward (`-test-months`), and `-explain`.
The only new code is a `core` package, per-ticker packages, a registry, one new
indicator (Stochastic), and a `case "reversion"` branch in `cmd/backtest/main.go`.

## 2. Layout (mirrors momentum)

```
internal/service/trading_strategy/reversion/strategy/
  core/        core.go, core_test.go        — pure decision core + impure shell
  afks/ gazp/ mdmg/ nvtk/ plzl/ rusal/ sber/ ydex/   — Ticker + DefaultParams each
internal/service/backtest/reversion_registry.go (+ _test)
pkg/indicators/stochastic.go (+ _test)       — NEW indicator
cmd/backtest/main.go                          — case "reversion"
docs/reversion/strategy.md                    — human-readable explainer
```

The 8 per-ticker packages mirror the momentum basket (AFKS, GAZP, MDMG, NVTK, PLZL,
RUSAL, SBER, YDEX), each exposing `Ticker` and `DefaultParams()`. A
`genericReversionDefaults()` provides a neutral baseline for any other ticker,
intentionally independent of any single ticker's defaults (same pattern as
`genericMomentumDefaults`).

## 3. Timeframe

Hourly (`Hour1`) by default, configurable via `-interval`, matching momentum. The
regime EMAs are computed on the trading-timeframe (hourly) closes — no daily/HTF
feed is required for this strategy.

## 4. Entry gates (short-circuit order)

Evaluated in this order; the first failing gate blocks entry and is what `Explain`
reports as the blocker. `decide` uses the same short-circuit order.

1. **Regime (uptrend):** `EMA(FastEMA) > EMA(SlowEMA)` **AND** `close > EMA(SlowEMA)`.
   Defaults 50 / 200. The second clause prevents buying dips that have already
   broken the major trend.
2. **Dip trigger** (`EntryMode`):
   - `EntryMode = 0` (**confirmed**, default): RSI(`RSIPeriod`, default 6) crosses
     **up** through `RSIOversold` (default 40): `rsiPrev <= RSIOversold < rsiNow`.
     The up-cross itself implies RSI was below the level on the previous bar, so a
     dip occurred and the bounce has started. Edge-triggered (fires once per cross).
   - `EntryMode = 1` (**knife**): RSI crosses **down** through `RSIOversold`:
     `rsiPrev >= RSIOversold > rsiNow`. Edge-triggered.
3. **Volume:** `VolumeConfirmed(volumes, VolLookback, VolMultiplier)` — last volume
   exceeds `VolMultiplier × SMA(VolLookback)`. Required (no entry below average).
4. **Stochastic confirmation (optional, `UseStoch` 0/1, default 0):** when
   `UseStoch = 1`, require `%K < StochOversold` (default 20) — price in the
   oversold zone. When `UseStoch = 0`, the gate is skipped entirely.
5. **Protective-stop sanity:** `StopLossPct > 0` is required (safety is not
   optional). The frozen stop is `entry × (1 − StopLossPct)`; `risk = entry ×
   StopLossPct > 0`. If `StopLossPct <= 0`, entry is blocked ("стоп не задан").

On a Buy: `Signal.StopLoss = entry × (1 − StopLossPct)`, `Signal.ATR = ATR(ATRPeriod)`
(display only — see §7), and `Signal.EntryReason` is a human-readable rationale
listing each gate's value.

## 5. Position management (manage — first trigger wins; protective first)

Checked in this order on each bar an open position exists:

1. **Hard stop:** `barLow <= StopLoss` (frozen `entry × (1 − StopLossPct)`) → reason `SL`.
2. **Time-stop:** `barsInPosition >= MaxHoldBars` → reason `TIME` (the bounce never
   came; exit at close). Disabled when `MaxHoldBars <= 0`.
3. **RSI take-profit:** RSI crosses **up** through `RSIOverbought` (default 70):
   `rsiPrev < RSIOverbought <= rsiNow` → reason `RSI`. This is the "exit above 70"
   thesis exit.

No trailing stop or partial exits in v1 (YAGNI; the thesis exit is RSI ≥ overbought).

## 6. Params

All fields are `int` or `float64` (flags as int 0/1) so reflection-based grid
calibration can sweep them.

| Field | Type | Role |
|---|---|---|
| `FastEMA` | int | fast regime EMA (default 50) |
| `SlowEMA` | int | slow regime EMA + price floor (default 200) |
| `RSIPeriod` | int | RSI length (default 6) |
| `RSIOversold` | float64 | dip-trigger level (default 40) |
| `RSIOverbought` | float64 | exit level (default 70) |
| `EntryMode` | int | 0 = confirmed up-cross (default), 1 = knife down-cross |
| `VolLookback` | int | SMA window for the volume baseline |
| `VolMultiplier` | float64 | last volume must exceed this × SMA |
| `UseStoch` | int | 1 = require Stochastic oversold confirmation; 0 = skip |
| `StochPeriod` | int | %K lookback |
| `StochSmooth` | int | %D smoothing of %K |
| `StochOversold` | float64 | %K oversold threshold (default 20) |
| `StopLossPct` | float64 | hard stop = entry × (1 − StopLossPct); must be > 0 |
| `MaxHoldBars` | int | time-stop bar count; <= 0 disables |
| `ATRPeriod` | int | ATR length — display only, never gates logic |

## 7. Purity & state

- `decide(in decideInput) model.Signal` is a pure function over already-computed
  indicator values.
- The impure `Strategy` shell holds one mutable counter, `barsInPosition`: it
  increments on each bar an open position exists and resets to 0 when flat
  (`pos == nil`). This mirrors how the momentum shell carries `barsSinceMACDCross`.
  `Position` carries no bars-held field, so this counter supplies the time-stop.
  Not safe for concurrent use; the backtest and live runners drive `Decide`
  sequentially, one bar at a time.
- `ATR` is computed only to populate `Signal.ATR` for the trade journal; it is not
  read by any gate or exit. (`Signal.ATR` documents "0 when n/a", so this is purely
  to keep report parity with momentum/levels.)
- `Lookback()` = max(`SlowEMA`, `RSIPeriod + 1`, `VolLookback + 1`, `ATRPeriod + 1`,
  `StochPeriod + StochSmooth`) + a small buffer.

## 8. New indicator: Stochastic

`Stochastic(highs, lows, closes []float64, kPeriod, dSmooth int) (k, d float64)` in
`pkg/indicators`, returning the latest %K and %D. %K = 100 × (close − lowestLow) /
(highestHigh − lowestLow) over `kPeriod`; %D = SMA of %K over `dSmooth`. Modeled on
the existing `RSISeries` / `ADX` style. Returns 0 when history is insufficient.

## 9. Registry & wiring

- `reversion_registry.go`: `reversionBindingFor(ticker, defaults)` builds a `Binding`
  (`DefaultParams` / `Build` via `core.NewWithParams` / `ParseParams` from JSON over
  defaults), a `reversionRegistry` map of the 8 tickers, `genericReversionDefaults()`,
  and `ReversionLookupOrGeneric(ticker)`.
- `cmd/backtest/main.go`: add `case "reversion": binding = svc.ReversionLookupOrGeneric(ticker)`
  to the strategy switch, and list `reversion` in the `-strategy` usage string and
  the "unknown strategy" error. No other command changes.

## 10. Explain

`Explain(md)` re-runs the entry gates in the same short-circuit order, printing each
gate's value and verdict (✓ pass / ✗ block), stopping at the first blocker — mirroring
momentum's `Explain`. Diagnostic only; never on the trading path; never mutates
`barsInPosition`.

## 11. Testing (TDD)

- `pkg/indicators/stochastic_test.go` — table-driven %K/%D correctness on reference
  series, including the insufficient-history (return 0) case.
- `core_test.go` — each gate in isolation (blocks / passes), both `EntryMode`
  values, all three exits, the protective-first ordering on a bar that hits both
  stop and RSI, the time-stop, `StopLossPct <= 0` rejection, and the optional
  Stochastic gate on/off.
- `reversion_registry_test.go` — defaults are valid (e.g. `StopLossPct > 0`,
  `SlowEMA > FastEMA`), and lookup of known vs generic tickers.

## 12. Out of scope for v1

`-basket` walk-forward (the basket runner in `cmd/backtest/main.go` is currently
hardcoded to momentum, reading `momentum_grid.json`). Generalizing it to select the
strategy is a separate iteration. v1 ships: single-run, `-calibrate` (with
`-test-months`), and `-explain`.
