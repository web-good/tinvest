# hammer — hourly hammer in the RSI(5) oversold zone

Date: 2026-10-21. Status: design, approved by the owner before planning.

## Intent

The owner wants a new backtest strategy, `hammer`: on hourly bars, buy the next bar after a
bullish hammer candle that forms while a short RSI is deeply oversold. The idea is a classic
reversal setup — exhaustion (RSI(5) < 30) plus a rejection candle (long lower tail) — exited
either on a modest daily-scale move or when the short RSI swings to overbought.

The first goal is a working backtest on GAZP, SBER and NVTK over 12 months with the owner's rules
as given. The stop size (0.5 daily ATR) is a placeholder; the owner will calibrate it later.

## Goals

- `go run ./cmd/backtest -ticker <T> -strategy hammer -interval Hour1 -months 12` runs and
  reports trades with entry stops.
- Rules exactly as the owner stated them (below), all tunables as Params fields so the existing
  reflection grid calibration can sweep them.
- Per-ticker packages for GAZP, SBER, NVTK (literals = the owner's defaults for now), a registry,
  a starter grid, a strategy doc and a CLAUDE.md line — the same shape as `pdl_zone`.

## Non-goals

- No live runner, no config/env universe, no Telegram.
- No volume, trend, time-of-day or index filters. Nothing off-by-default "just in case".
- No calibration in this task — that comes after the owner sees the first results.
- Other timeframes: Hour1 only; any other interval is rejected by the registry.

## Anti-goals

- Do not touch `pdl_reclaim` or `pdl_zone` code. The working tree has uncommitted owner work in
  `pdl_reclaim`; it must stay exactly as is and must not be staged into hammer commits.

## Rules (decided by the owner)

1. **Timeframe:** Hour1 bars, long only.
2. **Zone:** RSI with period 5 (Wilder, `indicators.RSISeries`) over hourly closes.
3. **Signal bar:** a hammer whose own close has RSI(5) < 30. The RSI and the hammer are on the
   same bar.
4. **Hammer definition:** the same as `isHammer` in
   `internal/service/trading_strategy/pdl_reclaim/strategy/core/setup.go`. The bar can be any
   colour; its range must be > 0; the body is at most a third of the range and sits in the upper
   third, so the lower tail is ≥ 2/3 of the range and ≥ 2 bodies. The owner's own phrasing was
   "tail at least two bodies"; they accepted the stricter pdl_reclaim definition.
5. **Entry:** buy at the open of the next hourly bar (`EntryAtNextOpen`).
6. **Stop:** fill price − `StopDailyATR` (0.5) × daily ATR. The ATR is Wilder `DailyATRPeriod`
   (14) over completed weekday dailies, as in pdl_reclaim/pdl_zone, frozen at the signal. The
   stop never moves.
7. **Target:** fill price + `TargetRangeMult` (0.5) × yesterday's daily range (high − low of
   the last completed weekday daily as of the signal), frozen at the signal.
8. **RSI exit:** when a bar closes with RSI(5) > `RSIExit` (70), sell. Whichever comes first —
   stop, target or RSI exit — closes the position.

## Parameters

| Field | Default | Meaning |
|---|---|---|
| `RSIPeriod` | 5 | RSI period on hourly closes |
| `RSIZone` | 30 | the hammer bar's RSI must be below this |
| `RSIExit` | 70 | exit when a bar's RSI closes above this; 0 — off |
| `StopDailyATR` | 0.5 | stop distance below the fill in daily ATRs |
| `TargetRangeMult` | 0.5 | target distance above the fill as a share of yesterday's range; 0 — off |
| `DailyATRPeriod` | 14 | Wilder period over weekday dailies |

All fields are int or float64.

## Builder's calls (not discussed with the owner; flag them in the hand-off)

- **Name and layout:** strategy `hammer`; core in
  `internal/service/trading_strategy/hammer/strategy/core/`, ticker packages
  `hammer/strategy/{gazp,sber,nvtk}/` with `Params()` returning `core.DefaultParams()` and a
  doc comment noting "owner's rules, not calibrated"; registry
  `internal/service/backtest/hammer_registry.go` mirroring `pdl_zone_registry.go`
  (`HammerLookupOrGeneric`, Hour1 only); `cmd/backtest` gets the `hammer` case, the flag help
  text and `EntryAtNextOpen` for it.
- **isHammer is copied**, not shared: extracting it would edit pdl_reclaim, which carries
  uncommitted owner work. A short comment points to the original.
- **Levels off the fill:** the buy signal carries stop and target off the signal close (for
  `-risk-pct` sizing). In `manage` they are rebuilt off `pos.PurchasePrice`: stop =
  fill − `StopDailyATR` × `pos.EntryATR`; target = fill + the frozen half-range. The half-range is
  recovered from the frozen signal levels: (`pos.TakeProfit` − `pos.StopLoss`) −
  `StopDailyATR` × `pos.EntryATR`, i.e. the target distance at the signal, which does not depend on
  the reference price. If that turns out awkward, carrying the distance another way is fine as long
  as it is frozen at the signal.
- **Exit priority within one bar:** stop (bar low ≤ stop) first, then target (bar high ≥ target),
  then RSI exit on the close. Pessimistic, the same order as the sibling strategies.
- **RSI exit fill:** whatever the engine does for an indicator-close exit (drop_bounce's
  `RSIExit` is the reference); do not invent a new fill rule.
- **Calendar:** entries only on weekdays (MSK), and only when the buy bar (signal + 1h) starts
  on the signal's MSK date, so the strategy never buys on the next day's opening gap. The position
  itself carries overnight and over weekends.
- **Re-entry:** allowed on any later signal after an exit; one position at a time.
- **Warm-up:** no signal until there are enough bars for RSI(5) and enough weekday dailies for the
  ATR (`DailyATRPeriod` + 1); invalid ATR or range ≤ 0 means no signal.
- **Reasons:** `EntryReason`/`ExitReason` in Russian, in the style of pdl_zone (hammer bar time
  in MSK, RSI value, ATR, yesterday's range, stop and target; exit reasons `SL`/`TP`/`RSI`).
- **RSI series:** RSI(5) runs over every hourly bar the engine supplies (morning, main and evening
  sessions, weekend sessions too), the same as the intraday series of pdl_zone/drop_bounce; only
  the dailies drop weekends. Entries are still weekday-only.
- **StopDailyATR ≤ 0** is invalid: no signal.
- **Docs:** `docs/hammer/strategy.md` (mechanics only, no results), a short `hammer` entry in
  CLAUDE.md's strategy list, `data/params/hammer/grid_example.json` sweeping `StopDailyATR`
  (0.3/0.5/0.7/1.0), `TargetRangeMult` (0.3/0.5/0.7) and `RSIExit` (60/70/80).
- **Branch:** a new branch `feat/hammer` off the current HEAD; stage only hammer files,
  `cmd/backtest/main.go` hunks for hammer and the CLAUDE.md line (CLAUDE.md and
  `cmd/backtest/main.go` already carry uncommitted owner edits — stage only the hammer hunks or
  ask).

## Testing

Unit tests in the core: hammer shape (accept/reject cases incl. zero range and body in the
middle), RSI gate on the hammer bar itself, next-open entry and the same-date rule, weekend
rejection, stop/target off the fill, exit priority (SL before TP on the same bar), RSI exit,
warm-up. A registry test like `pdl_zone_registry_test.go`. `./bin/mage ci` green.

## Done when

The three backtests (`-months 12`, Hour1) run twice each — with the default fixed `-fraction`
sizing and with `-risk-pct 1` (the sizing used for sibling calibrations) — and the owner gets a
short table per ticker and sizing: trades, win rate, PF, net result.
