# Reversion strategy (RSI + Stochastic dual confirmation)

Long-only, **daily timeframe** (`-interval Day1`). A mean-reversion core driven by the
agreement of two oscillators — RSI and the Stochastic %D line. It buys when one oscillator
is already inside its oversold zone and the other crosses into it. It exits on three signals
— RSI crossing 50 downward, a flag-selected middle exit (RSIOS or ATR stop), and a bearish
FastEMA/SlowEMA cross. Two optional entry filters narrow the buys: a trend filter (confirmed
uptrend only) and a volume filter (above-average entry-bar volume only).

The Stochastic working line is **%D** (SMA of %K over `StochDSmooth`; `StochDSmooth=1`
gives the raw %K). The time-stop and the mandatory fixed-% stop from earlier versions are
gone; volume gating returns only as the optional `UseVolume` entry filter.

## Entry (gates in short-circuit order)

1. **Trend filter (optional, `UseTrend`):** `1` (default) requires
   `EMA(FastEMA) > EMA(SlowEMA)` and `close > EMA(SlowEMA)` (defaults 50/200); `0` ignores
   trend.
2. **Dual oversold confirmation** — at least one of:
   - RSI(`RSIPeriod`) crosses **down** through `RSIOversold` **and** Stoch %D is already
     `< StochOversold`;
   - Stoch %D crosses **down** through `StochOversold` **and** RSI is already
     `< RSIOversold`.
   Both crossing into the zone on the same bar also fires. A warm-up bar without two valid
   oscillator readings can never satisfy this gate (the sentinel `0` is not treated as
   "in zone").
3. **Volume filter (optional, `UseVolume`):** `0` (default) ignores volume; `1` blocks the
   buy when the entry bar's volume is below `avg × VolMult`, where `avg` is the mean volume
   of the preceding `VolAvgPeriod` bars with weekend (Sat/Sun MSK) bars excluded and the
   entry bar itself excluded from its own average. The gate is skipped (entry **allowed**)
   when the baseline cannot be trusted — no per-bar timestamps means weekends are not
   excluded (the average still uses all preceding bars); no surviving sample or a
   non-positive entry-bar volume means no block. This degradation keeps the filter inert in
   live trading until per-bar timestamps are wired there.

## Exit (first trigger wins)

The middle exit is selected by the `UseATRStop` flag. An open long exits on one of three
signals, filled at the bar close, in this precedence order:

1. **RSI50:** RSI crosses the 50 line from above (`prev ≥ 50`, `now < 50`) — the primary
   momentum-fade exit.
2. **Middle exit (flag-selected):**
   - `UseATRStop = 0` → **RSIOS:** RSI crosses `RSIOversold` from above
     (`prev ≥ RSIOversold`, `now < RSIOversold`) — the failed-bounce exit. It cannot fire
     on the bar right after entry, where RSI is already below the zone.
   - `UseATRStop = 1` → **ATRSL:** price falls to/below `PurchasePrice − StopATRMult ×
     EntryATR`, where `EntryATR` is the daily ATR (length `ATRPeriod`) frozen at entry.
     Guarded by `EntryATR > 0` and `StopATRMult > 0`, so it stays inert in live trading
     (entry ATR is not persisted) and on a zero multiplier.
3. **EMAX:** bearish EMA cross — `EMA(FastEMA)` drops below `EMA(SlowEMA)`. A slow
   regime-break backstop; reuses the same EMAs as the trend filter.

If several fire on the same bar the earliest in this order is reported; the fill (close)
is identical either way.

## Run

```bash
# single run (daily timeframe)
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -months 12 -out ./reports/SBER

# grid calibration with walk-forward OOS (Stochastic zones/periods are swept)
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -calibrate data/params/sber/reversion_grid.json -out ./reports/SBER \
  -months 24 -test-months 6 -min-trades 20 -metric profit_factor

# diagnose one bar
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -explain '2026-03-14 12:00' -months 12
```

## Params

All 14 tunables live in `core.Params`. Every field is `int` or `float64` (flags encoded as
int `0/1`) so the reflection-based grid calibrator can sweep any of them. The RSI-50 exit
level is a fixed constant (`50`), **not** a param. Per-ticker starting values live in each
`reversion/strategy/<ticker>` package; calibrate with `-calibrate` and hardcode the winners.

**Trend filter**
- `UseTrend` — `1` (default) requires a confirmed uptrend before any buy
  (`EMA(FastEMA) > EMA(SlowEMA)` and `close > EMA(SlowEMA)`); `0` ignores trend.
- `FastEMA` — fast regime EMA length (e.g. 50). Doubles as the fast line of the `EMAX`
  bearish-cross exit.
- `SlowEMA` — slow regime EMA length and price floor (e.g. 200). Doubles as the slow line
  of the `EMAX` exit. Also the dominant term in the candle lookback window.

**RSI (entry trigger + exits)**
- `RSIPeriod` — RSI length; required (`> 0`).
- `RSIOversold` — RSI oversold zone used on the entry side (dual confirmation) and, when
  `UseATRStop=0`, as the `RSIOS` exit boundary.

**Stochastic (entry trigger)**
- `StochKPeriod` — Stochastic %K lookback; required (`> 0`).
- `StochDSmooth` — %D smoothing (SMA of %K); required (`> 0`). `1` = raw %K. The working
  line everywhere is %D.
- `StochOversold` — Stochastic oversold zone on the entry side.

**Middle exit selector (ATR stop)**
- `UseATRStop` — `0` (default): use `RSIOS` as the middle exit; `1`: use the ATR-based hard
  stop (`ATRSL`) instead.
- `ATRPeriod` — daily ATR lookback length; consulted only when `UseATRStop=1`.
- `StopATRMult` — stop distance multiplier: stop placed at `PurchasePrice − StopATRMult ×
  EntryATR`; default `1.0`; `0` disables the stop even when `UseATRStop=1`.

**Volume filter (entry)**
- `UseVolume` — `0` (default): no volume filter; `1`: block entries on below-average-volume
  bars.
- `VolAvgPeriod` — number of preceding bars averaged for the volume baseline (default `20`);
  weekend (Sat/Sun MSK) bars are excluded from that average, and the entry bar is excluded
  from its own average. Consulted only when `UseVolume=1`.
- `VolMult` — entry-volume threshold multiplier: a buy needs `entryVolume ≥ avg × VolMult`;
  default `1.0` (strictly at/above average). Raising it demands stronger participation.

## Not yet supported

`-basket` walk-forward (the basket runner is currently momentum-only).
