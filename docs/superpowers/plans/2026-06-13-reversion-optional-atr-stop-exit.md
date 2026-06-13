# Reversion Optional ATR-Stop Exit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the reversion strategy's middle exit a flag-selected choice between the existing RSI-oversold-breakdown (RSIOS) and a new daily-ATR stop (ATRSL: price has fallen below entry by `StopATRMult × EntryATR`).

**Architecture:** Add three int/float Params (`UseATRStop`, `ATRPeriod`, `StopATRMult`) so grid calibration can sweep them. Freeze the daily ATR at entry (stamp `sig.ATR` on Buy → engine persists `Position.EntryATR`) and read it in `manage()`. The ATRSL branch replaces RSIOS only when `UseATRStop==1`; RSI50 and EMAX are untouched. Close-fill with `Reason="ATRSL"`. A guard (`EntryATR > 0 && StopATRMult > 0`) keeps the branch inert in live trading where entry state is not persisted.

**Tech Stack:** Go 1.25, package `internal/service/trading_strategy/reversion/strategy/core`, `pkg/indicators.ATR`, backtest engine in `internal/domain/backtest`.

---

### Task 1: Params, ATR plumbing, Lookback

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go`
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `core_test.go`:

```go
func TestEntryStampsEntryATR(t *testing.T) {
	p := defaultParams()
	p.UseATRStop = 1
	p.ATRPeriod = 14
	p.StopATRMult = 1.0
	s := NewWithParams("TEST", p)

	in := passingInput()
	in.atr = 2.5
	sig := s.decide(in)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got kind=%v", sig.Kind)
	}
	if sig.ATR != 2.5 {
		t.Fatalf("entry must stamp sig.ATR for freeze; want 2.5 got %v", sig.ATR)
	}
}

func TestLookbackIncludesATRWhenStopOn(t *testing.T) {
	p := defaultParams()
	p.FastEMA, p.SlowEMA = 50, 200
	p.RSIPeriod, p.StochKPeriod, p.StochDSmooth = 14, 14, 3
	p.ATRPeriod = 300 // dominates SlowEMA when the stop is on

	p.UseATRStop = 1
	if got := NewWithParams("T", p).Lookback(); got != 305 {
		t.Fatalf("UseATRStop=1: want ATRPeriod+1+5=305, got %d", got)
	}

	p.UseATRStop = 0
	if got := NewWithParams("T", p).Lookback(); got != 205 {
		t.Fatalf("UseATRStop=0: ATRPeriod ignored, want SlowEMA+5=205, got %d", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'TestEntryStampsEntryATR|TestLookbackIncludesATRWhenStopOn' -v`
Expected: compile error — `Params` has no field `UseATRStop`/`ATRPeriod`/`StopATRMult`, `decideInput` has no field `atr`.

- [ ] **Step 3: Add the three Params fields**

In `core.go`, inside `type Params struct`, after the `StochOversold` line, add:

```go
	UseATRStop  int     // 0 = RSIOS exit (RSI breaks oversold zone down); 1 = ATRSL exit (price below entry by the daily ATR)
	ATRPeriod   int     // daily ATR length; consulted only when UseATRStop=1
	StopATRMult float64 // ATRSL distance: stop = PurchasePrice - StopATRMult*EntryATR (default 1.0)
```

- [ ] **Step 4: Add the `atr` field to `decideInput`**

In `core.go`, inside `type decideInput struct`, after the `stochOK bool` line, add:

```go
	atr         float64 // daily ATR over the window (0 when ATRPeriod<=0); stamped onto sig.ATR at entry to freeze EntryATR
```

- [ ] **Step 5: Compute the ATR in `buildInput`**

In `core.go` `buildInput`, after the Stochastic block (the `if s.p.StochKPeriod > 0 ...` block) and before the `return decideInput{` line, add:

```go
	var atr float64
	if s.p.ATRPeriod > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	}
```

Then add `atr: atr,` to the returned `decideInput{...}` literal (e.g. right after the `stochOK: stochOK,` line).

- [ ] **Step 6: Stamp `sig.ATR` at entry**

In `core.go` `decide`, in the Buy block, after `sig.EntryReason = s.entryReason(in)` add:

```go
	sig.ATR = in.atr
```

- [ ] **Step 7: Extend `Lookback` to cover the ATR window**

Replace the body of `Lookback` in `core.go` with:

```go
func (s *Strategy) Lookback() int {
	m := s.p.SlowEMA
	cands := []int{
		s.p.FastEMA,
		s.p.RSIPeriod + 1,
		s.p.StochKPeriod + s.p.StochDSmooth + 1,
	}
	if s.p.UseATRStop == 1 {
		cands = append(cands, s.p.ATRPeriod+1)
	}
	for _, c := range cands {
		if c > m {
			m = c
		}
	}
	return m + 5
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'TestEntryStampsEntryATR|TestLookbackIncludesATRWhenStopOn' -v`
Expected: PASS for both.

- [ ] **Step 9: Run the full core package to confirm no regression**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/`
Expected: ok (all existing tests still pass — `UseATRStop` defaults to 0, so behavior is unchanged).

- [ ] **Step 10: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): plumb daily ATR freeze and ATR-stop params

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: ATRSL exit branch in `manage()`

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go:229-246` (the `manage` method)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `core_test.go`:

```go
// atrStopParams: ATR-stop branch on, multiplier 1.0, RSI oversold 30.
func atrStopParams() Params {
	p := defaultParams()
	p.UseATRStop = 1
	p.ATRPeriod = 14
	p.StopATRMult = 1.0
	p.RSIOversold = 30
	return p
}

func TestExitATRStopFires(t *testing.T) {
	s := NewWithParams("T", atrStopParams())
	in := openInput() // neutral RSI/EMA: no RSI50, no EMAX
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 94 // <= 100 - 1.0*5 = 95
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "ATRSL" {
		t.Fatalf("price below entry-ATR: want ATRSL sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestNoATRStopAboveThreshold(t *testing.T) {
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 96 // > 95 threshold
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("price above threshold: should hold, got sell %q", sig.Reason)
	}
}

func TestATRStopSkippedWhenEntryATRZero(t *testing.T) {
	// Live-trading guard: EntryATR is not persisted (0), so the stop must never fire,
	// even though price (1) is far below PurchasePrice (100).
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 0}
	in.price = 1
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "ATRSL" {
		t.Fatalf("EntryATR=0 must skip ATRSL (live-trading guard)")
	}
}

func TestRSIOSInertWhenATRStopOn(t *testing.T) {
	// With the ATR stop on, the RSIOS branch is disabled: an RSI break of the oversold
	// zone must NOT produce a sell (price is above the ATR threshold).
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 99 // above the 95 ATR threshold
	in.rsiOK = true
	in.rsiPrev, in.rsiNow = 32, 28 // would be an RSIOS down-cross of 30
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("UseATRStop=1: RSIOS must be inert, got sell %q", sig.Reason)
	}
}

func TestExitPrecedenceRSI50OverATR(t *testing.T) {
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 94               // ATRSL would fire
	in.rsiPrev, in.rsiNow = 55, 45 // RSI50 also fires
	if sig := s.decide(in); sig.Reason != "RSI50" {
		t.Fatalf("RSI50 must win over ATRSL, got %q", sig.Reason)
	}
}

func TestExitPrecedenceATROverEMA(t *testing.T) {
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 94                  // ATRSL fires
	in.rsiPrev, in.rsiNow = 60, 58 // no RSI50
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // EMAX also fires
	if sig := s.decide(in); sig.Reason != "ATRSL" {
		t.Fatalf("ATRSL must win over EMAX, got %q", sig.Reason)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'ATRStop|ATROver|RSI50OverATR|RSIOSInert' -v`
Expected: FAIL — no ATRSL branch yet, so `TestExitATRStopFires`/`TestExitPrecedenceATROverEMA` get no sell or the wrong reason, and `TestRSIOSInertWhenATRStopOn` produces an RSIOS sell.

- [ ] **Step 3: Add the flag gate and the ATRSL branch in `manage`**

In `core.go`, replace the `case in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold):` block with the gated RSIOS case plus the new ATRSL case. The full updated `switch` in `manage` becomes:

```go
	switch {
	case in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, rsiExitLevel):
		sig.Kind, sig.Reason = model.SignalSell, "RSI50"
		sig.ExitReason = fmt.Sprintf("RSI50: RSI %.2f→%.2f пересёк 50 сверху вниз", in.rsiPrev, in.rsiNow)
	case s.p.UseATRStop == 0 && in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold):
		sig.Kind, sig.Reason = model.SignalSell, "RSIOS"
		sig.ExitReason = fmt.Sprintf("RSIOS: RSI %.2f→%.2f пробил зону перепроданности %.0f сверху вниз",
			in.rsiPrev, in.rsiNow, s.p.RSIOversold)
	case s.p.UseATRStop == 1 && in.pos.EntryATR > 0 && s.p.StopATRMult > 0 &&
		in.price <= in.pos.PurchasePrice-s.p.StopATRMult*in.pos.EntryATR:
		stop := in.pos.PurchasePrice - s.p.StopATRMult*in.pos.EntryATR
		sig.Kind, sig.Reason = model.SignalSell, "ATRSL"
		sig.ExitReason = fmt.Sprintf("ATRSL: цена %.4f ≤ вход %.4f − %.2g×ATR %.4f (порог %.4f)",
			in.price, in.pos.PurchasePrice, s.p.StopATRMult, in.pos.EntryATR, stop)
	case in.emaOK && crossDown(in.emaFastPrev-in.emaSlowPrev, in.emaFast-in.emaSlow, 0):
		sig.Kind, sig.Reason = model.SignalSell, "EMAX"
		sig.ExitReason = fmt.Sprintf("EMAX: FastEMA%d %.4f ушла под SlowEMA%d %.4f (медвежий кросс)",
			s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow)
	}
```

- [ ] **Step 4: Update the `manage` doc comment**

In `core.go`, replace the `manage` method's doc comment (lines describing the three exit signals) with:

```go
// manage handles an open long. There is no protective price stop other than the optional
// ATR stop below. It exits on one of three signals, evaluated in precedence order (all
// fills at close):
//   - RSI50: RSI crosses the 50 midline downward — primary momentum fade.
//   - middle branch, selected by UseATRStop:
//       UseATRStop==0 -> RSIOS: RSI breaks back down through the oversold zone from above
//         (failed-bounce breakdown); fires when RSI was at/above RSIOversold last bar and
//         is now below it.
//       UseATRStop==1 -> ATRSL: price has fallen to/below PurchasePrice - StopATRMult*EntryATR,
//         where EntryATR is the daily ATR frozen at entry. Guarded by EntryATR>0 and
//         StopATRMult>0 so it stays inert in live trading (EntryATR not persisted) and on
//         a misconfigured zero multiplier.
//   - EMAX: FastEMA drops below SlowEMA (bearish EMA cross) — regime-break backstop.
//
// When multiple signals fire on the same bar the first in the list wins; the fill price
// (close) is identical either way.
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'ATRStop|ATROver|RSI50OverATR|RSIOSInert' -v`
Expected: PASS for all six.

- [ ] **Step 6: Run the full core package**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/`
Expected: ok — existing RSIOS/RSI50/EMAX/precedence tests still pass (they run with `UseATRStop=0`).

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): add optional daily ATR stop exit (UseATRStop)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Per-ticker defaults and calibration grids

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/{afks,rusal,gazp,ydex,mdmg,nvtk,plzl,sber}/<ticker>.go`
- Modify: `data/params/{afks,rual,gazp,ydex,mdmg,nvtk,plzl,sber}/reversion_grid.json`

- [ ] **Step 1: Add the new fields to every per-ticker `DefaultParams`**

In each of the 8 ticker files, add `UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,` to the returned `core.Params{...}` literal (new line after the `StochKPeriod ...` line). `UseATRStop: 0` preserves current behavior. Example for `afks/afks.go`:

```go
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 10, SlowEMA: 200,
		RSIPeriod: 5, RSIOversold: 30,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,
	}
}
```

Apply the identical extra line to `rusal`, `gazp`, `ydex`, `mdmg`, `nvtk`, `plzl`, `sber` (keep each file's existing UseTrend/EMA/RSI/Stoch values untouched — only append the three new fields).

- [ ] **Step 2: Add the ATR-stop sweep to each calibration grid**

In each of the 8 `reversion_grid.json` files, add three keys to the `"grid"` object of the `entry` phase, alongside the existing keys:

```json
        "UseATRStop": [0, 1],
        "ATRPeriod": [14],
        "StopATRMult": [1.0, 1.5, 2.0]
```

Do not remove or change existing keys. The trailing comma rules of JSON apply — ensure the previous key's line ends with a comma and the object stays valid.

- [ ] **Step 3: Validate every grid file is valid JSON**

Run: `for f in data/params/*/reversion_grid.json; do python3 -m json.tool "$f" >/dev/null && echo "OK $f" || echo "BAD $f"; done`
Expected: `OK` for all 8 files.

- [ ] **Step 4: Build the whole module to confirm the per-ticker literals compile**

Run: `go build ./...`
Expected: no output (success).

- [ ] **Step 5: Run the reversion strategy packages' tests**

Run: `go test ./internal/service/trading_strategy/reversion/...`
Expected: ok for all reversion packages.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy data/params/*/reversion_grid.json
git commit -m "feat(reversion): default ATR-stop params and grid sweep per ticker

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: Documentation

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go:1-10` (package doc)
- Modify: `docs/reversion/strategy.md:24-40` (Exit section)

- [ ] **Step 1: Update the package doc comment**

In `core.go`, replace the existing sentence describing the three exit signals (the part starting "It exits an open long on one of three signals:" through "as a regime-break backstop.") with:

```
// It exits an open long on one of three signals: RSI crossing the 50 line downward
// (primary momentum fade); a middle exit selected by the UseATRStop flag — either RSI
// breaking back down through the oversold zone (RSIOS, failed bounce) or price falling
// below the ATR stop PurchasePrice − StopATRMult×EntryATR with EntryATR frozen at entry
// (ATRSL); and a bearish EMA cross (FastEMA below SlowEMA) as a regime-break backstop.
```

- [ ] **Step 2: Update the Exit section of `docs/reversion/strategy.md`**

Replace the numbered exit list (items 1–3 and the surrounding "There is no protective price stop." sentence) with:

```markdown
The middle exit is selected by the `UseATRStop` flag. An open long exits on one of three
signals, filled at the bar close, in this precedence order:

1. **RSI50:** RSI crosses the 50 line from above (`prev ≥ 50`, `now < 50`) — the primary
   momentum-fade exit.
2. **Middle exit (flag-selected):**
   - `UseATRStop = 0` → **RSIOS:** RSI crosses `RSIOversold` from above
     (`prev ≥ RSIOversold`, `now < RSIOversold`) — the failed-bounce exit. It cannot fire
     on the bar right after entry, where RSI is already below the zone.
   - `UseATRStop = 1` → **ATRSL:** price falls to/below `PurchasePrice − StopATRMult ×
     EntryATR`, where `EntryATR` is the daily ATR (length `ATRPeriod`) frozen at entry.
     Guarded by `EntryATR > 0` and `StopATRMult > 0`, so it stays inert in live trading
     (entry ATR is not persisted) and on a zero multiplier.
3. **EMAX:** bearish EMA cross — `EMA(FastEMA)` drops below `EMA(SlowEMA)`. A slow
   regime-break backstop; reuses the same EMAs as the trend filter.

If several fire on the same bar the earliest in this order is reported; the fill (close)
is identical either way.
```

- [ ] **Step 3: Build to confirm the doc comment did not break the package**

Run: `go build ./internal/service/trading_strategy/reversion/...`
Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go docs/reversion/strategy.md
git commit -m "docs(reversion): document UseATRStop flag and ATRSL exit

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Verification

- [ ] `go test ./internal/service/trading_strategy/reversion/...` — all green.
- [ ] `go build ./...` — clean.
- [ ] Sanity backtest with the stop on (manual, optional):
  `go run ./cmd/backtest -ticker RUAL -strategy reversion -interval Day1 -months 12 -out ./reports/RUAL`
  then a calibration run that exercises the new grid keys:
  `go run ./cmd/backtest -ticker RUAL -strategy reversion -interval Day1 -calibrate data/params/rual/reversion_grid.json -out ./reports/RUAL -months 24 -test-months 6`
  Expected: report shows `ATRSL` exits among trades for `UseATRStop=1` survivors.
