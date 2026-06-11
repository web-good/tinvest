# Backtest report: verbose exit reason

## Problem

The backtest trade journal records a terse exit **code** (`"SL"`/`"TP"`/`"TRAIL"`/
`"MACD"`/`"RSI"`) in the "Причина" column, while entries get a full human-readable
rationale in "Причина входа" (`Trade.EntryReason`, built by the strategy at entry).
There is no descriptive explanation of *why* a position was closed — the reader
cannot see, e.g., which level the stop sat at or what RSI values triggered the
exit without cross-referencing other columns.

## Goal

Give exits a verbose, human-readable reason symmetric with the entry rationale.
Keep the short code column for quick scanning and **add** a new descriptive
"Причина выхода" column at the end of the journal, populated by the strategy at
exit time.

## Non-goals

- No change to exit *logic* or trade outcomes — display/threading only.
- No enrichment of the levels/scalping strategies' exits (they keep an empty
  exit-reason cell; can follow the same pattern later if wanted).
- No new exit triggers.

## Data flow

The verbose text is produced where the exit decision is made (the strategy core),
then threaded through the same path the entry rationale already uses:

`strategy core (manage) → model.Signal.ExitReason → engine → portfolio.close → Trade.ExitReason → report`

### `internal/service/trading_strategy/scalping/model/signal.go`

Add a field to `Signal` (shared by momentum/levels/scalping):

```go
ExitReason string // human-readable exit rationale (set on Sell); empty for buys
```
Keep the existing `Reason` (code) field unchanged.

### `internal/service/trading_strategy/momentum/strategy/core/core.go` (`manage`)

In each of the five exit `case`s, set `sig.ExitReason` alongside the existing
`sig.Reason` code, using values already available in `manage`/`decideInput`
(`in.barLow`, `in.barHigh`, `hardSL`, `tp`, `chandelier`, `in.recentHigh`,
`in.atr`, `in.macdNow`, `in.rsiPrev`, `in.rsiNow`) and `s.p` params. Formats:

| Code | ExitReason text (Russian) |
|---|---|
| SL | `SL: low %.4f ≤ стоп %.4f (зафиксирован на входе)` — `in.barLow`, `hardSL` |
| TRAIL | `TRAIL: low %.4f ≤ шанделье %.4f (recentHigh %.4f − %.2g×ATR %.4f)` — `in.barLow`, `chandelier`, `in.recentHigh`, `s.p.TrailMult`, `in.atr` |
| TP | `TP: high %.4f ≥ цель %.4f (%.2gR)` — `in.barHigh`, `tp`, `s.p.TakeProfitRR` |
| MACD | `MACD: медвежий кросс сигнальной линии (MACD=%.4f)` — `in.macdNow` |
| RSI | `RSI: %.2f → %.2f, пересёк границу %.2g сверху вниз` — `in.rsiPrev`, `in.rsiNow`, `s.p.RSIOverbought` |

The text is built inline in each case (mirrors how `entryReason` is its own
concern; here each case has distinct variables, so inline `fmt.Sprintf` is
clearer than one helper with a giant switch).

### `internal/domain/backtest/portfolio.go` (`close`)

Change the signature:
```go
func (p *portfolio) close(price float64, t time.Time, reason, exitReason string) Trade
```
Set `ExitReason: exitReason` on the returned `Trade` (next to `Reason: reason`).

### `internal/domain/backtest/engine.go`

Both call sites pass the new arg (both have `sig` in scope):
- `Run` (line ~125): `p.close(exitPrice, c.Time, sig.Reason, sig.ExitReason)`
- `Trace` (line ~207): `p.close(exitPrice, c.Time, sig.Reason, sig.ExitReason)`

These are the only two callers of `close`.

### `internal/domain/backtest/types.go` (`Trade`)

Add after `EntryReason`:
```go
ExitReason string // human-readable exit rationale captured at exit; empty when n/a
```

### `internal/domain/backtest/report.go`

- **Markdown** (`RenderMarkdown`): append a "Причина выхода" column as the **last**
  column of the trade-journal table (header and each row), value `t.ExitReason`.
- **CSV** (`RenderTradesCSV`): append an `exit_reason` column (last), wrapped with
  the existing `csvField` helper since the text contains commas/spaces.

## Backward compatibility

`ExitReason` is the zero value (`""`) for any strategy that does not set it
(levels, scalping). Their reports render an empty "Причина выхода" cell — no
breakage. The momentum strategy is the only one populating it in this change.

## Testing

- `core_test.go`: extend/assert that each exit sets a non-empty `sig.ExitReason`
  containing its key substring — SL → `"стоп"`, TP → `"цель"`, TRAIL → `"шанделье"`,
  MACD → `"кросс"`, RSI → `"пересёк границу"`. Reuse the existing exit tests'
  setups (`inPositionMD`, `inPositionMDWithCloses`); add the assertion or new
  focused tests.
- `report_test.go`: assert `RenderMarkdown` output contains the "Причина выхода"
  header and a trade's `ExitReason` text; assert `RenderTradesCSV` header includes
  `exit_reason` and the value is present (quoted via `csvField`).
- A portfolio-level check that `close` threads `exitReason` into `Trade.ExitReason`
  (can be covered by the report test using a `Trade` with `ExitReason` set, or a
  small `portfolio_test.go` case).

## Files touched

- `internal/service/trading_strategy/scalping/model/signal.go`
- `internal/service/trading_strategy/momentum/strategy/core/core.go` + `core_test.go`
- `internal/domain/backtest/portfolio.go`
- `internal/domain/backtest/engine.go`
- `internal/domain/backtest/types.go`
- `internal/domain/backtest/report.go` + `report_test.go`

## Out of scope / follow-ups

- Verbose exit reasons for levels/scalping strategies.
- Any change to the entry-reason rendering.
