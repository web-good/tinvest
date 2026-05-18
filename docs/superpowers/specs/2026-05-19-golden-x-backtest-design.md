# Golden X — Stage D0 + D3: Postgres removal & backtest engine

**Status:** Design approved 2026-05-19. Pending implementation plan.
**Branch target:** Will run on a fresh branch after `feature/golden-x-stage-a` lands.
**Master plan:** `~/.claude/plans/internal-service-trading-strategy-golde-woolly-quill.md`.

---

## 1. Scope and non-goals

Two sub-stages bundled because they reinforce each other: D0 strips the dead
Postgres infrastructure that was the original D1 substrate, and D3 introduces
the backtest engine that replaces D1 as the foundation under "measuring the
strategy". The user has confirmed live-signal journaling is **not** wanted —
backtest reconstructs signals from historical candles on demand, no DB needed.

**In scope:**

- **D0**: rip out the unused Postgres stack end-to-end (code, deps, infra,
  config, migrations). Single PR-sized change; no business logic depends on it.
- **D3 v1**: backtest engine for the Golden X strategy, exposed as a standalone
  binary `cmd/backtest`. Markdown report on disk plus console summary.
  Disk-cached candles to avoid burning the Tinkoff API on repeat runs.

**Not in scope (intentionally deferred):**

- Filter ablation (compare strategy with/without trend/divergence/volume
  filters in one run). Considered for D3 v2 if v1 reveals it's wanted.
- Dividend ex-date awareness, broker commissions, slippage — that's E1.
- Live-signal journaling, D4 metrics, Kelly sizing — separate stages.
- Backtesting other strategies (MACD-RSI, SuperTrend, etc.) — Golden X only.
- Parallelization. The data volume (≈20 shares × ≈260 weekly candles × a few
  years) finishes in seconds on a single CPU; no need.

---

## 2. D0 — Postgres removal

The DB is currently wired through `service_provider.GetDbClient` and pinged at
startup in `internal/app/init_database.go`, but **no business logic reads or
writes anything**. The only migration is a `users` table with no consuming code.
Removing it is a pure cleanup.

**Files/directories to delete:**

- `internal/app/init_database.go` (and the call site in `internal/app/app.go`).
- `pkg/client/db/` (entire subtree: `db.go`, `pg/pg.go`, `pg/client.go`,
  `pg/tx_manager.go`).
- `migrations/` (only contains the unused `users` table migration).
- `migration.sh`, `migration.Dockerfile`.
- The `postgres` service (and its volume) in `docker-compose.yml`.

**Files to edit:**

- `internal/service_provider/client.go` — remove `dbClient` field, `GetDbClient`
  method, `db`/`pg` imports.
- `internal/config/config.go` — remove the `Storage` block (DSN, host, port,
  user, pass, name).
- `env/local.env.example` — remove DB-related env keys.
- `go.mod` / `go.sum` — `go mod tidy` drops `github.com/jmoiron/sqlx`,
  `github.com/lib/pq`, and transitives.
- `Makefile` — drop any `migrate` / `migration` targets if present.

**Untouched:** `pkg/client/grpc`, `pkg/client/telegram`, `pkg/closer`,
everything under `internal/service/*`, `internal/domain/*`. They have no DB
dependency.

**Verification:**

- `go build ./...` and `go vet ./...` clean.
- `go run ./cmd/main` boots in dev mode without any `DB_*` env vars set.
- `go.mod` diff shows only deletions of `sqlx`/`lib/pq` and their transitives.

---

## 3. D3 — Architecture

### 3.1 Package layout

New package: `internal/service/trading_strategy/golden_x/backtest/`.

| File | Responsibility |
|------|----------------|
| `detector.go` | Pure signal detection function. Extracted from `trade.go`. |
| `position.go` | Open-position model + partial-exit accounting. |
| `replay.go` | Per-share replay engine: iterate closed candles, evaluate exits then entries. |
| `cache.go` | File-based candle cache with lazy fetch via existing gRPC client. |
| `report.go` | `[]Trade` → metrics aggregation + Markdown formatter. |
| `*_test.go` | Tests for each (see §6). |

New binary: `cmd/backtest/main.go`.

### 3.2 Extracted `Detect` function

The block in `trade.go` lines ~92–161 (fetch already-closed candles, compute
adaptive RSI / thresholds / trend / divergence / volume / ATR stop, pick
buy & sell tier) is pure given the closed-candle slice. Extract it into
`backtest/detector.go`:

```go
func Detect(
    closed []*model.CandleItemTechAnalyse,
    share *dto.Share,
    kind dto.StrategyKind,
    useTrendFilter bool,
) (Signal, error)

type Signal struct {
    BuyTier         tier
    SellTier        tier
    RSI             float64
    LastClose       float64
    Stop            dto.Stop
    SellThresholds  dto.SellThresholds
    TrendOK         bool
    DivergenceOK    bool
    VolumeOK        bool
}
```

After extraction, `service.Trade()` calls `Detect` per share — its body
becomes orchestration around the same pure call (gRPC + dedup + Telegram).
This is the smallest refactor that lets prod and backtest share the detector
without duplicating logic.

`Detect` does **not** call `closedWeeklyCandles` — backtest passes a slice
that's already trimmed to closed weeks. The caller is responsible for handing
in closed candles. `service.Trade()` keeps its existing `closedWeeklyCandles`
call before invoking `Detect`.

### 3.3 Binary: `cmd/backtest/main.go`

Initializes only what's needed:

- Config (for gRPC token and base URL).
- gRPC `MarketDataServiceClient` — but **only** if cache is missing or
  `--refresh` is passed.
- ShareList from `internal/app/init_collection.go`. **Caveat**: the two
  share-list builders (`initCompanyListForGoldenStrategy`, `initGrowthShare`)
  are currently package-private inside `package app` and not reachable from
  `cmd/backtest`. As a small prerequisite refactor under D3 step 1, move them
  (and the `Collection` type) into a new neutral package
  `internal/service/trading_strategy/golden_x/shares` so both `cmd/main` and
  `cmd/backtest` can import them without duplicating the source-of-truth list.

Explicitly **not** initialized: Telegram client, scheduler, signal handlers
that belong to prod loops.

Flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--kind` | `Dividend` | `Dividend` or `Growth`. Selects ShareList + strategy params. |
| `--from` | `2022-01-01` | Inclusive start date (MSK) for signal evaluation. |
| `--to` | today | Inclusive end date. |
| `--shares` | (all) | Comma-separated share IDs to limit the run. |
| `--refresh` | false | Force re-fetch from Tinkoff API, overwriting cache. |
| `--out-dir` | `cache/backtests` | Where the Markdown report lands. |

---

## 4. D3 — Data flow

### 4.1 Candle cache

One JSON file per share: `cache/candles/<shareID>_W.json`. Contents are an
array of:

```json
{"date": "2024-01-08T00:00:00+03:00", "open": 273.41, "high": 281.20,
 "low": 270.05, "close": 278.10, "volume": 18420000}
```

Prices stored as `float64` (already converted via `utils.CombinePrice` —
JSON-friendly, avoids carrying `Quotation{Units, Nano}` into disk format).
`date` is the start of the weekly candle.

`cache.Get(shareID, interval) ([]Candle, error)`:

1. If file exists and `--refresh` not set: read and return.
2. Else: fetch via gRPC in 300-candle batches (same chunking as prod), serialize
   to disk, return.
3. On `--refresh`: same as cache-miss path.

Cache directory is created on demand. Files are git-ignored (add to
`.gitignore` as part of D3).

### 4.2 Replay loop (per share)

```
candles := cache.Get(share.ID)            // all closed weekly candles
startIdx := max(rsiPeriod + adaptiveWindowMin, warmupForEMA200)

for t := startIdx; t < len(candles); t++ {
    closed := candles[:t+1]               // history up to and including week t
    weekT  := candles[t]                  // week t itself
    sig, err := Detect(closed, share, kind, useTrendFilter)
    if err != nil { continue }            // insufficient history etc.

    // 1. Exit open position (order matters)
    if pos != nil {
        // 1a. Stop hit on week t's Low
        if weekT.Low <= pos.StopPrice {
            recordExit(pos, weekT, pos.StopPrice, ExitStop)
            pos = nil
        } else if exits := pos.evaluateSellExits(sig, weekT); len(exits) > 0 {
            // 1b. Sell-tier RSI exit (partial for Gold, full for Growth)
            for _, e := range exits { recordPartial(pos, e) }
            if pos.fullyClosed() { pos = nil }
        } else if pos.WeeksHeld() >= 52 {
            // 1c. Timeout
            recordExit(pos, weekT, weekT.Close, ExitTimeout)
            pos = nil
        }
    }

    // 2. Entry: only on green tier, only if flat
    if pos == nil && sig.BuyTier == tierGreen {
        pos = openPosition(weekT, sig, kind)
    }
}

// Any position still open at the end is recorded with ExitOpen,
// exit price = last candle's Close (mark-to-market).
```

**Why this order:**

- **Stop first** — intra-week, Low can be reached before the close-RSI is known.
  Realistic, conservative.
- **Sell-tier second** — evaluated at week-close, the natural moment after a
  stop wasn't hit.
- **Timeout last** — only if nothing organic exited the trade.

### 4.3 Partial exits for Gold

A Gold position carries:

```
Units float64        // 1.0 at open
SoldP80, SoldP90, SoldP95 bool
```

On each open week:

- First time `RSI ≥ p80` (and `!SoldP80`) → record sell of `1/3` units at
  `weekT.Close`, set `SoldP80 = true`, decrement `Units`.
- Same pattern for p90 and p95.
- When all three flags are set (`Units == 0`), position is fully closed.

Each partial is a separate `Trade` row with its own `ReturnPct` (computed
against the same shared entry price) and `Units = 0.333…`. Win-rate counts each
partial as one trade. Rationale: this matches how the strategy is *intended* to
behave — three independent decisions, three independent outcomes.

### 4.4 Growth exit

Single sell-tier (the existing adaptive p90 in prod for Growth) → full exit at
`weekT.Close`. One `Trade` row with `Units = 1.0`.

---

## 5. Metrics and report

### 5.1 Internal types

```go
type Trade struct {
    ShareID    string
    EntryDate  time.Time
    EntryPrice float64
    ExitDate   time.Time
    ExitPrice  float64
    Units      float64        // 1.0 for Growth, ~0.333 per Gold partial
    ExitReason string         // sell_p80|sell_p90|sell_p95|stop|timeout|open
    ReturnPct  float64        // (Exit-Entry)/Entry * 100
    WeeksHeld  int
}

type Stats struct {
    Count, Wins, Losses, Open  int
    WinRate                    float64       // wins / (wins+losses); excludes Open
    AvgReturnPct, MedianReturn float64
    CumulativeReturn           float64       // Σ(ReturnPct * Units)
    MaxDrawdown                float64       // computed on equity curve
    AvgWeeksHeld               float64
    ExitReasons                map[string]int
}

type Report struct {
    Kind     dto.StrategyKind
    From, To time.Time
    Trades   []Trade
    PerShare map[string]Stats
    Overall  Stats
}
```

**Win/Loss:** `ReturnPct > 0` → win, `< 0` → loss, `== 0` → neither. Open
positions counted separately, never as win or loss.

**Max drawdown:** build equity curve as cumulative `ReturnPct * Units` ordered
by `EntryDate`. Standard `max((peak − value) / peak)` over the curve.

### 5.2 Output

**File:** `<out-dir>/<YYYY-MM-DD_HHMM>_<kind>.md`. Sections in order:

1. Header (kind, date range, generated-at, filter flags).
2. `## Overall` — single-row metrics table.
3. `## Exit reasons` — counts and avg return per reason.
4. `## Per share` — per-share table.
5. `## Trades (chronological)` — one row per `Trade`.

**Console:** prints the `## Overall` block and `## Exit reasons` block, then
`Report written to: <path>`. Nothing else — no progress logs above warnings.

---

## 6. Testing

### 6.1 New unit tests

- `detector_test.go` — table-driven golden tests on synthetic candles.
  Scenarios: green buy, yellow buy, sell p80, insufficient history, trend
  filter off vs on. Use existing helpers from sibling `*_test.go` files.
- `position_test.go` — open, stop exit, Gold partial exits (three triggers in
  three different weeks), Growth full exit, timeout exit.
- `replay_test.go` — synthetic 60-week history → expected `Trade` sequence.
  Specifically covers the 1a→1b→1c exit ordering.
- `report_test.go` — fixed `[]Trade` → expected `Stats`. Drawdown formula is
  the highest-risk math; cover it with at least one non-monotonic equity curve.
- `cache_test.go` — read/write JSON round-trip, fallback to fetch on cache
  miss, full overwrite on `--refresh`.

### 6.2 Existing-test regression check

After extracting `Detect` from `trade.go`, the existing tests must pass
unchanged: `dedup_test`, `percentile_test`, `divergence_test`, `stop_test`,
`trend_filter_test`, `rsi_test`, `candle_test`. If any of them shift, the
extraction broke a contract — revert and reconsider boundaries.

### 6.3 End-to-end checks

- `go build ./...` — both `cmd/main` and `cmd/backtest` compile.
- `go run ./cmd/backtest --kind=Dividend --from=2022-01-01 --shares=SBER` —
  first run fetches and caches; second run finishes without any gRPC calls
  (verify by network log or by setting an invalid token after the first run).
- Sanity bounds on output: trade count in the dozens (not zero, not thousands);
  `MaxDrawdown ∈ [0, 100]`; `CumulativeReturn ≈ Σ(ReturnPct * Units)`.

### 6.4 D0 verification

- `go build ./... && go vet ./...` clean.
- `go mod tidy` produces a diff that only removes `sqlx`, `lib/pq`, and their
  transitives.
- `go run ./cmd/main` boots in dev mode without `DB_*` env vars.

---

## 7. Implementation order

Two sub-stages, in this order:

1. **D0** — Postgres removal. Self-contained, no dependencies on D3 work.
   Lands as one commit (or two: code + go.mod tidy).
2. **D3** — backtest engine. Order within D3:
   1. Move share-list builders out of `internal/app/init_collection.go` into a
      new neutral package (see §3.3). Update `internal/app` callers. No
      behavior change; existing tests still pass.
   2. Extract `Detect` into `backtest/detector.go` + regression-test existing
      tests still pass.
   3. Cache layer (`cache.go` + tests).
   4. Position model (`position.go` + tests).
   5. Replay engine (`replay.go` + tests).
   6. Report aggregation and Markdown (`report.go` + tests).
   7. `cmd/backtest/main.go` — CLI wiring.
   8. E2E sanity run on `SBER` Dividend.

Each step independently buildable and testable. Subagent-driven-development
fits naturally — one subagent per step in a fresh context.
