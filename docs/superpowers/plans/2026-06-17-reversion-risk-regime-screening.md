# Reversion: Risk Sizing, Regime Gate, Exit Asymmetry, Screener — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add opt-in risk-based position sizing with an always-on catastrophic ATR stop, an ADX range-regime entry gate, an ATR trailing stop with a toggleable RSI50 exit, and a variance-ratio universe screener — all for the backtest research path only.

**Architecture:** Strategy params drive the new behaviour in the pure core (`internal/service/trading_strategy/reversion/strategy/core/core.go`); the catastrophic stop price is emitted on the Buy signal via the existing `model.Signal.StopLoss` field and consumed by the backtest engine for both sizing (`internal/domain/backtest/portfolio.go`) and the stop exit. The ADX indicator already exists in `pkg/indicators`. The screener is a pure variance-ratio function plus a `-screen` flag in `cmd/backtest`.

**Tech Stack:** Go 1.25, standard `testing` package (table-driven tests), existing `pkg/indicators`.

## Global Constraints

- All new behaviour is **opt-in**; defaults MUST preserve current behaviour and keep existing tests green.
- Scope is **backtest only** — do NOT wire ATR-based mechanics into live position state.
- Timeframe stays **Hour1**; do not change interval handling.
- Follow existing code style: pure decision core, warm-up discipline (a zero indicator value means "not warmed", never a real reading), table-driven tests mirroring `core_test.go`.
- Reuse `model.Signal.StopLoss` for the stop price; do NOT add a new signal field.
- New ATR-based features require `ATRPeriod > 0`; the catastrophic stop and trailing freeze `EntryATR` at entry (via `sig.ATR`).

---

## File Structure

- `internal/domain/backtest/types.go` — add `Config.RiskFractionPct`.
- `internal/domain/backtest/portfolio.go` — risk-based sizing branch in `open`.
- `internal/domain/backtest/variance_ratio.go` (new) — pure VR + autocorrelation + returns helper + verdict.
- `internal/service/trading_strategy/reversion/strategy/core/core.go` — new `Params` fields, ATR gating in `buildInput`, regime gate in `decide`, emit `StopLoss` on Buy, `CatSL`/`TRAIL` exits and `RSI50` toggle in `manage`, `Lookback` and `Explain` updates.
- `cmd/backtest/main.go` — `-risk-pct` flag wiring into `Config`; `-screen` mode + rendering.

Test files: each `*.go` above gets/extends its sibling `*_test.go`.

---

## Phase 1 — Risk-based sizing + catastrophic stop (Component A)

### Task 1: Risk-based position sizing in the portfolio

**Files:**
- Modify: `internal/domain/backtest/types.go` (Config struct)
- Modify: `internal/domain/backtest/portfolio.go:31-60` (`open`)
- Test: `internal/domain/backtest/portfolio_test.go` (create if absent)

**Interfaces:**
- Consumes: `Config{InitialCash, Fraction, Commission, Lot}`, `portfolio.open(price, t, level, target, atr, stop, entryReason)`.
- Produces: `Config.RiskFractionPct float64`; `open` sizes by risk when `RiskFractionPct>0 && stop>0 && price>stop`, else legacy `Fraction`.

- [ ] **Step 1: Write the failing test**

Create `internal/domain/backtest/portfolio_test.go`:

```go
package backtest

import (
	"testing"
	"time"
)

func TestOpenRiskBasedSizing(t *testing.T) {
	// equity 100000, risk 1% = 1000; stop 2.0 below entry 100 -> per-share risk 2;
	// target shares = 1000/2 = 500; lot 1 -> 500 shares.
	cfg := Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1, RiskFractionPct: 1.0}
	p := newPortfolio(cfg)
	p.open(100, time.Time{}, 0, 0, 2.0, 98.0, "")
	if p.qty != 500 {
		t.Fatalf("risk-sized qty = %d, want 500", p.qty)
	}
}

func TestOpenRiskBasedCappedByCash(t *testing.T) {
	// per-share risk 0.1 -> target shares 10000 -> cost 1,000,000 > cash 100000.
	// Capped to affordable floor(100000/100) = 1000 shares.
	cfg := Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1, RiskFractionPct: 1.0}
	p := newPortfolio(cfg)
	p.open(100, time.Time{}, 0, 0, 0.1, 99.9, "")
	if p.qty != 1000 {
		t.Fatalf("cash-capped qty = %d, want 1000", p.qty)
	}
}

func TestOpenLegacyFractionUnchanged(t *testing.T) {
	// RiskFractionPct 0 -> legacy Fraction path: floor(100000/100) = 1000.
	cfg := Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1, RiskFractionPct: 0}
	p := newPortfolio(cfg)
	p.open(100, time.Time{}, 0, 0, 0, 0, "")
	if p.qty != 1000 {
		t.Fatalf("legacy qty = %d, want 1000", p.qty)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestOpenRisk -v`
Expected: FAIL — `RiskFractionPct` is not a field of `Config` (compile error).

- [ ] **Step 3: Add the Config field**

In `internal/domain/backtest/types.go`, extend `Config`:

```go
// Config controls the mock portfolio and fills.
type Config struct {
	InitialCash     float64 // starting mock cash
	Fraction        float64 // fraction of current cash deployed per Buy (1.0 = all-in); used when RiskFractionPct == 0
	Commission      float64 // commission as a fraction of turnover (e.g. 0.0005)
	Lot             int32   // share lot size (orders are whole lots)
	RiskFractionPct float64 // >0 = risk this % of equity per trade, sized off the entry stop distance; 0 = legacy Fraction sizing
}
```

- [ ] **Step 4: Implement risk-based sizing in `open`**

Replace the sizing block in `internal/domain/backtest/portfolio.go` `open` (the `budget`/`lots` lines) with:

```go
	var lots int64
	if p.cfg.RiskFractionPct > 0 && stop > 0 && price > stop {
		// Risk-based sizing: position size so a stop-out loses RiskFractionPct% of equity.
		riskCapital := p.cfg.RiskFractionPct / 100 * p.equity(price)
		perShareRisk := price - stop
		lots = int64(math.Floor(riskCapital / perShareRisk / float64(p.cfg.Lot)))
		// Never deploy more than the cash on hand allows.
		if affordable := int64(math.Floor(p.cash / lotCost)); lots > affordable {
			lots = affordable
		}
	} else {
		budget := p.cfg.Fraction * p.cash
		lots = int64(math.Floor(budget / lotCost))
	}
	if lots <= 0 {
		return
	}
```

(`p.equity(price)` with `qty==0` equals `p.cash`; using it keeps the intent explicit.)

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/domain/backtest/ -run TestOpen -v`
Expected: PASS (all three).

- [ ] **Step 6: Commit**

```bash
git add internal/domain/backtest/types.go internal/domain/backtest/portfolio.go internal/domain/backtest/portfolio_test.go
git commit -m "feat(backtest): risk-based position sizing off the entry stop"
```

---

### Task 2: Emit the catastrophic stop on the Buy signal

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (Params, buildInput ATR gating, decide Buy block, Lookback)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `decideInput.atr`, `decideInput.price`, `model.Signal.StopLoss`.
- Produces: `Params.CatStopATRMult float64`; when `CatStopATRMult>0 && atr>0`, a Buy sets `sig.StopLoss = price - CatStopATRMult*atr`.

- [ ] **Step 1: Write the failing test**

Append to `core_test.go`. Uses the existing fixtures in that file: `defaultParams()` and `passingInput()` (a `decideInput` at `price=100` that fires a buy).

```go
func TestBuyEmitsCatStop(t *testing.T) {
	p := defaultParams()
	p.CatStopATRMult = 2.0
	p.ATRPeriod = 14
	s := NewWithParams("TEST", p)
	in := passingInput() // price = 100, all entry gates pass
	in.atr = 3.0
	sig := s.decide(in)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("expected Buy, got %v", sig.Kind)
	}
	want := in.price - 2.0*3.0 // 100 - 6 = 94
	if sig.StopLoss != want {
		t.Fatalf("StopLoss = %.4f, want %.4f", sig.StopLoss, want)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestBuyEmitsCatStop -v`
Expected: FAIL — `CatStopATRMult` is not a field of `Params` (compile error).

- [ ] **Step 3: Add the Param field**

In `core.go`, add to `Params` (after `StopATRMult`):

```go
	CatStopATRMult  float64 // catastrophic stop distance in EntryATR multiples; >0 = always-on backstop + risk-sizing anchor; 0 = off
```

- [ ] **Step 4: Compute ATR when the catastrophic stop is on**

In `buildInput`, change the ATR gate condition:

```go
	var atr float64
	if (s.p.UseATRStop == 1 || s.p.UseBreakeven == 1 || s.p.CatStopATRMult > 0) && s.p.ATRPeriod > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	}
```

- [ ] **Step 5: Emit the stop on Buy**

In `decide`, in the Buy block (after `sig.ATR = in.atr`):

```go
	sig.Kind = model.SignalBuy
	sig.RSI = in.rsiNow
	sig.EntryReason = s.entryReason(in)
	sig.ATR = in.atr
	if s.p.CatStopATRMult > 0 && in.atr > 0 {
		sig.StopLoss = in.price - s.p.CatStopATRMult*in.atr
	}
	return sig
```

- [ ] **Step 6: Extend Lookback**

In `Lookback`, change the ATR candidate condition:

```go
	if (s.p.UseATRStop == 1 || s.p.UseBreakeven == 1 || s.p.CatStopATRMult > 0) && s.p.ATRPeriod > 0 {
		cands = append(cands, s.p.ATRPeriod+1)
	}
```

- [ ] **Step 7: Run the test to verify it passes**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestBuyEmitsCatStop -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): emit catastrophic ATR stop on entry signal"
```

---

### Task 3: Catastrophic stop exit in manage()

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (`manage`)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `Params.CatStopATRMult`, `Position.PurchasePrice`, `Position.EntryATR`, `decideInput.price`.
- Produces: a Sell with `Reason == "SL"` and `sig.StopLoss == PurchasePrice - CatStopATRMult*EntryATR` when `price <= that stop`.

- [ ] **Step 1: Write the failing test**

Append to `core_test.go`. Uses `openInput()` (an open position with neutral signals: RSI 60→62 so no RSI50/RSIOS cross, EMAs not crossing, OB off in `defaultParams`).

```go
func TestManageCatStopExit(t *testing.T) {
	p := defaultParams()
	p.CatStopATRMult = 2.0
	s := NewWithParams("TEST", p)
	in := openInput() // in.pos != nil, neutral signals -> no other exit fires
	in.pos.PurchasePrice = 100
	in.pos.EntryATR = 3.0
	in.price = 100 - 2.0*3.0 - 0.01 // 93.99 < stop 94
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("got Kind=%v Reason=%q, want Sell/SL", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 94.0 {
		t.Fatalf("StopLoss = %.4f, want 94.0", sig.StopLoss)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestManageCatStopExit -v`
Expected: FAIL — no `SL` reason is produced (falls through to no signal or another reason).

- [ ] **Step 3: Add the CatSL case to manage()**

In `manage`, insert this case immediately AFTER the `OB` case and BEFORE the `RSI50` case:

```go
	case s.p.CatStopATRMult > 0 && in.pos.EntryATR > 0 &&
		in.price <= in.pos.PurchasePrice-s.p.CatStopATRMult*in.pos.EntryATR:
		stop := in.pos.PurchasePrice - s.p.CatStopATRMult*in.pos.EntryATR
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = stop
		sig.ExitReason = fmt.Sprintf("SL: цена %.4f ≤ катастрофический стоп %.4f (вход %.4f − %.2g×ATR %.4f)",
			in.price, stop, in.pos.PurchasePrice, s.p.CatStopATRMult, in.pos.EntryATR)
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestManageCatStopExit -v`
Expected: PASS.

- [ ] **Step 5: Run the full core suite to confirm no regressions**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -v`
Expected: PASS (all pre-existing tests still green — CatStopATRMult defaults to 0).

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): catastrophic stop exit (SL) in manage"
```

---

### Task 4: Wire the -risk-pct CLI flag

**Files:**
- Modify: `cmd/backtest/main.go` (flag declaration, `run` signature, `Config` construction)

**Interfaces:**
- Consumes: `Config.RiskFractionPct` (Task 1).
- Produces: `-risk-pct` flag flowing into every `domain.Config{...}` built in `main.go` (single-run, calibration, basket).

- [ ] **Step 1: Declare the flag**

In `main`, alongside the other `flag.Float64` declarations:

```go
		riskPct = flag.Float64("risk-pct", 0, "risk-based sizing: risk this %% of equity per trade off the entry stop (0 = fixed -fraction sizing)")
```

- [ ] **Step 2: Thread it through `run`**

Add `riskPct float64` to the `run` signature and pass `*riskPct` from `main`. In `run`, set it on the `domain.Config` literal:

```go
	cfg := domain.Config{InitialCash: cash, Fraction: fraction, Commission: commission, Lot: share.Lot, RiskFractionPct: riskPct}
```

Also set `RiskFractionPct: riskPct` on the `domain.Config` built inside `runBasket` (pass `riskPct` into `runBasket` too).

- [ ] **Step 3: Build to verify it compiles**

Run: `go build ./cmd/backtest/`
Expected: builds with no errors.

- [ ] **Step 4: Smoke-run help to confirm the flag is registered**

Run: `go run ./cmd/backtest -h 2>&1 | grep risk-pct`
Expected: the `-risk-pct` line is printed.

- [ ] **Step 5: Commit**

```bash
git add cmd/backtest/main.go
git commit -m "feat(backtest): -risk-pct flag for risk-based sizing"
```

---

## Phase 2 — ADX regime gate (Component B)

### Task 5: ADX range-regime entry gate

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (Params, decideInput, buildInput, decide, Lookback, Explain)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `indicators.ADX(highs, lows, closes, period) (adx, diPlus, diMinus float64)` (already exists; returns 0,0,0 when `len < 2*period+1`).
- Produces: `Params.UseRegime int`, `Params.ADXPeriod int`, `Params.ADXMax float64`; `decideInput.adx float64`, `decideInput.adxOK bool`. When `UseRegime==1`, entry is blocked unless `adxOK && adx < ADXMax`.

- [ ] **Step 1: Write the failing test**

Append to `core_test.go`. Uses `defaultParams()` + `passingInput()` (fires a buy when the regime gate allows).

```go
func TestRegimeGateBlocksTrend(t *testing.T) {
	p := defaultParams()
	p.UseRegime = 1
	p.ADXPeriod = 14
	p.ADXMax = 25
	s := NewWithParams("TEST", p)

	in := passingInput()
	in.adxOK = true
	in.adx = 30 // trending -> blocked
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("ADX 30 >= 25 should block the buy")
	}

	in = passingInput()
	in.adxOK = true
	in.adx = 15 // ranging -> allowed
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("ADX 15 < 25 should allow the buy")
	}
}

func TestRegimeGateBlocksWhenUnwarmed(t *testing.T) {
	p := defaultParams()
	p.UseRegime = 1
	p.ADXPeriod = 14
	p.ADXMax = 25
	s := NewWithParams("TEST", p)
	in := passingInput()
	in.adxOK = false // not warmed -> protective block
	in.adx = 0
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("un-warmed ADX must block the buy")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestRegimeGate -v`
Expected: FAIL — `UseRegime` not a field of `Params` (compile error).

- [ ] **Step 3: Add Param fields**

In `core.go` `Params` (after the HTF field):

```go
	UseRegime int     // 1 = require a low-ADX (range) regime before buying; 0 = off
	ADXPeriod int     // ADX (Wilder) length; consulted only when UseRegime==1
	ADXMax    float64 // enter only when ADX < ADXMax (range regime); consulted only when UseRegime==1
```

- [ ] **Step 4: Add fields to decideInput**

In `decideInput`:

```go
	adx   float64 // Wilder ADX (0 unless UseRegime==1 and warmed)
	adxOK bool    // true when ADX is warmed (len >= 2*ADXPeriod+1); false -> regime not confirmed
```

- [ ] **Step 5: Compute ADX in buildInput**

In `buildInput`, before the `return decideInput{...}`:

```go
	var adx float64
	adxOK := false
	if s.p.UseRegime == 1 && s.p.ADXPeriod > 0 && len(md.Closes) >= 2*s.p.ADXPeriod+1 {
		adx, _, _ = indicators.ADX(md.Highs, md.Lows, md.Closes, s.p.ADXPeriod)
		adxOK = true
	}
```

Add `adx: adx, adxOK: adxOK,` to the returned `decideInput{...}` literal.

- [ ] **Step 6: Add the gate in decide**

In `decide`, insert AFTER the trend filter (step 1) and BEFORE the dual-oversold check (step 2):

```go
	// 1b. Optional range-regime filter: trade reversion only when ADX is low (no strong
	// trend). Un-warmed ADX (adxOK=false) blocks — a protective gate must not trade when
	// it cannot confirm the regime.
	if s.p.UseRegime == 1 && !(in.adxOK && in.adx < s.p.ADXMax) {
		return sig
	}
```

- [ ] **Step 7: Extend Lookback**

In `Lookback`, add to `cands`:

```go
	if s.p.UseRegime == 1 && s.p.ADXPeriod > 0 {
		cands = append(cands, 2*s.p.ADXPeriod+1)
	}
```

- [ ] **Step 8: Add the gate to Explain**

In `explainFrom`, after the trend-filter block and before the dual-confirmation block:

```go
	// 1b. Optional range-regime (ADX) filter.
	if s.p.UseRegime == 1 {
		if !(in.adxOK && in.adx < s.p.ADXMax) {
			return block("Режим: нужен ADX<%.0f при прогретом ADX (adxOK=%v, ADX=%.2f)", s.p.ADXMax, in.adxOK, in.adx)
		}
		pass("Режим: ADX %.2f < %.0f (боковик)", in.adx, s.p.ADXMax)
	}
```

- [ ] **Step 9: Run the tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestRegimeGate -v`
Expected: PASS.

- [ ] **Step 10: Run the full core suite**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/`
Expected: PASS (UseRegime defaults to 0 — no behaviour change).

- [ ] **Step 11: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): ADX range-regime entry gate"
```

---

## Phase 3 — Trailing stop + RSI50 toggle (Component C)

### Task 6: ATR trailing exit and toggleable RSI50

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (Params, buildInput ATR gating, manage, Lookback)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `Params.UseTrail`, `Params.TrailATRMult`, `Params.UseRSI50`, `Position.MaxFavorablePrice`, `Position.EntryATR`.
- Produces: a Sell with `Reason == "TRAIL"` and `sig.StopLoss == MaxFavorablePrice - TrailATRMult*EntryATR` when price falls to that level; the `RSI50` exit only fires when `UseRSI50==1`. `UseRSI50` default 1 preserves behaviour.

- [ ] **Step 1: Write the failing test**

Append to `core_test.go`. Uses `openInput()`; `defaultParams()` will carry `UseRSI50: 1` after Step 7.

```go
func TestManageTrailExit(t *testing.T) {
	p := defaultParams()
	p.UseTrail = 1
	p.TrailATRMult = 1.5
	s := NewWithParams("TEST", p)
	in := openInput() // CatStopATRMult is 0 here, so CatSL does not fire
	in.pos.PurchasePrice = 100
	in.pos.EntryATR = 4.0
	in.pos.MaxFavorablePrice = 120  // ran up to 120
	in.price = 120 - 1.5*4.0 - 0.01 // 113.99 < trail 114 -> exit
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("got Kind=%v Reason=%q, want Sell/TRAIL", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 114.0 {
		t.Fatalf("StopLoss = %.4f, want 114.0", sig.StopLoss)
	}
}

func TestRSI50ExitToggledOff(t *testing.T) {
	p := defaultParams()
	p.UseRSI50 = 0 // disabled
	s := NewWithParams("TEST", p)
	in := openInput()
	in.rsiPrev, in.rsiNow = 55, 45 // a 50 cross-down that WOULD fire if enabled
	sig := s.decide(in)
	if sig.Kind == model.SignalSell && sig.Reason == "RSI50" {
		t.Fatalf("RSI50 exit fired while UseRSI50==0")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'TestManageTrailExit|TestRSI50ExitToggledOff' -v`
Expected: FAIL — `UseTrail` not a field of `Params` (compile error).

- [ ] **Step 3: Add Param fields**

In `core.go` `Params` (after `BreakevenArmATR`):

```go
	UseTrail     int     // 1 = ATR trailing stop on the running max favourable price; 0 = off
	TrailATRMult float64 // trail distance in EntryATR multiples below MaxFavorablePrice; consulted only when UseTrail==1
	UseRSI50     int     // 1 = exit when RSI crosses 50 downward (default); 0 = disable the RSI50 momentum-fade exit
```

- [ ] **Step 4: Compute ATR when trailing is on**

In `buildInput`, extend the ATR gate to also fire for trailing:

```go
	var atr float64
	if (s.p.UseATRStop == 1 || s.p.UseBreakeven == 1 || s.p.CatStopATRMult > 0 || s.p.UseTrail == 1) && s.p.ATRPeriod > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	}
```

Also extend the matching `Lookback` ATR-candidate condition to include `|| s.p.UseTrail == 1`.

- [ ] **Step 5: Add TRAIL exit and gate RSI50 in manage()**

Insert the `TRAIL` case AFTER the `SL` (CatSL) case and BEFORE the `RSI50` case:

```go
	case s.p.UseTrail == 1 && in.pos.EntryATR > 0 && s.p.TrailATRMult > 0 &&
		in.price <= in.pos.MaxFavorablePrice-s.p.TrailATRMult*in.pos.EntryATR:
		trail := in.pos.MaxFavorablePrice - s.p.TrailATRMult*in.pos.EntryATR
		sig.Kind, sig.Reason = model.SignalSell, "TRAIL"
		sig.StopLoss = trail
		sig.ExitReason = fmt.Sprintf("TRAIL: цена %.4f ≤ трейлинг %.4f (макс %.4f − %.2g×ATR %.4f)",
			in.price, trail, in.pos.MaxFavorablePrice, s.p.TrailATRMult, in.pos.EntryATR)
```

Change the existing `RSI50` case guard from:

```go
	case in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, rsiExitLevel):
```

to:

```go
	case s.p.UseRSI50 == 1 && in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, rsiExitLevel):
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'TestManageTrailExit|TestRSI50ExitToggledOff' -v`
Expected: PASS.

- [ ] **Step 7: Keep the test fixture's always-on RSI50**

`defaultParams()` in `core_test.go` does not set `UseRSI50`, so with the new field it defaults to 0 and the existing `TestExitRSI50` / `TestExitPrecedenceRSIWhenBoth` would break. Update the `defaultParams()` helper to preserve the always-on behaviour:

```go
func defaultParams() Params {
	return Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
		UseRSI50: 1,
	}
}
```

Then run the full core suite:

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/`
Expected: PASS (TestExitRSI50 and TestExitPrecedenceRSIWhenBoth still green; new toggle test green).

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): ATR trailing exit and toggleable RSI50"
```

---

### Task 7: Preserve RSI50 default in per-ticker and generic defaults

**Files:**
- Modify: `internal/service/backtest/reversion_registry.go:51-59` (`genericReversionDefaults`)
- Modify: each `internal/service/trading_strategy/reversion/strategy/<ticker>/<ticker>.go` `DefaultParams`
- Test: `internal/service/backtest/reversion_registry_test.go`

**Interfaces:**
- Consumes: `core.Params.UseRSI50`.
- Produces: every registered + generic default sets `UseRSI50: 1` so existing strategy behaviour (always-on RSI50) is preserved now that the field defaults to 0.

- [ ] **Step 1: Write the failing test**

Append to `internal/service/backtest/reversion_registry_test.go`:

```go
func TestGenericReversionDefaultsKeepRSI50(t *testing.T) {
	p := genericReversionDefaults()
	if p.UseRSI50 != 1 {
		t.Fatalf("generic defaults UseRSI50 = %d, want 1 (preserve always-on RSI50)", p.UseRSI50)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestGenericReversionDefaultsKeepRSI50 -v`
Expected: FAIL — `UseRSI50` is 0.

- [ ] **Step 3: Set UseRSI50 in generic defaults**

In `genericReversionDefaults`, add `UseRSI50: 1` to the returned `core.Params{...}` (e.g. on the `UseOverbought` line group).

- [ ] **Step 4: Set UseRSI50 in every per-ticker DefaultParams**

For each of `nvtk, gazp, plzl, rual, sber, ydex, afks, mdmg` under `internal/service/trading_strategy/reversion/strategy/<ticker>/<ticker>.go`, add `UseRSI50: 1,` to the `core.Params{...}` returned by `DefaultParams`. (Find them with: `grep -rl "func DefaultParams" internal/service/trading_strategy/reversion/strategy/`.)

- [ ] **Step 5: Run the registry tests + full reversion suite**

Run: `go test ./internal/service/backtest/ -run Reversion -v && go test ./internal/service/trading_strategy/reversion/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/backtest/reversion_registry.go internal/service/backtest/reversion_registry_test.go internal/service/trading_strategy/reversion/strategy/
git commit -m "feat(reversion): keep always-on RSI50 in defaults via UseRSI50=1"
```

---

## Phase 4 — Variance-ratio screener (Component D)

### Task 8: Pure variance-ratio / autocorrelation statistics

**Files:**
- Create: `internal/domain/backtest/variance_ratio.go`
- Test: `internal/domain/backtest/variance_ratio_test.go`

**Interfaces:**
- Produces:
  - `func SimpleReturns(closes []float64) []float64` — `(c[i]-c[i-1])/c[i-1]`; empty when `len<2`.
  - `func VarianceRatio(returns []float64, q int) float64` — `Var_q / (q*Var_1)` with overlapping q-sums; `0` when undefined.
  - `func Autocorr1(returns []float64) float64` — lag-1 autocorrelation; `0` when undefined.
  - `func MeanReversionVerdict(vr2 float64) string` — `"mean-reverting"` (`vr2<0.95`), `"trending"` (`vr2>1.05`), else `"neutral"`.

- [ ] **Step 1: Write the failing test**

Create `internal/domain/backtest/variance_ratio_test.go`:

```go
package backtest

import (
	"math"
	"testing"
)

func approxVR(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestSimpleReturns(t *testing.T) {
	r := SimpleReturns([]float64{100, 110, 99})
	if len(r) != 2 || !approxVR(r[0], 0.1) || !approxVR(r[1], -0.1) {
		t.Fatalf("returns = %v, want [0.1 -0.1]", r)
	}
	if len(SimpleReturns([]float64{100})) != 0 {
		t.Fatalf("single close must yield no returns")
	}
}

func TestVarianceRatioMeanReverting(t *testing.T) {
	// Alternating returns: 2-period overlapping sums are ~0 -> VR << 1.
	r := []float64{0.02, -0.02, 0.02, -0.02, 0.02, -0.02, 0.02, -0.02}
	vr := VarianceRatio(r, 2)
	if vr >= 1.0 {
		t.Fatalf("alternating series VR(2) = %.4f, want < 1", vr)
	}
}

func TestVarianceRatioZeroVarGuard(t *testing.T) {
	// Constant returns -> Var_1 == 0 -> guard returns 0, not NaN/Inf.
	r := []float64{0.01, 0.01, 0.01, 0.01, 0.01}
	if vr := VarianceRatio(r, 2); vr != 0 {
		t.Fatalf("zero-variance VR = %.4f, want 0", vr)
	}
}

func TestAutocorr1Alternating(t *testing.T) {
	r := []float64{1, -1, 1, -1, 1, -1}
	if ac := Autocorr1(r); ac >= 0 {
		t.Fatalf("alternating autocorr = %.4f, want negative", ac)
	}
}

func TestMeanReversionVerdict(t *testing.T) {
	if MeanReversionVerdict(0.8) != "mean-reverting" {
		t.Fatalf("0.8 should be mean-reverting")
	}
	if MeanReversionVerdict(1.2) != "trending" {
		t.Fatalf("1.2 should be trending")
	}
	if MeanReversionVerdict(1.0) != "neutral" {
		t.Fatalf("1.0 should be neutral")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/domain/backtest/ -run 'VarianceRatio|SimpleReturns|Autocorr1|Verdict' -v`
Expected: FAIL — undefined functions (compile error).

- [ ] **Step 3: Implement the statistics**

Create `internal/domain/backtest/variance_ratio.go`:

```go
package backtest

// SimpleReturns converts a close series (oldest-first) into 1-bar simple returns.
// Returns an empty slice when fewer than two closes are supplied.
func SimpleReturns(closes []float64) []float64 {
	if len(closes) < 2 {
		return nil
	}
	out := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		if closes[i-1] == 0 {
			out = append(out, 0)
			continue
		}
		out = append(out, (closes[i]-closes[i-1])/closes[i-1])
	}
	return out
}

// popVar returns the population variance of xs (0 for fewer than two points).
func popVar(xs []float64) float64 {
	n := len(xs)
	if n < 2 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(n)
	var ss float64
	for _, x := range xs {
		d := x - mean
		ss += d * d
	}
	return ss / float64(n)
}

// VarianceRatio is the Lo-MacKinlay variance ratio at horizon q:
// Var(q-bar overlapping returns) / (q * Var(1-bar returns)).
// VR < 1 indicates mean reversion, VR > 1 trending, VR ~ 1 a random walk.
// Returns 0 when undefined (q < 2, too few returns, or zero 1-bar variance).
func VarianceRatio(returns []float64, q int) float64 {
	if q < 2 || len(returns) < q+1 {
		return 0
	}
	var1 := popVar(returns)
	if var1 == 0 {
		return 0
	}
	qsums := make([]float64, 0, len(returns)-q+1)
	for i := 0; i+q <= len(returns); i++ {
		var s float64
		for j := i; j < i+q; j++ {
			s += returns[j]
		}
		qsums = append(qsums, s)
	}
	varq := popVar(qsums)
	return varq / (float64(q) * var1)
}

// Autocorr1 returns the lag-1 autocorrelation of returns (0 when undefined).
// A negative value indicates short-horizon mean reversion.
func Autocorr1(returns []float64) float64 {
	n := len(returns)
	if n < 3 {
		return 0
	}
	var sum float64
	for _, x := range returns {
		sum += x
	}
	mean := sum / float64(n)
	var num, den float64
	for i := 0; i < n; i++ {
		d := returns[i] - mean
		den += d * d
		if i > 0 {
			num += (returns[i-1] - mean) * d
		}
	}
	if den == 0 {
		return 0
	}
	return num / den
}

// MeanReversionVerdict classifies a 2-bar variance ratio into a label.
func MeanReversionVerdict(vr2 float64) string {
	switch {
	case vr2 > 0 && vr2 < 0.95:
		return "mean-reverting"
	case vr2 > 1.05:
		return "trending"
	default:
		return "neutral"
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/domain/backtest/ -run 'VarianceRatio|SimpleReturns|Autocorr1|Verdict' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/variance_ratio.go internal/domain/backtest/variance_ratio_test.go
git commit -m "feat(backtest): variance-ratio and autocorrelation mean-reversion stats"
```

---

### Task 9: -screen mode in cmd/backtest

**Files:**
- Create: `internal/service/backtest/screen.go` (pure rendering over the stats)
- Test: `internal/service/backtest/screen_test.go`
- Modify: `cmd/backtest/main.go` (`-screen` flag + mode dispatch)

**Interfaces:**
- Consumes: `domain.SimpleReturns`, `domain.VarianceRatio`, `domain.Autocorr1`, `domain.MeanReversionVerdict` (Task 8); the existing `NewCandleProvider`, `grpcclient`, `resolveShare` wiring in `cmd/backtest`.
- Produces: `func RenderScreenMarkdown(rows []ScreenRow) string` and a `ScreenRow{Ticker string; VR2, VR4, VR8, Autocorr1 float64; Verdict, Note string}` type; a `-screen <csv>` CLI mode.

- [ ] **Step 1: Write the failing test**

Create `internal/service/backtest/screen_test.go`:

```go
package backtest

import (
	"strings"
	"testing"
)

func TestRenderScreenMarkdown(t *testing.T) {
	rows := []ScreenRow{
		{Ticker: "NVTK", VR2: 0.80, VR4: 0.75, VR8: 0.70, Autocorr1: -0.20, Verdict: "mean-reverting"},
		{Ticker: "AFKS", Note: "нет свечей"},
	}
	out := RenderScreenMarkdown(rows)
	if !strings.Contains(out, "NVTK") || !strings.Contains(out, "mean-reverting") {
		t.Fatalf("missing NVTK row:\n%s", out)
	}
	if !strings.Contains(out, "AFKS") || !strings.Contains(out, "нет свечей") {
		t.Fatalf("skipped row should surface its note:\n%s", out)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestRenderScreenMarkdown -v`
Expected: FAIL — `ScreenRow`/`RenderScreenMarkdown` undefined (compile error).

- [ ] **Step 3: Implement the renderer**

Create `internal/service/backtest/screen.go`:

```go
package backtest

import (
	"fmt"
	"sort"
	"strings"
)

// ScreenRow is one ticker's mean-reversion screen result.
type ScreenRow struct {
	Ticker    string
	VR2       float64
	VR4       float64
	VR8       float64
	Autocorr1 float64
	Verdict   string
	Note      string // non-empty when the ticker was skipped (e.g. no candles)
}

// RenderScreenMarkdown renders the screen as a Markdown table ranked by VR(2)
// ascending (most mean-reverting first). Skipped rows (Note set) sort last.
func RenderScreenMarkdown(rows []ScreenRow) string {
	sorted := make([]ScreenRow, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		if (sorted[i].Note == "") != (sorted[j].Note == "") {
			return sorted[i].Note == "" // scored rows before skipped
		}
		return sorted[i].VR2 < sorted[j].VR2
	})

	var b strings.Builder
	b.WriteString("# Скрининг возврата к среднему (variance ratio)\n\n")
	b.WriteString("VR<1 — возврат к среднему; VR>1 — тренд. Ранжир по VR(2).\n\n")
	b.WriteString("| Тикер | VR(2) | VR(4) | VR(8) | Autocorr(1) | Вердикт |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, r := range sorted {
		if r.Note != "" {
			fmt.Fprintf(&b, "| %s | — | — | — | — | %s |\n", r.Ticker, r.Note)
			continue
		}
		fmt.Fprintf(&b, "| %s | %.3f | %.3f | %.3f | %.3f | %s |\n",
			r.Ticker, r.VR2, r.VR4, r.VR8, r.Autocorr1, r.Verdict)
	}
	return b.String()
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/service/backtest/ -run TestRenderScreenMarkdown -v`
Expected: PASS.

- [ ] **Step 5: Add the -screen flag and mode dispatch in main.go**

In `main`, declare:

```go
		screen = flag.String("screen", "", "screen mode: comma-separated tickers; ranks them by variance ratio (ignores -ticker/-strategy)")
```

Pass `*screen` into `run`. In `run`, AFTER the grpc client is built and BEFORE the `basketCSV`/`ticker` handling, add:

```go
	if screenCSV != "" {
		return runScreen(ctx, client, splitTickers(screenCSV), interval, months, outDir, refresh)
	}
```

Add a mutual-exclusion guard near the top of `run`:

```go
	if screenCSV != "" && (calibratePath != "" || basketCSV != "" || explain != "") {
		return fmt.Errorf("-screen is standalone (not combined with -calibrate/-basket/-explain)")
	}
```

Implement `runScreen` in `main.go` (mirrors `runBasket`'s candle loading):

```go
func runScreen(ctx context.Context, client grpcclient.GrpcClient, tickers []string,
	interval enum.Interval, months int, outDir string, refresh bool,
) error {
	if len(tickers) == 0 {
		return fmt.Errorf("-screen: no tickers parsed")
	}
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return fmt.Errorf("load shares: %w", err)
	}
	byTicker := make(map[string]shareInfo, len(shares))
	for _, s := range shares {
		byTicker[s.Ticker] = shareInfo{ID: s.ID, Lot: s.Lot}
	}
	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	to := time.Now()
	from := to.AddDate(0, -months, 0)

	var rows []svc.ScreenRow
	for _, ticker := range tickers {
		share, ok := byTicker[ticker]
		if !ok {
			rows = append(rows, svc.ScreenRow{Ticker: ticker, Note: "инструмент не найден"})
			continue
		}
		candles, err := provider.Load(ctx, ticker, share.ID, interval, from, to, refresh)
		if err != nil {
			return fmt.Errorf("%s: load candles: %w", ticker, err)
		}
		closes := make([]float64, len(candles))
		for i, c := range candles {
			closes[i] = c.Close
		}
		rets := domain.SimpleReturns(closes)
		if len(rets) < 9 {
			rows = append(rows, svc.ScreenRow{Ticker: ticker, Note: "мало свечей"})
			continue
		}
		vr2 := domain.VarianceRatio(rets, 2)
		rows = append(rows, svc.ScreenRow{
			Ticker: ticker, VR2: vr2,
			VR4:       domain.VarianceRatio(rets, 4),
			VR8:       domain.VarianceRatio(rets, 8),
			Autocorr1: domain.Autocorr1(rets),
			Verdict:   domain.MeanReversionVerdict(vr2),
		})
		fmt.Printf("screen %s: VR(2)=%.3f autocorr=%.3f\n", ticker, vr2, domain.Autocorr1(rets))
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	path := filepath.Join(outDir, fmt.Sprintf("screen_%s_%s.md", interval.String(), stamp))
	if err := writeFile(path, svc.RenderScreenMarkdown(rows)); err != nil {
		return err
	}
	fmt.Printf("screen report: %s (tickers=%d)\n", path, len(rows))
	return nil
}
```

Add the `screenCSV string` parameter to `run`'s signature and pass `*screen` from `main`.

- [ ] **Step 6: Build and run the package tests**

Run: `go build ./cmd/backtest/ && go test ./internal/service/backtest/ ./internal/domain/backtest/`
Expected: builds; tests PASS.

- [ ] **Step 7: Confirm the flag is registered**

Run: `go run ./cmd/backtest -h 2>&1 | grep screen`
Expected: the `-screen` line prints.

- [ ] **Step 8: Commit**

```bash
git add cmd/backtest/main.go internal/service/backtest/screen.go internal/service/backtest/screen_test.go
git commit -m "feat(backtest): -screen variance-ratio universe screener"
```

---

## Final verification

- [ ] **Run the full test suite**

Run: `go test ./...`
Expected: PASS across the repo.

- [ ] **Run `go vet`**

Run: `go vet ./internal/domain/backtest/ ./internal/service/backtest/ ./internal/service/trading_strategy/reversion/... ./cmd/backtest/`
Expected: no findings.

## Validation (user-run, not a code task)

After implementation, the user re-runs walk-forward on NVTK / GAZP / PLZL with a trimmed grid that sweeps `CatStopATRMult`, `ADXMax`, `TrailATRMult` (and fixes the rest), with `-risk-pct 1`, e.g.:

```
go run ./cmd/backtest -ticker NVTK -strategy reversion \
  -calibrate data/params/nvtk/reversion_grid_trimmed.json \
  -out ./reports/NVTK -months 48 -train-months 12 -test-months 6 \
  -metric expectancy -risk-pct 1
```

Compare per-fold OOS consistency and parameter stability against the 2026-06-17 expectancy baseline. Also run `-screen NVTK,GAZP,PLZL,RUAL,SBER,YDEX,AFKS,MDMG` and confirm the VR ranking aligns with which tickers actually trade well.
```
