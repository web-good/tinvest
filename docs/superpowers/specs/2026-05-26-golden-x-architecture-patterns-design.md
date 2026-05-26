# Golden X Architecture Patterns Refactoring

**Date:** 2026-05-26
**Status:** Draft
**Scope:** `internal/service/trading_strategy/golden_x/`

## Context

Golden X actively evolves: RSI, EMA200 trend filter, divergence, volume confirmation, ATR stops were added stage by stage (C2-D2). Each addition extended the monolithic `Detect()` function and added maps to `Trade()`. The code works but has accumulated structural debt:

1. `Detect()` is 87 lines doing 8 sequential operations with no composability.
2. `Trade()` manages 7 parallel maps passed as 9 arguments to `notification.Trade()`.
3. Kind-specific behavior (Dividend vs Growth) is scattered across `kForKind()`, `sellTierFromAdaptive()`, and `EvaluateSellExits()`.
4. No validation of `Settings` — silently accepts `BuyGreen > BuyYellow`.

## Goals

- **Extensibility**: adding a new indicator stage should not require editing `Detect()`.
- **Reusability**: Kind-specific logic centralized in one place, usable by both live and backtest paths.
- **Clean architecture**: idiomatic Go patterns, fail-fast validation, reduced function signatures.

## Approach: Pipeline + StrategyProfile + Invariant + ShareResult

---

### 1. Pipeline Pattern (Detect refactoring)

#### 1.1 Core types

```go
// pipeline.go (new file, package golden_x)

type DetectContext struct {
    // Input — set once by caller, immutable during pipeline run
    Candles    []*model.CandleItemTechAnalyse
    RSIPeriod  int
    Profile    *StrategyProfile

    // Intermediate — populated by early stages, read by later ones
    Closes    []float64
    RSISeries []float64
    LastRSI   float64
    BuyTier   alertTier

    // Output — enriched incrementally
    Signal dto.Signal
}

type Stage interface {
    Run(ctx *DetectContext) error
}

type Pipeline struct {
    stages []Stage
}

func (p *Pipeline) Run(ctx *DetectContext) (dto.Signal, error) {
    for _, s := range p.stages {
        if err := s.Run(ctx); err != nil {
            return dto.Signal{}, err
        }
    }
    return ctx.Signal, nil
}
```

#### 1.2 Stages

Each stage is a small struct implementing `Stage`. Existing math functions remain unchanged — stages orchestrate calls.

| # | Struct | Source function(s) | Writes to DetectContext |
|---|--------|--------------------|------------------------|
| 1 | `RSIStage` | `adaptiveRSIForShare` | `.Closes`, `.RSISeries`, `.LastRSI`, `.Signal.RSI` |
| 2 | `BuyThresholdsStage` | `adaptiveThresholds` | `.Signal.Thresholds` |
| 3 | `SellThresholdsStage` | `adaptiveSellThresholds` | `.Signal.SellThresholds` |
| 4 | `TrendFilterStage` | `trendStatusFromClosed` | `.Signal.TrendStatus` |
| 5 | `BuyTierStage` | `tierFromAdaptive` | `.BuyTier`, `.Signal.GreenBuy`, `.Signal.YellowBuy` |
| 6 | `DivergenceStage` | `lowsAlignedToRSI`, `bullishDivergence` | `.Signal.DivergenceOK` |
| 7 | `VolumeStage` | `indicators.VolumeConfirmed` | `.Signal.VolumeOK` |
| 8 | `StopStage` | `indicators.ATR`, `stopFromATR` | `.Signal.Stop`, `.Signal.LastClose` |

**Conditional stages** check context before running:
- `TrendFilterStage`: no-op if `!ctx.Profile.UseTrendFilter`
- `DivergenceStage`, `VolumeStage`, `StopStage`: no-op if `ctx.BuyTier == tierNone`

Stages 6-8 all gate on BuyTier. When BuyTier is None, `StopStage` still sets `Signal.LastClose` from the last candle (current behavior preserved).

#### 1.3 File layout

All stages go into one file `stages.go` (they are small — 10-20 lines each). The `Pipeline` and `DetectContext` types go into `pipeline.go`.

#### 1.4 Backward compatibility

`Detect()` function signature changes to accept `*StrategyProfile` instead of separate `kind`, `useTrendFilter`, `settings` params:

```go
func Detect(
    closed []*model.CandleItemTechAnalyse,
    rsiPeriod int,
    profile *StrategyProfile,
) (dto.Signal, error)
```

Internally, `Detect()` constructs a `DetectContext` and calls `NewDetectPipeline(profile).Run(ctx)`.

The backtest `DetectFunc` type updates to match the new signature.

---

### 2. StrategyProfile (Factory Pattern)

#### 2.1 Type definition

```go
// profile.go (new file, package golden_x)

type SellBehavior int

const (
    SellPartialThreeTiers SellBehavior = iota // Dividend: P80/P90/P95 each 1/3
    SellFullSingleTier                         // Growth: P90 full exit
)

type StrategyProfile struct {
    Kind           dto.StrategyKind
    Settings       dto.Settings
    UseTrendFilter bool
    ATRMultiplier  float64
    SellBehavior   SellBehavior
}

func NewDividendProfile(settings dto.Settings, useTrendFilter bool) *StrategyProfile {
    return &StrategyProfile{
        Kind:           dto.StrategyKindDividend,
        Settings:       settings,
        UseTrendFilter: useTrendFilter,
        ATRMultiplier:  settings.ATRMultiplierDividend,
        SellBehavior:   SellPartialThreeTiers,
    }
}

func NewGrowthProfile(settings dto.Settings, useTrendFilter bool) *StrategyProfile {
    return &StrategyProfile{
        Kind:           dto.StrategyKindGrowth,
        Settings:       settings,
        UseTrendFilter: useTrendFilter,
        ATRMultiplier:  settings.ATRMultiplierGrowth,
        SellBehavior:   SellFullSingleTier,
    }
}
```

#### 2.2 What it replaces

- `kForKind(kind, settings)` in stop.go -> `profile.ATRMultiplier`
- Switch on `kind` in `sellTierFromAdaptive()` -> `profile.SellBehavior`
- Switch on `p.kind` in `Position.EvaluateSellExits()` -> profile.SellBehavior (backtest receives profile)
- Inline construction of Kind + Settings + UseTrendFilter in `Trade()` and `app.go` -> single `NewDividendProfile()`/`NewGrowthProfile()` call

#### 2.3 Pipeline factory

```go
func NewDetectPipeline(profile *StrategyProfile) *Pipeline {
    stages := []Stage{
        &RSIStage{},
        &BuyThresholdsStage{},
        &SellThresholdsStage{},
    }
    if profile.UseTrendFilter {
        stages = append(stages, &TrendFilterStage{})
    }
    stages = append(stages,
        &BuyTierStage{},
        &DivergenceStage{},
        &VolumeStage{},
        &StopStage{},
    )
    return &Pipeline{stages: stages}
}
```

---

### 3. Invariant Pattern

#### 3.1 Settings validation

Added as a method on `dto.Settings`:

```go
func (s Settings) Validate() error {
    // Buy percentiles ordered
    if s.BuyGreen >= s.BuyYellow {
        return fmt.Errorf("BuyGreen (%.0f) must be < BuyYellow (%.0f)", s.BuyGreen, s.BuyYellow)
    }
    // Sell percentiles ordered
    if s.SellYellow >= s.SellOrange || s.SellOrange >= s.SellRed {
        return fmt.Errorf("sell percentiles must be ordered: %.0f < %.0f < %.0f",
            s.SellYellow, s.SellOrange, s.SellRed)
    }
    // Positive periods
    if s.ATRPeriod <= 0 { ... }
    if s.ATRMultiplierDividend <= 0 || s.ATRMultiplierGrowth <= 0 { ... }
    // Window bounds
    if s.AdaptiveWindowMin <= 0 || s.AdaptiveWindowMax <= 0 ||
       s.AdaptiveWindowMin > s.AdaptiveWindowMax { ... }
    // Volume params
    if s.VolumeSMALookback <= 0 || s.VolumeMultiplier <= 0 { ... }
    // Trend EMA
    if s.TrendEMAPeriod <= 0 { ... }
    // Divergence lookback
    if s.DivergenceLookbackWeeks <= 0 { ... }
    return nil
}
```

Called once at the start of `Trade()` and `Replay()` — fail-fast before any RPC or computation.

#### 3.2 Signal invariant (debug/test aid)

```go
func (s Signal) Invariant() error {
    if s.GreenBuy && s.YellowBuy {
        return errors.New("GreenBuy and YellowBuy are mutually exclusive")
    }
    if s.RSI < 0 || s.RSI > 100 {
        return fmt.Errorf("RSI out of [0,100]: %.2f", s.RSI)
    }
    if s.Stop.Price < 0 {
        return fmt.Errorf("Stop.Price negative: %.2f", s.Stop.Price)
    }
    return nil
}
```

Called in tests and optionally at the end of the pipeline. Not called in production hot path (pure check, no side effects).

---

### 4. ShareResult (Trade method cleanup)

#### 4.1 Type

```go
// share_result.go (new file, package golden_x)

type ShareResult struct {
    ShareID   string
    ShareName string
    Signal    dto.Signal
    BuyTier   alertTier
    SellTier  alertTier
}
```

#### 4.2 Trade() refactoring

Current: 7 maps + loop + 9-param notification call.
After: single `[]ShareResult` slice + 2-param notification call.

```go
func (s *service) Trade(ctx context.Context, in dto.Trade) error {
    profile := profileFromTrade(in)
    if err := profile.Settings.Validate(); err != nil {
        return err
    }

    var results []ShareResult
    for _, share := range in.ShareList.All() {
        candles, err := s.fetchWeeklyCandles(ctx, share.ID, in.Interval, dateNow)
        if err != nil { continue }

        sig, err := Detect(compactCandles(candles), share.RSILength, profile)
        if errors.Is(err, ErrAdaptiveInsufficientHistory) ||
           errors.Is(err, ErrInsufficientHistory) { continue }
        if err != nil { continue }

        results = append(results, ShareResult{
            ShareID:   share.ID,
            ShareName: share.Name,
            Signal:    sig,
            BuyTier:   tierFromAdaptive(sig.RSI, sig.Thresholds.P5, sig.Thresholds.P15),
            SellTier:  sellTierFromAdaptive(sig.RSI, sig.SellThresholds, profile),
        })
    }

    if len(results) > 0 {
        msg := notif.Trade(results, in.Kind)
        return s.tgClient.SendMessage(msg)
    }
    return nil
}
```

#### 4.3 notification.Trade() simplification

```go
// notification/notifications.go
func Trade(results []golden_x.ShareResult, kind dto.StrategyKind) string
```

Replaces the current 9-parameter signature. Each `ShareResult` carries its own signal with thresholds, trend, divergence, volume, and stop — no separate maps needed.

---

## Files Changed

| File | Action | Description |
|------|--------|-------------|
| `pipeline.go` | **new** | `Pipeline`, `DetectContext`, `Stage` interface |
| `stages.go` | **new** | 8 stage implementations |
| `profile.go` | **new** | `StrategyProfile`, `SellBehavior`, factory functions |
| `share_result.go` | **new** | `ShareResult` struct |
| `detector.go` | **modify** | `Detect()` delegates to Pipeline |
| `trade.go` | **modify** | Use `ShareResult[]`, call `Settings.Validate()` |
| `stop.go` | **modify** | `stopFromATR` uses `profile.ATRMultiplier`, remove `kForKind()` |
| `percentile.go` | **modify** | `sellTierFromAdaptive` accepts `*StrategyProfile` (uses `.SellBehavior`) instead of `dto.StrategyKind` |
| `dto/settings.go` | **modify** | Add `Validate()` method |
| `dto/signal.go` | **modify** | Add `Invariant()` method |
| `notification/notifications.go` | **modify** | Accept `[]ShareResult` instead of 9 params |
| `notification/notifications_test.go` | **modify** | Update test signatures |
| `backtest/replay.go` | **modify** | `DetectFunc` signature update, accept profile |
| `backtest/position.go` | **modify** | `EvaluateSellExits` uses `SellBehavior` |
| `settings.go` | **no change** | `DefaultSettings()` unchanged |

## What Does NOT Change

- Math functions: `computeRSISeries`, `percentile`, `bullishDivergence`, `computeEMA`, `stopFromATR` — signature and logic intact.
- `shares/shares.go` — instrument lists unchanged.
- `scheduler/trade.go` — decorator pattern already in place.
- `dto/trade.go`, `dto/strategy_kind.go`, `dto/thresholds.go`, `dto/sell_thresholds.go`, `dto/stop.go`, `dto/trend_status.go` — unchanged.

## Verification

1. **Unit tests**: all existing tests in `detector_test.go`, `divergence_test.go`, `rsi_test.go`, `percentile_test.go`, `stop_test.go`, `trend_filter_test.go`, `settings_test.go` must pass with identical output.
2. **Backtest parity**: run backtest with `DefaultSettings()` for both Dividend and Growth — results must match pre-refactoring output byte-for-byte.
3. **Invariant tests**: add test cases for `Settings.Validate()` (valid + invalid combos) and `Signal.Invariant()`.
4. **Pipeline composition test**: verify that `NewDetectPipeline` produces correct stage order for Dividend (no trend) vs Growth (with trend).
5. **Notification test**: verify `notification.Trade([]ShareResult, kind)` produces the same HTML as the current 9-param version.
6. **Compilation**: `go build ./...` and `go vet ./...` clean.
