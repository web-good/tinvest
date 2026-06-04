# RUSAL HTF Trend Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a daily (higher-timeframe) EMA trend filter that blocks long entries in the RUSAL scalping strategy when price is below the daily EMA(N), to stop the stop-loss bleed from buying against a downtrend.

**Architecture:** A new `DailyCloses` slice on `strategy.MarketData` carries oldest-first closes of *completed* daily candles. The backtest engine and the live runner each populate it with only the days that closed at/before the current bar (no lookahead). The RUSAL strategy gains a `TrendFilterPeriod` param (default 200, 0 = off); when on, it computes a daily EMA and gates both entry branches on `lastDailyClose > dailyEMA`.

**Tech Stack:** Go 1.25, existing `internal/domain/ema`, `internal/domain/backtest` engine, `internal/service/backtest` candle provider/calibration, gRPC market-data client.

**Spec:** `docs/superpowers/specs/2026-06-05-rusal-htf-trend-filter-design.md`

---

## File Structure

- `internal/service/trading_strategy/scalping/strategy/strategy.go` — add `DailyCloses` field to `MarketData`.
- `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go` — add `TrendFilterPeriod` param, compute `trendUp`, gate both entry branches.
- `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go` — gate tests.
- `internal/domain/backtest/engine.go` — `Run` takes `dailyCandles`, builds `md.DailyCloses` without lookahead.
- `internal/domain/backtest/engine_test.go` — alignment helper test + gating Run test; update existing `Run` calls.
- `internal/service/backtest/calibrate.go` — `RunGrid` takes `dailyCandles`.
- `internal/service/backtest/calibrate_test.go` — update `RunGrid` calls.
- `cmd/backtest/main.go` — load Day1 candles, thread them into `runSingle`/`runCalibration`.
- `internal/service/trading_strategy/scalping/trade.go` — second daily `GetCandles`, populate `md.DailyCloses` from completed days.
- `internal/service/trading_strategy/scalping/trade_test.go` — unaffected (fakeStrategy ignores DailyCloses); no change needed, just stays green.

---

## Task 1: Strategy param + entry gate (pure decision core)

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go`
- Modify: `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`
- Test: `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go`

- [ ] **Step 1: Add `DailyCloses` field to MarketData**

In `strategy.go`, inside `type MarketData struct { ... }`, add after `Volumes`:

```go
	// DailyCloses are oldest-first closes of COMPLETED daily candles, aligned so the
	// last element is the most recent day closed at/before the current bar. Empty if
	// no higher-timeframe data is supplied or the filter is disabled.
	DailyCloses []float64
```

- [ ] **Step 2: Write failing gate tests in the pure core**

In `rusal_test.go`, add these cases to the `tests` slice inside `TestDecideCore` (after the existing `"range entry: at lower band + rsi cross"` case). They use the new `trendFilterOn` / `trendUp` fields on `decideInput`:

```go
		{
			name: "trend entry blocked by HTF filter (filter on, trend down)",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
				trendFilterOn: true, trendUp: false,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "trend entry allowed when HTF filter passes",
			in: decideInput{
				price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
				adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
				trendFilterOn: true, trendUp: true,
			},
			wantKind: model.SignalBuy, wantTP: 104, wantSL: 98,
		},
		{
			name: "range entry blocked by HTF filter (filter on, trend down)",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
				trendFilterOn: true, trendUp: false,
			},
			wantKind: model.SignalNone,
		},
		{
			name: "range entry allowed when HTF filter passes",
			in: decideInput{
				price: 100, atr: 2, rsiPrev: 30, rsiNow: 36,
				adx: 15, donUpper: 110, donLower: 99.8,
				trendFilterOn: true, trendUp: true,
			},
			wantKind: model.SignalBuy, wantTP: 104.9, wantSL: 98,
		},
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/ -run TestDecideCore -v`
Expected: COMPILE FAILURE — `unknown field 'trendFilterOn' in struct literal of type decideInput`.

- [ ] **Step 4: Add fields + param + gate**

In `rusal.go`:

(a) Add the param to `Params` (after `BandTol`):

```go
	TrendFilterPeriod int // daily EMA period for the higher-timeframe long filter; 0 disables
```

(b) Add to `DefaultParams()` return (after `BandTol: 0.003,`):

```go
		TrendFilterPeriod: 200,
```

(c) Add two fields to `decideInput` (after `pos`):

```go
	trendFilterOn  bool
	trendUp        bool
```

(d) In `Decide`, after computing the other indicators and before building `in := decideInput{...}`, compute the filter:

```go
	trendFilterOn := s.p.TrendFilterPeriod > 0
	trendUp := false
	if trendFilterOn && len(md.DailyCloses) >= s.p.TrendFilterPeriod {
		emaD := ema.Compute(md.DailyCloses, s.p.TrendFilterPeriod)
		lastDaily := md.DailyCloses[len(md.DailyCloses)-1]
		trendUp = lastDaily > emaD[len(emaD)-1]
	}
```

Then add the two fields to the `decideInput{...}` literal:

```go
		trendFilterOn:  trendFilterOn,
		trendUp:        trendUp,
```

(e) In the pure `decide`, gate both entry branches. Change the trend branch condition:

```go
	case regimeTrend:
		crossedUp := in.rsiPrev < s.p.RSITrendLevel && in.rsiNow >= s.p.RSITrendLevel
		if in.diPlus > in.diMinus && in.emaTouched && crossedUp && in.price > in.emaNow &&
			(!in.trendFilterOn || in.trendUp) {
```

and the range branch condition:

```go
	case regimeRange:
		crossedUp := in.rsiPrev < s.p.RSIRangeLevel && in.rsiNow >= s.p.RSIRangeLevel
		if in.price <= in.donLower*(1+s.p.BandTol) && crossedUp &&
			(!in.trendFilterOn || in.trendUp) {
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/ -run TestDecideCore -v`
Expected: PASS (all existing cases — which leave `trendFilterOn` false — plus the four new ones).

- [ ] **Step 6: Write a Decide-level test for the daily EMA → trendUp path**

In `rusal_test.go`, add a new test. It uses a small `TrendFilterPeriod` so the synthetic daily series is short. The market setup is a copy of `TestDecide_FlatUptrendIsNone`'s rising series (which on its own yields `None`); we instead drive a *range buy* by also supplying a daily series and checking the gate. Simplest: assert that the same flat market that would be `None` stays `None`, and focus the daily-path assertions on the helper via `Decide` with a crafted range setup is hard end-to-end — so test the daily computation directly through a tiny exported-free helper is not available. Instead, test through `Decide` using filter on/off on the rising series and assert `trendUp` is reflected by NOT panicking and returning a valid signal. Concretely:

```go
func TestDecide_DailyFilterGate(t *testing.T) {
	p := testParams()
	p.TrendFilterPeriod = 3 // tiny period so a short daily series is enough

	// A 200-bar rising intraday series (same shape as TestDecide_FlatUptrendIsNone).
	highs := make([]float64, 200)
	lows := make([]float64, 200)
	closes := make([]float64, 200)
	for i := 0; i < 200; i++ {
		base := 100.0 + float64(i)
		highs[i] = base + 1
		lows[i] = base - 1
		closes[i] = base
	}

	// Daily series ABOVE its own EMA(3) -> trendUp true.
	upDaily := []float64{10, 20, 30, 40} // last (40) > EMA3 of the series
	// Daily series BELOW its own EMA(3) -> trendUp false.
	downDaily := []float64{40, 30, 20, 10} // last (10) < EMA3 of the series

	mk := func(daily []float64) strategy.MarketData {
		return strategy.MarketData{
			Price: closes[199], Highs: highs, Lows: lows, Closes: closes,
			DailyCloses: daily,
		}
	}

	s := NewWithParams(p)
	// This rising-only market produces None regardless (no pullback/cross); the point
	// of this test is that the daily filter path executes without error and the
	// gate never turns a None into a Buy. Stronger gate behavior is covered by
	// TestDecideCore.
	if got := s.Decide(mk(upDaily)); got.Kind == model.SignalBuy {
		t.Fatalf("unexpected Buy on rising-only market (up daily)")
	}
	if got := s.Decide(mk(downDaily)); got.Kind == model.SignalBuy {
		t.Fatalf("unexpected Buy on rising-only market (down daily)")
	}

	// Cold start: filter on but fewer daily closes than period -> trendUp stays false.
	if got := s.Decide(mk([]float64{1, 2})); got.Kind == model.SignalBuy {
		t.Fatalf("unexpected Buy on cold-start daily series")
	}

	// Filter off reproduces prior behavior (still None on this market).
	p0 := testParams()
	p0.TrendFilterPeriod = 0
	if got := NewWithParams(p0).Decide(mk(nil)); got.Kind != model.SignalNone {
		t.Fatalf("filter off changed behavior: got %v, want None", got.Kind)
	}
}
```

- [ ] **Step 7: Run the full package tests**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/ -v`
Expected: PASS, including `TestDefaultParams` (verify it still passes — if it asserts exact field values it may need `TrendFilterPeriod: 200` added; check and update the expected `Params` literal in that test if so).

- [ ] **Step 8: Verify the whole module still builds**

Run: `go build ./...`
Expected: success.

- [ ] **Step 9: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/strategy.go \
        internal/service/trading_strategy/scalping/strategy/rusal/rusal.go \
        internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go
git commit -m "feat(rusal): daily EMA trend filter gating both entries"
```

---

## Task 2: Backtest engine — daily series with no-lookahead alignment

**Files:**
- Modify: `internal/domain/backtest/engine.go`
- Test: `internal/domain/backtest/engine_test.go`

- [ ] **Step 1: Write a failing test for the alignment helper**

In `engine_test.go`, add:

```go
func TestVisibleDailyCloses(t *testing.T) {
	msk, _ := time.LoadLocation("Europe/Moscow")
	day := func(y int, m time.Month, d int) Candle {
		return Candle{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC), Close: float64(d)}
	}
	daily := []Candle{day(2026, 1, 1), day(2026, 1, 2), day(2026, 1, 3)}

	// An intraday bar on Jan 3 sees only Jan 1 and Jan 2 (Jan 3 not yet closed).
	t3 := time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)
	got := visibleDailyCloses(daily, t3, msk)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("visible on Jan 3 = %v, want [1 2]", got)
	}

	// A bar on Jan 1 sees nothing (no prior day closed).
	t1 := time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)
	if got := visibleDailyCloses(daily, t1, msk); len(got) != 0 {
		t.Fatalf("visible on Jan 1 = %v, want []", got)
	}

	// A bar after the last daily sees all three.
	t9 := time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC)
	if got := visibleDailyCloses(daily, t9, msk); len(got) != 3 {
		t.Fatalf("visible on Jan 9 = %v, want 3 closes", got)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestVisibleDailyCloses -v`
Expected: COMPILE FAILURE — `undefined: visibleDailyCloses`.

- [ ] **Step 3: Implement the helper and thread daily into Run**

In `engine.go`, add the import `"time"` (and keep existing imports). Add a package-level Moscow location and the helper:

```go
// mskLoc anchors the trading-day boundary used to decide which daily candles are
// already closed at a given intraday bar. Fallback to UTC if the tz DB is absent.
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// startOfDay returns midnight of t's calendar day in loc.
func startOfDay(t time.Time, loc *time.Location) time.Time {
	tl := t.In(loc)
	return time.Date(tl.Year(), tl.Month(), tl.Day(), 0, 0, 0, 0, loc)
}

// visibleDailyCloses returns closes of daily candles whose calendar day (in loc) is
// strictly before t's calendar day — i.e. days that have fully closed by t. This is
// the no-lookahead rule: the current, still-forming day is never visible.
func visibleDailyCloses(daily []Candle, t time.Time, loc *time.Location) []float64 {
	bound := startOfDay(t, loc)
	out := make([]float64, 0, len(daily))
	for _, c := range daily {
		if c.Time.Before(bound) {
			out = append(out, c.Close)
		}
	}
	return out
}
```

Change the `Run` signature to accept daily candles:

```go
func Run(s strategy.Strategy, candles []Candle, dailyCandles []Candle, cfg Config) Result {
```

Inside the per-bar loop, after `md := buildMarketData(...)` and before `md.Position = ...`, add:

```go
		md.DailyCloses = visibleDailyCloses(dailyCandles, c.Time, mskLoc)
```

Note: `c := candles[i]` is already defined just below in the current code; move the `c := candles[i]` line up so it is available, OR use `candles[i].Time` directly:

```go
		md.DailyCloses = visibleDailyCloses(dailyCandles, candles[i].Time, mskLoc)
```

- [ ] **Step 4: Update existing Run callers in engine_test.go**

Every existing `Run(s, candles, Config{...})` call in `engine_test.go` must pass a daily arg. Existing tests don't exercise the filter, so pass `nil`:

Replace each `Run(s, candles, Config{` with `Run(s, candles, nil, Config{` in `engine_test.go`. (There are several; update all.)

- [ ] **Step 5: Run the engine tests**

Run: `go test ./internal/domain/backtest/ -v`
Expected: PASS — `TestVisibleDailyCloses` plus all existing engine tests (now passing `nil` daily).

- [ ] **Step 6: Add a Run-level gating test**

In `engine_test.go`, add a test that a strategy reading `DailyCloses` is gated by the supplied daily series:

```go
func TestEngineSuppliesDailyCloses(t *testing.T) {
	// 1H candles across two calendar days (Jan 1 and Jan 2, UTC ~ MSK+3 still same days here).
	candles := []Candle{
		{Time: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
		{Time: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
		{Time: time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
	}
	daily := []Candle{
		{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Close: 1},
		{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Close: 2},
	}

	var seen [][]float64
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		seen = append(seen, append([]float64(nil), md.DailyCloses...))
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, daily, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})

	if len(seen) != 3 {
		t.Fatalf("decided %d bars, want 3", len(seen))
	}
	if len(seen[0]) != 0 {
		t.Errorf("bar on Jan 1 daily = %v, want empty", seen[0])
	}
	if len(seen[1]) != 1 || seen[1][0] != 1 {
		t.Errorf("bar on Jan 2 daily = %v, want [1]", seen[1])
	}
	if len(seen[2]) != 2 || seen[2][1] != 2 {
		t.Errorf("bar on Jan 3 daily = %v, want [1 2]", seen[2])
	}
}
```

- [ ] **Step 7: Run it**

Run: `go test ./internal/domain/backtest/ -run TestEngineSuppliesDailyCloses -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): supply no-lookahead daily closes to strategies"
```

---

## Task 3: Calibration + CLI wiring (load and thread daily candles)

**Files:**
- Modify: `internal/service/backtest/calibrate.go`
- Test: `internal/service/backtest/calibrate_test.go`
- Modify: `cmd/backtest/main.go`

- [ ] **Step 1: Update RunGrid signature and its engine call**

In `calibrate.go`, change `RunGrid` to accept daily candles and pass them through:

```go
func RunGrid(b Binding, grid Grid, candles []backtest.Candle, dailyCandles []backtest.Candle,
	cfg backtest.Config, metric string, periodDays float64,
) ([]CalibResult, error) {
```

Inside the combo loop, change the engine call:

```go
		res := backtest.Run(b.Build(params), candles, dailyCandles, cfg)
```

- [ ] **Step 2: Update calibrate_test.go RunGrid calls**

In `calibrate_test.go`, every `RunGrid(b, grid, candles, cfg, ...)` becomes `RunGrid(b, grid, candles, nil, cfg, ...)`. Update all call sites.

- [ ] **Step 3: Run the calibration tests**

Run: `go test ./internal/service/backtest/ -v`
Expected: PASS (now passing `nil` daily).

- [ ] **Step 4: Load Day1 candles in the CLI and thread them through**

In `cmd/backtest/main.go`, in `run(...)`, after the existing H1 `provider.Load(...)` block (around the `candles, err := provider.Load(...)` lines), add the daily load:

```go
	dailyFrom := from.AddDate(-1, 0, 0) // ~250 trading days of lead-in to warm the daily EMA
	dailyCandles, err := provider.Load(ctx, ticker, share.ID, enum.Day1, dailyFrom, to, refresh)
	if err != nil {
		return err
	}
```

Change the `runSingle` / `runCalibration` calls to pass `dailyCandles`:

```go
	if calibratePath != "" {
		return runCalibration(binding, calibratePath, candles, dailyCandles, cfg, metric, periodDays, base,
			metaCommon(ticker, interval, from, to, cfg))
	}

	return runSingle(binding, paramsPath, candles, dailyCandles, cfg, periodDays, base,
		metaCommon(ticker, interval, from, to, cfg))
```

Update `runSingle` signature and its `domain.Run` call:

```go
func runSingle(b svc.Binding, paramsPath string, candles []domain.Candle, dailyCandles []domain.Candle,
	cfg domain.Config, periodDays float64, base string, meta domain.Meta,
) error {
```

and both `domain.Run(b.Build(params), candles, cfg)` → `domain.Run(b.Build(params), candles, dailyCandles, cfg)`.

Note: `runSingle` also calls `b.Build(params).Lookback()` for the candle-count warning — leave that as is.

Update `runCalibration` signature, its `svc.RunGrid` call, and its best-combo `domain.Run` call:

```go
func runCalibration(b svc.Binding, gridPath string, candles []domain.Candle, dailyCandles []domain.Candle,
	cfg domain.Config, metric string, periodDays float64, base string, meta domain.Meta,
) error {
```

- `results, err := svc.RunGrid(b, grid, candles, cfg, metric, periodDays)` → `svc.RunGrid(b, grid, candles, dailyCandles, cfg, metric, periodDays)`
- `res := domain.Run(b.Build(best), candles, cfg)` → `res := domain.Run(b.Build(best), candles, dailyCandles, cfg)`

- [ ] **Step 5: Build the command**

Run: `go build ./cmd/backtest`
Expected: success.

- [ ] **Step 6: Build and vet everything**

Run: `go build ./... && go vet ./...`
Expected: success, no vet warnings from the changed files.

- [ ] **Step 7: Commit**

```bash
git add internal/service/backtest/calibrate.go internal/service/backtest/calibrate_test.go cmd/backtest/main.go
git commit -m "feat(backtest): load Day1 candles and thread them into engine + grid"
```

---

## Task 4: Live runner — fetch daily candles and populate DailyCloses

**Files:**
- Modify: `internal/service/trading_strategy/scalping/trade.go`
- Test: `internal/service/trading_strategy/scalping/trade_test.go` (verify still green; no new behavior asserted because `fakeStrategy` ignores `DailyCloses`)

- [ ] **Step 1: Add a daily fetch + DailyCloses builder in trade.go**

In `trade.go`, inside `Trade`, after the existing 1H candle fetch + `md := buildMarketData(candles)` block and before `sig := st.Decide(md)`, add a daily fetch and populate `md.DailyCloses`. Insert after line that builds `md` (currently `md := buildMarketData(candles)`):

```go
		const dailyLookback = 250 // ~trading days of lead-in to warm the daily EMA(200)
		dailyLimit := int32(dailyLookback)
		time.Sleep(300 * time.Millisecond)
		dailyCandles, dailyErr := s.marketDataClient.GetCandles(ctx, &id, enum.Day1.ToNumberInvestApi(),
			utils.TimeStampPbGenerator(dateNow, -int64(dailyLookback), enum.Day1), timestamppb.New(dateNow), &dailyLimit, true)
		if dailyErr != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: daily candles %s skipped", st.Ticker()))
		} else {
			md.DailyCloses = completedDailyCloses(dailyCandles)
		}
```

Add the `enum` import to the file's import block: `"tinvest/internal/enum"`.

Add the helper at the bottom of `trade.go`:

```go
// completedDailyCloses returns oldest-first closes of completed daily candles only,
// dropping the still-forming current day so the strategy never sees an unclosed bar.
func completedDailyCloses(candles []*imodel.CandleItemTechAnalyse) []float64 {
	out := make([]float64, 0, len(candles))
	for _, c := range candles {
		if !c.IsComplete {
			continue
		}
		out = append(out, utils.CombinePrice(c.Close.Units, c.Close.Nano))
	}
	return out
}
```

- [ ] **Step 2: Build the package**

Run: `go build ./internal/service/trading_strategy/scalping/`
Expected: success.

- [ ] **Step 3: Run the live runner tests**

Run: `go test ./internal/service/trading_strategy/scalping/ -v`
Expected: PASS. The `stubMarket` returns `oneCandle()` for every `GetCandles` (including the daily one); those candles have `IsComplete=false`, so `completedDailyCloses` returns empty, and `fakeStrategy` ignores `DailyCloses` anyway. The `"no position skips instrument before fetching candles"` case still does zero fetches because the daily fetch sits after the SellOnly skip.

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/scalping/trade.go
git commit -m "feat(scalping): fetch daily candles and feed DailyCloses to strategies (live)"
```

---

## Task 5: Full regression + manual validation run

**Files:** none (verification only).

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: PASS across the repo.

- [ ] **Step 2: Reproduce prior behavior with the filter OFF (sanity)**

Requires `T_BANK` in `./env/local.env` or `./env/token.env` and network. Create `data/params/rusal/trend_nofilter.json` as a copy of `data/params/rusal/trend.json` with `"TrendFilterPeriod": 0` added, then:

Run: `go run ./cmd/backtest -ticker RUAL -months 12 -params data/params/rusal/trend_nofilter.json`
Expected: a report under `reports/`; metrics should match the pre-change `trend` profile within rounding (filter off = unchanged logic). This validates the wiring did not alter baseline behavior.

- [ ] **Step 3: Run WITH the filter (default 200) and compare**

Run: `go run ./cmd/backtest -ticker RUAL -months 12 -params data/params/rusal/trend.json`
(Recall: `trend.json` has no `TrendFilterPeriod`, so it inherits the default 200.)
Expected: a report under `reports/`. Compare the SL bucket and max drawdown against the filter-off run; the intent is fewer against-trend long entries and a smaller SL loss bucket.

- [ ] **Step 4: Record findings**

Summarize the before/after metric comparison (profit factor, net PnL, max drawdown, SL bucket, trade count, exposure) back to the user. Do not over-interpret a single window — note whether a follow-up across `-months 6/18/24` is warranted.

---

## Self-Review Notes

- **Spec coverage:** MarketData field (Task 1), param + gate both branches (Task 1), daily EMA rule with cold-start block (Task 1), no-lookahead alignment in MSK (Task 2), Run signature change (Task 2), CLI Day1 load with 1-year lead-in (Task 3), RunGrid signature (Task 3), live runner daily fetch + completed-only (Task 4), backward-compat via `TrendFilterPeriod=0` and ParseParams-over-defaults (Tasks 1 & 5), validation (Task 5). All covered.
- **Backward-compat caveat surfaced:** existing JSON profiles inherit filter 200 on rerun; the filter-off baseline uses an explicit `trend_nofilter.json` (Task 5 Step 2).
- **Type consistency:** `decideInput.trendFilterOn` / `decideInput.trendUp`, `visibleDailyCloses`, `completedDailyCloses`, and the `Run(s, candles, dailyCandles, cfg)` / `RunGrid(b, grid, candles, dailyCandles, cfg, metric, periodDays)` signatures are used identically across tasks.
- **Possible pre-existing test to update:** `TestDefaultParams` in `rusal_test.go` may assert an exact `Params` value; Task 1 Step 7 flags updating it to include `TrendFilterPeriod: 200`.
