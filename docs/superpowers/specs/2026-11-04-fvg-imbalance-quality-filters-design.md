# FVG imbalance — quality filters and execution controls

## Intent

Improve the backtest-only, long-only GAZP M30 `fvg_imbalance` strategy without
silently replacing its current baseline. Add optional higher-timeframe trend,
liquidity-sweep, bullish-break-of-structure, and first-retest gates together
with optional entry-distance, stop-buffer, and time-exit controls. Calibration
must be able to measure every addition against the existing behavior.

The work does not assume that an FVG is profitable by itself and does not
promote a calibrated combination into GAZP defaults automatically. It produces
reproducible mechanics, a staged grid, and walk-forward evidence for a later
default-selection decision.

## Goals

- Preserve the current GAZP baseline when every new control is disabled:
  `ATRPeriod=14`, `MinGapATR=0.8`, `MaxRetestBars=8`,
  `RequireCloseAboveZone=0`, and `TakeProfitR=1.5`.
- Keep the strategy pure and stateless: every decision reconstructs eligible
  structure and zones from the supplied completed candles.
- Use only information that was confirmed by the decision timestamp.
- Support independent and combined calibration of the new gates.
- Continue next-M30-open execution and cancel an entry if that actual open is
  too far above the selected FVG.
- Measure holding time in processed M30 bars rather than wall-clock time.
- Keep missing or insufficient H4 data fail-closed when the H4 gate is enabled.

## Non-goals

- Live trading, persisted zones, scanners, short/bearish FVGs, trailing stops,
  partial exits, or automatic order placement.
- Changing the existing three-candle FVG formation rule, including its bullish
  middle-body crossing requirement.
- Automatically changing GAZP defaults to the best in-sample or walk-forward
  combination.
- Proving profitability or encoding a profit-factor threshold in production
  code.

## Constraints

- The strategy remains M30-only and backtest-only.
- All toggles remain numeric (`0` or `1`) for the reflection-based grid.
- Existing parameter JSON remains valid through partial overlay.
- Disabled new controls reproduce the old signals, fills, stops, and exits.
- New generic engine fields must be zero-value compatible for every existing
  strategy.
- Completed H4 candles are aligned by the existing backtest HTF cursor; an H4
  candle still forming at an M30 decision cannot be visible.
- Per-ticker calibration results stay in reports or parameter comments, not in
  `docs/fvg_imbalance/strategy.md`.

## Approaches considered

### 1. Stateless reconstruction with small generic engine extensions — chosen

Reconstruct fractals, sweep, BOS, FVG age, and first-touch status from each
`MarketData` window. Add a zero-compatible maximum-entry price to `Signal` and
expose the engine's existing bar-age calculation through `Position`.

This preserves deterministic replay, safe parallel calibration, and the
existing pure-strategy interface. The tradeoff is a larger lookback and careful
event-time rules.

### 2. Mutable zone state inside the strategy

Remember sweeps, structure breaks, and consumed zones between calls. This maps
directly to a trading state machine but makes replay, restart recovery, and
parallel parameter evaluation depend on hidden mutable state. Rejected.

### 3. FVG-specific state in the common backtest engine

Let the engine own zones and mitigation state. This would couple a generic
portfolio/execution component to one entry model and spread strategy semantics
outside the strategy package. Rejected.

## Parameters

Add these numeric fields to `core.Params`:

| Field | Baseline | Validation | Meaning |
|---|---:|---|---|
| `RequireFirstRetest` | `0` | exactly `0` or `1` | when enabled, only the first post-formation touch of the zone may signal |
| `RequireBullishBOS` | `0` | exactly `0` or `1` | require a bullish break of confirmed structure before/by FVG formation |
| `RequireLiquiditySweep` | `0` | exactly `0` or `1` | require a preceding bullish liquidity sweep |
| `SwingK` | `2` | positive | candles on each side of a confirmed fractal pivot |
| `StructureLookbackBars` | `48` | positive | maximum M30 history searched before FVG formation for sweep/BOS context |
| `MaxSweepToFVGBars` | `8` | positive | maximum `c-sweep` distance when the sweep gate is enabled |
| `HTFTrendEMA` | `0` | non-negative | EMA period on completed H4 closes; `0` disables the gate |
| `MaxEntryDistanceATR` | `0` | finite, non-negative | maximum next-open price above `upper`, in frozen formation ATRs; `0` disables |
| `MaxHoldBars` | `0` | non-negative | time exit after this many engine bars in position; `0` disables |
| `StopBufferATR` | `0` | finite, non-negative | stop distance below `lower`, in frozen formation ATRs |

Positive structural helper parameters remain required even when their gates are
off. This keeps partial JSON behavior deterministic and avoids configurations
whose validity changes when a toggle changes.

The core defaults use the baseline column above. GAZP repeats all fields
explicitly. Its existing `MinGapATR=0.8` and `RequireCloseAboveZone=0` are user
changes and must be preserved; the stale literal test, strategy documentation,
and calibration grid are reconciled with them.

## Confirmed fractals

For `SwingK=K`, pivot `i` is a confirmed swing high when its high is strictly
greater than every high in `[i-K, i)` and `[i+1, i+K]`. A swing low is defined
symmetrically with strict lower-than comparisons. Equal highs/lows do not form
a pivot.

A pivot is usable by event bar `j` only when `i+K <= j`; reconstruction may not
use later candles to pretend a pivot was known earlier. Searches choose the
most recent eligible pivot and stay inside `StructureLookbackBars` before the
FVG formation candle `c`.

## Liquidity sweep and bullish BOS

Let `a,b,c` be the existing FVG formation bars.

A bullish liquidity sweep at bar `s` requires:

- a most-recent confirmed swing low that was usable before `s`;
- `low[s] < swingLow`; and
- `close[s] > swingLow`.

Equality does not count as either the break or the reclaim. The sweep may be
the middle impulse candle `b`, but not the later formation candle `c`; therefore
`s <= b`. When `RequireLiquiditySweep=1`, use the newest qualifying sweep with
`1 <= c-s <= MaxSweepToFVGBars`.

A bullish BOS requires a close strictly above a confirmed swing high:

- with a required sweep, the reference is the most recent swing high usable at
  `s`, and a close on any bar in `[s,c]` may break it;
- without a required sweep, the reference is the most recent swing high usable
  at `a`, and either `b` or `c` must close above it.

When both gates are enabled the required order is `sweep -> BOS -> completed
FVG`; BOS may occur on the sweep candle itself when that candle both reclaims
the swing low and closes above the reference swing high. When only the sweep
gate is enabled, no structural-high break is implied. When only the BOS gate is
enabled, no sweep is implied.

## First-retest semantics

The current bar `k` remains a protected retest when
`lower < low[k] <= upper` and it satisfies the existing optional close-above
gate. With `RequireFirstRetest=1`, it may signal only if every earlier bar in
`[c+1,k)` has `low > upper`.

Any earlier touch consumes the zone for entry purposes, even when it did not
close above the zone, occurred before a weekend/missing-slot cancellation, or
would have failed another optional entry gate. A touch at or below `lower`
continues to invalidate the zone under the existing rule. This definition
prevents a second trade from the same reconstructed FVG without mutable state.

## H4 trend gate

`cmd/backtest` loads Hour4 candles with the existing one-year lead-in for
`fvg_imbalance`, sets `Config.HTFInterval=4*time.Hour`, and passes the completed
series through the existing HTF cursor.

When `HTFTrendEMA>0`, entry requires at least that many completed H4 closes, a
finite EMA, and:

```text
last completed H4 close > EMA(HTFCloses, HTFTrendEMA)
```

Missing, short, or non-finite H4 data blocks entry. `HTFTrendEMA=0` ignores H4
data completely. The gate is evaluated on the retest decision, not frozen at
FVG formation. Position exits never depend on the H4 gate.

## Entry-distance control

When `MaxEntryDistanceATR>0`, a buy signal carries:

```text
MaxEntryPrice = upper + MaxEntryDistanceATR * formationATR
```

Add `MaxEntryPrice float64` to the shared `model.Signal`. Zero means no maximum.
The backtest engine's pending-fill path drops the buy when the actual next bar
opens strictly above this price. Equality is fillable. The engine performs this
check together with `FillBy`, before opening the portfolio, so risk sizing and
the fill-bar exit logic see only accepted fills.

The field is generic execution metadata; no existing strategy sets it, so their
behavior is unchanged. This control limits chasing above the FVG but does not
reject a gap down. A fill at or below the stop retains the existing immediate
stop behavior.

## Stop buffer and target

The entry stop becomes:

```text
stop = lower - StopBufferATR * formationATR
```

`StopBufferATR=0` exactly preserves the current lower-boundary stop. The
formation ATR remains frozen. Provisional and actual-fill targets continue to
use `TakeProfitR * (entry-stop)`, so widening the stop also widens the target
and feeds the wider risk into risk-based sizing.

## Time exit

Add `BarsHeld int` to the shared strategy `Position`. The backtest portfolio
sets it to `currentBar-entryBar`, matching the existing report definition;
the fill bar is `0`. Zero remains safe for live builders that do not populate
it.

When `MaxHoldBars>0`, a position with `BarsHeld >= MaxHoldBars` emits a sell
with reason `TIME` and fills at the current M30 close under existing generic
sell pricing. Exit priority is:

1. stop loss;
2. take profit;
3. time exit.

Thus a bar touching a price exit does not get a more favorable close-based time
exit. `MaxHoldBars=0` preserves unlimited holding.

## Lookback and malformed data

The M30 lookback must retain, through the last eligible retest:

- the anchored `2*ATRPeriod` formation ATR window;
- all three FVG formation bars;
- `StructureLookbackBars` plus the right-side `SwingK` confirmation needed by
  structural pivots; and
- `MaxRetestBars` later bars.

The implementation derives one deterministic formula covering the maximum of
the ATR and structural histories, then adds `MaxRetestBars`. Structural gates
fail closed when aligned OHLC/time data or sufficient history is missing. With
all structural gates disabled, the old minimum-history behavior remains
available and no HTF data is required.

## Calibration

Update the GAZP grid with these exact staged phases so the Cartesian product
remains tractable and each phase has an interpretable purpose:

1. `fvg`, `keepTop=8`: `MinGapATR=[0.25,0.5,0.8,1.0]`,
   `MaxRetestBars=[4,8,12]`, `RequireCloseAboveZone=[0,1]`, and
   `TakeProfitR=[1,1.5,2]` (72 combinations). `ATRPeriod` stays fixed at 14.
2. `context`, `keepTop=8`: `RequireFirstRetest=[0,1]`,
   `RequireBullishBOS=[0,1]`, `RequireLiquiditySweep=[0,1]`, and
   `HTFTrendEMA=[0,20,50,100]` (32 expansions per survivor).
3. `execution`, `keepTop=8`: `MaxEntryDistanceATR=[0,0.25,0.5,1.0]` and
   `StopBufferATR=[0,0.1,0.25,0.5]` (16 expansions per survivor).
4. `holding`, `keepTop=8`: `MaxHoldBars=[0,8,16,32,48]` (five expansions per
   survivor).

`SwingK`, `StructureLookbackBars`, and `MaxSweepToFVGBars` initially stay fixed
at `2`, `48`, and `8`; they are mechanics, not part of the first broad sweep.
Every phase includes the disabled value for each new control it varies. Use
risk-based sizing for comparative reports so different stop widths risk the
same equity fraction. Walk-forward output is evidence for a later human choice;
the implementation does not rewrite `gazp.Params()` from the winner.

The required evidence run uses the repository's established 24-month GAZP
window and this exact primary protocol:

```bash
go run ./cmd/backtest \
  -ticker GAZP -strategy fvg_imbalance -interval Minutes30 \
  -calibrate data/params/fvg_imbalance/gazp/grid.json \
  -out reports/GAZP/fvg_imbalance/r20261104/quality_wf \
  -months 24 -train-months 12 -test-months 3 \
  -min-trades 20 -metric profit_factor \
  -risk-pct 1 -commission 0.0005 -refresh
```

This yields four rolling OOS folds. The requested data cutoff is the execution
date, 2026-11-04 MSK. `-refresh` is required because the existing local H4 cache
ends before the M30 cache; if API refresh cannot reach the cutoff, report the
actual common M30/H4 cutoff and do not interpret the uncovered tail as strategy
evidence. The `min-trades` floor applies to each train ranking; if every
combination in a phase is below it, the report must call out that the phase had
no statistically adequate survivor even though the calibrator still returns a
ranking.

Also run the unchanged baseline over the same refreshed 24-month range with
`-risk-pct 1` and `-commission 0.0005`, writing it to
`reports/GAZP/fvg_imbalance/r20261104/baseline_full`. This is a comparability
check, not an in-sample promotion gate. No additional control-window or
commission-stress run is required by this implementation slice.

## Documentation and diagnostics

Update `docs/fvg_imbalance/strategy.md` with mechanics only: exact event order,
parameter meanings, first-touch consumption, completed-H4 behavior, entry
cancellation, buffered stop, and time-exit priority. Reconcile its GAZP baseline
table with the current literal. Do not add GAZP performance results.

Buy `EntryReason` should state which enabled gates passed and include the FVG,
frozen ATR, stop, and optional maximum-entry price. `TIME` exits receive a
human-readable `ExitReason`. No separate `Explain` interface is required by
this change.

## Verification

Tests must cover at least:

- strict fractal detection, equal-value rejection, and event-time confirmation;
- sweep reclaim boundaries and maximum sweep-to-FVG age;
- BOS close-vs-wick behavior, standalone BOS, standalone sweep, and ordered
  sweep+BOS;
- first touch allowed and every later touch consumed, including a first touch
  that could not fill;
- H4 uptrend pass, downtrend block, missing/short fail-closed, disabled bypass,
  and exclusion of a still-forming H4 candle;
- maximum-entry equality fill, above-limit cancellation, zero bypass, and no
  effects on existing strategies;
- buffered stop and actual-fill R target;
- exact `BarsHeld` values, disabled time exit, time exit at the threshold, and
  SL/TP priority on the threshold bar;
- legacy signals and engine fills with every new control disabled;
- GAZP literal, partial JSON overlay, validation, grid expansion, CLI H4 loading,
  and M30-only enforcement.

Run the focused packages, the repository's applicable race-enabled Go suite,
mock-drift check, build command documented in `AGENTS.md`, and `git diff --check`.
After implementation, run the staged walk-forward calibration from cached data
with non-zero `-risk-pct`; report the evidence without promoting defaults.
