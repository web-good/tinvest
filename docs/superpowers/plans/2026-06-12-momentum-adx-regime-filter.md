# Momentum ADX Regime Filter — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a fixed hourly ADX(14) ≥ 25 regime gate to the momentum strategy's entry, applied identically to all tickers and never swept, so a basket walk-forward can test whether filtering out chop produces a real edge.

**Architecture:** The ADX indicator already exists and is tested in `pkg/indicators/adx.go` (returns the last-bar ADX). We add two in-package constants in `core` (`adxPeriod=14`, `adxThreshold=25.0`), thread the computed ADX through the existing `decideInput`, and gate `decide()` on it as the FIRST check — before confluence. The threshold is a constant, NOT a `Params` field, so reflection grid calibration can never sweep it. Entry rationale and the `Explain()` diagnostic gain an ADX line.

**Tech Stack:** Go 1.25, existing `pkg/indicators.ADX`, the momentum `core` package, table-driven `testing`.

---

## Context for the implementer

- Spec: `docs/superpowers/specs/2026-06-12-momentum-adx-regime-filter-design.md`.
- The strategy is long-only, hourly. Entry today = a MACD↔RSI confluence + uptrend + volume + risk/RR gates. We are adding ONE gate on top: skip entry when the instrument is in chop (low ADX).
- `pkg/indicators.ADX` signature (already implemented, do not rewrite):
  ```go
  func ADX(highs, lows, closes []float64, period int) (adx, diPlus, diMinus float64)
  ```
  It returns `(0,0,0)` on insufficient history (`len < 2*period+1`) — that reads as "no trend" and safely blocks entry. We only use the first return value (`adx`); ignore `diPlus`/`diMinus`.
- The pure decision core lives in `internal/service/trading_strategy/momentum/strategy/core/core.go`:
  - `decideInput` (struct of pre-computed indicator values) is the boundary between the impure shell (`buildInput`, computes indicators from `MarketData`) and the pure `decide()`.
  - `decide()` short-circuits gate by gate; the FIRST gate today is `confluence(...)`. The ADX gate goes BEFORE it.
  - `Explain()` mirrors `decide()`'s gate order for diagnostics and must gain a matching ADX line as gate #1.
  - `entryReason(...)` renders the trade-journal rationale string.
- Test file: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`.
  - `passingInput()` returns a `decideInput` that passes every gate; decide-level tests build on it.
  - `buildEntryMD()` builds a 260-bar **rising** series (`base = 100 + 0.5*i`) with a dip-then-pop at the end. Because it is a sustained uptrend, its ADX(14) is very high (≈100), so the real-`MarketData` entry/Explain tests clear the new gate with no fixture change. A guard sub-test will assert this explicitly.
- **No `Params` change** → the frozen-baseline test `TestGenericMomentumDefaultsAreFrozenBaseline` and all `momentum_registry_test.go` tests stay green untouched. Do not modify them.
- Run the core package tests with:
  `go test ./internal/service/trading_strategy/momentum/strategy/core/...`

---

## Task 1: ADX regime gate in the pure core

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Add the constants, the `decideInput` field, the `buildInput` wiring, and extend `Lookback`**

In `core.go`, just below the existing `signalSaturate` const block (around line 25), add:

```go
// adxPeriod / adxThreshold define the regime gate: entry is allowed only when the
// hourly ADX(adxPeriod) is at least adxThreshold, i.e. the instrument is trending
// rather than ranging. The threshold is a fixed in-package constant — deliberately
// NOT a Params field — so grid calibration can never tune it per ticker. This makes
// the basket walk-forward an honest test of whether filtering chop yields real edge.
// Ablation: set adxThreshold to 0 to disable the gate.
const (
	adxPeriod    = 14
	adxThreshold = 25.0
)
```

In the `decideInput` struct (around line 91), add the field right after `emaTrend`:

```go
	adx float64 // hourly ADX(adxPeriod); the regime gate requires adx >= adxThreshold
```

In `buildInput` (around line 144, alongside the ATR computation), compute the ADX from the same hourly H/L/C series and set it on the returned struct. Add near the top of the function:

```go
	adx, _, _ := indicators.ADX(md.Highs, md.Lows, md.Closes, adxPeriod)
```

and add `adx: adx,` to the returned `decideInput{...}` literal (place it next to `emaTrend:`).

In `Lookback()` (around line 73), add `2*adxPeriod + 1` to the slice of candidate minimums so the window always covers ADX's double-smoothing warmup (EMAPeriod already dominates, but make it explicit):

```go
		2*adxPeriod + 1,
```

- [ ] **Step 2: Run the full core suite — expect still green (field computed but unused by `decide`)**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/...`
Expected: PASS (the ADX value is wired but no gate reads it yet).

- [ ] **Step 3: Set `passingInput()` to clear the gate, then write the failing gate test**

In `core_test.go`, add `adx: 30,` to the `passingInput()` literal (any value ≥ 25; 30 is comfortably above threshold) so all existing decide-level tests keep passing once the gate lands:

```go
func passingInput() decideInput {
	return decideInput{
		price:              100,
		emaTrend:           90,
		adx:                30,
		atr:                1,
		volumeOK:           true,
		recentLow:          98,
		barsSinceMACDCross: signalSaturate,
		barsSinceRSICross:  signalSaturate,
	}
}
```

Then add this new test (place it just after `TestGateBlocks`):

```go
// --- ADX regime gate ---

func TestADXRegimeGate(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 0

	// A fully entry-qualifying input (confluence + trend + volume + RR all pass).
	base := passingInput()
	base.macdFired = true
	base.rsiFired = true
	base.barsSinceMACDCross = 0
	base.barsSinceRSICross = 0

	t.Run("blocks_below_threshold", func(t *testing.T) {
		s := NewWithParams("TEST", p)
		in := base
		in.adx = 20 // < 25 -> chop -> no entry despite full confluence
		if sig := s.decide(in); sig.Kind == model.SignalBuy {
			t.Fatal("want no Buy when ADX below threshold (chop)")
		}
	})

	t.Run("allows_at_threshold", func(t *testing.T) {
		s := NewWithParams("TEST", p)
		in := base
		in.adx = adxThreshold // exactly 25 -> allowed (gate is >=)
		if sig := s.decide(in); sig.Kind != model.SignalBuy {
			t.Fatalf("want Buy when ADX == threshold, got kind=%v", sig.Kind)
		}
	})
}
```

- [ ] **Step 4: Run the new test — expect the block case to FAIL**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/... -run TestADXRegimeGate -v`
Expected: `blocks_below_threshold` FAILS (decide still returns Buy because no gate exists yet); `allows_at_threshold` passes.

- [ ] **Step 5: Add the gate as the FIRST check in `decide()`**

In `core.go`, in `decide()`, immediately after the open-position early return (`if in.pos != nil { return s.manage(in, sig) }`) and BEFORE the confluence check, add:

```go
	// Regime gate: only trade when the instrument is trending (ADX >= threshold),
	// never in chop. Fixed threshold, identical for every ticker, never calibrated.
	if in.adx < adxThreshold {
		return sig
	}
```

- [ ] **Step 6: Run the gate test, then the whole core suite — expect green**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/... -run TestADXRegimeGate -v`
Expected: both sub-cases PASS.

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/...`
Expected: PASS (all decide-level tests use `passingInput()` with `adx:30`).

- [ ] **Step 7: Add a guard sub-test confirming `buildEntryMD` clears the gate**

Append to `TestBuildInputWiring` a sub-test so the real-`MarketData` entry/Explain tests' reliance on a high ADX is explicit and self-documenting:

```go
	t.Run("adx_above_threshold_on_trending_fixture", func(t *testing.T) {
		p := defaultParams()
		s := NewWithParams("TEST", p)
		in := s.buildInput(buildEntryMD())
		if in.adx < adxThreshold {
			t.Fatalf("buildEntryMD ADX=%.2f want >= %.0f (sustained uptrend should be strongly trending)", in.adx, adxThreshold)
		}
	})
```

- [ ] **Step 8: Run the suite + vet + gofmt**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/...`
Expected: PASS (the guard sub-test confirms ADX ≈ 100 ≥ 25).

Run: `go vet ./internal/service/trading_strategy/momentum/strategy/core/... && gofmt -l internal/service/trading_strategy/momentum/strategy/core/`
Expected: no vet errors; `gofmt -l` prints nothing.

- [ ] **Step 9: Commit**

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go \
        internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "feat(momentum): gate entry on fixed hourly ADX(14) >= 25 regime filter

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Surface ADX in entry rationale and the Explain diagnostic

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing assertion for the entry-reason ADX prefix**

In `core_test.go`, inside `TestEndToEndEntryDecide`'s `fires_with_rsiCrossLevel75` sub-test, after the existing `EntryReason missing MACD or RSI` check (around line 313), add:

```go
		if !strings.Contains(sig.EntryReason, "ADX") {
			t.Fatalf("EntryReason missing ADX prefix: %q", sig.EntryReason)
		}
```

- [ ] **Step 2: Run it — expect FAIL**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/... -run TestEndToEndEntryDecide -v`
Expected: FAIL — `EntryReason missing ADX prefix`.

- [ ] **Step 3: Prepend the ADX clause to `entryReason`**

In `core.go`, in `entryReason(...)`, prepend an ADX clause to the returned string. Change the leading `"Тренд↑ ..."` so the format string starts with the ADX segment and the args list leads with `in.adx`:

```go
	return fmt.Sprintf(
		"ADX %.1f ≥ %.0f (тренд подтверждён); Тренд↑ (close %.4f > EMA%d %.4f); MACD бычий кросс (%s, %.4f) %d бар(ов) назад; RSI пересёк %.0f↑ (%.2f→%.2f) %d бар(ов) назад, зазор ≤ %d; объём > %.2g×ср(%d); SL=%.4f (-%.4f); TP=%.4f (+%.4f, %.2gR)",
		in.adx, adxThreshold,
		in.price, s.p.EMAPeriod, in.emaTrend, zero, in.macdNow, macdAge,
		s.p.RSICrossLevel, in.rsiPrev, in.rsiNow, rsiAge, s.p.SignalValidBars,
		s.p.VolMultiplier, s.p.VolLookback,
		stop, risk, target, target-in.price, s.p.TakeProfitRR,
	)
```

- [ ] **Step 4: Run the entry test — expect PASS**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/... -run TestEndToEndEntryDecide -v`
Expected: PASS.

- [ ] **Step 5: Write the failing Explain assertion**

In `core_test.go`, in `TestExplain`'s `rsi_fires_at_level75_all_pass` sub-test, after the `ВХОД` check (around line 374), add:

```go
			if !strings.Contains(out, "ADX") {
				t.Fatalf("Explain should report the ADX gate, got: %q", out)
			}
```

- [ ] **Step 6: Run it — expect FAIL**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/... -run TestExplain -v`
Expected: FAIL — `Explain should report the ADX gate`.

- [ ] **Step 7: Add the ADX gate as gate #1 in `Explain()`**

In `core.go`, in `Explain()`, insert the ADX gate immediately before the `// 1. MACD↔RSI confluence` block (so it mirrors `decide()`'s order — ADX is checked first):

```go
	// 1. Regime gate (ADX).
	if in.adx < adxThreshold {
		return block("ADX %.1f < %.0f — боковик, торговля запрещена", in.adx, adxThreshold)
	}
	pass("ADX %.1f ≥ %.0f (тренд подтверждён)", in.adx, adxThreshold)
```

Renumber the existing inline comments below it (`// 2. Uptrend`, `// 3. Volume`, `// 4. Risk / RR sanity`) accordingly; the confluence block becomes step 2, etc. (comment-only edits.)

- [ ] **Step 8: Run Explain + the whole core suite — expect green**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/... -run TestExplain -v`
Expected: PASS. (For `buildEntryMD` ADX ≈ 100, so at `RSICrossLevel=50` the ADX line passes and confluence still blocks → the `ВХОДА НЕТ` sub-test stays valid.)

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/...`
Expected: PASS.

- [ ] **Step 9: vet + gofmt + commit**

Run: `go vet ./internal/service/trading_strategy/momentum/strategy/core/... && gofmt -l internal/service/trading_strategy/momentum/strategy/core/`
Expected: clean.

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go \
        internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "feat(momentum): show ADX gate in EntryReason and Explain

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Document the ADX regime gate

**Files:**
- Modify: `docs/momentum/strategy.md`

- [ ] **Step 1: Read the current entry section**

Run: `sed -n '1,120p' docs/momentum/strategy.md` (or open it) to find the entry-gates section and its formatting/heading style.

- [ ] **Step 2: Add the ADX gate as the first entry gate**

Insert a subsection describing the ADX regime gate as the FIRST entry gate, before the confluence/trend/volume gates. Content to convey (match the doc's existing Russian prose style and heading level):

- **Режимный фильтр (ADX).** Перед любыми сигналами входа стратегия проверяет силу тренда индикатором ADX(14) на часовых барах. Вход разрешён только при `ADX ≥ 25`; ниже — рынок считается боковиком, и стратегия не торгует, сколько бы сигналов ни совпало.
- **Почему 25.** 25 — классическая граница «тренд / боковик» для ADX. Ниже неё направленного движения по сути нет, и MACD-кроссы в такой зоне дают ложные входы (основной источник убытка в боковых бумагах вроде GAZP/SBER).
- **Почему порог зафиксирован.** Порог — жёсткая внутренняя константа, общая для всех тикеров, и НЕ участвует в калибровке (его нет в `Params` и в гридах). Это сделано намеренно: если бы порог подбирался под каждый тикер, фильтр просто переобучился бы. Зафиксированный общий порог превращает корзинный walk-forward в честную проверку — есть ли у стратегии реальное преимущество.

- [ ] **Step 3: Commit**

```bash
git add docs/momentum/strategy.md
git commit -m "docs(momentum): describe the fixed ADX regime gate

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## After all tasks: verification & acceptance

- [ ] **Full build, vet, gofmt across the repo**

Run: `go build ./... && go vet ./... && gofmt -l internal/ pkg/ cmd/`
Expected: builds clean, no vet errors, `gofmt -l` prints nothing.

- [ ] **Full test suite**

Run: `go test ./...`
Expected: PASS. In particular the momentum registry/frozen-baseline tests pass unchanged (no `Params` change).

- [ ] **Acceptance (user-run, requires gRPC API access)**

The user reruns the basket walk-forward and reads the pooled OOS Profit Factor:

```bash
go run ./cmd/backtest -strategy momentum -interval Hour1 \
  -basket RUAL,AFKS,YDEX,PLZL,SBER,GAZP,NVTK,MDMG \
  -months 24 -test-months 12 -min-trades 10 \
  -metric profit_factor -out ./reports/basket
```

- **Pooled OOS PF > 1.0** → the ADX gate filtered chop and the edge is real → proceed to per-ticker recalibration + hardcoding winners.
- **Pooled OOS PF ≤ 1.0** even with the fixed gate → momentum core is a research dead-end → close the branch honestly.
