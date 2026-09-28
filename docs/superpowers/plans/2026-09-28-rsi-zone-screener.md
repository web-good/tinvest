# RSI zone screener Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `cmd/zonescreen`, a ticker screener that replays the real `rsi_zone` engine over a fixed 24-configuration grid across the RUB share universe and ranks tickers by median profit factor under stressed costs, with a detailed per-ticker breakdown (half-year stability, exit profile, trend regime, price-tick cost).

**Architecture:** Same split as `cmd/pullscreen`: I/O in `cmd/zonescreen/main.go`, pure scoring in `internal/service/backtest/zone_screen.go` + `zone_screen_report.go` (same package as the pullback screener, so its helpers `profitFactor`, `splitTrades`, `medianF`, `clampPF`, `MeanDailyATRPct`, `ScreenOpts` are reused directly). The CLI plumbing shared by both screeners (universe loader, token loader, worker pacing, refresh cap) moves out of `cmd/pullscreen` into a new package `internal/service/backtest/screenrun`.

**Tech Stack:** Go 1.25, standard `testing`, existing backtest engine `internal/domain/backtest.Run`, strategy `internal/service/trading_strategy/rsi_zone/strategy/core`.

**Spec:** `docs/superpowers/specs/2026-09-28-rsi-zone-screener-design.md`

## Global Constraints

- Go 1.25; no new third-party dependencies.
- Grid: `RSIPeriod` {4, 6} × `RSILower` {15, 25} × `EMAPeriod` {50, 100, 200} × `RSIUpper` {65, 75}; `StopDailyATR` pinned 1.0, `DailyATRPeriod` pinned 14; 24 configurations, order `RSIPeriod → RSILower → EMAPeriod → RSIUpper`.
- Strategy is built with `core.NewWithParams(ticker, p)`, never via `RSIZoneLookupOrGeneric`.
- Stress model: `stressedPnL = PnL − Commission × (EntryPrice + ExitPrice) × Quantity − 2 × MinPriceIncrement × Quantity`.
- `Tick% = 2 × MinPriceIncrement / lastClose × 100`; unknown tick (`≤ 0`) or `lastClose ≤ 0` → `Tick% = 0`, `TickUnknown = true`, passes the tick gate.
- Gates: turnover ≥ `-min-turnover` (50), daily ATR ≥ `-min-atr-pct` (1.5), `Tick%` ≤ `-max-tick-pct` (0.3; 0 disables).
- Ranking key: `PFMedStress` descending, tiebreak ticker ascending.
- Half-years are calendar halves of ENTRY time in Europe/Moscow (Jan–Jun = H1, Jul–Dec = H2).
- Report path: `reports/zone_screen/zone_screen_Minutes30_<20060102_150405>.md`.
- `cmd/pullscreen` behaviour, flags and report format must not change.
- Docs under `docs/rsi_zone/` describe mechanics only: no per-ticker run results.
- Every commit message ends with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Build check: `go build ./internal/... ./pkg/... ./cmd/...` (`go build ./...` fails on `magefiles`). Full gate: `./bin/mage ci`.

## Review Focus

1. A REGISTERED ticker (AFKS, DOMRF, SBER) must be screened on the grid config, not its calibrated literal — pinned by `TestScreenZoneTickerRunsTheGridParamsVerbatim` (Task 3).
2. The API returns no price increment (`MinPriceIncrement = 0`): the row must pass the tick gate, render `?` in `Tick%`, and stress only by commission — pinned by `TestTickPctUnknown`, `TestStressTradesWithoutTick` (Task 2), `TestFilterAndRankZoneUnknownTickPasses`, `TestRenderZoneMarksUnknownTick` (Task 4).
3. A trade entered at 2026-06-30 21:30 UTC is 2026-07-01 00:30 MSK and belongs to 2026H2, not H1 — pinned by `TestHalfLabelUsesMoscowTime` (Task 2).
4. Every configuration silent on train: `HalvesPos`, `TopHalf`, `SLShare`, `HoldDays` are 0, `TopHalf` renders `—`, `Best` is still a real grid entry — pinned by `TestAggregateZoneAllSilentStillPicksRealBest` (Task 3), `TestRenderZoneDashesMissingTopHalf` (Task 4).
5. `-max-tick-pct 0` disables the tick gate instead of rejecting every ticker — pinned by `TestFilterAndRankZoneTickGateDisabledAtZero` (Task 4).

---

## File Structure

| File | Responsibility |
|---|---|
| Create `internal/service/backtest/screenrun/screenrun.go` | `ShareInfo`, `FilterShares`, `LoadUniverse`, `LoadToken`, `RefreshWorkerCap`, `EffectiveWorkers`, `PauseAfterTicker`, `SplitCSV`, `APIAddress`, `CacheDir` |
| Create `internal/service/backtest/screenrun/screenrun_test.go` | tests moved from `cmd/pullscreen/main_test.go` + new `FilterShares`/`SplitCSV` tests |
| Modify `cmd/pullscreen/main.go` | switch to `screenrun`, delete moved code |
| Delete `cmd/pullscreen/main_test.go` | tests moved |
| Create `internal/service/backtest/zone_screen.go` | grid, pure metrics, `ZoneRow`, `AggregateZone`, `ScreenZoneTicker` |
| Create `internal/service/backtest/zone_screen_test.go` | tests for the above |
| Modify `internal/service/backtest/pullback_screen_report.go` | extract `distributionOf([]float64)` from `Distribution` |
| Create `internal/service/backtest/zone_screen_report.go` | gates, ranking, distribution, Markdown |
| Create `internal/service/backtest/zone_screen_report_test.go` | tests for the report |
| Create `cmd/zonescreen/main.go`, `cmd/zonescreen/main_test.go` | CLI |
| Create `docs/rsi_zone/screener.md`; modify `docs/rsi_zone/strategy.md`, `docs/rsi_pullback/screener.md`, `CLAUDE.md` | docs |

---

### Task 1: Extract shared screener plumbing into `screenrun`

**Files:**
- Create: `internal/service/backtest/screenrun/screenrun.go`
- Create: `internal/service/backtest/screenrun/screenrun_test.go`
- Modify: `cmd/pullscreen/main.go` (lines 30–33 constants, 80–122 pause/shareInfo/refresh cap, 124–219 `run` call sites, 221–270 loaders)
- Delete: `cmd/pullscreen/main_test.go`

**Interfaces:**
- Consumes: `tinvest/internal/model.Share` (fields `Ticker`, `Name`, `ID`, `Lot`, `Currency`, `Trading`, `MinPriceIncrement`), `grpcclient.GrpcClient`.
- Produces (package `screenrun`):
  - `const APIAddress = "invest-public-api.tinkoff.ru:443"`, `const CacheDir = "data/candles"`, `const RefreshWorkerCap = 2`
  - `type ShareInfo struct { Ticker, Name, ID string; Lot int32; MinPriceIncrement float64 }`
  - `func FilterShares(shares []*model.Share, only []string) []ShareInfo`
  - `func LoadUniverse(ctx context.Context, client grpcclient.GrpcClient, only []string) ([]ShareInfo, error)`
  - `func LoadToken() (string, error)`
  - `func EffectiveWorkers(requested int, refresh bool) (workers int, capped bool)`
  - `func PauseAfterTicker(ctx context.Context, d time.Duration) bool`
  - `func SplitCSV(s string) []string`

- [ ] **Step 1: Write the failing tests**

Create `internal/service/backtest/screenrun/screenrun_test.go`:

```go
package screenrun

import (
	"context"
	"reflect"
	"testing"
	"time"

	"tinvest/internal/model"
)

func TestPauseAfterTickerSleepsForTheRequestedDelay(t *testing.T) {
	start := time.Now()
	if !PauseAfterTicker(context.Background(), 30*time.Millisecond) {
		t.Fatal("PauseAfterTicker returned false on a live context, want true")
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("slept %v, want at least 30ms — the whole point of -pause is to idle the CPU", elapsed)
	}
}

func TestPauseAfterTickerIsFreeWhenDisabled(t *testing.T) {
	// -pause 0 is the default: no timer, no allocation, no measurable delay.
	start := time.Now()
	if !PauseAfterTicker(context.Background(), 0) {
		t.Fatal("PauseAfterTicker returned false for a zero delay, want true")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Millisecond {
		t.Fatalf("slept %v on a zero delay, want an immediate return", elapsed)
	}
}

func TestPauseAfterTickerAbandonsTheSleepOnCancel(t *testing.T) {
	// Ctrl+C during a paced overnight run must not wait out the pause on every
	// in-flight worker before the pool unwinds.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if PauseAfterTicker(ctx, time.Hour) {
		t.Fatal("PauseAfterTicker returned true on a cancelled context, want false")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("took %v to notice cancellation, want an immediate return", elapsed)
	}
}

func TestEffectiveWorkersCapsOnRefresh(t *testing.T) {
	// 8 workers against the market-data API is roughly 26 req/s, well above what the
	// candle provider's cache writer can survive: Load(refresh=true) logs a failed
	// chunk and keeps going, then writes whatever it got over the existing cache file —
	// silently punching holes in the 540MB local cache the whole screener depends on.
	// -refresh must therefore run gentle regardless of what -workers asked for.
	got, capped := EffectiveWorkers(8, true)
	if got != 2 || !capped {
		t.Fatalf("EffectiveWorkers(8, true) = %d,%v, want 2,true", got, capped)
	}
}

func TestEffectiveWorkersLeavesLowRequestAlone(t *testing.T) {
	got, capped := EffectiveWorkers(1, true)
	if got != 1 || capped {
		t.Fatalf("EffectiveWorkers(1, true) = %d,%v, want 1,false — already at or below the refresh cap", got, capped)
	}
}

func TestEffectiveWorkersIgnoresCapWithoutRefresh(t *testing.T) {
	got, capped := EffectiveWorkers(8, false)
	if got != 8 || capped {
		t.Fatalf("EffectiveWorkers(8, false) = %d,%v, want 8,false — the cache-corruption risk is specific to -refresh", got, capped)
	}
}

func TestFilterSharesKeepsTradableRubAndCarriesTick(t *testing.T) {
	shares := []*model.Share{
		{Ticker: "SBER", Name: "Сбербанк", ID: "id-sber", Lot: 10, Currency: "rub", Trading: true, MinPriceIncrement: 0.01},
		{Ticker: "USDX", Name: "Dollar", ID: "id-usd", Lot: 1, Currency: "usd", Trading: true, MinPriceIncrement: 0.01},
		{Ticker: "DEAD", Name: "Halted", ID: "id-dead", Lot: 1, Currency: "RUB", Trading: false, MinPriceIncrement: 0.1},
		nil,
	}
	got := FilterShares(shares, nil)
	want := []ShareInfo{{Ticker: "SBER", Name: "Сбербанк", ID: "id-sber", Lot: 10, MinPriceIncrement: 0.01}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FilterShares = %+v, want %+v", got, want)
	}
}

func TestFilterSharesExplicitListIgnoresCurrencyAndCase(t *testing.T) {
	// -tickers is a diagnostics escape hatch: it must reach a ticker the universe
	// filter would drop, matched case-insensitively.
	shares := []*model.Share{
		{Ticker: "SBER", ID: "id-sber", Currency: "rub", Trading: true},
		{Ticker: "DEAD", ID: "id-dead", Currency: "rub", Trading: false, MinPriceIncrement: 0.1},
	}
	got := FilterShares(shares, []string{"dead"})
	if len(got) != 1 || got[0].Ticker != "DEAD" || got[0].MinPriceIncrement != 0.1 {
		t.Fatalf("FilterShares(only=dead) = %+v, want only DEAD with its tick", got)
	}
}

func TestSplitCSV(t *testing.T) {
	if got := SplitCSV("  "); got != nil {
		t.Fatalf("SplitCSV(blank) = %v, want nil", got)
	}
	got := SplitCSV(" SBER, ,AFKS ,")
	if !reflect.DeepEqual(got, []string{"SBER", "AFKS"}) {
		t.Fatalf("SplitCSV = %v, want [SBER AFKS]", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/screenrun/`
Expected: FAIL — build error, `PauseAfterTicker`, `EffectiveWorkers`, `FilterShares`, `SplitCSV` undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/service/backtest/screenrun/screenrun.go`:

```go
// Package screenrun holds the I/O plumbing shared by the ticker screeners
// (cmd/pullscreen, cmd/zonescreen): the tradable-universe loader, the API token loader
// and the worker-pool pacing rules. The scoring itself lives in
// internal/service/backtest and stays pure.
package screenrun

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"tinvest/internal/model"
	grpcclient "tinvest/pkg/client/grpc"
)

const (
	// APIAddress is the Tinkoff Invest public API endpoint.
	APIAddress = "invest-public-api.tinkoff.ru:443"
	// CacheDir is the local candle cache every screener replays from.
	CacheDir = "data/candles"
)

// RefreshWorkerCap is the most concurrent tickers -refresh may run with. The candle
// provider (internal/service/backtest/candles.go) logs a failed chunk and keeps going on
// error, and Load(refresh=true) then writes whatever it managed to fetch back over the
// existing cache file — so a slow-down from rate limiting is not the risk, a partially
// overwritten 540MB local cache is. 8 workers sit around 26 req/s against market-data,
// well above what the API tolerates before chunks start failing.
const RefreshWorkerCap = 2

// EffectiveWorkers returns the worker count a screener should actually use, and whether
// it clamped the caller's request. Pure so the -refresh safety rule is unit-testable
// without standing up a gRPC client.
func EffectiveWorkers(requested int, refresh bool) (workers int, capped bool) {
	if refresh && requested > RefreshWorkerCap {
		return RefreshWorkerCap, true
	}
	return requested, false
}

// PauseAfterTicker idles a worker between tickers so a full-universe run does not
// pin every core for 20+ minutes. It reports whether the pause completed: a
// cancelled context abandons the sleep immediately, so Ctrl+C unwinds the pool at
// once instead of once per outstanding pause.
func PauseAfterTicker(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// ShareInfo is the per-ticker metadata a screener's worker pool needs.
type ShareInfo struct {
	Ticker            string
	Name              string
	ID                string
	Lot               int32
	MinPriceIncrement float64 // price tick in currency units; 0 when the API did not report one
}

// FilterShares keeps the tradable RUB shares, or — when only is non-empty — exactly the
// listed tickers (case-insensitive, regardless of currency or trading status).
func FilterShares(shares []*model.Share, only []string) []ShareInfo {
	want := make(map[string]bool, len(only))
	for _, t := range only {
		want[strings.ToUpper(t)] = true
	}
	var out []ShareInfo
	for _, s := range shares {
		if s == nil {
			continue
		}
		if len(want) > 0 {
			if !want[strings.ToUpper(s.Ticker)] {
				continue
			}
		} else if !strings.EqualFold(s.Currency, "rub") || !s.Trading {
			continue
		}
		out = append(out, ShareInfo{
			Ticker: s.Ticker, Name: s.Name, ID: s.ID, Lot: s.Lot, MinPriceIncrement: s.MinPriceIncrement,
		})
	}
	return out
}

// LoadUniverse returns the tradable RUB share universe, or just the requested tickers.
func LoadUniverse(ctx context.Context, client grpcclient.GrpcClient, only []string) ([]ShareInfo, error) {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return nil, fmt.Errorf("load shares: %w", err)
	}
	universe := FilterShares(shares, only)
	if len(universe) == 0 {
		return nil, fmt.Errorf("no matching shares found")
	}
	return universe, nil
}

// SplitCSV splits a comma-separated flag value, dropping blanks.
func SplitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// LoadToken reads the T_BANK API token from the environment or the local env files.
func LoadToken() (string, error) {
	_ = godotenv.Load("./env/local.env")
	_ = godotenv.Load("./env/token.env")
	token := os.Getenv("T_BANK")
	if token == "" {
		return "", fmt.Errorf("T_BANK is not set (checked env + ./env/local.env, ./env/token.env)")
	}
	return token, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/screenrun/ -v`
Expected: PASS, 9 tests.

- [ ] **Step 5: Switch `cmd/pullscreen` to `screenrun`**

In `cmd/pullscreen/main.go`:
1. Delete the `const ( apiAddress …; cacheDir … )` block, `pauseAfterTicker`, `shareInfo`, `refreshWorkerCap`, `effectiveWorkers`, `loadUniverse`, `splitCSV`, `loadToken`.
2. Remove now-unused imports `strings`, `github.com/joho/godotenv`; add `"tinvest/internal/service/backtest/screenrun"`.
3. Replace call sites exactly:
   - `splitCSV(*tickersCSV)` → `screenrun.SplitCSV(*tickersCSV)`
   - `effectiveWorkers(cfg.workers, cfg.refresh)` → `screenrun.EffectiveWorkers(cfg.workers, cfg.refresh)`
   - the printf text `(see refreshWorkerCap)` → `(see screenrun.RefreshWorkerCap)`
   - `loadToken()` → `screenrun.LoadToken()`
   - `grpcclient.NewClientGrpc(apiAddress, token)` → `grpcclient.NewClientGrpc(screenrun.APIAddress, token)`
   - `loadUniverse(ctx, client, cfg.tickers)` → `screenrun.LoadUniverse(ctx, client, cfg.tickers)`
   - `svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)` → `svc.NewCandleProvider(client.MarketDataServiceClient(), screenrun.CacheDir)`
   - `go func(u shareInfo)` → `go func(u screenrun.ShareInfo)`
   - `pauseAfterTicker(ctx, cfg.pause)` → `screenrun.PauseAfterTicker(ctx, cfg.pause)`

Delete `cmd/pullscreen/main_test.go` (its tests now live in `screenrun_test.go`).

Update the doc path in `docs/rsi_pullback/screener.md`: in the `-refresh` paragraph replace `` (см. `refreshWorkerCap` в `cmd/pullscreen/main.go`) `` with `` (см. `RefreshWorkerCap` в `internal/service/backtest/screenrun/screenrun.go`) ``, and in the «Если прогон греет машину» section replace `` Защита `refreshWorkerCap` `` with `` Защита `screenrun.RefreshWorkerCap` ``.

- [ ] **Step 6: Build and run the affected tests**

Run: `go build ./cmd/pullscreen/ && go vet ./cmd/pullscreen/ ./internal/service/backtest/screenrun/ && go test ./internal/service/backtest/screenrun/`
Expected: no output from build/vet, tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/service/backtest/screenrun cmd/pullscreen docs/rsi_pullback/screener.md
git commit -m "refactor(pullscreen): выносим обвязку скринеров в пакет screenrun

Universe/token loaders, worker pacing and the -refresh worker cap move to
internal/service/backtest/screenrun so a second screener reuses them instead
of copying the cache-safety rule. ShareInfo now carries MinPriceIncrement.
pullscreen behaviour is unchanged.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Zone grid and pure per-trade metrics

**Files:**
- Create: `internal/service/backtest/zone_screen.go`
- Create: `internal/service/backtest/zone_screen_test.go`

**Interfaces:**
- Consumes: `screenMSK` (`pullback_screen.go`), `medianF`, `backtest.Trade`, `backtest.Candle`, `ema.Compute(closes []float64, period int) []float64` (zero-filled before `period-1`).
- Produces (package `backtest`, service layer):
  - `func ZoneGrid() []zonecore.Params` (import alias `zonecore "tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"`)
  - `func stressTrades(trades []backtest.Trade, commission, tick float64) []backtest.Trade`
  - `func tickPct(tick, lastClose float64) (pct float64, unknown bool)`
  - `type HalfResult struct { Label string; PnL float64; Trades int }`
  - `func halfLabel(t time.Time) string` → e.g. `"2026H1"`
  - `func halfYearResults(trades []backtest.Trade) []HalfResult` (sorted by label)
  - `func halvesStats(halves []HalfResult) (posShare, topShare float64, hasProfit bool)`
  - `func slShare(trades []backtest.Trade) float64`
  - `func medianHoldDays(trades []backtest.Trade) float64`
  - `func aboveEMAShare(bars []backtest.Candle, period int) float64`

- [ ] **Step 1: Write the failing tests**

Create `internal/service/backtest/zone_screen_test.go`:

```go
package backtest

import (
	"math"
	"reflect"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	zonecore "tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

func TestZoneGridHas24UniqueConfigsInOrder(t *testing.T) {
	grid := ZoneGrid()
	if len(grid) != 24 {
		t.Fatalf("grid size = %d, want 24 (2 RSIPeriod x 2 RSILower x 3 EMAPeriod x 2 RSIUpper)", len(grid))
	}
	seen := make(map[zonecore.Params]bool, len(grid))
	for _, p := range grid {
		if seen[p] {
			t.Fatalf("duplicate config in grid: %+v", p)
		}
		seen[p] = true
	}
	// Deterministic order: RSIPeriod -> RSILower -> EMAPeriod -> RSIUpper.
	first := zonecore.Params{RSIPeriod: 4, RSILower: 15, RSIUpper: 65, EMAPeriod: 50, DailyATRPeriod: 14, StopDailyATR: 1.0}
	second := zonecore.Params{RSIPeriod: 4, RSILower: 15, RSIUpper: 75, EMAPeriod: 50, DailyATRPeriod: 14, StopDailyATR: 1.0}
	last := zonecore.Params{RSIPeriod: 6, RSILower: 25, RSIUpper: 75, EMAPeriod: 200, DailyATRPeriod: 14, StopDailyATR: 1.0}
	if grid[0] != first || grid[1] != second || grid[23] != last {
		t.Fatalf("grid order = %+v, %+v ... %+v", grid[0], grid[1], grid[23])
	}
}

func TestZoneGridSweepsOnlyTheFourAxes(t *testing.T) {
	periods, lowers, emas, uppers := map[int]bool{}, map[float64]bool{}, map[int]bool{}, map[float64]bool{}
	for _, p := range ZoneGrid() {
		periods[p.RSIPeriod], lowers[p.RSILower], emas[p.EMAPeriod], uppers[p.RSIUpper] = true, true, true, true
		// The stop is a property of a tuned point, not of the instrument.
		if p.StopDailyATR != 1.0 || p.DailyATRPeriod != 14 {
			t.Fatalf("stop/ATR = %v/%d, want pinned 1.0/14", p.StopDailyATR, p.DailyATRPeriod)
		}
	}
	if !reflect.DeepEqual(emas, map[int]bool{50: true, 100: true, 200: true}) {
		t.Fatalf("EMA axis = %v, want {50,100,200} — EMA 50 is the lead lever on DOMRF and AFKS", emas)
	}
	if len(periods) != 2 || len(lowers) != 2 || len(uppers) != 2 {
		t.Fatalf("axes = %d/%d/%d, want 2/2/2", len(periods), len(lowers), len(uppers))
	}
}

func TestStressTradesAddsCommissionAndTwoTicks(t *testing.T) {
	trades := []backtest.Trade{{EntryPrice: 100, ExitPrice: 110, Quantity: 10, PnL: 99}}
	got := stressTrades(trades, 0.0005, 0.1)
	// extra = 0.0005*(100+110)*10 + 2*0.1*10 = 1.05 + 2 = 3.05
	if want := 99 - 3.05; math.Abs(got[0].PnL-want) > 1e-9 {
		t.Fatalf("stressed PnL = %v, want %v", got[0].PnL, want)
	}
	if trades[0].PnL != 99 {
		t.Fatalf("input mutated: PnL = %v, want 99", trades[0].PnL)
	}
}

func TestStressTradesWithoutTick(t *testing.T) {
	// Unknown tick (0) or a nonsensical negative one stresses by commission only.
	for _, tick := range []float64{0, -1} {
		got := stressTrades([]backtest.Trade{{EntryPrice: 100, ExitPrice: 100, Quantity: 1, PnL: 0}}, 0.001, tick)
		if math.Abs(got[0].PnL-(-0.2)) > 1e-9 {
			t.Fatalf("tick %v: stressed PnL = %v, want -0.2", tick, got[0].PnL)
		}
	}
	if got := stressTrades(nil, 0.001, 0.1); got != nil {
		t.Fatalf("stressTrades(nil) = %v, want nil", got)
	}
}

func TestTickPct(t *testing.T) {
	pct, unknown := tickPct(0.1, 50) // KMAZ-shaped: two ticks = 0.4% of price
	if unknown || math.Abs(pct-0.4) > 1e-9 {
		t.Fatalf("tickPct(0.1, 50) = %v,%v, want 0.4,false", pct, unknown)
	}
}

func TestTickPctUnknown(t *testing.T) {
	for _, c := range []struct{ tick, close float64 }{{0, 50}, {0.1, 0}, {-0.1, 50}} {
		pct, unknown := tickPct(c.tick, c.close)
		if pct != 0 || !unknown {
			t.Fatalf("tickPct(%v, %v) = %v,%v, want 0,true", c.tick, c.close, pct, unknown)
		}
	}
}

func TestHalfLabelUsesMoscowTime(t *testing.T) {
	// 21:30 UTC on June 30 is 00:30 MSK on July 1: second half, not first.
	if got := halfLabel(time.Date(2026, 6, 30, 21, 30, 0, 0, time.UTC)); got != "2026H2" {
		t.Fatalf("halfLabel = %q, want 2026H2", got)
	}
	if got := halfLabel(time.Date(2026, 6, 30, 20, 30, 0, 0, time.UTC)); got != "2026H1" {
		t.Fatalf("halfLabel = %q, want 2026H1", got)
	}
	if got := halfLabel(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)); got != "2025H1" {
		t.Fatalf("halfLabel = %q, want 2025H1", got)
	}
}

func TestHalfYearResultsGroupsAndSorts(t *testing.T) {
	trades := []backtest.Trade{
		{EntryTime: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC), PnL: 5},
		{EntryTime: time.Date(2025, 8, 1, 10, 0, 0, 0, time.UTC), PnL: -2},
		{EntryTime: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), PnL: 1},
	}
	got := halfYearResults(trades)
	want := []HalfResult{{Label: "2025H2", PnL: -2, Trades: 1}, {Label: "2026H1", PnL: 6, Trades: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("halfYearResults = %+v, want %+v", got, want)
	}
}

func TestHalvesStats(t *testing.T) {
	pos, top, ok := halvesStats([]HalfResult{{PnL: 3}, {PnL: -1}, {PnL: 1}, {PnL: 0}})
	if !ok || pos != 0.5 || top != 0.75 {
		t.Fatalf("halvesStats = %v,%v,%v, want 0.5,0.75,true", pos, top, ok)
	}
	pos, top, ok = halvesStats([]HalfResult{{PnL: -3}, {PnL: 0}})
	if ok || pos != 0 || top != 0 {
		t.Fatalf("no profitable half: halvesStats = %v,%v,%v, want 0,0,false", pos, top, ok)
	}
	if pos, top, ok = halvesStats(nil); ok || pos != 0 || top != 0 {
		t.Fatalf("empty: halvesStats = %v,%v,%v, want 0,0,false", pos, top, ok)
	}
}

func TestSLShareAndHoldDays(t *testing.T) {
	base := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	trades := []backtest.Trade{
		{Reason: "SL", EntryTime: base, ExitTime: base.Add(24 * time.Hour)},
		{Reason: "RSI", EntryTime: base, ExitTime: base.Add(72 * time.Hour)},
		{Reason: "RSI", EntryTime: base, ExitTime: base.Add(48 * time.Hour)},
		{Reason: "SL", EntryTime: base, ExitTime: base.Add(12 * time.Hour)},
	}
	if got := slShare(trades); got != 0.5 {
		t.Fatalf("slShare = %v, want 0.5", got)
	}
	// days: 1, 3, 2, 0.5 -> median 1.5
	if got := medianHoldDays(trades); got != 1.5 {
		t.Fatalf("medianHoldDays = %v, want 1.5", got)
	}
	if slShare(nil) != 0 || medianHoldDays(nil) != 0 {
		t.Fatal("empty trades must give 0 for both metrics")
	}
}

func TestAboveEMAShareSkipsWarmup(t *testing.T) {
	// Period 3 on closes 1..6: EMA is zero on bars 0 and 1 (warm-up) and excluded.
	// Warmed EMA = 2, 3, 4, 5 on bars 2..5 against closes 3..6: every warmed bar is
	// above its EMA, so 4/4 — and the two warm-up bars must not dilute that to 4/6.
	bars := make([]backtest.Candle, 6)
	for i := range bars {
		bars[i].Close = float64(i + 1)
	}
	if got := aboveEMAShare(bars, 3); got != 1 {
		t.Fatalf("aboveEMAShare(rising) = %v, want 1", got)
	}
	// Falling series: every warmed bar is below its EMA.
	for i := range bars {
		bars[i].Close = float64(10 - i)
	}
	if got := aboveEMAShare(bars, 3); got != 0 {
		t.Fatalf("aboveEMAShare(falling) = %v, want 0", got)
	}
	if aboveEMAShare(bars[:2], 3) != 0 || aboveEMAShare(bars, 0) != 0 {
		t.Fatal("too-short series or non-positive period must give 0")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TestZoneGrid|TestStressTrades|TestTickPct|TestHalf|TestSLShare|TestAboveEMA'`
Expected: FAIL — build error, `ZoneGrid` etc. undefined.

- [ ] **Step 3: Write the implementation**

Create `internal/service/backtest/zone_screen.go`:

```go
package backtest

import (
	"fmt"
	"sort"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/domain/ema"
	zonecore "tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// Pinned zone-screener axes. Like the pullback screener, it answers "is this ticker
// worth calibrating", so it sweeps only the parameters whose useful range is
// instrument-specific. EMA 50 is on the axis because it was the lead lever on both
// DOMRF and AFKS. See docs/superpowers/specs/2026-09-28-rsi-zone-screener-design.md, §3.
var (
	zoneGridRSIPeriods = []int{4, 6}
	zoneGridRSILowers  = []float64{15, 25}
	zoneGridEMAPeriods = []int{50, 100, 200}
	zoneGridRSIUppers  = []float64{65, 75}
)

// ZoneGrid returns the 24 configurations every ticker is screened on, in a
// deterministic order. The stop is pinned at 1.0 daily ATR: it is a property of a
// tuned point, not of the instrument, and 1.0 was accepted on every calibrated ticker.
func ZoneGrid() []zonecore.Params {
	out := make([]zonecore.Params, 0,
		len(zoneGridRSIPeriods)*len(zoneGridRSILowers)*len(zoneGridEMAPeriods)*len(zoneGridRSIUppers))
	for _, period := range zoneGridRSIPeriods {
		for _, lower := range zoneGridRSILowers {
			for _, emaPeriod := range zoneGridEMAPeriods {
				for _, upper := range zoneGridRSIUppers {
					out = append(out, zonecore.Params{
						RSIPeriod:      period,
						RSILower:       lower,
						RSIUpper:       upper,
						EMAPeriod:      emaPeriod,
						DailyATRPeriod: 14,
						StopDailyATR:   1.0,
					})
				}
			}
		}
	}
	return out
}

// stressTrades returns copies of trades with the round trip made more expensive: one
// more commission on each side plus two price ticks (entry rounds up, exit rounds down).
// Recomputing from the recorded trades instead of rerunning the engine ignores the tiny
// sizing drift a poorer cash curve would cause; on a profit-factor ratio it is negligible.
// A non-positive tick (the API did not report one) stresses by commission only.
func stressTrades(trades []backtest.Trade, commission, tick float64) []backtest.Trade {
	if len(trades) == 0 {
		return nil
	}
	tick = max(tick, 0)
	out := make([]backtest.Trade, len(trades))
	for i, t := range trades {
		q := float64(t.Quantity)
		t.PnL -= commission*(t.EntryPrice+t.ExitPrice)*q + 2*tick*q
		out[i] = t
	}
	return out
}

// tickPct is the share of two price ticks in the current price, in percent. unknown is
// true when either input is non-positive: the gate lets such a row through and the
// report flags it rather than inventing a cost.
func tickPct(tick, lastClose float64) (pct float64, unknown bool) {
	if tick <= 0 || lastClose <= 0 {
		return 0, true
	}
	return 2 * tick / lastClose * 100, false
}

// HalfResult is the summed raw PnL of the trades entered in one calendar half-year.
type HalfResult struct {
	Label  string // "2026H1" (January–June) or "2026H2" (July–December), Moscow time
	PnL    float64
	Trades int
}

// halfLabel names the calendar half-year of t in Moscow time.
func halfLabel(t time.Time) string {
	tl := t.In(screenMSK)
	half := 1
	if tl.Month() >= time.July {
		half = 2
	}
	return fmt.Sprintf("%dH%d", tl.Year(), half)
}

// halfYearResults groups trades by the half-year of their ENTRY and sorts by label
// (four-digit years make the lexical order chronological).
func halfYearResults(trades []backtest.Trade) []HalfResult {
	idx := make(map[string]int)
	var out []HalfResult
	for _, t := range trades {
		label := halfLabel(t.EntryTime)
		i, ok := idx[label]
		if !ok {
			i = len(out)
			idx[label] = i
			out = append(out, HalfResult{Label: label})
		}
		out[i].PnL += t.PnL
		out[i].Trades++
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// halvesStats returns the share of profitable halves among the given ones and the share
// of the best half in the sum of profitable halves (1.0 = all profit from one half).
// hasProfit is false when no half made money; topShare is then 0 and meaningless.
func halvesStats(halves []HalfResult) (posShare, topShare float64, hasProfit bool) {
	if len(halves) == 0 {
		return 0, 0, false
	}
	var pos int
	var sum, best float64
	for _, h := range halves {
		if h.PnL <= 0 {
			continue
		}
		pos++
		sum += h.PnL
		best = max(best, h.PnL)
	}
	posShare = float64(pos) / float64(len(halves))
	if sum <= 0 {
		return posShare, 0, false
	}
	return posShare, best / sum, true
}

// slShare is the fraction of trades closed by the protective stop (exit reason "SL").
func slShare(trades []backtest.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	var n int
	for _, t := range trades {
		if t.Reason == "SL" {
			n++
		}
	}
	return float64(n) / float64(len(trades))
}

// medianHoldDays is the median trade duration in calendar days. Measured by wall clock,
// not BarsHeld: evening sessions and weekends make the bar count per day uneven.
func medianHoldDays(trades []backtest.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	days := make([]float64, 0, len(trades))
	for _, t := range trades {
		days = append(days, t.ExitTime.Sub(t.EntryTime).Hours()/24)
	}
	return medianF(days)
}

// aboveEMAShare is the fraction of bars whose close sits strictly above EMA(period),
// computed with the same ema.Compute the strategy core uses. Warm-up bars (EMA == 0)
// are left out of the denominator.
func aboveEMAShare(bars []backtest.Candle, period int) float64 {
	if period <= 0 || len(bars) == 0 {
		return 0
	}
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}
	line := ema.Compute(closes, period)
	var above, n int
	for i, c := range closes {
		if line[i] <= 0 {
			continue
		}
		n++
		if c > line[i] {
			above++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(above) / float64(n)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestZoneGrid|TestStressTrades|TestTickPct|TestHalf|TestSLShare|TestAboveEMA' -v`
Expected: PASS, 11 tests.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/zone_screen.go internal/service/backtest/zone_screen_test.go
git commit -m "feat(zonescreen): сетка и чистые метрики сделок — стресс издержек, полугодия, выходы, тренд

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Ticker aggregation and engine replay

**Files:**
- Modify: `internal/service/backtest/zone_screen.go` (append)
- Modify: `internal/service/backtest/zone_screen_test.go` (append)

**Interfaces:**
- Consumes (Task 2): `ZoneGrid`, `stressTrades`, `tickPct`, `HalfResult`, `halfYearResults`, `halvesStats`, `slShare`, `medianHoldDays`, `aboveEMAShare`. From `pullback_screen.go`: `profitFactor`, `splitTrades`, `medianF`, `clampPF`, `ScreenOpts`, `DefaultScreenOpts`, `MeanDailyATRPct`, `screenDailyATRPeriod`, `screenMSK`. From test files: `dailyCandlesMSK(n int, closePrice, rangePct float64)`, `tinyCandles(n int)`.
- Produces:
  - `type ZoneConfigResult struct { Params zonecore.Params; Trades []backtest.Trade }`
  - `type ZoneRow struct { Ticker, Name string; TurnoverM, DailyATRPct float64; Bars int; TickPct float64; TickUnknown bool; PFMed, PFMedStress, TradesMed, Plateau float64; Capped, SilentCfg int; PFMedHO, PFMedStressHO, TradesMedHO float64; HalvesPos, TopHalf, SLShare, HoldDays float64; AboveEMA50, AboveEMA200 float64; Best zonecore.Params; BestPF float64; BestHalves []HalfResult; NoSignals bool }`
  - `func AggregateZone(ticker, name string, results []ZoneConfigResult, split time.Time, tick float64, opts ScreenOpts) ZoneRow`
  - `type ZoneTickerInput struct { Ticker, Name string; Bars, Daily []backtest.Candle; Lot int32; MinPriceIncrement float64 }`
  - `func ScreenZoneTicker(in ZoneTickerInput, cfgs []zonecore.Params, split time.Time, opts ScreenOpts) ZoneRow`

- [ ] **Step 1: Write the failing tests**

Append to `internal/service/backtest/zone_screen_test.go`:

```go
// zoneTrade is a one-lot trade entered at `entry` with the given PnL and exit reason.
func zoneTrade(entry time.Time, pnl float64, reason string) backtest.Trade {
	return backtest.Trade{
		EntryTime: entry, ExitTime: entry.Add(48 * time.Hour),
		EntryPrice: 100, ExitPrice: 100 + pnl, Quantity: 1, PnL: pnl, Reason: reason,
	}
}

func TestAggregateZoneRanksByStressedMedian(t *testing.T) {
	// Two configurations with the same raw PF 2.0: the stress (commission 0.01 per
	// side, tick 1) knocks ~4 off every 1-share trade at price ~100, turning +4/-2 into
	// -0.04/-5.98 -> no gross profit, stressed PF 0. PFMed stays 2.
	d := time.Date(2025, 3, 3, 10, 0, 0, 0, time.UTC)
	trades := []backtest.Trade{zoneTrade(d, 4, "RSI"), zoneTrade(d.AddDate(0, 0, 7), -2, "SL")}
	grid := ZoneGrid()[:2]
	results := []ZoneConfigResult{{Params: grid[0], Trades: trades}, {Params: grid[1], Trades: trades}}
	opts := DefaultScreenOpts()
	opts.Commission = 0.01

	row := AggregateZone("XXXX", "Test", results, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 1, opts)

	if row.PFMed != 2 {
		t.Fatalf("PFMed = %v, want 2", row.PFMed)
	}
	if row.PFMedStress != 0 {
		t.Fatalf("PFMedStress = %v, want 0 (stress wipes the edge)", row.PFMedStress)
	}
}

func TestAggregateZoneDetailMetrics(t *testing.T) {
	// Config A trades in 2025H1 (+3, +1) and 2025H2 (-1); config B is silent.
	d1 := time.Date(2025, 2, 3, 10, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 9, 1, 10, 0, 0, 0, time.UTC)
	grid := ZoneGrid()
	results := []ZoneConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{zoneTrade(d1, 3, "RSI"), zoneTrade(d1.AddDate(0, 0, 5), 1, "RSI"), zoneTrade(d2, -1, "SL")}},
		{Params: grid[1]},
	}
	row := AggregateZone("XXXX", "Test", results, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 0, DefaultScreenOpts())

	if row.SilentCfg != 1 {
		t.Fatalf("SilentCfg = %d, want 1", row.SilentCfg)
	}
	// Silent configs stay out of the detail medians: they have no halves to judge.
	if row.HalvesPos != 0.5 {
		t.Fatalf("HalvesPos = %v, want 0.5 (one of two halves profitable)", row.HalvesPos)
	}
	if row.TopHalf != 1 {
		t.Fatalf("TopHalf = %v, want 1 (all profit from 2025H1)", row.TopHalf)
	}
	if math.Abs(row.SLShare-1.0/3) > 1e-9 {
		t.Fatalf("SLShare = %v, want 1/3", row.SLShare)
	}
	if row.HoldDays != 2 {
		t.Fatalf("HoldDays = %v, want 2", row.HoldDays)
	}
	if row.Best != grid[0] {
		t.Fatalf("Best = %+v, want grid[0]", row.Best)
	}
	want := []HalfResult{{Label: "2025H1", PnL: 4, Trades: 2}, {Label: "2025H2", PnL: -1, Trades: 1}}
	if !reflect.DeepEqual(row.BestHalves, want) {
		t.Fatalf("BestHalves = %+v, want %+v", row.BestHalves, want)
	}
}

func TestAggregateZoneAllSilentStillPicksRealBest(t *testing.T) {
	grid := ZoneGrid()
	results := []ZoneConfigResult{{Params: grid[0]}, {Params: grid[1]}}
	row := AggregateZone("XXXX", "Test", results, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 0.1, DefaultScreenOpts())

	if !row.NoSignals || row.SilentCfg != 2 {
		t.Fatalf("NoSignals/SilentCfg = %v/%d, want true/2", row.NoSignals, row.SilentCfg)
	}
	if row.HalvesPos != 0 || row.TopHalf != 0 || row.SLShare != 0 || row.HoldDays != 0 {
		t.Fatalf("detail metrics = %v/%v/%v/%v, want all 0", row.HalvesPos, row.TopHalf, row.SLShare, row.HoldDays)
	}
	if row.Best != grid[0] {
		t.Fatalf("Best = %+v, want grid[0] — a zero Params would render as a fake configuration", row.Best)
	}
}

func TestAggregateZoneHoldoutOnlyTradesAreSignals(t *testing.T) {
	split := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	grid := ZoneGrid()
	results := []ZoneConfigResult{{Params: grid[0], Trades: []backtest.Trade{
		zoneTrade(split.AddDate(0, 1, 0), 2, "RSI"), zoneTrade(split.AddDate(0, 2, 0), -1, "SL"),
	}}}
	row := AggregateZone("XXXX", "Test", results, split, 0, DefaultScreenOpts())

	if row.NoSignals {
		t.Fatal("NoSignals = true, want false: the holdout traded")
	}
	if row.PFMedHO != 2 || row.TradesMedHO != 2 {
		t.Fatalf("holdout PF/trades = %v/%v, want 2/2", row.PFMedHO, row.TradesMedHO)
	}
	if row.PFMedStressHO >= row.PFMedHO {
		t.Fatalf("PFMedStressHO = %v, want below raw holdout PF %v", row.PFMedStressHO, row.PFMedHO)
	}
}

// zoneFixtureCandles builds n weekday MSK 30-minute bars the rsi_zone grid trades: a
// +0.3%/bar uptrend (close stays above EMA 50) broken every 60 bars by three -1% bars,
// which drive RSI(4) from ~100 to ~18 — a cross below 25 — before the trend resumes and
// RSI crosses back above 65/75.
func zoneFixtureCandles(n int) []backtest.Candle {
	const (
		driftPct = 0.003
		dipPct   = -0.01
		dipEvery = 60
		dipLen   = 3
	)
	t := time.Date(2026, 2, 2, 0, 0, 0, 0, screenMSK) // Monday
	out := make([]backtest.Candle, 0, n)
	price := 100.0
	for i := 0; len(out) < n; {
		if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
			t = t.Add(30 * time.Minute)
			continue
		}
		pct := driftPct
		if i%(dipEvery+dipLen) >= dipEvery {
			pct = dipPct
		}
		open, closeP := price, price*(1+pct)
		out = append(out, backtest.Candle{
			Time: t, Open: open, High: max(open, closeP), Low: min(open, closeP), Close: closeP, Volume: 10000,
		})
		price = closeP
		t = t.Add(30 * time.Minute)
		i++
	}
	return out
}

func TestScreenZoneTickerRunsTheGridParamsVerbatim(t *testing.T) {
	// AFKS is a REGISTERED rsi_zone ticker with a calibrated literal (RSI 4, 25/75,
	// EMA 50, stop 1.0). The screener must grade it on the grid config like every other
	// ticker. One grid config through ScreenZoneTicker must reproduce a direct engine
	// run with that config; the fixture is built to trade so the comparison is not
	// 0 vs 0. On this fixture every trade may win, which caps PF at -pf-cap on BOTH
	// paths — so the test also compares the summed PnL (via BestHalves), which the
	// RSIUpper 65 vs 75 difference does move.
	bars := zoneFixtureCandles(1200)
	daily := dailyCandlesMSK(150, 100, 2)
	split := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	opts := DefaultScreenOpts()
	cfg := ZoneGrid()[10] // RSI 4, 25/65, EMA 200: RSI(4) crosses 25 on the fixture; exit 65 differs from the literal's 75
	if cfg.RSIPeriod != 4 || cfg.RSILower != 25 || cfg.RSIUpper != 65 || cfg.EMAPeriod != 200 {
		t.Fatalf("ZoneGrid()[10] = %+v, want RSI 4 25/65 EMA 200 — grid order changed", cfg)
	}

	want := backtest.Run(zonecore.NewWithParams("AFKS", cfg), bars, daily, nil,
		backtest.Config{InitialCash: opts.Cash, Fraction: opts.Fraction, Commission: opts.Commission, Lot: 1})
	train, _ := splitTrades(want.Trades, split)
	wantPF, wantN := profitFactor(train)
	if wantN == 0 {
		t.Fatal("fixture produces no train trades: the test cannot tell the grid config from the registered literal")
	}
	wantPF, _ = clampPF(wantPF, opts.PFCap)

	row := ScreenZoneTicker(ZoneTickerInput{
		Ticker: "AFKS", Name: "calibrated", Bars: bars, Daily: daily, Lot: 1, MinPriceIncrement: 0.001,
	}, []zonecore.Params{cfg}, split, opts)

	if row.PFMed != wantPF || row.TradesMed != float64(wantN) {
		t.Fatalf("PFMed/TradesMed = %v/%v, want %v/%d — ScreenZoneTicker must run the grid config", row.PFMed, row.TradesMed, wantPF, wantN)
	}
	if want := halfYearResults(train); !reflect.DeepEqual(row.BestHalves, want) {
		t.Fatalf("BestHalves = %+v, want %+v — ScreenZoneTicker must run the grid config, not the registered literal", row.BestHalves, want)
	}
	if row.Bars != len(bars) || row.TurnoverM <= 0 || row.DailyATRPct <= 0 {
		t.Fatalf("Bars/TurnoverM/DailyATRPct = %d/%v/%v, want %d/>0/>0", row.Bars, row.TurnoverM, row.DailyATRPct, len(bars))
	}
	if row.TickUnknown || row.TickPct <= 0 {
		t.Fatalf("TickPct/TickUnknown = %v/%v, want >0/false", row.TickPct, row.TickUnknown)
	}
	if row.AboveEMA50 <= 0.5 {
		t.Fatalf("AboveEMA50 = %v, want most of an uptrend above EMA 50", row.AboveEMA50)
	}
}

func TestScreenZoneTickerFlatSeriesProducesNoSignals(t *testing.T) {
	bars := tinyCandles(600)
	for i := range bars {
		bars[i].Open, bars[i].High, bars[i].Low, bars[i].Close = 100, 100, 100, 100
	}
	row := ScreenZoneTicker(ZoneTickerInput{
		Ticker: "XXXX", Name: "Test", Bars: bars, Daily: dailyCandlesMSK(60, 100, 2), Lot: 1,
	}, ZoneGrid(), time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), DefaultScreenOpts())

	if !row.NoSignals {
		t.Fatalf("NoSignals = false on a flat series, row = %+v", row)
	}
	if !row.TickUnknown {
		t.Fatal("TickUnknown = false with MinPriceIncrement 0, want true")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TestAggregateZone|TestScreenZoneTicker'`
Expected: FAIL — build error, `AggregateZone`, `ZoneConfigResult`, `ScreenZoneTicker`, `ZoneTickerInput` undefined.

- [ ] **Step 3: Write the implementation**

Append to `internal/service/backtest/zone_screen.go`:

```go
// ZoneConfigResult is one grid configuration's trade list on one ticker.
type ZoneConfigResult struct {
	Params zonecore.Params
	Trades []backtest.Trade
}

// ZoneRow is one ticker's zone-screening result. Medians run across grid configurations.
type ZoneRow struct {
	Ticker      string
	Name        string
	TurnoverM   float64 // mean daily turnover, millions of RUB
	DailyATRPct float64 // mean weekday daily ATR, percent of close
	Bars        int     // 30-minute bars replayed
	TickPct     float64 // two price ticks as a percentage of the last close
	TickUnknown bool    // the API gave no tick (or no price): TickPct is 0 and unmeasured

	PFMed       float64 // median raw train profit factor
	PFMedStress float64 // median train profit factor under stressed costs: the ranking key
	TradesMed   float64
	Plateau     float64 // share of configurations with PF >= PlateauPF at >= PlateauTrades trades
	Capped      int
	SilentCfg   int

	PFMedHO       float64 // holdout, never a ranking key
	PFMedStressHO float64
	TradesMedHO   float64

	HalvesPos float64 // median share of profitable half-years (trading configurations only)
	TopHalf   float64 // median share of the best half in summed profitable halves; 0 = none profitable
	SLShare   float64 // median share of stop exits (trading configurations only)
	HoldDays  float64 // median of per-configuration median holding time, calendar days

	AboveEMA50  float64 // share of train bars with close > EMA(50)
	AboveEMA200 float64 // share of train bars with close > EMA(200)

	Best       zonecore.Params // configuration with the highest raw train PF (reference only)
	BestPF     float64
	BestHalves []HalfResult // half-year breakdown of Best's train trades
	NoSignals  bool         // no configuration traded in either window
}

// AggregateZone reduces one ticker's per-configuration results to a report row. tick is
// the instrument's price increment used by the cost stress (0 = unknown).
func AggregateZone(ticker, name string, results []ZoneConfigResult, split time.Time, tick float64, opts ScreenOpts) ZoneRow {
	row := ZoneRow{Ticker: ticker, Name: name, NoSignals: true}
	n := len(results)
	pfs, pfsStress, counts := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	pfsHO, pfsStressHO, countsHO := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	var halvesPos, topHalf, sl, hold []float64
	var plateau int
	var haveBest bool
	var bestTrain []backtest.Trade

	for _, r := range results {
		train, holdout := splitTrades(r.Trades, split)

		pf, cnt := profitFactor(train)
		if cnt > 0 {
			row.NoSignals = false
		} else {
			row.SilentCfg++
		}
		// The first configuration claims Best unconditionally, so Best is always a real
		// grid entry even when every raw PF is 0 (see Aggregate in pullback_screen.go).
		if !haveBest || pf > row.BestPF {
			row.BestPF, row.Best, bestTrain = pf, r.Params, train
			haveBest = true
		}
		pf, capped := clampPF(pf, opts.PFCap)
		if capped {
			row.Capped++
		}
		if pf >= opts.PlateauPF && cnt >= opts.PlateauTrades {
			plateau++
		}
		pfs = append(pfs, pf)
		counts = append(counts, float64(cnt))

		pfS, _ := profitFactor(stressTrades(train, opts.Commission, tick))
		pfS, _ = clampPF(pfS, opts.PFCap)
		pfsStress = append(pfsStress, pfS)

		pfHO, cntHO := profitFactor(holdout)
		if cntHO > 0 {
			row.NoSignals = false
		}
		pfHO, _ = clampPF(pfHO, opts.PFCap)
		pfsHO = append(pfsHO, pfHO)
		countsHO = append(countsHO, float64(cntHO))
		pfSHO, _ := profitFactor(stressTrades(holdout, opts.Commission, tick))
		pfSHO, _ = clampPF(pfSHO, opts.PFCap)
		pfsStressHO = append(pfsStressHO, pfSHO)

		if cnt == 0 {
			continue // a silent configuration has no halves, exits or holding time to judge
		}
		pos, top, hasProfit := halvesStats(halfYearResults(train))
		halvesPos = append(halvesPos, pos)
		if hasProfit {
			topHalf = append(topHalf, top)
		}
		sl = append(sl, slShare(train))
		hold = append(hold, medianHoldDays(train))
	}

	row.BestPF, _ = clampPF(row.BestPF, opts.PFCap)
	row.BestHalves = halfYearResults(bestTrain)
	row.PFMed = medianF(pfs)
	row.PFMedStress = medianF(pfsStress)
	row.TradesMed = medianF(counts)
	row.PFMedHO = medianF(pfsHO)
	row.PFMedStressHO = medianF(pfsStressHO)
	row.TradesMedHO = medianF(countsHO)
	row.HalvesPos = medianF(halvesPos)
	row.TopHalf = medianF(topHalf)
	row.SLShare = medianF(sl)
	row.HoldDays = medianF(hold)
	if n > 0 {
		row.Plateau = float64(plateau) / float64(n)
	}
	return row
}

// ZoneTickerInput is everything ScreenZoneTicker needs about one instrument.
type ZoneTickerInput struct {
	Ticker            string
	Name              string
	Bars              []backtest.Candle // 30-minute bars over the whole window (train + holdout)
	Daily             []backtest.Candle // daily bars with a warm-up lead-in
	Lot               int32
	MinPriceIncrement float64 // 0 when unknown
}

// ScreenZoneTicker replays every grid configuration over one ticker and reduces the runs
// to a report row. The strategy is built directly with zonecore.NewWithParams and NOT
// through RSIZoneLookupOrGeneric: registered tickers carry calibrated literals, and
// grading them on those would make their rows incomparable with the rest.
func ScreenZoneTicker(in ZoneTickerInput, cfgs []zonecore.Params, split time.Time, opts ScreenOpts) ZoneRow {
	cfg := backtest.Config{
		InitialCash: opts.Cash,
		Fraction:    opts.Fraction,
		Commission:  opts.Commission,
		Lot:         in.Lot,
	}
	results := make([]ZoneConfigResult, 0, len(cfgs))
	for _, p := range cfgs {
		// rsi_zone needs no higher-timeframe series: htfCandles is nil.
		res := backtest.Run(zonecore.NewWithParams(in.Ticker, p), in.Bars, in.Daily, nil, cfg)
		results = append(results, ZoneConfigResult{Params: p, Trades: res.Trades})
	}
	row := AggregateZone(in.Ticker, in.Name, results, split, in.MinPriceIncrement, opts)
	row.Bars = len(in.Bars)
	row.TurnoverM = backtest.MeanDailyTurnoverM(in.Bars, in.Lot)
	row.DailyATRPct = MeanDailyATRPct(in.Daily, screenDailyATRPeriod)

	var lastClose float64
	if len(in.Bars) > 0 {
		lastClose = in.Bars[len(in.Bars)-1].Close
	}
	row.TickPct, row.TickUnknown = tickPct(in.MinPriceIncrement, lastClose)

	var train []backtest.Candle
	for _, b := range in.Bars {
		if b.Time.Before(split) {
			train = append(train, b)
		}
	}
	row.AboveEMA50 = aboveEMAShare(train, 50)
	row.AboveEMA200 = aboveEMAShare(train, 200)
	return row
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestAggregateZone|TestScreenZoneTicker|TestZoneGrid|TestStressTrades|TestTickPct|TestHalf|TestSLShare|TestAboveEMA' -v`
Expected: PASS. If `TestScreenZoneTickerRunsTheGridParamsVerbatim` fails on its "fixture produces no train trades" guard, the fixture — not the code — is wrong: raise `dipPct` to `-0.015` in `zoneFixtureCandles` and rerun; never weaken the guard.

- [ ] **Step 5: Mutation check**

Temporarily change `zonecore.NewWithParams(in.Ticker, p)` in `ScreenZoneTicker` to `RSIZoneLookupOrGeneric(in.Ticker).Build(RSIZoneLookupOrGeneric(in.Ticker).DefaultParams())`, run `go test ./internal/service/backtest/ -run TestScreenZoneTickerRunsTheGridParamsVerbatim`, confirm FAIL, then revert. Do not commit the mutation.

- [ ] **Step 6: Commit**

```bash
git add internal/service/backtest/zone_screen.go internal/service/backtest/zone_screen_test.go
git commit -m "feat(zonescreen): агрегация тикера и прогон сетки через движок rsi_zone

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Gates, ranking and Markdown report

**Files:**
- Modify: `internal/service/backtest/pullback_screen_report.go:66-96` (`Distribution`)
- Create: `internal/service/backtest/zone_screen_report.go`
- Create: `internal/service/backtest/zone_screen_report_test.go`

**Interfaces:**
- Consumes (Task 3): `ZoneRow`, `HalfResult`, `ZoneGrid`. Existing: `PFDist`, `medianF`.
- Produces:
  - `func distributionOf(vals []float64) PFDist` (unexported; `Distribution` delegates to it)
  - `type ZoneGates struct { MinTurnoverM, MinATRPct, MaxTickPct float64 }`
  - `func FilterAndRankZone(rows []ZoneRow, g ZoneGates) (ranked, noSignals, tickRejected, rejected []ZoneRow)`
  - `func ZoneDistribution(ranked []ZoneRow) PFDist`
  - `type ZoneScreenMeta struct { Months, HoldoutMonths, TopN int; Split time.Time; Gates ZoneGates; PFCap, Commission, Cash float64; Scanned, Skipped, Rejected int }`
  - `func RenderZoneScreenMarkdown(ranked, noSignals, tickRejected []ZoneRow, meta ZoneScreenMeta) string`

- [ ] **Step 1: Refactor `Distribution` (behaviour-preserving)**

Replace the body of `Distribution` in `pullback_screen_report.go` with a delegate and move the math into `distributionOf`:

```go
// Distribution summarizes PFMed across the ranked rows. For N=1, quartiles
// equal the median to avoid misleading 0.00 values in the report.
func Distribution(ranked []PullbackRow) PFDist {
	vals := make([]float64, 0, len(ranked))
	for _, r := range ranked {
		vals = append(vals, r.PFMed)
	}
	return distributionOf(vals)
}

// distributionOf summarizes a set of profit factors: quartiles, extremes and the share
// at or above 1.5. Shared by both screeners, each passing its own ranking key.
func distributionOf(vals []float64) PFDist {
	if len(vals) == 0 {
		return PFDist{}
	}
	s := append([]float64(nil), vals...)
	sort.Float64s(s)
	var above int
	for _, v := range s {
		if v >= 1.5 {
			above++
		}
	}
	median := medianF(s)
	q1, q3 := median, median // default for N=1
	if len(s) > 1 {
		q1 = medianF(s[:len(s)/2])
		q3 = medianF(s[(len(s)+1)/2:])
	}
	return PFDist{
		Min:          s[0],
		Q1:           q1,
		Median:       median,
		Q3:           q3,
		Max:          s[len(s)-1],
		ShareAbove15: float64(above) / float64(len(s)),
		N:            len(s),
	}
}
```

Run: `go test ./internal/service/backtest/ -run 'TestDistribution|TestRenderPullback'`
Expected: PASS (existing tests unchanged).

- [ ] **Step 2: Write the failing tests**

Create `internal/service/backtest/zone_screen_report_test.go`:

```go
package backtest

import (
	"strings"
	"testing"
	"time"
)

func zoneRow(ticker string, stress float64) ZoneRow {
	return ZoneRow{Ticker: ticker, Name: ticker + " name", TurnoverM: 100, DailyATRPct: 2, TickPct: 0.05,
		PFMed: stress + 0.5, PFMedStress: stress, TradesMed: 30, Best: ZoneGrid()[0]}
}

var testZoneGates = ZoneGates{MinTurnoverM: 50, MinATRPct: 1.5, MaxTickPct: 0.3}

func TestFilterAndRankZoneGates(t *testing.T) {
	lowTurnover := zoneRow("LOWT", 3)
	lowTurnover.TurnoverM = 10
	lowATR := zoneRow("LOWA", 3)
	lowATR.DailyATRPct = 1
	wideTick := zoneRow("KMAZ", 3)
	wideTick.TickPct = 0.4
	silent := zoneRow("SILN", 0)
	silent.NoSignals = true
	good := zoneRow("GOOD", 1.2)

	ranked, noSignals, tickRejected, rejected := FilterAndRankZone(
		[]ZoneRow{lowTurnover, lowATR, wideTick, silent, good}, testZoneGates)

	if len(ranked) != 1 || ranked[0].Ticker != "GOOD" {
		t.Fatalf("ranked = %+v, want only GOOD", ranked)
	}
	if len(noSignals) != 1 || noSignals[0].Ticker != "SILN" {
		t.Fatalf("noSignals = %+v, want only SILN", noSignals)
	}
	if len(tickRejected) != 1 || tickRejected[0].Ticker != "KMAZ" {
		t.Fatalf("tickRejected = %+v, want only KMAZ", tickRejected)
	}
	if len(rejected) != 2 {
		t.Fatalf("rejected = %d rows, want 2 (turnover, ATR)", len(rejected))
	}
}

func TestFilterAndRankZoneUnknownTickPasses(t *testing.T) {
	r := zoneRow("NOTK", 2)
	r.TickPct, r.TickUnknown = 0, true
	ranked, _, tickRejected, _ := FilterAndRankZone([]ZoneRow{r}, testZoneGates)
	if len(ranked) != 1 || len(tickRejected) != 0 {
		t.Fatalf("ranked/tickRejected = %d/%d, want 1/0 — an unmeasured tick is flagged, not rejected", len(ranked), len(tickRejected))
	}
}

func TestFilterAndRankZoneTickGateDisabledAtZero(t *testing.T) {
	r := zoneRow("KMAZ", 2)
	r.TickPct = 5
	g := testZoneGates
	g.MaxTickPct = 0
	ranked, _, tickRejected, _ := FilterAndRankZone([]ZoneRow{r}, g)
	if len(ranked) != 1 || len(tickRejected) != 0 {
		t.Fatalf("ranked/tickRejected = %d/%d, want 1/0 — -max-tick-pct 0 disables the gate", len(ranked), len(tickRejected))
	}
}

func TestFilterAndRankZoneSortsByStressedPFThenTicker(t *testing.T) {
	a, b, c := zoneRow("BBBB", 1.5), zoneRow("AAAA", 1.5), zoneRow("CCCC", 2.0)
	c.PFMed = 0.1 // raw PF must not drive the order
	ranked, _, _, _ := FilterAndRankZone([]ZoneRow{a, b, c}, testZoneGates)
	got := []string{ranked[0].Ticker, ranked[1].Ticker, ranked[2].Ticker}
	if strings.Join(got, ",") != "CCCC,AAAA,BBBB" {
		t.Fatalf("order = %v, want CCCC,AAAA,BBBB", got)
	}
}

func TestZoneDistributionUsesStressedPF(t *testing.T) {
	d := ZoneDistribution([]ZoneRow{zoneRow("A", 1), zoneRow("B", 2), zoneRow("C", 3)})
	if d.Median != 2 || d.N != 3 || d.Min != 1 || d.Max != 3 {
		t.Fatalf("distribution = %+v, want median 2 over stressed PF 1..3", d)
	}
}

func testZoneMeta() ZoneScreenMeta {
	return ZoneScreenMeta{Months: 36, HoldoutMonths: 6, TopN: 50, Split: time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC),
		Gates: testZoneGates, PFCap: 10, Commission: 0.0005, Cash: 100000, Scanned: 5}
}

func TestRenderZoneScreenMarkdownSections(t *testing.T) {
	r := zoneRow("GOOD", 1.8)
	r.HalvesPos, r.TopHalf, r.SLShare, r.HoldDays, r.AboveEMA50, r.AboveEMA200 = 0.75, 0.4, 0.2, 3.5, 0.6, 0.55
	r.BestHalves = []HalfResult{{Label: "2025H1", PnL: 3100}, {Label: "2025H2", PnL: -1200}}
	tick := zoneRow("KMAZ", 1)
	tick.TickPct = 0.4
	silent := zoneRow("SILN", 0)

	md := RenderZoneScreenMarkdown([]ZoneRow{r}, []ZoneRow{silent}, []ZoneRow{tick}, testZoneMeta())

	for _, want := range []string{
		"# RSI zone screener",
		"## Распределение PFmed×2",
		"## Рейтинг",
		"| # | Ticker | Name | Оборот, млн | ATR% дн | Tick% | Бары | TradesMed | PFmed | PFmed×2 |",
		"## Детали топа",
		"| Ticker | Halves+ | TopHalf | SL% | HoldD | >EMA50 | >EMA200 | Лучшая конфигурация | Полугодия лучшей конфигурации |",
		"| GOOD | 75% | 40% | 20% | 3.5 | 60% | 55% |",
		"RSI 4 15/65, EMA 50",
		"2025H1 +3.1% · 2025H2 -1.2%",
		"## Нет сигналов",
		"SILN",
		"## Отсеяно гейтом шага цены",
		"| KMAZ | 0.40 |",
		"## Как это читать",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("report missing %q:\n%s", want, md)
		}
	}
}

func TestRenderZoneDashesMissingTopHalf(t *testing.T) {
	r := zoneRow("NOPF", 0.5)
	r.TopHalf = 0
	md := RenderZoneScreenMarkdown([]ZoneRow{r}, nil, nil, testZoneMeta())
	if !strings.Contains(md, "| NOPF | 0% | — |") {
		t.Fatalf("TopHalf 0 must render as a dash:\n%s", md)
	}
}

func TestRenderZoneMarksUnknownTick(t *testing.T) {
	r := zoneRow("NOTK", 1.1)
	r.TickPct, r.TickUnknown = 0, true
	md := RenderZoneScreenMarkdown([]ZoneRow{r}, nil, nil, testZoneMeta())
	if !strings.Contains(md, "| 2.00 | ? |") {
		t.Fatalf("unknown tick must render as ?:\n%s", md)
	}
}

func TestRenderZoneRespectsTopNAndOmitsEmptySections(t *testing.T) {
	meta := testZoneMeta()
	meta.TopN = 1
	md := RenderZoneScreenMarkdown([]ZoneRow{zoneRow("AAAA", 2), zoneRow("BBBB", 1)}, nil, nil, meta)
	if strings.Contains(md, "BBBB") {
		t.Fatalf("TopN=1 rendered the second row:\n%s", md)
	}
	if strings.Contains(md, "## Нет сигналов") || strings.Contains(md, "## Отсеяно гейтом шага цены") {
		t.Fatalf("empty sections must be omitted:\n%s", md)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TestFilterAndRankZone|TestZoneDistribution|TestRenderZone'`
Expected: FAIL — build error, `FilterAndRankZone`, `ZoneGates`, `ZoneDistribution`, `ZoneScreenMeta`, `RenderZoneScreenMarkdown` undefined.

- [ ] **Step 4: Write the implementation**

Create `internal/service/backtest/zone_screen_report.go`:

```go
package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ZoneGates are the zone screener's hard gates. MaxTickPct 0 disables the tick gate.
type ZoneGates struct {
	MinTurnoverM float64 // mean daily turnover, millions of RUB
	MinATRPct    float64 // mean weekday daily ATR, percent of close
	MaxTickPct   float64 // two price ticks as a percentage of price
}

// FilterAndRankZone applies the hard gates in order (turnover, daily ATR, price tick),
// sends signal-less survivors to their own bucket and ranks the rest by the median
// STRESSED profit factor, descending, ticker ascending on ties. A row whose tick is
// unknown passes the tick gate: the report flags it instead of guessing a cost.
func FilterAndRankZone(rows []ZoneRow, g ZoneGates) (ranked, noSignals, tickRejected, rejected []ZoneRow) {
	for _, r := range rows {
		switch {
		case r.TurnoverM < g.MinTurnoverM || r.DailyATRPct < g.MinATRPct:
			rejected = append(rejected, r)
		case g.MaxTickPct > 0 && !r.TickUnknown && r.TickPct > g.MaxTickPct:
			tickRejected = append(tickRejected, r)
		case r.NoSignals:
			noSignals = append(noSignals, r)
		default:
			ranked = append(ranked, r)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].PFMedStress != ranked[j].PFMedStress {
			return ranked[i].PFMedStress > ranked[j].PFMedStress
		}
		return ranked[i].Ticker < ranked[j].Ticker
	})
	byTicker := func(rs []ZoneRow) {
		sort.SliceStable(rs, func(i, j int) bool { return rs[i].Ticker < rs[j].Ticker })
	}
	byTicker(noSignals)
	byTicker(tickRejected)
	return ranked, noSignals, tickRejected, rejected
}

// ZoneDistribution summarizes the ranking key (stressed median PF) across the ranking.
func ZoneDistribution(ranked []ZoneRow) PFDist {
	vals := make([]float64, 0, len(ranked))
	for _, r := range ranked {
		vals = append(vals, r.PFMedStress)
	}
	return distributionOf(vals)
}

// ZoneScreenMeta carries the run context shown in the report header.
type ZoneScreenMeta struct {
	Months        int
	HoldoutMonths int
	TopN          int
	Split         time.Time
	Gates         ZoneGates
	PFCap         float64
	Commission    float64 // per side, fraction of turnover; the stress adds this again per side
	Cash          float64 // mock starting cash; half-year results are shown as a percentage of it
	Scanned       int
	Skipped       int
	Rejected      int // rows that failed the turnover or daily-ATR gate
}

func pct0(v float64) string { return fmt.Sprintf("%.0f%%", v*100) }

// RenderZoneScreenMarkdown renders the zone screening report.
func RenderZoneScreenMarkdown(ranked, noSignals, tickRejected []ZoneRow, meta ZoneScreenMeta) string {
	var b strings.Builder
	d := ZoneDistribution(ranked)
	gridSize := len(ZoneGrid())

	b.WriteString("# RSI zone screener\n\n")
	fmt.Fprintf(&b, "Окно: %d мес., holdout: последние %d мес. (срез %s).\n",
		meta.Months, meta.HoldoutMonths, meta.Split.Format("2006-01-02"))
	fmt.Fprintf(&b, "Сетка: %d конфигураций (RSIPeriod x RSILower x EMAPeriod x RSIUpper), стоп 1.0 дневного ATR.\n", gridSize)
	fmt.Fprintf(&b, "Гейты: оборот >= %.0f млн ₽/день, дневной ATR >= %.2f%%, два шага цены <= %.2f%% цены (0 — гейт выключен). PF зажат сверху на %.1f.\n",
		meta.Gates.MinTurnoverM, meta.Gates.MinATRPct, meta.Gates.MaxTickPct, meta.PFCap)
	fmt.Fprintf(&b, "Стресс издержек (`PFmed×2`): к комиссии %.4f за сторону добавлена ещё одна такая же и два шага цены на круг.\n",
		meta.Commission)
	fmt.Fprintf(&b, "Вселенная: scanned=%d ranked=%d no-signal=%d tick-rejected=%d rejected=%d skipped=%d.\n\n",
		meta.Scanned, len(ranked), len(noSignals), len(tickRejected), meta.Rejected, meta.Skipped)

	b.WriteString("## Распределение PFmed×2 по ранжированной вселенной\n\n")
	fmt.Fprintf(&b, "min %.2f · Q1 %.2f · медиана %.2f · Q3 %.2f · max %.2f · доля PFmed×2 >= 1.5: %.0f%% (n=%d)\n\n",
		d.Min, d.Q1, d.Median, d.Q3, d.Max, d.ShareAbove15*100, d.N)
	b.WriteString("Читать эту строку раньше первой строки топа: если планку проходит половина вселенной, планка ничего не значит.\n\n")

	top := ranked
	if meta.TopN > 0 && meta.TopN < len(top) {
		top = top[:meta.TopN]
	}

	b.WriteString("## Рейтинг\n\n")
	b.WriteString("| # | Ticker | Name | Оборот, млн | ATR% дн | Tick% | Бары | TradesMed | PFmed | PFmed×2 | Capped | SilentCfg | Plateau | PFmed×2 HO | Trades HO |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range top {
		tick := fmt.Sprintf("%.2f", r.TickPct)
		if r.TickUnknown {
			tick = "?"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %.0f | %.2f | %s | %d | %.0f | %.2f | %.2f | %d/%d | %d/%d | %s | %.2f | %.0f |\n",
			i+1, r.Ticker, r.Name, r.TurnoverM, r.DailyATRPct, tick, r.Bars, r.TradesMed,
			r.PFMed, r.PFMedStress, r.Capped, gridSize, r.SilentCfg, gridSize,
			pct0(r.Plateau), r.PFMedStressHO, r.TradesMedHO)
	}
	b.WriteString("\n`Tick%` `?` — API не отдал шаг цены: издержки шага не измерены, стресс учитывает только комиссию.\n")
	b.WriteString("`PFmed×2 HO` в сортировке не участвует: это красный флаг («работало и развалилось»), а не критерий отбора.\n\n")

	b.WriteString("## Детали топа\n\n")
	b.WriteString("| Ticker | Halves+ | TopHalf | SL% | HoldD | >EMA50 | >EMA200 | Лучшая конфигурация | Полугодия лучшей конфигурации |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|\n")
	for _, r := range top {
		topHalf := "—"
		if r.TopHalf > 0 {
			topHalf = pct0(r.TopHalf)
		}
		best := fmt.Sprintf("RSI %d %.0f/%.0f, EMA %d", r.Best.RSIPeriod, r.Best.RSILower, r.Best.RSIUpper, r.Best.EMAPeriod)
		halves := make([]string, 0, len(r.BestHalves))
		for _, h := range r.BestHalves {
			share := 0.0
			if meta.Cash > 0 {
				share = h.PnL / meta.Cash * 100
			}
			halves = append(halves, fmt.Sprintf("%s %+.1f%%", h.Label, share))
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %.1f | %s | %s | %s | %s |\n",
			r.Ticker, pct0(r.HalvesPos), topHalf, pct0(r.SLShare), r.HoldDays,
			pct0(r.AboveEMA50), pct0(r.AboveEMA200), best, strings.Join(halves, " · "))
	}
	b.WriteString("\n`Halves+` — доля прибыльных полугодий; `TopHalf` — доля лучшего полугодия в сумме прибыльных (100% — весь профит из одного отрезка, «—» — прибыльных нет); ")
	b.WriteString("`SL%` — доля выходов по стопу; `HoldD` — медианная длительность сделки, дней; `>EMA50`/`>EMA200` — доля train-баров выше EMA. ")
	b.WriteString("Полугодия лучшей конфигурации — результат train-сделок в процентах от стартового кэша.\n\n")

	if len(noSignals) > 0 {
		b.WriteString("## Нет сигналов\n\n")
		b.WriteString("Тикеры, прошедшие гейты, но не давшие ни одной сделки ни в одной из конфигураций:\n\n")
		names := make([]string, 0, len(noSignals))
		for _, r := range noSignals {
			names = append(names, r.Ticker)
		}
		fmt.Fprintf(&b, "%s\n\n", strings.Join(names, ", "))
	}

	if len(tickRejected) > 0 {
		b.WriteString("## Отсеяно гейтом шага цены\n\n")
		b.WriteString("| Ticker | Tick% | PFmed | PFmed×2 |\n|---|---|---|---|\n")
		for _, r := range tickRejected {
			fmt.Fprintf(&b, "| %s | %.2f | %.2f | %.2f |\n", r.Ticker, r.TickPct, r.PFMed, r.PFMedStress)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Как это читать\n\n")
	b.WriteString("Шортлист — это **кандидаты на калибровку**, а не доказательство edge. ")
	b.WriteString("Верх такого рейтинга по построению содержит везунчиков: испытаний столько же, сколько тикеров умножить на конфигурации. ")
	b.WriteString("Планка приёмки не меняется — pooled OOS profit factor >= 1.5 в персональном walk-forward (docs/rsi_zone/strategy.md).\n")
	return b.String()
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestFilterAndRankZone|TestZoneDistribution|TestRenderZone|TestDistribution|TestRenderPullback' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/backtest/pullback_screen_report.go internal/service/backtest/zone_screen_report.go internal/service/backtest/zone_screen_report_test.go
git commit -m "feat(zonescreen): гейты, ранжирование по PFmed×2 и отчёт с деталями топа

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: CLI `cmd/zonescreen`, docs and smoke run

**Files:**
- Create: `cmd/zonescreen/main.go`
- Create: `cmd/zonescreen/main_test.go`
- Create: `docs/rsi_zone/screener.md`
- Modify: `docs/rsi_zone/strategy.md` (header block, after the `Спека:` line)
- Modify: `CLAUDE.md` (the `rsi_zone` sentence in Layout)

**Interfaces:**
- Consumes: `screenrun.*` (Task 1), `svc.ZoneGrid`, `svc.ZoneTickerInput`, `svc.ScreenZoneTicker`, `svc.FilterAndRankZone`, `svc.ZoneGates`, `svc.ZoneScreenMeta`, `svc.RenderZoneScreenMarkdown`, `svc.DefaultScreenOpts`, `svc.NewCandleProvider`, `enum.Minutes30`, `enum.Day1`.
- Produces: binary `cmd/zonescreen`; `func validate(cfg runCfg) error` (unexported, tested).

- [ ] **Step 1: Write the failing test**

Create `cmd/zonescreen/main_test.go`:

```go
package main

import "testing"

func TestValidateRejectsHoldoutNotShorterThanWindow(t *testing.T) {
	if err := validate(runCfg{months: 6, holdoutMonths: 6, workers: 1}); err == nil {
		t.Fatal("validate accepted -holdout-months == -months, want an error: the train window would be empty")
	}
}

func TestValidateRejectsZeroWorkers(t *testing.T) {
	if err := validate(runCfg{months: 36, holdoutMonths: 6, workers: 0}); err == nil {
		t.Fatal("validate accepted -workers 0, want an error")
	}
}

func TestValidateRejectsNegativeTickGate(t *testing.T) {
	if err := validate(runCfg{months: 36, holdoutMonths: 6, workers: 1, maxTickPct: -0.1}); err == nil {
		t.Fatal("validate accepted -max-tick-pct < 0, want an error (0 is the off switch)")
	}
}

func TestValidateAcceptsDefaults(t *testing.T) {
	if err := validate(runCfg{months: 36, holdoutMonths: 6, workers: 8, maxTickPct: 0.3}); err != nil {
		t.Fatalf("validate rejected defaults: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/zonescreen/`
Expected: FAIL — no Go files / `validate` undefined.

- [ ] **Step 3: Write the CLI**

Create `cmd/zonescreen/main.go`:

```go
// Command zonescreen ranks the tradable RUB share universe by how well the rsi_zone
// strategy fits each ticker. Like cmd/pullscreen it replays the real strategy engine
// over a fixed 24-configuration grid, but it ranks by the median profit factor under
// STRESSED costs (double commission plus two price ticks) and adds a detailed
// breakdown per ticker: half-year stability, exit profile and trend regime. All
// gRPC/file I/O lives here; the scoring is pure (internal/service/backtest/zone_screen*.go).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/backtest/screenrun"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

func main() {
	var (
		months        = flag.Int("months", 36, "lookback period in months")
		holdoutMonths = flag.Int("holdout-months", 6, "trailing months held out of the ranking window")
		minTurnoverM  = flag.Float64("min-turnover", 50, "gate: minimum mean daily turnover in millions of RUB")
		minATRPct     = flag.Float64("min-atr-pct", 1.5, "gate: minimum mean weekday daily ATR as a percentage of close")
		maxTickPct    = flag.Float64("max-tick-pct", 0.3, "gate: maximum two price ticks as a percentage of the last close (0 = off)")
		topN          = flag.Int("top", 50, "ranking rows in the report (0 = all)")
		workers       = flag.Int("workers", 8, "concurrent tickers")
		pfCap         = flag.Float64("pf-cap", 10, "cap on profit factor for ranking (0 = no cap)")
		cash          = flag.Float64("cash", 100000, "starting mock cash")
		fraction      = flag.Float64("fraction", 1.0, "fraction of cash per entry")
		commission    = flag.Float64("commission", 0.0005, "commission as a fraction of turnover, per side")
		tickersCSV    = flag.String("tickers", "", "comma-separated tickers instead of the full universe (diagnostics/smoke runs)")
		outDir        = flag.String("out", "reports/zone_screen", "report output directory")
		refresh       = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
		pause         = flag.Duration("pause", 0, "idle time after each ticker, e.g. 500ms or 2s: trades wall-clock for a cooler CPU")
	)
	flag.Parse()
	logger.Init()

	opts := svc.DefaultScreenOpts()
	opts.PFCap = *pfCap
	opts.Cash = *cash
	opts.Fraction = *fraction
	opts.Commission = *commission

	if err := run(context.Background(), runCfg{
		months: *months, holdoutMonths: *holdoutMonths, topN: *topN, workers: *workers,
		minTurnoverM: *minTurnoverM, minATRPct: *minATRPct, maxTickPct: *maxTickPct,
		tickers: screenrun.SplitCSV(*tickersCSV), outDir: *outDir, refresh: *refresh, pause: *pause, opts: opts,
	}); err != nil {
		log.Fatalf("zonescreen: %v", err)
	}
}

type runCfg struct {
	months, holdoutMonths, topN, workers int
	minTurnoverM, minATRPct, maxTickPct  float64
	tickers                              []string
	outDir                               string
	refresh                              bool
	pause                                time.Duration
	opts                                 svc.ScreenOpts
}

// validate rejects flag combinations that would silently produce a meaningless run.
func validate(cfg runCfg) error {
	if cfg.holdoutMonths >= cfg.months {
		return fmt.Errorf("-holdout-months (%d) must be smaller than -months (%d)", cfg.holdoutMonths, cfg.months)
	}
	if cfg.workers < 1 {
		return fmt.Errorf("-workers must be at least 1")
	}
	if cfg.maxTickPct < 0 {
		return fmt.Errorf("-max-tick-pct must be >= 0 (0 disables the gate)")
	}
	return nil
}

func run(ctx context.Context, cfg runCfg) error {
	if err := validate(cfg); err != nil {
		return err
	}
	if workers, capped := screenrun.EffectiveWorkers(cfg.workers, cfg.refresh); capped {
		fmt.Printf("zonescreen: -refresh forces -workers %d -> %d to avoid punching holes in the local candle cache (see screenrun.RefreshWorkerCap)\n",
			cfg.workers, workers)
		cfg.workers = workers
	}
	token, err := screenrun.LoadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(screenrun.APIAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}
	universe, err := screenrun.LoadUniverse(ctx, client, cfg.tickers)
	if err != nil {
		return err
	}

	to := time.Now()
	from := to.AddDate(0, -cfg.months, 0)
	// A year of lead-in warms the daily ATR, exactly as cmd/backtest does.
	dailyFrom := from.AddDate(-1, 0, 0)
	split := to.AddDate(0, -cfg.holdoutMonths, 0)

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), screenrun.CacheDir)
	grid := svc.ZoneGrid()

	sem := semaphore.New(cfg.workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.ZoneRow
	var done, skipped int32

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u screenrun.ShareInfo) {
			defer wg.Done()
			defer sem.Release()

			bars, err := provider.Load(ctx, u.Ticker, u.ID, enum.Minutes30, from, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("zonescreen %s: skip (load 30m: %v)\n", u.Ticker, err)
				return
			}
			daily, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, dailyFrom, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("zonescreen %s: skip (load daily: %v)\n", u.Ticker, err)
				return
			}

			row := svc.ScreenZoneTicker(svc.ZoneTickerInput{
				Ticker: u.Ticker, Name: u.Name, Bars: bars, Daily: daily, Lot: u.Lot, MinPriceIncrement: u.MinPriceIncrement,
			}, grid, split, cfg.opts)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("zonescreen [%d/%d] %s: PFmed=%.2f PFmed×2=%.2f trades=%.0f tick=%.2f%% turnover=%.0fM atr=%.2f%% bars=%d\n",
				n, len(universe), u.Ticker, row.PFMed, row.PFMedStress, row.TradesMed, row.TickPct, row.TurnoverM, row.DailyATRPct, row.Bars)

			mu.Lock()
			rows = append(rows, row)
			mu.Unlock()

			// Held inside the semaphore slot on purpose: releasing first would let the
			// next ticker start while this one idles, and the pool would stay just as hot.
			screenrun.PauseAfterTicker(ctx, cfg.pause)
		}(u)
	}
	wg.Wait()

	gates := svc.ZoneGates{MinTurnoverM: cfg.minTurnoverM, MinATRPct: cfg.minATRPct, MaxTickPct: cfg.maxTickPct}
	ranked, noSignals, tickRejected, rejected := svc.FilterAndRankZone(rows, gates)
	meta := svc.ZoneScreenMeta{
		Months: cfg.months, HoldoutMonths: cfg.holdoutMonths, TopN: cfg.topN, Split: split, Gates: gates,
		PFCap: cfg.opts.PFCap, Commission: cfg.opts.Commission, Cash: cfg.opts.Cash,
		Scanned: len(universe), Skipped: int(skipped), Rejected: len(rejected),
	}

	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("zone_screen_Minutes30_%s.md", time.Now().Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(svc.RenderZoneScreenMarkdown(ranked, noSignals, tickRejected, meta)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("zonescreen report: %s (scanned=%d ranked=%d no-signal=%d tick-rejected=%d rejected=%d skipped=%d)\n",
		path, len(universe), len(ranked), len(noSignals), len(tickRejected), len(rejected), skipped)
	return nil
}
```

- [ ] **Step 4: Run tests and build**

Run: `go test ./cmd/zonescreen/ -v && go build ./internal/... ./pkg/... ./cmd/...`
Expected: 4 tests PASS, build silent.

- [ ] **Step 5: Write the docs**

Create `docs/rsi_zone/screener.md`:

````markdown
# Скринер тикеров под rsi_zone

Код: `cmd/zonescreen/main.go` (ввод-вывод), `internal/service/backtest/zone_screen.go` и
`zone_screen_report.go` (чистое ядро), `internal/service/backtest/screenrun` (обвязка, общая с
`cmd/pullscreen`). Спека: `docs/superpowers/specs/2026-09-28-rsi-zone-screener-design.md`.

## Что он делает

Прогоняет **настоящий движок `rsi_zone`** по каждому тикеру торгуемой RUB-вселенной MOEX на
фиксированной сетке из 24 конфигураций и ранжирует тикеры по медиане profit factor **при стрессе
издержек**. Отвечает на один вопрос: кому стоит потратить фазовую калибровку. Это не
доказательство edge: планка приёмки — pooled OOS profit factor ≥ 1.5 в персональном
walk-forward (`docs/rsi_zone/strategy.md`).

От `pullscreen` (`docs/rsi_pullback/screener.md`) отличается ключом сортировки и детальной
таблицей: стабильность по полугодиям, профиль выходов, режим тренда и доля шага цены.

## Запуск

```bash
# Полная вселенная, 36 месяцев, holdout — последние 6.
go run ./cmd/zonescreen -months 36 -holdout-months 6 -top 50

# Диагностика по конкретным тикерам.
go run ./cmd/zonescreen -tickers SBER,AFKS -months 24 -top 10

# Спокойный ночной прогон.
GOMAXPROCS=4 go run ./cmd/zonescreen -months 36 -holdout-months 6 -top 50 -workers 2 -pause 1s
```

Отчёт — `reports/zone_screen/zone_screen_Minutes30_<stamp>.md`. Про нагрузку, `-pause`,
`GOMAXPROCS` и опасность `-refresh` (дыры в кэше свечей, ограничение `-workers` до
`screenrun.RefreshWorkerCap`) — всё как в `docs/rsi_pullback/screener.md`.

## Сетка

`RSIPeriod` {4, 6} × `RSILower` {15, 25} × `EMAPeriod` {50, 100, 200} × `RSIUpper` {65, 75};
`StopDailyATR` закреплён на 1.0, `DailyATRPeriod` — 14. Стоп — свойство настроенной точки, а не
инструмента, поэтому в сетку он не входит. Движок строится через `core.NewWithParams`, в обход
реестра тикеров: откалиброванные тикеры оцениваются на той же сетке, что и все остальные.

## Флаги

| Флаг | Дефолт | Смысл |
|---|---|---|
| `-months` | 36 | глубина окна |
| `-holdout-months` | 6 | хвост окна, не участвующий в ранжировании |
| `-min-turnover` | 50 | гейт: средний дневной оборот, млн ₽ |
| `-min-atr-pct` | 1.5 | гейт: средний дневной ATR, % от цены |
| `-max-tick-pct` | 0.3 | гейт: два шага цены, % от последней цены; 0 выключает гейт |
| `-top` | 50 | строк в рейтинге и деталях (0 = все) |
| `-workers` | 8 | параллельных тикеров (до 2 при `-refresh`) |
| `-pf-cap` | 10 | потолок PF при ранжировании (0 — без потолка) |
| `-tickers` | — | список тикеров вместо всей вселенной |
| `-out` | `reports/zone_screen` | каталог отчёта |
| `-refresh` | false | перекачать свечи, игнорируя кэш |
| `-pause` | 0 | простой после каждого тикера |
| `-cash` | 100000 | мок-портфель: стартовый кэш; полугодия лучшей конфигурации показаны в % от него |
| `-fraction` | 1.0 | мок-портфель: доля кэша на вход |
| `-commission` | 0.0005 | мок-портфель: комиссия за сторону; стресс добавляет её ещё раз |

## Гейты

По порядку: оборот, дневной ATR, шаг цены. Строка, для которой API не отдал шаг цены (`Tick%` = `?`),
гейт шага проходит: издержки шага у неё не измерены, и отчёт это показывает. Прошедшие гейты тикеры
без единой сделки попадают в секцию «Нет сигналов», отсеянные гейтом шага — в отдельную секцию с
их PF, чтобы отсев был виден глазами.

## Как читать рейтинг

- **`PFmed×2`** — ключ сортировки (вторичный — тикер). Медиана по 24 конфигурациям train-PF,
  пересчитанного из тех же сделок со стрессом издержек:
  `PnL − Commission × (вход + выход) × бумаги − 2 × шаг цены × бумаги`. Изменение размера позиции от
  иначе сложившегося кэша не моделируется — на отношение PF оно пренебрежимо.
- **`PFmed`** — та же медиана без стресса. Большой разрыв с `PFmed×2` — edge живёт на грани издержек.
- **`Tick%`** — два шага цены в процентах от последней цены.
- **`Capped`**, **`SilentCfg`**, **`Plateau`**, **`TradesMed`** — как в `pullscreen`: конфигурация без
  сделок входит в медиану как PF = 0, читать `PFmed×2` всегда вместе с `SilentCfg`.
- **`PFmed×2 HO`** — то же на holdout. В сортировке не участвует: красный флаг, а не критерий.
- **Распределение `PFmed×2`** в шапке — читать раньше первой строки топа.

## Как читать детали топа

Медианы берутся только по конфигурациям, которые торговали на train.

- **`Halves+`** — доля прибыльных календарных полугодий (по времени входа, MSK) среди полугодий
  со сделками.
- **`TopHalf`** — доля лучшего полугодия в сумме прибыльных. 100% — весь профит из одного отрезка;
  «—» — прибыльных полугодий нет.
- **`SL%`** — доля выходов по стопу. Около нуля — стоп на 1.0 ATR не срабатывает и не защищает;
  почти 100% — RSI-выход не успевает.
- **`HoldD`** — медианная длительность сделки в календарных днях.
- **`>EMA50`**, **`>EMA200`** — доля train-баров с close выше EMA. Низкие значения объясняют
  низкий `TradesMed`: стратегия покупает только в тренде.
- **Лучшая конфигурация и её полугодия** — справочная стартовая точка для калибровки и
  раскладка её train-результата по полугодиям в % от `-cash`.

## Что скринер не меряет

- Проскальзывание сверх двух шагов цены и внутрибарные проколы стопа.
- Дивидендные гэпы.
- Ширину стопа: она закреплена и меряется в калибровке.
- Лот, размер счёта, конкуренцию за капитал с другими стратегиями.
````

In `docs/rsi_zone/strategy.md`, after the line `Спека: \`docs/superpowers/specs/2026-09-28-rsi-zone-design.md\`` add:

```markdown
Скринер тикеров: `docs/rsi_zone/screener.md` (`go run ./cmd/zonescreen`).
```

In `CLAUDE.md`, in the Layout bullet for `rsi_zone`, replace
`(`-strategy rsi_zone`, docs: `docs/rsi_zone/strategy.md`);` with
`(`-strategy rsi_zone`, docs: `docs/rsi_zone/strategy.md`; подбор тикеров — cmd/zonescreen, docs/rsi_zone/screener.md);`
and in the `pkg`/`internal/service` description nothing else changes.

- [ ] **Step 6: Smoke run on real data**

Run: `go run ./cmd/zonescreen -tickers AFKS,SBER,DOMRF -months 12 -holdout-months 3 -top 10 -workers 2`
Expected: three `zonescreen [n/3]` lines and a `zonescreen report: reports/zone_screen/...` line. Open the report and check: all three sections render, `Tick%` is a number (not `?`) for all three, `SL%` and `>EMA50` are non-zero for at least one ticker. If `T_BANK` is not set, report that the smoke run was skipped instead of claiming it passed. Do NOT commit the report (`reports/` output is not versioned — check `git status` shows it untracked or ignored and leave it).

- [ ] **Step 7: Full gate**

Run: `./bin/mage ci`
Expected: lint clean, `go test -race ./...` PASS, mock-drift check clean.

- [ ] **Step 8: Commit**

```bash
git add cmd/zonescreen docs/rsi_zone/screener.md docs/rsi_zone/strategy.md CLAUDE.md
git commit -m "feat(zonescreen): CLI скринера тикеров под rsi_zone и документация

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**Before staging `CLAUDE.md` and `docs/rsi_zone/strategy.md`:** both files already carry uncommitted owner edits from the rsi_zone calibration work on this branch. Run `git diff CLAUDE.md docs/rsi_zone/strategy.md` first. If any hunk other than the screener line is present, do not stage those two files — commit only `cmd/zonescreen` and `docs/rsi_zone/screener.md`, leave `CLAUDE.md`/`strategy.md` edited in the working tree, and report that to the owner. Interactive staging (`git add -p`) is not available.
