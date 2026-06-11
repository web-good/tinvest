# Verbose Exit Reason in Backtest Report — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a descriptive "Причина выхода" column to the backtest trade journal (markdown + CSV), populated by the momentum strategy at exit time, symmetric with the existing "Причина входа".

**Architecture:** Thread a new `Signal.ExitReason` (full text) through the same path the entry rationale uses: the momentum `manage()` sets it per exit case → engine passes it to `portfolio.close` → `Trade.ExitReason` → report renderers. The short code column ("Причина") stays. Strategies that don't set it (levels/scalping) render an empty cell.

**Tech Stack:** Go 1.25; backtest domain (`internal/domain/backtest`), shared signal model (`internal/service/trading_strategy/scalping/model`), momentum core.

**Spec:** `docs/superpowers/specs/2026-06-11-backtest-exit-reason-report-design.md`

---

## File Structure

- **Modify** `internal/service/trading_strategy/scalping/model/signal.go` — add `ExitReason string` to `Signal`.
- **Modify** `internal/domain/backtest/types.go` — add `ExitReason string` to `Trade`.
- **Modify** `internal/domain/backtest/portfolio.go` — `close` takes `exitReason`, sets it on the `Trade`.
- **Modify** `internal/domain/backtest/engine.go` — both `close` call sites pass `sig.ExitReason`.
- **Modify** `internal/domain/backtest/report.go` — markdown column + CSV field.
- **Modify** `internal/domain/backtest/report_test.go` — assert new column/field; fix the CSV header suffix assertion.
- **Modify** `internal/service/trading_strategy/momentum/strategy/core/core.go` (`manage`) + `core_test.go` — populate `sig.ExitReason` per exit + assert it.

Two tasks: Task 1 threads + renders the field (testable with a hand-built `Trade`); Task 2 makes the momentum strategy populate it.

---

## Task 1: Thread `ExitReason` and render it in the report

**Files:**
- Modify: `internal/service/trading_strategy/scalping/model/signal.go`
- Modify: `internal/domain/backtest/types.go`
- Modify: `internal/domain/backtest/portfolio.go`
- Modify: `internal/domain/backtest/engine.go`
- Modify: `internal/domain/backtest/report.go`
- Test: `internal/domain/backtest/report_test.go`

- [ ] **Step 1: Write/adjust the failing report tests**

In `report_test.go`, add a markdown test and extend the CSV test. Add this new test:

```go
func TestRenderMarkdownHasExitReason(t *testing.T) {
	m := Metrics{TotalTrades: 1, Wins: 1, WinRate: 1.0, ProfitFactor: 2.0, NetPnL: 100}
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
		EntryReason: "Тренд↑ вход", ExitReason: "TP: high 110.0000 ≥ цель 110.0000 (2R)",
	}}
	out := RenderMarkdown(sampleMeta(), m, trades, eqCurve([]float64{100000, 100100}))
	if !strings.Contains(out, "Причина выхода") {
		t.Fatalf("markdown missing 'Причина выхода' header: %q", out)
	}
	if !strings.Contains(out, "TP: high 110.0000 ≥ цель 110.0000 (2R)") {
		t.Fatalf("markdown missing exit reason text: %q", out)
	}
}
```

Then update the existing `TestRenderTradesCSVHeaderAndRow`: set an `ExitReason` on the trade and fix the header-suffix assertion. Replace the trade literal and the suffix check:

Replace:
```go
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
		SupportLevel: 99, ResistanceLevel: 112, ATR: 1.25,
	}}
```
with:
```go
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
		SupportLevel: 99, ResistanceLevel: 112, ATR: 1.25,
		ExitReason: "TP: high 110.0000 ≥ цель 110.0000 (2R)",
	}}
```
Replace:
```go
	if !strings.HasSuffix(lines[0], "support_level,resistance_level,atr,entry_reason") {
		t.Fatalf("header missing new columns: %q", lines[0])
	}
```
with:
```go
	if !strings.HasSuffix(lines[0], "support_level,resistance_level,atr,entry_reason,exit_reason") {
		t.Fatalf("header missing new columns: %q", lines[0])
	}
	if !strings.Contains(lines[1], "цель") {
		t.Fatalf("row missing exit reason: %q", lines[1])
	}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/backtest/ -run 'TestRenderMarkdownHasExitReason|TestRenderTradesCSVHeaderAndRow' -v`
Expected: FAIL — the package will not compile yet (`Trade` has no `ExitReason` field). That is the expected red state.

- [ ] **Step 3: Add `ExitReason` to the `Signal` model**

In `internal/service/trading_strategy/scalping/model/signal.go`, in the `Signal` struct, after the `EntryReason` field add:

```go
	ExitReason     string  // human-readable exit rationale (set on Sell); empty for buys
```

- [ ] **Step 4: Add `ExitReason` to `Trade`**

In `internal/domain/backtest/types.go`, in the `Trade` struct, after the `EntryReason` field add:

```go
	ExitReason      string  // human-readable exit rationale captured at exit; empty when n/a
```

- [ ] **Step 5: Thread `exitReason` through `portfolio.close`**

In `internal/domain/backtest/portfolio.go`, change the signature:
```go
func (p *portfolio) close(price float64, t time.Time, reason string) Trade {
```
to:
```go
func (p *portfolio) close(price float64, t time.Time, reason, exitReason string) Trade {
```
and in the `Trade{...}` literal it builds, after `EntryReason: p.entryReason,` add:
```go
		ExitReason:      exitReason,
```

- [ ] **Step 6: Update the two `close` call sites in the engine**

In `internal/domain/backtest/engine.go`:
- In `Run`, change `res.Trades = append(res.Trades, p.close(exitPrice, c.Time, sig.Reason))` to:
```go
				res.Trades = append(res.Trades, p.close(exitPrice, c.Time, sig.Reason, sig.ExitReason))
```
- In `Trace`, change `p.close(exitPrice, c.Time, sig.Reason)` to:
```go
				p.close(exitPrice, c.Time, sig.Reason, sig.ExitReason)
```

- [ ] **Step 7: Render the column in markdown and CSV**

In `internal/domain/backtest/report.go`, in `RenderMarkdown`, change the trade-journal header line:
```go
	b.WriteString("\n## Журнал сделок\n\n| № | Вход | Цена входа | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
```
to:
```go
	b.WriteString("\n## Журнал сделок\n\n| № | Вход | Цена входа | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
```
and change the per-row `Fprintf` (add a trailing `%s` column and `t.ExitReason`):
```go
		fmt.Fprintf(&b, "| %d | %s | %.4f | %s | %.4f | %s | %d | %.2f | %.2f%% | %.4f | %.4f | %.4f | %s |\n",
			i+1, t.EntryTime.Format(tsLayout), t.EntryPrice, t.ExitTime.Format(tsLayout),
			t.ExitPrice, t.Reason, t.BarsHeld, t.PnL, t.PnLPct*100,
			t.SupportLevel, t.ResistanceLevel, t.ATR, t.EntryReason)
```
to:
```go
		fmt.Fprintf(&b, "| %d | %s | %.4f | %s | %.4f | %s | %d | %.2f | %.2f%% | %.4f | %.4f | %.4f | %s | %s |\n",
			i+1, t.EntryTime.Format(tsLayout), t.EntryPrice, t.ExitTime.Format(tsLayout),
			t.ExitPrice, t.Reason, t.BarsHeld, t.PnL, t.PnLPct*100,
			t.SupportLevel, t.ResistanceLevel, t.ATR, t.EntryReason, t.ExitReason)
```
In `RenderTradesCSV`, change the header line:
```go
	b.WriteString("idx,entry_time,entry_price,exit_time,exit_price,qty,reason,pnl,pnl_pct,bars_held,support_level,resistance_level,atr,entry_reason\n")
```
to:
```go
	b.WriteString("idx,entry_time,entry_price,exit_time,exit_price,qty,reason,pnl,pnl_pct,bars_held,support_level,resistance_level,atr,entry_reason,exit_reason\n")
```
and the per-row `Fprintf` (add a trailing `,%s` and `csvField(t.ExitReason)`):
```go
		fmt.Fprintf(&b, "%d,%s,%.6f,%s,%.6f,%d,%s,%.6f,%.6f,%d,%.6f,%.6f,%.6f,%s\n",
			i+1, t.EntryTime.UTC().Format(time.RFC3339), t.EntryPrice,
			t.ExitTime.UTC().Format(time.RFC3339), t.ExitPrice, t.Quantity,
			t.Reason, t.PnL, t.PnLPct, t.BarsHeld,
			t.SupportLevel, t.ResistanceLevel, t.ATR, csvField(t.EntryReason))
```
to:
```go
		fmt.Fprintf(&b, "%d,%s,%.6f,%s,%.6f,%d,%s,%.6f,%.6f,%d,%.6f,%.6f,%.6f,%s,%s\n",
			i+1, t.EntryTime.UTC().Format(time.RFC3339), t.EntryPrice,
			t.ExitTime.UTC().Format(time.RFC3339), t.ExitPrice, t.Quantity,
			t.Reason, t.PnL, t.PnLPct, t.BarsHeld,
			t.SupportLevel, t.ResistanceLevel, t.ATR, csvField(t.EntryReason), csvField(t.ExitReason))
```

- [ ] **Step 8: Run the backtest domain tests + full build**

Run: `go test ./internal/domain/backtest/ -run 'TestRenderMarkdownHasExitReason|TestRenderTradesCSVHeaderAndRow|TestRenderMarkdownHasSections' -v`
Expected: all PASS.
Run: `go test ./internal/domain/backtest/`
Expected: `ok` (all existing tests still pass).
Run: `go build ./...`
Expected: clean (confirms the `close` signature change compiles everywhere, incl. engine `Run`/`Trace`).

- [ ] **Step 9: gofmt/vet + commit**

Run: `gofmt -l internal/domain/backtest/report.go internal/domain/backtest/types.go internal/domain/backtest/portfolio.go internal/domain/backtest/engine.go internal/domain/backtest/report_test.go internal/service/trading_strategy/scalping/model/signal.go` (expect empty) and `go vet ./internal/domain/backtest/ ./internal/service/trading_strategy/scalping/model/`.

```bash
git add internal/service/trading_strategy/scalping/model/signal.go internal/domain/backtest/types.go internal/domain/backtest/portfolio.go internal/domain/backtest/engine.go internal/domain/backtest/report.go internal/domain/backtest/report_test.go
git commit -m "feat(backtest): thread Signal.ExitReason into Trade + render 'Причина выхода'

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Populate the verbose exit reason in momentum `manage()`

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Write failing tests asserting `sig.ExitReason` per exit**

Add to `core_test.go` (reuses helpers `inPositionMD`, `inPositionMDWithCloses`, `risingThenDropCloses`, and the `indicators` import, all already present):

```go
func TestExitReasonSL(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(94, 101, 100, pos)) // barLow 94 <= SL 95
	if sig.Reason != "SL" || !strings.Contains(sig.ExitReason, "стоп") {
		t.Fatalf("reason=%q exitReason=%q want SL with 'стоп'", sig.Reason, sig.ExitReason)
	}
}

func TestExitReasonTP(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(100, 111, 100, pos)) // barHigh 111 >= TP 110
	if sig.Reason != "TP" || !strings.Contains(sig.ExitReason, "цель") {
		t.Fatalf("reason=%q exitReason=%q want TP with 'цель'", sig.Reason, sig.ExitReason)
	}
}

func TestExitReasonTrail(t *testing.T) {
	p := defaultParams()
	p.UseTrail = 1
	p.TrailArmATR = 0
	p.TakeProfitRR = 0 // isolate trail
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 120}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMD(117, 121, 120, pos)) // barLow 117 <= chandelier ~117.5
	if sig.Reason != "TRAIL" || !strings.Contains(sig.ExitReason, "шанделье") {
		t.Fatalf("reason=%q exitReason=%q want TRAIL with 'шанделье'", sig.Reason, sig.ExitReason)
	}
}

func TestExitReasonMACD(t *testing.T) {
	closes := risingThenDropCloses(60, 100, 0.5, 6)
	p := defaultParams()
	p.UseMACDExit = 1
	p.TakeProfitRR = 0
	p.UseTrail = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMDWithCloses(closes, pos))
	if sig.Reason != "MACD" || !strings.Contains(sig.ExitReason, "кросс") {
		t.Fatalf("reason=%q exitReason=%q want MACD with 'кросс'", sig.Reason, sig.ExitReason)
	}
}

func TestExitReasonRSI(t *testing.T) {
	const period = 14
	closes := risingThenDropCloses(40, 100, 1.0, 4)
	r := indicators.RSISeries(closes, period)
	prev, now := r[len(r)-2], r[len(r)-1]
	if !(prev > now) {
		t.Fatalf("test setup: need prev RSI > now RSI, got prev=%v now=%v", prev, now)
	}
	p := defaultParams()
	p.RSIPeriod = period
	p.RSIOverbought = (prev + now) / 2
	p.TakeProfitRR = 0
	p.UseTrail = 0
	p.UseMACDExit = 0
	pos := &strategy.Position{PurchasePrice: closes[0], StopLoss: 1, EntryATR: 1, MaxFavorablePrice: 1000}
	s := NewWithParams("TEST", p)
	sig := s.Decide(inPositionMDWithCloses(closes, pos))
	if sig.Reason != "RSI" || !strings.Contains(sig.ExitReason, "пересёк границу") {
		t.Fatalf("reason=%q exitReason=%q want RSI with 'пересёк границу'", sig.Reason, sig.ExitReason)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestExitReason' -v`
Expected: all five FAIL — `sig.ExitReason` is empty (the `Contains` checks fail). The package compiles because `Signal.ExitReason` exists from Task 1.

- [ ] **Step 3: Set `sig.ExitReason` in each `manage` case**

In `core.go`, in `manage`, replace the `switch` block:
```go
	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
	case s.p.UseTrail == 1 && trailArmed && in.barLow <= chandelier:
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
	case s.p.TakeProfitRR > 0 && in.barHigh >= tp:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
	case s.p.UseMACDExit == 1 && in.macdCrossDown:
		sig.Kind, sig.Reason = model.SignalSell, "MACD"
	case s.p.RSIPeriod > 0 && in.rsiPrev > s.p.RSIOverbought && in.rsiNow <= s.p.RSIOverbought:
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
	}
```
with:
```go
	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (зафиксирован на входе)", in.barLow, hardSL)
	case s.p.UseTrail == 1 && trailArmed && in.barLow <= chandelier:
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
		sig.ExitReason = fmt.Sprintf("TRAIL: low %.4f ≤ шанделье %.4f (recentHigh %.4f − %.2g×ATR %.4f)",
			in.barLow, chandelier, in.recentHigh, s.p.TrailMult, in.atr)
	case s.p.TakeProfitRR > 0 && in.barHigh >= tp:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цель %.4f (%.2gR)", in.barHigh, tp, s.p.TakeProfitRR)
	case s.p.UseMACDExit == 1 && in.macdCrossDown:
		sig.Kind, sig.Reason = model.SignalSell, "MACD"
		sig.ExitReason = fmt.Sprintf("MACD: медвежий кросс сигнальной линии (MACD=%.4f)", in.macdNow)
	case s.p.RSIPeriod > 0 && in.rsiPrev > s.p.RSIOverbought && in.rsiNow <= s.p.RSIOverbought:
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: %.2f → %.2f, пересёк границу %.2g сверху вниз",
			in.rsiPrev, in.rsiNow, s.p.RSIOverbought)
	}
```
(`fmt` is already imported in `core.go`.)

- [ ] **Step 4: Run the new tests + full package**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestExitReason' -v`
Expected: all five PASS.
Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/`
Expected: `ok` (all existing tests still pass).

- [ ] **Step 5: gofmt/vet + commit**

Run: `gofmt -l internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go` (expect empty) and `go vet ./internal/service/trading_strategy/momentum/strategy/core/`.

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "feat(momentum): set verbose Signal.ExitReason per exit trigger

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Final verification (after both tasks)

- [ ] `go build ./...` — clean.
- [ ] `go test ./internal/domain/backtest/ ./internal/service/trading_strategy/momentum/...` — all `ok`.
- [ ] `gofmt -l internal/domain/backtest/ internal/service/trading_strategy/momentum/ internal/service/trading_strategy/scalping/model/` — empty.
- [ ] Optional spot-check: a fresh momentum backtest report shows the new "Причина выхода" column with text like `RSI: 72.30 → 68.50, пересёк границу 70 сверху вниз` next to the code column.

## Notes / out of scope

- Levels/scalping strategies leave `ExitReason` empty (empty cell). Enriching their exits is a follow-up.
- The momentum `manage` is the only producer of the verbose text; `Explain`/`Trace` are unaffected (they do not build trade records).
