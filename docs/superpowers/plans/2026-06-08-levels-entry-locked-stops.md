# Levels Entry-Locked Protective Stops — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the levels hard stop and chandelier trail actually fire by freezing the hard stop at entry and tracking a monotonic favourable-price maximum, both carried through `Position` and populated by the backtest engine/portfolio.

**Architecture:** The backtest portfolio already remembers entry-time state. Extend it with the frozen stop and a running favourable-price max; expose both (plus entry ATR) through the shared `strategy.Position`. The levels core stops recomputing protective levels each bar from a self-including window and instead reads the frozen values from `Position`. Live wiring is deferred (documented gap in the spec).

**Tech Stack:** Go 1.25, standard `testing` package, table-driven tests.

**Spec:** `docs/superpowers/specs/2026-06-08-levels-entry-locked-stops-design.md`

---

## File Structure

- `internal/service/trading_strategy/scalping/strategy/strategy.go` — shared `Position` contract; add entry-context fields.
- `internal/domain/backtest/portfolio.go` — store frozen stop + running favourable max; expose via `strategyPosition()`.
- `internal/domain/backtest/engine.go` — pass the entry stop into `open`; mark the favourable max each bar.
- `internal/service/trading_strategy/levels/strategy/core/core.go` — management branch reads frozen stop + monotonic arm.
- Tests: `internal/domain/backtest/portfolio_test.go`, `internal/domain/backtest/engine_test.go`, `internal/service/trading_strategy/levels/strategy/core/core_test.go`.

---

## Task 1: Extend the `Position` contract

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go:5-9`

No unit test: this is a purely additive struct change with no behaviour. It is exercised by the downstream tasks and verified by `go build`.

- [ ] **Step 1: Add entry-context fields to `Position`**

Replace the struct (currently lines 5-9):

```go
// Position is an open long position in the strategy's instrument.
type Position struct {
	PurchasePrice float64
	Quantity      int64
	// StopLoss is the hard stop frozen at entry. Zero means "not set" (e.g. live
	// trading, which does not yet persist entry state — see the levels
	// entry-locked-stops spec). The backtest engine always populates it.
	StopLoss float64
	// EntryATR is the ATR captured at entry, used as the arm threshold unit.
	EntryATR float64
	// MaxFavorablePrice is the highest close seen since entry (monotonic
	// non-decreasing); it makes the trail's arming latch monotonic.
	MaxFavorablePrice float64
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: builds clean (scalping `trade.go` and backtest `portfolio.go` set the existing fields by name, so the additions are non-breaking).

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/strategy.go
git commit -m "feat(levels): add entry-context fields to Position"
```

---

## Task 2: Portfolio freezes the entry stop and tracks the favourable max

**Files:**
- Modify: `internal/domain/backtest/portfolio.go:10-95`
- Modify: `internal/domain/backtest/engine.go:55-64`
- Test: `internal/domain/backtest/portfolio_test.go`

After this task the engine stores the new state but the levels core does not read it yet, so observable behaviour is unchanged. All existing tests stay green once the `open()` call sites are updated for the new signature.

- [ ] **Step 1: Write the failing portfolio tests**

Add to `internal/domain/backtest/portfolio_test.go`:

```go
func TestPortfolioOpenFreezesStopAndSeedsFavorable(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.open(100, time.Unix(0, 0), 0, 0, 2, 95) // atr 2, stop 95
	pos := p.strategyPosition()
	if pos == nil {
		t.Fatal("expected a position after open")
	}
	if !approx(pos.StopLoss, 95) {
		t.Errorf("StopLoss = %v, want 95 (frozen at entry)", pos.StopLoss)
	}
	if !approx(pos.EntryATR, 2) {
		t.Errorf("EntryATR = %v, want 2", pos.EntryATR)
	}
	if !approx(pos.MaxFavorablePrice, 100) {
		t.Errorf("MaxFavorablePrice = %v, want 100 (seeded to entry price)", pos.MaxFavorablePrice)
	}
}

func TestPortfolioMarkRaisesFavorableMonotonically(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.open(100, time.Unix(0, 0), 0, 0, 1, 95) // maxFavorable seeded to 100
	p.mark(105)
	if !approx(p.maxFavorable, 105) {
		t.Fatalf("maxFavorable = %v, want 105 after mark(105)", p.maxFavorable)
	}
	p.mark(103) // lower -> no change
	if !approx(p.maxFavorable, 105) {
		t.Fatalf("maxFavorable = %v, want 105 (monotonic, mark(103) is a no-op)", p.maxFavorable)
	}
}

func TestPortfolioMarkIsNoOpWhenFlat(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.mark(200) // flat: must not record anything
	if !approx(p.maxFavorable, 0) {
		t.Fatalf("maxFavorable = %v, want 0 (mark while flat is a no-op)", p.maxFavorable)
	}
}

func TestPortfolioCloseResetsEntryState(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.bar = 1
	p.open(100, time.Unix(0, 0), 0, 0, 1, 95)
	p.mark(110)
	p.bar = 2
	p.close(108, time.Unix(1, 0), "TRAIL")
	if !approx(p.entryStop, 0) || !approx(p.maxFavorable, 0) {
		t.Fatalf("entryStop=%v maxFavorable=%v, want 0/0 after close", p.entryStop, p.maxFavorable)
	}
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

Run: `go test ./internal/domain/backtest/ -run 'TestPortfolio(OpenFreezesStop|MarkRaises|MarkIsNoOp|CloseResetsEntry)' -v`
Expected: compile failure — `p.open` takes 5 args not 6, and `p.mark` / `p.entryStop` / `p.maxFavorable` do not exist yet.

- [ ] **Step 3: Add the portfolio fields, the `stop` param, `mark`, and resets**

In `internal/domain/backtest/portfolio.go`, add the two fields to the struct (after `entryATR`):

```go
	entryATR    float64 // ATR captured at entry
	entryStop   float64 // hard stop frozen at entry
	maxFavorable float64 // highest close seen since entry (monotonic)
	bar         int     // current bar index, set by the engine each iteration
```

Change `open` to accept and store the stop, and seed `maxFavorable` (signature line and the assignment block):

```go
func (p *portfolio) open(price float64, t time.Time, level, target, atr, stop float64) {
	if p.qty != 0 {
		return
	}
	lotCost := price * float64(p.cfg.Lot) * (1 + p.cfg.Commission)
	if lotCost <= 0 {
		return
	}
	budget := p.cfg.Fraction * p.cash
	lots := int64(math.Floor(budget / lotCost))
	if lots <= 0 {
		return
	}
	qty := lots * int64(p.cfg.Lot)
	cost := float64(qty) * price
	commission := cost * p.cfg.Commission
	p.cash -= cost + commission
	p.qty = qty
	p.entryPrice = price
	p.entryTime = t
	p.entryBar = p.bar
	p.entryLevel = level
	p.entryTarget = target
	p.entryATR = atr
	p.entryStop = stop
	p.maxFavorable = price
}
```

Add the `mark` method (place it right after `open`):

```go
// mark raises the running favourable-price maximum toward price. No-op when flat
// or when price is not a new high, which keeps maxFavorable monotonic.
func (p *portfolio) mark(price float64) {
	if p.qty == 0 {
		return
	}
	if price > p.maxFavorable {
		p.maxFavorable = price
	}
}
```

In `close`, reset the two new fields alongside the existing resets:

```go
	p.qty = 0
	p.entryPrice = 0
	p.entryLevel = 0
	p.entryTarget = 0
	p.entryATR = 0
	p.entryStop = 0
	p.maxFavorable = 0
	return tr
```

Expose the new fields in `strategyPosition`:

```go
func (p *portfolio) strategyPosition() *strategy.Position {
	if p.qty == 0 {
		return nil
	}
	return &strategy.Position{
		PurchasePrice:     p.entryPrice,
		Quantity:          p.qty,
		StopLoss:          p.entryStop,
		EntryATR:          p.entryATR,
		MaxFavorablePrice: p.maxFavorable,
	}
}
```

- [ ] **Step 4: Update the engine to pass the stop and mark each bar**

In `internal/domain/backtest/engine.go`, inside the loop, add the mark call before building `Position` and pass `sig.StopLoss` to `open`. The relevant section becomes:

```go
		p.bar = i
		md := buildMarketData(candles[i-l+1 : i+1])
		md.DailyCloses = visibleDailyCloses(dailyCandles, candles[i].Time, mskLoc)
		if p.qty != 0 {
			p.mark(candles[i].Close)
		}
		md.Position = p.strategyPosition()

		c := candles[i]
		sig := s.Decide(md)
		switch sig.Kind {
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR, sig.StopLoss)
			}
```

- [ ] **Step 5: Update the existing `open` call sites in portfolio_test.go**

The six existing `p.open(100, ..., 0, 0, 0)` calls must take the new sixth arg `0`. Change each of the existing calls (lines 15, 27, 40, 52, 72, 83) from:

```go
	p.open(100, time.Unix(0, 0), 0, 0, 0)
```

to:

```go
	p.open(100, time.Unix(0, 0), 0, 0, 0, 0)
```

(The call at line 52 uses `time.Unix(3, 0)`; keep its timestamp, just append the trailing `, 0`.)

- [ ] **Step 6: Run the backtest package tests**

Run: `go test ./internal/domain/backtest/ -v`
Expected: PASS — new portfolio tests pass and all pre-existing engine/portfolio tests stay green.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/backtest/portfolio.go internal/domain/backtest/engine.go internal/domain/backtest/portfolio_test.go
git commit -m "feat(levels): freeze entry stop and track favourable max in portfolio"
```

---

## Task 3: Core management branch reads the frozen stop and monotonic arm

**Files:**
- Modify: `internal/service/trading_strategy/levels/strategy/core/core.go:152-169`
- Test: `internal/service/trading_strategy/levels/strategy/core/core_test.go`

This task lands the behaviour fix. The management branch stops recomputing `hardSL` from `in.recentLow` (the self-including-window bug) and stops deriving `armed` from the current price (the non-monotonic bug); it reads `in.pos.StopLoss` and uses `in.pos.MaxFavorablePrice` against `in.pos.EntryATR`. The entry branch is untouched.

- [ ] **Step 1: Write the two discriminating regression tests**

Add to `internal/service/trading_strategy/levels/strategy/core/core_test.go`:

```go
func TestDecideHardStopFrozenIgnoresRecentLow(t *testing.T) {
	s := newCore()
	in := bounceInput()
	// Frozen entry stop is 98.5. recentLow is far below; the old code would
	// recompute hardSL = 80 - 1 = 79 and hold. The fix must use the frozen 98.5.
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 98.5, EntryATR: 1}
	in.recentLow = 80
	in.price = 98.4
	in.barLow = 98.4 // > 79 (old: hold) but <= 98.5 (new: SL)
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("frozen stop must fire SL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 98.5 {
		t.Errorf("stop = %v, want 98.5 (frozen, not recomputed from recentLow)", sig.StopLoss)
	}
}

func TestDecideArmIsMonotonic(t *testing.T) {
	s := newCore()
	in := bounceInput()
	// Price is back below the arm threshold (entry+1*ATR = 101), but the trade
	// already reached 102, so MaxFavorablePrice latches the trail armed. The old
	// code (armed from current price) would hold; the fix must TRAIL.
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 90, EntryATR: 1, MaxFavorablePrice: 102}
	in.recentHigh = 110 // chandelier = 110 - 2.5 = 107.5
	in.price = 100.5    // below the arm threshold
	in.barLow = 107     // <= chandelier
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("monotonic arm must let the trail fire, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 107.5 {
		t.Errorf("trail stop = %v, want 107.5", sig.StopLoss)
	}
}
```

- [ ] **Step 2: Run the two new tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run 'TestDecideHardStopFrozenIgnoresRecentLow|TestDecideArmIsMonotonic' -v`
Expected: FAIL — current code recomputes `hardSL` from `recentLow` (80-1=79, so no SL) and derives `armed` from `in.price` (100.5 < 101, so holds).

- [ ] **Step 3: Rewrite the management branch in core.go**

Replace the position-management block (currently lines 152-169) with:

```go
	// Manage an open long position: hard stop then armed chandelier trail. Both
	// protective levels are anchored at entry and carried through Position — the
	// hard stop is frozen (it must not slide down with price) and the trail's arm
	// latch is monotonic via MaxFavorablePrice (it must not disarm on a pullback).
	if in.pos != nil {
		entry := in.pos.PurchasePrice
		hardSL := in.pos.StopLoss
		chandelier := in.recentHigh - s.p.TrailMult*in.atr
		// The trail arms once the trade has been in profit by TrailArmATR*EntryATR
		// at any point since entry; MaxFavorablePrice makes that latch monotonic.
		// TrailArmATR<=0 arms immediately.
		armed := s.p.TrailArmATR <= 0 || in.pos.MaxFavorablePrice >= entry+s.p.TrailArmATR*in.pos.EntryATR
		sig.StopLoss = hardSL
		switch {
		case in.barLow <= hardSL:
			sig.Kind, sig.Reason = model.SignalSell, "SL"
		case armed && in.barLow <= chandelier:
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
		}
		return sig
	}
```

- [ ] **Step 4: Run the two new tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run 'TestDecideHardStopFrozenIgnoresRecentLow|TestDecideArmIsMonotonic' -v`
Expected: PASS.

- [ ] **Step 5: Update the existing management-branch tests for the new Position fields**

The management-branch tests previously relied on `in.recentLow` for the live hard stop and on `in.price` for arming. Update each `in.pos` literal as follows.

`TestDecideHardStopExit` (set the frozen stop; drop the recentLow dependency):

```go
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 98.5, EntryATR: 1}
	in.price = 98.4
	in.barLow = 98.4 // <= frozen hard SL 98.5
```

`TestDecideTrailExit` (arm via MaxFavorablePrice; keep hard SL out of the way):

```go
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 90, EntryATR: 1, MaxFavorablePrice: 101}
	in.recentHigh = 110
	in.price = 107
	in.barLow = 107 // chandelier 107.5
```

`TestDecideTrailNotArmed` (MaxFavorablePrice below the 101 threshold -> unarmed):

```go
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 90, EntryATR: 1, MaxFavorablePrice: 100.5}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 100.5
	in.barLow = 100.5 // below chandelier but unarmed -> hold
```

`TestDecideInProfitHold` (armed, but low stays above the chandelier):

```go
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 90, EntryATR: 1, MaxFavorablePrice: 111}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 108.5
	in.barLow = 108.0 // above chandelier -> hold
```

`TestDecideHardStopOnLowWhileCloseAbove` (frozen stop, close above but low pierces):

```go
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 98.5, EntryATR: 1}
	in.price = 99.0  // close above the stop
	in.barLow = 98.4 // low pierces the frozen hard SL
```

`TestDecideTrailOnLowWhileCloseAbove` (armed, close above chandelier but low pierces):

```go
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 90, EntryATR: 1, MaxFavorablePrice: 108}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 108
	in.barLow = 107.0 // low pierces the chandelier
```

`TestDecideNoStopWhenLowAboveStops` (armed, low above both stops):

```go
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 90, EntryATR: 1, MaxFavorablePrice: 108}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 108
	in.barLow = 107.8 // above chandelier and hard SL -> hold
```

`TestDecideTrailArmStaysOnClose` — rename to `TestDecideTrailUnarmedByMaxFavorableHolds` and update body so the rename reflects the new arm source:

```go
func TestDecideTrailUnarmedByMaxFavorableHolds(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 90, EntryATR: 1, MaxFavorablePrice: 100.5}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 100.5    // never reached the arm threshold (101) -> unarmed
	in.barLow = 100.2   // below chandelier but above hard SL (90) -> arming holds
	sig := s.decide(in)
	if sig.Kind != model.SignalNone {
		t.Fatalf("unarmed trail must hold even if low below chandelier, got %v/%q", sig.Kind, sig.Reason)
	}
}
```

`TestDecideHardStopUsesSwingLow` — rename to `TestDecideHardStopUsesFrozenStop` and rewrite so it asserts the live branch uses the frozen `pos.StopLoss`:

```go
func TestDecideHardStopUsesFrozenStop(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1, StopLoss: 96, EntryATR: 1}
	in.price = 95.9
	in.barLow = 95.9 // pierces the frozen stop 96
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want Sell/SL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 96 {
		t.Errorf("stop = %v, want 96 (frozen pos.StopLoss)", sig.StopLoss)
	}
}
```

`TestDecideHardStopWiderThanEntryAnchor` — delete it. Its intent (the structural stop is wider than entry−ATR) is now an entry-branch property already covered by `TestDecideBounceBuy` (entry stop = recentLow−ATR = 98.5), and the "does not slide" guarantee is covered by `TestDecideHardStopFrozenIgnoresRecentLow`.

`TestDecideEntryAndLiveStopShareAnchor` — rewrite so it asserts the entry stop (from `recentLow`) is what the live branch echoes once frozen into the position:

```go
func TestDecideEntryAndLiveStopShareAnchor(t *testing.T) {
	s := newCore()
	in := bounceInput() // recentLow 99.5, atr 1, SLMult 1 -> entry stop 98.5
	entrySig := s.decide(in)
	if entrySig.Kind != model.SignalBuy {
		t.Fatalf("want Buy entry, got %v", entrySig.Kind)
	}
	if entrySig.StopLoss != 98.5 {
		t.Errorf("entry stop = %v, want 98.5 (recentLow 99.5 - SLMult 1 * atr 1)", entrySig.StopLoss)
	}
	// The engine freezes entrySig.StopLoss into the position; the live branch must
	// surface that same level.
	in.pos = &strategy.Position{PurchasePrice: in.price, Quantity: 1, StopLoss: entrySig.StopLoss, EntryATR: in.atr}
	liveSig := s.decide(in)
	if liveSig.StopLoss != entrySig.StopLoss {
		t.Fatalf("live stop %v != frozen entry stop %v", liveSig.StopLoss, entrySig.StopLoss)
	}
}
```

- [ ] **Step 6: Run the full core package tests**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -v`
Expected: PASS — all entry-branch tests (`TestDecideBounceBuy`, `TestRecentLow`, `TestLookbackIncludesSwingLowWindow`, integration tests) plus the updated/added management tests are green.

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/levels/strategy/core/core.go internal/service/trading_strategy/levels/strategy/core/core_test.go
git commit -m "fix(levels): read frozen entry stop and monotonic arm in management branch"
```

---

## Task 4: Engine integration test — entry stop freezes and a drifting trade exits on SL

**Files:**
- Test: `internal/domain/backtest/engine_test.go`

End-to-end proof that `sig.StopLoss` is frozen into the position, the favourable max tracks, and a position that drifts down exits via SL instead of being held to the end of data.

- [ ] **Step 1: Write the failing integration test**

Add to `internal/domain/backtest/engine_test.go`:

```go
func TestEngineFreezesEntryStopAndTracksFavorable(t *testing.T) {
	candles := flatCandles([]float64{10, 100, 105, 98})
	// Buy at price 100 (bar 1) with a frozen stop of 95. On bar 2 (price 105) the
	// favourable max should rise to 105; on bar 3 (price 98) the position must
	// still carry the frozen stop 95 and the latched max 105, then sell SL.
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy, StopLoss: 95}
		}
		if md.Position != nil && md.Price == 98 {
			if md.Position.StopLoss != 95 {
				t.Errorf("frozen StopLoss = %v, want 95", md.Position.StopLoss)
			}
			if md.Position.MaxFavorablePrice != 105 {
				t.Errorf("MaxFavorablePrice = %v, want 105 (latched high)", md.Position.MaxFavorablePrice)
			}
			return model.Signal{Kind: model.SignalSell, Reason: "SL", StopLoss: 95}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if res.Trades[0].Reason != "SL" {
		t.Fatalf("exit reason = %q, want SL", res.Trades[0].Reason)
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./internal/domain/backtest/ -run TestEngineFreezesEntryStopAndTracksFavorable -v`
Expected: PASS (the supporting engine/portfolio changes already landed in Task 2). If it fails, the failure pinpoints which carry (stop freeze or favourable mark) regressed.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/backtest/engine_test.go
git commit -m "test(levels): engine freezes entry stop and tracks favourable max"
```

---

## Task 5: Full suite green

**Files:** none (verification only).

- [ ] **Step 1: Run the whole test suite**

Run: `go test ./...`
Expected: PASS across all packages, zero failures.

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean build.

---

## Self-Review

**Spec coverage:**
- `Position` contract (spec §Components 1) → Task 1.
- Portfolio `entryStop`/`maxFavorable`/`mark`/`open` param/`close` reset/`strategyPosition` (spec §Components 2) → Task 2.
- Engine `mark` each bar + pass `sig.StopLoss` to `open` (spec §Components 3) → Task 2 Step 4.
- Core management branch frozen stop + monotonic arm (spec §Components 4) → Task 3.
- Tests: core SL fires / does-not-slide / arm-monotonic (spec §Testing core) → Task 3 Steps 1,5; portfolio open/mark/close (spec §Testing portfolio) → Task 2 Step 1; engine drift-down exits SL (spec §Testing engine) → Task 4.
- Known live gap (spec §Known gap) → documented in the `Position.StopLoss` comment (Task 1) and the spec; no code path added, by design.

**Placeholder scan:** none — every code and test block is complete.

**Type/signature consistency:** `open(price, t, level, target, atr, stop float64)` used identically in portfolio.go, engine.go, and all test call sites; `mark(price float64)`, `entryStop`, `maxFavorable` consistent; `Position.StopLoss/EntryATR/MaxFavorablePrice` field names match across Task 1, the portfolio, the engine test, and the core tests.
