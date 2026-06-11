# Momentum-Based Exits Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the momentum strategy's either/or exit mode with five independently-toggleable exit triggers (hard stop, trailing stop, fixed TP, MACD bearish cross, RSI overbought cross-down), closing on the first to fire.

**Architecture:** All exit logic lives in the pure `decide`/`manage` core of `internal/service/trading_strategy/momentum/strategy/core/core.go`. New exits are gated by params (`<=0`/`0` disables, idiomatic to the file). The two new exits (MACD, RSI) default OFF, so behavior is unchanged until calibration turns them on. The backtest engine is untouched: it already fills any reason it does not special-case (`SL`/`TP`) at the bar close, which is exactly what the close-confirmed MACD/RSI exits need.

**Tech Stack:** Go 1.25, existing `pkg/indicators` (`MACD`, `RSISeries`, `ATR`), `internal/domain/ema`. Tests are table-style in `core_test.go` (package-internal, `package core`).

**Spec:** `docs/superpowers/specs/2026-06-11-momentum-momentum-based-exits-design.md`

---

## File Structure

- **Modify** `internal/service/trading_strategy/momentum/strategy/core/core.go` — `Params` (3 new fields, 2 comment changes), `Lookback`, `buildInput`, `decideInput`, `decide` (MinRR decoupling), `manage` (independent toggles + MACD/RSI cases), `Explain` (MinRR decoupling).
- **Modify** `internal/service/trading_strategy/momentum/strategy/core/core_test.go` — new helper `inPositionMDWithCloses` + new exit tests.
- **Modify** 8 per-ticker `DefaultParams()` files: `afks/afks.go`, `mdmg/mdmg.go`, `sber/sber.go`, `nvtk/nvtk.go`, `ydex/ydex.go`, `plzl/plzl.go`, `gazp/gazp.go`, `rusal/rusal.go` (each under `internal/service/trading_strategy/momentum/strategy/`).
- **Modify** `data/params/rual/momentum_grid.json` — add an exit-tuning phase.

The `_test.go` import block already has `strings`, `testing`, `model`, `strategy`. Task 3 adds `tinvest/pkg/indicators` to the test imports.

---

## Task 1: Independent TP/trail toggles + MinRR decoupling

Decouple the fixed take-profit from the trailing stop (today `UseTrail` switches between them) and stop `TakeProfitRR<=0` from blocking all entries via the `MinRR` filter.

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Write failing tests**

Add to `core_test.go`:

```go
func TestEntryFiresWithFixedTPDisabled(t *testing.T) {
	p := defaultParams()
	p.TakeProfitRR = 0 // no fixed TP -> MinRR filter must not block the entry
	s := NewWithParams("TEST", p)
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire when TakeProfitRR=0 (MinRR check skipped)")
	}
}

func TestNoTakeProfitExitWhenDisabled(t *testing.T) {
	p := defaultParams()
	p.TakeProfitRR = 0 // fixed TP disabled
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", p)
	// barHigh 999 would smash any fixed TP; with TakeProfitRR=0 there must be no TP exit.
	sig := s.Decide(inPositionMD(100, 999, 100, pos))
	if sig.Kind == model.SignalSell && sig.Reason == "TP" {
		t.Fatal("no TP exit should fire when TakeProfitRR=0")
	}
}

func TestFixedTPFiresEvenWithTrailEnabled(t *testing.T) {
	// New behavior: TP and trail are independent. Trail on, but TP target hit first.
	p := defaultParams()
	p.UseTrail = 1
	p.TrailArmATR = 0   // armed immediately
	p.TakeProfitRR = 2  // TP = 100 + 2*5 = 110
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", p)
	// recentHigh 100 -> chandelier ~97.5; barLow 111 stays above it (no trail);
	// barHigh 111 >= TP 110 -> fixed TP must still fire even though UseTrail=1.
	sig := s.Decide(inPositionMD(111, 111, 100, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("kind=%v reason=%q want Sell/TP (TP independent of trail)", sig.Kind, sig.Reason)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestEntryFiresWithFixedTPDisabled|TestNoTakeProfitExitWhenDisabled|TestFixedTPFiresEvenWithTrailEnabled' -v`
Expected: `TestEntryFiresWithFixedTPDisabled` FAILS (entry blocked by MinRR), and `TestFixedTPFiresEvenWithTrailEnabled` FAILS (current code drops TP when UseTrail=1). `TestNoTakeProfitExitWhenDisabled` may already pass — that is fine.

- [ ] **Step 3: Decouple MinRR in `decide`**

In `core.go`, in `decide`, replace this block:

```go
	target := in.price + s.p.TakeProfitRR*risk
	if s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return sig
	}
```
with:
```go
	target := in.price + s.p.TakeProfitRR*risk
	// MinRR only gates the fixed-TP reward. With no fixed TP (TakeProfitRR<=0) the
	// trade is managed by trail/MACD/RSI exits, so the RR filter does not apply.
	if s.p.TakeProfitRR > 0 && s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return sig
	}
```

- [ ] **Step 4: Decouple MinRR in `Explain`**

In `core.go`, in `Explain` (section `// 7. Risk / RR sanity.`), replace:

```go
	target := in.price + s.p.TakeProfitRR*risk
	if s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return block("RR: цель %.4f даёт %.2gR < MinRR %.2g", target, (target-in.price)/risk, s.p.MinRR)
	}
```
with:
```go
	target := in.price + s.p.TakeProfitRR*risk
	if s.p.TakeProfitRR > 0 && s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return block("RR: цель %.4f даёт %.2gR < MinRR %.2g", target, (target-in.price)/risk, s.p.MinRR)
	}
```

- [ ] **Step 5: Rewrite `manage` for independent toggles**

In `core.go`, replace the whole `manage` body:

```go
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	entry := in.pos.PurchasePrice
	hardSL := in.pos.StopLoss
	// TP is reconstructed from the frozen entry stop (Position carries no target):
	// risk and stop are both fixed at entry, so this is deterministic.
	risk := entry - hardSL
	tp := entry + s.p.TakeProfitRR*risk

	sig.StopLoss = hardSL
	sig.TakeProfit = tp

	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
	case s.p.UseTrail == 0 && in.barHigh >= tp:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
	case s.p.UseTrail == 1:
		chandelier := in.recentHigh - s.p.TrailMult*in.atr
		armed := s.p.TrailArmATR <= 0 || in.pos.MaxFavorablePrice >= entry+s.p.TrailArmATR*in.pos.EntryATR
		if armed && in.barLow <= chandelier {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
		}
	}
	return sig
}
```
with:
```go
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	entry := in.pos.PurchasePrice
	hardSL := in.pos.StopLoss
	// TP is reconstructed from the frozen entry stop (Position carries no target):
	// risk and stop are both fixed at entry, so this is deterministic.
	risk := entry - hardSL
	tp := entry + s.p.TakeProfitRR*risk
	chandelier := in.recentHigh - s.p.TrailMult*in.atr
	trailArmed := s.p.TrailArmATR <= 0 || in.pos.MaxFavorablePrice >= entry+s.p.TrailArmATR*in.pos.EntryATR

	sig.StopLoss = hardSL
	if s.p.TakeProfitRR > 0 {
		sig.TakeProfit = tp
	}

	// Exit on the first trigger; protective/intrabar stops are checked first so the
	// worst case for the position wins ties on a bar. MACD/RSI are close-confirmed.
	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
	case s.p.UseTrail == 1 && trailArmed && in.barLow <= chandelier:
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
	case s.p.TakeProfitRR > 0 && in.barHigh >= tp:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
	}
	return sig
}
```
(The MACD and RSI cases are added in Tasks 2 and 3.)

- [ ] **Step 6: Run the three new tests + full package**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestEntryFiresWithFixedTPDisabled|TestNoTakeProfitExitWhenDisabled|TestFixedTPFiresEvenWithTrailEnabled' -v`
Expected: all three PASS.
Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/`
Expected: `ok` — all existing tests still pass (the existing `TestExitTakeProfit` keeps `TakeProfitRR=2`, `TestExitTrailWhenEnabled` keeps `UseTrail=1` with `TakeProfitRR=2` but its barHigh 121 ≥ TP 110, so it now exits TP, not TRAIL — see note). 

> **Note for the implementer:** `TestExitTrailWhenEnabled` uses `inPositionMD(117, 121, 120, pos)` with `defaultParams()` (`TakeProfitRR=2` → TP=110). Under the new independent toggles, `barHigh 121 >= TP 110` fires **TP before TRAIL**, and the trail level (chandelier ≈ 117.5) sits *above* the TP, so the two cannot be separated by adjusting the bar. To keep the test exercising the trail in isolation, add `p.TakeProfitRR = 0` to that test (it already builds a local `p := defaultParams()`), keeping the existing `inPositionMD(117, 121, 120, pos)` call. With no fixed TP, `barLow 117 <= chandelier 117.5` fires `"TRAIL"`. Do not weaken the assertion.

- [ ] **Step 7: Verify gofmt/vet and commit**

Run: `gofmt -l internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go` (expect empty) and `go vet ./internal/service/trading_strategy/momentum/strategy/core/` (expect no output).

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "refactor(momentum): independent TP/trail exits + decouple MinRR from TakeProfitRR

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: MACD bearish-cross exit

Add an exit that fires when the MACD line crosses below its signal line (mirror of the bullish entry cross), gated by `UseMACDExit`.

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Write failing tests + shared helper**

Add to `core_test.go`. First the shared helper (reused in Task 3):

```go
// inPositionMDWithCloses builds an in-position snapshot from an explicit close
// series so indicator-driven exits (MACD/RSI) can be engineered. Highs/lows
// straddle each close by 0.3; the last bar's high/low are taken from the series
// too. Position keys the manage branch.
func inPositionMDWithCloses(closes []float64, pos *strategy.Position) strategy.MarketData {
	n := len(closes)
	highs := make([]float64, n)
	lows := make([]float64, n)
	vols := make([]int64, n)
	for i := 0; i < n; i++ {
		highs[i], lows[i], vols[i] = closes[i]+0.3, closes[i]-0.3, 1000
	}
	return strategy.MarketData{
		Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: vols, Position: pos,
	}
}

// risingThenDropCloses builds a steady uptrend of `up` bars (+step each) followed
// by a single sharp drop of `drop`, which flips a MACD bullish posture to a
// bearish cross on the last bar.
func risingThenDropCloses(up int, start, step, drop float64) []float64 {
	closes := make([]float64, up+1)
	for i := 0; i < up; i++ {
		closes[i] = start + float64(i)*step
	}
	closes[up] = closes[up-1] - drop
	return closes
}

func TestExitMACDBearishCross(t *testing.T) {
	closes := risingThenDropCloses(60, 100, 0.5, 6) // long rise, then -6 last bar
	p := defaultParams()
	p.UseMACDExit = 1
	p.TakeProfitRR = 0 // isolate: no TP
	p.UseTrail = 0
	// SL far below the last bar low so only the MACD exit can fire.
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMDWithCloses(closes, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "MACD" {
		t.Fatalf("kind=%v reason=%q want Sell/MACD", sig.Kind, sig.Reason)
	}
}

func TestNoMACDExitWhenDisabled(t *testing.T) {
	closes := risingThenDropCloses(60, 100, 0.5, 6)
	p := defaultParams()
	p.UseMACDExit = 0 // disabled
	p.TakeProfitRR = 0
	p.UseTrail = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	if sig := s.Decide(inPositionMDWithCloses(closes, pos)); sig.Kind == model.SignalSell && sig.Reason == "MACD" {
		t.Fatal("no MACD exit should fire when UseMACDExit=0")
	}
}

func TestExitStopLossWinsOverMACD(t *testing.T) {
	closes := risingThenDropCloses(60, 100, 0.5, 6)
	p := defaultParams()
	p.UseMACDExit = 1
	p.TakeProfitRR = 0
	p.UseTrail = 0
	// StopLoss above the last bar low (lastClose-0.3) so SL triggers on the same bar.
	last := closes[len(closes)-1]
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: last + 0.1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	if sig := s.Decide(inPositionMDWithCloses(closes, pos)); sig.Reason != "SL" {
		t.Fatalf("reason=%q want SL (priority over MACD)", sig.Reason)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail / validate precondition**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run TestExitMACDBearishCross -v`
Expected: FAIL — `UseMACDExit` is not a field yet, so the file will not compile (`unknown field UseMACDExit`). That is the expected failing state for this step.

- [ ] **Step 3: Add the `UseMACDExit` param**

In `core.go`, in the `Params` struct, after the `UseTrail` field add:

```go
	UseMACDExit       int     // 1 = exit when MACD crosses below its signal line
```

- [ ] **Step 4: Compute `macdCrossDown` in `buildInput` and carry it in `decideInput`**

In `core.go`, in `decideInput`, after the `macdAboveSignal` field add:

```go
	macdCrossDown   bool // MACD line just crossed below its signal line (bearish)
```

In `buildInput`, replace the MACD block:

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
with:
```go
	macdNow, crossUp, crossDown, macdAboveSignal := 0.0, false, false, false
	if m, sg := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal); len(m) >= 2 {
		prevDiff := m[len(m)-2] - sg[len(sg)-2]
		currDiff := m[len(m)-1] - sg[len(sg)-1]
		macdNow = m[len(m)-1]
		crossUp = prevDiff <= 0 && currDiff > 0
		crossDown = prevDiff >= 0 && currDiff < 0
		macdAboveSignal = currDiff > 0
	}
```
Then in the `return decideInput{...}` literal, after `crossUp: crossUp,` add:
```go
		macdCrossDown:   crossDown,
```

- [ ] **Step 5: Add the MACD case to `manage`**

In `core.go`, in `manage`, add a new case to the switch **after** the `TakeProfitRR > 0 && in.barHigh >= tp` case and before the closing `}`:

```go
	case s.p.UseMACDExit == 1 && in.macdCrossDown:
		sig.Kind, sig.Reason = model.SignalSell, "MACD"
```

- [ ] **Step 6: Run the new tests + full package**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestExitMACDBearishCross|TestNoMACDExitWhenDisabled|TestExitStopLossWinsOverMACD' -v`
Expected: all PASS. If `TestExitMACDBearishCross` fails with a non-MACD reason, the engineered series did not produce a bearish cross — increase `drop` (e.g. 6 → 10) until the MACD line crosses below signal on the last bar.
Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/`
Expected: `ok`.

- [ ] **Step 7: gofmt/vet + commit**

Run: `gofmt -l <both files>` (empty) and `go vet ./internal/service/trading_strategy/momentum/strategy/core/`.

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "feat(momentum): exit on bearish MACD cross (UseMACDExit)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: RSI overbought cross-down exit

Add an exit that fires when RSI (parameterized length) crosses the overbought line from above, gated by `RSIPeriod > 0`.

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Add the `indicators` import to the test file**

In `core_test.go`, extend the import block to:

```go
import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)
```

- [ ] **Step 2: Write failing tests**

Add to `core_test.go`. The overbought line is chosen at runtime to sit strictly between the last two RSI values, so the test is robust to the exact Wilder-RSI numbers — it only requires that RSI declines on the last bar (which `risingThenDropCloses` guarantees):

```go
func TestExitRSIOverboughtCrossDown(t *testing.T) {
	const period = 14
	closes := risingThenDropCloses(40, 100, 1.0, 4) // rise then a drop -> RSI falls last bar
	r := indicators.RSISeries(closes, period)
	prev, now := r[len(r)-2], r[len(r)-1]
	if !(prev > now) {
		t.Fatalf("test setup: need prev RSI > now RSI, got prev=%v now=%v", prev, now)
	}
	ob := (prev + now) / 2 // overbought line strictly between -> a top-down cross

	p := defaultParams()
	p.RSIPeriod = period
	p.RSIOverbought = ob
	p.TakeProfitRR = 0
	p.UseTrail = 0
	p.UseMACDExit = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMDWithCloses(closes, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("kind=%v reason=%q want Sell/RSI (prev=%.2f now=%.2f ob=%.2f)", sig.Kind, sig.Reason, prev, now, ob)
	}
}

func TestNoRSIExitWhenDisabled(t *testing.T) {
	closes := risingThenDropCloses(40, 100, 1.0, 4)
	p := defaultParams()
	p.RSIPeriod = 0 // RSI exit disabled
	p.TakeProfitRR = 0
	p.UseTrail = 0
	p.UseMACDExit = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	if sig := s.Decide(inPositionMDWithCloses(closes, pos)); sig.Kind == model.SignalSell && sig.Reason == "RSI" {
		t.Fatal("no RSI exit should fire when RSIPeriod=0")
	}
}

func TestNoRSIExitWhenLineNotCrossed(t *testing.T) {
	const period = 14
	closes := risingThenDropCloses(40, 100, 1.0, 4)
	r := indicators.RSISeries(closes, period)
	prev, now := r[len(r)-2], r[len(r)-1]
	p := defaultParams()
	p.RSIPeriod = period
	p.RSIOverbought = now - 5 // line below both values -> no top-down cross
	p.TakeProfitRR = 0
	p.UseTrail = 0
	p.UseMACDExit = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	if sig := s.Decide(inPositionMDWithCloses(closes, pos)); sig.Kind == model.SignalSell && sig.Reason == "RSI" {
		t.Fatalf("no RSI exit when line %.2f is below both prev=%.2f now=%.2f", now-5, prev, now)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run TestExitRSIOverboughtCrossDown -v`
Expected: FAIL — `RSIPeriod`/`RSIOverbought` are not fields yet, so the package will not compile (`unknown field RSIPeriod`). Expected failing state.

- [ ] **Step 4: Add the RSI params**

In `core.go`, in the `Params` struct, after the `UseMACDExit` field add:

```go
	RSIPeriod         int     // RSI length for the overbought-exit; 0 disables the RSI exit
	RSIOverbought     float64 // RSI overbought line for the exit (e.g. 70); used when RSIPeriod>0
```

- [ ] **Step 5: Add RSI to `Lookback`**

In `core.go`, in `Lookback`, add `s.p.RSIPeriod + 1` to the candidate slice:

```go
	for _, c := range []int{
		s.p.MACDSlow + s.p.MACDSignal,
		s.p.VolLookback + 1,
		s.p.ATRPeriod + 1,
		s.p.SwingLowWindow,
		s.p.ChandelierWindow,
		s.p.RSIPeriod + 1,
	} {
```

- [ ] **Step 6: Compute RSI in `buildInput` and carry it in `decideInput`**

In `core.go`, in `decideInput`, after the `macdCrossDown` field add:

```go
	rsiNow          float64 // latest RSI value (0 when RSI exit disabled / insufficient history)
	rsiPrev         float64 // previous-bar RSI value
```

In `buildInput`, after the MACD block (before the `var barHigh, barLow` block) add:

```go
	var rsiNow, rsiPrev float64
	if s.p.RSIPeriod > 0 {
		if r := indicators.RSISeries(md.Closes, s.p.RSIPeriod); len(r) >= 2 {
			rsiNow, rsiPrev = r[len(r)-1], r[len(r)-2]
		}
	}
```

In the `return decideInput{...}` literal, after `macdCrossDown: crossDown,` add:

```go
		rsiNow:          rsiNow,
		rsiPrev:         rsiPrev,
```

- [ ] **Step 7: Add the RSI case to `manage`**

In `core.go`, in `manage`, add the RSI case to the switch **after** the MACD case (added in Task 2), as the last case:

```go
	case s.p.RSIPeriod > 0 && in.rsiPrev > s.p.RSIOverbought && in.rsiNow <= s.p.RSIOverbought:
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
```

- [ ] **Step 8: Run the new tests + full package**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestExitRSIOverboughtCrossDown|TestNoRSIExitWhenDisabled|TestNoRSIExitWhenLineNotCrossed' -v`
Expected: all PASS.
Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/`
Expected: `ok`.

- [ ] **Step 9: gofmt/vet + commit**

Run: `gofmt -l <both files>` (empty) and `go vet ./internal/service/trading_strategy/momentum/strategy/core/`.

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "feat(momentum): exit on RSI overbought cross-down (RSIPeriod/RSIOverbought)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Defaults across 8 tickers + RUAL grid

Add the new params to every per-ticker `DefaultParams()` (keeping the new exits OFF for backward compatibility) and extend the RUAL calibration grid with an exit-tuning phase. This is a configuration/data task; correctness is verified by build + the full test suite + JSON validity, so it adds no new unit test.

**Files:**
- Modify: 8 files `internal/service/trading_strategy/momentum/strategy/{afks/afks,mdmg/mdmg,sber/sber,nvtk/nvtk,ydex/ydex,plzl/plzl,gazp/gazp,rusal/rusal}.go`
- Modify: `data/params/rual/momentum_grid.json`

- [ ] **Step 1: Add the new fields to all 8 `DefaultParams()`**

In each of the 8 files, the `core.Params{...}` literal currently contains a line like:
```go
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
```
Append the three new fields to that literal (on that line or the next). For example, in `rusal/rusal.go` change:
```go
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 20, SignalValidBars: 4,
```
to:
```go
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 20, SignalValidBars: 4,
		UseMACDExit: 0, RSIPeriod: 0, RSIOverbought: 70,
```
Apply the same `UseMACDExit: 0, RSIPeriod: 0, RSIOverbought: 70,` addition to the other 7 files' `DefaultParams()` literals. Keep each file gofmt-clean (align/format as the file already does).

- [ ] **Step 2: Verify build + full momentum test suite**

Run: `go build ./...`
Expected: builds with no errors.
Run: `go test ./internal/service/trading_strategy/momentum/...`
Expected: `ok` for all momentum packages.

- [ ] **Step 3: Extend the RUAL calibration grid**

Replace the contents of `data/params/rual/momentum_grid.json` with (adds a third `exits` phase sweeping the new toggles; keeps the existing `core` and `gates` phases intact):

```json
{
  "phases": [
    {
      "name": "core",
      "keepTop": 5,
      "grid": {
        "EMAPeriod": [200, 150, 100],
        "SLMult": [0.5, 1.0, 1.5],
        "TakeProfitRR": [1.5, 2.0, 3.0],
        "VolMultiplier": [1.0, 1.2, 1.5],
        "CooldownBars": [0, 6, 12, 24],
        "MaxDailyATRUsed": [0.5, 0.7],
        "MACDBelowZeroOnly": [0, 1]
      }
    },
    {
      "name": "gates",
      "grid": {
        "MACDFast": [12, 10, 8],
        "MACDSlow": [21, 26, 18],
        "CooldownBars": [0, 6, 12, 24],
        "DailyTrendPeriod": [0, 10, 20],
        "SignalValidBars": [0, 2, 4],
        "MaxDriftATR": [0, 1.0]
      }
    },
    {
      "name": "exits",
      "grid": {
        "UseTrail": [0, 1],
        "TakeProfitRR": [0, 2.0, 3.0],
        "UseMACDExit": [0, 1],
        "RSIPeriod": [0, 14],
        "RSIOverbought": [70]
      }
    }
  ]
}
```

- [ ] **Step 4: Validate the grid JSON parses and a calibration run starts**

Run: `python3 -c "import json,sys; json.load(open('data/params/rual/momentum_grid.json')); print('json ok')"`
Expected: `json ok`.
Run (sanity, short window — confirms the new phase keys are accepted by the calibrator without error; Ctrl-C after it prints the calibration header is fine if it is slow):
`go run ./cmd/backtest -ticker RUAL -strategy momentum -calibrate data/params/rual/momentum_grid.json -out ./reports/RUAL -months 12 -min-trades 5 -test-months 3 -metric profit_factor`
Expected: it runs and writes reports without an "unknown parameter" / reflection error. (If grid keys are rejected, the param name does not match a `core.Params` field — fix the key.)

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/momentum/strategy/afks/afks.go internal/service/trading_strategy/momentum/strategy/mdmg/mdmg.go internal/service/trading_strategy/momentum/strategy/sber/sber.go internal/service/trading_strategy/momentum/strategy/nvtk/nvtk.go internal/service/trading_strategy/momentum/strategy/ydex/ydex.go internal/service/trading_strategy/momentum/strategy/plzl/plzl.go internal/service/trading_strategy/momentum/strategy/gazp/gazp.go internal/service/trading_strategy/momentum/strategy/rusal/rusal.go data/params/rual/momentum_grid.json
git commit -m "feat(momentum): default new exit params off across tickers + RUAL exit grid phase

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Final verification (after all tasks)

- [ ] Run `go build ./...` — clean.
- [ ] Run `go test ./internal/service/trading_strategy/momentum/...` — all `ok`.
- [ ] Run `gofmt -l internal/service/trading_strategy/momentum/` — empty.
- [ ] Confirm default behavior unchanged: the only param semantics that changed (`TakeProfitRR<=0`, `UseTrail` independence) do not alter any shipped `DefaultParams()` (all keep `TakeProfitRR=2`, and TP+trail were never both enabled in defaults).

## Notes / out of scope

- Validating the new exits on out-of-sample data (walk-forward) is a follow-up via `-calibrate`/`-test-months`; the strategy is known to be overfit, so this is a hypothesis to test, not a confirmed improvement.
- Grids for the other 7 tickers are not changed here.
- `entryReason` still prints the TP/`R` figures even when `TakeProfitRR=0` (renders `+0.0000, 0R`); harmless and left as-is to keep this change focused.
