# Reversion RSI-oversold breakdown exit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a third exit to the reversion strategy — close an open long when RSI crosses `RSIOversold` downward (failed-bounce / momentum stop replacement).

**Architecture:** One new `case` in the pure `manage()` core, gated on `in.rsiOK`, using the existing `RSIOversold` Param and `crossDown` helper. No new params, no engine change.

**Tech Stack:** Go 1.25, existing reversion `core` package, table-driven tests.

---

### Task 1: Add the RSIOS exit branch (TDD)

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (the `manage()` switch ~L221-234, its doc comment ~L218-220, package doc ~L4-6)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing tests**

Add two tests. Use the existing `openInput()` helper (returns a `decideInput` with a non-nil `pos` and `emaOK=true`); set fields so only the RSIOS condition is exercised. Match the style of the existing `TestExitRSI50` / `TestExitEMACross` tests already in the file (read them first to reuse the helper and assertion shape).

```go
func TestExitRSIOversoldBreakdown(t *testing.T) {
    s := NewWithParams("T", Params{FastEMA: 50, SlowEMA: 200, RSIPeriod: 14, RSIOversold: 30, StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20})
    in := openInput()
    in.rsiOK = true
    in.rsiPrev = 32 // above the oversold zone
    in.rsiNow = 28  // crossed down through 30
    // keep EMAs non-crossing and RSI50 inactive (prev already < 50)
    sig := s.decide(in)
    if sig.Kind != model.SignalSell || sig.Reason != "RSIOS" {
        t.Fatalf("expected RSIOS sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
    }
}

func TestNoRSIOSExitJustAfterEntry(t *testing.T) {
    // On/after the entry bar RSI is already below the oversold zone, so prev < level
    // and crossDown must not fire.
    s := NewWithParams("T", Params{FastEMA: 50, SlowEMA: 200, RSIPeriod: 14, RSIOversold: 30, StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20})
    in := openInput()
    in.rsiOK = true
    in.rsiPrev = 25 // already inside the zone
    in.rsiNow = 22  // still falling, but no fresh down-cross of 30
    sig := s.decide(in)
    if sig.Kind == model.SignalSell && sig.Reason == "RSIOS" {
        t.Fatalf("RSIOS must not fire when prev already below oversold (prev=%.0f now=%.0f)", in.rsiPrev, in.rsiNow)
    }
}
```

If `openInput()`'s defaults make RSI50 or EMAX fire, adjust the input fields in the tests (not the helper) so those branches stay inactive — the goal is to isolate RSIOS.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'RSIOversold|RSIOSExit' -v`
Expected: FAIL (`Reason` is empty / not `"RSIOS"`).

- [ ] **Step 3: Add the RSIOS case to `manage()`**

Insert between the RSI50 case and the EMAX case:

```go
	case in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold):
		sig.Kind, sig.Reason = model.SignalSell, "RSIOS"
		sig.ExitReason = fmt.Sprintf("RSIOS: RSI %.2f→%.2f пробил зону перепроданности %.0f сверху вниз",
			in.rsiPrev, in.rsiNow, s.p.RSIOversold)
```

- [ ] **Step 4: Update the `manage()` doc comment and package doc**

`manage()` comment: list three exits — RSI50 (primary momentum fade), RSIOS (failed-bounce breakdown of the oversold zone), EMAX (regime backstop); precedence RSI50 → RSIOS → EMAX, all fill at close.

Package doc (top of `core.go`): extend the exit sentence to mention the RSI-oversold breakdown exit alongside RSI50 and EMAX.

- [ ] **Step 5: Run the full core test package**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -v`
Expected: PASS (new tests + all existing RSI50/EMAX/precedence/entry tests).

- [ ] **Step 6: vet + full build**

Run: `go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): add RSI-oversold breakdown exit"
```

---

### Task 2: Update strategy docs

**Files:**
- Modify: `docs/reversion/strategy.md` (Exit section)

- [ ] **Step 1: Rewrite the Exit section**

List three triggers in precedence order: RSI50 (primary), RSIOS (RSI crosses `RSIOversold` from above — failed bounce / stop replacement), EMAX (regime backstop). Note all fill at close and there is still no protective price stop.

- [ ] **Step 2: Commit**

```bash
git add docs/reversion/strategy.md
git commit -m "docs(reversion): document RSI-oversold breakdown exit"
```
