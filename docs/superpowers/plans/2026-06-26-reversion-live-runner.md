# Reversion Live Runner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run the `reversion` mean-reversion strategy live against Tinkoff Invest market data on a dedicated account, placing real market orders (full position in / full position out) gated by independent trade/notify env flags, reusing the backtest's `core.Decide` and `MarketData` assembly 1:1.

**Architecture:** A new package `internal/service/trading_strategy/reversion/live/` with focused sub-units (statestore, sizing, executor, marketdata, registry, notifier, reconstruct) orchestrated by a `service` with two passes (buy / manage). A scheduler registers two cron jobs. Wired into `app.go` via a `service_provider` getter and a new `ReversionConfig`. The backtest's per-bar `MarketData` assembly is extracted into an exported helper so backtest and live build the snapshot identically.

**Tech Stack:** Go 1.25, Tinkoff Invest gRPC API (`internal/pb/v1`), `robfig/cron/v3` (via `pkg/scheduler`), `heetch/confita` (env config), `google/uuid` (order idempotency), `stretchr/testify`.

## Global Constraints

- **Main timeframe is HOURLY.** The `core.go` package doc comment ("Run with `-interval Day1`") is **stale**; the validated reversion strategy and this design run hourly. Treat hourly as the strategy timeframe everywhere.
- **`core.Decide` reads ONLY these `strategy.MarketData` fields:** `Closes`, `Highs`, `Lows`, `Price`, `Volumes` + `Times` (only when `UseVolume==1`), `HTFCloses` (only when `HTFTrendEMA>0`), and `Position`. It does **not** read `DailyCloses`, `DailyHighs`, `DailyLows`, `TodayHigh`, `TodayLow`, `HTFHighs`, `HTFLows`. Verified in `internal/service/trading_strategy/reversion/strategy/core/core.go:146-230`.
- **`EntryATR` is computed on the HOURLY window**, not daily: `indicators.ATR(md.Highs, md.Lows, md.Closes, ATRPeriod)` (`core.go:170`). The `Params.ATRPeriod` doc says "daily" but the code uses the main window. The live runner must NOT fetch daily candles for ATR.
- **Live therefore fetches only hourly + 4H candles.** 4H is fetched only when an active ticker has `HTFTrendEMA>0` (today only NVTK, `HTFTrendEMA=150`). Daily candles are never fetched (the core ignores them, so omitting them yields identical decisions).
- **HTF completeness anchor (`cur`) = the open-time of the last completed hourly candle** (`window[len-1].Time`), NOT `time.Now()`. This mirrors the backtest, where `cur = candles[i].Time` is the current bar's open-time (`engine.go:118`).
- **`EntryATR` on a new entry comes from `sig.ATR`** (set at `core.go:375`), exactly as the backtest engine does (`engine.go:129`, `p.open(..., sig.ATR, ...)`). Never recompute it separately on entry.
- **All series are oldest-first.** Prices convert via `utils.CombinePrice(units, nano)`.
- **Position entry-state (`EntryATR`, `MaxFavorablePrice`) is load-bearing.** UGLD and NVTK enable `UseTrail=1`/`UseBreakeven=1`, which read `Position.EntryATR`. Without persisted state these protective/profit exits silently never fire (see the warning at `scalping/trade.go:41-45`). The state file is the primary source; reconstruct-from-API is the fallback.
- **Order type:** market (`OrderType_ORDER_TYPE_MARKET`), one order per signal, whole position. No partial fills / averaging / limit orders (YAGNI per spec).
- **Two independent flags:** `REVERSION_TRADE_ENABLED` (place real orders) and `REVERSION_NOTIFY_ENABLED` (send Telegram). Both off = dry-run to log only.
- **Dedicated account:** `REVERSION_ACCOUNT_ID`, separate from `SCALPING_ACCOUNT_ID`.
- **Production universe at write time:** `UGLD,EUTR,NVTK`. `ASTR`/`SFIN` exist in the registry but are NOT in the default universe (SFIN is "DO NOT TRADE").
- Reference spec: `docs/superpowers/specs/2026-06-25-reversion-live-runner-design.md`.

---

## File Structure

**New package `internal/service/trading_strategy/reversion/live/`:**
- `live.go` — `Service` interface, `service` struct, narrow client interfaces, constructor, options.
- `buy.go` — buy-pass orchestration.
- `manage.go` — manage-pass orchestration.
- `registry.go` — `ticker → core.Params` map and `StrategyFor`.
- `dto/run.go` — `Run{Scheduler, Mode}` and `Mode`.
- `statestore/statestore.go` — atomic JSON load/save of per-ticker entry-state.
- `sizing/sizing.go` — pure lot computation.
- `executor/executor.go` — build + place market order (or dry-run).
- `marketdata/marketdata.go` — fetch + convert + assemble `MarketData` (reuses backtest assembly).
- `notifier/notifier.go` — Telegram message rendering.
- `reconstruct/reconstruct.go` — rebuild entry-state from the API.
- `scheduler/scheduler.go` — two cron jobs (buy, manage).

**Modified existing files:**
- `internal/domain/backtest/engine.go` — extract exported `AssembleMarketData`.
- `internal/config/reversion.go` (new) + `internal/config/config.go` + `internal/app/init_config.go` — config.
- `pkg/client/grpc/operations_service_client.go` + `pkg/client/grpc/model/operation.go` — add `GetAvailableCash` and `GetInstrumentTrades`.
- `internal/service_provider/service.go` — add `GetReversionLiveService` getter.
- `internal/app/app.go` — start the two workers in `runProd` (and `runDev`).

---

## Task 1: Extract shared `MarketData` assembly from the backtest engine

**Files:**
- Modify: `internal/domain/backtest/engine.go:99-158` (refactor `Run`), add exported `AssembleMarketData`.
- Test: `internal/domain/backtest/engine_test.go` (add one test; keep existing green).

**Interfaces:**
- Produces: `func AssembleMarketData(window, daily, htf []Candle, cur time.Time) strategy.MarketData` — builds the per-bar snapshot exactly as `Run` does (minus `TodayHigh/TodayLow`, which `Run` sets separately and the reversion core ignores). The live `marketdata` unit (Task 6) consumes it.

- [ ] **Step 1: Write the failing test**

Add to `internal/domain/backtest/engine_test.go`:

```go
func TestAssembleMarketData_MatchesPerBarFields(t *testing.T) {
	base := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC) // Monday 10:00
	window := []Candle{
		{Time: base, Open: 10, High: 11, Low: 9, Close: 10, Volume: 100},
		{Time: base.Add(time.Hour), Open: 10, High: 12, Low: 10, Close: 11, Volume: 200},
	}
	daily := []Candle{
		{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Close: 8, High: 9, Low: 7},  // completed
		{Time: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), Close: 10, High: 11, Low: 9}, // current day -> excluded
	}
	htf := []Candle{
		{Time: base.Add(-4 * time.Hour), Close: 9, High: 9.5, Low: 8.5}, // closed by base+1h
		{Time: base.Add(4 * time.Hour), Close: 12, High: 13, Low: 11},   // not closed
	}
	cur := window[len(window)-1].Time

	md := AssembleMarketData(window, daily, htf, cur)

	if md.Price != 11 {
		t.Fatalf("Price = %v, want 11", md.Price)
	}
	if len(md.Closes) != 2 || md.Closes[1] != 11 {
		t.Fatalf("Closes = %v, want last 11", md.Closes)
	}
	if len(md.DailyCloses) != 1 || md.DailyCloses[0] != 8 {
		t.Fatalf("DailyCloses = %v, want [8]", md.DailyCloses)
	}
	if len(md.HTFCloses) != 1 || md.HTFCloses[0] != 9 {
		t.Fatalf("HTFCloses = %v, want [9]", md.HTFCloses)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestAssembleMarketData_MatchesPerBarFields -v`
Expected: FAIL — `undefined: AssembleMarketData`.

- [ ] **Step 3: Add `AssembleMarketData` and refactor `Run`/`Trace` to use it**

In `internal/domain/backtest/engine.go`, add after `buildMarketData` (near line 257):

```go
// AssembleMarketData builds the per-bar snapshot from an oldest-first hourly window
// plus completed-daily and 4H series, identically to Run's per-bar assembly — minus
// TodayHigh/TodayLow, which the caller sets separately and the reversion core ignores.
// cur is the open-time of the current (latest) bar; it anchors the no-lookahead
// completeness test for daily and 4H series.
func AssembleMarketData(window, daily, htf []Candle, cur time.Time) strategy.MarketData {
	md := buildMarketData(window)
	md.DailyCloses = visibleDailyCloses(daily, cur, mskLoc)
	md.DailyHighs, md.DailyLows = visibleDailyHighsLows(daily, cur, mskLoc)
	md.HTFCloses, md.HTFHighs, md.HTFLows = visibleCompletedHTF(htf, cur, htfInterval)
	return md
}
```

Then in `Run`, replace lines 114-118 with:

```go
		md := AssembleMarketData(candles[i-l+1:i+1], dailyCandles, htfCandles, candles[i].Time)
		md.TodayHigh, md.TodayLow = todayExtent(candles, i, mskLoc)
```

And in `Trace`, replace the equivalent block (lines 178-182) the same way.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/backtest/ -v`
Expected: PASS — the new test passes and ALL existing backtest tests stay green (the refactor is behavior-preserving).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "refactor(backtest): extract AssembleMarketData for live reuse

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: `ReversionConfig`

**Files:**
- Create: `internal/config/reversion.go`
- Modify: `internal/config/config.go:1-12`, `internal/app/init_config.go:32-41`
- Test: `internal/config/reversion_test.go`

**Interfaces:**
- Produces: `config.ReversionConfig{AccountID string; Tickers []string; BuyPct float64; TradeEnabled bool; NotifyEnabled bool}` and `config.NewReversionConfig() *ReversionConfig` (with defaults). Consumed by the service_provider getter (Task 13).

- [ ] **Step 1: Write the failing test**

Create `internal/config/reversion_test.go`:

```go
package config

import "testing"

func TestNewReversionConfig_Defaults(t *testing.T) {
	c := NewReversionConfig()
	if len(c.Tickers) != 3 || c.Tickers[0] != "UGLD" || c.Tickers[1] != "EUTR" || c.Tickers[2] != "NVTK" {
		t.Fatalf("default Tickers = %v, want [UGLD EUTR NVTK]", c.Tickers)
	}
	if c.BuyPct != 10 {
		t.Fatalf("default BuyPct = %v, want 10", c.BuyPct)
	}
	if c.TradeEnabled {
		t.Fatalf("TradeEnabled default = true, want false (safe default)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestNewReversionConfig_Defaults -v`
Expected: FAIL — `undefined: NewReversionConfig`.

- [ ] **Step 3: Create `internal/config/reversion.go`**

```go
package config

// ReversionConfig configures the live reversion runner. Trade and notify are
// independent: both off = dry-run to log only.
type ReversionConfig struct {
	AccountID     string   `config:"REVERSION_ACCOUNT_ID,required,backend=env"`
	Tickers       []string `config:"REVERSION_TICKERS,backend=env"`
	BuyPct        float64  `config:"REVERSION_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"REVERSION_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"REVERSION_NOTIFY_ENABLED,backend=env"`
}

// NewReversionConfig returns the config pre-seeded with safe defaults. confita
// overrides any field whose env var is set; unset fields keep these values.
// TradeEnabled defaults to false so a missing flag never places real orders.
func NewReversionConfig() *ReversionConfig {
	return &ReversionConfig{
		Tickers: []string{"UGLD", "EUTR", "NVTK"},
		BuyPct:  10,
	}
}
```

- [ ] **Step 4: Register on `Config` and load in `init_config.go`**

In `internal/config/config.go`, add the field to the `Config` struct:

```go
	Reversion      *ReversionConfig
```

In `internal/app/init_config.go`, add to the `cfg` literal (after `Scalping:`):

```go
		Reversion:      config.NewReversionConfig(),
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/ -v && go build ./...`
Expected: PASS and a clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/config/reversion.go internal/config/reversion_test.go internal/config/config.go internal/app/init_config.go
git commit -m "feat(config): add ReversionConfig for the live runner

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: `statestore` — atomic per-ticker entry-state

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/statestore/statestore.go`
- Test: `internal/service/trading_strategy/reversion/live/statestore/statestore_test.go`

**Interfaces:**
- Produces:
  - `type Entry struct { Ticker string; EntryTime time.Time; EntryPrice, EntryATR, MaxFav float64; Quantity int64 }`
  - `type Store interface { Load() (map[string]Entry, error); Save(map[string]Entry) error }` (map keyed by ticker)
  - `func New(path string) *FileStore` and `*FileStore` implements `Store`.
  Consumed by the service (Tasks 11) and reconstruct (Task 10).

- [ ] **Step 1: Write the failing test**

Create `statestore_test.go`:

```go
package statestore

import (
	"path/filepath"
	"testing"
	"time"
)

func TestFileStore_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reversion_acc.json")
	s := New(path)

	in := map[string]Entry{
		"UGLD": {Ticker: "UGLD", EntryTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
			EntryPrice: 100.5, EntryATR: 2.3, MaxFav: 105.0, Quantity: 10},
	}
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := out["UGLD"]
	if got.EntryPrice != 100.5 || got.EntryATR != 2.3 || got.MaxFav != 105.0 || got.Quantity != 10 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestFileStore_LoadMissingFileIsEmpty(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "does_not_exist.json"))
	out, err := s.Load()
	if err != nil {
		t.Fatalf("Load missing file should not error, got %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("Load missing file = %v, want empty", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/live/statestore/ -v`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement `statestore.go`**

```go
// Package statestore persists per-ticker reversion entry-state (EntryATR,
// MaxFavorablePrice, etc.) that the broker does not return. The state file is the
// primary source for the protective/profit exits; reconstruct is the fallback.
package statestore

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Entry is the persisted entry-state for one open position.
type Entry struct {
	Ticker     string    `json:"ticker"`
	EntryTime  time.Time `json:"entryTime"`
	EntryPrice float64   `json:"entryPrice"`
	EntryATR   float64   `json:"entryATR"`
	MaxFav     float64   `json:"maxFav"`
	Quantity   int64     `json:"quantity"`
}

// Store loads and saves the full per-ticker entry-state map.
type Store interface {
	Load() (map[string]Entry, error)
	Save(map[string]Entry) error
}

// FileStore persists the map as a single JSON file with atomic writes.
type FileStore struct {
	path string
}

// New returns a FileStore backed by path.
func New(path string) *FileStore { return &FileStore{path: path} }

// Load reads the state map; a missing file yields an empty map (not an error).
func (s *FileStore) Load() (map[string]Entry, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]Entry{}, nil
		}
		return nil, err
	}
	out := map[string]Entry{}
	if len(b) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Save writes the map atomically: marshal to a temp file in the same dir, then rename.
func (s *FileStore) Save(m map[string]Entry) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), filepath.Base(s.path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, s.path)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/live/statestore/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/statestore/
git commit -m "feat(reversion/live): atomic per-ticker entry-state store

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: `sizing` — pure lot computation

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/sizing/sizing.go`
- Test: `internal/service/trading_strategy/reversion/live/sizing/sizing_test.go`

**Interfaces:**
- Produces: `func Lots(buyPct, accountValue, cash, price float64, lot int32) (lots int64, ok bool, reason string)` — sizes by `buyPct% × accountValue`, capped by affordable cash; `ok=false` (with a human reason) when the budget buys < 1 lot or cash cannot cover one lot. Consumed by the buy-pass (Task 11).

- [ ] **Step 1: Write the failing test**

Create `sizing_test.go`:

```go
package sizing

import "testing"

func TestLots(t *testing.T) {
	tests := []struct {
		name                                   string
		buyPct, accountValue, cash, price      float64
		lot                                    int32
		wantLots                               int64
		wantOK                                 bool
	}{
		// 10% of 100000 = 10000 budget; price 100, lot 1 -> 100 shares = 100 lots; cash ample.
		{"basic", 10, 100000, 100000, 100, 1, 100, true},
		// lot size 10: 10000 budget / (100*10=1000) = 10 lots.
		{"lot10", 10, 100000, 100000, 100, 10, 10, true},
		// budget buys < 1 lot -> skip.
		{"sub_lot_budget", 1, 1000, 1000, 100, 10, 0, false},
		// budget allows 10 lots but cash only covers 3.
		{"cash_capped", 10, 100000, 3500, 100, 10, 3, true},
		// cash cannot cover even one lot -> skip.
		{"insufficient_cash", 10, 100000, 500, 100, 10, 0, false},
		// zero/garbage price -> skip, no panic.
		{"zero_price", 10, 100000, 100000, 0, 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lots, ok, reason := Lots(tt.buyPct, tt.accountValue, tt.cash, tt.price, tt.lot)
			if lots != tt.wantLots || ok != tt.wantOK {
				t.Fatalf("Lots(%+v) = (%d, %v, %q), want (%d, %v)", tt, lots, ok, reason, tt.wantLots, tt.wantOK)
			}
			if !ok && reason == "" {
				t.Fatalf("skip case must carry a reason")
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/live/sizing/ -v`
Expected: FAIL — `undefined: Lots`.

- [ ] **Step 3: Implement `sizing.go`**

```go
// Package sizing computes whole-lot order quantities for the reversion runner from a
// percentage of total account value, capped by available cash.
package sizing

import (
	"fmt"
	"math"
)

// Lots returns the number of lots to buy. It sizes by buyPct% of accountValue, then
// caps by what cash can afford. ok is false (with a human-readable reason) when the
// budget buys fewer than one lot or cash cannot cover a single lot. Inputs that make
// a lot priceless (price<=0, lot<=0) also yield ok=false.
func Lots(buyPct, accountValue, cash, price float64, lot int32) (int64, bool, string) {
	if price <= 0 || lot <= 0 {
		return 0, false, fmt.Sprintf("некорректная цена/лот (price=%.4f, lot=%d)", price, lot)
	}
	lotCost := price * float64(lot)
	budget := buyPct / 100 * accountValue
	lots := int64(math.Floor(budget / lotCost))
	if lots <= 0 {
		return 0, false, fmt.Sprintf("бюджета %.2f не хватает на 1 лот (%.2f)", budget, lotCost)
	}
	affordable := int64(math.Floor(cash / lotCost))
	if affordable <= 0 {
		return 0, false, fmt.Sprintf("кэша %.2f не хватает на 1 лот (%.2f)", cash, lotCost)
	}
	if lots > affordable {
		lots = affordable
	}
	return lots, true, ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/live/sizing/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/sizing/
git commit -m "feat(reversion/live): pct-of-account lot sizing

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: `registry` — ticker → calibrated strategy

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/registry.go`
- Test: `internal/service/trading_strategy/reversion/live/registry_test.go`

**Interfaces:**
- Produces:
  - `func StrategyFor(ticker string) (*core.Strategy, bool)` — returns the calibrated `*core.Strategy` for a known ticker.
  - `func MaxHTFTrendEMA(tickers []string) int` — the largest `HTFTrendEMA` across the given tickers (0 if none), so the marketdata unit knows whether/how much 4H to fetch.
  Consumed by the service (Task 11) and marketdata wiring.

- [ ] **Step 1: Write the failing test**

Create `registry_test.go`:

```go
package live

import "testing"

func TestStrategyFor(t *testing.T) {
	st, ok := StrategyFor("UGLD")
	if !ok {
		t.Fatal("UGLD should be registered")
	}
	if st.Ticker() != "UGLD" {
		t.Fatalf("Ticker() = %q, want UGLD", st.Ticker())
	}
	if _, ok := StrategyFor("NOSUCH"); ok {
		t.Fatal("unknown ticker must return ok=false")
	}
}

func TestMaxHTFTrendEMA(t *testing.T) {
	// NVTK has HTFTrendEMA=150; UGLD/EUTR have 0.
	if got := MaxHTFTrendEMA([]string{"UGLD", "EUTR"}); got != 0 {
		t.Fatalf("MaxHTFTrendEMA(UGLD,EUTR) = %d, want 0", got)
	}
	if got := MaxHTFTrendEMA([]string{"UGLD", "NVTK"}); got != 150 {
		t.Fatalf("MaxHTFTrendEMA(UGLD,NVTK) = %d, want 150", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/live/ -run 'TestStrategyFor|TestMaxHTFTrendEMA' -v`
Expected: FAIL — `undefined: StrategyFor`.

- [ ] **Step 3: Implement `registry.go`**

```go
package live

import (
	"tinvest/internal/service/trading_strategy/reversion/strategy/astr"
	"tinvest/internal/service/trading_strategy/reversion/strategy/core"
	"tinvest/internal/service/trading_strategy/reversion/strategy/eutr"
	"tinvest/internal/service/trading_strategy/reversion/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/reversion/strategy/sfin"
	"tinvest/internal/service/trading_strategy/reversion/strategy/ugld"
)

// paramsByTicker maps every reversion ticker the runner knows to its calibrated
// params. The configured universe (env) selects which of these actually trade; SFIN
// is registered for completeness but is "DO NOT TRADE" and must not be in the universe.
var paramsByTicker = map[string]core.Params{
	ugld.Ticker: ugld.DefaultParams(),
	eutr.Ticker: eutr.DefaultParams(),
	nvtk.Ticker: nvtk.DefaultParams(),
	astr.Ticker: astr.DefaultParams(),
	sfin.Ticker: sfin.DefaultParams(),
}

// StrategyFor returns the calibrated strategy for a known ticker, ok=false otherwise.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := paramsByTicker[ticker]
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}

// MaxHTFTrendEMA returns the largest HTFTrendEMA across the given tickers (0 = no 4H
// filter needed by any). Unknown tickers contribute 0.
func MaxHTFTrendEMA(tickers []string) int {
	m := 0
	for _, t := range tickers {
		if p, ok := paramsByTicker[t]; ok && p.HTFTrendEMA > m {
			m = p.HTFTrendEMA
		}
	}
	return m
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/live/ -run 'TestStrategyFor|TestMaxHTFTrendEMA' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/registry.go internal/service/trading_strategy/reversion/live/registry_test.go
git commit -m "feat(reversion/live): ticker->params registry

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: `marketdata` — fetch, convert, assemble (parity-tested)

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/marketdata/marketdata.go`
- Test: `internal/service/trading_strategy/reversion/live/marketdata/marketdata_test.go`

**Interfaces:**
- Consumes (narrow client): `type CandleClient interface { GetCandles(ctx context.Context, instrumentUid *string, interval int32, from, to *timestamppb.Timestamp, limit *int32, withHoliday bool) ([]*imodel.CandleItemTechAnalyse, error) }`
- Produces:
  - `func ToCandles(in []*imodel.CandleItemTechAnalyse, completedOnly bool) []backtest.Candle` — convert + optionally drop the still-forming last bar.
  - `func Assemble(ctx context.Context, c CandleClient, instrumentID string, lookbackBars, htfEMAPeriod int, now time.Time) (strategy.MarketData, error)` — fetches hourly (and 4H when `htfEMAPeriod>0`), trims to completed bars, and calls `backtest.AssembleMarketData`. `Position` is left nil; the caller sets it. Returns an error when there are fewer than `lookbackBars` completed hourly candles.
  Consumed by buy-pass and manage-pass (Task 11).

- [ ] **Step 1: Write the failing parity test**

Create `marketdata_test.go`. This is the **key parity test**: the same OHLCV series, expressed as both API candles and domain candles, must yield identical snapshots through `Assemble` and `backtest.AssembleMarketData`.

```go
package marketdata

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	imodel "tinvest/internal/model"
)

// fakeCandleClient returns a fixed hourly series and an empty 4H series.
type fakeCandleClient struct {
	hourly []*imodel.CandleItemTechAnalyse
}

func (f *fakeCandleClient) GetCandles(_ context.Context, _ *string, interval int32,
	_, _ *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	// Hour1 == 4 per enum.ToNumberInvestApi mapping; anything else is the 4H request.
	if interval == 4 {
		return f.hourly, nil
	}
	return nil, nil
}

func apiCandle(ts time.Time, o, h, l, c float64, v int64, complete bool) *imodel.CandleItemTechAnalyse {
	q := func(f float64) imodel.Quotation {
		units := int64(f)
		nano := int32((f - float64(units)) * 1e9)
		return imodel.Quotation{Units: units, Nano: nano}
	}
	return &imodel.CandleItemTechAnalyse{
		Time: ts, Open: q(o), High: q(h), Low: q(l), Close: q(c), Volume: v, IsComplete: complete,
	}
}

func TestAssemble_ParityWithBacktest(t *testing.T) {
	base := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	var api []*imodel.CandleItemTechAnalyse
	var dom []backtest.Candle
	for i := 0; i < 60; i++ {
		ts := base.Add(time.Duration(i) * time.Hour)
		o, h, l, c, v := 100.0+float64(i), 101.0+float64(i), 99.0+float64(i), 100.5+float64(i), int64(1000+i)
		api = append(api, apiCandle(ts, o, h, l, c, v, true))
		dom = append(dom, backtest.Candle{Time: ts, Open: o, High: h, Low: l, Close: c, Volume: v})
	}
	// Append a still-forming bar that the live path must drop.
	ts := base.Add(60 * time.Hour)
	api = append(api, apiCandle(ts, 999, 999, 999, 999, 9, false))

	const lookback = 50
	c := &fakeCandleClient{hourly: api}
	live, err := Assemble(context.Background(), c, "uid", lookback, 0, ts.Add(time.Hour))
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Backtest reference: last `lookback` completed domain candles, cur = last bar open.
	window := dom[len(dom)-lookback:]
	want := backtest.AssembleMarketData(window, nil, nil, window[len(window)-1].Time)

	if diff := cmp.Diff(want, live); diff != "" {
		t.Fatalf("snapshot parity mismatch (-backtest +live):\n%s", diff)
	}
}

func TestAssemble_ErrorsOnInsufficientCandles(t *testing.T) {
	c := &fakeCandleClient{hourly: []*imodel.CandleItemTechAnalyse{
		apiCandle(time.Now(), 1, 1, 1, 1, 1, true),
	}}
	if _, err := Assemble(context.Background(), c, "uid", 50, 0, time.Now()); err == nil {
		t.Fatal("expected error when completed candles < lookback")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/live/marketdata/ -v`
Expected: FAIL — `undefined: Assemble`.

- [ ] **Step 3: Implement `marketdata.go`**

Note the warm-up sizing: hourly trading bars are sparse in calendar time (markets closed nights/weekends), so request a generous calendar window and trim to the last `lookbackBars` completed bars. `barsPerCalendarDayHourly`/`...HTF` are conservative lower bounds on MOEX trading bars per calendar day.

```go
// Package marketdata assembles the live reversion MarketData snapshot from Tinkoff
// candles, reusing the backtest's AssembleMarketData so live and backtest build
// identical inputs. Only hourly + 4H are fetched: the reversion core reads neither
// daily series nor TodayHigh/Low, and computes ATR on the hourly window.
package marketdata

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
)

// CandleClient is the slice of the market-data client the assembler needs.
type CandleClient interface {
	GetCandles(ctx context.Context, instrumentUid *string, interval int32,
		from, to *timestamppb.Timestamp, limit *int32, withHoliday bool) ([]*imodel.CandleItemTechAnalyse, error)
}

// Conservative lower bounds on completed trading bars per calendar day on MOEX, used
// to size the fetch window so it always contains enough warm-up bars after weekends
// and holidays are excluded. Intentionally pessimistic: over-fetching is cheap, the
// snapshot trims to the exact lookback.
const (
	barsPerCalendarDayHourly = 6
	barsPerCalendarDayHTF    = 2
	warmupBufferFactor       = 3
)

// ToCandles converts oldest-first API candles to domain candles. When completedOnly is
// true the still-forming trailing bar (IsComplete=false) is dropped.
func ToCandles(in []*imodel.CandleItemTechAnalyse, completedOnly bool) []backtest.Candle {
	out := make([]backtest.Candle, 0, len(in))
	for _, c := range in {
		if completedOnly && !c.IsComplete {
			continue
		}
		out = append(out, backtest.Candle{
			Time:   c.Time,
			Open:   utils.CombinePrice(c.Open.Units, c.Open.Nano),
			High:   utils.CombinePrice(c.High.Units, c.High.Nano),
			Low:    utils.CombinePrice(c.Low.Units, c.Low.Nano),
			Close:  utils.CombinePrice(c.Close.Units, c.Close.Nano),
			Volume: c.Volume,
		})
	}
	return out
}

// fetchCompleted pulls `bars` completed candles of one interval ending at `now`,
// returning the last `bars` (oldest-first). It requests a calendar window generous
// enough to survive non-trading hours.
func fetchCompleted(ctx context.Context, c CandleClient, instrumentID string,
	interval enum.Interval, bars, barsPerDay int, now time.Time) ([]backtest.Candle, error) {
	if bars <= 0 {
		return nil, nil
	}
	calendarDays := bars/barsPerDay + 1
	calendarDays *= warmupBufferFactor
	from := now.AddDate(0, 0, -calendarDays)
	limit := int32(bars * warmupBufferFactor * 2)
	raw, err := c.GetCandles(ctx, &instrumentID, interval.ToNumberInvestApi(),
		timestamppb.New(from), timestamppb.New(now), &limit, true)
	if err != nil {
		return nil, err
	}
	completed := ToCandles(raw, true)
	if len(completed) > bars {
		completed = completed[len(completed)-bars:]
	}
	return completed, nil
}

// Assemble builds the MarketData snapshot. lookbackBars is the hourly window size
// (Strategy.Lookback()); htfEMAPeriod>0 triggers a 4H fetch warmed to that period.
// Position is left nil for the caller to set.
func Assemble(ctx context.Context, c CandleClient, instrumentID string,
	lookbackBars, htfEMAPeriod int, now time.Time) (strategy.MarketData, error) {
	window, err := fetchCompleted(ctx, c, instrumentID, enum.Hour1, lookbackBars, barsPerCalendarDayHourly, now)
	if err != nil {
		return strategy.MarketData{}, fmt.Errorf("reversion marketdata: hourly candles: %w", err)
	}
	if len(window) < lookbackBars {
		return strategy.MarketData{}, fmt.Errorf("reversion marketdata: %d completed hourly candles < lookback %d", len(window), lookbackBars)
	}

	var htf []backtest.Candle
	if htfEMAPeriod > 0 {
		// Warm the 4H EMA with a comfortable margin over the period itself.
		htf, err = fetchCompleted(ctx, c, instrumentID, enum.Hour4, htfEMAPeriod+20, barsPerCalendarDayHTF, now)
		if err != nil {
			return strategy.MarketData{}, fmt.Errorf("reversion marketdata: 4H candles: %w", err)
		}
	}

	cur := window[len(window)-1].Time
	return backtest.AssembleMarketData(window, nil, htf, cur), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/live/marketdata/ -v`
Expected: PASS — parity holds; insufficient-candle case errors.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/marketdata/
git commit -m "feat(reversion/live): live MarketData assembly with backtest parity test

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: `executor` — market order placement / dry-run

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/executor/executor.go`
- Test: `internal/service/trading_strategy/reversion/live/executor/executor_test.go`

**Interfaces:**
- Consumes (narrow client): `type OrdersClient interface { PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error) }`
- Produces:
  - `type Result struct { Placed bool; FilledLots int64; FillPrice float64 }`
  - `func New(c OrdersClient, accountID string, tradeEnabled bool) *Executor`
  - `func (e *Executor) Buy(ctx, instrumentID string, lots int64) (Result, error)`
  - `func (e *Executor) Sell(ctx, instrumentID string, lots int64) (Result, error)`
  When `tradeEnabled==false`, no order is placed and `Result{Placed:false}` is returned (caller falls back to the signal price/qty). Consumed by the service (Task 11).

- [ ] **Step 1: Write the failing test**

Create `executor_test.go`:

```go
package executor

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
)

type fakeOrders struct {
	last *investapi.PostOrderRequest
	resp *investapi.PostOrderResponse
}

func (f *fakeOrders) PostOrder(_ context.Context, in *investapi.PostOrderRequest, _ ...grpc.CallOption) (*investapi.PostOrderResponse, error) {
	f.last = in
	return f.resp, nil
}

func TestBuy_PlacesMarketOrder(t *testing.T) {
	f := &fakeOrders{resp: &investapi.PostOrderResponse{
		LotsExecuted:       7,
		ExecutedOrderPrice: &investapi.MoneyValue{Units: 101, Nano: 500000000},
	}}
	e := New(f, "acc-1", true)

	res, err := e.Buy(context.Background(), "uid-1", 7)
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if !res.Placed || res.FilledLots != 7 || res.FillPrice != 101.5 {
		t.Fatalf("Result = %+v, want Placed lots=7 price=101.5", res)
	}
	if f.last.Direction != investapi.OrderDirection_ORDER_DIRECTION_BUY {
		t.Fatalf("direction = %v, want BUY", f.last.Direction)
	}
	if f.last.OrderType != investapi.OrderType_ORDER_TYPE_MARKET {
		t.Fatalf("order type = %v, want MARKET", f.last.OrderType)
	}
	if f.last.Quantity != 7 || f.last.InstrumentId != "uid-1" || f.last.AccountId != "acc-1" {
		t.Fatalf("request fields wrong: %+v", f.last)
	}
	if len(f.last.OrderId) == 0 || len(f.last.OrderId) > 36 {
		t.Fatalf("OrderId must be a non-empty UID <=36 chars, got %q", f.last.OrderId)
	}
}

func TestSell_Direction(t *testing.T) {
	f := &fakeOrders{resp: &investapi.PostOrderResponse{LotsExecuted: 3}}
	e := New(f, "acc-1", true)
	if _, err := e.Sell(context.Background(), "uid-1", 3); err != nil {
		t.Fatalf("Sell: %v", err)
	}
	if f.last.Direction != investapi.OrderDirection_ORDER_DIRECTION_SELL {
		t.Fatalf("direction = %v, want SELL", f.last.Direction)
	}
}

func TestBuy_DryRunPlacesNoOrder(t *testing.T) {
	f := &fakeOrders{}
	e := New(f, "acc-1", false) // trade disabled
	res, err := e.Buy(context.Background(), "uid-1", 5)
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if res.Placed {
		t.Fatal("dry-run must not place an order")
	}
	if f.last != nil {
		t.Fatal("PostOrder must not be called in dry-run")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/live/executor/ -v`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Add the `google/uuid` dependency**

```bash
go get github.com/google/uuid@v1.6.0
```

- [ ] **Step 4: Implement `executor.go`**

```go
// Package executor places (or dry-runs) whole-position market orders for the
// reversion runner. Each order carries a fresh UUID order_id for idempotency.
package executor

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/utils"
)

// OrdersClient is the slice of the orders client the executor needs.
type OrdersClient interface {
	PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error)
}

// Result reports the effect of an order attempt. When Placed is false (dry-run) the
// caller falls back to the signal's price and the requested quantity.
type Result struct {
	Placed     bool
	FilledLots int64
	FillPrice  float64
}

// Executor builds and places market orders, or dry-runs when tradeEnabled is false.
type Executor struct {
	client       OrdersClient
	accountID    string
	tradeEnabled bool
}

// New returns an Executor bound to an account.
func New(c OrdersClient, accountID string, tradeEnabled bool) *Executor {
	return &Executor{client: c, accountID: accountID, tradeEnabled: tradeEnabled}
}

// Buy places a market BUY of `lots` lots (or dry-runs).
func (e *Executor) Buy(ctx context.Context, instrumentID string, lots int64) (Result, error) {
	return e.place(ctx, instrumentID, lots, investapi.OrderDirection_ORDER_DIRECTION_BUY)
}

// Sell places a market SELL of `lots` lots (or dry-runs).
func (e *Executor) Sell(ctx context.Context, instrumentID string, lots int64) (Result, error) {
	return e.place(ctx, instrumentID, lots, investapi.OrderDirection_ORDER_DIRECTION_SELL)
}

func (e *Executor) place(ctx context.Context, instrumentID string, lots int64, dir investapi.OrderDirection) (Result, error) {
	if !e.tradeEnabled {
		return Result{Placed: false}, nil
	}
	req := &investapi.PostOrderRequest{
		InstrumentId: instrumentID,
		Quantity:     lots,
		Direction:    dir,
		AccountId:    e.accountID,
		OrderType:    investapi.OrderType_ORDER_TYPE_MARKET,
		OrderId:      uuid.NewString(),
	}
	resp, err := e.client.PostOrder(ctx, req)
	if err != nil {
		return Result{}, err
	}
	res := Result{Placed: true, FilledLots: resp.GetLotsExecuted()}
	if p := resp.GetExecutedOrderPrice(); p != nil {
		res.FillPrice = utils.CombinePrice(p.GetUnits(), int32(p.GetNano()))
	}
	return res, nil
}
```

Note: `MoneyValue.Nano` is `int32` in this proto; `utils.CombinePrice` takes `(int64, int32)`. If the generated `GetNano()` returns `int32`, drop the cast — match the actual signature when implementing.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/live/executor/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/executor/ go.mod go.sum
git commit -m "feat(reversion/live): market-order executor with dry-run

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: `notifier` — Telegram messages

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/notifier/notifier.go`
- Test: `internal/service/trading_strategy/reversion/live/notifier/notifier_test.go`

**Interfaces:**
- Produces pure render functions (no client) so they're trivially testable:
  - `func Entry(ticker string, price float64, lots int64, qty int64, paper bool) string`
  - `func Exit(ticker, reason string, price float64, qty int64, paper bool) string`
  - `func Skip(ticker, reason string) string`
  - `func Alert(ticker, message string) string`
  Consumed by the service (Task 11), which sends the returned string via `telegram.Client.SendMessage` only when `NotifyEnabled`.

- [ ] **Step 1: Write the failing test**

Create `notifier_test.go`:

```go
package notifier

import (
	"strings"
	"testing"
)

func TestEntry(t *testing.T) {
	msg := Entry("UGLD", 100.5, 10, 100, false)
	for _, want := range []string{"UGLD", "100.5", "🟢"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Entry msg %q missing %q", msg, want)
		}
	}
	if !strings.Contains(Entry("UGLD", 1, 1, 1, true), "БУМАЖНАЯ") {
		t.Fatal("paper-mode entry must be flagged")
	}
}

func TestExitAndSkipAndAlert(t *testing.T) {
	if !strings.Contains(Exit("NVTK", "OB", 200, 50, false), "NVTK") {
		t.Fatal("Exit must name the ticker")
	}
	if !strings.Contains(Skip("EUTR", "кэша не хватает"), "кэша не хватает") {
		t.Fatal("Skip must carry the reason")
	}
	if !strings.Contains(Alert("UGLD", "стейт потерян"), "⚠️") {
		t.Fatal("Alert must be visibly flagged")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/live/notifier/ -v`
Expected: FAIL — `undefined: Entry`.

- [ ] **Step 3: Implement `notifier.go`**

```go
// Package notifier renders Telegram messages for the reversion runner. Functions are
// pure; the caller sends the string only when NotifyEnabled.
package notifier

import "fmt"

func paperTag(paper bool) string {
	if paper {
		return " <i>(БУМАЖНАЯ сделка, ордер не выставлен)</i>"
	}
	return ""
}

// Entry renders a buy notification.
func Entry(ticker string, price float64, lots, qty int64, paper bool) string {
	return fmt.Sprintf("🟢 <b>Вход %s</b>%s\n  Цена: %.4f | Лотов: %d | Штук: %d",
		ticker, paperTag(paper), price, lots, qty)
}

// Exit renders a sell notification with the exit reason code (OB/RSI50/BE/TRAIL/...).
func Exit(ticker, reason string, price float64, qty int64, paper bool) string {
	return fmt.Sprintf("🔴 <b>Выход %s</b> [%s]%s\n  Цена: %.4f | Штук: %d",
		ticker, reason, paperTag(paper), price, qty)
}

// Skip renders a skipped-entry notification (e.g. sub-lot budget, insufficient cash).
func Skip(ticker, reason string) string {
	return fmt.Sprintf("⏭️ <b>Пропуск %s</b>\n  %s", ticker, reason)
}

// Alert renders an operational alert (e.g. state reconstructed, order rejected).
func Alert(ticker, message string) string {
	return fmt.Sprintf("⚠️ <b>Reversion %s</b>\n  %s", ticker, message)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/live/notifier/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/notifier/
git commit -m "feat(reversion/live): Telegram message renderers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 9: Operations client — `GetAvailableCash` + `GetInstrumentTrades`

**Files:**
- Modify: `pkg/client/grpc/operations_service_client.go`
- Modify: `pkg/client/grpc/model/operation.go` (add `Trade` model)
- Test: `pkg/client/grpc/operations_trade_test.go`

**Interfaces:**
- Produces (added to `OperationsServiceClient`):
  - `GetAvailableCash(ctx context.Context, accountID string) (float64, error)` — RUB cash from `PortfolioResponse.TotalAmountCurrencies`.
  - `GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]model.Trade, error)` — per-trade fills (date, price, quantity, direction) for one instrument, paginated via `GetOperationsByCursor`.
  - `type model.Trade struct { Date time.Time; Price float64; Quantity int64; IsBuy bool }`
  Consumed by the buy-pass sizing (cash) and reconstruct (Task 10).

- [ ] **Step 1: Add the `Trade` model**

In `pkg/client/grpc/model/operation.go`, add:

```go
// Trade is one executed fill of an instrument (from GetOperationsByCursor trades).
type Trade struct {
	Date     time.Time
	Price    float64
	Quantity int64
	IsBuy    bool
}
```

(Ensure `time` is imported in that file.)

- [ ] **Step 2: Write the failing test**

Create `pkg/client/grpc/operations_trade_test.go`. Because the real client wraps a gRPC stub, the test targets a small **pure converter** that maps cursor operation items to `[]model.Trade`; extract that converter so it is unit-testable. Define the converter in the same package.

```go
package grpc

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	investapi "tinvest/internal/pb/v1"
)

func TestTradesFromCursorItems_FiltersByInstrumentAndDirection(t *testing.T) {
	ts := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	items := []*investapi.OperationItem{
		{
			InstrumentUid: "uid-1",
			Type:          investapi.OperationType_OPERATION_TYPE_BUY,
			Date:          timestamppb.New(ts),
			TradesInfo: &investapi.OperationItemTrades{Trades: []*investapi.OperationItemTrade{
				{Num: "t1", DateTime: timestamppb.New(ts), Quantity: 5,
					Price: &investapi.MoneyValue{Units: 100, Nano: 250000000}},
			}},
		},
		{InstrumentUid: "uid-OTHER", Type: investapi.OperationType_OPERATION_TYPE_BUY}, // filtered out
	}

	got := tradesFromCursorItems(items, "uid-1")
	if len(got) != 1 {
		t.Fatalf("got %d trades, want 1", len(got))
	}
	if !got[0].IsBuy || got[0].Quantity != 5 || got[0].Price != 100.25 {
		t.Fatalf("trade = %+v, want buy qty5 price100.25", got[0])
	}
}
```

> Implementer note: confirm the exact generated names for the cursor operation item, its trades wrapper, and the per-trade message in `internal/pb/v1/operations.pb.go` (e.g. `OperationItem`, `OperationItemTrades`, `OperationItemTrade`, fields `TradesInfo`, `InstrumentUid`, `Type`, `DateTime`, `Price`, `Quantity`). Adjust the test and converter to match the real symbols; the shape above reflects the Tinkoff `GetOperationsByCursor` schema.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./pkg/client/grpc/ -run TestTradesFromCursorItems -v`
Expected: FAIL — `undefined: tradesFromCursorItems`.

- [ ] **Step 4: Implement the converter + the two methods**

Add the converter (pure) and methods to `operations_service_client.go`, and the methods to the `OperationsServiceClient` interface:

```go
// tradesFromCursorItems flattens cursor operation items into per-instrument Trades.
func tradesFromCursorItems(items []*investapi.OperationItem, instrumentID string) []model.Trade {
	var out []model.Trade
	for _, it := range items {
		if it.GetInstrumentUid() != instrumentID {
			continue
		}
		isBuy := it.GetType() == investapi.OperationType_OPERATION_TYPE_BUY
		isSell := it.GetType() == investapi.OperationType_OPERATION_TYPE_SELL
		if !isBuy && !isSell {
			continue
		}
		for _, tr := range it.GetTradesInfo().GetTrades() {
			price := 0.0
			if p := tr.GetPrice(); p != nil {
				price = float64(p.GetUnits()) + float64(p.GetNano())/1e9
			}
			out = append(out, model.Trade{
				Date:     tr.GetDateTime().AsTime(),
				Price:    price,
				Quantity: tr.GetQuantity(),
				IsBuy:    isBuy,
			})
		}
	}
	return out
}

// GetAvailableCash returns RUB cash (TotalAmountCurrencies) for the account.
func (o *operationsServiceClient) GetAvailableCash(ctx context.Context, accountID string) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cur := investapi.PortfolioRequest_RUB
	resp, err := o.operationApi.GetPortfolio(ctx, &investapi.PortfolioRequest{
		AccountId: accountID, Currency: &cur,
	}, NewRPCCredential(o.auth))
	if err != nil {
		return 0, err
	}
	c := resp.GetTotalAmountCurrencies()
	if c == nil {
		return 0, nil
	}
	return float64(c.GetUnits()) + float64(c.GetNano())/1e9, nil
}

// GetInstrumentTrades returns per-trade fills for one instrument over [from,to].
func (o *operationsServiceClient) GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]model.Trade, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	state := investapi.OperationState_OPERATION_STATE_EXECUTED
	limit := int32(1000)
	var out []model.Trade
	var cursor string
	for {
		req := &investapi.GetOperationsByCursorRequest{
			AccountId: accountID,
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
			State:     &state,
			Limit:     &limit,
		}
		if cursor != "" {
			req.Cursor = &cursor
		}
		resp, err := o.operationApi.GetOperationsByCursor(ctx, req, NewRPCCredential(o.auth))
		if err != nil {
			return nil, err
		}
		out = append(out, tradesFromCursorItems(resp.GetItems(), instrumentID)...)
		if !resp.GetHasNext() || resp.GetNextCursor() == "" {
			break
		}
		cursor = resp.GetNextCursor()
	}
	return out, nil
}
```

Add to the interface (lines 15-20):

```go
	GetAvailableCash(ctx context.Context, accountID string) (float64, error)
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]model.Trade, error)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/client/grpc/ -v && go build ./...`
Expected: PASS and clean build.

- [ ] **Step 6: Commit**

```bash
git add pkg/client/grpc/operations_service_client.go pkg/client/grpc/model/operation.go pkg/client/grpc/operations_trade_test.go
git commit -m "feat(grpc): expose available cash and per-instrument trade fills

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 10: `reconstruct` — rebuild entry-state from the API

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/reconstruct/reconstruct.go`
- Test: `internal/service/trading_strategy/reversion/live/reconstruct/reconstruct_test.go`

**Interfaces:**
- Consumes:
  - `type TradesClient interface { GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error) }`
  - `type CandleClient` (from Task 6 `marketdata.CandleClient`) for recomputing ATR + maxFav.
- Produces: `func Entry(ctx context.Context, tc TradesClient, cc marketdata.CandleClient, accountID, instrumentID, ticker string, purchasePrice float64, atrPeriod, lookbackBars int, now time.Time) (statestore.Entry, error)`.
  Strategy: `EntryPrice = purchasePrice` (broker-supplied avg); `EntryTime` = the most recent BUY trade time; `EntryATR` = ATR over the hourly window ending at the entry bar; `MaxFav` = max(EntryPrice, max hourly close since EntryTime). Consumed by manage-pass (Task 11).

- [ ] **Step 1: Write the failing test**

Create `reconstruct_test.go`. Fakes reuse the same shapes as Task 6; assert `EntryTime`/`EntryPrice` and that `MaxFav >= EntryPrice`.

```go
package reconstruct

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	imodel "tinvest/internal/model"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

type fakeTrades struct{ trades []grpcmodel.Trade }

func (f *fakeTrades) GetInstrumentTrades(_ context.Context, _, _ string, _, _ time.Time) ([]grpcmodel.Trade, error) {
	return f.trades, nil
}

type fakeCandles struct{ candles []*imodel.CandleItemTechAnalyse }

func (f *fakeCandles) GetCandles(_ context.Context, _ *string, _ int32, _, _ *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	return f.candles, nil
}

func q(f float64) imodel.Quotation { return imodel.Quotation{Units: int64(f)} }

func TestEntry_RebuildsFromMostRecentBuy(t *testing.T) {
	entryTime := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	now := entryTime.Add(20 * time.Hour)

	tc := &fakeTrades{trades: []grpcmodel.Trade{
		{Date: entryTime.Add(-48 * time.Hour), Price: 90, Quantity: 10, IsBuy: true}, // older buy
		{Date: entryTime, Price: 100, Quantity: 10, IsBuy: true},                      // most recent buy
	}}

	var candles []*imodel.CandleItemTechAnalyse
	for i := 0; i < 40; i++ {
		ts := entryTime.Add(time.Duration(i-20) * time.Hour)
		c := 100.0 + float64(i) // rising -> later closes are higher
		candles = append(candles, &imodel.CandleItemTechAnalyse{
			Time: ts, Open: q(c), High: q(c + 1), Low: q(c - 1), Close: q(c), Volume: 1000, IsComplete: true,
		})
	}
	cc := &fakeCandles{candles: candles}

	got, err := Entry(context.Background(), tc, cc, "acc", "uid", "UGLD", 100, 14, 50, now)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if !got.EntryTime.Equal(entryTime) {
		t.Fatalf("EntryTime = %v, want %v", got.EntryTime, entryTime)
	}
	if got.EntryPrice != 100 {
		t.Fatalf("EntryPrice = %v, want 100 (broker avg)", got.EntryPrice)
	}
	if got.MaxFav < got.EntryPrice {
		t.Fatalf("MaxFav %v < EntryPrice %v", got.MaxFav, got.EntryPrice)
	}
	if got.EntryATR <= 0 {
		t.Fatalf("EntryATR = %v, want > 0", got.EntryATR)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/live/reconstruct/ -v`
Expected: FAIL — `undefined: Entry`.

- [ ] **Step 3: Implement `reconstruct.go`**

```go
// Package reconstruct rebuilds reversion entry-state from the broker API when the
// local state file is missing but a position is open. EntryPrice uses the broker's
// average purchase price; EntryTime is the most recent BUY fill; EntryATR and MaxFav
// are recomputed from hourly candles around/after entry.
package reconstruct

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/reversion/live/marketdata"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	"tinvest/internal/utils"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/indicators"
)

// TradesClient is the slice of the operations client reconstruct needs.
type TradesClient interface {
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)
}

// Entry reconstructs the entry-state for an open position.
func Entry(ctx context.Context, tc TradesClient, cc marketdata.CandleClient,
	accountID, instrumentID, ticker string, purchasePrice float64,
	atrPeriod, lookbackBars int, now time.Time) (statestore.Entry, error) {

	trades, err := tc.GetInstrumentTrades(ctx, accountID, instrumentID, now.AddDate(0, 0, -120), now)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("reconstruct: trades: %w", err)
	}
	var entryTime time.Time
	for _, tr := range trades {
		if tr.IsBuy && tr.Date.After(entryTime) {
			entryTime = tr.Date
		}
	}
	if entryTime.IsZero() {
		return statestore.Entry{}, fmt.Errorf("reconstruct: no BUY fill found for %s", ticker)
	}

	// Hourly candles from before entry (for ATR warm-up) through now (for maxFav).
	from := entryTime.AddDate(0, 0, -(lookbackBars/4 + 10))
	limit := int32(lookbackBars * 6)
	raw, err := cc.GetCandles(ctx, &instrumentID, enum.Hour1.ToNumberInvestApi(),
		timestamppb.New(from), timestamppb.New(now), &limit, true)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("reconstruct: candles: %w", err)
	}
	candles := completed(raw)

	atr := atrAtEntry(candles, entryTime, atrPeriod, lookbackBars)
	maxFav := purchasePrice
	for _, c := range candles {
		if !c.Time.Before(entryTime) && c.Close > maxFav {
			maxFav = c.Close
		}
	}

	return statestore.Entry{
		Ticker:     ticker,
		EntryTime:  entryTime,
		EntryPrice: purchasePrice,
		EntryATR:   atr,
		MaxFav:     maxFav,
	}, nil
}

type bar struct {
	Time               time.Time
	High, Low, Close   float64
}

func completed(raw []*imodel.CandleItemTechAnalyse) []bar {
	out := make([]bar, 0, len(raw))
	for _, c := range raw {
		if !c.IsComplete {
			continue
		}
		out = append(out, bar{
			Time:  c.Time,
			High:  utils.CombinePrice(c.High.Units, c.High.Nano),
			Low:   utils.CombinePrice(c.Low.Units, c.Low.Nano),
			Close: utils.CombinePrice(c.Close.Units, c.Close.Nano),
		})
	}
	return out
}

// atrAtEntry computes ATR over the hourly window ending at the entry bar (the bar at
// or just before entryTime), mirroring how the core stamps EntryATR at entry.
func atrAtEntry(bars []bar, entryTime time.Time, atrPeriod, lookbackBars int) float64 {
	end := -1
	for i, b := range bars {
		if !b.Time.After(entryTime) {
			end = i
		}
	}
	if end < 0 {
		return 0
	}
	start := end - lookbackBars + 1
	if start < 0 {
		start = 0
	}
	window := bars[start : end+1]
	highs := make([]float64, len(window))
	lows := make([]float64, len(window))
	closes := make([]float64, len(window))
	for i, b := range window {
		highs[i], lows[i], closes[i] = b.High, b.Low, b.Close
	}
	return indicators.ATR(highs, lows, closes, atrPeriod)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/live/reconstruct/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/reconstruct/
git commit -m "feat(reversion/live): reconstruct entry-state from API fallback

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 11: `service` — buy-pass + manage-pass orchestration

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/dto/run.go`
- Create: `internal/service/trading_strategy/reversion/live/live.go`
- Create: `internal/service/trading_strategy/reversion/live/buy.go`
- Create: `internal/service/trading_strategy/reversion/live/manage.go`
- Test: `internal/service/trading_strategy/reversion/live/service_test.go`

**Interfaces:**
- Consumes narrow clients:
  - `instrumentsClient interface { Shares(ctx) ([]*imodel.Share, error) }`
  - `marketDataClient = marketdata.CandleClient`
  - `operationsClient interface { GetPortfolio(ctx, accountID) ([]*grpcmodel.Position, error); GetPortfolioTotal(ctx, accountID) (float64, error); GetAvailableCash(ctx, accountID) (float64, error); GetInstrumentTrades(...) ([]grpcmodel.Trade, error) }`
  - `ordersClient = executor.OrdersClient`
  - `tgClient telegram.Client`
- Produces:
  - `dto.Run{Scheduler string; Mode Mode}`, `Mode` (`ModeBuy`, `ModeManage`).
  - `Service interface { Run(ctx context.Context, in dto.Run) error }`.
  - `func NewService(instrumentsClient, marketDataClient, operationsClient, ordersClient, tgClient, cfg *config.ReversionConfig) *service`.
  Consumed by the scheduler (Task 12) and service_provider (Task 13).

- [ ] **Step 1: Create the dto**

`internal/service/trading_strategy/reversion/live/dto/run.go`:

```go
package dto

// Mode selects which pass a scheduled run executes.
type Mode int

const (
	ModeBuy Mode = iota
	ModeManage
)

// Run is one scheduled invocation of the reversion runner.
type Run struct {
	Scheduler string
	Mode      Mode
}
```

- [ ] **Step 2: Write the failing test**

Create `service_test.go`. It drives the flag matrix and both passes with fakes. Use a candle generator that produces a clean dual-oversold entry is heavy; instead, inject the strategy decision by testing at the orchestration seam: the test provides candles that yield no signal (assert no order, no state) for the happy "wiring compiles + flag matrix" path, plus a manage-pass test where state exists and the broker holds the position (assert maxFav updates and persists).

```go
package live

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/reversion/live/dto"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// --- fakes ---

type fakeInstruments struct{ shares []*imodel.Share }

func (f *fakeInstruments) Shares(context.Context) ([]*imodel.Share, error) { return f.shares, nil }

type fakeMarket struct{ hourly []*imodel.CandleItemTechAnalyse }

func (f *fakeMarket) GetCandles(_ context.Context, _ *string, interval int32, _, _ *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	if interval == 4 {
		return f.hourly, nil
	}
	return nil, nil
}

type fakeOps struct {
	positions []*grpcmodel.Position
	total     float64
	cash      float64
}

func (f *fakeOps) GetPortfolio(context.Context, string) ([]*grpcmodel.Position, error) { return f.positions, nil }
func (f *fakeOps) GetPortfolioTotal(context.Context, string) (float64, error)          { return f.total, nil }
func (f *fakeOps) GetAvailableCash(context.Context, string) (float64, error)           { return f.cash, nil }
func (f *fakeOps) GetInstrumentTrades(context.Context, string, string, time.Time, time.Time) ([]grpcmodel.Trade, error) {
	return nil, nil
}

type fakeTg struct{ sent []string }

func (f *fakeTg) SendMessage(m string) error              { f.sent = append(f.sent, m); return nil }
func (f *fakeTg) SendMessageToChat(int64, string) error   { return nil }

func q(f float64) imodel.Quotation { return imodel.Quotation{Units: int64(f)} }

// flatHourly builds a flat (no-signal) hourly series long enough for the lookback.
func flatHourly(n int) []*imodel.CandleItemTechAnalyse {
	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	var out []*imodel.CandleItemTechAnalyse
	for i := 0; i < n; i++ {
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time: base.Add(time.Duration(i) * time.Hour),
			Open: q(100), High: q(100), Low: q(100), Close: q(100), Volume: 1000, IsComplete: true,
		})
	}
	return out
}

func cfg(dir string) *config.ReversionConfig {
	return &config.ReversionConfig{
		AccountID: "acc", Tickers: []string{"UGLD"}, BuyPct: 10,
		TradeEnabled: false, NotifyEnabled: true,
	}
}

func TestBuyPass_NoSignal_NoOrderNoState(t *testing.T) {
	dir := t.TempDir()
	c := cfg(dir)
	svc := NewService(
		&fakeInstruments{shares: []*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1, Trading: true}}},
		&fakeMarket{hourly: flatHourly(400)},
		&fakeOps{total: 100000, cash: 100000},
		nil, // ordersClient unused in dry-run no-signal path
		&fakeTg{},
		c,
	)
	svc.statePath = filepath.Join(dir, "state.json")

	if err := svc.Run(context.Background(), dto.Run{Mode: dto.ModeBuy}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(svc.statePath).Load()
	if len(st) != 0 {
		t.Fatalf("flat series must produce no entry, got %v", st)
	}
}

func TestManagePass_UpdatesMaxFavAndPersists(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	// Seed state with an open UGLD position at entry 100, maxFav 100.
	_ = statestore.New(statePath).Save(map[string]statestore.Entry{
		"UGLD": {Ticker: "UGLD", EntryPrice: 100, EntryATR: 2, MaxFav: 100, Quantity: 10,
			EntryTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)},
	})

	// Hourly series ending at a higher close (110) -> maxFav should rise.
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	var hourly []*imodel.CandleItemTechAnalyse
	for i := 0; i < 400; i++ {
		c := 100.0
		if i == 399 {
			c = 110.0
		}
		hourly = append(hourly, &imodel.CandleItemTechAnalyse{
			Time: base.Add(time.Duration(i) * time.Hour),
			Open: q(c), High: q(c), Low: q(c), Close: q(c), Volume: 1000, IsComplete: true,
		})
	}

	svc := NewService(
		&fakeInstruments{shares: []*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1, Trading: true}}},
		&fakeMarket{hourly: hourly},
		&fakeOps{positions: []*grpcmodel.Position{{ShareID: "uid-ugld", InstrumentType: "share", Quantity: 10,
			PurchasePrice: q(100)}}},
		nil,
		&fakeTg{},
		cfg(dir),
	)
	svc.statePath = statePath

	if err := svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(statePath).Load()
	if st["UGLD"].MaxFav != 110 {
		t.Fatalf("MaxFav = %v, want 110 (raised from latest close)", st["UGLD"].MaxFav)
	}
}
```

> Implementer note: `Quotation` here uses `Units` only for round numbers; `imodel.Quotation` fields are `Units int64`, `Nano int32`.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/live/ -run 'TestBuyPass|TestManagePass' -v`
Expected: FAIL — `undefined: NewService`.

- [ ] **Step 4: Implement `live.go` (service + wiring)**

```go
package live

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/reversion/live/dto"
	"tinvest/internal/service/trading_strategy/reversion/live/executor"
	"tinvest/internal/service/trading_strategy/reversion/live/marketdata"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/client/telegram"
)

type instrumentsClient interface {
	Shares(ctx context.Context) ([]*imodel.Share, error)
}

type operationsClient interface {
	GetPortfolio(ctx context.Context, accountID string) ([]*grpcmodel.Position, error)
	GetPortfolioTotal(ctx context.Context, accountID string) (float64, error)
	GetAvailableCash(ctx context.Context, accountID string) (float64, error)
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)
}

// Service runs one scheduled reversion pass.
type Service interface {
	Run(ctx context.Context, in dto.Run) error
}

type service struct {
	instruments instrumentsClient
	market      marketdata.CandleClient
	ops         operationsClient
	exec        *executor.Executor
	tg          telegram.Client
	cfg         *config.ReversionConfig
	statePath   string
}

// NewService wires the live reversion service. The orders client may be nil only when
// TradeEnabled is false and no order will ever be placed (tests/dry-run).
func NewService(
	instruments instrumentsClient,
	market marketdata.CandleClient,
	ops operationsClient,
	orders executor.OrdersClient,
	tg telegram.Client,
	cfg *config.ReversionConfig,
) *service {
	return &service{
		instruments: instruments,
		market:      market,
		ops:         ops,
		exec:        executor.New(orders, cfg.AccountID, cfg.TradeEnabled),
		tg:          tg,
		cfg:         cfg,
		statePath:   filepath.Join("data", "state", "reversion_"+cfg.AccountID+".json"),
	}
}

// Run dispatches to the buy or manage pass.
func (s *service) Run(ctx context.Context, in dto.Run) error {
	switch in.Mode {
	case dto.ModeBuy:
		return s.buyPass(ctx)
	case dto.ModeManage:
		return s.managePass(ctx)
	default:
		return fmt.Errorf("reversion: unknown run mode %d", in.Mode)
	}
}

// notify sends a Telegram message only when NotifyEnabled.
func (s *service) notify(msg string) {
	if s.cfg.NotifyEnabled {
		_ = s.tg.SendMessage(msg)
	}
}

// sharesByTicker indexes tradable shares for the configured universe.
func (s *service) sharesByTicker(ctx context.Context) (map[string]*imodel.Share, error) {
	all, err := s.instruments.Shares(ctx)
	if err != nil {
		return nil, fmt.Errorf("reversion: load shares: %w", err)
	}
	out := make(map[string]*imodel.Share, len(all))
	for _, sh := range all {
		out[sh.Ticker] = sh
	}
	return out, nil
}

// heldByShareID indexes the account's share positions with qty > 0.
func (s *service) heldByShareID(ctx context.Context) (map[string]*grpcmodel.Position, error) {
	positions, err := s.ops.GetPortfolio(ctx, s.cfg.AccountID)
	if err != nil {
		return nil, fmt.Errorf("reversion: load portfolio: %w", err)
	}
	out := make(map[string]*grpcmodel.Position, len(positions))
	for _, p := range positions {
		if p.InstrumentType == "share" && p.Quantity > 0 {
			out[p.ShareID] = p
		}
	}
	return out, nil
}

func nowMSK() time.Time {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(loc)
}
```

- [ ] **Step 5: Implement `buy.go`**

```go
package live

import (
	"context"
	"fmt"

	"tinvest/internal/service/trading_strategy/reversion/live/marketdata"
	"tinvest/internal/service/trading_strategy/reversion/live/notifier"
	"tinvest/internal/service/trading_strategy/reversion/live/sizing"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

func (s *service) buyPass(ctx context.Context) error {
	shares, err := s.sharesByTicker(ctx)
	if err != nil {
		return err
	}
	held, err := s.heldByShareID(ctx)
	if err != nil {
		return err
	}
	store := statestore.New(s.statePath)
	state, err := store.Load()
	if err != nil {
		return fmt.Errorf("reversion: load state: %w", err)
	}

	now := nowMSK()
	for _, ticker := range s.cfg.Tickers {
		st, ok := StrategyFor(ticker)
		if !ok {
			s.notify(notifier.Alert(ticker, "тикер не зарегистрирован в reversion — пропуск"))
			continue
		}
		sh, ok := shares[ticker]
		if !ok || !sh.Trading {
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s not tradable, skip", ticker))
			continue
		}
		// Already in a position (broker or our state) -> manage-pass handles it.
		if _, h := held[sh.ID]; h {
			continue
		}
		if _, h := state[ticker]; h {
			continue
		}

		md, err := marketdata.Assemble(ctx, s.market, sh.ID, st.Lookback(), MaxHTFTrendEMA([]string{ticker}), now)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s marketdata: %v", ticker, err))
			continue
		}
		md.Position = nil

		sig := st.Decide(md)
		if sig.Kind != model.SignalBuy {
			continue
		}

		total, err := s.ops.GetPortfolioTotal(ctx, s.cfg.AccountID)
		if err != nil {
			return fmt.Errorf("reversion: portfolio total: %w", err)
		}
		cash, err := s.ops.GetAvailableCash(ctx, s.cfg.AccountID)
		if err != nil {
			return fmt.Errorf("reversion: cash: %w", err)
		}
		lots, ok, reason := sizing.Lots(s.cfg.BuyPct, total, cash, sig.Price, sh.Lot)
		if !ok {
			s.notify(notifier.Skip(ticker, reason))
			continue
		}

		res, err := s.exec.Buy(ctx, sh.ID, lots)
		if err != nil {
			s.notify(notifier.Alert(ticker, "ордер на покупку отклонён: "+err.Error()))
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s buy rejected: %v", ticker, err))
			continue // state unchanged; retried next tick
		}

		fillPrice := sig.Price
		filledLots := lots
		if res.Placed {
			if res.FillPrice > 0 {
				fillPrice = res.FillPrice
			}
			if res.FilledLots > 0 {
				filledLots = res.FilledLots
			}
		}
		qty := filledLots * int64(sh.Lot)

		state[ticker] = statestore.Entry{
			Ticker:     ticker,
			EntryTime:  now,
			EntryPrice: fillPrice,
			EntryATR:   sig.ATR,
			MaxFav:     fillPrice,
			Quantity:   qty,
		}
		if err := store.Save(state); err != nil {
			return fmt.Errorf("reversion: save state after buy %s: %w", ticker, err)
		}
		_ = utils.CombinePrice // keep import if unused after edits
		s.notify(notifier.Entry(ticker, fillPrice, filledLots, qty, !res.Placed))
	}
	return nil
}
```

> Implementer note: remove the `utils` import / `_ = utils.CombinePrice` line if `utils` ends up unused.

- [ ] **Step 6: Implement `manage.go`**

```go
package live

import (
	"context"
	"fmt"

	"tinvest/internal/service/trading_strategy/reversion/live/marketdata"
	"tinvest/internal/service/trading_strategy/reversion/live/notifier"
	"tinvest/internal/service/trading_strategy/reversion/live/reconstruct"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

func (s *service) managePass(ctx context.Context) error {
	shares, err := s.sharesByTicker(ctx)
	if err != nil {
		return err
	}
	held, err := s.heldByShareID(ctx)
	if err != nil {
		return err
	}
	store := statestore.New(s.statePath)
	state, err := store.Load()
	if err != nil {
		return fmt.Errorf("reversion: load state: %w", err)
	}

	now := nowMSK()
	for _, ticker := range s.cfg.Tickers {
		st, ok := StrategyFor(ticker)
		if !ok {
			continue
		}
		sh, ok := shares[ticker]
		if !ok || !sh.Trading {
			continue
		}
		pos, isHeld := held[sh.ID]
		if !isHeld {
			// Position gone (e.g. sold elsewhere) — drop any stale state entry.
			if _, ok := state[ticker]; ok {
				delete(state, ticker)
				_ = store.Save(state)
			}
			continue
		}

		entry, hasState := state[ticker]
		if !hasState {
			// Reconstruct from API + alert.
			rebuilt, err := reconstruct.Entry(ctx, s.ops, s.market, s.cfg.AccountID, sh.ID, ticker,
				utils.CombinePrice(pos.PurchasePrice.Units, pos.PurchasePrice.Nano),
				atrPeriodFor(ticker), st.Lookback(), now)
			if err != nil {
				s.notify(notifier.Alert(ticker, "позиция без локального стейта, реконструкция не удалась: "+err.Error()))
				logger.ErrorContext(ctx, fmt.Sprintf("reversion: reconstruct %s: %v", ticker, err))
				continue
			}
			rebuilt.Quantity = pos.Quantity
			entry = rebuilt
			state[ticker] = entry
			_ = store.Save(state)
			s.notify(notifier.Alert(ticker, fmt.Sprintf("стейт восстановлен из API: вход %.4f, ATR %.4f", entry.EntryPrice, entry.EntryATR)))
		}

		md, err := marketdata.Assemble(ctx, s.market, sh.ID, st.Lookback(), MaxHTFTrendEMA([]string{ticker}), now)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s marketdata: %v", ticker, err))
			continue
		}

		// Raise maxFav from the latest completed close, then persist (monotonic).
		if md.Price > entry.MaxFav {
			entry.MaxFav = md.Price
			state[ticker] = entry
			if err := store.Save(state); err != nil {
				return fmt.Errorf("reversion: save maxFav %s: %w", ticker, err)
			}
		}

		md.Position = &strategy.Position{
			PurchasePrice:     entry.EntryPrice,
			Quantity:          pos.Quantity,
			EntryATR:          entry.EntryATR,
			MaxFavorablePrice: entry.MaxFav,
		}

		sig := st.Decide(md)
		if sig.Kind != model.SignalSell {
			continue
		}

		lots := pos.Quantity / int64(sh.Lot)
		res, err := s.exec.Sell(ctx, sh.ID, lots)
		if err != nil {
			s.notify(notifier.Alert(ticker, "ордер на продажу отклонён: "+err.Error()))
			logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s sell rejected: %v", ticker, err))
			continue // state unchanged; retried next tick
		}

		exitPrice := sig.Price
		if res.Placed && res.FillPrice > 0 {
			exitPrice = res.FillPrice
		}
		delete(state, ticker)
		if err := store.Save(state); err != nil {
			return fmt.Errorf("reversion: save state after sell %s: %w", ticker, err)
		}
		s.notify(notifier.Exit(ticker, sig.Reason, exitPrice, pos.Quantity, !res.Placed))
	}
	return nil
}

// atrPeriodFor returns the ticker's ATRPeriod for reconstruct's ATR recomputation.
func atrPeriodFor(ticker string) int {
	if p, ok := paramsByTicker[ticker]; ok && p.ATRPeriod > 0 {
		return p.ATRPeriod
	}
	return 14
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/live/ -v && go build ./...`
Expected: PASS and clean build.

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/dto/ internal/service/trading_strategy/reversion/live/live.go internal/service/trading_strategy/reversion/live/buy.go internal/service/trading_strategy/reversion/live/manage.go internal/service/trading_strategy/reversion/live/service_test.go
git commit -m "feat(reversion/live): buy-pass and manage-pass orchestration

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 12: `scheduler` — two cron jobs

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/scheduler/scheduler.go`
- Test: `internal/service/trading_strategy/reversion/live/scheduler/scheduler_test.go`

**Interfaces:**
- Produces: `func NewSchedulerService(svc live.Service) live.Service` — wraps a `Service` so its `Run` registers the cron job from `in.Scheduler` and blocks until ctx is done (mirrors `scalping/scheduler`). Consumed by `app.go` (Task 13).

- [ ] **Step 1: Write the failing test**

Create `scheduler_test.go` (verifies the wrapper registers a job and returns on ctx cancel without invoking the inner service synchronously):

```go
package scheduler

import (
	"context"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/reversion/live/dto"
)

type fakeSvc struct{ calls int }

func (f *fakeSvc) Run(context.Context, dto.Run) error { f.calls++; return nil }

func TestSchedulerService_ReturnsOnContextCancel(t *testing.T) {
	inner := &fakeSvc{}
	sch := NewSchedulerService(inner)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sch.Run(ctx, dto.Run{Scheduler: "0 8-23 * * 1-5", Mode: dto.ModeBuy}) }()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/live/scheduler/ -v`
Expected: FAIL — `undefined: NewSchedulerService`.

- [ ] **Step 3: Implement `scheduler.go`** (mirror `scalping/scheduler/trade.go`)

```go
// Package scheduler runs the live reversion service on a cron schedule.
package scheduler

import (
	"context"
	"time"

	"tinvest/internal/service/trading_strategy/reversion/live"
	"tinvest/internal/service/trading_strategy/reversion/live/dto"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service live.Service
}

// NewSchedulerService wraps a live.Service so Run registers a cron job and blocks.
func NewSchedulerService(service live.Service) live.Service {
	return &schedulerService{sh: scheduler.NewScheduler(), service: service}
}

func (s *schedulerService) Run(ctx context.Context, in dto.Run) error {
	jobTicker := time.NewTicker(time.Hour)
	defer jobTicker.Stop()

	err := s.sh.AddJob(in.Scheduler, func() {
		logger.InfoContext(ctx, "Воркер Reversion начал работу")
		if err := s.service.Run(ctx, in); err != nil {
			logger.ErrorContext(ctx, "Ошибка в ходе работы job Reversion", err)
		}
	})
	if err != nil {
		return err
	}

	s.sh.Start()
	defer s.sh.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-jobTicker.C:
			logger.InfoContext(ctx, "Worker Reversion is running")
		}
	}
}
```

> Implementer note: confirm `scheduler.AddJob`'s callback signature (the scalping version uses `func()`); `logger.ErrorContext`'s third arg matches the scalping call site.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/live/scheduler/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/scheduler/
git commit -m "feat(reversion/live): cron scheduler wrapper

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 13: Wire into `service_provider` and `app.go`

**Files:**
- Modify: `internal/service_provider/service.go` (add field + getter)
- Modify: `internal/app/app.go` (start two workers in `runProd`; optionally `runDev`)
- Test: `go build ./...` (wiring is verified by compilation; behavior is covered by Task 11/12)

**Interfaces:**
- Consumes: `live.NewService`, `scheduler.NewSchedulerService`, `dto.Run`, `config.Reversion`.
- Produces: `func (*ServiceProvider) GetReversionLiveService() live.Service`.

- [ ] **Step 1: Add the provider field**

In `internal/service_provider/service.go`, add to the `service` struct (near `scalpingTradingService`, line 25):

```go
	reversionLiveService live.Service
```

Add the import:

```go
	"tinvest/internal/service/trading_strategy/reversion/live"
```

- [ ] **Step 2: Add the getter** (mirror `GetScalpingTradingService`, lines 211-225)

```go
func (*ServiceProvider) GetReversionLiveService() live.Service {
	if serviceProvider.service.reversionLiveService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.reversionLiveService = live.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			grpcClient.OperationsServiceClient(),
			grpcClient.OrdersServiceClient(),
			tgClient,
			serviceProvider.appConfig.Reversion,
		)
	}
	return serviceProvider.service.reversionLiveService
}
```

- [ ] **Step 3: Start the workers in `runProd`**

In `internal/app/app.go`, add imports:

```go
	reversiondto "tinvest/internal/service/trading_strategy/reversion/live/dto"
	reversionscheduler "tinvest/internal/service/trading_strategy/reversion/live/scheduler"
```

In `runProd`, add two goroutines next to the scalping worker (mirror lines 325-337). Increment the `wg.Add` count accordingly:

```go
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := reversionscheduler.NewSchedulerService(a.sp.GetReversionLiveService()).Run(
			ctx,
			reversiondto.Run{Scheduler: "0 8-23 * * 1-5", Mode: reversiondto.ModeBuy},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Reversion buy", err.Error())
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := reversionscheduler.NewSchedulerService(a.sp.GetReversionLiveService()).Run(
			ctx,
			reversiondto.Run{Scheduler: "0 7-23,0 * * *", Mode: reversiondto.ModeManage},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Reversion manage", err.Error())
		}
	}()
```

> Implementer note: the two schedulers each wrap their own `GetReversionLiveService()` call; since the getter memoizes a single `service` instance, both share it (stateless across runs — state lives in the file). That is intended.

- [ ] **Step 4: Verify build**

Run: `go build ./... && go vet ./internal/service/trading_strategy/reversion/... && go test ./...`
Expected: clean build, vet clean, all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/service_provider/service.go internal/app/app.go
git commit -m "feat(app): wire live reversion buy/manage workers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 14: Operator docs + `.env` example

**Files:**
- Create: `docs/reversion/live-runner.md`
- Modify: `env/local.env.example` (add the new env vars)

**Interfaces:** none (documentation).

- [ ] **Step 1: Add env vars to `env/local.env.example`**

Append:

```bash
# Reversion live runner
REVERSION_ACCOUNT_ID=
REVERSION_TICKERS=UGLD,EUTR,NVTK
REVERSION_BUY_PCT=10
REVERSION_TRADE_ENABLED=false
REVERSION_NOTIFY_ENABLED=true
```

- [ ] **Step 2: Write `docs/reversion/live-runner.md`**

Document: the flag matrix (paper/live/silent/dry-run), the two cron schedules, the state file location (`data/state/reversion_<account>.json`) and what it holds, the accepted backtest divergence (next-bar fill, no gap-fill exits), the universe env list, and the reconstruct fallback behaviour. Reference the design spec.

- [ ] **Step 3: Commit**

```bash
git add docs/reversion/live-runner.md env/local.env.example
git commit -m "docs(reversion): live runner operator guide and env example

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:**
- Buy full / sell full, market orders → Tasks 7, 11. ✓
- Independent trade/notify flags → Tasks 2, 7, 11 (flag matrix). ✓
- Hourly buy schedule / daily manage schedule → Task 13 cron strings. ✓
- Per-ticker params from `DefaultParams()` (single source with backtest) → Task 5. ✓
- Local state file primary + reconstruct fallback + alert → Tasks 3, 10, 11. ✓
- Fresh rolling window per run, no cache, warm-up sized from params → Task 6 (`Lookback()`, `MaxHTFTrendEMA`). ✓
- % of full account value sizing, skip+alert on insufficient cash, one position per ticker → Tasks 4, 9, 11. ✓
- Dedicated account id, env config → Tasks 2, 13. ✓
- Reuse backtest `MarketData` assembly + parity test → Tasks 1, 6. ✓
- `EntryATR = sig.ATR` on entry; `maxFav` updated each manage tick → Task 11. ✓
- Edge cases (ticker not in universe untouched; order rejected → log+alert, state unchanged; broker qty differs → record actual) → Task 11. ✓
- Operations client extension for fill data → Task 9. ✓
- Unit tests for sizing / statestore / executor / flag matrix / reconstruct; parity test → Tasks 3,4,6,7,10,11. ✓

**Known deviations from the spec text, intentionally encoded:**
- The spec's warm-up note mentions a daily timeframe and "daily ATR"; the **code** computes ATR on the hourly window and `core.Decide` never reads daily series. The plan therefore **does not fetch daily candles** (Global Constraints + Task 6). This is the "max like backtest" choice: identical inputs to what the core actually consumes.

**Type consistency:** `statestore.Entry`, `executor.Result`, `marketdata.CandleClient`, `dto.Run/Mode`, `live.Service`, `grpcmodel.Trade` names are used identically across producing and consuming tasks. `strategy.Position`/`strategy.MarketData`/`model.Signal` come from `scalping/strategy` and `scalping/model` (the packages `reversion/strategy/core` already imports).

**Placeholder scan:** No TBD/TODO/"handle errors appropriately"; every code step carries full code. Two implementer-notes flag generated-symbol confirmations (pb cursor types in Task 9, `MoneyValue.Nano` width in Task 7, scheduler callback signature in Task 12) — these are verification prompts against generated code, not missing logic.
