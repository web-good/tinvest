# Reversion Breakeven Exit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an optional pure-breakeven exit (`BE`) to the reversion strategy that, after price runs `BreakevenArmATR×EntryATR` in favor, closes the position at the first close back at/below entry — fixing the give-back losses on MDMG.

**Architecture:** Two new tunable `Params` (`UseBreakeven`, `BreakevenArmATR`). The exit lives in the pure `manage()` core, reading the already-engine-maintained `Position.MaxFavorablePrice` (no engine change). ATR is made available to the exit by widening the existing ATR compute/lookback gate to also fire when `UseBreakeven==1`. Default off everywhere; enabled + calibrated on MDMG only.

**Tech Stack:** Go 1.25, existing reversion `core` package + backtest engine; table-style unit tests mirroring `core_test.go`.

**Spec:** `docs/superpowers/specs/2026-06-16-reversion-breakeven-exit-design.md`

**Pre-verified facts (no action needed):**
- `Position.MaxFavorablePrice` is populated by the engine for every strategy (`internal/domain/backtest/portfolio.go:121`, monotonic high-water of closes, seeded to entry). Reversion can read it via `in.pos`.
- The engine fills any exit at `c.Close` except reasons `"SL"/"TRAIL"/"TP"` (`engine.go:138,221`). New reason `"BE"` therefore fills at close — consistent with `ATRSL`/`RSIOS`/`RSI50`/`EMAX`.
- `decide()` already stamps `sig.ATR = in.atr` on Buy → engine persists `Position.EntryATR`. Task 1 makes `in.atr` non-zero when `UseBreakeven==1`, so `EntryATR` is frozen at entry for the BE arm unit.

---

## Task 1: Add params + make ATR available to the breakeven exit

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (Params struct ~line 51; `buildInput` ATR gate ~line 150-153; `Lookback` ATR gate ~line 76-78)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `core_test.go`:

```go
func TestBuildInputATRForBreakeven(t *testing.T) {
	n := 60
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100 + float64(i)
		highs[i] = closes[i] + 1
		lows[i] = closes[i] - 1
	}
	md := strategy.MarketData{Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: make([]int64, n)}

	p := defaultParams()
	p.ATRPeriod = 14
	p.UseATRStop = 0

	p.UseBreakeven = 1
	if in := NewWithParams("T", p).buildInput(md); in.atr <= 0 {
		t.Fatalf("UseBreakeven=1: want atr>0 (BE needs EntryATR), got %v", in.atr)
	}

	p.UseBreakeven = 0
	if in := NewWithParams("T", p).buildInput(md); in.atr != 0 {
		t.Fatalf("UseBreakeven=0 & UseATRStop=0: ATR must not be computed, got %v", in.atr)
	}
}

func TestLookbackIncludesATRForBreakeven(t *testing.T) {
	p := defaultParams()
	p.FastEMA, p.SlowEMA = 50, 200
	p.RSIPeriod, p.StochKPeriod, p.StochDSmooth = 14, 14, 3
	p.ATRPeriod = 300 // dominates SlowEMA when ATR is needed
	p.UseATRStop = 0

	p.UseBreakeven = 1
	if got := NewWithParams("T", p).Lookback(); got != 306 {
		t.Fatalf("UseBreakeven=1: ATRPeriod=300 dominates, want 306, got %d", got)
	}

	p.UseBreakeven = 0
	if got := NewWithParams("T", p).Lookback(); got != 205 {
		t.Fatalf("UseBreakeven=0: ATRPeriod ignored, want SlowEMA+5=205, got %d", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'Breakeven' -v`
Expected: COMPILE FAIL — `p.UseBreakeven` undefined (field does not exist yet).

- [ ] **Step 3: Add the two fields to `Params`**

In `core.go`, in the `Params` struct, after the `HTFTrendEMA` line:

```go
	HTFTrendEMA     int     // EMA period on the 4H timeframe for the higher-timeframe trend filter; 0 = off
	UseBreakeven    int     // 1 = after price runs BreakevenArmATR×EntryATR in favor, exit at the first close back at/below entry; 0 = off
	BreakevenArmATR float64 // breakeven arm threshold in EntryATR multiples (e.g. 1.0); consulted only when UseBreakeven=1
```

- [ ] **Step 4: Widen the `buildInput` ATR gate**

In `core.go` `buildInput`, replace:

```go
	var atr float64
	if s.p.UseATRStop == 1 && s.p.ATRPeriod > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	}
```

with:

```go
	var atr float64
	if (s.p.UseATRStop == 1 || s.p.UseBreakeven == 1) && s.p.ATRPeriod > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	}
```

- [ ] **Step 5: Widen the `Lookback` ATR gate**

In `core.go` `Lookback`, replace:

```go
	if s.p.UseATRStop == 1 && s.p.ATRPeriod > 0 {
		cands = append(cands, s.p.ATRPeriod+1)
	}
```

with:

```go
	if (s.p.UseATRStop == 1 || s.p.UseBreakeven == 1) && s.p.ATRPeriod > 0 {
		cands = append(cands, s.p.ATRPeriod+1)
	}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'Breakeven' -v`
Expected: PASS (both tests).

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): add UseBreakeven/BreakevenArmATR params + ATR availability"
```

---

## Task 2: Implement the `BE` exit in `manage()` with correct precedence

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (`manage()` switch ~line 386-409; doc-comment above `manage()` ~line 365-382)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `core_test.go`:

```go
// breakevenParams: breakeven exit on, arm at 1.0×ATR, RSI oversold 30. ATR stop OFF so
// the middle exit is RSIOS and cannot mask the BE branch.
func breakevenParams() Params {
	p := defaultParams()
	p.UseBreakeven = 1
	p.BreakevenArmATR = 1.0
	p.RSIOversold = 30
	return p
}

func TestExitBreakevenFiresAfterArm(t *testing.T) {
	s := NewWithParams("T", breakevenParams())
	in := openInput() // neutral RSI/EMA: no RSI50, no EMAX
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106}
	in.price = 99 // <= entry 100, armed (106 >= 100 + 1.0*5 = 105)
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "BE" {
		t.Fatalf("armed + price<=entry: want BE sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestNoBreakevenBeforeArm(t *testing.T) {
	s := NewWithParams("T", breakevenParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 103} // 103 < 105: not armed
	in.price = 99
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("not armed: should hold, got sell %q", sig.Reason)
	}
}

func TestBreakevenOffByFlag(t *testing.T) {
	p := breakevenParams()
	p.UseBreakeven = 0
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106} // armed
	in.price = 99
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "BE" {
		t.Fatalf("UseBreakeven=0 must skip BE")
	}
}

func TestBreakevenInertWhenEntryATRZero(t *testing.T) {
	// Live-trading guard: EntryATR not persisted (0). Without the guard the arm threshold
	// collapses to PurchasePrice and BE would fire on any pullback.
	s := NewWithParams("T", breakevenParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 0, MaxFavorablePrice: 200}
	in.price = 1
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "BE" {
		t.Fatalf("EntryATR=0 must skip BE (live-trading guard)")
	}
}

func TestBreakevenInertWhenArmZero(t *testing.T) {
	// Zero arm: threshold equals PurchasePrice (armed immediately), turning BE into a plain
	// entry-price stop. Guard keeps it inert.
	p := breakevenParams()
	p.BreakevenArmATR = 0
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106}
	in.price = 99
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "BE" {
		t.Fatalf("BreakevenArmATR=0 must skip BE (zero-arm guard)")
	}
}

func TestExitPrecedenceBreakevenOverMiddle(t *testing.T) {
	// Armed, ATR stop also on. Price is below the ATR threshold (ATRSL would fire) AND
	// below entry (BE fires). BE has higher precedence -> reason BE.
	p := breakevenParams()
	p.UseATRStop = 1
	p.ATRPeriod = 14
	p.StopATRMult = 1.0
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106} // armed
	in.price = 94 // <= 95 ATR threshold AND <= 100 entry
	if sig := s.decide(in); sig.Reason != "BE" {
		t.Fatalf("BE must win over ATRSL, got %q", sig.Reason)
	}
}

func TestExitPrecedenceRSI50OverBreakeven(t *testing.T) {
	s := NewWithParams("T", breakevenParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106} // armed
	in.price = 99                  // BE would fire
	in.rsiPrev, in.rsiNow = 55, 45 // RSI50 also fires
	if sig := s.decide(in); sig.Reason != "RSI50" {
		t.Fatalf("RSI50 must win over BE, got %q", sig.Reason)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'Breakeven|BreakevenOverMiddle|RSI50OverBreakeven' -v`
Expected: FAIL — `TestExitBreakevenFiresAfterArm` gets no sell / wrong reason; others depend on the new branch.

- [ ] **Step 3: Insert the `BE` case into `manage()`**

In `core.go` `manage()`, insert this case **between** the `RSI50` case (`case in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, rsiExitLevel):`) and the RSIOS case (`case s.p.UseATRStop == 0 && ...`):

```go
	case s.p.UseBreakeven == 1 && in.pos.EntryATR > 0 && s.p.BreakevenArmATR > 0 &&
		in.pos.MaxFavorablePrice >= in.pos.PurchasePrice+s.p.BreakevenArmATR*in.pos.EntryATR &&
		in.price <= in.pos.PurchasePrice:
		sig.Kind, sig.Reason = model.SignalSell, "BE"
		sig.ExitReason = fmt.Sprintf("BE: цена %.4f ≤ вход %.4f после взвода (макс %.4f ≥ вход + %.2g×ATR %.4f)",
			in.price, in.pos.PurchasePrice, in.pos.MaxFavorablePrice, s.p.BreakevenArmATR, in.pos.EntryATR)
```

- [ ] **Step 4: Update the `manage()` doc-comment**

In `core.go`, in the comment block above `manage()`, add a bullet describing `BE` after the `RSI50` bullet and update the precedence sentence to read `OB → RSI50 → BE → middle(RSIOS xor ATRSL) → EMAX`:

```go
//   - BE: after price ran BreakevenArmATR×EntryATR in favor (armed via the monotonic
//     Position.MaxFavorablePrice), price has fallen back to/below PurchasePrice — pure
//     breakeven floor (gated by UseBreakeven=1, EntryATR>0, BreakevenArmATR>0). Fills at
//     close, so the exit is at the first close back at/below entry (may be a small loss).
```

- [ ] **Step 5: Run the full core test suite**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -v`
Expected: PASS (all existing tests + the 7 new ones). The existing `TestExitPrecedenceRSI50OverATR`, `TestExitATRStopFires`, `TestRSIOSInertWhenATRStopOn` must still pass (BE is off in `atrStopParams()`).

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): add BE breakeven exit (precedence OB>RSI50>BE>middle>EMAX)"
```

---

## Task 3: Enable + calibrate-seed on MDMG, update docs

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/mdmg/mdmg.go`
- Modify: `data/params/mdmg/reversion_grid.json`
- Modify: `docs/reversion/strategy.md`
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (package doc-comment, top of file)

- [ ] **Step 1: Update MDMG DefaultParams to the calibration winner + breakeven on**

Replace the body of `DefaultParams()` in `mdmg.go` and refresh the package/func comments:

```go
// Package mdmg supplies the ticker and calibrated reversion Params for MDMG (MD Medical).
package mdmg

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "MDMG"

// DefaultParams returns MDMG's calibrated reversion parameters.
//
// UseBreakeven is ON: a loss analysis (reports/_analysis/reversion_loss_analysis_2026-06-16.md)
// showed 7 of 12 losing trades ran >+1% in favor then reversed into the ATR stop (give-back).
// The breakeven floor (arm at 1.0×EntryATR) cuts those back to ~0. Other values are the
// 2026-06-16 calibration winner; the ATR stop is KEPT (removing it made MDMG worse).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 5, SlowEMA: 100,
		RSIPeriod:    6,
		RSIOversold:  30,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		UseATRStop: 1, ATRPeriod: 14, StopATRMult: 0.8,
		UseVolume: 1, VolAvgPeriod: 20, VolMult: 1.2,
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
		HTFTrendEMA:     10,
		UseBreakeven:    1,
		BreakevenArmATR: 1.0,
	}
}
```

- [ ] **Step 2: Add a breakeven phase to the MDMG grid**

In `data/params/mdmg/reversion_grid.json`, add this phase object to the `"phases"` array (after the `"HTFTrendEMA"` phase):

```json
    {
      "name": "breakeven",
      "grid": {
        "UseBreakeven": [0, 1],
        "BreakevenArmATR": [0.5, 1.0, 1.5]
      }
    }
```

- [ ] **Step 3: Document the BE exit in the strategy doc**

In `docs/reversion/strategy.md`, in the "Выход" section, update the intro line to "по одному из пяти сигналов" and add `BE` as a numbered exit after `OB` (it is precedence #2, shifting RSI50→3 etc.) — keep numbering consistent with `OB → RSI50 → BE → средний → EMAX`:

```markdown
2. **BE (опционально, `UseBreakeven`):** чистый безубыток. После того как цена ушла в плюс
   на `BreakevenArmATR × EntryATR` (взвод через монотонный максимум хода), при возврате к/ниже
   цены входа позиция закрывается. Исполнение по закрытию, поэтому выход — по первому закрытию
   на/ниже входа (возможен небольшой минус). Требует `EntryATR>0` и `BreakevenArmATR>0`
   (инертен в live). При `UseBreakeven=0` выключен.
```

And in the "Параметры" section add a block (and bump the count "18" → "20"):

```markdown
**Выход в безубыток (BE)**
- `UseBreakeven` — `1` включает безубыток-выход; `0` (по умолчанию) выключает.
- `BreakevenArmATR` — порог взвода в кратных EntryATR (по умолчанию `1.0`); учитывается
  только при `UseBreakeven=1`. EntryATR берётся из ATR рабочего ТФ, посчитанного на входе
  (при `UseBreakeven=1` ATR считается, даже если ATR-стоп выключен).
```

- [ ] **Step 4: Update the core package doc-comment**

In `core.go`, in the top-of-file package comment, add to the exit enumeration a mention of BE between RSI50 and the middle exit, e.g. append after the OB/RSI50 description:

```go
// a breakeven floor that, once price has run BreakevenArmATR×EntryATR in favor, exits at the
// first close back at/below entry (BE, gated by UseBreakeven);
```

- [ ] **Step 5: Build, vet, full reversion test**

Run: `go build ./... && go vet ./internal/service/trading_strategy/reversion/... && go test ./internal/service/trading_strategy/reversion/... ./internal/service/backtest/...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/mdmg/mdmg.go data/params/mdmg/reversion_grid.json docs/reversion/strategy.md internal/service/trading_strategy/reversion/strategy/core/core.go
git commit -m "feat(reversion): enable breakeven on MDMG + grid + docs"
```

---

## Task 4: Propagate the param to the other defaults + grids (consistency)

**Files:**
- Modify: `internal/service/backtest/reversion_registry.go` (`genericReversionDefaults` ~line 51)
- Modify: per-ticker defaults: `afks/afks.go`, `gazp/gazp.go`, `nvtk/nvtk.go`, `plzl/plzl.go`, `rusal/rusal.go`, `sber/sber.go`, `ydex/ydex.go` (all under `internal/service/trading_strategy/reversion/strategy/`)
- Modify: per-ticker grids: `data/params/{afks,gazp,nvtk,plzl,rual,sber,ydex}/reversion_grid.json`

> NOTE: NVTK was already calibrated (UseATRStop=0). Keep its other params; only ADD the two breakeven fields set OFF. RUSAL's grid folder is `rual`.

- [ ] **Step 1: Add the OFF default to `genericReversionDefaults`**

In `reversion_registry.go`, add to the returned `core.Params{...}` literal:

```go
		UseBreakeven: 0, BreakevenArmATR: 1.0,
```

- [ ] **Step 2: Add the OFF default to each per-ticker `DefaultParams`**

In each of the 7 files (`afks, gazp, nvtk, plzl, rusal, sber, ydex`), add the same line inside the `core.Params{...}` literal returned by `DefaultParams()`:

```go
		UseBreakeven: 0, BreakevenArmATR: 1.0,
```

- [ ] **Step 3: Add the breakeven sweep phase to each per-ticker grid**

In each of the 7 grid files (`data/params/{afks,gazp,nvtk,plzl,rual,sber,ydex}/reversion_grid.json`), append this phase to the `"phases"` array:

```json
    {
      "name": "breakeven",
      "grid": {
        "UseBreakeven": [0, 1],
        "BreakevenArmATR": [0.5, 1.0, 1.5]
      }
    }
```

- [ ] **Step 4: Build, vet, full repo test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all PASS. (The registry test `reversion_registry_test.go` exercises `genericReversionDefaults` + `ParseParams`; confirm it still passes — partial JSON overrides layer over the new fields automatically.)

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/reversion_registry.go internal/service/trading_strategy/reversion/strategy/ data/params/
git commit -m "chore(reversion): propagate breakeven params (off) to all defaults + grids"
```

---

## Task 5: Verification A/B on MDMG (OOS)

**Files:**
- Create: `data/params/mdmg/reversion_be.json` (pinned-winner grid sweeping only the breakeven toggle)

- [ ] **Step 1: Create the pinned A/B grid**

Create `data/params/mdmg/reversion_be.json`:

```json
{
  "phases": [{
    "name": "be-ab", "keepTop": 6,
    "grid": {
      "UseTrend":[1], "FastEMA":[5], "SlowEMA":[100],
      "RSIPeriod":[6], "RSIOversold":[30],
      "StochKPeriod":[14], "StochDSmooth":[1], "StochOversold":[20],
      "UseATRStop":[1], "ATRPeriod":[14], "StopATRMult":[0.8],
      "UseVolume":[1], "VolAvgPeriod":[20], "VolMult":[1.2],
      "UseOverbought":[1], "RSIOverbought":[70], "StochOverbought":[80],
      "HTFTrendEMA":[10],
      "UseBreakeven":[0,1], "BreakevenArmATR":[0.5,1.0,1.5]
    }
  }]
}
```

- [ ] **Step 2: Run the OOS backtest (same protocol as the ATR-stop A/B)**

Run:
```bash
go run ./cmd/backtest -ticker MDMG -strategy reversion \
  -calibrate data/params/mdmg/reversion_be.json -out ./reports/MDMG \
  -months 50 -test-months 12 -metric net_pnl
```
Expected: writes a new `reports/MDMG/MDMG_reversion_Hour1_<ts>_calibration.md` (4 combos: UseBreakeven 0 × arm-values collapse to one OFF result + 3 ON arm-values) and `..._best.md` (OOS holdout of the winning combo).

- [ ] **Step 3: Read and compare**

Open the new `_best.md` and the calibration table. Compare against the baseline (UseBreakeven OFF = current MDMG OOS: net PnL −985, PF 0.87). Success criterion: the breakeven-ON combo's OOS net PnL and PF beat the OFF baseline at a comparable trade count, with the give-back losers (table in the analysis report) now exiting near breakeven instead of at the ATR stop.

- [ ] **Step 4: Report the verdict (no commit)**

Summarize: did BE improve MDMG OOS? If yes, the hardcoded `UseBreakeven:1, BreakevenArmATR:<best arm>` from Task 3 stands (update the arm value in `mdmg.go` if the winner differs from 1.0). If no improvement, report it honestly and propose reverting MDMG to `UseBreakeven:0` — the param stays in the codebase (off) for future tickers.

---

## Self-Review

**Spec coverage:**
- New params `UseBreakeven`/`BreakevenArmATR` → Task 1. ✓
- Arm via `MaxFavorablePrice`, no engine change → Task 2 (reads `in.pos`). ✓
- Triple guard (`UseBreakeven==1 && EntryATR>0 && BreakevenArmATR>0`) → Task 2 case + `TestBreakevenInertWhenEntryATRZero`/`TestBreakevenInertWhenArmZero`. ✓
- Precedence `OB → RSI50 → BE → middle → EMAX` → Task 2 insertion point + `TestExitPrecedenceBreakevenOverMiddle`/`TestExitPrecedenceRSI50OverBreakeven`. ✓
- ATR availability widening (buildInput + Lookback) → Task 1 + its 2 tests. ✓
- close-fill nuance → documented in Task 2 Step 4 comment + Task 3 doc. ✓
- MDMG enable + winner + grids → Task 3. ✓
- All-ticker propagation (convention) → Task 4. ✓
- A/B verification → Task 5. ✓
- All 9 spec tests present (2 in Task 1, 7 in Task 2). ✓

**Placeholder scan:** No TBD/TODO; every code step shows full code; commands have expected output. ✓

**Type consistency:** `UseBreakeven int`, `BreakevenArmATR float64`, reason string `"BE"`, fields `MaxFavorablePrice`/`PurchasePrice`/`EntryATR` (from `strategy.Position`) used identically across Tasks 1-3. Grid phase name `"breakeven"` consistent. ✓
