# Long-Only Scalping Quality Rework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Raise the per-trade quality of the long-only adaptive scalping strategy by removing the parasite losers (TP-below-entry, bar-1 trail stop-outs, commission churn) and replacing the exposure/count-blind calibration objective — without changing the dual-regime architecture and without adding shorts.

**Architecture:** The pure decision core lives in `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go`. We fix one unconditional bug (Block 1) and add four behavior-preserving knobs on `adaptive.Params` (Blocks 2–3), each a no-op at its zero value so existing behavior is reproduced until a ticker opts in. The backtest layer (`internal/domain/backtest/metrics.go`, `internal/service/backtest/calibrate.go`) gains a Sortino metric and a trade-count floor so calibration stops rewarding rare overfitted combos. Tasks 7–8 add basket + walk-forward validation tooling so the new params are not re-overfit to RUAL.

**Tech Stack:** Go 1.25, standard `testing`, the existing pure backtest engine and `cmd/backtest` runner.

**Behavior-preservation note:** The spec's "byte-for-byte journal" gate is realized here at the decision-core level: every new knob has an explicit "zero value = no-op" unit test (Tasks 2, 3). A full-journal golden is not used because it requires live candle fetches the test suite cannot perform. Block 1 (Task 1) is a deliberate, unconditional behavior change (the bug fix) and is covered by its own regression test.

**Sequencing note:** Tasks 1–6 deliver the strategy improvement and are the core of this plan. Tasks 7–8 add anti-overfitting validation infrastructure (basket + walk-forward); they are independent and may be deferred if the user wants to ship and calibrate Blocks 1–6 first.

---

## File Structure

- `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go` — Blocks 1–3: TP-above-entry fix, trail arming, entry-quality gate, four new `Params` fields, new `entryQualifies` helper.
- `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go` — core decision tests for the above.
- `internal/domain/backtest/types.go` — add `Sortino` field to `Metrics`.
- `internal/domain/backtest/metrics.go` — compute Sortino over per-trade PnL.
- `internal/domain/backtest/metrics_test.go` — Sortino test.
- `internal/service/backtest/calibrate.go` — `sortino` ranking metric, trade-count floor in `rankResults`/`RunGrid`.
- `internal/service/backtest/calibrate_test.go` — floor + sortino ranking tests.
- `internal/service/backtest/registry.go` — `LookupOrGeneric` + `genericDefaults` for basket validation.
- `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`, `.../afks/afks.go` — opt-in starting values for the new knobs; AFKS HTF filter on.
- `internal/service/trading_strategy/scalping/strategy/afks/afks_test.go` — update HTF-on expectation.
- `cmd/backtest/main.go` — `-min-trades` flag, default metric `expectancy`, `-test-months` walk-forward split.
- `internal/service/backtest/candles.go` (or a new `split.go`) — `SplitByTime` helper.
- `data/params/{rusal,afks}/scalp.json`, `data/params/{rusal,afks}/grid.json` — refreshed with new knobs; no `TrendFilterPeriod:0`.

---

## Task 1: Block 1 — Donchian-mid take-profit fires only above entry

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go:161-171`
- Test: `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go`

- [ ] **Step 1: Write the failing test**

Add this standalone test to `adaptive_test.go` (the table helper returns early on `None`, so the phantom-TP assertion needs its own function):

```go
func TestDecide_RangeMidBelowEntryNoPhantomTP(t *testing.T) {
	s := NewWithParams("TST", testParams()) // SLMult 1, ATR via input
	// Open long at 100; channel has slid so mid = (102+96)/2 = 99 < entry.
	in := decideInput{
		price: 99, atr: 2, adx: 15, donUpper: 102, donLower: 96,
		pos: &strategy.Position{PurchasePrice: 100},
	}
	got := s.decide(in)
	if got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None (mid below entry must not TP)", got.Kind)
	}
	if got.TakeProfit != 0 {
		t.Errorf("TakeProfit = %v, want 0 (no phantom target below entry)", got.TakeProfit)
	}
	// Hard stop must still be reported (entry - SLMult*ATR = 100 - 1*2 = 98).
	if got.StopLoss != 98 {
		t.Errorf("StopLoss = %v, want 98", got.StopLoss)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/adaptive/ -run TestDecide_RangeMidBelowEntryNoPhantomTP -v`
Expected: FAIL — current code sets `sig.TakeProfit = mid` (99) and fires `TP` because `price(99) >= mid(99) && mid > 0`.

- [ ] **Step 3: Implement the fix**

In `adaptive.go`, replace the range/dead-zone management block (currently lines 161-171):

```go
		// range or dead zone -> mean-reversion management.
		mid := (in.donUpper + in.donLower) / 2
		sig.StopLoss = hardSL
		// The Donchian mid is a take-profit only when it sits above the entry price.
		// In a sliding channel mid can fall below entry, which previously dumped the
		// position at a loss mislabeled "TP". Below entry, only the hard stop manages.
		validTP := mid > in.pos.PurchasePrice
		if validTP {
			sig.TakeProfit = mid
		}
		switch {
		case in.price <= hardSL:
			sig.Kind, sig.Reason = model.SignalSell, "SL"
		case validTP && in.price >= mid:
			sig.Kind, sig.Reason = model.SignalSell, "TP"
		}
		return sig
```

- [ ] **Step 4: Run the adaptive tests to verify pass**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/adaptive/ -v`
Expected: PASS — the new test passes; existing `TestDecideCore` cases "range exit: take profit at mid" (mid 105 > entry 100) and "range exit: degenerate donchian" (mid 0, None) are unchanged.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go
git commit -m "fix(scalping): Donchian-mid TP fires only above entry"
```

---

## Task 2: Block 2 — Arm the trailing stop only after the trade is in profit

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go:13-30` (Params), `:146-160` (trend management)
- Test: `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go`

- [ ] **Step 1: Write the failing test**

Add to `adaptive_test.go`:

```go
func TestDecide_TrailArmsOnlyInProfit(t *testing.T) {
	p := testParams()
	p.TrailArmATR = 1.0 // arm only after +1 ATR of profit
	s := NewWithParams("TST", p)

	// Not yet armed: profit 0.5 < 1*ATR(2). Price is below the chandelier
	// (chandelierHigh 105 - TrailMult 2*ATR 2 = 101) but trail must NOT fire.
	notArmed := decideInput{
		price: 100.5, atr: 2, adx: 30, chandelierHigh: 105,
		pos: &strategy.Position{PurchasePrice: 100},
	}
	if got := s.decide(notArmed); got.Kind != model.SignalNone {
		t.Fatalf("unarmed trail fired: Kind = %v, want None", got.Kind)
	}

	// Armed: profit 3 >= 1*ATR(2). chandelierHigh 110 - 4 = 106; price 103 <= 106.
	armed := decideInput{
		price: 103, atr: 2, adx: 30, chandelierHigh: 110,
		pos: &strategy.Position{PurchasePrice: 100},
	}
	got := s.decide(armed)
	if got.Kind != model.SignalSell || got.Reason != "TRAIL" {
		t.Fatalf("armed trail did not fire: Kind=%v Reason=%q, want Sell/TRAIL", got.Kind, got.Reason)
	}
	if got.StopLoss != 106 {
		t.Errorf("StopLoss = %v, want 106 (chandelier)", got.StopLoss)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/adaptive/ -run TestDecide_TrailArmsOnlyInProfit -v`
Expected: FAIL to compile — `p.TrailArmATR` is an unknown field.

- [ ] **Step 3: Add the field and arming logic**

In `adaptive.go`, append the field to the `Params` struct (after `TrendFilterPeriod`):

```go
	TrendFilterPeriod int     // daily EMA period for the higher-timeframe long filter; 0 disables
	TrailArmATR       float64 // chandelier trail arms only after price >= entry + TrailArmATR*ATR; <=0 arms immediately
```

Replace the trend management block (currently lines 148-159):

```go
		if reg == regimeTrend {
			chandelier := in.chandelierHigh - s.p.TrailMult*in.atr
			// Report the protective floor even on a hold (mirrors the range branch).
			// Trend has no fixed take-profit — the chandelier trails instead, so TakeProfit stays 0.
			sig.StopLoss = hardSL
			// The trail only arms once the trade is in profit by TrailArmATR*ATR, so a
			// fresh entry near a recent high is not stopped out on the first down-tick.
			// TrailArmATR<=0 arms immediately (preserves the original always-on trail).
			armed := s.p.TrailArmATR <= 0 ||
				in.price >= in.pos.PurchasePrice+s.p.TrailArmATR*in.atr
			switch {
			case in.price <= hardSL:
				sig.Kind, sig.Reason = model.SignalSell, "SL"
			case armed && in.price <= chandelier:
				sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
			}
			return sig
		}
```

- [ ] **Step 4: Run the adaptive tests to verify pass**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/adaptive/ -v`
Expected: PASS — new test passes; existing trend-exit cases run with `TrailArmATR == 0` (default) so `armed` is always true → unchanged behavior (proves the zero-value no-op).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go
git commit -m "feat(scalping): arm chandelier trail only after +TrailArmATR*ATR profit"
```

---

## Task 3: Block 3 — Entry-quality gate (ADX margin, min R:R, min ATR)

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go` (Params, entry switch, new helper)
- Test: `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go`

- [ ] **Step 1: Write the failing test**

Add to `adaptive_test.go`:

```go
func TestDecide_EntryQualityGate(t *testing.T) {
	// Base: a trend entry that fires under testParams (TP 104, SL 98, reward 4 / risk 2 = RR 2).
	base := decideInput{
		price: 100, atr: 2, emaNow: 99, rsiPrev: 40, rsiNow: 46,
		adx: 30, diPlus: 25, diMinus: 10, emaTouched: true,
	}

	t.Run("ADX margin blocks a barely-trending entry", func(t *testing.T) {
		p := testParams()
		p.ADXMargin = 5 // need adx >= ADXTrendLevel(25)+5 = 30
		in := base
		in.adx = 28 // regime still trend (>=25) but below 30
		if got := NewWithParams("TST", p).decide(in); got.Kind != model.SignalNone {
			t.Fatalf("Kind = %v, want None (adx below margin)", got.Kind)
		}
		in.adx = 31
		if got := NewWithParams("TST", p).decide(in); got.Kind != model.SignalBuy {
			t.Fatalf("Kind = %v, want Buy (adx clears margin)", got.Kind)
		}
	})

	t.Run("min reward:risk blocks a thin-edge entry", func(t *testing.T) {
		p := testParams() // TrailMult 2 -> reward 4, risk 2, RR 2
		p.MinRR = 3
		if got := NewWithParams("TST", p).decide(base); got.Kind != model.SignalNone {
			t.Fatalf("Kind = %v, want None (RR 2 < MinRR 3)", got.Kind)
		}
		p.MinRR = 1.5
		if got := NewWithParams("TST", p).decide(base); got.Kind != model.SignalBuy {
			t.Fatalf("Kind = %v, want Buy (RR 2 >= MinRR 1.5)", got.Kind)
		}
	})

	t.Run("min ATR fraction blocks a low-volatility entry", func(t *testing.T) {
		p := testParams() // price 100, atr 2 -> frac 0.02
		p.MinATRFrac = 0.03
		if got := NewWithParams("TST", p).decide(base); got.Kind != model.SignalNone {
			t.Fatalf("Kind = %v, want None (atr frac 0.02 < 0.03)", got.Kind)
		}
		p.MinATRFrac = 0.01
		if got := NewWithParams("TST", p).decide(base); got.Kind != model.SignalBuy {
			t.Fatalf("Kind = %v, want Buy (atr frac 0.02 >= 0.01)", got.Kind)
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/adaptive/ -run TestDecide_EntryQualityGate -v`
Expected: FAIL to compile — `ADXMargin`, `MinRR`, `MinATRFrac` are unknown fields.

- [ ] **Step 3: Add fields, the gate helper, and wire them into entries**

In `adaptive.go`, append to `Params` (after `TrailArmATR`):

```go
	TrailArmATR       float64 // chandelier trail arms only after price >= entry + TrailArmATR*ATR; <=0 arms immediately
	ADXMargin         float64 // entry needs ADX past its regime threshold by this margin; 0 = no extra margin
	MinRR             float64 // reject entry if (target-price) < MinRR*(price-stop); <=0 disables
	MinATRFrac        float64 // reject entry if ATR < MinATRFrac*price (anti-churn); <=0 disables
```

Replace the flat-entry switch (currently lines 174-192) with:

```go
	// Flat -> regime-specific entries (long only), each filtered by the quality gate.
	switch reg {
	case regimeTrend:
		crossedUp := in.rsiPrev < s.p.RSITrendLevel && in.rsiNow >= s.p.RSITrendLevel
		if in.diPlus > in.diMinus && in.emaTouched && crossedUp && in.price > in.emaNow &&
			(!in.trendFilterOn || in.trendUp) && in.adx >= s.p.ADXTrendLevel+s.p.ADXMargin {
			stop := in.price - s.p.SLMult*in.atr
			target := in.price + s.p.TrailMult*in.atr
			if s.entryQualifies(in.price, stop, target, in.atr) {
				sig.Kind, sig.StopLoss, sig.TakeProfit = model.SignalBuy, stop, target
			}
		}
	case regimeRange:
		crossedUp := in.rsiPrev < s.p.RSIRangeLevel && in.rsiNow >= s.p.RSIRangeLevel
		if in.price <= in.donLower*(1+s.p.BandTol) && crossedUp &&
			(!in.trendFilterOn || in.trendUp) && in.adx <= s.p.ADXRangeLevel-s.p.ADXMargin {
			stop := in.price - s.p.SLMult*in.atr
			target := (in.donUpper + in.donLower) / 2
			if s.entryQualifies(in.price, stop, target, in.atr) {
				sig.Kind, sig.StopLoss, sig.TakeProfit = model.SignalBuy, stop, target
			}
		}
	}
	return sig
```

Add the helper near the bottom of the file (after `decide`):

```go
// entryQualifies applies the trade-quality gates: a minimum ATR (anti-churn) and a
// minimum reward:risk ratio. Either check is skipped when its param is <= 0, so the
// zero-value Params reproduce the pre-gate behavior.
func (s *Strategy) entryQualifies(price, stop, target, atr float64) bool {
	if s.p.MinATRFrac > 0 && atr < s.p.MinATRFrac*price {
		return false
	}
	if s.p.MinRR > 0 {
		risk := price - stop
		if risk <= 0 || (target-price) < s.p.MinRR*risk {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run the adaptive tests to verify pass**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/adaptive/ -v`
Expected: PASS — new sub-tests pass; existing entry cases run with `ADXMargin/MinRR/MinATRFrac == 0` → gate is a no-op (zero-value preservation).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go
git commit -m "feat(scalping): entry-quality gate (ADX margin, min R:R, min ATR)"
```

---

## Task 4: Block 4a — Sortino metric over per-trade PnL

**Files:**
- Modify: `internal/domain/backtest/types.go:53-73` (Metrics struct)
- Modify: `internal/domain/backtest/metrics.go:30-40` (Compute)
- Test: `internal/domain/backtest/metrics_test.go`

- [ ] **Step 1: Write the failing test**

Add to `metrics_test.go`:

```go
func TestComputeSortino(t *testing.T) {
	// Trades 100,-40,60,-20: mean=25; downside sq mean = (40^2+20^2)/4 = 500;
	// downside dev = sqrt(500) = 22.360679...; Sortino = 25 / 22.360679 = 1.118034.
	r := Result{
		Trades:      []Trade{{PnL: 100}, {PnL: -40}, {PnL: 60}, {PnL: -20}},
		InitialCash: 1000, FinalEquity: 1100,
	}
	m := Compute(r, 0, 0, 0)
	if !approx(m.Sortino, 25.0/math.Sqrt(500)) {
		t.Fatalf("Sortino = %f, want %f", m.Sortino, 25.0/math.Sqrt(500))
	}
}

func TestComputeSortinoNoDownside(t *testing.T) {
	// All winners: downside dev 0, mean > 0 -> Sortino = mean (mirrors PF convention).
	r := Result{Trades: []Trade{{PnL: 50}, {PnL: 30}}, InitialCash: 1000, FinalEquity: 1080}
	m := Compute(r, 0, 0, 0)
	if !approx(m.Sortino, 40) { // mean = (50+30)/2
		t.Fatalf("Sortino = %f, want 40", m.Sortino)
	}
}
```

Note: `metrics_test.go` already imports `math` (it references `math.Pi`).

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestComputeSortino -v`
Expected: FAIL to compile — `m.Sortino` is an unknown field.

- [ ] **Step 3: Add the field and compute it**

In `types.go`, add to the `Metrics` struct after `Expectancy`:

```go
	Expectancy     float64 // average PnL per trade
	Sortino        float64 // mean trade PnL / downside deviation of trade PnL
```

In `metrics.go`, inside `Compute`, after the `Expectancy` assignment (line 33), add:

```go
	m.Sortino = sortino(r.Trades, m.Expectancy)
```

Add this function to `metrics.go` (below `maxDrawdown`):

```go
// sortino returns mean trade PnL divided by the downside deviation of trade PnL
// (the root-mean-square of the negative PnLs). With no losing trades the downside
// deviation is zero; a positive mean then returns the mean itself (mirrors the
// ProfitFactor convention), otherwise 0.
func sortino(trades []Trade, mean float64) float64 {
	n := len(trades)
	if n == 0 {
		return 0
	}
	var sqSum float64
	for _, t := range trades {
		if t.PnL < 0 {
			sqSum += t.PnL * t.PnL
		}
	}
	dd := math.Sqrt(sqSum / float64(n))
	switch {
	case dd > 0:
		return mean / dd
	case mean > 0:
		return mean
	default:
		return 0
	}
}
```

- [ ] **Step 4: Run the domain backtest tests to verify pass**

Run: `go test ./internal/domain/backtest/ -v`
Expected: PASS — both Sortino tests pass; existing metrics tests unchanged.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/types.go internal/domain/backtest/metrics.go internal/domain/backtest/metrics_test.go
git commit -m "feat(backtest): add Sortino metric over per-trade PnL"
```

---

## Task 5: Block 4b — Calibration: trade-count floor, sortino ranking, expectancy default

**Files:**
- Modify: `internal/service/backtest/calibrate.go` (`RunGrid`, `rankResults`, `metricValue`, `supportedMetrics`)
- Modify: `internal/service/backtest/calibrate_test.go` (call-site signature, new tests)
- Modify: `cmd/backtest/main.go` (`-min-trades` flag, default metric, RunGrid call)

- [ ] **Step 1: Write the failing tests**

In `calibrate_test.go`, replace `TestRunGridRanksByMetric` with a version that passes the floor and add two new tests:

```go
func TestRunGridRanksByMetric(t *testing.T) {
	in := []CalibResult{
		{Metrics: backtest.Metrics{ProfitFactor: 1.2, MaxDrawdown: 500, TotalTrades: 30}},
		{Metrics: backtest.Metrics{ProfitFactor: 2.5, MaxDrawdown: 900, TotalTrades: 30}},
		{Metrics: backtest.Metrics{ProfitFactor: 0.8, MaxDrawdown: 100, TotalTrades: 30}},
	}
	byPF := rankResults(append([]CalibResult(nil), in...), "profit_factor", 0)
	if byPF[0].Metrics.ProfitFactor != 2.5 {
		t.Fatalf("top PF = %f, want 2.5", byPF[0].Metrics.ProfitFactor)
	}
	byDD := rankResults(append([]CalibResult(nil), in...), "max_drawdown", 0)
	if byDD[0].Metrics.MaxDrawdown != 100 {
		t.Fatalf("top DD = %f, want 100", byDD[0].Metrics.MaxDrawdown)
	}
}

func TestRankResultsMinTradesFloor(t *testing.T) {
	in := []CalibResult{
		{Metrics: backtest.Metrics{ProfitFactor: 3.0, TotalTrades: 2}},  // high PF, too few trades
		{Metrics: backtest.Metrics{ProfitFactor: 1.2, TotalTrades: 25}}, // qualified
	}
	got := rankResults(append([]CalibResult(nil), in...), "profit_factor", 10)
	if got[0].Metrics.TotalTrades != 25 {
		t.Fatalf("top trades = %d, want 25 (qualified combo ranks ahead of a 2-trade fluke)", got[0].Metrics.TotalTrades)
	}
}

func TestRankResultsSortino(t *testing.T) {
	in := []CalibResult{
		{Metrics: backtest.Metrics{Sortino: 0.5, TotalTrades: 30}},
		{Metrics: backtest.Metrics{Sortino: 1.5, TotalTrades: 30}},
	}
	got := rankResults(append([]CalibResult(nil), in...), "sortino", 0)
	if got[0].Metrics.Sortino != 1.5 {
		t.Fatalf("top sortino = %f, want 1.5", got[0].Metrics.Sortino)
	}
}
```

Also update the existing `RunGrid` call in `TestRunGridCartesianProduct` to pass the new floor arg (append `, 0` before the metric/periodDays — see Step 3 for the final signature):

```go
	results, err := RunGrid(b, grid, tinyCandles(400), nil, cfg, "profit_factor", 0, 16)
```

And in `TestRunGridUnknownMetricErrors` / `TestRunGridUnknownFieldErrors`, add the `0` floor arg the same way:

```go
	if _, err := RunGrid(b, Grid{}, tinyCandles(400), nil, cfg, "sharpe", 0, 16); err == nil {
```
```go
	if _, err := RunGrid(b, Grid{"Bogus": {1, 2}}, tinyCandles(400), nil, cfg, "profit_factor", 0, 16); err == nil {
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TestRunGridRanksByMetric|TestRankResults' -v`
Expected: FAIL to compile — `rankResults` takes 2 args, `RunGrid` lacks the floor param, `"sortino"` is not yet a metric.

- [ ] **Step 3: Implement the floor, sortino metric, and signature changes**

In `calibrate.go`, change `RunGrid` to thread a `minTrades` arg (insert it before `metric`):

```go
func RunGrid(b Binding, grid Grid, candles []backtest.Candle, dailyCandles []backtest.Candle,
	cfg backtest.Config, metric string, minTrades int, periodDays float64,
) ([]CalibResult, error) {
	if err := validateMetric(metric); err != nil {
		return nil, err
	}
	combos, err := expandGrid(b.DefaultParams(), grid)
	if err != nil {
		return nil, err
	}
	results := make([]CalibResult, 0, len(combos))
	for _, params := range combos {
		res := backtest.Run(b.Build(params), candles, dailyCandles, cfg)
		m := backtest.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)
		results = append(results, CalibResult{Params: params, Metrics: m})
	}
	return rankResults(results, metric, minTrades), nil
}
```

Replace `rankResults` with a floor-aware version:

```go
// rankResults sorts best-first. Combos with fewer than minTrades trades are treated
// as statistically unreliable and sink below all qualified combos, regardless of
// their metric. Within each group: ascending for max_drawdown, descending otherwise.
func rankResults(results []CalibResult, metric string, minTrades int) []CalibResult {
	qualifies := func(m backtest.Metrics) bool { return m.TotalTrades >= minTrades }
	sort.SliceStable(results, func(i, j int) bool {
		qi, qj := qualifies(results[i].Metrics), qualifies(results[j].Metrics)
		if qi != qj {
			return qi // qualified ranks ahead of unqualified
		}
		a, b := metricValue(results[i].Metrics, metric), metricValue(results[j].Metrics, metric)
		if metric == "max_drawdown" {
			return a < b
		}
		return a > b
	})
	return results
}
```

Add `sortino` to `metricValue` (before the `default`):

```go
	case "sortino":
		return m.Sortino
```

Add `sortino` to `supportedMetrics`:

```go
var supportedMetrics = map[string]struct{}{
	"profit_factor": {}, "net_pnl": {}, "win_rate": {}, "max_drawdown": {}, "expectancy": {}, "sortino": {},
}
```

In `cmd/backtest/main.go`, change the metric default and add the flag (lines 39-41 region):

```go
		metric     = flag.String("metric", "expectancy", "ranking metric: profit_factor|net_pnl|win_rate|max_drawdown|expectancy|sortino")
		minTrades  = flag.Int("min-trades", 15, "calibration: combos with fewer trades sink below qualified ones")
```

Thread `*minTrades` into `run(...)` and `runCalibration(...)`. Update `run`'s signature and the call in `main`, and the `RunGrid` call in `runCalibration`:

```go
	results, err := svc.RunGrid(b, grid, candles, dailyCandles, cfg, metric, minTrades, periodDays)
```

(Plumb `minTrades int` through `run` → `runCalibration`; `runSingle` does not need it.)

- [ ] **Step 4: Run the service backtest tests and build the command**

Run: `go test ./internal/service/backtest/ -v && go build ./cmd/backtest/`
Expected: PASS and a clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/calibrate.go internal/service/backtest/calibrate_test.go cmd/backtest/main.go
git commit -m "feat(backtest): trade-count floor + sortino metric, expectancy default"
```

---

## Task 6: Per-ticker opt-in values + refreshed params files

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`
- Modify: `internal/service/trading_strategy/scalping/strategy/afks/afks.go`
- Modify: `internal/service/trading_strategy/scalping/strategy/afks/afks_test.go`
- Modify: `data/params/rusal/scalp.json`, `data/params/rusal/grid.json`
- Modify: `data/params/afks/scalp.json`, `data/params/afks/grid.json`

- [ ] **Step 1: Update the AFKS test to expect HTF-on**

In `afks_test.go`, replace the `TrendFilterPeriod` assertion:

```go
	if p.TrendFilterPeriod <= 0 {
		t.Errorf("TrendFilterPeriod = %d, want > 0 (HTF filter on by default)", p.TrendFilterPeriod)
	}
	if p.TrailArmATR <= 0 || p.MinRR <= 0 || p.MinATRFrac <= 0 {
		t.Errorf("quality knobs must be opted in (non-zero): %+v", p)
	}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/afks/ -v`
Expected: FAIL — AFKS `DefaultParams` still has `TrendFilterPeriod: 0` and zero quality knobs.

- [ ] **Step 3: Opt the tickers into the new knobs**

In `rusal.go`, extend the returned `adaptive.Params` (keep existing fields, add after `TrendFilterPeriod`):

```go
		TrendFilterPeriod: 100, // calibrated: beats 0/50/200 across 6/12/18/24mo windows
		TrailArmATR:       1.0, // arm trail after +1 ATR profit (kills bar-1 stop-outs)
		ADXMargin:         2.0, // require ADX to clear its regime threshold by 2
		MinRR:             1.5, // skip setups whose target is < 1.5x the risk
		MinATRFrac:        0.003, // skip sub-0.3%-ATR setups (anti-churn)
```

In `afks.go`, change the doc comment and the params so the HTF filter is on and the knobs are opted in:

```go
// DefaultParams returns generic, NOT-yet-calibrated starting values for AFKS with the
// quality knobs opted in and the HTF daily trend filter on. Calibration refines these.
func DefaultParams() adaptive.Params {
	return adaptive.Params{
		EMAPeriod:         21,
		ADXPeriod:         14,
		ADXTrendLevel:     25,
		ADXRangeLevel:     20,
		RSIPeriod:         14,
		RSITrendLevel:     45,
		RSIRangeLevel:     35,
		PullbackWindow:    5,
		DonchianPeriod:    20,
		ATRPeriod:         14,
		SLMult:            1.0,
		TrailMult:         2.5,
		ChandelierWindow:  20,
		EMATouchTol:       0.002,
		BandTol:           0.003,
		TrendFilterPeriod: 100,
		TrailArmATR:       1.0,
		ADXMargin:         2.0,
		MinRR:             1.5,
		MinATRFrac:        0.003,
	}
}
```

- [ ] **Step 4: Refresh the params JSON files**

Overwrite `data/params/rusal/scalp.json` (add the new knobs to the existing tuned profile):

```json
{
  "EMAPeriod": 13,
  "ADXPeriod": 10,
  "ADXTrendLevel": 22,
  "ADXRangeLevel": 18,
  "RSIPeriod": 9,
  "RSITrendLevel": 50,
  "RSIRangeLevel": 40,
  "PullbackWindow": 4,
  "DonchianPeriod": 14,
  "ATRPeriod": 10,
  "SLMult": 1.0,
  "TrailMult": 2.0,
  "ChandelierWindow": 14,
  "EMATouchTol": 0.002,
  "BandTol": 0.003,
  "TrendFilterPeriod": 100,
  "TrailArmATR": 1.0,
  "ADXMargin": 2.0,
  "MinRR": 1.5,
  "MinATRFrac": 0.003
}
```

Overwrite `data/params/rusal/grid.json` (sweep the new quality knobs; keep the product modest):

```json
{
  "TrailArmATR": [0.5, 1.0, 1.5],
  "ADXMargin": [0, 2, 4],
  "MinRR": [1.0, 1.5, 2.0],
  "MinATRFrac": [0.002, 0.003, 0.004]
}
```

Overwrite `data/params/afks/scalp.json` (mirror AFKS defaults; HTF on, no `TrendFilterPeriod:0`):

```json
{
  "EMAPeriod": 21,
  "ADXPeriod": 14,
  "ADXTrendLevel": 25,
  "ADXRangeLevel": 20,
  "RSIPeriod": 14,
  "RSITrendLevel": 45,
  "RSIRangeLevel": 35,
  "PullbackWindow": 5,
  "DonchianPeriod": 20,
  "ATRPeriod": 14,
  "SLMult": 1.0,
  "TrailMult": 2.5,
  "ChandelierWindow": 20,
  "EMATouchTol": 0.002,
  "BandTol": 0.003,
  "TrendFilterPeriod": 100,
  "TrailArmATR": 1.0,
  "ADXMargin": 2.0,
  "MinRR": 1.5,
  "MinATRFrac": 0.003
}
```

Overwrite `data/params/afks/grid.json`:

```json
{
  "TrailArmATR": [0.5, 1.0, 1.5],
  "ADXMargin": [0, 2, 4],
  "MinRR": [1.0, 1.5, 2.0],
  "MinATRFrac": [0.002, 0.003, 0.004]
}
```

- [ ] **Step 5: Run the strategy + registry tests**

Run: `go test ./internal/service/trading_strategy/scalping/... ./internal/service/backtest/ -v`
Expected: PASS — AFKS test now expects HTF-on; `rusal_test` (`TrendFilterPeriod == 100`, `Lookback == 154`) and `registry_test` (`ParamRows EMAPeriod=21`) are unaffected by the appended fields.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/rusal/rusal.go internal/service/trading_strategy/scalping/strategy/afks/afks.go internal/service/trading_strategy/scalping/strategy/afks/afks_test.go data/params/rusal/scalp.json data/params/rusal/grid.json data/params/afks/scalp.json data/params/afks/grid.json
git commit -m "feat(scalping): opt RUAL/AFKS into quality knobs, HTF on, refresh grids"
```

---

## Task 7: Generic ticker binding for basket validation

**Files:**
- Modify: `internal/service/backtest/registry.go` (`genericDefaults`, `LookupOrGeneric`)
- Modify: `internal/service/backtest/registry_test.go` (new test)
- Modify: `cmd/backtest/main.go` (use `LookupOrGeneric`)

- [ ] **Step 1: Write the failing test**

Add to `registry_test.go`:

```go
func TestLookupOrGenericFallback(t *testing.T) {
	// A registered ticker keeps its own binding.
	b := LookupOrGeneric("RUAL")
	if s := b.Build(b.DefaultParams()); s.Ticker() != "RUAL" {
		t.Fatalf("RUAL built ticker = %q, want RUAL", s.Ticker())
	}
	// An unregistered ticker gets a generic binding bound to that ticker.
	g := LookupOrGeneric("SBER")
	if g.DefaultParams == nil || g.Build == nil || g.ParseParams == nil {
		t.Fatal("generic binding has nil funcs")
	}
	s := g.Build(g.DefaultParams())
	if s.Ticker() != "SBER" {
		t.Fatalf("generic built ticker = %q, want SBER", s.Ticker())
	}
	// Generic defaults must keep the HTF filter on and the quality knobs opted in.
	p := g.DefaultParams().(adaptive.Params)
	if p.TrendFilterPeriod <= 0 || p.MinRR <= 0 || p.TrailArmATR <= 0 {
		t.Fatalf("generic defaults must opt into quality knobs: %+v", p)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestLookupOrGenericFallback -v`
Expected: FAIL to compile — `LookupOrGeneric` is undefined.

- [ ] **Step 3: Implement the generic binding**

In `registry.go`, add the generic defaults and the fallback lookup (leave `Lookup` unchanged so `TestLookupKnownAndUnknown` still sees `NOPE` miss):

```go
// genericDefaults are neutral baseline params for tickers without a dedicated config,
// used for basket validation. HTF filter on; quality knobs opted in at starting values.
func genericDefaults() adaptive.Params {
	return adaptive.Params{
		EMAPeriod: 21, ADXPeriod: 14, ADXTrendLevel: 25, ADXRangeLevel: 20,
		RSIPeriod: 14, RSITrendLevel: 45, RSIRangeLevel: 35,
		PullbackWindow: 5, DonchianPeriod: 20, ATRPeriod: 14,
		SLMult: 1.0, TrailMult: 2.5, ChandelierWindow: 20,
		EMATouchTol: 0.002, BandTol: 0.003, TrendFilterPeriod: 100,
		TrailArmATR: 1.0, ADXMargin: 2.0, MinRR: 1.5, MinATRFrac: 0.003,
	}
}

// LookupOrGeneric returns the registered binding for a ticker, or a generic binding
// bound to that ticker (with genericDefaults) when none is registered. This lets the
// backtest command validate the strategy on any ticker without a dedicated package.
func LookupOrGeneric(ticker string) Binding {
	if b, ok := registry[ticker]; ok {
		return b
	}
	return bindingFor(ticker, genericDefaults)
}
```

In `cmd/backtest/main.go`, replace the registry lookup + error guard (lines 71-74):

```go
	binding := svc.LookupOrGeneric(ticker)
```

(Remove the now-dead `ok`/`!ok` error block; `resolveShare` still validates the ticker exists at the broker.)

- [ ] **Step 4: Run the tests and build**

Run: `go test ./internal/service/backtest/ -v && go build ./cmd/backtest/`
Expected: PASS and clean build; `TestLookupKnownAndUnknown` still passes (`Lookup` unchanged).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/registry.go internal/service/backtest/registry_test.go cmd/backtest/main.go
git commit -m "feat(backtest): generic ticker binding for basket validation"
```

---

## Task 8: Walk-forward split in the backtest command

**Files:**
- Create: `internal/service/backtest/split.go`
- Create: `internal/service/backtest/split_test.go`
- Modify: `cmd/backtest/main.go` (`-test-months` flag, calibrate-on-train/report-on-test)

- [ ] **Step 1: Write the failing test**

Create `internal/service/backtest/split_test.go`:

```go
package backtest

import (
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
)

func TestSplitByTime(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]backtest.Candle, 10)
	for i := range candles {
		candles[i] = backtest.Candle{Time: base.AddDate(0, 0, i)}
	}
	boundary := base.AddDate(0, 0, 6) // first 6 days train, rest test

	train, test := SplitByTime(candles, boundary)
	if len(train) != 6 || len(test) != 4 {
		t.Fatalf("split sizes = %d/%d, want 6/4", len(train), len(test))
	}
	if !train[len(train)-1].Time.Before(boundary) {
		t.Error("last train candle must be before boundary")
	}
	if test[0].Time.Before(boundary) {
		t.Error("first test candle must be at/after boundary")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestSplitByTime -v`
Expected: FAIL to compile — `SplitByTime` is undefined.

- [ ] **Step 3: Implement the split helper**

Create `internal/service/backtest/split.go`:

```go
package backtest

import (
	"time"

	"tinvest/internal/domain/backtest"
)

// SplitByTime partitions oldest-first candles into a train slice (strictly before
// boundary) and a test slice (at/after boundary). Either slice may be empty.
func SplitByTime(candles []backtest.Candle, boundary time.Time) (train, test []backtest.Candle) {
	for i, c := range candles {
		if !c.Time.Before(boundary) {
			return candles[:i], candles[i:]
		}
	}
	return candles, nil
}
```

- [ ] **Step 4: Wire walk-forward into the command**

In `cmd/backtest/main.go`, add the flag (in the `flag` block):

```go
		testMonths = flag.Int("test-months", 0, "walk-forward: calibrate on the earlier window, report best on the last N months")
```

Plumb `*testMonths` through `run` → `runCalibration`. In `runCalibration`, when `testMonths > 0`, split both candle series at `to.AddDate(0, -testMonths, 0)`, run the grid on the train slices, and produce the `_best.md` report on the test slices:

```go
func runCalibration(b svc.Binding, gridPath string, candles []domain.Candle, dailyCandles []domain.Candle,
	cfg domain.Config, metric string, minTrades, testMonths int, to time.Time, periodDays float64, base string, meta domain.Meta,
) error {
	raw, err := os.ReadFile(gridPath)
	if err != nil {
		return fmt.Errorf("read grid: %w", err)
	}
	var grid svc.Grid
	if err := json.Unmarshal(raw, &grid); err != nil {
		return fmt.Errorf("parse grid: %w", err)
	}

	gridCandles, gridDaily := candles, dailyCandles
	bestCandles, bestDaily := candles, dailyCandles
	bestDays := periodDays
	if testMonths > 0 {
		boundary := to.AddDate(0, -testMonths, 0)
		gridCandles, bestCandles = svc.SplitByTime(candles, boundary)
		gridDaily, _ = svc.SplitByTime(dailyCandles, boundary)
		_, bestDaily = svc.SplitByTime(dailyCandles, boundary)
		bestDays = float64(testMonths) * 30.0
	}

	results, err := svc.RunGrid(b, grid, gridCandles, gridDaily, cfg, metric, minTrades, periodDays)
	if err != nil {
		return err
	}
	calibPath := base + "_calibration.md"
	if err := writeFile(calibPath, svc.RenderCalibrationMarkdown(metric, results, 20)); err != nil {
		return err
	}

	if len(results) > 0 {
		best := results[0].Params
		res := domain.Run(b.Build(best), bestCandles, bestDaily, cfg)
		m := domain.Compute(res, res.BarsInMarket, len(res.Equity), bestDays)
		meta.Params = svc.ParamRows(best)
		meta.OpenPosition = openPosition(res)
		if err := writeFile(base+"_best.md", domain.RenderMarkdown(meta, m, res.Trades, res.Equity)); err != nil {
			return err
		}
	}
	fmt.Printf("calibration: %s (combos=%d, test_months=%d)\n", calibPath, len(results), testMonths)
	return nil
}
```

Update the `runCalibration(...)` call in `run` to pass `*minTrades`, `*testMonths`, and `to`.

- [ ] **Step 5: Run the tests and build**

Run: `go test ./internal/service/backtest/ -v && go build ./cmd/backtest/`
Expected: PASS and clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/service/backtest/split.go internal/service/backtest/split_test.go cmd/backtest/main.go
git commit -m "feat(backtest): walk-forward split (calibrate on train, report on test)"
```

---

## Final verification

- [ ] **Run the full suite**

Run: `go test ./... && go build ./...`
Expected: all packages PASS and build clean.

---

## Operational validation & calibration run (requires API token; not a TDD step)

This is run by the engineer after Tasks 1–8 land. It needs `T_BANK` in `./env/local.env` and network access; candles are cached under `data/candles/` (gitignored).

- [ ] **Baseline diff on RUAL/AFKS** — confirm the bug fixes reduced churn losers:

```bash
go run ./cmd/backtest -ticker RUAL -months 24 -metric expectancy
go run ./cmd/backtest -ticker AFKS -months 24 -metric expectancy
```

Inspect `reports/RUAL_*_trades.csv` / `reports/AFKS_*_trades.csv`: the `TP`-labeled negative rows and the 1-bar `TRAIL` rows from the prior journals should be gone.

- [ ] **Walk-forward calibration per ticker** — calibrate on the earlier window, judge on the last 6 months:

```bash
go run ./cmd/backtest -ticker RUAL -months 24 -calibrate data/params/rusal/grid.json -metric expectancy -min-trades 15 -test-months 6
go run ./cmd/backtest -ticker AFKS -months 24 -calibrate data/params/afks/grid.json -metric expectancy -min-trades 15 -test-months 6
```

Read each `reports/*_best.md`: acceptance is out-of-sample (test window) expectancy > 0 and profit factor > 1.2; trade count is unconstrained (few is fine).

- [ ] **Basket sweep (anti-overfit)** — run the generic binding across 5–10 liquid MOEX tickers to confirm the edge is not RUAL-specific:

```bash
for T in RUAL AFKS SBER GAZP LKOH GMKN MGNT MOEX; do \
  go run ./cmd/backtest -ticker "$T" -months 24 -calibrate data/params/rusal/grid.json -metric expectancy -min-trades 15 -test-months 6; \
done
```

Record the per-ticker out-of-sample expectancy/PF in `reports/cmp/SUMMARY.md`. If the median test-window expectancy across the basket is positive, the rework holds; otherwise iterate on the gate thresholds, not the metric.

- [ ] **Fold any winning per-ticker params** back into `data/params/<ticker>/scalp.json` and, for promoted tickers, a dedicated `strategy/<ticker>` config package mirroring `rusal`.
