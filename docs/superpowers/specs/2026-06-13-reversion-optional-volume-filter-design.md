# Reversion: optional vertical-volume entry filter (UseVolume)

Date: 2026-06-13
Branch: feat/reversion-rsi-dip
Package: `internal/service/trading_strategy/reversion/strategy/core`

## Problem

The reversion strategy enters a long on a confirmed dual oversold reading (optionally
behind a trend filter). It pays no attention to how much the entry bar actually traded.
A bounce signal that fires on a thin, low-participation bar is weaker than one backed by
above-average turnover.

We want an **optional** entry gate on the bar's vertical volume (per-bar traded volume,
not volume-by-price): at entry, look at the entry bar's volume; if it is below the
average volume, do not enter. When computing that average, **bars of weekend trading
sessions (Saturday/Sunday) are excluded** — MOEX weekend sessions trade at much lower
volume and would otherwise drag the baseline down.

The strategy runs on the daily timeframe (`-interval Day1`), so each bar is one calendar
day and a "weekend bar" is a daily candle whose date falls on Saturday or Sunday.

## Design

### Params (all int/float for grid calibration via reflection)

Add three fields to `core.Params`, mirroring the existing `UseTrend` / `UseATRStop`
optional-filter pattern:

- `UseVolume int` — `0` (default, off) keeps current behavior; `1` enables the volume
  gate. Mutually a toggle, like `UseTrend`.
- `VolAvgPeriod int` — number of **preceding** bars over which the average-volume
  baseline is computed (default `20`). Consulted only when `UseVolume == 1`.
- `VolMult float64` — threshold multiplier (default `1.0`): entry is allowed when
  `entryVolume >= avg * VolMult`. At `1.0` the rule is "strictly at/above average"; the
  multiplier is tunable so calibration can demand a stronger volume confirmation.

### Entry logic

The gate is added as the **last** entry filter, after the dual oversold confirmation.
Entry order becomes: trend (optional) → dual oversold → volume (optional). Placing it
last does not change correctness (all gates must pass); it only affects `Explain`
ordering and keeps the cheap data-driven check after the oscillator logic.

When `UseVolume == 1` and a baseline could be computed:

```
entryVol = Volumes[last]                       // the entry (current) bar's volume
avg      = mean of the last VolAvgPeriod bars BEFORE the entry bar, EXCLUDING Sat/Sun
block entry if entryVol < avg * VolMult
```

When `UseVolume == 0`, the gate is inert and entries behave exactly as today.

### Average-volume baseline

- The average is computed over the `VolAvgPeriod` bars **before** the entry bar; the
  entry bar itself is NOT included in its own average (no self-reference).
- Within that window, any bar whose timestamp (MSK) is Saturday or Sunday is dropped
  from the mean. The window is the `VolAvgPeriod` preceding bars by position; weekend
  bars inside it simply do not contribute to the sum/count (so the effective sample can
  be smaller than `VolAvgPeriod`).
- The baseline reports an `ok` flag: `ok == false` when no non-weekend bars with a
  positive volume remain in the window (degenerate). When `ok == false` the gate does
  NOT block — missing data must never silently suppress all entries.

### Weekday detection requires timestamps

`MarketData` currently carries no per-bar timestamps, so the core cannot tell which bars
are weekend bars. Add a `Times []time.Time` field to `strategy.MarketData` (additive;
oldest-first, index-aligned to `Closes`/`Volumes`) and populate it in
`backtest.buildMarketData` from `Candle.Time`. Other strategies ignore the new field.

The MSK location already used by the engine (`Europe/Moscow`, UTC fallback) is the
reference for the weekday; the core determines weekend-ness via `t.In(mskLoc).Weekday()`
∈ {Saturday, Sunday}. The core gets `mskLoc` from a small package-level loader mirroring
`engine.go`'s, so it has no dependency on the backtest package.

### Graceful degradation (live / incomplete data)

The filter must never block entries just because optional data is absent:

- **No `Times`** (`len(md.Times) == 0`, e.g. a live runner that does not populate it):
  weekend exclusion is skipped — every bar in the window counts. The average is still
  computed from `Volumes`; the gate still works, it just cannot drop weekend bars.
- **No baseline** (zero non-weekend, positive-volume bars in the window) or
  **entry volume not available** (`entryVol <= 0`): `volOK == false` → gate is skipped,
  entry allowed. Same discipline as the `rsiOK` / `emaOK` warm-up sentinels.

### Wiring

- `decideInput` gains three fields: `entryVol float64`, `avgVol float64`, `volOK bool`.
- `buildInput` computes them (next to where it computes `atr`), gated on
  `UseVolume == 1 && VolAvgPeriod > 0`; otherwise leaves them zero/false. It calls a new
  helper `averageVolumeExcludingWeekends(vols []int64, times []time.Time, period int)
  (avg float64, ok bool)` that:
  - takes the last `period` volumes **before** the final (entry) bar,
  - drops index `j` when `len(times) > 0 && isWeekend(times[j])`,
  - drops non-positive volumes,
  - returns `(0, false)` when no samples survive, else `(sum/count, true)`.
- `decide()` applies the gate in the entry path (after `entryFired`):
  `if s.p.UseVolume == 1 && in.volOK && in.entryVol < in.avgVol*s.p.VolMult { return sig }`.
- `Explain()` reports the volume gate as the final step (✓/✗) with the measured
  `entryVol`, `avgVol`, and threshold.

### Fill / exit semantics

Unchanged. This is an entry-side filter only; it never affects exits, fills, or position
management.

### Lookback / defaults / per-ticker / grid

- `Lookback()` accounts for `VolAvgPeriod + 1` when `UseVolume == 1 && VolAvgPeriod > 0`
  (need `VolAvgPeriod` preceding bars + the entry bar), under the same gate style as the
  existing ATR branch.
- Per-ticker defaults (all 8: `afks`, `rusal`, `gazp`, `ydex`, `mdmg`, `nvtk`, `plzl`,
  `sber`): add `UseVolume: 0, VolAvgPeriod: 20, VolMult: 1.0` — preserves current
  behavior (gate off by default).
- Each `data/params/<ticker>/reversion_grid.json` entry phase: add
  `"UseVolume": [0, 1]`, `"VolAvgPeriod": [20]`, `"VolMult": [1.0, 1.5]`. (RUSAL ticker
  is `RUAL`, data folder is `rual`.)

### Tests (`core_test.go`)

- `averageVolumeExcludingWeekends` excludes Sat/Sun bars from the mean (window with
  high-volume weekdays + low-volume weekend → average reflects weekdays only).
- `averageVolumeExcludingWeekends` excludes the entry (final) bar from its own average.
- `averageVolumeExcludingWeekends` returns `ok == false` when no valid samples remain.
- Gate blocks entry when `entryVol < avg * VolMult` (UseVolume=1).
- Gate allows entry when `entryVol >= avg * VolMult` (UseVolume=1).
- `VolMult > 1.0` raises the bar (an entry passing at 1.0 is blocked at 1.5).
- `UseVolume == 0` → no volume gating (entry that would be blocked still fires).
- Degradation: empty `Times` → weekend not excluded (gate still computes over all bars).
- Degradation: `volOK == false` → entry allowed (gate skipped).
- `buildInput` computes `entryVol`/`avgVol`/`volOK` only when the gate is on.
- `Lookback()` includes `VolAvgPeriod + 1` when `UseVolume == 1`.

### Docs

- Update the `core` package docstring (entry list now mentions the optional volume gate).
- Update `docs/reversion/strategy.md` Entry and Params sections to describe `UseVolume`,
  `VolAvgPeriod`, `VolMult`, and the weekend-exclusion rule.

## Out of scope

- Horizontal volume / volume-by-price (volume profile). This is per-bar vertical volume.
- Live persistence/population of `MarketData.Times` for the live runner — the filter
  degrades gracefully (no weekend exclusion) until that exists; not changed here.
- Any change to exits, stops, fills, or position management.
- Changes to the trend or dual-oversold entry gates.
