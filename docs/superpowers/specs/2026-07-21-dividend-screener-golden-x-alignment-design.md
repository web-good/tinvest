# Dividend Screener — Golden X Alignment (Liquidity Filter + Absolute Bonus Bands)

**Date:** 2026-07-21
**Status:** Design approved, pending spec review
**Branch:** feat/dividend-screener (continues the gate-calibration work)

## Problem

The dividend fundamental screener ranks the whole Moscow dividend universe (~191
names) and feeds two consumers:

1. **Golden X `FundamentalBonus`** — `RankBonus(instrumentID)` folds a bonus into
   the Golden X buy score for the instrument it is currently evaluating.
2. **`/dividend_screener` Telegram command** — renders `Top(N)` as a leaderboard.

Live validation (2026-07-21, gate-calibration diagnostic) exposed the core defect:
the bonus is a **relative decile** over a universe polluted by illiquid micro-caps.
Small-cap regional energy retailers (Пермэнергосбыт, ТНС энерго, Россети regionals,
Мордовская энергосбытовая) have higher raw yields and mid-range payouts, so they
occupy the top of the rank, while the recognizable, tradable blue chips sink below
the median:

| Name  | Rank      | Decile bonus (current) |
|-------|-----------|------------------------|
| PHOR  | #31/110   | +2                     |
| TATN  | #37/110   | +2                     |
| SBER  | #53/110   | +1                     |
| SNGSP | #57/110   | +1                     |
| ROSN  | #64/110   | 0                      |
| LKOH  | #77/110   | 0                      |

**Decisive fact:** Golden X does **not** trade the broad universe. Its dividend
list is a curated 11 instruments (`internal/service/trading_strategy/golden_x/shares/shares.go`
`Dividend()`): Сургут-прив, Татнефть-прив, Роснефть, Лукойл, Сбер-прив, Северсталь,
НЛМК, ММК, ФосАгро, Транснефть, Банк СПб. `RankBonus` is only ever consulted for
these. Because all 11 cluster into the same low decile band against 191 names
(most illiquid), the bonus barely differentiates them — the fundamental tilt the
feature exists to provide does not materialize.

## Purpose (agreed)

The screener's **primary role is to serve Golden X (role A):** surface fundamentally
strong *liquid, tradable* dividend names and give Golden X a bonus that actually
differentiates the names it trades. The Telegram list is a secondary window into the
same rank and becomes useful as a byproduct.

## Approach (agreed)

Two levers, applied together:

- **A1 — Liquidity filter on the universe** (market-cap floor, lever **B1**): rank
  only liquid, sizeable dividend names. Shrinking the survivor pool to liquid peers
  also fixes the three percentile pillars (DivGrowth/Quality/Valuation are computed
  among survivors), so blue-chip composites become comparable instead of being
  crushed by thin-equity small-caps.
- **A3 — Absolute bonus bands**: convert the bonus from a relative decile to fixed
  composite score bands, so bonus differentiation is robust to universe composition.

Rejected alternatives:
- **A2 (rank within Golden X's own 11 names)** — too thin a base (top decile = 1
  name), zero discovery value.
- **B2 (`liquidity_flag`) / B3 (flag AND cap)** — B2 imports a foreign definition and
  carries the bonds-screener risk of over-filtering legitimate instruments; B3 is
  harder to explain in the Telegram отсев breakdown. A single tunable market-cap
  floor is transparent and validates cleanly on live data. ADV/turnover can be added
  later if the cap floor proves insufficient.

## Components / Changes

All ranking changes stay in the pure core `internal/service/screener/dividend/rank/`
(no I/O); the bonus mechanic lives in the orchestration layer `service.go`.

### 1. Fundamentals model + converter

Add one field to `internal/model/fundamentals.go`:

```go
MarketCapitalization float64 // рыночная капитализация, валюта инструмента
```

Map it in `ConvertFundamentalsFromPb` from
`GetAssetFundamentalsResponse_StatisticResponse.MarketCapitalization`, following the
existing nil-skip pattern. No new RPC — market cap arrives in the same
`GetAssetFundamentals` batch already fetched. It is `proto3 float64` with `omitempty`,
so **0 means "no data"** (same convention as every other fundamental field).

### 2. Liquidity filter in the pure core

Add to `rank.Config`:

```go
MinMarketCap float64 // ниже (в т.ч. 0 = нет данных) — отсев как неликвид
```

Add a new gate reason and branch in `gate()` (keeps the function pure). Placement:
**after** the `reasonNoDividend` check (we only care about liquid *dividend payers*)
and before the yield-trap / leverage / payout checks:

```go
const reasonIlliquid = "низкая ликвидность"

// inside gate(), after the y <= 0 check:
if f.MarketCapitalization < cfg.MinMarketCap {
    return reasonIlliquid, false
}
```

A name with missing market cap (0) is excluded — safe for role A, since we only tilt
Golden X toward names we can confirm are tradable, and the curated 11 all have real
caps (confirmed in the live step).

**Effect multiplier:** the survivor pool shrinks to liquid names, so the percentile
pools (`divGrowth`, `roic`, `evEbitda` in `Rank`) are built from liquid peers only.
No code change to the pillar computation — it automatically ranks blue chips against
blue chips.

### 3. Absolute bonus bands (A3)

Replace `bonusFromRank(idx, total)` in `service.go` with a score-band function:

```go
// bonusFromScore maps a composite (0..100) to a Golden X fundamental bonus.
// Thresholds are calibration points (see live-validation step).
func bonusFromScore(composite float64, cfg Config) int {
    switch {
    case composite >= cfg.BonusScoreT3:
        return 3
    case composite >= cfg.BonusScoreT2:
        return 2
    case composite >= cfg.BonusScoreT1:
        return 1
    default:
        return 0
    }
}
```

- The three thresholds (`BonusScoreT1/T2/T3`) live in `rank.Config` alongside
  `MinMarketCap`, so every calibration knob stays in one `DefaultConfig()`. `service`
  already holds the `rank.Config` it passes to `rank.Rank`, so `bonusFromScore` reads
  them from there.
- `refresh()` stores each ranked instrument's composite; `RankBonus(instrumentID)`
  returns `bonusFromScore(composite, cfg)` for that instrument, `0` for
  gated/unknown instruments (the existing never-panic default).
- The decile logic (`bonusFromRank`, `idx`/`total`) is removed.

### 4. Telegram render

No mechanic change. `Top(N)`/`Render` now naturally show the liquid dividend
leaderboard, and the отсев breakdown gains a "низкая ликвидность" line automatically
(it flows through the generic `Stats.ByReason` map).

## Data Flow

```
GetAssetFundamentals (batch, +MarketCapitalization)
  → []model.Fundamentals
    → rank.Rank(universe, cfg)
        gate(): drop non-payers ("нет дивиденда"),
                drop illiquid payers ("низкая ликвидность"),
                drop yield-traps / high-leverage / unsustainable
        → survivors ranked; percentile pillars over LIQUID peers
        → []ScoredCompany (composite per liquid survivor)
    → service caches composites
        RankBonus(id) = bonusFromScore(composite[id], cfg)  → Golden X buy score
        Top(N) + Render                                     → Telegram leaderboard
```

## Testing

Pure table-driven tests in `rank_test.go`:
- Market-cap gate: missing cap (0) → `reasonIlliquid`; below floor → `reasonIlliquid`;
  at/above floor → survives (given it is otherwise a valid payer).
- Ordering: illiquid names excluded from percentile pools (a survivor's pillar is
  computed only among liquid peers).

Tests in `service` (or wherever `bonusFromRank` was tested):
- `bonusFromScore` bands: composites straddling T1/T2/T3 map to 0/1/2/3.
- `RankBonus` returns 0 for gated/unknown instruments.
- Update/replace any existing test that asserted the old decile bonus.

`./bin/mage ci` (lint + `go test -race ./...` + mock-drift) must pass. Adding a
`MarketCapitalization` field to the mocked-through path does not change any mocked
interface, so `./bin/mage mocks` is not expected to be needed (confirm via
mock-drift check).

## Live-validation step (calibration, not committed)

Mirror the gate-calibration Task 3 pattern: a temporary `cmd/divcheck`-style
diagnostic (git-ignored, deleted after use) that hits the live Invest API and prints,
for the liquid universe:

1. **Unit check for `MarketCapitalization`** — confirm the magnitude/units the API
   returns (RUB vs millions), exactly as the previous plan confirmed yield/payout are
   percents. Seed `MinMarketCap` is provisional until this is read off live data.
2. **Curated-11 survival** — every Golden X `Dividend()` name passes the cap floor and
   lands in a sensible bonus band.
3. **Micro-cap exclusion** — Мордовская энергосбытовая, ТНС энерго regionals, etc.
   drop with reason "низкая ликвидность".
4. **Band distribution** — the liquid universe's composite distribution is inspected
   to set `BonusScoreT1/T2/T3` so the bands meaningfully split strong vs weak liquid
   payers.

**Calibration seeds (provisional, to be nailed in the live step):**
- `MinMarketCap`: start ~50e9 (₽50 bn) — admits mid/large caps, drops micro; adjust
  once units are confirmed and the curated-11 are checked.
- `BonusScoreT1/T2/T3`: start 55 / 65 / 75 — will shift after liquidity filtering
  recomputes the percentile pillars; set from the observed liquid distribution.

Final `MinMarketCap` and band thresholds are committed as a `config` calibration
change (like `chore(screener): calibrate ...`).

## Known limitations (deliberately not addressed)

- **Stale/odd API snapshots** (e.g. LKOH returned ROE −20%, payout 0): such a name
  stays in the liquid universe (large cap) but earns a low composite → low/zero bonus.
  This is correct behavior — the bonus reflects current reported fundamentals; the
  screener does not second-guess the API. Data-quality repair is out of scope.
- **Market cap ≠ turnover.** A large-cap that is thinly traded would pass the floor.
  Not observed on Moscow blue chips; ADV filter deferred until evidence demands it.
- **Missing-net-debt optimism** and **unused `indicators.Percentile` (R-7)** — carried
  over from the prior plan's known-limitations, still out of scope.
