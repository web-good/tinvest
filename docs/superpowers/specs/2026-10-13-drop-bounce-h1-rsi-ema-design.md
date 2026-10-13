# drop_bounce: rewrite as 1h RSI zone above EMA with unspent daily range

Date: 2026-10-13. Status: approved in brainstorming, ready for the implementation plan.

## Goal

Replace the drop_bounce rules completely. The volume-spike version (red bar with volume ≥ a share
of the average day) is dropped: no ticker passed walk-forward on it in a way the owner wants to
trade. The new drop_bounce is a long-only, multi-day, 1h strategy: buy oversold RSI inside an
uptrend while the day still has range left, exit on a daily-ATR stop or an RSI cross up.

Success: the new core runs in `cmd/backtest -strategy drop_bounce -interval Hour1`, every
parameter below is sweepable through a multi-phase grid, and the calibration skill and screener
work on 1h with the new rules.

## What the owner specified vs what was assumed

Owner: timeframe 1h; RSI with length about 6, picked by the grid; close above EMA 200, EMA length
picked by the grid; RSI below 30, or RSI crossing 30 upward while not above a cap of 50, the cap
picked by the grid; enter only if the daily ATR is not used up yet, the threshold picked by the
grid; the stop distance picked by the grid; calibration is multi-phase.

Assumed and accepted: exits stay as in the current core (daily-ATR stop frozen at the signal, RSI
crossing `RSIExit` upward); fill at the next bar's open on the same date; Mon–Fri, `StartHour` 7;
positions carry overnight; old ticker packages are deleted.

## Entry

Evaluated at the close of 1h bar `i`. All conditions must hold.

1. **Window.** The bar is Mon–Fri MSK, starts at or after `StartHour:00`, and the fill bar
   (`t + 1h`) lies on the same MSK date. A signal on the day's last bar is not carried to the morning.
2. **Trend.** `close[i] > EMA(EMAPeriod)[i]` (strictly above).
3. **RSI zone.** With `r = RSI(RSIPeriod)` (Wilder, `indicators.RSISeries`), either
   - **ZONE:** `r[i] < RSIZone`, or
   - **CROSS:** `r[i-1] < RSIZone ≤ r[i]` and `r[i] ≤ RSICap`.
   The entry reason names the branch.
4. **Unspent day.** `used = TodayHigh − TodayLow` (the current MSK day's range up to and including
   bar `i`, as the engine supplies it, no lookahead). Entry needs `dailyATR > 0` and
   `used < DaySpentATR · dailyATR`. Missing or broken day extent (`TodayHigh ≤ 0`, `TodayLow ≤ 0`,
   `TodayHigh < TodayLow`) means no entry.
5. **Stop sanity.** `close[i] − StopDailyATR · dailyATR > 0`.

On a signal: `Kind = SignalBuy`, `ATR = dailyATR` (the engine freezes it on the position),
`FillBy = t + 1h`, fill at the next bar's open (`Config.EntryAtNextOpen`). While a position is open,
signals are ignored. There is no cooldown: after a stop the next bar may enter again if the
conditions hold — that is the rule as specified, the calibration measures it.

`dailyATR` is unchanged from the current core: Wilder `DailyATRPeriod` over completed Mon–Fri
dailies (weekend sessions dropped), 0 when there are fewer than `DailyATRPeriod + 1` days.

## Exit

Unchanged from the current core, in priority order on each bar:

1. **SL** intrabar: `low ≤ entry − StopDailyATR · ATR`, ATR frozen at the signal (today's ATR if the
   position carries none); the engine fills at `min(stop, open)`.
2. **RSI** at the close: `RSI(RSIPeriod)` crosses `RSIExit` upward (`prev < RSIExit ≤ cur`),
   whatever the price.

No end-of-day exit. The RSI of the entry and of the exit is the same series (`RSIPeriod`).

## Parameters

All `int` or `float64` so reflection grids can sweep them.

| field | default | meaning |
|---|---|---|
| `RSIPeriod` | 6 | RSI length for entry and exit |
| `RSIZone` | 30 | oversold level: ZONE below it, CROSS through it upward |
| `RSICap` | 50 | CROSS is valid only if RSI at the cross bar is ≤ this |
| `EMAPeriod` | 200 | trend EMA on 1h closes; 0 = no entries |
| `DaySpentATR` | 0.7 | entry only while today's range < this · daily ATR; 0 = no entries |
| `StopDailyATR` | 1.0 | stop below the fill in daily ATRs; 0 = off |
| `DailyATRPeriod` | 14 | Wilder daily ATR length over weekday dailies |
| `RSIExit` | 70 | exit on an upward cross; 0 = off |
| `StartHour` | 7 | signal bars start at/after this MSK hour |

Removed: `VolDailyMult`, `VolAvgPeriod` and the average-daily-volume code.
`MarketData.DailyVolumes` stays in the engine (other code may read it).

`Lookback = max(64, 3 · EMAPeriod, 4 · RSIPeriod + 2)` — EMA 200 on 1h needs about 600 bars to converge.

## Timeframe

The core is built for 1h only. `DropBounceLookupOrGeneric` rejects every interval except `Hour1`
with an error. The step stays a constructor argument of the core (tests and the screener pass it).

## Calibration grid (multi-phase)

`grids.py calstop <T> Hour1` writes a phase grid (`keepTop 3`, each phase sweeps its axes over the
three best points of the previous one). Every point has a stop; `StartHour` is pinned to 7.

| phase | axes | combos |
|---|---|---|
| 1 entry | `RSIPeriod` 4/5/6/7 × `RSIZone` 25/30/35 × `RSICap` 40/50/60 | 36 |
| 2 filter | `EMAPeriod` 100/150/200 × `DaySpentATR` 0.3/0.5/0.7/1.0 | 12 × 3 |
| 3 risk | `StopDailyATR` 0.5/0.7/1.0/1.5/2.0 × `DailyATRPeriod` 7/10/14 | 15 × 3 |
| 4 exit | `RSIExit` 60/65/70/75/80 | 5 × 3 |

These lists are the owner's; the skill must not change them without a request.

## Surrounding changes

1. **Core** `internal/service/trading_strategy/drop_bounce/strategy/core` rewritten in place;
   `core_test.go`, `exits_test.go`, `engine_test.go` rewritten for the new rules.
2. **Ticker packages** (enpg, gazp, magn, moex, nlmk, nvtk, plzl, rual, tbank, vsmo) and
   `data/params/drop_bounce/<ticker>/` deleted; history stays in git. `dropBounceRegistry` is empty;
   the per-interval literal struct collapses to one `func() core.Params` per ticker (Hour1 only).
   `grid_example.json` rewritten for the new fields.
3. **Config/env.** `DROP_BOUNCE_TICKERS_M15` and `DROP_BOUNCE_TICKERS_M30` removed from
   `config.DropBounceConfig`, `env/*.env*`; `DROP_BOUNCE_TICKERS_H1` stays, empty.
   `TestDropBounceLiveTickersAreRegistered` and the config tests follow.
4. **Screener** `cmd/dropscreen` / `drop_screen.go`: Hour1 only; fixed 16-config grid
   `RSIZone` {25, 35} × `RSICap` {40, 60} × `DaySpentATR` {0.5, 1.0} × `RSIExit` {60, 75}, pinned
   `RSIPeriod` 6, `EMAPeriod` 200, `StopDailyATR` 1.0, `DailyATRPeriod` 14, `StartHour` 7. Gates,
   ranking and report unchanged in structure.
5. **Calibration skill** `.claude/skills/drop-bounce-calibrate`: Hour1 only, the phase grid above,
   ticker package with a single `Params()` literal, references to volume and GREENVOL removed;
   `grids.py`, `report.py`, `overlap.py`, `live_env.py` adjusted to one timeframe.
6. **Docs.** `docs/drop_bounce/strategy.md` and `screener.md` rewritten; the drop_bounce entry in
   `CLAUDE.md` replaced with the new description and an empty ticker list.
7. **Order.** The branch's current uncommitted work (2026-10-13 calibrations, screener changes,
   `live_env.py`) is committed as-is first, so the old version stays in history; the rewrite
   follows in separate commits.

## Testing

Unit tests on the core with synthetic 1h bars:
- ZONE entry fires below `RSIZone` above EMA; not below EMA; not at `close == EMA`.
- CROSS entry fires on the cross with RSI ≤ cap; not above cap; not when the previous RSI was
  already at or above the zone.
- Day gate: fires at `used` just below `DaySpentATR · ATR`, not at or above; not without ATR; not
  with a broken day extent; `DaySpentATR = 0` and `EMAPeriod = 0` give no entries.
- Window: weekend, before `StartHour`, last bar of the date — no entry.
- Signal carries `ATR` and `FillBy = t + 1h`.
- Exits: SL with frozen ATR and gap fill; RSI cross up; SL before RSI on the same bar; no exit
  otherwise.
- Engine test: a short end-to-end run through `backtest.Run` on Hour1 bars.
- Registry: only Hour1 accepted; JSON overrides layer over defaults.
- `./bin/mage ci` green.

## Out of scope

Live runner; cooldown after a stop; other timeframes; recalibrating any ticker (done afterwards
through the skill).
