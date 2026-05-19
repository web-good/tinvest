# Golden X Stage D2 — Strategy Knobs to `Settings` Struct

## Context

Branch `feature/golden-x-stage-a` already carries Stages A, B, C1–C5, D0 (Postgres-infra removal) and the full D3 backtest arc (Tasks 2.1–2.9). The master plan (`~/.claude/plans/internal-service-trading-strategy-golde-woolly-quill.md`) describes D2 as: «Вынести пороги RSI в конфиг/константы (сейчас 31/35/40 захардкожены в нескольких местах)».

**Reality check.** The literals 31/35/40 are already gone — C2 (adaptive p5/p15 buy tiers) and B (adaptive p80/p90/p95 sell tiers) replaced them with per-share percentiles. So the original D2 wording is stale. What *remains* hardcoded is the cluster of **strategy knobs**: percentile points (5/15/80/90/95), ATR stop parameters (period 14, multipliers 2.0/1.5), volume-confirmation parameters (SMA20, ×1.5), and history-window sizes (EMA200, adaptive min/max, divergence 52). D2 in its updated form extracts these into a single `Settings` struct.

**Why now.** With `cmd/backtest` shipped (D3), the natural next step is parameter sweeps — but every sweep currently requires a recompile because the knobs are unexported package-level `const`s. Even before the sweep mechanism itself, the prerequisite is a single value-type that holds all knobs and `DefaultSettings()` that returns today's values. That foundation is D2; the actual CLI/env-var override mechanism is deferred.

**Outcome (this iteration).** A new `dto.Settings` struct + `golden_x.DefaultSettings()` constructor. `Detect` accepts a 5th `settings` argument. All in-package `const` knobs are removed; production wiring and `cmd/backtest` both call `DefaultSettings()`. **No behavioral change** — the Telegram output and any backtest Markdown report are byte-identical to pre-D2 runs. Override mechanism (`--settings=path.json`, env-var injection, etc.) is **explicitly out of scope** and deferred to a follow-up stage.

## Locked decisions

| Question | Decision |
|---|---|
| What constants enter `Settings` | **All four groups:** buy/sell percentiles, ATR stop knobs, volume confirmation knobs, history windows. 14 fields total. |
| Override mechanism | **None this iteration.** Defaults only. CLI flags / JSON files / env-vars deferred. |
| Struct shape | **Flat** — all 14 fields at top level, grouped by comment block. Not nested groups. |
| Detect signature | **Append `settings dto.Settings` as 5th arg**, do not bundle existing args into a Params struct. |
| Where `Settings` lives | **`dto/settings.go`**, alongside `Signal`, `Thresholds`, `SellThresholds`. Keeps `dto` as the shared value-type bag, lets both `golden_x` and `golden_x/backtest` import it without cycles. |
| Where `DefaultSettings()` lives | **`golden_x` package** (returns `dto.Settings`). `dto` remains data-only. |
| Naming convention | Fields named by **role**, not by literal value (`BuyGreen`, not `BuyP5`). Survives future retuning without semantic drift. |
| Validation | **No `Settings.Validate()` method.** Will land together with overrides when an external constructor exists. |
| Per-share `RSILength` | **Stays per-share in `collection.Instrument`.** Out of scope for D2 — it's not a strategy knob, it's an instrument property. |
| Behavior parity | **Byte-identical** Telegram output and backtest Markdown vs. pre-D2 (modulo timestamps). |
| Future overrides | Tracked as separate stage (D2.1 or similar). Out of scope here. |

## Architecture overview

```
                         dto.Settings (value-type, 14 fields)
                              │
   ┌──────────────────────────┼──────────────────────────────┐
   │                          │                              │
   ▼                          ▼                              ▼
golden_x.DefaultSettings()    Detect(..., settings)          ReplayConfig.Settings
   │                          │                              │
   │  service.Trade calls it once per Trade() loop,          │
   │  cmd/backtest calls it once at startup;                 │
   │  passed by value into each Detect / Replay invocation.  │
   ▼                          ▼                              ▼
14 fields = today's          replaces all in-pkg consts:   replay.go's DetectFunc
hardcoded values             atrPeriod, volumeMultiplier,   signature widens to
                             trendEMAPeriod, etc.           accept settings too
```

`Settings` is small enough to copy by value on every call. No shared state, no mutation, no goroutine concerns.

## Components

### `dto/settings.go` (new)

```go
package dto

// Settings carries all tunable strategy knobs for Golden X. Each field has a
// well-defined default in golden_x.DefaultSettings(). Names use role-based
// terminology, not literal-value terminology, so the meaning survives retuning.
type Settings struct {
    // Buy-tier percentiles (RSI < BuyGreen → green; < BuyYellow → yellow).
    BuyGreen  float64 // default 5
    BuyYellow float64 // default 15

    // Sell-tier percentiles. Growth uses only SellOrange; Dividend uses all three.
    SellYellow float64 // default 80 (Dividend only)
    SellOrange float64 // default 90 (both kinds)
    SellRed    float64 // default 95 (Dividend only)

    // ATR-based stop.
    ATRPeriod             int     // default 14
    ATRMultiplierDividend float64 // default 2.0
    ATRMultiplierGrowth   float64 // default 1.5

    // Volume-confirmation indicator (last weekly volume > Multiplier × SMA of preceding Lookback weeks).
    VolumeSMALookback int     // default 20
    VolumeMultiplier  float64 // default 1.5

    // History windows.
    TrendEMAPeriod          int // default 200 (EMA200 W trend filter for Growth)
    AdaptiveWindowMin       int // default 100 (minimum closed-week RSI samples for adaptive tiers)
    AdaptiveWindowMax       int // default 200 (cap on closed-week RSI samples kept for percentiles)
    DivergenceLookbackWeeks int // default 52  (bullish-divergence pivot search horizon)
}
```

### `golden_x/settings.go` (new)

```go
package golden_x

import "tinvest/internal/service/trading_strategy/golden_x/dto"

// DefaultSettings returns the strategy knobs in use across the codebase prior
// to D2. Behavioral parity is the explicit contract: calling Detect with
// DefaultSettings() must produce byte-identical output to the pre-D2 code.
func DefaultSettings() dto.Settings {
    return dto.Settings{
        BuyGreen:                5,
        BuyYellow:               15,
        SellYellow:              80,
        SellOrange:              90,
        SellRed:                 95,
        ATRPeriod:               14,
        ATRMultiplierDividend:   2.0,
        ATRMultiplierGrowth:     1.5,
        VolumeSMALookback:       20,
        VolumeMultiplier:        1.5,
        TrendEMAPeriod:          200,
        AdaptiveWindowMin:       100,
        AdaptiveWindowMax:       200,
        DivergenceLookbackWeeks: 52,
    }
}
```

### `golden_x/detector.go` (modified)

Signature change:

```go
func Detect(
    closed []*model.CandleItemTechAnalyse,
    rsiPeriod int,
    kind dto.StrategyKind,
    useTrendFilter bool,
    settings dto.Settings,
) (dto.Signal, error)
```

Seven internal replacements (literal/const → field):
- `adaptiveRSIForShare(closed, rsiPeriod)` → `adaptiveRSIForShare(closed, rsiPeriod, settings.AdaptiveWindowMin, settings.AdaptiveWindowMax)`
- `adaptiveThresholds(rsiSeries)` → `adaptiveThresholds(rsiSeries, settings.BuyGreen, settings.BuyYellow)`
- `adaptiveSellThresholds(rsiSeries)` → `adaptiveSellThresholds(rsiSeries, settings.SellYellow, settings.SellOrange, settings.SellRed)`
- `trendStatusFromClosed(closed, trendEMAPeriod)` → `trendStatusFromClosed(closed, settings.TrendEMAPeriod)`
- `divergenceLookbackWeeks` (literal in two places) → `settings.DivergenceLookbackWeeks`
- `indicators.VolumeConfirmed(volumes, volumeSMALookback, volumeMultiplier)` → `indicators.VolumeConfirmed(volumes, settings.VolumeSMALookback, settings.VolumeMultiplier)`
- `indicators.ATR(highs, lowsF, closes, atrPeriod)` → `indicators.ATR(highs, lowsF, closes, settings.ATRPeriod)`
- `kForKind(kind)` → `kForKind(kind, settings)`

### `golden_x/percentile.go` (modified)

Functions become pure with explicit dependencies (no hidden literals):

```go
func adaptiveThresholds(rsiSeries []float64, pGreen, pYellow float64) dto.Thresholds {
    sorted := append([]float64(nil), rsiSeries...)
    sort.Float64s(sorted)
    return dto.Thresholds{
        P5:  percentile(sorted, pGreen),
        P15: percentile(sorted, pYellow),
    }
}

func adaptiveSellThresholds(rsiSeries []float64, pYellow, pOrange, pRed float64) dto.SellThresholds {
    sorted := append([]float64(nil), rsiSeries...)
    sort.Float64s(sorted)
    return dto.SellThresholds{
        P80: percentile(sorted, pYellow),
        P90: percentile(sorted, pOrange),
        P95: percentile(sorted, pRed),
    }
}
```

Note: `dto.Thresholds.P5/P15` and `dto.SellThresholds.P80/P90/P95` field names stay as-is. They represent slots, not strict percentile values — retunes only change which percentile fills each slot. (Renaming those fields is a separate, larger refactor and **out of scope** here.)

### `golden_x/stop.go` (modified)

```go
func kForKind(kind dto.StrategyKind, settings dto.Settings) float64 {
    if kind == dto.StrategyKindGrowth {
        return settings.ATRMultiplierGrowth
    }
    return settings.ATRMultiplierDividend
}
```

`stopFromATR` keeps current signature — it already takes `k` as an arg.

### `golden_x/trend_filter.go` (modified)

Remove `const trendEMAPeriod = 200`. `trendStatusFromClosed(closed, period)` already takes period as arg; no signature change needed.

### `golden_x/trade.go` (modified)

- Remove eight `const` declarations: `adaptiveWindowMax`, `adaptiveWindowMin`, `divergenceLookbackWeeks`, `volumeSMALookback`, `volumeMultiplier`, `atrPeriod`, `atrMultiplierDividend`, `atrMultiplierGrowth`.
- **Kept in place** (intentionally out of scope): `candleLookbackWeeks` (fetch-policy, not algorithm) and `divergenceFractalK` (narrow C3 internal — pivot width, not on the user's selected list of D2 knobs).
- `adaptiveRSIForShare(closed, rsiPeriod, minWin, maxWin int)` and `lowsAlignedToRSI(closed, rsiPeriod, rsiSeries, maxWin int)` accept window bounds explicitly.
- In `Trade()` orchestrator: build `settings := DefaultSettings()` once before the per-share loop; pass into each `Detect` call.

### `golden_x/backtest/replay.go` (modified)

```go
type DetectFunc func(
    closed []*model.CandleItemTechAnalyse,
    rsiPeriod int,
    kind dto.StrategyKind,
    useTrendFilter bool,
    settings dto.Settings,
) (dto.Signal, error)

type ReplayConfig struct {
    Kind           dto.StrategyKind
    RSIPeriod      int
    StartIdx       int
    MaxWeeks       int
    UseTrendFilter bool
    Settings       dto.Settings // NEW
}
```

Inside `Replay`, the single `detect(...)` call gains the `cfg.Settings` argument.

### `cmd/backtest/main.go` (modified)

One-liner addition: build `golden_x.DefaultSettings()` near the existing config parsing and write into `ReplayConfig.Settings` when constructing each per-share replay. No new CLI flag.

## Data flow

```
service.Trade()                       cmd/backtest/main.go
    │                                       │
    ├─→ settings := golden_x.              ├─→ settings := golden_x.
    │       DefaultSettings()              │       DefaultSettings()
    │                                       │
    ├─→ for each share:                    ├─→ ReplayConfig{
    │       Detect(closed, rsiP,           │       Settings: settings, ...}
    │              kind, useTrendFilter,   │
    │              settings)               ├─→ Replay(... cfg ...)
                                                    │
                                                    └─→ detect(closed, rsiP, kind,
                                                                useTrendFilter,
                                                                cfg.Settings)
```

`Settings` is value-type, copied on each call. No goroutine concerns, no mutation, no aliasing.

## Error handling

No new error paths. All existing returns preserved:
- `ErrInsufficientHistory` (in detector) — triggered when `len(closed) < settings.AdaptiveWindowMin`.
- `ErrAdaptiveInsufficientHistory` (in trade.go) — same condition surfaced at the call site.
- `Detect` returns `(dto.Signal{}, err)` for unprocessable histories.
- `Replay` continues to ignore `detect` errors via `continue`.

No `Settings.Validate()` method this iteration — `DefaultSettings()` is the only constructor and is statically correct.

## Testing

### Unit tests (existing — adapt)

Every test that calls `Detect(...)`, `adaptiveThresholds(...)`, `adaptiveSellThresholds(...)`, `adaptiveRSIForShare(...)`, `kForKind(...)`, `lowsAlignedToRSI(...)` updates to either pass `DefaultSettings()` or pass the specific scalar params it now needs. Behavioral assertions stay unchanged — that's the whole point of the byte-parity contract.

Files affected (verify during implementation; this list is best-effort):
- `golden_x/detector_test.go`
- `golden_x/percentile_test.go` (if present)
- `golden_x/trade_test.go`
- `golden_x/stop_test.go` (if present)
- `golden_x/backtest/replay_test.go` — fake `DetectFunc` signature widens by one arg; mock can ignore `settings`.

### New regression test

`golden_x/settings_test.go` (location: same package as `DefaultSettings()`):

```go
func TestDefaultSettings(t *testing.T) {
    s := DefaultSettings()
    want := dto.Settings{
        BuyGreen: 5, BuyYellow: 15,
        SellYellow: 80, SellOrange: 90, SellRed: 95,
        ATRPeriod: 14, ATRMultiplierDividend: 2.0, ATRMultiplierGrowth: 1.5,
        VolumeSMALookback: 20, VolumeMultiplier: 1.5,
        TrendEMAPeriod: 200, AdaptiveWindowMin: 100, AdaptiveWindowMax: 200,
        DivergenceLookbackWeeks: 52,
    }
    if s != want {
        t.Fatalf("DefaultSettings drift:\n got %+v\nwant %+v", s, want)
    }
}
```

Catches accidental fat-fingers in defaults during future edits.

### Gates

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./...` — all green
- Manual sanity: optionally run `cmd/backtest` against the cached Dividend full list and `diff` the Markdown report against a pre-D2 run; should be byte-identical (modulo the `generated <timestamp>` line in the header).

## Out of scope

- **CLI flag / JSON file / env-var overrides** for `Settings`. Will be a follow-up stage (provisionally D2.1 or merged into D4) once the user wants actual parameter sweeps.
- **`Settings.Validate()`** method. Lands together with overrides.
- **Renaming `dto.Thresholds.P5/P15` and `dto.SellThresholds.P80/P90/P95`** to role-based names. Larger refactor, doesn't enable any new functionality, deferred.
- **Per-share knob overrides** (e.g., a per-instrument ATR multiplier). Per-share data still lives in `collection.Instrument`; not relevant until a real use case appears.
- **Production env-var integration** through `heetch/confita` and `internal/config`. Same deferral.
- **`divergenceFractalK`** (the fractal-pivot half-width used by the C3 divergence detector, currently `= 2`). Narrow indicator-internal parameter, not on the user-selected list of D2 knobs. Can move into `Settings` later as an indicator-tuning follow-up.
- **`candleLookbackWeeks`** (currently `= 260`). It governs how many weekly candles we *fetch* from gRPC, not how the algorithm processes them — a fetch/policy knob rather than an algorithm knob. Out of scope.

## Acceptance

D2 is done when:

1. `dto.Settings` exists with all 14 fields and the documented defaults.
2. `golden_x.DefaultSettings()` returns those defaults.
3. `Detect` accepts a 5th `settings` argument; all in-package `const` knobs for strategy parameters are removed.
4. `service.Trade` and `cmd/backtest/main.go` both call `DefaultSettings()` and pass it through.
5. `backtest.ReplayConfig` carries `Settings dto.Settings`; `DetectFunc` matches the new `Detect` signature.
6. `go build ./... && go vet ./... && go test ./...` all clean.
7. New `TestDefaultSettings` regression test passes.
8. A backtest run against the cached Dividend full list produces byte-identical output (modulo timestamp) compared to pre-D2.
9. Commit lands on `feature/golden-x-stage-a` with a message in the style of the prior D3 commits (e.g. `refactor(golden_x): extract strategy knobs into dto.Settings`).
