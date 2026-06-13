# Reversion Overbought Take-Profit Exit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a fourth exit to the reversion strategy that closes an open long when RSI and Stochastic %D are simultaneously in their overbought zones (take-profit), without changing the existing RSI50 exit.

**Architecture:** A new highest-precedence branch in `core.manage()` gated by a new `UseOverbought` flag (default 1) and two new threshold params (`RSIOverbought`=70, `StochOverbought`=80). Pure decision logic, level-based trigger. Per-ticker defaults and the generic baseline enable it everywhere; grid calibration sweeps it automatically via JSON unmarshal.

**Tech Stack:** Go 1.25, standard `testing` (table/explicit tests in `core_test.go`).

Spec: `docs/superpowers/specs/2026-06-13-reversion-overbought-exit-design.md`

---

## File Structure

- `internal/service/trading_strategy/reversion/strategy/core/core.go` — add 3 `Params` fields, add the OB exit branch in `manage()`, update package + `manage()` doc comments.
- `internal/service/trading_strategy/reversion/strategy/core/core_test.go` — new exit tests.
- `internal/service/trading_strategy/reversion/strategy/{rusal,ydex,gazp,afks,nvtk,mdmg,sber,plzl}/*.go` — add the three fields to each `DefaultParams`.
- `internal/service/backtest/reversion_registry.go` — add the three fields to `genericReversionDefaults()`.

---

## Task 1: OB exit logic in core (Params + manage branch + docs)

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (Params struct ~32-47; `manage()` ~334-357; package doc 1-13; `manage()` doc 319-333)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `core_test.go`:

```go
func TestExitOverbought(t *testing.T) {
	p := defaultParams()
	p.UseOverbought = 1
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	// Both oscillators in their overbought zones -> sell OB.
	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75     // RSI >= 70
	in.stochPrev, in.stochNow = 82, 85 // Stoch >= 80
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "OB" {
		t.Fatalf("both overbought: want OB sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestNoOverboughtExitWhenOnlyOneZone(t *testing.T) {
	p := defaultParams()
	p.UseOverbought = 1
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	// Only RSI in zone -> hold.
	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75
	in.stochPrev, in.stochNow = 50, 55 // below 80
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("only RSI overbought: should NOT sell, got %q", sig.Reason)
	}

	// Only Stoch in zone -> hold.
	in = openInput()
	in.rsiPrev, in.rsiNow = 60, 62 // below 70
	in.stochPrev, in.stochNow = 82, 85
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("only Stoch overbought: should NOT sell, got %q", sig.Reason)
	}
}

func TestOverboughtExitOffByFlag(t *testing.T) {
	p := defaultParams() // UseOverbought defaults to 0
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75
	in.stochPrev, in.stochNow = 82, 85
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("UseOverbought=0: should NOT sell, got %q", sig.Reason)
	}
}

func TestOverboughtExitTakesPrecedence(t *testing.T) {
	p := defaultParams()
	p.UseOverbought = 1
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	// Both overbought AND a bearish EMA cross also fires; OB must win.
	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75
	in.stochPrev, in.stochNow = 82, 85
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // EMAX would fire
	if sig := s.decide(in); sig.Reason != "OB" {
		t.Fatalf("both OB and EMAX fire: want OB precedence, got %q", sig.Reason)
	}
}

func TestOverboughtExitSkippedAtWarmup(t *testing.T) {
	p := defaultParams()
	p.UseOverbought = 1
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75
	in.stochPrev, in.stochNow = 82, 85
	in.rsiOK = false // warm-up: no valid RSI reading
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "OB" {
		t.Fatalf("rsiOK=false: OB must not fire")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestOverbought -v`
Expected: FAIL — compile error `p.UseOverbought undefined` / `p.RSIOverbought undefined` / `p.StochOverbought undefined` (fields not yet added).

- [ ] **Step 3: Add the three fields to `Params`**

In `core.go`, inside the `Params` struct, add after the `VolMult` line (currently line 46):

```go
	VolMult         float64 // entry requires entryVolume >= avg*VolMult (default 1.0)
	UseOverbought   int     // 1 = exit when RSI and Stoch are simultaneously overbought; 0 = off
	RSIOverbought   float64 // RSI overbought zone for the OB exit (default 70); consulted only when UseOverbought=1
	StochOverbought float64 // Stoch %D overbought zone for the OB exit (default 80); consulted only when UseOverbought=1
```

- [ ] **Step 4: Add the OB branch as the first case in `manage()`**

In `core.go`, in `manage()`, insert as the FIRST `case` in the `switch` (immediately after `switch {`, before the RSI50 case):

```go
	case s.p.UseOverbought == 1 && in.rsiOK && in.stochOK &&
		in.rsiNow >= s.p.RSIOverbought && in.stochNow >= s.p.StochOverbought:
		sig.Kind, sig.Reason = model.SignalSell, "OB"
		sig.ExitReason = fmt.Sprintf("OB: RSI %.2f ≥ %.0f и Stoch %.2f ≥ %.0f — обе зоны перекупленности",
			in.rsiNow, s.p.RSIOverbought, in.stochNow, s.p.StochOverbought)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestOverbought -v`
Expected: PASS (all five new tests).

- [ ] **Step 6: Update the doc comments**

In `core.go`, replace the package-doc sentence (currently line 4, "It exits an open long on one of three signals:") through the end of that sentence (line 9, "There is no protective stop unless UseATRStop=1."):

```go
// It exits an open long on one of four signals: an overbought take-profit when both RSI
// and Stochastic %D are simultaneously in their overbought zones (OB, gated by UseOverbought);
// RSI crossing the 50 line downward (primary momentum fade); a middle exit selected by the
// UseATRStop flag — either RSI breaking back down through the oversold zone (RSIOS, failed
// bounce) or price falling below the ATR stop PurchasePrice − StopATRMult×EntryATR with
// EntryATR frozen at entry (ATRSL); and a bearish EMA cross (FastEMA below SlowEMA) as a
// regime-break backstop. There is no protective stop unless UseATRStop=1.
```

Then in the `manage()` doc comment (currently lines 319-321), change "It exits on one of three signals, evaluated in precedence order (all fills at close):" to "It exits on one of four signals, evaluated in precedence order (all fills at close):" and add this as the FIRST bullet (before the `RSI50:` bullet):

```go
//   - OB: RSI and Stochastic %D simultaneously in their overbought zones — take-profit
//     (gated by UseOverbought=1). Highest precedence.
```

- [ ] **Step 7: Run the full core package tests**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -v`
Expected: PASS (new tests plus all pre-existing exit/entry tests still green).

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go \
        internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): add overbought take-profit exit (OB)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Enable OB by default across tickers and generic baseline

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/rusal/rusal.go`
- Modify: `internal/service/trading_strategy/reversion/strategy/ydex/ydex.go`
- Modify: `internal/service/trading_strategy/reversion/strategy/gazp/gazp.go`
- Modify: `internal/service/trading_strategy/reversion/strategy/afks/afks.go`
- Modify: `internal/service/trading_strategy/reversion/strategy/nvtk/nvtk.go`
- Modify: `internal/service/trading_strategy/reversion/strategy/mdmg/mdmg.go`
- Modify: `internal/service/trading_strategy/reversion/strategy/sber/sber.go`
- Modify: `internal/service/trading_strategy/reversion/strategy/plzl/plzl.go`
- Modify: `internal/service/backtest/reversion_registry.go` (`genericReversionDefaults`, ~51-57)

- [ ] **Step 1: Add the OB defaults to each ticker file**

In each of the 8 ticker files, inside the `core.Params{...}` literal returned by `DefaultParams`, add a new line immediately after the existing `... VolMult: 1.0,` line:

```go
			UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
```

(The anchor line is `UseVolume: 1, VolAvgPeriod: 20, VolMult: 1.0,` in `rusal.go` and `UseVolume: 0, VolAvgPeriod: 20, VolMult: 1.0,` in the other seven; the new line is identical in all eight.)

- [ ] **Step 2: Add the OB defaults to the generic baseline**

In `internal/service/backtest/reversion_registry.go`, in `genericReversionDefaults()`, add to the returned `core.Params{...}` literal (after the `StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,` line):

```go
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
```

- [ ] **Step 3: Build and run the affected test suites**

Run: `go build ./... && go test ./internal/service/trading_strategy/reversion/... ./internal/service/backtest/...`
Expected: PASS (registry tests `TestReversionLookupGenericFallback` / `TestReversionDefaultsValid` still green — they assert FastEMA/SlowEMA and override layering, unaffected by the new fields).

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/ \
        internal/service/backtest/reversion_registry.go
git commit -m "feat(reversion): enable OB exit by default for all tickers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Final verification

- [ ] Run the whole build + test once more:

Run: `go build ./... && go test ./internal/service/trading_strategy/reversion/... ./internal/service/backtest/...`
Expected: all PASS.
