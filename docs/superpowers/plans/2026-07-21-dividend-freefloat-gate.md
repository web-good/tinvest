# Free-Float Liquidity Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a second, independent liquidity floor (free-float fraction) to the dividend screener gate so untradeable thin-float names (Удмуртнефть, float 0 %) stop ranking, and bump the bot report to top-25.

**Architecture:** The screener's pure ranking core (`internal/service/screener/dividend/rank`) already has a market-cap liquidity floor in `gate()`. We add a parallel `MinFreeFloat` threshold and a new gate reason. The `FreeFloat` field is already mapped into `model.Fundamentals` (committed foundation `b657995`). The report-size change is a one-constant edit in the service package. Verification uses the already-built `cmd/divscreen` CLI against the live API.

**Tech Stack:** Go 1.25, standard `testing`, `./bin/mage ci` (golangci-lint v2 + `go test -race` + mockery drift).

## Global Constraints

- Module path prefix: `tinvest/...`. Package under work: `tinvest/internal/service/screener/dividend` and its `rank` subpackage.
- Units: `FreeFloat` is a fraction in `[0, 1]` (e.g. `0.11` = 11 %). Gate comparison is strict `<` (a name exactly at the floor passes), matching the existing market-cap gate style.
- `FreeFloat == 0` is treated as illiquid (gated), consistent with how `MarketCapitalization == 0` is already gated — proto3 omitempty makes "no data" and "real zero" indistinguishable, and the safe choice here is to gate.
- Gate order after this change: `нет дивиденда` → `низкая ликвидность` (market-cap) → `тонкий free-float` → `yield trap` → `долг > порога` → `payout > порога`.
- Do NOT touch the market-cap floor (`MinMarketCap`) or any pillar/composite math.
- Build check for this repo: `go build ./internal/... ./pkg/... ./cmd/...` (never `go build ./...` — the `magefiles` package has no `main`).
- Commit messages end with the `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>` trailer.

---

### Task 1: Free-float gate in the ranking core

**Files:**
- Modify: `internal/service/screener/dividend/rank/config.go` (add `MinFreeFloat` field + seed)
- Modify: `internal/service/screener/dividend/rank/rank.go` (add reason const + gate check)
- Test: `internal/service/screener/dividend/rank/rank_test.go` (new test + fixture updates)
- Test: `internal/service/screener/dividend/service_test.go` (fixture updates)

**Interfaces:**
- Consumes: `model.Fundamentals.FreeFloat float64` (already present, commit `b657995`).
- Produces: `rank.Config.MinFreeFloat float64`; new unexported const `reasonThinFloat = "тонкий free-float"`; `gate()` now returns `reasonThinFloat` when `FreeFloat < MinFreeFloat`.

- [ ] **Step 1: Write the failing test**

Add to `internal/service/screener/dividend/rank/rank_test.go`. Also add a package-level helper next to the existing `aboveCapFloor` var (top of file, after line 12):

```go
// aboveFloatFloor — free-float заведомо выше DefaultConfig().MinFreeFloat,
// чтобы фильтр тонкого флоата не мешал тестам про другое поведение.
// Читает актуальный сид, поэтому устойчива к калибровке порога.
var aboveFloatFloor = DefaultConfig().MinFreeFloat + 0.01
```

And the new test (append to the file):

```go
func TestRank_GateThinFloat(t *testing.T) {
	cfg := DefaultConfig()
	u := []*model.Fundamentals{
		// Платит дивиденд, капа выше порога, но free-float ниже порога → отсев.
		{AssetUID: "thin", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: aboveCapFloor, FreeFloat: cfg.MinFreeFloat - 0.001},
		// free-float ровно на пороге → проходит (сравнение строгое <).
		{AssetUID: "edge", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: aboveCapFloor, FreeFloat: cfg.MinFreeFloat},
		// free-float 0 (нет данных / нулевой) при валидной капе и yield → отсев.
		{AssetUID: "zero", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: aboveCapFloor, FreeFloat: 0},
	}
	got := byUID(Rank(u, cfg))
	if got["thin"].GateReason != reasonThinFloat {
		t.Fatalf("thin: GateReason = %q, want %q", got["thin"].GateReason, reasonThinFloat)
	}
	if got["zero"].GateReason != reasonThinFloat {
		t.Fatalf("zero: GateReason = %q, want %q", got["zero"].GateReason, reasonThinFloat)
	}
	if got["edge"].GateReason != "" {
		t.Fatalf("edge must survive, gated: %q", got["edge"].GateReason)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestRank_GateThinFloat -v`
Expected: FAIL — `reasonThinFloat` and `Config.MinFreeFloat` are undefined (compile error).

- [ ] **Step 3: Add the config field and seed**

In `internal/service/screener/dividend/rank/config.go`, add the field to the `Config` struct next to `MinMarketCap` (after the `MinMarketCap float64` line):

```go
	MinFreeFloat float64 // доля акций в свободном обращении ниже этого (в т.ч. 0 = нет данных) — отсев как тонкий флоат
```

And in `DefaultConfig()`, add after the `MinMarketCap: ...` line:

```go
		MinFreeFloat: 0.07, // live-калибровка 2026-07-21: разрыв 5–9% между неликвидом (UDMN 0, BANE/GCHE 3, LEAS 4, AKRN/SIBN 5) и легитимными (min 9%); 0.07 посередине
```

- [ ] **Step 4: Add the gate reason and check**

In `internal/service/screener/dividend/rank/rank.go`, add the reason constant to the existing `const (...)` block (after `reasonIlliquid`):

```go
	reasonThinFloat = "тонкий free-float"
```

In `gate()`, insert the free-float check **immediately after** the market-cap check (`if f.MarketCapitalization < cfg.MinMarketCap { ... }`) and **before** the yield-trap check:

```go
	if f.FreeFloat < cfg.MinFreeFloat {
		return reasonThinFloat, false
	}
```

Also update the `gate` doc comment: extend the "Жёсткие основания отсева по данным" sentence to mention thin free-float alongside low liquidity.

- [ ] **Step 5: Run the new test to verify it passes**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestRank_GateThinFloat -v`
Expected: PASS

- [ ] **Step 6: Fix existing fixtures broken by the new gate**

The new gate rejects any fixture with `FreeFloat == 0`. Add `FreeFloat: aboveFloatFloor` to every **survivor** fixture (the ones that must pass the gate) in both test files.

In `internal/service/screener/dividend/rank/rank_test.go`, add `FreeFloat: aboveFloatFloor` to the `model.Fundamentals` literals in these tests (each must keep surviving): `TestRank_GateHighLeverage` ("lev" — it is gated by leverage, but leverage is checked AFTER free-float, so it MUST have float above floor to reach the leverage gate), `TestRank_YieldTrap` ("trap" — yield-trap is checked after free-float, so add float), `TestRank_MissingFundamentalsNoLongerGated` ("nodata"), `TestRank_KeepsBankLikeDividendPayer` ("bank"), `TestRank_NeutralSustainabilityWhenPayoutMissing` ("nopayout"), `TestRank_OrdersSurvivorsByComposite` ("strong", "weak"), `TestRank_DegenerateGrowthPoolIsNeutral` ("a", "b"), and in `TestRank_GateIlliquid` add `FreeFloat: aboveFloatFloor` to the "big" fixture only (the "small"/"nocap" fixtures are gated earlier by market-cap, so their float is irrelevant — but adding it to "big" is required so it still survives).

Note on ordering: `reasonThinFloat` is checked before `yield trap` and `долг > порога`. Fixtures that assert those later reasons (`"lev"`, `"trap"`) must have `FreeFloat >= MinFreeFloat` so they are NOT short-circuited by the float gate first.

In `internal/service/screener/dividend/service_test.go`, add `FreeFloat: aboveFloatFloor` to all four fixtures in `fundUniverse()` (`a-strong`, `a-mid`, `a-weak`, `a-gated`) and to both fixtures in `TestRankBonus_SharedAssetCoversAllInstruments` (`a-shared`, `a-weak`). The `service_test.go` already defines its own `aboveCapFloor`; add an analogous helper near it (after line 16):

```go
var aboveFloatFloor = rank.DefaultConfig().MinFreeFloat + 0.01
```

- [ ] **Step 7: Run the full screener test suites to verify everything passes**

Run: `go test ./internal/service/screener/...`
Expected: `ok` for both `dividend` and `dividend/rank`.

- [ ] **Step 8: Build and commit**

Run: `go build ./internal/... ./pkg/... ./cmd/...`
Expected: no output (success).

```bash
git add internal/service/screener/dividend/rank/config.go internal/service/screener/dividend/rank/rank.go internal/service/screener/dividend/rank/rank_test.go internal/service/screener/dividend/service_test.go
git commit -m "$(cat <<'EOF'
feat(screener): gate thin free-float names as illiquid

Adds a second liquidity floor Config.MinFreeFloat (0.07) parallel to the
market-cap floor. gate() now returns "тонкий free-float" when FreeFloat is
below it (0 included, consistent with MarketCapitalization==0). Removes
untradeable thin-float payers (e.g. Удмуртнефть, float 0%) that passed the
market-cap floor on total cap alone.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Live verification and threshold confirmation

**Files:**
- Read-only: run `cmd/divscreen` against the live API (needs `T_BANK` in `./env/token.env` or `./env/local.env` — already present).
- Modify only if calibration requires: `internal/service/screener/dividend/rank/config.go` (`MinFreeFloat` seed).

**Interfaces:**
- Consumes: `rank.Config.MinFreeFloat` from Task 1; the `cmd/divscreen` CLI (`-top`, `-gated`, `-probe`).
- Produces: confirmed `MinFreeFloat` value (no interface change).

- [ ] **Step 1: Run the screener against the live API**

Run: `go run ./cmd/divscreen -top 0 -gated`
Expected: a ranked table plus a "Отсеяно воротами" section. Inspect the `Float%` column of survivors and the `тонкий free-float` gate bucket.

- [ ] **Step 2: Verify the thin-float names are gated**

Confirm the `Отсеяно воротами` section now contains a `тонкий free-float` line that includes **UDMN, OZON, BANE, GCHE** (and any name with float < 7 %). Confirm none of these appear in the ranked table.

Run (targeted probe): `go run ./cmd/divscreen -probe "UDMN,OZON,BANE,GCHE,MSRS,DOMRF,SNGSP"`
Expected: UDMN/OZON/BANE/GCHE show `ОТСЕЯН: тонкий free-float`; MSRS/DOMRF/SNGSP show `проходит ворота`.

- [ ] **Step 3: Verify the ranked top-1 is a tradeable name and curated-11 survive**

Confirm the ranked table's #1 is now a liquid name (expected **MSRS**, float 11 %, composite ~83 — the previous #2). Spot-check that the Golden X curated names present in the dividend universe still pass (none should be newly gated by free-float): probe a sample.

Run: `go run ./cmd/divscreen -probe "SNGSP,TATNP,TATN,ROSN,LKOH,PHOR,BSPB,TRNFP"`
Expected: all show `проходит ворота` (note TATNP float ≈100 %, SNGSP ≈73 %, all well above 7 %).

- [ ] **Step 4: Confirm or adjust the threshold**

If live data still shows the clean gap (survivors either < 7 % clustered near 0–5 %, or ≥ 9 %), keep `MinFreeFloat: 0.07` — no change, no commit. Only if the live gap has shifted (e.g. a legitimate name now sits at 6–7 %) adjust the seed in `config.go` to sit inside the new gap, re-run Step 1, then commit:

```bash
git add internal/service/screener/dividend/rank/config.go
git commit -m "$(cat <<'EOF'
chore(screener): retune MinFreeFloat to live free-float gap

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

If no change was needed, state that explicitly in the task report and make no commit.

---

### Task 3: Bump the bot report to top-25

**Files:**
- Modify: `internal/service/screener/dividend/types.go:13` (`defaultTopN`)

**Interfaces:**
- Consumes: nothing new.
- Produces: `defaultTopN = 25` (used by `Send` and by `Top(ctx, n<=0)` fallback).

- [ ] **Step 1: Change the constant**

In `internal/service/screener/dividend/types.go`, change:

```go
const defaultTopN = 15
```

to:

```go
const defaultTopN = 25
```

- [ ] **Step 2: Run the service tests to confirm nothing breaks**

Run: `go test ./internal/service/screener/dividend/`
Expected: `ok`. (`TestTop_ReportsGateStats` pins `Top(ctx, 15)` explicitly and the 4-fixture universe has only 3 survivors, so the fallback-size tests still return the same counts.)

- [ ] **Step 3: Commit**

```bash
git add internal/service/screener/dividend/types.go
git commit -m "$(cat <<'EOF'
feat(screener): show top-25 in /dividend_screener report

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Documentation and final CI gate

**Files:**
- Modify: `docs/dividend/screener.md`
- Modify: `CLAUDE.md` (only if it references top-15 — verify first)

**Interfaces:**
- Consumes: the final `MinFreeFloat` value from Task 2 (0.07 unless retuned).

- [ ] **Step 1: Update the gate table and rules in `docs/dividend/screener.md`**

In the "Ворота отсева (`gate`)" table, add a row after the `низкая ликвидность` row:

```markdown
| `тонкий free-float` | `FreeFloat < MinFreeFloat` (0.07; в т.ч. 0 = нет данных) | отсекает имена с околонулевым свободным обращением (неторгуемые) |
```

Update the intro sentence of that section's ordered list and the "Что НЕ отсеивает" note is unaffected. In the "Осознанные ограничения" section, replace the market-cap-≠-float limitation (the `MarketCapitalization==0` follow-up wording) with a note that free-float now guards thin-float large-caps, and document the new deliberate risk:

```markdown
- **`FreeFloat == 0` → отсев.** Как и `MarketCapitalization == 0`, ноль в
  free-float трактуется как неликвид и отсеивает имя. Это может задеть
  легитимное ликвидное имя с дырой в данных (напр. OZON приходит с float 0),
  но такие имена и так с бонусом 0 и внизу списка; альтернатива (пропускать 0)
  вернула бы неторгуемые имена вроде Удмуртнефти в топ. Риск оставлен осознанно.
```

- [ ] **Step 2: Update report size and calibration history in `docs/dividend/screener.md`**

Change every "топ-15" / "top-15" reference (in the intro and the `/dividend_screener` bullet) to "топ-25". In the "История калибровки" list, append:

```markdown
- **2026-07-21 (free-float gate)** — добавлен второй порог ликвидности
  `MinFreeFloat` (0.07) в `gate()`: отсев `тонкий free-float` для имён с
  околонулевым свободным обращением (Удмуртнефть #1 с float 0 % уходит).
  Market-cap floor без изменений. Отчёт бота расширен до топ-25. Диагностика —
  `cmd/divscreen` (stdout, `-probe`).
```

- [ ] **Step 3: Check and update `CLAUDE.md`**

Run: `grep -n "топ-15\|top-15\|15 " CLAUDE.md`
If any line describes the dividend screener report as top-15, update it to top-25. If there is no such reference, make no change.

- [ ] **Step 4: Commit docs**

```bash
git add docs/dividend/screener.md CLAUDE.md
git commit -m "$(cat <<'EOF'
docs(screener): document free-float gate and top-25 report

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 5: Run the full CI gate**

Run: `./bin/mage ci`
Expected: EXIT 0 — golangci-lint clean, `go test -race ./...` all pass, no mock drift.

If lint flags the new code (e.g. gofmt), fix inline and amend the relevant commit, then re-run until green.

---

## Notes for the executor

- The `cmd/divscreen` CLI, the `FreeFloat`/`AverageDailyVolumeLast4Weeks` model+converter mapping, and `rank.GateDecision` are already committed on this branch (`b657995`) — do not re-create them.
- No mockery regeneration is expected (the `instrumentsClient` interface is unchanged). If `./bin/mage ci` reports mock drift, investigate before regenerating.
