# Momentum Deferred Entry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the momentum strategy enter a long when the confirming volume arrives a few bars *after* the MACD cross, instead of discarding the setup because volume lagged on the cross bar.

**Architecture:** Split the entry into a *trigger* (MACD cross + trend filters) that "arms" a deferred signal in the impure `Strategy` shell, and a *confirmation* (volume + ATR-room + RR + a price-drift cap) checked on each subsequent bar within a validity window. Two new calibratable params (`SignalValidBars`, `MaxDriftATR`), both defaulting to `0` so the engine behaves byte-for-byte as today until they are tuned. The arming state machine lives in the impure shell next to the existing cooldown counter; the pure `decide` core reads the armed state through `decideInput`.

**Tech Stack:** Go 1.25, standard `testing`. The strategy core is `internal/service/trading_strategy/momentum/strategy/core/core.go`; its test file `core_test.go` is in `package core`, so tests can build `decideInput` values and call unexported methods directly.

---

## Background the implementer must know

The strategy is **long-only, hourly**. The decision core is pure: `Decide(md)` computes indicators in `buildInput`, then calls the pure `decide(in decideInput)`. Mutable state (the cooldown counter `barsSinceExit`) lives on the `Strategy` struct and is advanced once per bar by `trackCooldown`, called from `Decide`. `Explain(md)` is a diagnostic that re-walks the entry gates over one snapshot and never mutates state.

Today the entry requires a fresh MACD bullish cross **on the current bar**:
```go
crossUp = prevDiff <= 0 && currDiff > 0   // core.go ~line 150
```
If volume is weak on that exact bar, `decide` returns no signal, and on the next bar `crossUp` is already `false` — the setup is lost. This plan keeps the cross as the *trigger* but lets the *confirmation* arrive later.

**Key types** (already defined, do not redefine):
- `model.Signal{ Kind, Price, StopLoss, TakeProfit, ATR, EntryReason, Reason, Ticker }`, with `model.SignalBuy`, `model.SignalSell`, `model.SignalNone`.
- `strategy.MarketData`, `strategy.Position`.

The calibration grid (`data/params/<ticker>/momentum_grid.json`) maps `Params` field names to swept values by reflection (`calibrate.go`); fields absent from a grid keep their `DefaultParams` value.

---

## File Structure

- **Modify** `internal/service/trading_strategy/momentum/strategy/core/core.go` — add the two params, the arming state + state machine, wire it into `Decide`, slim the `decide` entry path to be armed-driven, drift cap, deferred annotation in `entryReason`, and deferral-aware `Explain`.
- **Modify** `internal/service/trading_strategy/momentum/strategy/core/core_test.go` — add the test helper `deferredBars` plus all new tests.
- **Modify** `docs/momentum/strategy.md` — document the deferred entry (§2), the two new params (§5), and clear the placeholder in §8.
- **Modify** `data/params/{rusal,afks,ydex,plzl}/momentum_grid.json` — add `SignalValidBars` and `MaxDriftATR` sweep arrays.

No new files. No changes to exits (`manage`), `Lookback`, the ticker packages, or the registry.

---

## Task 1: Arming state machine + armed-driven entry

Adds the params, the deferred-signal state, the per-bar state machine (`advanceArm`/`qualifiesAsTrigger`/`clearArm`), wires it into `Decide`, and rewrites the `decide` entry path to require an armed signal. Backward compatibility (`SignalValidBars == 0` ⇒ identical behavior) is enforced by the existing test suite plus a new explicit test.

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Add the deferred-entry test helper**

Append to `core_test.go`:

```go
// deferredBars returns numBars consecutive hourly snapshots simulating the bars
// after a fresh MACD bullish cross. Snapshot 0 is the cross bar. Each later
// snapshot appends one "holding" bar that nudges price up slightly (no new cross,
// MACD stays bullish). Confirming volume (5000) is placed only on the snapshot at
// index volumeAtBar; every other last-bar volume is weak (100). Feed the snapshots
// to one Strategy in order to drive the arming state machine.
func deferredBars(numBars, volumeAtBar int) []strategy.MarketData {
	base := buildEntryMD()
	snaps := make([]strategy.MarketData, numBars)
	for k := 0; k < numBars; k++ {
		closes := append([]float64(nil), base.Closes...)
		highs := append([]float64(nil), base.Highs...)
		lows := append([]float64(nil), base.Lows...)
		vols := append([]int64(nil), base.Volumes...)
		last := closes[len(closes)-1]
		for j := 0; j < k; j++ { // append k holding bars
			last += 0.2
			closes = append(closes, last)
			highs = append(highs, last+0.5)
			lows = append(lows, last-0.2)
			vols = append(vols, 100)
		}
		if k == volumeAtBar {
			vols[len(vols)-1] = 5000
		} else {
			vols[len(vols)-1] = 100
		}
		md := base
		md.Closes, md.Highs, md.Lows, md.Volumes = closes, highs, lows, vols
		md.Price = closes[len(closes)-1]
		md.TodayHigh = md.Price + 0.5
		md.TodayLow = md.Price - 0.5
		snaps[k] = md
	}
	return snaps
}
```

- [ ] **Step 2: Write the failing deferred-entry tests**

Append to `core_test.go`:

```go
func TestDeferredEntryFiresWhenVolumeArrivesWithinWindow(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 5 // generous; drift is tested separately
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1) // cross bar weak volume; volume on bar 1
	if sig := s.Decide(bars[0]); sig.Kind == model.SignalBuy {
		t.Fatal("should NOT enter on the cross bar (volume weak)")
	}
	if sig := s.Decide(bars[1]); sig.Kind != model.SignalBuy {
		t.Fatal("should enter on the next bar when volume confirms within the window")
	}
}

func TestDeferredDisabledIgnoresLateVolume(t *testing.T) {
	p := defaultParams() // SignalValidBars == 0 -> deferral off
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1)
	if sig := s.Decide(bars[0]); sig.Kind == model.SignalBuy {
		t.Fatal("no entry on cross bar (weak volume)")
	}
	if sig := s.Decide(bars[1]); sig.Kind == model.SignalBuy {
		t.Fatal("with deferral off, late volume must NOT produce an entry")
	}
}

func TestDeferredEntryExpiresAfterWindow(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 1
	p.MaxDriftATR = 5
	s := NewWithParams("TEST", p)
	bars := deferredBars(3, 2) // cross(0) weak, hold(1) weak, volume(2) too late
	s.Decide(bars[0]) // arm
	s.Decide(bars[1]) // age to 1 (still within window=1), no volume
	if sig := s.Decide(bars[2]); sig.Kind == model.SignalBuy {
		t.Fatal("signal must expire after SignalValidBars and block the late entry")
	}
}
```

- [ ] **Step 3: Run the new tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'Deferred' -v`
Expected: compile error or FAIL — `Params` has no field `SignalValidBars`/`MaxDriftATR`, and current logic never enters on bar 1.

- [ ] **Step 4: Add the two params**

In `core.go`, extend the `Params` struct (append after `DailyTrendPeriod`):

```go
	DailyTrendPeriod  int     // daily-EMA period for the higher-timeframe slope filter (0 disables)
	SignalValidBars   int     // bars a MACD-cross trigger stays "armed" awaiting confirmation; 0 = enter only on the cross bar (no deferral)
	MaxDriftATR       float64 // max |price - crossPrice| in hourly-ATR units to still take a deferred entry; <=0 disables the drift cap
```

- [ ] **Step 5: Add the armed-signal state to the Strategy struct**

In `core.go`, extend `Strategy`:

```go
type Strategy struct {
	ticker          string
	p               Params
	barsSinceExit   int     // bars elapsed since the last exit; gates re-entry
	prevInPosition  bool    // whether the previous Decide saw an open position
	armedCrossPrice float64 // close at the bar that armed a deferred signal; 0 = no armed signal
	barsSinceArm    int     // bars elapsed since the signal was armed
}
```

- [ ] **Step 6: Add `macdAboveSignal` to decideInput and compute it**

In `core.go`, add the field to `decideInput` (after `crossUp`):

```go
	crossUp         bool
	macdAboveSignal bool    // MACD line currently above the signal line (momentum still bullish)
```

Add the deferred-state fields to `decideInput` (after `barsSinceExit`):

```go
	barsSinceExit   int     // bars since the last exit, for the cooldown gate
	armedCrossPrice float64 // armed deferred-signal price (0 = none); copied from the shell
	barsSinceArm    int     // age of the armed deferred signal; copied from the shell
```

In `buildInput`, set `macdAboveSignal`. Replace the MACD block:

```go
	macdNow, crossUp := 0.0, false
	if m, sg := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal); len(m) >= 2 {
		prevDiff := m[len(m)-2] - sg[len(sg)-2]
		currDiff := m[len(m)-1] - sg[len(sg)-1]
		macdNow = m[len(m)-1]
		crossUp = prevDiff <= 0 && currDiff > 0
	}
```

with:

```go
	macdNow, crossUp, macdAboveSignal := 0.0, false, false
	if m, sg := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal); len(m) >= 2 {
		prevDiff := m[len(m)-2] - sg[len(sg)-2]
		currDiff := m[len(m)-1] - sg[len(sg)-1]
		macdNow = m[len(m)-1]
		crossUp = prevDiff <= 0 && currDiff > 0
		macdAboveSignal = currDiff > 0
	}
```

and add `macdAboveSignal: macdAboveSignal,` to the returned `decideInput{...}` literal (next to `crossUp:`).

- [ ] **Step 7: Add the arming state machine**

In `core.go`, add these methods (place them right after `trackCooldown`):

```go
// advanceArm runs the deferred-entry state machine in the impure shell: it ages an
// armed signal, cancels it when the validity window lapses or the trigger thesis
// breaks (trend down, daily trend stalls, momentum dies), then (re)arms on a fresh
// qualifying MACD cross at this bar's price. Called once per bar from Decide,
// before decide reads the armed state. Explain never calls this (no mutation).
func (s *Strategy) advanceArm(in decideInput) {
	// In a position, or cooling down after an exit: no signal may live. Cooldown
	// has priority so we never enter on a stale cross right as cooldown lapses.
	if in.pos != nil || (s.p.CooldownBars > 0 && in.barsSinceExit < s.p.CooldownBars) {
		s.clearArm()
		return
	}
	if s.armedCrossPrice > 0 {
		s.barsSinceArm++
		trendBroke := !(in.emaTrend > 0 && in.price > in.emaTrend)
		dailyBroke := s.p.DailyTrendPeriod > 0 && in.dailyTrendKnown && !(in.dailyEMANow > in.dailyEMAPast)
		if s.barsSinceArm > s.p.SignalValidBars || trendBroke || dailyBroke || !in.macdAboveSignal {
			s.clearArm()
		}
	}
	if s.qualifiesAsTrigger(in) {
		s.armedCrossPrice = in.price
		s.barsSinceArm = 0
	}
}

// qualifiesAsTrigger reports whether this bar is a fresh MACD cross that meets the
// trigger filters (uptrend, rising daily trend if enabled, below-zero if required).
// These are the conditions that arm a deferred signal; confirmation gates (volume,
// ATR-room, RR, drift) are checked later in decide.
func (s *Strategy) qualifiesAsTrigger(in decideInput) bool {
	if !in.crossUp {
		return false
	}
	if !(in.emaTrend > 0 && in.price > in.emaTrend) {
		return false
	}
	if s.p.DailyTrendPeriod > 0 && in.dailyTrendKnown && !(in.dailyEMANow > in.dailyEMAPast) {
		return false
	}
	if s.p.MACDBelowZeroOnly == 1 && in.macdNow >= 0 {
		return false
	}
	return true
}

// clearArm drops any armed deferred signal.
func (s *Strategy) clearArm() {
	s.armedCrossPrice = 0
	s.barsSinceArm = 0
}
```

- [ ] **Step 8: Wire arming into Decide**

In `core.go`, replace `Decide`:

```go
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	s.trackCooldown(md.Position)
	sig := s.decide(s.buildInput(md))
	sig.Ticker = s.ticker
	return sig
}
```

with:

```go
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	s.trackCooldown(md.Position)
	in := s.buildInput(md)
	s.advanceArm(in)
	in.armedCrossPrice = s.armedCrossPrice
	in.barsSinceArm = s.barsSinceArm
	sig := s.decide(in)
	if sig.Kind == model.SignalBuy {
		s.clearArm() // consumed
	}
	sig.Ticker = s.ticker
	return sig
}
```

- [ ] **Step 9: Rewrite the decide entry path to be armed-driven**

In `core.go`, add `"math"` to the import block. Then replace the entry gates in `decide` — everything from the cooldown check through the `MinRR` rejection (the block that currently starts `if s.p.CooldownBars > 0 ...` and ends just before `sig.Kind = model.SignalBuy`) — with:

```go
	if s.p.CooldownBars > 0 && in.barsSinceExit < s.p.CooldownBars {
		return sig // still cooling down after the last exit
	}

	// Require a live armed signal. advanceArm arms it from a qualifying cross and
	// cancels it on window expiry / broken trend / dead momentum. With
	// SignalValidBars==0 a signal is armed only on the bar of the cross itself, so
	// this reduces to the original "the cross must be on this bar" behavior.
	if in.armedCrossPrice <= 0 {
		return sig
	}

	// Drift cap: don't chase — price must be within MaxDriftATR*ATR of the cross.
	if s.p.MaxDriftATR > 0 && math.Abs(in.price-in.armedCrossPrice) > s.p.MaxDriftATR*in.atr {
		return sig
	}

	// Confirmation gates.
	if !in.volumeOK {
		return sig
	}
	// Daily-ATR room: pass when daily data is absent (dailyATR<=0), else require room.
	if in.dailyATR > 0 && in.todayRange >= s.p.MaxDailyATRUsed*in.dailyATR {
		return sig
	}
	if s.p.MinATRFrac > 0 && in.atr < s.p.MinATRFrac*in.price {
		return sig
	}

	stop := in.recentLow - s.p.SLMult*in.atr
	risk := in.price - stop
	if risk <= 0 {
		return sig
	}
	target := in.price + s.p.TakeProfitRR*risk
	if s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return sig
	}
```

Leave the `sig.Kind = model.SignalBuy` ... `return sig` tail of `decide` unchanged.

- [ ] **Step 10: Run the new tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'Deferred' -v`
Expected: PASS (all three deferred tests).

- [ ] **Step 11: Run the whole core suite to confirm backward compatibility**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -v`
Expected: PASS — every pre-existing test still green (they all run with `SignalValidBars == 0`).

- [ ] **Step 12: Build and vet**

Run: `go build ./... && go vet ./...`
Expected: no output (clean).

- [ ] **Step 13: Commit**

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "$(cat <<'EOF'
feat(momentum): arm deferred MACD-cross signals for delayed confirmation

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Drift cap

The drift gate is already implemented in Task 1 (Step 9). This task adds the tests that pin its behavior in both directions.

**Files:**
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing drift tests**

Append to `core_test.go`:

```go
func TestDeferredEntryBlockedByPriceDrift(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 0.01 // ~zero tolerance: any drift from the cross price blocks
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1) // holding bar nudges price +0.2 from the cross
	s.Decide(bars[0])          // arm at the cross price
	if sig := s.Decide(bars[1]); sig.Kind == model.SignalBuy {
		t.Fatal("entry must be blocked when price drifted beyond MaxDriftATR*ATR")
	}
}

func TestDeferredEntryAllowedWithinDrift(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 5 // generous tolerance
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1)
	s.Decide(bars[0])
	if sig := s.Decide(bars[1]); sig.Kind != model.SignalBuy {
		t.Fatal("entry should be allowed when price stayed within the drift cap")
	}
}
```

- [ ] **Step 2: Run the drift tests**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'Drift' -v`
Expected: PASS — the drift gate from Task 1 already enforces this.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "$(cat <<'EOF'
test(momentum): pin deferred-entry price-drift cap behavior

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Cancellation guards (window, momentum-death, trend-break, cooldown)

Drive the `advanceArm` state machine directly with hand-built `decideInput` values (tests are in `package core`) to deterministically pin every cancellation branch. The logic already exists from Task 1; these tests lock it in.

**Files:**
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing state-machine tests**

Append to `core_test.go`:

```go
// armInput is a minimal decideInput that satisfies qualifiesAsTrigger:
// fresh cross, clear uptrend, MACD above signal, daily filter not engaged.
func armInput() decideInput {
	return decideInput{price: 100, emaTrend: 90, atr: 1, crossUp: true, macdAboveSignal: true}
}

func TestAdvanceArmArmsOnQualifyingCross(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 3
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput())
	if s.armedCrossPrice != 100 || s.barsSinceArm != 0 {
		t.Fatalf("want armed at 100 age 0, got price=%f age=%d", s.armedCrossPrice, s.barsSinceArm)
	}
}

func TestAdvanceArmExpiresAfterWindow(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 1
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput()) // age 0
	hold := decideInput{price: 100.1, emaTrend: 90, atr: 1, macdAboveSignal: true}
	s.advanceArm(hold) // age 1, still within window
	if s.armedCrossPrice == 0 {
		t.Fatal("should still be armed at age 1 with window 1")
	}
	s.advanceArm(hold) // age 2 > window 1 -> cancel
	if s.armedCrossPrice != 0 {
		t.Fatal("should cancel once age exceeds SignalValidBars")
	}
}

func TestAdvanceArmCancelsOnMomentumDeath(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput())
	dead := decideInput{price: 100.1, emaTrend: 90, atr: 1, macdAboveSignal: false}
	s.advanceArm(dead)
	if s.armedCrossPrice != 0 {
		t.Fatal("should cancel when MACD line falls back below the signal line")
	}
}

func TestAdvanceArmCancelsOnTrendBreak(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput())
	below := decideInput{price: 80, emaTrend: 90, atr: 1, macdAboveSignal: true} // price < EMA
	s.advanceArm(below)
	if s.armedCrossPrice != 0 {
		t.Fatal("should cancel when price falls back below the trend EMA")
	}
}

func TestAdvanceArmCancelsOnDailyTrendStall(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	p.DailyTrendPeriod = 5
	s := NewWithParams("TEST", p)
	armed := armInput()
	armed.dailyTrendKnown = true
	armed.dailyEMANow, armed.dailyEMAPast = 110, 100 // rising at arm time
	s.advanceArm(armed)
	stall := decideInput{price: 100.1, emaTrend: 90, atr: 1, macdAboveSignal: true,
		dailyTrendKnown: true, dailyEMANow: 100, dailyEMAPast: 110} // now falling
	s.advanceArm(stall)
	if s.armedCrossPrice != 0 {
		t.Fatal("should cancel when the daily trend stops rising")
	}
}

func TestAdvanceArmDoesNotArmDuringCooldown(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	p.CooldownBars = 3
	s := NewWithParams("TEST", p)
	in := armInput()
	in.barsSinceExit = 1 // inside cooldown (1 < 3)
	s.advanceArm(in)
	if s.armedCrossPrice != 0 {
		t.Fatal("should not arm a signal while cooling down")
	}
}

func TestAdvanceArmClearsWhenInPosition(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput()) // arm
	in := decideInput{price: 100, emaTrend: 90, atr: 1, pos: &strategy.Position{PurchasePrice: 100}}
	s.advanceArm(in)
	if s.armedCrossPrice != 0 {
		t.Fatal("should clear the armed signal while a position is open")
	}
}
```

- [ ] **Step 2: Run the state-machine tests**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'AdvanceArm' -v`
Expected: PASS — all branches already implemented in Task 1.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "$(cat <<'EOF'
test(momentum): pin deferred-signal cancellation guards

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Diagnostics — deferred entry reason + deferral-aware Explain

Annotate the trade-journal reason when an entry was deferred, and make `Explain` report the armed/waiting state and the drift gate instead of falsely claiming "no cross" on a waiting bar.

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing diagnostics tests**

Append to `core_test.go`:

```go
func TestEntryReasonNotesDeferral(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 5
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1)
	s.Decide(bars[0]) // arm
	sig := s.Decide(bars[1])
	if sig.Kind != model.SignalBuy {
		t.Fatalf("expected deferred entry, got kind=%v", sig.Kind)
	}
	if !strings.Contains(sig.EntryReason, "отложенный вход") {
		t.Fatalf("EntryReason should note the deferral, got: %q", sig.EntryReason)
	}
}

func TestImmediateEntryReasonHasNoDeferralNote(t *testing.T) {
	p := defaultParams() // SignalValidBars == 0 -> immediate entry only
	s := NewWithParams("TEST", p)
	sig := s.Decide(buildEntryMD())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("expected immediate entry, got kind=%v", sig.Kind)
	}
	if strings.Contains(sig.EntryReason, "отложенный вход") {
		t.Fatalf("immediate entry must not carry a deferral note, got: %q", sig.EntryReason)
	}
}

func TestExplainReportsArmedWaitingSignal(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 5
	s := NewWithParams("TEST", p)
	bars := deferredBars(3, 2)
	s.Decide(bars[0]) // arm; bar[1] has no fresh cross and weak volume
	out := s.Explain(bars[1])
	if !strings.Contains(out, "взвед") {
		t.Fatalf("Explain should report the armed waiting signal, got: %q", out)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'Deferral|ArmedWaiting' -v`
Expected: FAIL — no deferral note in `entryReason`; `Explain` blocks at the cross gate on a waiting bar.

- [ ] **Step 3: Annotate entryReason on deferred entries**

In `core.go`, replace the `entryReason` method body's final `return fmt.Sprintf(...)` with a captured string plus a deferral suffix. The method currently ends:

```go
	return fmt.Sprintf(
		"Тренд↑ (close %.4f > EMA%d %.4f); MACD бычий кросс %s (%.4f); объём > %.2g×ср(%d); дневной ATR-запас %.0f%% (прошло %.4f из %.4f); ATR(ч)=%.4f, ATR(д)=%.4f; SL=%.4f (-%.4f); TP=%.4f (+%.4f, %.2gR)",
		in.price, s.p.EMAPeriod, in.emaTrend, zero, in.macdNow,
		s.p.VolMultiplier, s.p.VolLookback,
		roomPct, in.todayRange, in.dailyATR,
		in.atr, in.dailyATR,
		stop, risk, target, target-in.price, s.p.TakeProfitRR,
	)
```

Replace with:

```go
	reason := fmt.Sprintf(
		"Тренд↑ (close %.4f > EMA%d %.4f); MACD бычий кросс %s (%.4f); объём > %.2g×ср(%d); дневной ATR-запас %.0f%% (прошло %.4f из %.4f); ATR(ч)=%.4f, ATR(д)=%.4f; SL=%.4f (-%.4f); TP=%.4f (+%.4f, %.2gR)",
		in.price, s.p.EMAPeriod, in.emaTrend, zero, in.macdNow,
		s.p.VolMultiplier, s.p.VolLookback,
		roomPct, in.todayRange, in.dailyATR,
		in.atr, in.dailyATR,
		stop, risk, target, target-in.price, s.p.TakeProfitRR,
	)
	if in.barsSinceArm > 0 {
		reason += fmt.Sprintf("; отложенный вход: ждали %d бар(ов), снос от цены кросса %+.4f",
			in.barsSinceArm, in.price-in.armedCrossPrice)
	}
	return reason
```

- [ ] **Step 4: Make Explain deferral-aware**

In `core.go`, in `Explain`, replace the MACD cross + below-zero block (the section commented `// 2. MACD bullish cross.` through the `// 3. Below-zero requirement.` block, ending before `// 4. Volume.`) with:

```go
	// 2. MACD trigger: a fresh bullish cross, or a still-armed deferred signal.
	armed := s.p.SignalValidBars > 0 && s.armedCrossPrice > 0
	switch {
	case in.crossUp:
		pass("MACD: бычий кросс (MACD=%.4f)", in.macdNow)
	case armed:
		pass("MACD: сигнал взведён %d бар(ов) назад по цене %.4f (окно %d), MACD над сигнальной: %v",
			s.barsSinceArm, s.armedCrossPrice, s.p.SignalValidBars, in.macdAboveSignal)
	default:
		return block("MACD: нет бычьего кросса и нет взведённого сигнала (MACD=%.4f)", in.macdNow)
	}

	// 3. Below-zero requirement (a trigger condition — checked only on a fresh cross).
	if s.p.MACDBelowZeroOnly == 1 && in.crossUp {
		if in.macdNow >= 0 {
			return block("MACD: кросс над нулём (%.4f), а требуется под нулём (MACDBelowZeroOnly=1)", in.macdNow)
		}
		pass("MACD: кросс под нулём (%.4f)", in.macdNow)
	}

	// 3b. Price-drift cap (only while a deferred signal is armed).
	if armed && s.p.MaxDriftATR > 0 {
		drift := math.Abs(in.price - s.armedCrossPrice)
		if drift > s.p.MaxDriftATR*in.atr {
			return block("Снос цены: |%.4f − %.4f| = %.4f > %.2g×ATR %.4f",
				in.price, s.armedCrossPrice, drift, s.p.MaxDriftATR, in.atr)
		}
		pass("Снос цены: %.4f ≤ %.2g×ATR %.4f", drift, s.p.MaxDriftATR, in.atr)
	}
```

(`math` is already imported from Task 1.)

- [ ] **Step 5: Run the diagnostics tests**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'Deferral|ArmedWaiting' -v`
Expected: PASS.

- [ ] **Step 6: Run the full core suite**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -v`
Expected: PASS — including the existing `TestExplainReports*` tests (the cross/below-zero rewrite preserves their behavior with `SignalValidBars == 0`).

- [ ] **Step 7: Build, vet, commit**

```bash
go build ./... && go vet ./...
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "$(cat <<'EOF'
feat(momentum): surface deferred entry in trade reason and Explain

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Documentation + calibration grids

Document the deferred entry for humans and add the two new knobs to all four calibration grids.

**Files:**
- Modify: `docs/momentum/strategy.md`
- Modify: `data/params/rusal/momentum_grid.json`
- Modify: `data/params/afks/momentum_grid.json`
- Modify: `data/params/ydex/momentum_grid.json`
- Modify: `data/params/plzl/momentum_grid.json`

- [ ] **Step 1: Document the deferred entry in §2**

In `docs/momentum/strategy.md`, after the entry-conditions table in §2 (after the line that begins "Условия легко расширяются..."), insert:

```markdown
### Отложенный вход (deferred entry)

Бывает, что MACD дал бычий кросс при выполненном тренде, но на этом баре не хватило
**объёма** (вечер, неликвид) — а утром объём приходит. Чтобы не терять такой сетап,
стратегия разделяет **триггер** и **подтверждение**:

- **Триггер** (кросс MACD вверх + цена > EMA + дневной тренд/below-zero, если включены)
  «взводит» сигнал и запоминает цену кросса.
- **Подтверждение** ищется на следующих барах: первый бар в окне `SignalValidBars`,
  где сходятся объём, запас дневного ATR и RR — это и есть вход.

Пока сигнал «взведён», он гасится, если: истекло окно `SignalValidBars`, цена ушла
от цены кросса дальше `MaxDriftATR × ATR(час)` (симметрично — и вверх, и вниз),
сломался тренд (`close < EMA`), перестал расти дневной тренд, либо MACD-линия ушла
обратно под сигнальную (импульс умер). SL/TP считаются по бару фактического входа.
При `SignalValidBars = 0` механизм выключен — вход возможен только на самом баре кросса
(поведение по умолчанию). Во время кулдауна сигнал не взводится.
```

- [ ] **Step 2: Add the two params to the §5 table**

In `docs/momentum/strategy.md` §5, add two rows after the `CooldownBars` row:

```markdown
| `SignalValidBars` | 0 | Сколько баров после кросса сигнал остаётся «взведён» в ожидании объёма (0 = вход только на баре кросса, отложенный вход выключен). |
| `MaxDriftATR` | 0 | Макс. снос цены от цены кросса в долях часового ATR для отложенного входа (≤0 = лимит выключен). |
```

- [ ] **Step 3: Clear the §8 placeholder**

In `docs/momentum/strategy.md` §8, replace:

```markdown
## 8. Заготовки на будущее (выключены по умолчанию)

- (пока нет активных заготовок)
```

with:

```markdown
## 8. Заготовки на будущее (выключены по умолчанию)

- **Отложенный вход** (`SignalValidBars`, `MaxDriftATR`) — реализован, выключен по
  умолчанию (см. §2). Включается калибровкой.
```

- [ ] **Step 4: Extend all four calibration grids**

In each of `data/params/{rusal,afks,ydex,plzl}/momentum_grid.json`, add two keys before the closing brace (the file currently ends after the `"DailyTrendPeriod"` line — add a comma there and append the two keys). The resulting file must read:

```json
{
  "MACDSlow": [21, 26],
  "MACDSignal": [9],
  "MACDBelowZeroOnly": [0, 1],
  "SLMult": [0.5, 1.0],
  "TakeProfitRR": [1.5, 2.0, 3.0],
  "VolMultiplier": [1.0, 1.2, 1.5],
  "MaxDailyATRUsed": [0.5, 0.6, 0.7],
  "EMAPeriod": [200, 150, 100],
  "MACDFast": [12, 10, 8],
  "UseTrail": [0, 1],
  "DailyTrendPeriod": [0, 10, 20],
  "SignalValidBars": [0, 2, 4],
  "MaxDriftATR": [0, 1.0]
}
```

Note for the human calibrator: this multiplies the grid by 6 (`SignalValidBars` ×3 × `MaxDriftATR` ×2). Trim the arrays if calibration runtime is too long; `[0]` in either key reproduces the current grid for that knob.

- [ ] **Step 5: Verify the grids are valid JSON**

Run: `for f in data/params/{rusal,afks,ydex,plzl}/momentum_grid.json; do python3 -m json.tool "$f" >/dev/null && echo "OK $f"; done`
Expected: four `OK` lines, no errors.

- [ ] **Step 6: Full test + build + vet**

Run: `go test ./... && go build ./... && go vet ./...`
Expected: all green, no FAIL.

- [ ] **Step 7: Commit**

```bash
git add docs/momentum/strategy.md data/params/rusal/momentum_grid.json data/params/afks/momentum_grid.json data/params/ydex/momentum_grid.json data/params/plzl/momentum_grid.json
git commit -m "$(cat <<'EOF'
docs(momentum): document deferred entry and add it to calibration grids

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- Trigger vs confirmation split → Task 1 (`advanceArm`/`qualifiesAsTrigger` + armed-driven `decide`). ✓
- `SignalValidBars` / `MaxDriftATR`, both default 0/off → Task 1 Step 4; backward-compat tests in Task 1 Steps 2/11. ✓
- Symmetric drift cap in ATR units → Task 1 Step 9; Task 2 tests. ✓
- MACD must stay bullish; trend/daily-trend still hold; window expiry → Task 1 `advanceArm`; Task 3 tests. ✓
- SL/TP from the actual entry bar → unchanged `decide` tail (uses current `in.atr`/`in.recentLow`). ✓
- Cooldown has priority (no arming during cooldown) → Task 1 `advanceArm`; Task 3 test. ✓
- `entryReason` deferral note + deferral-aware `Explain` → Task 4. ✓
- Docs (§2, §5, §8) + grids → Task 5. ✓
- Backward compatibility (`SignalValidBars==0` identical) → enforced by the whole existing suite (Task 1 Step 11) plus `TestDeferredDisabledIgnoresLateVolume`. ✓

**Placeholder scan:** No TBD/TODO/"similar to"; every code step shows full code and exact commands. ✓

**Type consistency:** New `Params` fields `SignalValidBars int`, `MaxDriftATR float64`; `decideInput` fields `macdAboveSignal bool`, `armedCrossPrice float64`, `barsSinceArm int`; `Strategy` fields `armedCrossPrice`, `barsSinceArm`; methods `advanceArm`, `qualifiesAsTrigger`, `clearArm` — names used consistently across Tasks 1–4. Tests rely on `package core` access to unexported `advanceArm`/`decideInput`/`armedCrossPrice`, which holds (the test file is `package core`). ✓

**Out of scope (unchanged):** exits/`manage`, `Lookback`, ticker packages, registry, running `-calibrate`, hardcoding winners.
