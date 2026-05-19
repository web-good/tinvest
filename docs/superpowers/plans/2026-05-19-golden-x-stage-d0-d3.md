# Golden X — Stage D0 + D3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-05-19-golden-x-backtest-design.md`
**Target branch:** continue on `feature/golden-x-stage-a` (user confirmed 2026-05-19)
**Plan layout:** one file, two phases. Phase 1 = D0 (Postgres removal). Phase 2 = D3 (backtest engine).

---

## Context

The Tinkoff trading bot ships unused Postgres infrastructure: `pkg/client/db/*`, `internal/app/init_database.go`, the `Storage` config block, a `users` migration, docker-compose `db` + `migrator` services, and `sqlx`+`lib/pq` dependencies. No business code reads or writes anything via DB. Removing this is pure cleanup (Phase 1 / D0).

After D0 lands, Phase 2 / D3 adds a backtest engine for the Golden X strategy. The engine reuses the live signal logic by extracting it into a pure `Detect` function callable from both prod (`service.Trade`) and a new standalone binary `cmd/backtest`. Backtest runs entirely offline using a file-based candle cache (`cache/candles/<shareID>_W.json`) lazily refreshed via the existing gRPC client. Output is a Markdown report (`cache/backtests/<ts>_<kind>.md`) plus an Overall+Exit-reasons console summary. Per-share replay enforces stop-first → sell-tier → 52-week timeout exit ordering, with Gold producing three partial exits (p80/p90/p95) and Growth one full exit (p90). Metrics include win-rate, cumulative return, max drawdown (built from a date-sorted equity curve), avg-weeks-held, and per-exit-reason breakdown.

**Why D0 first:** §7 of the spec orders D0 before D3 because the new `cmd/backtest` binary should not inherit dead DB initialization paths. D0 is self-contained.

**Goal of this plan:** decompose both phases into bite-sized, TDD-friendly tasks executable in fresh subagent contexts, each ending in a `go build ./... && go test ./...` checkpoint and a commit.

---

## Architecture decisions (deviations from spec / clarifications)

1. **`Detect` lives in `package golden_x`, file `detector.go`** — not in the `backtest/` subpackage. Reason: `Detect` needs unexported helpers (`closedWeeklyCandles`, `adaptiveRSIForShare`, `tierFromAdaptive`, `bullishDivergence`, `lowsAlignedToRSI`, `computeRSISeries`, ATR-via-`pkg/indicators`, `kForKind`, `stopFromATR`, `adaptiveSellThresholds`, `trendStatusFromCandles`). Moving it under `backtest/` would force renaming a dozen private symbols. Backtest subpackage imports `golden_x.Detect`. Spec §3.2 specifies contents, not location — this is a small, justified adjustment.

2. **`Signal` carries booleans, not tier constants.** Spec example had `BuyTier`/`SellTier tier` fields; we use `GreenBuy bool` plus `RSI`, `SellThresholds`, `Stop`, etc. so backtest performs its own partial-exit threshold comparisons. This avoids exporting the private `alertTier` enum. Detect still computes both tiers internally so `service.Trade()` can keep its current dedup/alert path unchanged.

3. **Share list move** (spec §3.3, user confirmed): create new package `internal/service/trading_strategy/golden_x/shares` exporting `Dividend()` and `Growth()` returning `*collection.InstrumentCollection`. `internal/app/init_collection.go` becomes a thin shim that calls them. `cmd/backtest/main.go` imports the same package.

4. **Trend filter in Detect:** add a closed-candles-only variant `trendStatusFromClosed(closed, period) (dto.TrendStatus, error)` that skips re-filtering. `trendStatusFromCandles` (the prod-facing version) keeps its current signature so `service.Trade()` doesn't need to change at that call site.

5. **`Detect` signature:**
   ```go
   func Detect(
       closed []*model.CandleItemTechAnalyse,
       rsiPeriod int,
       kind dto.StrategyKind,
       useTrendFilter bool,
   ) (dto.Signal, error)
   ```
   Pure: no time/loc args, no I/O. Caller passes already-closed candles. `rsiPeriod` is the per-share RSILength (from `collection.Instrument`).

6. **`Detect` errors:** returns `ErrAdaptiveInsufficientHistory` or `ErrInsufficientHistory` on insufficient candles. Backtest's replay loop ignores both (`continue`).

7. **D3 binary uses `flag` stdlib** (no extra dep). Config loaded by calling a slimmed `loadConfig()` that only requires `T_BANK` env var; no Telegram, no DB. New helper in `internal/config/` or local to `cmd/backtest/main.go` — see Task 2.7.

---

## Phase 0: Save plan to repo

Plan-mode wrote this file under `~/.claude/plans/`. The user's workflow keeps committed plans under `docs/superpowers/plans/`. Implementation begins by copying this plan there.

### Task 0.1: Save plan file under repo

**Files:**
- Create: `docs/superpowers/plans/2026-05-19-golden-x-stage-d0-d3.md`

- [ ] **Step 1:** Copy `~/.claude/plans/effervescent-dancing-hejlsberg.md` → `docs/superpowers/plans/2026-05-19-golden-x-stage-d0-d3.md`.

  Run: `cp ~/.claude/plans/effervescent-dancing-hejlsberg.md docs/superpowers/plans/2026-05-19-golden-x-stage-d0-d3.md`

- [ ] **Step 2: Commit**

  ```bash
  git add docs/superpowers/plans/2026-05-19-golden-x-stage-d0-d3.md
  git commit -m "docs(golden_x): D0+D3 implementation plan"
  ```

---

# Phase 1: D0 — Postgres removal

Net effect: `go build ./...`, `go vet ./...`, `go test ./...` all clean; `go.mod` loses `github.com/jmoiron/sqlx` and `github.com/lib/pq` (plus transitives); `go run ./cmd` boots in dev mode without any `DB_*` env vars.

## Task 1.1: Delete DB source code

**Files (delete):**
- `internal/app/init_database.go`
- `pkg/client/db/db.go`
- `pkg/client/db/pg/pg.go`
- `pkg/client/db/pg/client.go`
- `pkg/client/db/pg/tx_manager.go`
- `internal/config/storage.go`
- `migrations/20250116195134_tabde_user.sql` (and any siblings)
- `migration.sh`
- `migration.Dockerfile`

- [ ] **Step 1: Delete the files**

  Run:
  ```bash
  rm internal/app/init_database.go
  rm -r pkg/client/db
  rm internal/config/storage.go
  rm -r migrations
  rm migration.sh migration.Dockerfile
  ```

- [ ] **Step 2: Verify build fails with expected errors**

  Run: `go build ./...`
  Expected: errors about undefined `Storage`, `GetDbClient`, missing `pkg/client/db` import. Confirms removals are reachable; next tasks fix the callers.

---

## Task 1.2: Strip DB wiring from `service_provider/client.go`

**Files:**
- Modify: `internal/service_provider/client.go`

- [ ] **Step 1: Edit file**

  Remove imports `tinvest/pkg/client/db` and `tinvest/pkg/client/db/pg`. Remove `dbClient db.Client` field from `type client struct`. Remove the entire `GetDbClient` method (the function from `func (s *ServiceProvider) GetDbClient(...) ... {` through its closing brace). Also remove the now-orphan `"fmt"` import if `fmt` is no longer used in the file — check after deletion.

  After edits the file should contain only: the `client` struct with `grpcClient` + `telegramBot` fields, `GetGrpcClient`, and `GetTelegramBotClient`.

- [ ] **Step 2: Verify**

  Run: `go build ./internal/service_provider/...`
  Expected: PASS.

---

## Task 1.3: Update config, app entry, env files

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/app/app.go`
- Modify: `env/local.env`

- [ ] **Step 1: Edit `internal/config/config.go`**

  Delete the `Storage Storage` line. Final contents:
  ```go
  package config

  var ConfigPath string

  type Config struct {
      AppEnv         string `config:"APP_ENV,backend=env"`
      AppName        string `config:"APP_NAME,required,backend=env"`
      GrpcClient     *GrpcClient
      TelegramClient *TelegramClient
  }
  ```

- [ ] **Step 2: Edit `internal/app/app.go`**

  Remove the commented line `//	a.initDatabase,` from `initializationLoop` (currently line ~63). Leave the surrounding init list intact.

- [ ] **Step 3: Edit `env/local.env`**

  Remove these lines:
  ```
  DB_HOST=localhost
  DB_PORT=5432
  DB_NAME=tinvest
  DB_PASS=tinvest_password
  DB_USER=tinvest
  MIGRATION_DIR=./migrations
  ```
  Keep `APP_PORT`, `SWAGGER_PORT`, `APP_ENV`, `APP_NAME`, `PROFILES`.

- [ ] **Step 4: Verify**

  Run: `go build ./... && go vet ./...`
  Expected: PASS clean (no DB-related errors).

---

## Task 1.4: Strip `db` + `migrator` from docker-compose

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1: Edit `docker-compose.yml`**

  Remove the `db:` service, the `migrator:` service, and the `postgres_data:` named volume. After edit, the file should contain only `version: '3.8'` and an empty `services:` block (or be deleted entirely if nothing remains). If nothing remains and the file has no other services, prefer leaving the file with a top-level `version: '3.8'` + `services: {}` empty block so future services can slot in; or delete it. Choose deletion since the file currently contains only postgres+migrator.

  Run: `rm docker-compose.yml`

- [ ] **Step 2: Verify**

  Run: `git status`
  Expected: deletion of `docker-compose.yml` listed.

---

## Task 1.5: `go mod tidy` and full test pass

**Files:**
- Modify: `go.mod`, `go.sum` (via `go mod tidy`)

- [ ] **Step 1: Run tidy**

  Run: `go mod tidy`

- [ ] **Step 2: Inspect diff**

  Run: `git diff go.mod`
  Expected: `github.com/jmoiron/sqlx` and `github.com/lib/pq` are removed from the `require` block (and any transitive removals visible in `go.sum`).

- [ ] **Step 3: Full clean build + vet + tests**

  Run: `go build ./... && go vet ./... && go test ./...`
  Expected: all pass.

---

## Task 1.6: Smoke run

- [ ] **Step 1: Run app in dev mode**

  Run: `go run ./cmd` (the entry is `cmd/main.go` — `go run ./cmd` builds and runs it).
  Expected: app starts, logs "init t-invest application" and either runs the Golden X dev goroutine or exits cleanly on SIGINT. **No** "could not connect to database" errors. **No** panics from missing `DB_*` vars.

  Stop with Ctrl-C after startup logs confirm OK.

---

## Task 1.7: Commit Phase 1

- [ ] **Step 1: Stage and commit**

  ```bash
  git add -A
  git commit -m "$(cat <<'EOF'
  chore(golden_x): D0 — remove unused Postgres infrastructure

  Strips the dead DB stack end-to-end: pkg/client/db, init_database.go,
  Storage config, migrations + migration.sh/Dockerfile, docker-compose
  db+migrator services, sqlx + lib/pq deps. No business logic touched the
  DB; this is pure cleanup before the D3 backtest engine lands.
  EOF
  )"
  ```

- [ ] **Step 2: Confirm**

  Run: `git log --oneline -1`
  Expected: the new commit is on top of branch `feature/golden-x-stage-a`.

---

# Phase 2: D3 — Backtest engine

Order within Phase 2 follows §7 of the spec. Each task ends with a build+test checkpoint and a commit.

## Task 2.1: Move share lists into neutral package `golden_x/shares`

**Files:**
- Create: `internal/service/trading_strategy/golden_x/shares/shares.go`
- Modify: `internal/app/init_collection.go`

- [ ] **Step 1: Create the new package**

  Write `internal/service/trading_strategy/golden_x/shares/shares.go`:
  ```go
  // Package shares is the source-of-truth share list for the Golden X
  // strategy. Importable from prod (cmd) and backtest (cmd/backtest)
  // without dragging in package app.
  package shares

  import "tinvest/pkg/collection"

  // Dividend returns the curated Gold (long-hold dividend) share list.
  // Per-share RSILength tracks each ticker's empirically chosen smoothing.
  func Dividend() *collection.InstrumentCollection {
      c := collection.NewCollection()
      c.
          Add(collection.Instrument{ID: "a797f14a-8513-4b84-b15e-a3b98dc4cc00", RSILength: 10, Name: "Сургутнефтегаз - прив"}).
          Add(collection.Instrument{ID: "efdb54d3-2f92-44da-b7a3-8849e96039f6", RSILength: 9,  Name: "Татнефть - прив"}).
          Add(collection.Instrument{ID: "fd417230-19cf-4e7b-9623-f7c9ca18ec6b", RSILength: 9,  Name: "Роснефть"}).
          Add(collection.Instrument{ID: "02cfdf61-6298-4c0f-a9ca-9cabc82afaf3", RSILength: 9,  Name: "Лукойл"}).
          Add(collection.Instrument{ID: "c190ff1f-1447-4227-b543-316332699ca5", RSILength: 8,  Name: "Сбер Банк - прив"}).
          Add(collection.Instrument{ID: "fa6aae10-b8d5-48c8-bbfd-d320d925d096", RSILength: 11, Name: "Северсталь"}).
          Add(collection.Instrument{ID: "161eb0d0-aaac-4451-b374-f5d0eeb1b508", RSILength: 8,  Name: "НЛМК"}).
          Add(collection.Instrument{ID: "7132b1c9-ee26-4464-b5b5-1046264b61d9", RSILength: 9,  Name: "ММК"}).
          Add(collection.Instrument{ID: "9978b56f-782a-4a80-a4b1-a48cbecfd194", RSILength: 7,  Name: "ФосАгро"}).
          Add(collection.Instrument{ID: "653d47e9-dbd4-407a-a1c3-47f897df4694", RSILength: 9,  Name: "Транс нефть"}).
          Add(collection.Instrument{ID: "1e19953d-01c6-4ecd-a5f4-53ae3ed44029", RSILength: 8,  Name: "Банк Санкт-Петербург"})
      return c
  }

  // Growth returns the curated growth share list (single-sell-tier strategy).
  func Growth() *collection.InstrumentCollection {
      c := collection.NewCollection()
      c.
          Add(collection.Instrument{ID: "0d53d29a-3794-41c6-ba72-556d46bacb46", RSILength: 7, Name: "Мать и дитя"}).
          Add(collection.Instrument{ID: "962e2a95-02a9-4171-abd7-aa198dbe643a", RSILength: 8, Name: "Газпром"}).
          Add(collection.Instrument{ID: "7de75794-a27f-4d81-a39b-492345813822", RSILength: 7, Name: "Яндекс"}).
          Add(collection.Instrument{ID: "10620843-28ce-44e8-80c2-f26ceb1bd3e1", RSILength: 7, Name: "Полюс"}).
          Add(collection.Instrument{ID: "87db07bc-0e02-4e29-90bb-05e8ef791d7b", RSILength: 8, Name: "Т-Технологии"}).
          Add(collection.Instrument{ID: "0da66728-6c30-44c4-9264-df8fac2467ee", RSILength: 9, Name: "НОВАТЭК"})
      return c
  }
  ```

- [ ] **Step 2: Slim `internal/app/init_collection.go`**

  Replace its body with a thin shim that delegates to the new package:
  ```go
  package app

  import (
      "context"
      "tinvest/internal/service/trading_strategy/golden_x/shares"
      "tinvest/pkg/collection"
      "tinvest/pkg/logger"
  )

  type Collection struct {
      GoldInstruments *collection.InstrumentCollection
      GrowthShare     *collection.InstrumentCollection
  }

  func (a *App) initCollection(ctx context.Context) error {
      logger.InfoContext(ctx, "Start init list")
      a.collection = &Collection{
          GoldInstruments: shares.Dividend(),
          GrowthShare:     shares.Growth(),
      }
      logger.InfoContext(ctx, "End init list")
      return nil
  }
  ```

- [ ] **Step 3: Verify**

  Run: `go build ./... && go vet ./... && go test ./...`
  Expected: all pass; existing tests unchanged.

- [ ] **Step 4: Commit**

  ```bash
  git add internal/service/trading_strategy/golden_x/shares/shares.go internal/app/init_collection.go
  git commit -m "refactor(golden_x): move share lists into shares subpackage"
  ```

---

## Task 2.2: Extract pure `Detect` + `Signal`

**Files:**
- Create: `internal/service/trading_strategy/golden_x/dto/signal.go`
- Create: `internal/service/trading_strategy/golden_x/detector.go`
- Modify: `internal/service/trading_strategy/golden_x/trade.go` (rewire to call `Detect`)
- Modify: `internal/service/trading_strategy/golden_x/trend_filter.go` (add `trendStatusFromClosed`)
- Create: `internal/service/trading_strategy/golden_x/detector_test.go`

- [ ] **Step 1: Define `dto.Signal`**

  Write `internal/service/trading_strategy/golden_x/dto/signal.go`:
  ```go
  package dto

  // Signal is the pure output of golden_x.Detect for a single share against a
  // closed-week candle history. Backtest replay derives entry/exit decisions
  // from this directly; service.Trade composes it with dedup + Telegram.
  type Signal struct {
      RSI            float64
      LastClose      float64
      Stop           Stop
      Thresholds     Thresholds
      SellThresholds SellThresholds
      TrendStatus    TrendStatus // TrendUnknown if useTrendFilter was false
      GreenBuy       bool        // RSI < Thresholds.P5
      YellowBuy      bool        // RSI < Thresholds.P15 && !GreenBuy
      DivergenceOK   bool        // bullish RSI divergence on the buy side
      VolumeOK       bool        // last-week volume > multiplier × SMA20
  }
  ```

- [ ] **Step 2: Add closed-candles trend helper**

  Edit `internal/service/trading_strategy/golden_x/trend_filter.go`. Add a new function after `trendStatusFromCandles`:
  ```go
  // trendStatusFromClosed is the closed-candle-aware variant used by Detect
  // (which receives already-trimmed candles). Equivalent to
  // trendStatusFromCandles but skips the closedWeeklyCandles filter.
  func trendStatusFromClosed(closed []*model.CandleItemTechAnalyse, period int) (dto.TrendStatus, error) {
      if len(closed) < period {
          return dto.TrendUnknown, ErrInsufficientHistory
      }
      closes := make([]float64, len(closed))
      for i, c := range closed {
          closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
      }
      ema := computeEMA(closes, period)
      lastClose := closes[len(closes)-1]
      lastEMA := ema[len(ema)-1]
      if lastClose > lastEMA {
          return dto.TrendWith, nil
      }
      return dto.TrendAgainst, nil
  }
  ```

- [ ] **Step 3: Write `Detect`**

  Create `internal/service/trading_strategy/golden_x/detector.go`:
  ```go
  package golden_x

  import (
      "tinvest/internal/model"
      "tinvest/internal/service/trading_strategy/golden_x/dto"
      "tinvest/internal/utils"
      "tinvest/pkg/indicators"
  )

  // Detect runs the full Golden X signal pipeline against an already-closed
  // weekly candle history for a single share. It is pure: no I/O, no time
  // dependency, no telemetry. Callers (service.Trade and backtest.Replay)
  // are responsible for trimming to closed weeks before invoking.
  //
  // Returns ErrAdaptiveInsufficientHistory or ErrInsufficientHistory when
  // history is too short for the adaptive RSI window or the EMA200 trend
  // filter respectively. Other errors are not currently surfaced.
  func Detect(
      closed []*model.CandleItemTechAnalyse,
      rsiPeriod int,
      kind dto.StrategyKind,
      useTrendFilter bool,
  ) (dto.Signal, error) {
      lastRSI, rsiSeries, thresholds, err := adaptiveRSIForShare(closed, rsiPeriod)
      if err != nil {
          return dto.Signal{}, err
      }

      sig := dto.Signal{
          RSI:            lastRSI,
          Thresholds:     thresholds,
          SellThresholds: adaptiveSellThresholds(rsiSeries),
      }

      if useTrendFilter {
          status, terr := trendStatusFromClosed(closed, trendEMAPeriod)
          if terr != nil {
              return dto.Signal{}, terr
          }
          sig.TrendStatus = status
      }

      buyTier := tierFromAdaptive(lastRSI, thresholds.P5, thresholds.P15)
      sig.GreenBuy = buyTier == tierGreen
      sig.YellowBuy = buyTier == tierYellow

      if buyTier != tierNone {
          lows := lowsAlignedToRSI(closed, rsiPeriod, rsiSeries)
          if len(lows) > divergenceLookbackWeeks {
              lows = lows[len(lows)-divergenceLookbackWeeks:]
          }
          rsiTail := rsiSeries
          if len(rsiTail) > divergenceLookbackWeeks {
              rsiTail = rsiTail[len(rsiTail)-divergenceLookbackWeeks:]
          }
          sig.DivergenceOK = bullishDivergence(lows, rsiTail, divergenceFractalK)

          volumes := make([]int64, len(closed))
          for i, c := range closed {
              volumes[i] = c.Volume
          }
          sig.VolumeOK = indicators.VolumeConfirmed(volumes, volumeSMALookback, volumeMultiplier)

          highs := make([]float64, len(closed))
          lowsF := make([]float64, len(closed))
          closes := make([]float64, len(closed))
          for i, c := range closed {
              highs[i] = utils.CombinePrice(c.High.Units, c.High.Nano)
              lowsF[i] = utils.CombinePrice(c.Low.Units, c.Low.Nano)
              closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
          }
          if atrValue := indicators.ATR(highs, lowsF, closes, atrPeriod); atrValue > 0 {
              lastClose := closes[len(closes)-1]
              sig.LastClose = lastClose
              sig.Stop = stopFromATR(lastClose, atrValue, kForKind(kind))
          } else {
              sig.LastClose = closes[len(closes)-1]
          }
      } else if len(closed) > 0 {
          last := closed[len(closed)-1]
          sig.LastClose = utils.CombinePrice(last.Close.Units, last.Close.Nano)
      }

      _ = kind // already consumed via kForKind / sell thresholds; kept for symmetry
      return sig, nil
  }
  ```

- [ ] **Step 4: Rewire `service.Trade()` to call `Detect`**

  Edit `internal/service/trading_strategy/golden_x/trade.go`. Replace the per-share body of the `for _, share := range in.ShareList.All()` loop (lines ~91–193) so that it:
  1. Fetches candles via `s.fetchWeeklyCandles`.
  2. Filters via `closedWeeklyCandles(candles, dateNow, loc)`.
  3. Calls `sig, err := Detect(closed, share.RSILength, in.Kind, in.UseTrendFilter)`. On `ErrAdaptiveInsufficientHistory` or `ErrInsufficientHistory`, log and `continue`.
  4. Stores `sig.Thresholds` into `thresholds[share.ID]`, `sig.SellThresholds` into `sellThresholds[share.ID]`, and if `in.UseTrendFilter` then `sig.TrendStatus` into `trends[share.ID]`.
  5. If `sig.GreenBuy || sig.YellowBuy`, sets `divergences[share.ID] = sig.DivergenceOK`, `volumesConfirmed[share.ID] = sig.VolumeOK`, and `stops[share.ID] = sig.Stop` (when `sig.Stop != (dto.Stop{})`).
  6. Writes `RSIInfo` row exactly as before.
  7. Recomputes `sellTier` from `sig.RSI` + `sig.SellThresholds` via `sellTierFromAdaptive`. Combines with buy tier exactly as before (buy XOR sell).
  8. Calls `s.state.ShouldAlert(...)` and `buyInfo` / `sellInfo` writes unchanged.

  The bottom of the function (notif.Trade / notif.RSIList) stays unchanged.

  Net diff: per-share inline pipeline disappears, replaced by a `Detect` call plus map-population glue. Everything outside the loop is untouched.

- [ ] **Step 5: Detector unit tests**

  Write `internal/service/trading_strategy/golden_x/detector_test.go`. Use the existing `candleAt` helper from `trend_filter_test.go` (already in package). Add a new helper for full OHLCV candles:
  ```go
  func ohlcCandle(t time.Time, o, h, l, c int64, vol int64) *model.CandleItemTechAnalyse {
      return &model.CandleItemTechAnalyse{
          Time:   t,
          Open:   model.Quotation{Units: o},
          High:   model.Quotation{Units: h},
          Low:    model.Quotation{Units: l},
          Close:  model.Quotation{Units: c},
          Volume: vol,
      }
  }
  ```

  Add table-driven tests for:
  - `insufficient_history`: 50 candles → expect `ErrAdaptiveInsufficientHistory`.
  - `green_buy_no_trend_filter`: build a series whose last close yields `RSI < P5`. Assert `sig.GreenBuy == true`, `sig.YellowBuy == false`, `sig.LastClose > 0`.
  - `yellow_buy`: tune series for `P5 < RSI < P15`. Assert `sig.YellowBuy == true`, `sig.GreenBuy == false`.
  - `no_buy`: rising series whose final RSI is above P15. Assert both false; `sig.Stop` zero (no buy zone).
  - `trend_filter_on_with_strong_uptrend`: useTrendFilter=true, last close > EMA200 over 200+ closed weeks. Assert `sig.TrendStatus == dto.TrendWith`.
  - `stop_computed_when_buy_and_atr_positive`: green buy with non-flat highs/lows. Assert `sig.Stop.Price > 0` and `sig.Stop.DistancePct > 0`.

  Each subtest constructs at least `adaptiveWindowMin + rsiPeriod + 1` candles (≥101 for `rsiPeriod=7`, etc.) and asserts the fields above.

- [ ] **Step 6: Run all golden_x tests**

  Run: `go test ./internal/service/trading_strategy/golden_x/...`
  Expected: all existing tests (`dedup_test`, `percentile_test`, `divergence_test`, `stop_test`, `trend_filter_test`, `rsi_test`, `candle_test`, `notification/notifications_test`) PASS unchanged; the new `detector_test` PASS.

- [ ] **Step 7: Full repo build**

  Run: `go build ./... && go vet ./...`
  Expected: clean.

- [ ] **Step 8: Commit**

  ```bash
  git add internal/service/trading_strategy/golden_x/
  git commit -m "refactor(golden_x): extract pure Detect + Signal from Trade"
  ```

---

## Task 2.3: Position model + partial-exit accounting (TDD)

**Files:**
- Create: `internal/service/trading_strategy/golden_x/backtest/position.go`
- Create: `internal/service/trading_strategy/golden_x/backtest/position_test.go`

- [ ] **Step 1: Write `position_test.go` (failing first)**

  ```go
  package backtest

  import (
      "math"
      "testing"
      "time"

      "tinvest/internal/service/trading_strategy/golden_x/dto"
  )

  func TestPosition_OpenAndStop(t *testing.T) {
      entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
      p := OpenPosition("SBER", entry, 100.0, dto.Stop{Price: 90.0, DistancePct: 10}, dto.StrategyKindDividend)

      if p.UnitsRemaining() != 1.0 {
          t.Fatalf("UnitsRemaining = %v, want 1.0", p.UnitsRemaining())
      }
      if p.FullyClosed() {
          t.Fatalf("FullyClosed = true at open, want false")
      }
      exitWeek := entry.AddDate(0, 0, 7)
      tr := p.CloseAll(exitWeek, 90.0, ExitReasonStop)
      if tr.ExitReason != ExitReasonStop {
          t.Fatalf("ExitReason = %v, want %v", tr.ExitReason, ExitReasonStop)
      }
      if !p.FullyClosed() {
          t.Fatalf("FullyClosed = false after CloseAll")
      }
      // Returns: (90-100)/100 * 100 = -10
      if math.Abs(tr.ReturnPct - -10.0) > 1e-9 {
          t.Fatalf("ReturnPct = %v, want -10", tr.ReturnPct)
      }
      if tr.WeeksHeld != 1 {
          t.Fatalf("WeeksHeld = %d, want 1", tr.WeeksHeld)
      }
  }

  func TestPosition_GoldPartialExits(t *testing.T) {
      entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
      p := OpenPosition("SBER", entry, 100.0, dto.Stop{Price: 80.0}, dto.StrategyKindDividend)

      // Week 4: RSI crosses p80 first time → 1/3 sold at 110.
      w4 := entry.AddDate(0, 0, 7*3)
      partials := p.EvaluateSellExits(w4, 110.0, dto.SellThresholds{P80: 70, P90: 80, P95: 90}, 65)
      if len(partials) != 0 {
          t.Fatalf("RSI 65 below P80 70 → no exits, got %d", len(partials))
      }
      partials = p.EvaluateSellExits(w4, 110.0, dto.SellThresholds{P80: 70, P90: 80, P95: 90}, 72)
      if len(partials) != 1 || partials[0].ExitReason != ExitReasonSellP80 {
          t.Fatalf("expected one P80 partial, got %+v", partials)
      }
      if math.Abs(partials[0].Units-1.0/3.0) > 1e-9 {
          t.Fatalf("Units = %v, want 1/3", partials[0].Units)
      }
      if math.Abs(p.UnitsRemaining()-2.0/3.0) > 1e-9 {
          t.Fatalf("UnitsRemaining after P80 = %v, want 2/3", p.UnitsRemaining())
      }

      // Week 5: RSI now in P90 zone → P90 partial. P80 already triggered, does NOT re-fire.
      w5 := entry.AddDate(0, 0, 7*4)
      partials = p.EvaluateSellExits(w5, 120.0, dto.SellThresholds{P80: 70, P90: 80, P95: 90}, 85)
      if len(partials) != 1 || partials[0].ExitReason != ExitReasonSellP90 {
          t.Fatalf("expected one P90 partial, got %+v", partials)
      }

      // Week 6: RSI in P95 zone → P95 partial. Position fully closed.
      w6 := entry.AddDate(0, 0, 7*5)
      partials = p.EvaluateSellExits(w6, 130.0, dto.SellThresholds{P80: 70, P90: 80, P95: 90}, 95)
      if len(partials) != 1 || partials[0].ExitReason != ExitReasonSellP95 {
          t.Fatalf("expected one P95 partial, got %+v", partials)
      }
      if !p.FullyClosed() {
          t.Fatalf("FullyClosed = false after all 3 partials")
      }
  }

  func TestPosition_GoldAllTiersInOneWeek(t *testing.T) {
      // Spec §4.3 framing: "three independent decisions". If a single week
      // jumps RSI past P95 with no prior partials, all three must fire.
      entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
      p := OpenPosition("X", entry, 100.0, dto.Stop{Price: 80.0}, dto.StrategyKindDividend)
      st := dto.SellThresholds{P80: 70, P90: 80, P95: 90}
      partials := p.EvaluateSellExits(entry.AddDate(0, 0, 7), 120.0, st, 95)
      if len(partials) != 3 {
          t.Fatalf("expected 3 partials in one week, got %d: %+v", len(partials), partials)
      }
      reasons := [3]ExitReason{partials[0].ExitReason, partials[1].ExitReason, partials[2].ExitReason}
      if reasons != [3]ExitReason{ExitReasonSellP80, ExitReasonSellP90, ExitReasonSellP95} {
          t.Fatalf("reason order = %v, want P80,P90,P95", reasons)
      }
      if !p.FullyClosed() {
          t.Fatalf("FullyClosed = false after all three fired")
      }
  }

  func TestPosition_GrowthSingleSellExit(t *testing.T) {
      entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
      p := OpenPosition("YNDX", entry, 100.0, dto.Stop{}, dto.StrategyKindGrowth)

      w4 := entry.AddDate(0, 0, 7*3)
      // Below P90 → no exit.
      partials := p.EvaluateSellExits(w4, 110.0, dto.SellThresholds{P90: 80}, 70)
      if len(partials) != 0 {
          t.Fatalf("RSI below P90 → no exits, got %d", len(partials))
      }
      partials = p.EvaluateSellExits(w4, 110.0, dto.SellThresholds{P90: 80}, 85)
      if len(partials) != 1 || partials[0].ExitReason != ExitReasonSellP90 {
          t.Fatalf("expected one P90 full exit, got %+v", partials)
      }
      if math.Abs(partials[0].Units-1.0) > 1e-9 {
          t.Fatalf("Growth exit Units = %v, want 1.0", partials[0].Units)
      }
      if !p.FullyClosed() {
          t.Fatalf("FullyClosed = false after Growth P90 exit")
      }
  }

  func TestPosition_WeeksHeld(t *testing.T) {
      entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
      p := OpenPosition("X", entry, 100.0, dto.Stop{}, dto.StrategyKindDividend)
      // Exactly 52 weeks later → WeeksHeld must equal 52.
      exit := entry.AddDate(0, 0, 7*52)
      if got := p.WeeksHeldAt(exit); got != 52 {
          t.Fatalf("WeeksHeldAt = %d, want 52", got)
      }
  }
  ```

- [ ] **Step 2: Verify tests fail**

  Run: `go test ./internal/service/trading_strategy/golden_x/backtest/...`
  Expected: FAIL (package missing).

- [ ] **Step 3: Implement `position.go`**

  ```go
  // Package backtest provides the per-share replay engine for the Golden X
  // strategy. It is read-only against historical candles and does not touch
  // the live trading services.
  package backtest

  import (
      "time"

      "tinvest/internal/service/trading_strategy/golden_x/dto"
  )

  // ExitReason enumerates the cause of a closed trade (or partial exit).
  type ExitReason string

  const (
      ExitReasonSellP80   ExitReason = "sell_p80"
      ExitReasonSellP90   ExitReason = "sell_p90"
      ExitReasonSellP95   ExitReason = "sell_p95"
      ExitReasonStop      ExitReason = "stop"
      ExitReasonTimeout   ExitReason = "timeout"
      ExitReasonOpen      ExitReason = "open" // marked-to-market at end of run
  )

  // Trade is a closed (or marked-to-market) backtest row. Each Gold partial
  // exit is one Trade with Units ≈ 1/3; Growth exits produce a single Trade
  // with Units = 1.0. ReturnPct uses the shared entry price.
  type Trade struct {
      ShareID    string
      EntryDate  time.Time
      EntryPrice float64
      ExitDate   time.Time
      ExitPrice  float64
      Units      float64
      ExitReason ExitReason
      ReturnPct  float64
      WeeksHeld  int
  }

  // Position is an open backtest position. Gold positions track three sell
  // flags (P80/P90/P95) that each consume 1/3 units. Growth positions track
  // only P90 and consume the full 1.0 unit at the first trigger.
  type Position struct {
      shareID    string
      entryDate  time.Time
      entryPrice float64
      stopPrice  float64 // 0 means no stop
      kind       dto.StrategyKind
      units      float64
      soldP80    bool
      soldP90    bool
      soldP95    bool
  }

  func OpenPosition(shareID string, entryDate time.Time, entryPrice float64, stop dto.Stop, kind dto.StrategyKind) *Position {
      return &Position{
          shareID:    shareID,
          entryDate:  entryDate,
          entryPrice: entryPrice,
          stopPrice:  stop.Price,
          kind:       kind,
          units:      1.0,
      }
  }

  func (p *Position) ShareID() string       { return p.shareID }
  func (p *Position) EntryDate() time.Time  { return p.entryDate }
  func (p *Position) EntryPrice() float64   { return p.entryPrice }
  func (p *Position) StopPrice() float64    { return p.stopPrice }
  func (p *Position) UnitsRemaining() float64 { return p.units }
  func (p *Position) FullyClosed() bool      { return p.units <= 1e-9 }
  func (p *Position) WeeksHeldAt(t time.Time) int {
      return int(t.Sub(p.entryDate).Hours() / (24 * 7))
  }

  // CloseAll emits a single Trade covering all remaining units. Used for
  // stop hits, 52-week timeouts, and end-of-history mark-to-market.
  func (p *Position) CloseAll(exitDate time.Time, exitPrice float64, reason ExitReason) Trade {
      units := p.units
      p.units = 0
      return Trade{
          ShareID:    p.shareID,
          EntryDate:  p.entryDate,
          EntryPrice: p.entryPrice,
          ExitDate:   exitDate,
          ExitPrice:  exitPrice,
          Units:      units,
          ExitReason: reason,
          ReturnPct:  (exitPrice - p.entryPrice) / p.entryPrice * 100,
          WeeksHeld:  p.WeeksHeldAt(exitDate),
      }
  }

  // EvaluateSellExits returns any partials triggered by the current week's
  // close price + RSI vs the share's sell thresholds. For Gold: up to one
  // partial per call (P80, then P90, then P95 in three different weeks).
  // For Growth: one full exit at the first P90 trigger.
  func (p *Position) EvaluateSellExits(weekEnd time.Time, weekClose float64, st dto.SellThresholds, rsi float64) []Trade {
      if p.FullyClosed() {
          return nil
      }
      var out []Trade
      switch p.kind {
      case dto.StrategyKindGrowth:
          if !p.soldP90 && rsi > st.P90 {
              p.soldP90 = true
              units := p.units
              p.units = 0
              out = append(out, Trade{
                  ShareID:    p.shareID,
                  EntryDate:  p.entryDate,
                  EntryPrice: p.entryPrice,
                  ExitDate:   weekEnd,
                  ExitPrice:  weekClose,
                  Units:      units,
                  ExitReason: ExitReasonSellP90,
                  ReturnPct:  (weekClose - p.entryPrice) / p.entryPrice * 100,
                  WeeksHeld:  p.WeeksHeldAt(weekEnd),
              })
          }
      case dto.StrategyKindDividend:
          third := 1.0 / 3.0
          if !p.soldP80 && rsi > st.P80 {
              p.soldP80 = true
              p.units -= third
              out = append(out, p.partialTrade(weekEnd, weekClose, ExitReasonSellP80, third))
          }
          if !p.soldP90 && rsi > st.P90 {
              p.soldP90 = true
              p.units -= third
              out = append(out, p.partialTrade(weekEnd, weekClose, ExitReasonSellP90, third))
          }
          if !p.soldP95 && rsi > st.P95 {
              p.soldP95 = true
              p.units = 0 // zero out to absorb 1/3 + 1/3 + 1/3 ≠ 1.0 rounding residual
              out = append(out, p.partialTrade(weekEnd, weekClose, ExitReasonSellP95, third))
          }
      }
      // NOTE: the three checks are independent (not else-if). A week with a
      // sudden RSI spike past P95 with no prior partials will consume all
      // three tiers in a single call — matches spec §4.3's "three independent
      // decisions" framing. Add a test in position_test.go for this case
      // (RSI=95 against thresholds {70, 80, 90}, expect 3 partials).
      return out
  }

  func (p *Position) partialTrade(weekEnd time.Time, weekClose float64, reason ExitReason, units float64) Trade {
      return Trade{
          ShareID:    p.shareID,
          EntryDate:  p.entryDate,
          EntryPrice: p.entryPrice,
          ExitDate:   weekEnd,
          ExitPrice:  weekClose,
          Units:      units,
          ExitReason: reason,
          ReturnPct:  (weekClose - p.entryPrice) / p.entryPrice * 100,
          WeeksHeld:  p.WeeksHeldAt(weekEnd),
      }
  }
  ```

- [ ] **Step 4: Verify tests pass**

  Run: `go test ./internal/service/trading_strategy/golden_x/backtest/...`
  Expected: PASS.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/service/trading_strategy/golden_x/backtest/
  git commit -m "feat(golden_x/backtest): add Position model with partial-exit accounting"
  ```

---

## Task 2.4: Candle cache (TDD)

**Files:**
- Create: `internal/service/trading_strategy/golden_x/backtest/cache.go`
- Create: `internal/service/trading_strategy/golden_x/backtest/cache_test.go`

- [ ] **Step 1: Write `cache_test.go` (failing first)**

  ```go
  package backtest

  import (
      "context"
      "errors"
      "os"
      "path/filepath"
      "testing"
      "time"

      "tinvest/internal/model"
  )

  type fakeFetcher struct {
      calls   int
      candles []*model.CandleItemTechAnalyse
      err     error
  }

  func (f *fakeFetcher) Fetch(_ context.Context, _ string) ([]*model.CandleItemTechAnalyse, error) {
      f.calls++
      return f.candles, f.err
  }

  func sampleCandles() []*model.CandleItemTechAnalyse {
      return []*model.CandleItemTechAnalyse{
          {Time: time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
              Open:   model.Quotation{Units: 273, Nano: 410000000},
              High:   model.Quotation{Units: 281, Nano: 200000000},
              Low:    model.Quotation{Units: 270, Nano: 50000000},
              Close:  model.Quotation{Units: 278, Nano: 100000000},
              Volume: 18420000},
          {Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
              Open:   model.Quotation{Units: 279},
              High:   model.Quotation{Units: 282},
              Low:    model.Quotation{Units: 275},
              Close:  model.Quotation{Units: 280},
              Volume: 15000000},
      }
  }

  func TestCache_FetchOnMiss(t *testing.T) {
      dir := t.TempDir()
      f := &fakeFetcher{candles: sampleCandles()}
      c := NewCache(dir, f, false)

      got, err := c.Get(context.Background(), "SBER")
      if err != nil {
          t.Fatalf("Get: %v", err)
      }
      if len(got) != 2 {
          t.Fatalf("len=%d, want 2", len(got))
      }
      if f.calls != 1 {
          t.Fatalf("fetcher called %d times, want 1", f.calls)
      }
      // File should exist.
      if _, err := os.Stat(filepath.Join(dir, "SBER_W.json")); err != nil {
          t.Fatalf("expected cache file: %v", err)
      }
  }

  func TestCache_HitNoFetch(t *testing.T) {
      dir := t.TempDir()
      f := &fakeFetcher{candles: sampleCandles()}
      c := NewCache(dir, f, false)

      if _, err := c.Get(context.Background(), "SBER"); err != nil {
          t.Fatalf("priming Get: %v", err)
      }
      // Second call with a fetcher set to error should not call fetcher.
      f.err = errors.New("boom")
      f.candles = nil
      f.calls = 0

      got, err := c.Get(context.Background(), "SBER")
      if err != nil {
          t.Fatalf("cache hit Get: %v", err)
      }
      if len(got) != 2 {
          t.Fatalf("len=%d, want 2", len(got))
      }
      if f.calls != 0 {
          t.Fatalf("fetcher called on hit (%d calls)", f.calls)
      }
  }

  func TestCache_RefreshOverwrites(t *testing.T) {
      dir := t.TempDir()
      f := &fakeFetcher{candles: sampleCandles()}
      cWarm := NewCache(dir, f, false)
      if _, err := cWarm.Get(context.Background(), "SBER"); err != nil {
          t.Fatalf("warming: %v", err)
      }

      // Build a refresh cache with fresh data.
      f2 := &fakeFetcher{candles: append(sampleCandles(), &model.CandleItemTechAnalyse{
          Time:  time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC),
          Close: model.Quotation{Units: 290},
      })}
      cRefresh := NewCache(dir, f2, true)
      got, err := cRefresh.Get(context.Background(), "SBER")
      if err != nil {
          t.Fatalf("refresh Get: %v", err)
      }
      if len(got) != 3 {
          t.Fatalf("after refresh len=%d, want 3", len(got))
      }
      if f2.calls != 1 {
          t.Fatalf("refresh fetcher calls = %d, want 1", f2.calls)
      }
  }

  func TestCache_RoundTripPreservesFields(t *testing.T) {
      dir := t.TempDir()
      f := &fakeFetcher{candles: sampleCandles()}
      c := NewCache(dir, f, false)
      _, _ = c.Get(context.Background(), "SBER")

      // Drop in-memory state by creating a second cache pointing at the same dir.
      c2 := NewCache(dir, &fakeFetcher{err: errors.New("must not call")}, false)
      got, err := c2.Get(context.Background(), "SBER")
      if err != nil {
          t.Fatalf("re-read: %v", err)
      }
      want := sampleCandles()
      if len(got) != len(want) {
          t.Fatalf("len mismatch")
      }
      for i := range want {
          if !got[i].Time.Equal(want[i].Time) {
              t.Fatalf("Time[%d] mismatch", i)
          }
          if got[i].Close.Units != want[i].Close.Units || got[i].Close.Nano != want[i].Close.Nano {
              t.Fatalf("Close[%d] mismatch", i)
          }
          if got[i].Volume != want[i].Volume {
              t.Fatalf("Volume[%d] mismatch", i)
          }
      }
  }
  ```

- [ ] **Step 2: Verify tests fail**

  Run: `go test ./internal/service/trading_strategy/golden_x/backtest/... -run TestCache`
  Expected: FAIL (`NewCache` undefined).

- [ ] **Step 3: Implement `cache.go`**

  ```go
  package backtest

  import (
      "context"
      "encoding/json"
      "fmt"
      "os"
      "path/filepath"
      "time"

      "tinvest/internal/model"
      "tinvest/internal/utils"
  )

  // CandleFetcher returns the full weekly-candle history for a share. The
  // backtest cache calls it on misses (and on every call when --refresh).
  type CandleFetcher interface {
      Fetch(ctx context.Context, shareID string) ([]*model.CandleItemTechAnalyse, error)
  }

  // Cache stores weekly candles per share as JSON on disk. JSON encodes
  // float64 prices (already converted via utils.CombinePrice) rather than
  // the Quotation{Units, Nano} pair, to keep the file readable.
  type Cache struct {
      dir     string
      fetcher CandleFetcher
      refresh bool
  }

  func NewCache(dir string, fetcher CandleFetcher, refresh bool) *Cache {
      return &Cache{dir: dir, fetcher: fetcher, refresh: refresh}
  }

  type diskCandle struct {
      Date   time.Time `json:"date"`
      Open   float64   `json:"open"`
      High   float64   `json:"high"`
      Low    float64   `json:"low"`
      Close  float64   `json:"close"`
      Volume int64     `json:"volume"`
  }

  func (c *Cache) path(shareID string) string {
      return filepath.Join(c.dir, shareID+"_W.json")
  }

  func (c *Cache) Get(ctx context.Context, shareID string) ([]*model.CandleItemTechAnalyse, error) {
      if !c.refresh {
          if candles, ok, err := c.readDisk(shareID); err != nil {
              return nil, err
          } else if ok {
              return candles, nil
          }
      }
      fetched, err := c.fetcher.Fetch(ctx, shareID)
      if err != nil {
          return nil, fmt.Errorf("fetch candles for %s: %w", shareID, err)
      }
      if err := c.writeDisk(shareID, fetched); err != nil {
          return nil, fmt.Errorf("write cache for %s: %w", shareID, err)
      }
      return fetched, nil
  }

  func (c *Cache) readDisk(shareID string) ([]*model.CandleItemTechAnalyse, bool, error) {
      data, err := os.ReadFile(c.path(shareID))
      if errIsNotExist(err) {
          return nil, false, nil
      }
      if err != nil {
          return nil, false, err
      }
      var disk []diskCandle
      if err := json.Unmarshal(data, &disk); err != nil {
          return nil, false, err
      }
      out := make([]*model.CandleItemTechAnalyse, len(disk))
      for i, d := range disk {
          oU, oN := utils.SplitPrice(d.Open)
          hU, hN := utils.SplitPrice(d.High)
          lU, lN := utils.SplitPrice(d.Low)
          cU, cN := utils.SplitPrice(d.Close)
          out[i] = &model.CandleItemTechAnalyse{
              Time:   d.Date,
              Open:   model.Quotation{Units: oU, Nano: oN},
              High:   model.Quotation{Units: hU, Nano: hN},
              Low:    model.Quotation{Units: lU, Nano: lN},
              Close:  model.Quotation{Units: cU, Nano: cN},
              Volume: d.Volume,
          }
      }
      return out, true, nil
  }

  func (c *Cache) writeDisk(shareID string, candles []*model.CandleItemTechAnalyse) error {
      if err := os.MkdirAll(c.dir, 0o755); err != nil {
          return err
      }
      disk := make([]diskCandle, len(candles))
      for i, k := range candles {
          disk[i] = diskCandle{
              Date:   k.Time,
              Open:   utils.CombinePrice(k.Open.Units, k.Open.Nano),
              High:   utils.CombinePrice(k.High.Units, k.High.Nano),
              Low:    utils.CombinePrice(k.Low.Units, k.Low.Nano),
              Close:  utils.CombinePrice(k.Close.Units, k.Close.Nano),
              Volume: k.Volume,
          }
      }
      data, err := json.MarshalIndent(disk, "", "  ")
      if err != nil {
          return err
      }
      return os.WriteFile(c.path(shareID), data, 0o644)
  }

  func errIsNotExist(err error) bool { return err != nil && os.IsNotExist(err) }
  ```

  **Note on `utils.SplitPrice`:** that helper currently lives in `internal/utils/utils.go:19` — verify its semantics before relying on it for the JSON round-trip. If its int/frac decomposition is inverted (the existing implementation has `frac, intPart := math.Modf(price)` then returns them swapped), the test `TestCache_RoundTripPreservesFields` will fail at the second-pair candle's Nano comparison. In that case implement local helpers in `cache.go`:
  ```go
  func splitPriceLocal(p float64) (int64, int32) {
      units := int64(p)
      nano := int32(math.Round((p - float64(units)) * 1e9))
      return units, nano
  }
  ```
  and call those instead of `utils.SplitPrice`. Choose the working path; do not silently use a broken utility.

- [ ] **Step 4: Verify tests pass**

  Run: `go test ./internal/service/trading_strategy/golden_x/backtest/... -run TestCache`
  Expected: PASS.

- [ ] **Step 5: Add `cache/` to `.gitignore`**

  Edit `.gitignore` to append:
  ```
  /cache
  ```
  Verify nothing under `cache/` is tracked: `git status --ignored | grep cache`.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/service/trading_strategy/golden_x/backtest/cache.go internal/service/trading_strategy/golden_x/backtest/cache_test.go .gitignore
  git commit -m "feat(golden_x/backtest): add disk-cached candle store"
  ```

---

## Task 2.5: Replay engine (TDD)

**Files:**
- Create: `internal/service/trading_strategy/golden_x/backtest/replay.go`
- Create: `internal/service/trading_strategy/golden_x/backtest/replay_test.go`

- [ ] **Step 1: Write `replay_test.go` (failing first)**

  Use synthetic candles. The replay test focuses on exit ordering rather than recomputing Detect — provide a deterministic fake detector via a function type to keep the test fast.

  ```go
  package backtest

  import (
      "testing"
      "time"

      "tinvest/internal/model"
      "tinvest/internal/service/trading_strategy/golden_x/dto"
  )

  func mkCandle(t time.Time, low, close int64) *model.CandleItemTechAnalyse {
      return &model.CandleItemTechAnalyse{
          Time:  t,
          High:  model.Quotation{Units: close + 1},
          Low:   model.Quotation{Units: low},
          Close: model.Quotation{Units: close},
      }
  }

  // TestReplay_StopFiresBeforeSellTier verifies §4.2 ordering 1a → 1b.
  func TestReplay_StopFiresBeforeSellTier(t *testing.T) {
      // Setup: 5 weeks of candles. Week 3 triggers entry. Week 4: Low touches
      // stop AND RSI hits sell-P95 — stop must win.
      base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
      candles := []*model.CandleItemTechAnalyse{
          mkCandle(base, 100, 100),
          mkCandle(base.AddDate(0, 0, 7), 100, 100),
          mkCandle(base.AddDate(0, 0, 14), 100, 100), // entry week
          mkCandle(base.AddDate(0, 0, 21), 80, 110),  // Low 80 hits stop @ 90; close 110 also above P95
          mkCandle(base.AddDate(0, 0, 28), 110, 110),
      }
      // Fake detector: only week-index 2 produces green buy with stop=90; later
      // weeks produce a high RSI that, if reached, would emit P95 sell.
      fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool) (dto.Signal, error) {
          last := closed[len(closed)-1]
          switch last.Time {
          case candles[2].Time:
              return dto.Signal{GreenBuy: true, RSI: 5, LastClose: 100,
                  Stop:           dto.Stop{Price: 90},
                  SellThresholds: dto.SellThresholds{P80: 70, P90: 80, P95: 90}}, nil
          case candles[3].Time, candles[4].Time:
              return dto.Signal{RSI: 95, LastClose: 110,
                  SellThresholds: dto.SellThresholds{P80: 70, P90: 80, P95: 90}}, nil
          }
          return dto.Signal{RSI: 50, SellThresholds: dto.SellThresholds{P80: 70, P90: 80, P95: 90}}, nil
      }
      cfg := ReplayConfig{Kind: dto.StrategyKindDividend, StartIdx: 2, MaxWeeks: 52, UseTrendFilter: false}
      trades := Replay("X", candles, fake, cfg)

      if len(trades) != 1 || trades[0].ExitReason != ExitReasonStop {
          t.Fatalf("expected single stop trade, got %+v", trades)
      }
  }

  // TestReplay_SellSequenceProducesThreePartials covers Gold p80→p90→p95
  // across three different weeks (§4.3).
  func TestReplay_SellSequenceProducesThreePartials(t *testing.T) {
      base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
      const N = 8
      candles := make([]*model.CandleItemTechAnalyse, N)
      for i := 0; i < N; i++ {
          candles[i] = mkCandle(base.AddDate(0, 0, 7*i), 95, 100+int64(i))
      }
      st := dto.SellThresholds{P80: 70, P90: 80, P95: 90}
      // Entry at week 2; weeks 3,4,5 hit P80, P90, P95 respectively.
      fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool) (dto.Signal, error) {
          idx := len(closed) - 1
          switch idx {
          case 2:
              return dto.Signal{GreenBuy: true, LastClose: 102, Stop: dto.Stop{Price: 1}, SellThresholds: st}, nil
          case 3:
              return dto.Signal{RSI: 72, LastClose: 103, SellThresholds: st}, nil
          case 4:
              return dto.Signal{RSI: 85, LastClose: 104, SellThresholds: st}, nil
          case 5:
              return dto.Signal{RSI: 95, LastClose: 105, SellThresholds: st}, nil
          default:
              return dto.Signal{RSI: 50, SellThresholds: st}, nil
          }
      }
      trades := Replay("X", candles, fake, ReplayConfig{Kind: dto.StrategyKindDividend, StartIdx: 2, MaxWeeks: 52})
      if len(trades) != 3 {
          t.Fatalf("expected 3 partials, got %d: %+v", len(trades), trades)
      }
      reasons := [3]ExitReason{trades[0].ExitReason, trades[1].ExitReason, trades[2].ExitReason}
      want := [3]ExitReason{ExitReasonSellP80, ExitReasonSellP90, ExitReasonSellP95}
      if reasons != want {
          t.Fatalf("reasons=%v, want %v", reasons, want)
      }
  }

  // TestReplay_TimeoutAfter52Weeks covers exit-ordering rule 1c.
  func TestReplay_TimeoutAfter52Weeks(t *testing.T) {
      base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
      const N = 60
      candles := make([]*model.CandleItemTechAnalyse, N)
      for i := 0; i < N; i++ {
          candles[i] = mkCandle(base.AddDate(0, 0, 7*i), 90, 100)
      }
      fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool) (dto.Signal, error) {
          idx := len(closed) - 1
          if idx == 2 {
              return dto.Signal{GreenBuy: true, LastClose: 100, Stop: dto.Stop{Price: 80}}, nil
          }
          return dto.Signal{RSI: 50}, nil
      }
      trades := Replay("X", candles, fake, ReplayConfig{Kind: dto.StrategyKindDividend, StartIdx: 2, MaxWeeks: 52})
      if len(trades) != 1 || trades[0].ExitReason != ExitReasonTimeout {
          t.Fatalf("expected one timeout, got %+v", trades)
      }
      if trades[0].WeeksHeld < 52 {
          t.Fatalf("WeeksHeld = %d, want >=52", trades[0].WeeksHeld)
      }
  }

  // TestReplay_OpenAtEndOfHistory ensures positions still open at the last
  // candle are recorded with ExitReasonOpen + close-as-exit-price.
  func TestReplay_OpenAtEndOfHistory(t *testing.T) {
      base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
      const N = 5
      candles := make([]*model.CandleItemTechAnalyse, N)
      for i := 0; i < N; i++ {
          candles[i] = mkCandle(base.AddDate(0, 0, 7*i), 90, 100+int64(i))
      }
      fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool) (dto.Signal, error) {
          if len(closed) == 3 {
              return dto.Signal{GreenBuy: true, LastClose: 102, Stop: dto.Stop{Price: 50}}, nil
          }
          return dto.Signal{RSI: 50}, nil
      }
      trades := Replay("X", candles, fake, ReplayConfig{Kind: dto.StrategyKindDividend, StartIdx: 2, MaxWeeks: 52})
      if len(trades) != 1 || trades[0].ExitReason != ExitReasonOpen {
          t.Fatalf("expected one open trade, got %+v", trades)
      }
      // Exit price = last candle's close.
      lastClose := float64(candles[N-1].Close.Units)
      if trades[0].ExitPrice != lastClose {
          t.Fatalf("ExitPrice = %v, want %v", trades[0].ExitPrice, lastClose)
      }
  }
  ```

- [ ] **Step 2: Verify tests fail**

  Run: `go test ./internal/service/trading_strategy/golden_x/backtest/... -run TestReplay`
  Expected: FAIL (`Replay`, `ReplayConfig` undefined).

- [ ] **Step 3: Implement `replay.go`**

  ```go
  package backtest

  import (
      "tinvest/internal/model"
      "tinvest/internal/service/trading_strategy/golden_x/dto"
      "tinvest/internal/utils"
  )

  // DetectFunc is the signature of golden_x.Detect; injected so the replay
  // engine can be unit-tested with a fake detector.
  type DetectFunc func(closed []*model.CandleItemTechAnalyse, rsiPeriod int, kind dto.StrategyKind, useTrendFilter bool) (dto.Signal, error)

  type ReplayConfig struct {
      Kind           dto.StrategyKind
      RSIPeriod      int   // per-share; pulled from collection.Instrument by caller
      StartIdx       int   // first index at which we will evaluate a signal
      MaxWeeks       int   // timeout cap (52 per spec §4.2)
      UseTrendFilter bool
  }

  // Replay iterates the closed-weekly-candle slice and emits Trade rows per
  // §4.2 exit ordering: stop → sell-tier → timeout, with end-of-history
  // mark-to-market for any remaining open position. Detect is called on the
  // slice `candles[:t+1]` for each t; the caller is responsible for ensuring
  // `candles` is already trimmed to closed weeks.
  func Replay(shareID string, candles []*model.CandleItemTechAnalyse, detect DetectFunc, cfg ReplayConfig) []Trade {
      var (
          trades []Trade
          pos    *Position
      )
      for t := cfg.StartIdx; t < len(candles); t++ {
          closed := candles[:t+1]
          weekT := candles[t]
          sig, err := detect(closed, cfg.RSIPeriod, cfg.Kind, cfg.UseTrendFilter)
          if err != nil {
              continue
          }
          weekClose := utils.CombinePrice(weekT.Close.Units, weekT.Close.Nano)
          weekLow := utils.CombinePrice(weekT.Low.Units, weekT.Low.Nano)

          // 1. Exit-ordering for any open position.
          if pos != nil {
              if pos.StopPrice() > 0 && weekLow <= pos.StopPrice() {
                  // 1a. Stop hit on this week's Low.
                  trades = append(trades, pos.CloseAll(weekT.Time, pos.StopPrice(), ExitReasonStop))
                  pos = nil
              } else if partials := pos.EvaluateSellExits(weekT.Time, weekClose, sig.SellThresholds, sig.RSI); len(partials) > 0 {
                  // 1b. Sell-tier exits at week close.
                  trades = append(trades, partials...)
                  if pos.FullyClosed() {
                      pos = nil
                  }
              } else if pos.WeeksHeldAt(weekT.Time) >= cfg.MaxWeeks {
                  // 1c. Timeout.
                  trades = append(trades, pos.CloseAll(weekT.Time, weekClose, ExitReasonTimeout))
                  pos = nil
              }
          }

          // 2. Entry: only on green tier, only when flat.
          if pos == nil && sig.GreenBuy {
              pos = OpenPosition(shareID, weekT.Time, weekClose, sig.Stop, cfg.Kind)
          }
      }
      // End-of-history mark-to-market.
      if pos != nil && len(candles) > 0 {
          last := candles[len(candles)-1]
          lastClose := utils.CombinePrice(last.Close.Units, last.Close.Nano)
          trades = append(trades, pos.CloseAll(last.Time, lastClose, ExitReasonOpen))
      }
      return trades
  }
  ```

- [ ] **Step 4: Verify tests pass**

  Run: `go test ./internal/service/trading_strategy/golden_x/backtest/... -run TestReplay`
  Expected: PASS.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/service/trading_strategy/golden_x/backtest/replay.go internal/service/trading_strategy/golden_x/backtest/replay_test.go
  git commit -m "feat(golden_x/backtest): add per-share replay engine"
  ```

---

## Task 2.6: Report aggregation + Markdown formatter (TDD)

**Files:**
- Create: `internal/service/trading_strategy/golden_x/backtest/report.go`
- Create: `internal/service/trading_strategy/golden_x/backtest/report_test.go`

- [ ] **Step 1: Write `report_test.go` (failing first)**

  ```go
  package backtest

  import (
      "math"
      "strings"
      "testing"
      "time"

      "tinvest/internal/service/trading_strategy/golden_x/dto"
  )

  func makeTrade(share string, entry time.Time, retPct, units float64, reason ExitReason) Trade {
      return Trade{
          ShareID:    share,
          EntryDate:  entry,
          EntryPrice: 100,
          ExitDate:   entry.AddDate(0, 0, 7*4),
          ExitPrice:  100 + retPct,
          Units:      units,
          ExitReason: reason,
          ReturnPct:  retPct,
          WeeksHeld:  4,
      }
  }

  func TestStats_WinRateAndCumulative(t *testing.T) {
      base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
      trades := []Trade{
          makeTrade("A", base, 10, 1, ExitReasonSellP90),
          makeTrade("A", base.AddDate(0, 0, 7), -5, 1, ExitReasonStop),
          makeTrade("B", base, 0, 1, ExitReasonTimeout),
          makeTrade("C", base, 20, 1, ExitReasonOpen),
      }
      st := AggregateStats(trades)
      if st.Count != 4 || st.Wins != 1 || st.Losses != 1 || st.Open != 1 {
          t.Fatalf("counts: %+v", st)
      }
      // WinRate excludes Open and the zero-return trade.
      // Wins=1, Losses=1 → 0.5.
      if math.Abs(st.WinRate-0.5) > 1e-9 {
          t.Fatalf("WinRate=%v, want 0.5", st.WinRate)
      }
      // Cumulative ignores Open (mark-to-market), or you can choose to include
      // it — pick one convention and stick. Spec §5.1: "CumulativeReturn = Σ
      // (ReturnPct * Units)" — includes open as mark-to-market. So:
      // 10 + -5 + 0 + 20 = 25.
      if math.Abs(st.CumulativeReturn-25.0) > 1e-9 {
          t.Fatalf("CumulativeReturn=%v, want 25", st.CumulativeReturn)
      }
  }

  func TestStats_MaxDrawdownNonMonotonic(t *testing.T) {
      base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
      // Equity curve via dated entries: +20, -30, +10, -5
      // Cumulative: 20, -10, 0, -5.
      // Peak so far: 20, 20, 20, 20.
      // Drawdowns from peak: 0, 30, 20, 25.
      // MaxDrawdown = 30 (as absolute percentage points on the equity curve).
      trades := []Trade{
          makeTrade("A", base.AddDate(0, 0, 0), 20, 1, ExitReasonSellP90),
          makeTrade("A", base.AddDate(0, 0, 7), -30, 1, ExitReasonStop),
          makeTrade("A", base.AddDate(0, 0, 14), 10, 1, ExitReasonSellP90),
          makeTrade("A", base.AddDate(0, 0, 21), -5, 1, ExitReasonStop),
      }
      st := AggregateStats(trades)
      if math.Abs(st.MaxDrawdown-30.0) > 1e-9 {
          t.Fatalf("MaxDrawdown=%v, want 30", st.MaxDrawdown)
      }
  }

  func TestRenderMarkdown_ContainsSections(t *testing.T) {
      base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
      trades := []Trade{
          makeTrade("A", base, 10, 1, ExitReasonSellP90),
          makeTrade("B", base, -5, 1, ExitReasonStop),
      }
      r := Report{
          Kind:    dto.StrategyKindDividend,
          From:    base,
          To:      base.AddDate(0, 1, 0),
          Trades:  trades,
          Overall: AggregateStats(trades),
          PerShare: map[string]Stats{
              "A": AggregateStats(trades[:1]),
              "B": AggregateStats(trades[1:]),
          },
      }
      out := RenderMarkdown(r)
      for _, want := range []string{"## Overall", "## Exit reasons", "## Per share", "## Trades"} {
          if !strings.Contains(out, want) {
              t.Fatalf("missing section %q", want)
          }
      }
  }
  ```

- [ ] **Step 2: Verify tests fail**

  Run: `go test ./internal/service/trading_strategy/golden_x/backtest/... -run "TestStats|TestRender"`
  Expected: FAIL.

- [ ] **Step 3: Implement `report.go`**

  ```go
  package backtest

  import (
      "fmt"
      "sort"
      "strings"
      "time"

      "tinvest/internal/service/trading_strategy/golden_x/dto"
  )

  type Stats struct {
      Count, Wins, Losses, Open int
      WinRate                   float64
      AvgReturnPct              float64
      MedianReturn              float64
      CumulativeReturn          float64
      MaxDrawdown               float64
      AvgWeeksHeld              float64
      ExitReasons               map[ExitReason]int
  }

  type Report struct {
      Kind     dto.StrategyKind
      From, To time.Time
      Trades   []Trade
      PerShare map[string]Stats
      Overall  Stats
  }

  func AggregateStats(trades []Trade) Stats {
      out := Stats{ExitReasons: map[ExitReason]int{}}
      if len(trades) == 0 {
          return out
      }
      var sumReturn, sumWeeks float64
      returns := make([]float64, 0, len(trades))
      for _, tr := range trades {
          out.Count++
          out.ExitReasons[tr.ExitReason]++
          out.CumulativeReturn += tr.ReturnPct * tr.Units
          sumReturn += tr.ReturnPct
          sumWeeks += float64(tr.WeeksHeld)
          returns = append(returns, tr.ReturnPct)
          switch {
          case tr.ExitReason == ExitReasonOpen:
              out.Open++
          case tr.ReturnPct > 0:
              out.Wins++
          case tr.ReturnPct < 0:
              out.Losses++
          }
      }
      out.AvgReturnPct = sumReturn / float64(out.Count)
      out.AvgWeeksHeld = sumWeeks / float64(out.Count)
      sort.Float64s(returns)
      out.MedianReturn = median(returns)
      if out.Wins+out.Losses > 0 {
          out.WinRate = float64(out.Wins) / float64(out.Wins+out.Losses)
      }
      out.MaxDrawdown = maxDrawdown(trades)
      return out
  }

  func median(sorted []float64) float64 {
      n := len(sorted)
      if n == 0 {
          return 0
      }
      if n%2 == 1 {
          return sorted[n/2]
      }
      return (sorted[n/2-1] + sorted[n/2]) / 2
  }

  func maxDrawdown(trades []Trade) float64 {
      if len(trades) == 0 {
          return 0
      }
      ordered := make([]Trade, len(trades))
      copy(ordered, trades)
      sort.Slice(ordered, func(i, j int) bool {
          return ordered[i].EntryDate.Before(ordered[j].EntryDate)
      })
      var equity, peak, worst float64
      for _, tr := range ordered {
          equity += tr.ReturnPct * tr.Units
          if equity > peak {
              peak = equity
          }
          dd := peak - equity
          if dd > worst {
              worst = dd
          }
      }
      return worst
  }

  func RenderMarkdown(r Report) string {
      var b strings.Builder
      fmt.Fprintf(&b, "# Golden X backtest — %s\n\n", r.Kind.Medal())
      fmt.Fprintf(&b, "_Range: %s → %s · generated %s_\n\n",
          r.From.Format("2006-01-02"), r.To.Format("2006-01-02"), time.Now().Format(time.RFC3339))

      b.WriteString("## Overall\n\n")
      writeStatsTable(&b, r.Overall)
      b.WriteString("\n")

      b.WriteString("## Exit reasons\n\n")
      writeExitReasons(&b, r.Trades)
      b.WriteString("\n")

      b.WriteString("## Per share\n\n")
      writePerShare(&b, r.PerShare)
      b.WriteString("\n")

      b.WriteString("## Trades (chronological)\n\n")
      writeTrades(&b, r.Trades)
      return b.String()
  }

  func writeStatsTable(b *strings.Builder, s Stats) {
      b.WriteString("| Count | Wins | Losses | Open | WinRate | AvgReturn% | Median% | Cumulative% | MaxDD% | AvgWeeks |\n")
      b.WriteString("|------:|-----:|-------:|-----:|--------:|-----------:|--------:|------------:|-------:|---------:|\n")
      fmt.Fprintf(b, "| %d | %d | %d | %d | %.2f | %.2f | %.2f | %.2f | %.2f | %.1f |\n",
          s.Count, s.Wins, s.Losses, s.Open, s.WinRate*100, s.AvgReturnPct, s.MedianReturn,
          s.CumulativeReturn, s.MaxDrawdown, s.AvgWeeksHeld)
  }

  func writeExitReasons(b *strings.Builder, trades []Trade) {
      counts := map[ExitReason]int{}
      sums := map[ExitReason]float64{}
      for _, tr := range trades {
          counts[tr.ExitReason]++
          sums[tr.ExitReason] += tr.ReturnPct
      }
      reasons := make([]ExitReason, 0, len(counts))
      for r := range counts {
          reasons = append(reasons, r)
      }
      sort.Slice(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })
      b.WriteString("| Reason | Count | AvgReturn% |\n|---|---:|---:|\n")
      for _, r := range reasons {
          n := counts[r]
          fmt.Fprintf(b, "| %s | %d | %.2f |\n", r, n, sums[r]/float64(n))
      }
  }

  func writePerShare(b *strings.Builder, per map[string]Stats) {
      ids := make([]string, 0, len(per))
      for id := range per {
          ids = append(ids, id)
      }
      sort.Strings(ids)
      b.WriteString("| Share | Count | WinRate | Cumulative% | MaxDD% |\n|---|---:|---:|---:|---:|\n")
      for _, id := range ids {
          s := per[id]
          fmt.Fprintf(b, "| %s | %d | %.2f | %.2f | %.2f |\n",
              id, s.Count, s.WinRate*100, s.CumulativeReturn, s.MaxDrawdown)
      }
  }

  func writeTrades(b *strings.Builder, trades []Trade) {
      ordered := make([]Trade, len(trades))
      copy(ordered, trades)
      sort.Slice(ordered, func(i, j int) bool { return ordered[i].EntryDate.Before(ordered[j].EntryDate) })
      b.WriteString("| Share | Entry | Exit | EntryPx | ExitPx | Units | Reason | Return% | Weeks |\n|---|---|---|---:|---:|---:|---|---:|---:|\n")
      for _, tr := range ordered {
          fmt.Fprintf(b, "| %s | %s | %s | %.2f | %.2f | %.3f | %s | %.2f | %d |\n",
              tr.ShareID, tr.EntryDate.Format("2006-01-02"), tr.ExitDate.Format("2006-01-02"),
              tr.EntryPrice, tr.ExitPrice, tr.Units, tr.ExitReason, tr.ReturnPct, tr.WeeksHeld)
      }
  }
  ```

- [ ] **Step 4: Verify tests pass**

  Run: `go test ./internal/service/trading_strategy/golden_x/backtest/...`
  Expected: all PASS.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/service/trading_strategy/golden_x/backtest/report.go internal/service/trading_strategy/golden_x/backtest/report_test.go
  git commit -m "feat(golden_x/backtest): aggregate stats + Markdown report"
  ```

---

## Task 2.7: `cmd/backtest` binary

**Files:**
- Create: `cmd/backtest/main.go`

- [ ] **Step 1: Implement the binary**

  ```go
  package main

  import (
      "context"
      "flag"
      "fmt"
      "log"
      "os"
      "path/filepath"
      "strings"
      "time"

      "github.com/joho/godotenv"
      "google.golang.org/protobuf/types/known/timestamppb"

      "tinvest/internal/enum"
      "tinvest/internal/model"
      golden_x "tinvest/internal/service/trading_strategy/golden_x"
      "tinvest/internal/service/trading_strategy/golden_x/backtest"
      "tinvest/internal/service/trading_strategy/golden_x/dto"
      "tinvest/internal/service/trading_strategy/golden_x/shares"
      "tinvest/internal/utils"
      "tinvest/pkg/client/grpc"
      "tinvest/pkg/collection"
  )

  func main() {
      var (
          kindStr    = flag.String("kind", "Dividend", "Dividend | Growth")
          fromStr    = flag.String("from", "2022-01-01", "inclusive start date (MSK)")
          toStr      = flag.String("to", "", "inclusive end date (default: today)")
          sharesCSV  = flag.String("shares", "", "comma-separated share IDs (default: all)")
          refresh    = flag.Bool("refresh", false, "force re-fetch from Tinkoff API")
          outDir     = flag.String("out-dir", "cache/backtests", "report output directory")
          cacheDir   = flag.String("cache-dir", "cache/candles", "candle cache directory")
      )
      flag.Parse()

      kind, err := parseKind(*kindStr)
      if err != nil {
          log.Fatal(err)
      }
      msk, _ := time.LoadLocation("Europe/Moscow")
      from, err := time.ParseInLocation("2006-01-02", *fromStr, msk)
      if err != nil {
          log.Fatalf("parse --from: %v", err)
      }
      to := time.Now().In(msk)
      if *toStr != "" {
          to, err = time.ParseInLocation("2006-01-02", *toStr, msk)
          if err != nil {
              log.Fatalf("parse --to: %v", err)
          }
      }

      list := selectShareList(kind, *sharesCSV)
      if len(list.All()) == 0 {
          log.Fatalf("no shares selected")
      }

      // Token only needed if any share misses cache or --refresh is set.
      _ = godotenv.Load("env/local.env")
      _ = godotenv.Load("env/token.env")
      token := os.Getenv("T_BANK")
      var grpcClient grpc.GrpcClient
      needsFetch := *refresh || anyCacheMiss(*cacheDir, list)
      if needsFetch {
          if token == "" {
              log.Fatalf("T_BANK env var required (no cache and not --refresh)")
          }
          grpcClient, err = grpc.NewClientGrpc("invest-public-api.tinkoff.ru:443", token)
          if err != nil {
              log.Fatalf("grpc dial: %v", err)
          }
      }

      ctx := context.Background()
      report := backtest.Report{Kind: kind, From: from, To: to, PerShare: map[string]backtest.Stats{}}
      useTrendFilter := kind == dto.StrategyKindGrowth

      for _, instr := range list.All() {
          fetcher := newGrpcFetcher(grpcClient, to)
          cache := backtest.NewCache(*cacheDir, fetcher, *refresh)
          raw, ferr := cache.Get(ctx, instr.ID)
          if ferr != nil {
              log.Printf("WARN %s: %v", instr.Name, ferr)
              continue
          }
          closed := filterClosed(raw, msk)
          closed = trimByDateRange(closed, from, to)
          startIdx := chooseStartIdx(instr.RSILength)
          if len(closed) <= startIdx {
              continue
          }
          trades := backtest.Replay(instr.ID, closed,
              func(c []*model.CandleItemTechAnalyse, period int, k dto.StrategyKind, uft bool) (dto.Signal, error) {
                  return golden_x.Detect(c, period, k, uft)
              },
              backtest.ReplayConfig{
                  Kind: kind, RSIPeriod: instr.RSILength, StartIdx: startIdx,
                  MaxWeeks: 52, UseTrendFilter: useTrendFilter,
              })
          report.Trades = append(report.Trades, trades...)
          report.PerShare[instr.ID] = backtest.AggregateStats(trades)
      }
      report.Overall = backtest.AggregateStats(report.Trades)

      if err := os.MkdirAll(*outDir, 0o755); err != nil {
          log.Fatalf("mkdir: %v", err)
      }
      stamp := time.Now().Format("2006-01-02_1504")
      path := filepath.Join(*outDir, fmt.Sprintf("%s_%s.md", stamp, kindLabel(kind)))
      if err := os.WriteFile(path, []byte(backtest.RenderMarkdown(report)), 0o644); err != nil {
          log.Fatalf("write report: %v", err)
      }

      // Console summary.
      var b strings.Builder
      b.WriteString("## Overall\n\n")
      // Re-use the writer functions by hand.
      // (Already covered by RenderMarkdown; print its Overall + ExitReasons blocks via slicing the full report
      // would be brittle. Just print the two sections directly.)
      fmt.Println(consoleSummary(report))
      fmt.Printf("\nReport written to: %s\n", path)
  }

  func parseKind(s string) (dto.StrategyKind, error) {
      switch strings.ToLower(s) {
      case "dividend", "gold":
          return dto.StrategyKindDividend, nil
      case "growth":
          return dto.StrategyKindGrowth, nil
      default:
          return 0, fmt.Errorf("unknown --kind %q (expected Dividend|Growth)", s)
      }
  }

  func kindLabel(k dto.StrategyKind) string {
      if k == dto.StrategyKindGrowth {
          return "Growth"
      }
      return "Dividend"
  }

  func selectShareList(kind dto.StrategyKind, csv string) *collection.InstrumentCollection {
      base := shares.Dividend()
      if kind == dto.StrategyKindGrowth {
          base = shares.Growth()
      }
      if csv == "" {
          return base
      }
      want := map[string]bool{}
      for _, id := range strings.Split(csv, ",") {
          want[strings.TrimSpace(id)] = true
      }
      out := collection.NewCollection()
      for _, instr := range base.All() {
          if want[instr.ID] {
              out.Add(instr)
          }
      }
      return out
  }

  func anyCacheMiss(dir string, list *collection.InstrumentCollection) bool {
      for _, instr := range list.All() {
          if _, err := os.Stat(filepath.Join(dir, instr.ID+"_W.json")); os.IsNotExist(err) {
              return true
          }
      }
      return false
  }

  // grpcFetcher implements backtest.CandleFetcher using the existing Tinkoff
  // gRPC client. It pulls up to ~5 years of weekly candles in 300-candle
  // chunks (Tinkoff's per-call cap for weekly is 300).
  type grpcFetcher struct {
      client grpc.GrpcClient
      to     time.Time
  }

  func newGrpcFetcher(client grpc.GrpcClient, to time.Time) backtest.CandleFetcher {
      return &grpcFetcher{client: client, to: to}
  }

  func (g *grpcFetcher) Fetch(ctx context.Context, shareID string) ([]*model.CandleItemTechAnalyse, error) {
      // One call is enough at weekly interval: 300 candles × 7 days ≈ 5.7 years.
      limit := int32(300)
      from := utils.TimeStampPbGenerator(g.to, -300, enum.Week1)
      return g.client.MarketDataServiceClient().GetCandles(
          ctx, &shareID, enum.Week1.ToNumberInvestApi(),
          from, timestamppb.New(g.to), &limit, true,
      )
  }

  func filterClosed(candles []*model.CandleItemTechAnalyse, loc *time.Location) []*model.CandleItemTechAnalyse {
      now := time.Now().In(loc)
      // Reuse the same Monday-00:00 cutoff prod uses.
      weekday := (int(now.Weekday()) + 6) % 7
      y, m, d := now.Date()
      cutoff := time.Date(y, m, d-weekday, 0, 0, 0, 0, loc)
      out := make([]*model.CandleItemTechAnalyse, 0, len(candles))
      for _, c := range candles {
          if c != nil && c.Time.Before(cutoff) {
              out = append(out, c)
          }
      }
      return out
  }

  func trimByDateRange(candles []*model.CandleItemTechAnalyse, from, to time.Time) []*model.CandleItemTechAnalyse {
      out := make([]*model.CandleItemTechAnalyse, 0, len(candles))
      for _, c := range candles {
          if c.Time.Before(from) || c.Time.After(to) {
              continue
          }
          out = append(out, c)
      }
      return out
  }

  // chooseStartIdx leaves enough head-room for the adaptive RSI window plus
  // the EMA200 warmup the trend filter needs (≈200 closed weeks for Growth).
  func chooseStartIdx(rsiPeriod int) int {
      const adaptiveWindowMin = 100
      const emaWarmup = 200
      if emaWarmup > rsiPeriod+adaptiveWindowMin {
          return emaWarmup
      }
      return rsiPeriod + adaptiveWindowMin
  }

  func consoleSummary(r backtest.Report) string {
      var b strings.Builder
      b.WriteString("## Overall\n\n")
      // Inline format mirroring report.writeStatsTable.
      s := r.Overall
      fmt.Fprintf(&b, "Count=%d Wins=%d Losses=%d Open=%d WinRate=%.2f%% Cumulative=%.2f%% MaxDD=%.2f%% AvgWeeks=%.1f\n\n",
          s.Count, s.Wins, s.Losses, s.Open, s.WinRate*100, s.CumulativeReturn, s.MaxDrawdown, s.AvgWeeksHeld)
      b.WriteString("## Exit reasons\n\n")
      counts := map[backtest.ExitReason]int{}
      sums := map[backtest.ExitReason]float64{}
      for _, tr := range r.Trades {
          counts[tr.ExitReason]++
          sums[tr.ExitReason] += tr.ReturnPct
      }
      for reason, n := range counts {
          fmt.Fprintf(&b, "%-10s count=%d avg=%.2f%%\n", reason, n, sums[reason]/float64(n))
      }
      return b.String()
  }
  ```

- [ ] **Step 2: Build**

  Run: `go build ./cmd/backtest`
  Expected: PASS clean.

- [ ] **Step 3: Full build**

  Run: `go build ./... && go vet ./... && go test ./...`
  Expected: all clean.

- [ ] **Step 4: Commit**

  ```bash
  git add cmd/backtest/main.go
  git commit -m "feat(golden_x/backtest): add cmd/backtest binary"
  ```

---

## Task 2.8: End-to-end sanity run

This task is **manual** — requires a live `T_BANK` token. Engineer runs and confirms.

- [ ] **Step 1: First run (cache miss → API fetch)**

  Ensure `env/token.env` has `T_BANK=<token>`. Then run:
  ```bash
  go run ./cmd/backtest --kind=Dividend --from=2020-08-17
  ```
  (Full Dividend list, from the earliest weekly candle Tinkoff serves. A single ticker with a recent `--from` like `--from=2022-01-01 --shares=<SBER>` will yield `Count=0` — the 200-week EMA/RSI warmup window eats most of the `--from..--to` slice, leaving too few weeks for a p5-RSI dip to fire on any one share. Use the full list, or push `--from` back to ~2020 if you must restrict to one share.)

  Expected:
  - Console prints `## Overall` block with `Count` in the dozens (≈30 trades on the 11-share Dividend list over ~5.7 years is the empirical baseline).
  - Console prints `## Exit reasons` block.
  - Console prints `Report written to: cache/backtests/<ts>_Dividend.md`.
  - Cache files `cache/candles/<share-id>_W.json` now exist for every share in the list.

- [ ] **Step 2: Second run (cache hit, no API)**

  Temporarily invalidate the token so any API call would fail:
  ```bash
  T_BANK=bogus go run ./cmd/backtest --kind=Dividend --from=2020-08-17
  ```

  Expected: identical Overall section as run #1 (no API calls happened — verified by the bogus token not breaking anything). The binary should NOT print "T_BANK env var required" because `anyCacheMiss` returns false.

- [ ] **Step 3: Sanity bounds on report**

  Open the generated Markdown report. Confirm:
  - `Count` is in the dozens (single-digit number of trades over ~3 years is suspicious; thousands is wrong).
  - `MaxDD%` is reported as additive sum-of-return-pp drawdown (`equity += ReturnPct × Units`, no compounding) — values can exceed 100% when stop-loss streaks stack. Verify in `report.go:maxDrawdown`. Do NOT treat as equity-curve drawdown.
  - `Cumulative%` ≈ `Σ(ReturnPct × Units)` — eyeball-check by adding a few rows from the `## Trades` section.
  - Exit reasons are distributed (you should see a mix of `sell_p80`/`sell_p90`/`sell_p95`/`stop`/`timeout`/`open`).

  If any of these fails, debug before proceeding to commit.

---

## Task 2.9: Final commit

- [ ] **Step 1: Confirm clean state**

  Run: `git status`
  Expected: working tree clean (or only artifacts under `cache/` which `.gitignore` excludes).

- [ ] **Step 2: Update memory tracker**

  Edit `/home/oleg/.claude/projects/-home-oleg-GolandProjects-tinvest/memory/project_golden_x_stages.md` to move D0 and D3 from "Not yet started" to "Shipped on branch (not yet merged to main)". Add a sentence noting the new `internal/service/trading_strategy/golden_x/shares` package and the new `cmd/backtest` binary.

  This step does not require a commit — memory is outside the repo.

---

## Verification (end-to-end)

After completing all tasks, the following must hold simultaneously:

1. **Builds:** `go build ./... && go vet ./...` clean. Both binaries: `go build ./cmd && go build ./cmd/backtest`.
2. **Unit tests:** `go test ./...` passes. New tests cover detector, position, cache, replay, report.
3. **Existing regression tests unchanged:** all of `dedup_test`, `percentile_test`, `divergence_test`, `stop_test`, `trend_filter_test`, `rsi_test`, `candle_test`, `notification/notifications_test` pass with their original assertions.
4. **D0 dependency cleanup:** `git diff main -- go.mod` shows removal of `jmoiron/sqlx` and `lib/pq`. `find . -path ./vendor -prune -o -name '*.go' -print | xargs grep -l 'jmoiron\|lib/pq'` returns nothing.
5. **D0 runtime:** `go run ./cmd` boots in dev mode with `DB_*` env vars unset. No DB errors in logs.
6. **D3 cache behavior:** A second `go run ./cmd/backtest ...` after the first one issues zero gRPC calls (verified by bogus-token rerun).
7. **D3 report structure:** Generated Markdown contains `## Overall`, `## Exit reasons`, `## Per share`, `## Trades (chronological)` sections; numeric fields in plausible ranges.

---

## Critical files (touched or created)

**Phase 1 (deletes):**
- `internal/app/init_database.go`
- `pkg/client/db/db.go`, `pkg/client/db/pg/pg.go`, `pkg/client/db/pg/client.go`, `pkg/client/db/pg/tx_manager.go`
- `internal/config/storage.go`
- `migrations/*`, `migration.sh`, `migration.Dockerfile`
- `docker-compose.yml`

**Phase 1 (edits):**
- `internal/service_provider/client.go`
- `internal/config/config.go`
- `internal/app/app.go`
- `env/local.env`
- `go.mod`, `go.sum`

**Phase 2 (creates):**
- `internal/service/trading_strategy/golden_x/shares/shares.go`
- `internal/service/trading_strategy/golden_x/dto/signal.go`
- `internal/service/trading_strategy/golden_x/detector.go` + `detector_test.go`
- `internal/service/trading_strategy/golden_x/backtest/position.go` + `position_test.go`
- `internal/service/trading_strategy/golden_x/backtest/cache.go` + `cache_test.go`
- `internal/service/trading_strategy/golden_x/backtest/replay.go` + `replay_test.go`
- `internal/service/trading_strategy/golden_x/backtest/report.go` + `report_test.go`
- `cmd/backtest/main.go`
- `docs/superpowers/plans/2026-05-19-golden-x-stage-d0-d3.md` (this plan, committed)

**Phase 2 (edits):**
- `internal/service/trading_strategy/golden_x/trade.go` (Trade rewired to call Detect)
- `internal/service/trading_strategy/golden_x/trend_filter.go` (add `trendStatusFromClosed`)
- `internal/app/init_collection.go` (slim shim → delegate to `shares` package)
- `.gitignore` (add `/cache`)

---

## Reused existing utilities (do NOT re-implement)

- `pkg/indicators.ATR` (Wilder's ATR helper) — `pkg/indicators/atr.go`.
- `pkg/indicators.VolumeConfirmed` — `pkg/indicators/volume.go`.
- `internal/utils.CombinePrice`, `internal/utils.TimeStampPbGenerator` — `internal/utils/utils.go`.
- `pkg/collection.NewCollection` + `Instrument` + `InstrumentCollection.All()` — `pkg/collection/collection.go`.
- `pkg/client/grpc.NewClientGrpc` + `MarketDataServiceClient.GetCandles` — `pkg/client/grpc/grpc.go`, `pkg/client/grpc/market_data_service_client.go:141`.
- `golden_x.adaptiveRSIForShare`, `tierFromAdaptive`, `adaptiveSellThresholds`, `sellTierFromAdaptive`, `bullishDivergence`, `lowsAlignedToRSI`, `kForKind`, `stopFromATR`, `closedWeeklyCandles`, `trendStatusFromCandles`, `computeEMA`, `computeRSISeries` — all already in `internal/service/trading_strategy/golden_x/`.
- `dto.Stop`, `dto.SellThresholds`, `dto.Thresholds`, `dto.StrategyKind`, `dto.TrendStatus` — `internal/service/trading_strategy/golden_x/dto/`.
- Test helpers `mustLoad` and `candleAt` — `internal/service/trading_strategy/golden_x/candle_test.go`, `trend_filter_test.go`.
