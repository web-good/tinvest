# drop_bounce — red bar on a daily-volume spike (design)

Date: 2026-10-12. Status: approved in conversation, pending spec review.

## Goal

Replace the drop_bounce rules wholesale for the second time. The 2026-10-09 core (drop percent +
RSI(6) < 30 signal, first-green wait, day-range and volume gates, PROFIT / RSI / STOCH exits with
the deferred "exit in profit") is removed, and so are the five ticker packages calibrated on it
(GAZP, DOMRF, T, IVAT, WUSH — none passed walk-forward). The new strategy buys after a red bar
whose volume alone is a sizeable share of an average trading day, and holds — overnight if
needed — until RSI(6) crosses 70 upward or a daily-ATR stop fires.

Backtest-only. Target timeframe 15m; the core stays step-agnostic, so `-interval Minutes5 |
Minutes15 | Minutes30 | Hour1` stay accepted.

## Rules

### Entry

Evaluated at the close of the last bar `i` (the signal bar); the backtest fills at the open of
bar `i+1` (`Config.EntryAtNextOpen`, `Signal.FillBy = Times[i] + step`, as today).

1. Bar `i` is red: `Close[i] < Open[i]` (`Open[i] > 0`).
2. Volume spike: `Volume[i] ≥ VolDailyMult × VolAvg − ε`, where `VolAvg` is the average daily
   volume defined below.
3. Window (unchanged): bar `i` starts on a Mon–Fri MSK date, at or after `StartHour:00` MSK, and
   bar `i+1` lies on the same MSK date.
4. Data guards — no entry when any is missing:
   - `VolAvg` is unknown (no or misaligned `DailyVolumes`, fewer than `VolAvgPeriod` usable days).
     Volume is the signal itself, so missing volume blocks the entry (unlike the old volume gate).
   - `StopDailyATR > 0` and the daily ATR is unknown.
   - `StopDailyATR > 0` and `Close[i] − StopDailyATR × ATR ≤ 0`.

The signal carries the daily ATR (`Signal.ATR`) for the engine to freeze on the position, and an
`EntryReason` like `VOLUME: open … → close …, объём N = X.XX× среднего дня (VolAvg …, порог
0.40), дневной ATR …`.

### Average daily volume (`VolAvg`)

Input: `MarketData.DailyVolumes` aligned with `DailyTimes` (completed days only, no lookahead).

1. Drop Sat/Sun MSK days (same weekday filter as the daily ATR).
2. Drop zero-volume days (no trading; they would understate the average and fire the threshold on
   nothing).
3. Wilder smoothing over the remaining series: seed = SMA of the first `VolAvgPeriod` values, then
   `avg += (v − avg) / VolAvgPeriod` for each later value. Fewer than `VolAvgPeriod` values → unknown.

This is what the owner calls "ATR of volume": volume has no high/low, so the true-range part does
not apply and the average volume itself is the yardstick.

### Exits, in priority order

1. **SL** (intrabar): `Low ≤ entry − StopDailyATR × ATR`, ATR frozen at the signal (today's ATR
   when the position carries none); the engine fills at `min(stop, open)`. Off when
   `StopDailyATR = 0`.
2. **RSI** (bar close): RSI(`RSIPeriod`) crosses `RSIExit` upward (`prev < level ≤ cur`). Sells
   whatever the price — the deferred "exit in profit" is gone. Off when `RSIExit = 0`.

No end-of-day exit: positions carry overnight.

### Daily ATR

Unchanged: Wilder ATR(`DailyATRPeriod`) over completed weekday dailies.

## Params

All fields int or float64 (reflection grid calibration).

| field | default | meaning |
|---|---|---|
| `VolDailyMult` | 0.4 | signal bar volume ≥ this × `VolAvg` |
| `VolAvgPeriod` | 14 | Wilder length of the weekday daily-volume average |
| `RSIPeriod` | 6 | RSI length of the exit |
| `RSIExit` | 70 | sell on RSI crossing this upward; 0 = off |
| `StopDailyATR` | 1.0 | stop distance below the fill in daily ATRs; 0 = off |
| `DailyATRPeriod` | 14 | Wilder ATR length over weekday dailies |
| `StartHour` | 8 | signal bars start at/after this hour MSK |

Removed: `MinDropPct`, `RSIEntry`, `GreenWaitBars`, `StochPeriod`, `StochSmooth`, `StochExit`,
`ProfitPct`, `ExitInProfit`, `MaxDayRangeATR`, `VolMult`, `VolBars`.

`Lookback = max(64, 4×RSIPeriod + 2)` — no deferred-exit history any more.

## Data plumbing

`strategy.MarketData` gains `DailyVolumes []int64`: oldest-first volumes of the same completed
daily candles as `DailyCloses` (index-aligned), empty when no daily data. Consumers must degrade
gracefully when it is empty or misaligned.

`internal/domain/backtest/engine.go`:
- `visibleDaily` and `dailyCursor.visible` also return volumes; the cursor keeps a volumes backing
  array and returns a capped prefix (full slice expression), like the other series.
- All three snapshot builders (Run, Trace, `AssembleMarketDataWithHTFInterval`) fill
  `md.DailyVolumes`.
- Additive for every other strategy; live reversion / rsi_pullback get the field and ignore it.

Rejected alternative: summing intraday bars inside the strategy — the lookback would grow to ~900
bars on 15m (several thousand on 5m) and RSI recomputes over the whole window each bar.

## Ticker packages and registry

- First commit: the current uncommitted ticker packages (`drop_bounce/strategy/{domrf,gazp,ivat,
  tbank,wush}`) and their grids/results `data/params/drop_bounce/<ticker>/` as an archive, so the
  old calibration stays in git history.
- Then delete them. `internal/service/backtest/drop_bounce_registry.go` drops the per-ticker
  registry: every ticker on every supported timeframe runs on `core.DefaultParams`; partial JSON
  overrides still layer over it. Unsupported intervals are still rejected.
- New ticker packages come back only after a calibration on the new rules (the
  "ticker package always" rule applies then).

## Docs and tooling

- `docs/drop_bounce/strategy.md` rewritten for the new rules.
- `CLAUDE.md` drop_bounce line rewritten (rules, params, no ticker packages).
- `data/params/drop_bounce/grid_example.json` — new starter grid: `VolDailyMult` 0.05–0.6,
  `StopDailyATR` 0.5–2, `RSIExit` 60–80.
- `.claude/skills/drop-bounce-calibrate/` — adjusted to the new field names and grid.

## Testing (TDD)

Core:
- entry fires on red bar + volume at the threshold; not below it; not on a green or doji bar;
- no entry without `DailyVolumes`, with misaligned `DailyVolumes`, or with fewer than
  `VolAvgPeriod` usable days;
- weekend and zero-volume days are excluded from `VolAvg` (a known series gives a hand-computed
  Wilder value);
- window: before `StartHour`, on a weekend, last bar of the day (fill tomorrow) → no entry;
- no entry when the stop needs an ATR that is unknown, or lands at/below zero;
- exits: SL at the stop (intrabar), RSI cross 70 sells even below entry, SL beats RSI on the same
  bar, nothing fires otherwise.

Engine:
- `DailyVolumes` is index-aligned with `DailyCloses`, excludes the current day, and the cursor
  matches `visibleDaily`.

Registry:
- any ticker on Minutes5/15/30/Hour1 gets `core.DefaultParams`; other intervals rejected;
  partial JSON overrides apply.
