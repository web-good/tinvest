# RUSAL adaptive scalping strategy — design

Date: 2026-06-03
Branch: `feat/per-share-scalping-strategies`

## Problem

The current RUSAL strategy (`internal/service/trading_strategy/scalping/strategy/rusal`)
is a single-regime trend-follower: an EMA200 trend filter plus an upward RSI reversal
entry, with fixed ATR-based take-profit / stop-loss. It has one behaviour regardless of
what the market is actually doing.

RUAL on the hourly timeframe is a mid-liquidity cyclical MOEX share (driven by aluminium
prices, USD/RUB, news/sanctions). On H1 it tends to spend most of its time **range-bound**
with **episodic trending bursts**. A single-regime rule is wrong half the time: a
trend-follower bleeds in the range, a mean-reverter gets run over in the trend.

We want an **adaptive, regime-aware** strategy: detect whether the market is trending or
ranging, and switch entry/exit logic accordingly. Long-only, hourly, reusing the existing
`strategy.Strategy` contract.

> **Honesty note.** These rules and default parameters were NOT calibrated against live
> RUAL candle data — no market-data access was available in the design session, and the
> repo backtest harness (`internal/domain/backtest`) is a stub. The regime split and the
> indicator framework are standard, principled technical-analysis choices; the concrete
> numbers are **standard starting values that must be calibrated on real history.** They
> are therefore exposed as an external `Params` struct, not hard-coded constants.

## Decisions (locked in)

- **Hybrid with a regime filter.** Detect regime, then run mean-reversion in a range and
  momentum in a trend.
- **Regime detector: real ADX/DMI.** Add a pure Wilder ADX helper (with DI+/DI−) to
  `pkg/indicators`. DI+/DI− also give trend direction.
- **Long-only.** The `Signal`/`Position` model and the runner are unchanged.
- **Regime-dependent exits.** Range → mean-reversion target at the channel mid; Trend →
  ATR chandelier trailing stop. Both keep a hard initial ATR stop.
- **Trend entry: pullback to EMA + RSI reversal up** (gated by `ADX ≥ trendLevel` and
  `DI+ > DI−`).
- **Range entry: lower Donchian band + RSI reversal up** (gated by `ADX ≤ rangeLevel`).
- **`Decide` stays stateless** (re-evaluated each hour on fresh candles). Trailing/targets
  are computed from the current candle window, not from per-trade memory.
- **All knobs are externalised** into a `Params` struct with `DefaultParams()`, so they
  can be calibrated later without touching logic.
- The `strategy.Strategy` contract, `MarketData`, `Position`, `Signal`, the registry and
  the runner (`trade.go`) are **unchanged**.

## Architecture

The runner owns all I/O. The RUSAL package computes indicators from the raw candle
window handed to it and delegates the actual decision to a pure core.

```
pkg/indicators/
  adx.go          # ADX(highs, lows, closes, period) (adx, diPlus, diMinus float64)  [NEW]
  adx_test.go     #                                                                  [NEW]
  donchian.go     # Donchian(highs, lows, period) (upper, lower float64)             [NEW]
  donchian_test.go#                                                                  [NEW]

scalping/strategy/rusal/
  rusal.go        # Params + DefaultParams + Strategy + New + Decide + pure decide   [REWRITE]
  rusal_test.go   # helper-independent table tests over decide + e2e Decide          [REWRITE]
```

### New pure helpers (`pkg/indicators`)

Both mirror the existing `ATR`/`VolumeConfirmed` conventions: pure, no I/O, return the
**last** value(s), and silently return zeros on insufficient history or mismatched slice
lengths (no error).

**`ADX(highs, lows, closes []float64, period int) (adx, diPlus, diMinus float64)`** —
Wilder's Average Directional Index:
- Per bar `i ≥ 1`: `+DM = max(High_i − High_{i-1}, 0)` if it exceeds `−DM = max(Low_{i-1} − Low_i, 0)` else 0; symmetric for `−DM`. `TR` as in `ATR`.
- Wilder-smooth `TR`, `+DM`, `−DM` over `period`.
- `+DI = 100 · smoothed(+DM)/smoothed(TR)`, `−DI` symmetric.
- `DX = 100 · |+DI − −DI| / (+DI + −DI)`.
- `ADX` = Wilder average of `DX` over `period`.
- Returns last `(ADX, +DI, −DI)`. Returns `(0,0,0)` when `period ≤ 0`, slices differ in
  length, or `len(closes) < 2*period + 1` (needs warmup for the double smoothing).

**`Donchian(highs, lows []float64, period int) (upper, lower float64)`** —
`upper = max(High over last period)`, `lower = min(Low over last period)`. Returns
`(0,0)` when `period ≤ 0`, slices differ in length, or `len < period`. Channel mid is
derived by the caller as `(upper+lower)/2`.

### RUSAL strategy (`rusal.go`)

```go
type Params struct {
    EMAPeriod        int     // 21   fast EMA for trend pullback
    ADXPeriod        int     // 14
    ADXTrendLevel    float64 // 25   ADX >= -> trend
    ADXRangeLevel    float64 // 20   ADX <= -> range (between = dead zone)
    RSIPeriod        int     // 14
    RSITrendLevel    float64 // 45   RSI reversal threshold in trend (shallow pullbacks)
    RSIRangeLevel    float64 // 35   RSI reversal threshold in range (oversold)
    PullbackWindow   int     // 5    bars back over which an EMA "touch" still counts
    DonchianPeriod   int     // 20
    ATRPeriod        int     // 14
    SLMult           float64 // 1.0  initial stop = entry - SLMult*ATR
    TrailMult        float64 // 2.5  chandelier = max(High over window) - TrailMult*ATR
    ChandelierWindow int     // 20
    EMATouchTol      float64 // 0.002 EMA "touch" tolerance (0.2%)
    BandTol          float64 // 0.003 lower-band proximity tolerance (0.3%)
}

func DefaultParams() Params { /* the values above */ }

type Strategy struct{ p Params }

func New() *Strategy                 // DefaultParams()
func NewWithParams(p Params) *Strategy
```

`Ticker()` returns `"RUAL"`. `Lookback()` returns `6*ADXPeriod + DonchianPeriod + 50`
(≈ **154** with defaults) — sized to warm up ADX's double smoothing, the most
history-hungry indicator.

`Decide(md)` computes from `md`: `ema = ema.Compute(closes, EMAPeriod)`,
`rsi = indicators.RSISeries(closes, RSIPeriod)`, `atr = indicators.ATR(...)`,
`adx, diPlus, diMinus = indicators.ADX(...)`,
`upper, lower = indicators.Donchian(highs, lows, DonchianPeriod)`, plus a recent-window
EMA-touch flag and the chandelier high. It packs these scalars into the pure core.

### Pure decision core

```go
func (s *Strategy) decide(in decideInput) model.Signal
```

`decideInput` carries already-computed scalars/flags (price, atr, emaNow, rsiPrev,
rsiNow, adx, diPlus, diMinus, donchianUpper, donchianLower, emaTouched, chandelierHigh,
pos). No indicator math inside — trivially table-testable.

**Regime** from `adx`:
- `adx ≥ ADXTrendLevel` → **trend**
- `adx ≤ ADXRangeLevel` → **range**
- otherwise → **dead zone** (no new entry; an open position is managed as range)

**Entry (pos == nil):**
- **Trend & `diPlus > diMinus`:** `emaTouched` (on at least one of the last
  `PullbackWindow` bars, `Low ≤ ema*(1+EMATouchTol)` — price dipped to the EMA) AND RSI
  crossed up (`rsiPrev < RSITrendLevel && rsiNow ≥ RSITrendLevel`) AND `price > emaNow`
  (trend intact) → **Buy**.
- **Range:** `price ≤ donchianLower*(1+BandTol)` AND RSI crossed up
  (`rsiPrev < RSIRangeLevel && rsiNow ≥ RSIRangeLevel`) → **Buy**.
- On buy: `StopLoss = price − SLMult*ATR`; `TakeProfit` set as an orientation
  (range → channel mid; trend → an indicative `price + TrailMult*ATR`).
- Dead zone: no entry.

**Exit (pos != nil), by current regime:**
- **Range / dead zone:** `price ≥ mid` (mid `= (upper+lower)/2`) → **Sell** `Reason="TP"`;
  `price ≤ pos.PurchasePrice − SLMult*ATR` → **Sell** `Reason="SL"`.
- **Trend:** `chandelier = chandelierHigh − TrailMult*ATR`; `price ≤ chandelier` →
  **Sell** `Reason="TRAIL"`; `price ≤ pos.PurchasePrice − SLMult*ATR` → **Sell**
  `Reason="SL"`.
- Otherwise hold (`SignalNone`).

`Decide` stamps `sig.Ticker = "RUAL"` on the result, as today.

### Statelessness

`entry` price comes from `pos.PurchasePrice`. The chandelier high is a rolling
`max(Highs over ChandelierWindow)` — a standard N-bar chandelier exit, not a true
since-entry maximum, so no per-trade memory is needed. Regime is re-evaluated every bar;
a position opened in a trend that later decays to a range is correctly handed to the
range exit (take profit near the band), which is desirable.

## Testing

- **`ADX`:** a fixture series against reference `(ADX, +DI, −DI)`; `(0,0,0)` on short
  history and on mismatched slice lengths.
- **`Donchian`:** a fixture against known max/min; `(0,0)` on short history / length
  mismatch.
- **`decide` core (table-driven, ~11 cases):** trend entry (touch + RSI cross + DI+),
  trend no-pullback → None, trend `DI+ < DI−` → None, range entry at lower band, range
  mid-channel → None, dead zone flat → None, range exit TP(mid), range exit SL, trend
  exit TRAIL, trend exit initial SL, trend hold while rising → None.
- **`Decide` end-to-end:** a small-period deterministic `Params` driving each regime
  through real indicator computation, asserting `Ticker == "RUAL"` and the expected kind.

## Out of scope / follow-ups

- **Parameter calibration on real RUAL history.** Requires either a candle export fed
  into a sizing/backtest pass, or building out the empty `internal/domain/backtest`
  harness. The `Params` struct exists precisely to make this a values-only change.
- Short selling, breakout (Donchian-high + volume) trend entry as an optional second
  entry, multi-share rollout.
