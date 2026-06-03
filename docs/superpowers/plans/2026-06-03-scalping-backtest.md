# Scalping Backtest & Calibration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone `cmd/backtest` tool that replays the per-share scalping strategy over historical hourly candles, simulates trading on a mock portfolio, computes metrics, renders Markdown/CSV reports, and supports grid-search calibration of `rusal.Params`.

**Architecture:** A pure, fully table-tested core in `internal/domain/backtest` (types, replay engine, mock portfolio, metrics, report rendering) that mirrors the live runner's decision path (call `Decide` per bar, act only on `Buy`/`Sell`, fill at the bar `close`). All I/O (gRPC candle fetching with a file cache, flags, file writes) is isolated in `internal/service/backtest` and `cmd/backtest`. Calibration is a RUSAL-specific binding over the generic engine via a `ticker → (DefaultParams, Build, ParseParams)` registry.

**Tech Stack:** Go 1.25, Tinkoff Invest gRPC (`pkg/client/grpc`), existing indicators (`pkg/indicators`), `scalping/strategy` interface, `encoding/json`, `reflect` for grid application, table-driven tests.

---

## Reference facts (verified against the codebase)

These are exact signatures/behaviors the tasks below depend on. Do not re-derive them.

- Module path: `tinvest`.
- Strategy interface (`internal/service/trading_strategy/scalping/strategy/strategy.go`):
  ```go
  type Position struct { PurchasePrice float64; Quantity int64 }
  type MarketData struct {
      Price    float64
      Highs    []float64
      Lows     []float64
      Closes   []float64
      Volumes  []int64
      Position *Position // nil when flat
  }
  type Strategy interface {
      Ticker() string
      Lookback() int
      Decide(md MarketData) model.Signal
  }
  ```
- `model.Signal` (`scalping/model/signal.go`): fields `Kind SignalKind`, `Reason string`, `Price`, `TakeProfit`, `StopLoss`, `RSI`, `Ticker`. `SignalKind` constants: `SignalNone`, `SignalBuy`, `SignalSell`.
- RUSAL (`scalping/strategy/rusal/rusal.go`): `type Params struct {...}` (all exported `int`/`float64` fields), `func DefaultParams() Params`, `func NewWithParams(p Params) *Strategy`. Implements `strategy.Strategy`.
- gRPC market data (`pkg/client/grpc/market_data_service_client.go`):
  ```go
  GetCandles(ctx context.Context, instrumentUid *string, interval int32,
      from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool)
      ([]*model.CandleItemTechAnalyse, error)
  ```
  where `timestamp` is `github.com/golang/protobuf/ptypes/timestamp`. `timestamppb.New(t)` is assignable to `*timestamp.Timestamp` (type alias) — the live runner passes `timestamppb.New(...)` directly.
- `model.CandleItemTechAnalyse` (`internal/model/tech_analyse.go`): `Time time.Time`, `Open/Close/Low/High model.Quotation`, `Volume int64`, `IsComplete bool`. `model.Quotation{ Units int64; Nano int32 }`.
- `utils.CombinePrice(units int64, nano int32) float64` (`internal/utils/utils.go`).
- `enum.Interval` (`internal/enum/enum.go`): `Hour1 Interval = 4`, method `ToNumberInvestApi() int32`, `ToTimeDuration() time.Duration`, `String() string`.
- Instruments (`pkg/client/grpc/instruments_service_client.go`): `Shares(ctx) ([]*model.Share, error)`. `model.Share` has `Ticker string`, `ID string`, `Lot int32`, `Trading bool`, `Name string`.
- gRPC bootstrap: `grpc.NewClientGrpc(address, token string) (GrpcClient, error)` (`pkg/client/grpc/grpc.go`); address used elsewhere is `invest-public-api.tinkoff.ru:443`.
- Env loading uses `github.com/joho/godotenv`; token env var is `T_BANK` (`internal/config/grpc_client.go`).
- The stub to delete: `internal/domain/backtest/order_line.go` (`type OrderLine struct{}`).

---

## File structure

| File | Responsibility | I/O |
|---|---|---|
| `internal/domain/backtest/types.go` (create; replaces `order_line.go`) | `Candle`, `Config`, `Trade`, `EquityPoint`, `Result`, `Metrics`, `Meta`, `ParamLine` | no |
| `internal/domain/backtest/portfolio.go` (create) | mock portfolio: sizing, fills, commission, PnL | no |
| `internal/domain/backtest/engine.go` (create) | bar-by-bar replay, `Decide`, signal application | no |
| `internal/domain/backtest/metrics.go` (create) | metrics from trades + equity curve | no |
| `internal/domain/backtest/report.go` (create) | Markdown + trades/equity CSV rendering (returns strings) | no |
| `internal/service/backtest/candles.go` (create) | candle file cache, chunked Tinkoff fetch, merge/dedup, window slice | yes |
| `internal/service/backtest/registry.go` (create) | `ticker → Binding`, `ParamRows` reflection helper | no |
| `internal/service/backtest/calibrate.go` (create) | grid cartesian product, reflection apply, ranking, summary render | no |
| `cmd/backtest/main.go` (create) | flags, env+gRPC bootstrap, orchestration, file writes | yes |
| `.gitignore` (modify) | ignore `data/candles/` and `reports/` | n/a |

Each domain file is small and single-purpose. `engine.go` and `portfolio.go` share package `backtest`, so the engine can set the portfolio's bar counter directly.

---

## Task 1: Domain types (replace the stub)

**Files:**
- Delete: `internal/domain/backtest/order_line.go`
- Create: `internal/domain/backtest/types.go`

- [ ] **Step 1: Delete the stub**

```bash
git rm internal/domain/backtest/order_line.go
```

- [ ] **Step 2: Create the types file**

Note: `Result.BarsInMarket` is a deliberate addition to the spec's `Result` — the engine knows the position on every bar, so counting in-market bars there is exact (better than deriving exposure from closed trades).

```go
// Package backtest provides a pure, I/O-free replay engine, mock portfolio,
// metrics and report rendering for backtesting per-share scalping strategies.
package backtest

import "time"

// Candle is one OHLCV bar (series are oldest-first).
type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// Config controls the mock portfolio and fills.
type Config struct {
	InitialCash float64 // starting mock cash
	Fraction    float64 // fraction of current cash deployed per Buy (1.0 = all-in)
	Commission  float64 // commission as a fraction of turnover (e.g. 0.0005)
	Lot         int32   // share lot size (orders are whole lots)
}

// Trade is one completed round-trip (entry -> exit).
type Trade struct {
	EntryTime  time.Time
	EntryPrice float64
	ExitTime   time.Time
	ExitPrice  float64
	Quantity   int64   // shares (lots * Lot)
	Reason     string  // exit reason: "SL" / "TRAIL" / "TP"
	PnL        float64 // net of commission, in currency
	PnLPct     float64 // PnL relative to entry cost
	BarsHeld   int
}

// EquityPoint is portfolio value at one bar.
type EquityPoint struct {
	Time   time.Time
	Equity float64 // cash + position * close
}

// Result is the raw outcome of a single backtest run.
type Result struct {
	Trades       []Trade
	Equity       []EquityPoint
	InitialCash  float64
	FinalEquity  float64
	BarsInMarket int // bars with an open position (for exposure)
}

// Metrics are qualitative measures derived from a Result.
type Metrics struct {
	TotalTrades    int
	Wins, Losses   int
	WinRate        float64 // Wins/TotalTrades
	LossRate       float64
	GrossProfit    float64 // sum of positive PnL
	GrossLoss      float64 // sum of |negative PnL|
	ProfitFactor   float64 // GrossProfit/GrossLoss; if GrossLoss==0 and GrossProfit>0 -> GrossProfit
	NetPnL         float64 // FinalEquity - InitialCash
	NetPnLPct      float64
	MaxDrawdown    float64 // absolute, from the equity curve
	MaxDrawdownPct float64
	AvgWin         float64
	AvgLoss        float64
	Expectancy     float64 // average PnL per trade
	BestTrade      float64
	WorstTrade     float64
	ExposurePct    float64 // fraction of bars in market
	CAGR           float64 // annualized return over the run duration
}

// ParamLine is one strategy parameter rendered for the report header.
type ParamLine struct {
	Name  string
	Value string
}

// Meta is the report header context for a single run.
type Meta struct {
	Ticker       string
	Interval     string
	From         time.Time
	To           time.Time
	InitialCash  float64
	Fraction     float64
	Commission   float64
	Params       []ParamLine
	OpenPosition bool // a position was still open at the end
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./internal/domain/backtest/`
Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add internal/domain/backtest/types.go
git commit -m "feat(backtest): domain types; drop OrderLine stub"
```

---

## Task 2: Mock portfolio

**Files:**
- Create: `internal/domain/backtest/portfolio.go`
- Test: `internal/domain/backtest/portfolio_test.go`

The portfolio is unexported (engine-internal). Its `bar` field is set by the engine each iteration so `BarsHeld` is exact. Tests live in the same package and drive it directly.

- [ ] **Step 1: Write the failing test**

```go
package backtest

import (
	"math"
	"testing"
	"time"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestPortfolioOpenSizesByFractionAndLots(t *testing.T) {
	// cash 100000, fraction 0.5 -> budget 50000; price 100, lot 10,
	// commission 0 -> lotCost 1000; lots = floor(50) = 50; qty = 500.
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 0.5, Commission: 0, Lot: 10})
	p.open(100, time.Unix(0, 0))
	if p.qty != 500 {
		t.Fatalf("qty = %d, want 500", p.qty)
	}
	// cash spent = 500*100 = 50000.
	if !approx(p.cash, 50000) {
		t.Fatalf("cash = %f, want 50000", p.cash)
	}
}

func TestPortfolioOpenChargesCommission(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.001, Lot: 1})
	p.open(100, time.Unix(0, 0))
	// budget 100000; lotCost = 100*1*1.001 = 100.1; lots = floor(999.0) = 999.
	if p.qty != 999 {
		t.Fatalf("qty = %d, want 999", p.qty)
	}
	// cost 99900; commission 99.9; cash = 100000 - 99999.9 = 0.1.
	if !approx(p.cash, 0.1) {
		t.Fatalf("cash = %f, want 0.1", p.cash)
	}
}

func TestPortfolioRefusesEntryWhenCashTooSmallForALot(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 50, Fraction: 1.0, Commission: 0, Lot: 1})
	p.open(100, time.Unix(0, 0)) // one share costs 100 > 50
	if p.qty != 0 {
		t.Fatalf("qty = %d, want 0 (no entry)", p.qty)
	}
	if !approx(p.cash, 50) {
		t.Fatalf("cash = %f, want 50 (untouched)", p.cash)
	}
}

func TestPortfolioCloseComputesPnLAndBarsHeld(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.bar = 3
	p.open(100, time.Unix(3, 0)) // qty = 1000, cash = 0
	p.bar = 8
	tr := p.close(110, time.Unix(8, 0), "TP")
	// revenue 110000; entryCost 100000; PnL 10000; PnLPct 0.1.
	if !approx(tr.PnL, 10000) || !approx(tr.PnLPct, 0.1) {
		t.Fatalf("PnL=%f PnLPct=%f, want 10000 / 0.1", tr.PnL, tr.PnLPct)
	}
	if tr.BarsHeld != 5 {
		t.Fatalf("BarsHeld = %d, want 5", tr.BarsHeld)
	}
	if tr.Reason != "TP" || tr.Quantity != 1000 {
		t.Fatalf("Reason=%q Quantity=%d, want TP / 1000", tr.Reason, tr.Quantity)
	}
	if p.qty != 0 {
		t.Fatalf("qty = %d after close, want 0", p.qty)
	}
}

func TestPortfolioEquityMarksToMarket(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	p.open(100, time.Unix(0, 0)) // qty 1000, cash 0
	if !approx(p.equity(120), 120000) {
		t.Fatalf("equity = %f, want 120000", p.equity(120))
	}
}

func TestStrategyPositionNilWhenFlat(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if p.strategyPosition() != nil {
		t.Fatal("expected nil position when flat")
	}
	p.open(100, time.Unix(0, 0))
	pos := p.strategyPosition()
	if pos == nil || !approx(pos.PurchasePrice, 100) || pos.Quantity != 1000 {
		t.Fatalf("position = %+v, want {100, 1000}", pos)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestPortfolio -v`
Expected: FAIL — `undefined: newPortfolio`.

- [ ] **Step 3: Write the implementation**

```go
package backtest

import (
	"math"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// portfolio is the long-only mock account the engine trades against.
type portfolio struct {
	cfg        Config
	cash       float64
	qty        int64
	entryPrice float64
	entryTime  time.Time
	entryBar   int
	bar        int // current bar index, set by the engine each iteration
}

func newPortfolio(cfg Config) *portfolio {
	return &portfolio{cfg: cfg, cash: cfg.InitialCash}
}

// open deploys cfg.Fraction of cash into whole lots at price. No-op if already
// in a position or if there is not enough cash for a single lot.
func (p *portfolio) open(price float64, t time.Time) {
	if p.qty != 0 {
		return
	}
	lotCost := price * float64(p.cfg.Lot) * (1 + p.cfg.Commission)
	if lotCost <= 0 {
		return
	}
	budget := p.cfg.Fraction * p.cash
	lots := int64(math.Floor(budget / lotCost))
	if lots <= 0 {
		return
	}
	qty := lots * int64(p.cfg.Lot)
	cost := float64(qty) * price
	commission := cost * p.cfg.Commission
	p.cash -= cost + commission
	p.qty = qty
	p.entryPrice = price
	p.entryTime = t
	p.entryBar = p.bar
}

// close sells the whole position at price and returns the round-trip trade.
func (p *portfolio) close(price float64, t time.Time, reason string) Trade {
	revenue := float64(p.qty) * price
	commission := revenue * p.cfg.Commission
	p.cash += revenue - commission
	entryCost := float64(p.qty) * p.entryPrice * (1 + p.cfg.Commission)
	pnl := (revenue - commission) - entryCost
	pnlPct := 0.0
	if entryCost > 0 {
		pnlPct = pnl / entryCost
	}
	tr := Trade{
		EntryTime:  p.entryTime,
		EntryPrice: p.entryPrice,
		ExitTime:   t,
		ExitPrice:  price,
		Quantity:   p.qty,
		Reason:     reason,
		PnL:        pnl,
		PnLPct:     pnlPct,
		BarsHeld:   p.bar - p.entryBar,
	}
	p.qty = 0
	p.entryPrice = 0
	return tr
}

// strategyPosition exposes the open position to a strategy (nil when flat).
func (p *portfolio) strategyPosition() *strategy.Position {
	if p.qty == 0 {
		return nil
	}
	return &strategy.Position{PurchasePrice: p.entryPrice, Quantity: p.qty}
}

// equity is cash plus the position marked at price.
func (p *portfolio) equity(price float64) float64 {
	return p.cash + float64(p.qty)*price
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/backtest/ -run TestPortfolio -v`
Expected: PASS (all six tests).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/portfolio.go internal/domain/backtest/portfolio_test.go
git commit -m "feat(backtest): mock portfolio with lot sizing and commission"
```

---

## Task 3: Replay engine

**Files:**
- Create: `internal/domain/backtest/engine.go`
- Test: `internal/domain/backtest/engine_test.go`

The engine mirrors the live runner: per bar it builds `MarketData` from the lookback window, calls `Decide`, and acts only on `Buy` (when flat) / `Sell` (when in position), filling at that bar's `close`. Open positions at the end are marked-to-market, never force-closed.

- [ ] **Step 1: Write the failing test**

```go
package backtest

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// scriptedStrategy decides from MarketData via an injected function, so tests
// can drive Buy/Sell at specific bars by inspecting md.Price / md.Position.
type scriptedStrategy struct {
	lookback int
	decide   func(md strategy.MarketData) model.Signal
}

func (s scriptedStrategy) Ticker() string                       { return "TEST" }
func (s scriptedStrategy) Lookback() int                        { return s.lookback }
func (s scriptedStrategy) Decide(md strategy.MarketData) model.Signal { return s.decide(md) }

// flatCandles builds n bars at 1h steps; close[i] = closes[i], H/L = close, vol 1.
func flatCandles(closes []float64) []Candle {
	out := make([]Candle, len(closes))
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, c := range closes {
		out[i] = Candle{
			Time:   base.Add(time.Duration(i) * time.Hour),
			Open:   c, High: c, Low: c, Close: c, Volume: 1,
		}
	}
	return out
}

func TestEngineBuysFlatSellsInPosition(t *testing.T) {
	candles := flatCandles([]float64{10, 10, 100, 10, 110})
	// Lookback 1; buy when price==100 & flat, sell when price==110 & in position.
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy}
		}
		if md.Position != nil && md.Price == 110 {
			return model.Signal{Kind: model.SignalSell, Reason: "TP"}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if tr.EntryPrice != 100 || tr.ExitPrice != 110 || tr.Reason != "TP" {
		t.Fatalf("trade = %+v, want entry 100 exit 110 TP", tr)
	}
	if len(res.Equity) != 5 {
		t.Fatalf("equity points = %d, want 5", len(res.Equity))
	}
}

func TestEngineIgnoresBuyInPositionAndSellWhenFlat(t *testing.T) {
	candles := flatCandles([]float64{10, 100, 100, 100})
	// Always tries to buy; sell never triggers -> exactly one entry, no exit.
	buys := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		return model.Signal{Kind: model.SignalBuy}
	}}
	res := Run(buys, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 0 {
		t.Fatalf("trades = %d, want 0 (never sold)", len(res.Trades))
	}
	// A position is open: final equity != initial unless price flat across hold.
	if res.BarsInMarket == 0 {
		t.Fatal("expected some bars in market")
	}

	// Sell-when-flat must be ignored entirely.
	sells := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		return model.Signal{Kind: model.SignalSell, Reason: "SL"}
	}}
	res2 := Run(sells, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res2.Trades) != 0 || res2.BarsInMarket != 0 {
		t.Fatalf("sell-when-flat changed state: trades=%d inMarket=%d", len(res2.Trades), res2.BarsInMarket)
	}
}

func TestEngineEmptyWhenNotEnoughHistory(t *testing.T) {
	candles := flatCandles([]float64{10, 20})
	s := scriptedStrategy{lookback: 5, decide: func(md strategy.MarketData) model.Signal {
		return model.Signal{Kind: model.SignalBuy}
	}}
	res := Run(s, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 0 || len(res.Equity) != 0 {
		t.Fatalf("expected empty result, got %d trades %d equity", len(res.Trades), len(res.Equity))
	}
	if res.FinalEquity != 100000 {
		t.Fatalf("FinalEquity = %f, want 100000", res.FinalEquity)
	}
}

func TestEngineMarksOpenPositionToMarketAtEnd(t *testing.T) {
	candles := flatCandles([]float64{10, 100, 100, 200})
	// Buy at the first price==100 bar, never sell.
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	// qty = floor(100000/100) = 1000; cash 0; final close 200 -> equity 200000.
	if res.FinalEquity != 200000 {
		t.Fatalf("FinalEquity = %f, want 200000 (mark-to-market open position)", res.FinalEquity)
	}
}

func TestEngineWindowIsLookbackSized(t *testing.T) {
	candles := flatCandles([]float64{1, 2, 3, 4, 5})
	var seen int
	s := scriptedStrategy{lookback: 3, decide: func(md strategy.MarketData) model.Signal {
		seen = len(md.Closes) // must always equal lookback
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, Config{InitialCash: 1000, Fraction: 1.0, Lot: 1})
	if seen != 3 {
		t.Fatalf("window size = %d, want 3", seen)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestEngine -v`
Expected: FAIL — `undefined: Run`.

- [ ] **Step 3: Write the implementation**

```go
package backtest

import (
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// Run replays a strategy over oldest-first candles on a mock portfolio and
// returns the raw result. It mirrors the live runner: per bar it builds a
// lookback-sized MarketData, calls Decide, and acts only on Buy (when flat) or
// Sell (when in position), filling at the bar's close. An open position at the
// end is marked-to-market, never force-closed.
func Run(s strategy.Strategy, candles []Candle, cfg Config) Result {
	res := Result{InitialCash: cfg.InitialCash, FinalEquity: cfg.InitialCash}
	l := s.Lookback()
	if l <= 0 || len(candles) < l {
		return res
	}
	p := newPortfolio(cfg)
	lastClose := candles[len(candles)-1].Close
	for i := l - 1; i < len(candles); i++ {
		p.bar = i
		md := buildMarketData(candles[i-l+1 : i+1])
		md.Position = p.strategyPosition()

		c := candles[i]
		switch s.Decide(md).Kind {
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time)
			}
		case model.SignalSell:
			if p.qty != 0 {
				res.Trades = append(res.Trades, p.close(c.Close, c.Time, lastSignalReason(s, md)))
			}
		}

		res.Equity = append(res.Equity, EquityPoint{Time: c.Time, Equity: p.equity(c.Close)})
		if p.qty != 0 {
			res.BarsInMarket++
		}
		lastClose = c.Close
	}
	res.FinalEquity = p.equity(lastClose)
	return res
}

// lastSignalReason re-reads the sell reason from Decide. Decide is pure, so a
// second call on the same MarketData is deterministic; this keeps the switch
// above readable without stashing the Signal.
func lastSignalReason(s strategy.Strategy, md strategy.MarketData) string {
	return s.Decide(md).Reason
}

// buildMarketData converts an oldest-first window into a strategy snapshot,
// mirroring scalping/trade.go's buildMarketData.
func buildMarketData(window []Candle) strategy.MarketData {
	md := strategy.MarketData{
		Highs:   make([]float64, len(window)),
		Lows:    make([]float64, len(window)),
		Closes:  make([]float64, len(window)),
		Volumes: make([]int64, len(window)),
	}
	for i, c := range window {
		md.Highs[i] = c.High
		md.Lows[i] = c.Low
		md.Closes[i] = c.Close
		md.Volumes[i] = c.Volume
	}
	if n := len(window); n > 0 {
		md.Price = window[n-1].Close
	}
	return md
}
```

> Note on `lastSignalReason`: calling `Decide` twice is wasteful. Prefer capturing the signal once. **Use this cleaner body for the switch instead** (replace the `switch` block above with it):
>
> ```go
> sig := s.Decide(md)
> switch sig.Kind {
> case model.SignalBuy:
> 	if p.qty == 0 {
> 		p.open(c.Close, c.Time)
> 	}
> case model.SignalSell:
> 	if p.qty != 0 {
> 		res.Trades = append(res.Trades, p.close(c.Close, c.Time, sig.Reason))
> 	}
> }
> ```
>
> and delete `lastSignalReason`. The single-call form is correct and is what to implement; the two-call helper above is shown only to make the diff explicit — do not keep it.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/backtest/ -run TestEngine -v`
Expected: PASS (all five tests).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): bar-by-bar replay engine mirroring the live runner"
```

---

## Task 4: Metrics

**Files:**
- Create: `internal/domain/backtest/metrics.go`
- Test: `internal/domain/backtest/metrics_test.go`

`Compute` stays pure and takes `barsInMarket`, `totalBars`, `periodDays` explicitly (callers pass `r.BarsInMarket`, `len(r.Equity)`, and the candle span in days).

- [ ] **Step 1: Write the failing test**

```go
package backtest

import (
	"math"
	"testing"
	"time"
)

func eqCurve(vals []float64) []EquityPoint {
	out := make([]EquityPoint, len(vals))
	base := time.Unix(0, 0)
	for i, v := range vals {
		out[i] = EquityPoint{Time: base.Add(time.Duration(i) * time.Hour), Equity: v}
	}
	return out
}

func TestComputeBasicCounts(t *testing.T) {
	r := Result{
		Trades: []Trade{
			{PnL: 100}, {PnL: -40}, {PnL: 60}, {PnL: -20},
		},
		Equity:      eqCurve([]float64{1000, 1100, 1060, 1120, 1100}),
		InitialCash: 1000,
		FinalEquity: 1100,
	}
	m := Compute(r, 3, 5, 0)
	if m.TotalTrades != 4 || m.Wins != 2 || m.Losses != 2 {
		t.Fatalf("counts: total=%d wins=%d losses=%d", m.TotalTrades, m.Wins, m.Losses)
	}
	if !approx(m.WinRate, 0.5) || !approx(m.LossRate, 0.5) {
		t.Fatalf("rates: win=%f loss=%f", m.WinRate, m.LossRate)
	}
	if !approx(m.GrossProfit, 160) || !approx(m.GrossLoss, 60) {
		t.Fatalf("gross: profit=%f loss=%f", m.GrossProfit, m.GrossLoss)
	}
	if !approx(m.ProfitFactor, 160.0/60.0) {
		t.Fatalf("profit factor = %f", m.ProfitFactor)
	}
	if !approx(m.NetPnL, 100) || !approx(m.NetPnLPct, 0.1) {
		t.Fatalf("net: pnl=%f pct=%f", m.NetPnL, m.NetPnLPct)
	}
	if !approx(m.Expectancy, 25) { // (160-60)/4
		t.Fatalf("expectancy = %f, want 25", m.Expectancy)
	}
	if !approx(m.AvgWin, 80) || !approx(m.AvgLoss, 30) {
		t.Fatalf("avg: win=%f loss=%f", m.AvgWin, m.AvgLoss)
	}
	if !approx(m.BestTrade, 100) || !approx(m.WorstTrade, -40) {
		t.Fatalf("best/worst: %f / %f", m.BestTrade, m.WorstTrade)
	}
	if !approx(m.ExposurePct, 0.6) { // 3/5
		t.Fatalf("exposure = %f, want 0.6", m.ExposurePct)
	}
}

func TestComputeProfitFactorNoLosses(t *testing.T) {
	r := Result{Trades: []Trade{{PnL: 50}, {PnL: 30}}, InitialCash: 1000, FinalEquity: 1080}
	m := Compute(r, 0, 0, 0)
	if !approx(m.ProfitFactor, 80) { // GrossLoss==0, GrossProfit>0 -> GrossProfit
		t.Fatalf("profit factor = %f, want 80", m.ProfitFactor)
	}
}

func TestComputeMaxDrawdown(t *testing.T) {
	// peak 1200 at idx 2, trough 900 at idx 4 -> DD 300, pct 0.25.
	r := Result{Equity: eqCurve([]float64{1000, 1100, 1200, 1000, 900, 1050}), InitialCash: 1000, FinalEquity: 1050}
	m := Compute(r, 0, 6, 0)
	if !approx(m.MaxDrawdown, 300) || !approx(m.MaxDrawdownPct, 0.25) {
		t.Fatalf("drawdown: abs=%f pct=%f, want 300 / 0.25", m.MaxDrawdown, m.MaxDrawdownPct)
	}
}

func TestComputeCAGR(t *testing.T) {
	// double in 365 days -> CAGR = 1.0.
	r := Result{InitialCash: 1000, FinalEquity: 2000}
	m := Compute(r, 0, 0, 365)
	if !approx(m.CAGR, 1.0) {
		t.Fatalf("CAGR = %f, want 1.0", m.CAGR)
	}
	// periodDays 0 -> CAGR 0 (no divide).
	if c := Compute(r, 0, 0, 0).CAGR; c != 0 {
		t.Fatalf("CAGR with 0 days = %f, want 0", c)
	}
	_ = math.Pi
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestCompute -v`
Expected: FAIL — `undefined: Compute`.

- [ ] **Step 3: Write the implementation**

```go
package backtest

import "math"

// Compute derives metrics from a Result. barsInMarket/totalBars/periodDays are
// supplied by the caller (typically r.BarsInMarket, len(r.Equity), and the
// candle span in days) so Compute stays pure.
func Compute(r Result, barsInMarket, totalBars int, periodDays float64) Metrics {
	var m Metrics
	m.TotalTrades = len(r.Trades)
	for i, t := range r.Trades {
		if i == 0 {
			m.BestTrade, m.WorstTrade = t.PnL, t.PnL
		} else {
			if t.PnL > m.BestTrade {
				m.BestTrade = t.PnL
			}
			if t.PnL < m.WorstTrade {
				m.WorstTrade = t.PnL
			}
		}
		if t.PnL >= 0 {
			m.Wins++
			m.GrossProfit += t.PnL
		} else {
			m.Losses++
			m.GrossLoss += -t.PnL
		}
	}
	if m.TotalTrades > 0 {
		m.WinRate = float64(m.Wins) / float64(m.TotalTrades)
		m.LossRate = float64(m.Losses) / float64(m.TotalTrades)
		m.Expectancy = (m.GrossProfit - m.GrossLoss) / float64(m.TotalTrades)
	}
	if m.Wins > 0 {
		m.AvgWin = m.GrossProfit / float64(m.Wins)
	}
	if m.Losses > 0 {
		m.AvgLoss = m.GrossLoss / float64(m.Losses)
	}
	switch {
	case m.GrossLoss > 0:
		m.ProfitFactor = m.GrossProfit / m.GrossLoss
	case m.GrossProfit > 0:
		m.ProfitFactor = m.GrossProfit
	default:
		m.ProfitFactor = 0
	}
	m.NetPnL = r.FinalEquity - r.InitialCash
	if r.InitialCash > 0 {
		m.NetPnLPct = m.NetPnL / r.InitialCash
	}
	m.MaxDrawdown, m.MaxDrawdownPct = maxDrawdown(r.Equity)
	if totalBars > 0 {
		m.ExposurePct = float64(barsInMarket) / float64(totalBars)
	}
	if periodDays > 0 && r.InitialCash > 0 && r.FinalEquity > 0 {
		m.CAGR = math.Pow(r.FinalEquity/r.InitialCash, 365.0/periodDays) - 1
	}
	return m
}

// maxDrawdown returns the largest peak-to-trough drop on the equity curve, both
// absolute and as a fraction of the running peak.
func maxDrawdown(eq []EquityPoint) (abs, pct float64) {
	var peak float64
	for i, p := range eq {
		if i == 0 || p.Equity > peak {
			peak = p.Equity
		}
		dd := peak - p.Equity
		if dd > abs {
			abs = dd
			if peak > 0 {
				pct = dd / peak
			}
		}
	}
	return abs, pct
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/backtest/ -run TestCompute -v`
Expected: PASS (all four tests).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/metrics.go internal/domain/backtest/metrics_test.go
git commit -m "feat(backtest): metrics — win rate, profit factor, drawdown, CAGR"
```

---

## Task 5: Report rendering

**Files:**
- Create: `internal/domain/backtest/report.go`
- Test: `internal/domain/backtest/report_test.go`

Pure functions returning strings; the CLI writes them to disk.

- [ ] **Step 1: Write the failing test**

```go
package backtest

import (
	"strings"
	"testing"
	"time"
)

func sampleMeta() Meta {
	return Meta{
		Ticker:      "RUAL",
		Interval:    "Hour1",
		From:        time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		To:          time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		InitialCash: 100000,
		Fraction:    1.0,
		Commission:  0.0005,
		Params:      []ParamLine{{Name: "EMAPeriod", Value: "21"}, {Name: "ADXPeriod", Value: "14"}},
	}
}

func TestRenderMarkdownHasSections(t *testing.T) {
	m := Metrics{TotalTrades: 2, Wins: 1, Losses: 1, WinRate: 0.5, ProfitFactor: 1.5, NetPnL: 1000}
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
	}}
	eq := eqCurve([]float64{100000, 101000})
	out := RenderMarkdown(sampleMeta(), m, trades, eq)
	for _, want := range []string{"RUAL", "EMAPeriod", "Сводка метрик", "Журнал сделок", "Движение капитала", "TP"} {
		if !strings.Contains(out, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
}

func TestRenderTradesCSVHeaderAndRow(t *testing.T) {
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
	}}
	out := RenderTradesCSV(trades)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("csv lines = %d, want 2 (header + 1 row)", len(lines))
	}
	if !strings.HasPrefix(lines[0], "idx,entry_time,entry_price") {
		t.Fatalf("unexpected header: %q", lines[0])
	}
	if !strings.Contains(lines[1], "TP") {
		t.Fatalf("row missing reason: %q", lines[1])
	}
}

func TestRenderEquityCSV(t *testing.T) {
	out := RenderEquityCSV(eqCurve([]float64{100, 101, 102}))
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 { // header + 3 points
		t.Fatalf("equity csv lines = %d, want 4", len(lines))
	}
	if lines[0] != "time,equity" {
		t.Fatalf("header = %q, want time,equity", lines[0])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestRender -v`
Expected: FAIL — `undefined: RenderMarkdown`.

- [ ] **Step 3: Write the implementation**

```go
package backtest

import (
	"fmt"
	"strings"
	"time"
)

const tsLayout = "2006-01-02 15:04"

// RenderMarkdown renders the full single-run report as a Markdown string.
func RenderMarkdown(meta Meta, m Metrics, trades []Trade, equity []EquityPoint) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Бэктест %s (%s)\n\n", meta.Ticker, meta.Interval)
	fmt.Fprintf(&b, "- Период: %s — %s\n", meta.From.Format(tsLayout), meta.To.Format(tsLayout))
	fmt.Fprintf(&b, "- Стартовый кэш: %.2f\n", meta.InitialCash)
	fmt.Fprintf(&b, "- Fraction: %.4g; Commission: %.4g\n", meta.Fraction, meta.Commission)
	if meta.OpenPosition {
		b.WriteString("- ⚠️ На конце прогона осталась открытая позиция (оценена mark-to-market)\n")
	}
	b.WriteString("\n## Параметры стратегии\n\n| Параметр | Значение |\n|---|---|\n")
	for _, p := range meta.Params {
		fmt.Fprintf(&b, "| %s | %s |\n", p.Name, p.Value)
	}

	b.WriteString("\n## Сводка метрик\n\n| Метрика | Значение |\n|---|---|\n")
	fmt.Fprintf(&b, "| Всего сделок | %d |\n", m.TotalTrades)
	fmt.Fprintf(&b, "| Выигрышных / проигрышных | %d / %d |\n", m.Wins, m.Losses)
	fmt.Fprintf(&b, "| Win rate | %.2f%% |\n", m.WinRate*100)
	fmt.Fprintf(&b, "| Gross profit / loss | %.2f / %.2f |\n", m.GrossProfit, m.GrossLoss)
	fmt.Fprintf(&b, "| Profit factor | %.3f |\n", m.ProfitFactor)
	fmt.Fprintf(&b, "| Чистый PnL | %.2f (%.2f%%) |\n", m.NetPnL, m.NetPnLPct*100)
	fmt.Fprintf(&b, "| Стартовый / финальный капитал | %.2f / %.2f |\n", meta.InitialCash, meta.InitialCash+m.NetPnL)
	fmt.Fprintf(&b, "| Макс. просадка | %.2f (%.2f%%) |\n", m.MaxDrawdown, m.MaxDrawdownPct*100)
	fmt.Fprintf(&b, "| Средняя прибыль / убыток | %.2f / %.2f |\n", m.AvgWin, m.AvgLoss)
	fmt.Fprintf(&b, "| Expectancy | %.2f |\n", m.Expectancy)
	fmt.Fprintf(&b, "| Лучшая / худшая сделка | %.2f / %.2f |\n", m.BestTrade, m.WorstTrade)
	fmt.Fprintf(&b, "| Exposure | %.2f%% |\n", m.ExposurePct*100)
	fmt.Fprintf(&b, "| CAGR | %.2f%% |\n", m.CAGR*100)

	b.WriteString("\n## Журнал сделок\n\n| № | Вход | Цена входа | Выход | Цена выхода | Причина | Баров | PnL | PnL %% |\n|---|---|---|---|---|---|---|---|---|\n")
	for i, t := range trades {
		fmt.Fprintf(&b, "| %d | %s | %.4f | %s | %.4f | %s | %d | %.2f | %.2f%% |\n",
			i+1, t.EntryTime.Format(tsLayout), t.EntryPrice, t.ExitTime.Format(tsLayout),
			t.ExitPrice, t.Reason, t.BarsHeld, t.PnL, t.PnLPct*100)
	}

	b.WriteString("\n## Движение капитала\n\n")
	if len(equity) == 0 {
		b.WriteString("Нет данных equity.\n")
	} else {
		minEq, maxEq := equity[0].Equity, equity[0].Equity
		for _, p := range equity {
			if p.Equity < minEq {
				minEq = p.Equity
			}
			if p.Equity > maxEq {
				maxEq = p.Equity
			}
		}
		fmt.Fprintf(&b, "- Старт: %.2f\n- Мин: %.2f\n- Макс: %.2f\n- Финал: %.2f\n",
			equity[0].Equity, minEq, maxEq, equity[len(equity)-1].Equity)
		b.WriteString("\nПолная кривая — в `*_equity.csv`.\n")
	}
	return b.String()
}

// RenderTradesCSV renders the trade journal as CSV.
func RenderTradesCSV(trades []Trade) string {
	var b strings.Builder
	b.WriteString("idx,entry_time,entry_price,exit_time,exit_price,qty,reason,pnl,pnl_pct,bars_held\n")
	for i, t := range trades {
		fmt.Fprintf(&b, "%d,%s,%.6f,%s,%.6f,%d,%s,%.6f,%.6f,%d\n",
			i+1, t.EntryTime.UTC().Format(time.RFC3339), t.EntryPrice,
			t.ExitTime.UTC().Format(time.RFC3339), t.ExitPrice, t.Quantity,
			t.Reason, t.PnL, t.PnLPct, t.BarsHeld)
	}
	return b.String()
}

// RenderEquityCSV renders the equity curve as CSV.
func RenderEquityCSV(equity []EquityPoint) string {
	var b strings.Builder
	b.WriteString("time,equity\n")
	for _, p := range equity {
		fmt.Fprintf(&b, "%s,%.6f\n", p.Time.UTC().Format(time.RFC3339), p.Equity)
	}
	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/backtest/ -v`
Expected: PASS (all domain tests, including report).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/report.go internal/domain/backtest/report_test.go
git commit -m "feat(backtest): Markdown + trades/equity CSV rendering"
```

---

## Task 6: Candle provider (file cache + chunked fetch)

**Files:**
- Create: `internal/service/backtest/candles.go`
- Test: `internal/service/backtest/candles_test.go`

The provider holds a narrow `candleFetcher` interface (satisfied by the real gRPC client) so tests inject a fake. Pure helpers (`convertCandles`, `mergeCandles`, `sliceWindow`) are tested directly; `Load` is tested on `t.TempDir()` with the fake.

- [ ] **Step 1: Write the failing test**

```go
package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/model"
)

func qt(v float64) model.Quotation { return model.Quotation{Units: int64(v), Nano: 0} }

// fakeFetcher returns canned candles and records call count.
type fakeFetcher struct {
	candles []*model.CandleItemTechAnalyse
	calls   int
}

func (f *fakeFetcher) GetCandles(_ context.Context, _ *string, _ int32,
	_ *timestamp.Timestamp, _ *timestamp.Timestamp, _ *int32, _ bool,
) ([]*model.CandleItemTechAnalyse, error) {
	f.calls++
	return f.candles, nil
}

func bar(tm time.Time, c float64, complete bool) *model.CandleItemTechAnalyse {
	return &model.CandleItemTechAnalyse{
		Time: tm, Open: qt(c), High: qt(c), Low: qt(c), Close: qt(c), Volume: 1, IsComplete: complete,
	}
}

func TestConvertCandlesDropsIncomplete(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	in := []*model.CandleItemTechAnalyse{
		bar(base, 10, true),
		bar(base.Add(time.Hour), 11, false), // dropped
	}
	out := convertCandles(in)
	if len(out) != 1 || out[0].Close != 10 {
		t.Fatalf("convert = %+v, want 1 complete bar @10", out)
	}
}

func TestMergeCandlesDedupAndSort(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := []backtest.Candle{{Time: base.Add(time.Hour), Close: 2}, {Time: base, Close: 1}}
	b := []backtest.Candle{{Time: base.Add(time.Hour), Close: 99}, {Time: base.Add(2 * time.Hour), Close: 3}}
	out := mergeCandles(a, b)
	if len(out) != 3 {
		t.Fatalf("merged len = %d, want 3", len(out))
	}
	if !out[0].Time.Equal(base) || !out[1].Time.Equal(base.Add(time.Hour)) || !out[2].Time.Equal(base.Add(2*time.Hour)) {
		t.Fatalf("not sorted: %+v", out)
	}
	if out[1].Close != 2 { // first occurrence wins on dup Time
		t.Fatalf("dedup kept wrong value: %f, want 2", out[1].Close)
	}
}

func TestSliceWindow(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var c []backtest.Candle
	for i := 0; i < 5; i++ {
		c = append(c, backtest.Candle{Time: base.Add(time.Duration(i) * time.Hour)})
	}
	got := sliceWindow(c, base.Add(time.Hour), base.Add(3*time.Hour))
	if len(got) != 3 { // inclusive [1h, 3h]
		t.Fatalf("window len = %d, want 3", len(got))
	}
}

func TestLoadNoFileFetchesAndCaches(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f := &fakeFetcher{candles: []*model.CandleItemTechAnalyse{
		bar(base, 10, true),
		bar(base.Add(time.Hour), 11, true),
		bar(base.Add(2*time.Hour), 12, true),
	}}
	p := NewCandleProvider(f, t.TempDir())
	got, err := p.Load(context.Background(), "RUAL", "id-1", enum.Hour1, base, base.Add(2*time.Hour), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("loaded %d candles, want 3", len(got))
	}
	if f.calls == 0 {
		t.Fatal("expected at least one fetch on cold cache")
	}
	// Second load (no refresh): cache file exists; last cached == to, so no new
	// tail fetch is required.
	callsAfterFirst := f.calls
	if _, err := p.Load(context.Background(), "RUAL", "id-1", enum.Hour1, base, base.Add(2*time.Hour), false); err != nil {
		t.Fatal(err)
	}
	if f.calls != callsAfterFirst {
		t.Fatalf("warm cache refetched: calls %d -> %d", callsAfterFirst, f.calls)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'TestConvert|TestMerge|TestSlice|TestLoad' -v`
Expected: FAIL — `undefined: convertCandles` / `NewCandleProvider`.

- [ ] **Step 3: Write the implementation**

```go
// Package backtest wires the pure backtest engine to real candle data: a
// file-cached, chunked Tinkoff fetcher plus a strategy registry and grid
// calibration. All gRPC/file I/O lives here; the engine itself stays pure.
package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

// chunkDays bounds each GetCandles request window; H1 over a long range must be
// fetched in pieces. fetchPause throttles requests to respect API limits.
const (
	chunkDays  = 30
	fetchPause = 300 * time.Millisecond
)

// candleFetcher is the slice of the gRPC market-data client the provider needs.
// The real grpc.MarketDataServiceClient satisfies it.
type candleFetcher interface {
	GetCandles(ctx context.Context, instrumentUid *string, interval int32,
		from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool,
	) ([]*model.CandleItemTechAnalyse, error)
}

// CandleProvider loads candles for (ticker, interval), caching them in one JSON
// file per pair and topping up the tail on demand.
type CandleProvider struct {
	client candleFetcher
	dir    string // cache directory, e.g. data/candles
}

func NewCandleProvider(client candleFetcher, dir string) *CandleProvider {
	return &CandleProvider{client: client, dir: dir}
}

// Load returns oldest-first candles in [from, to]. With a warm cache it reads
// the file and only fetches the missing tail; refresh forces a full refetch.
func (p *CandleProvider) Load(ctx context.Context, ticker, instrumentID string,
	interval enum.Interval, from, to time.Time, refresh bool,
) ([]backtest.Candle, error) {
	path := p.cachePath(ticker, interval)

	if refresh {
		fetched, err := p.fetchRange(ctx, instrumentID, interval, from, to)
		if err != nil {
			return nil, err
		}
		if err := p.writeCache(path, fetched); err != nil {
			return nil, err
		}
		return sliceWindow(fetched, from, to), nil
	}

	cached, err := p.readCache(path)
	if err != nil {
		return nil, err
	}
	if len(cached) == 0 {
		fetched, ferr := p.fetchRange(ctx, instrumentID, interval, from, to)
		if ferr != nil {
			return nil, ferr
		}
		if werr := p.writeCache(path, fetched); werr != nil {
			return nil, werr
		}
		return sliceWindow(fetched, from, to), nil
	}

	last := cached[len(cached)-1].Time
	if last.Before(to) {
		tail, ferr := p.fetchRange(ctx, instrumentID, interval, last, to)
		if ferr != nil {
			return nil, ferr
		}
		cached = mergeCandles(cached, tail)
		if werr := p.writeCache(path, cached); werr != nil {
			return nil, werr
		}
	}
	return sliceWindow(cached, from, to), nil
}

// fetchRange pulls [from, to] in chunkDays windows, converting and merging.
func (p *CandleProvider) fetchRange(ctx context.Context, instrumentID string,
	interval enum.Interval, from, to time.Time,
) ([]backtest.Candle, error) {
	var all []backtest.Candle
	id := instrumentID
	num := interval.ToNumberInvestApi()
	for winFrom := from; winFrom.Before(to); {
		winTo := winFrom.Add(chunkDays * 24 * time.Hour)
		if winTo.After(to) {
			winTo = to
		}
		items, err := p.client.GetCandles(ctx, &id, num,
			timestamppb.New(winFrom), timestamppb.New(winTo), nil, true)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("backtest: candle chunk %s-%s failed: %v", winFrom, winTo, err))
		} else {
			all = mergeCandles(all, convertCandles(items))
		}
		winFrom = winTo
		time.Sleep(fetchPause)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("backtest: no candles fetched for %s in [%s, %s]", instrumentID, from, to)
	}
	return all, nil
}

func (p *CandleProvider) cachePath(ticker string, interval enum.Interval) string {
	return filepath.Join(p.dir, fmt.Sprintf("%s_%s.json", ticker, interval.String()))
}

func (p *CandleProvider) readCache(path string) ([]backtest.Candle, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("backtest: read cache %s: %w", path, err)
	}
	var out []backtest.Candle
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("backtest: parse cache %s: %w", path, err)
	}
	return out, nil
}

func (p *CandleProvider) writeCache(path string, candles []backtest.Candle) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("backtest: mkdir cache dir: %w", err)
	}
	data, err := json.MarshalIndent(candles, "", "  ")
	if err != nil {
		return fmt.Errorf("backtest: marshal cache: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("backtest: write cache %s: %w", path, err)
	}
	return nil
}

// convertCandles maps complete API candles to domain candles (drops the
// still-forming last bar).
func convertCandles(items []*model.CandleItemTechAnalyse) []backtest.Candle {
	out := make([]backtest.Candle, 0, len(items))
	for _, c := range items {
		if !c.IsComplete {
			continue
		}
		out = append(out, backtest.Candle{
			Time:   c.Time,
			Open:   utils.CombinePrice(c.Open.Units, c.Open.Nano),
			High:   utils.CombinePrice(c.High.Units, c.High.Nano),
			Low:    utils.CombinePrice(c.Low.Units, c.Low.Nano),
			Close:  utils.CombinePrice(c.Close.Units, c.Close.Nano),
			Volume: c.Volume,
		})
	}
	return out
}

// mergeCandles concatenates two series, dedups by Time (first occurrence wins)
// and returns them sorted oldest-first.
func mergeCandles(a, b []backtest.Candle) []backtest.Candle {
	seen := make(map[int64]struct{}, len(a)+len(b))
	var out []backtest.Candle
	for _, src := range [][]backtest.Candle{a, b} {
		for _, c := range src {
			key := c.Time.UnixNano()
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// sliceWindow returns the candles whose Time is within [from, to] inclusive.
func sliceWindow(candles []backtest.Candle, from, to time.Time) []backtest.Candle {
	var out []backtest.Candle
	for _, c := range candles {
		if c.Time.Before(from) || c.Time.After(to) {
			continue
		}
		out = append(out, c)
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/backtest/ -run 'TestConvert|TestMerge|TestSlice|TestLoad' -v`
Expected: PASS (all five tests).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/candles.go internal/service/backtest/candles_test.go
git commit -m "feat(backtest): cached, chunked candle provider"
```

---

## Task 7: Strategy registry

**Files:**
- Create: `internal/service/backtest/registry.go`
- Test: `internal/service/backtest/registry_test.go`

`Binding` decouples the generic engine from RUSAL specifics. `ParamRows` reflects a `Params` value into `[]backtest.ParamLine` for the report header (used by single runs and calibration).

- [ ] **Step 1: Write the failing test**

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

func TestLookupKnownAndUnknown(t *testing.T) {
	if _, ok := Lookup("NOPE"); ok {
		t.Fatal("expected unknown ticker to miss")
	}
	b, ok := Lookup("RUAL")
	if !ok {
		t.Fatal("expected RUAL binding")
	}
	if b.DefaultParams == nil || b.Build == nil || b.ParseParams == nil {
		t.Fatal("binding has nil funcs")
	}
}

func TestRUALBindingBuildsStrategy(t *testing.T) {
	b, _ := Lookup("RUAL")
	def := b.DefaultParams()
	s := b.Build(def)
	if s.Ticker() != "RUAL" {
		t.Fatalf("built strategy ticker = %q, want RUAL", s.Ticker())
	}
}

func TestRUALParseParamsOverridesDefaults(t *testing.T) {
	b, _ := Lookup("RUAL")
	raw := []byte(`{"EMAPeriod": 50}`)
	parsed, err := b.ParseParams(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := parsed.(rusal.Params)
	if p.EMAPeriod != 50 {
		t.Fatalf("EMAPeriod = %d, want 50 (override)", p.EMAPeriod)
	}
	if p.ADXPeriod != rusal.DefaultParams().ADXPeriod {
		t.Fatal("non-overridden field should keep its default")
	}
}

func TestParamRows(t *testing.T) {
	rows := ParamRows(rusal.DefaultParams())
	if len(rows) == 0 {
		t.Fatal("expected param rows")
	}
	var found bool
	for _, r := range rows {
		if r.Name == "EMAPeriod" && r.Value == "21" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected EMAPeriod=21 row")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'TestLookup|TestRUAL|TestParamRows' -v`
Expected: FAIL — `undefined: Lookup`.

- [ ] **Step 3: Write the implementation**

```go
package backtest

import (
	"encoding/json"
	"fmt"
	"reflect"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

// Binding adapts a concrete strategy's params to the generic engine: it builds
// the strategy from params, supplies defaults, and parses params from JSON.
type Binding struct {
	DefaultParams func() any                         // e.g. rusal.DefaultParams()
	Build         func(params any) strategy.Strategy // e.g. rusal.NewWithParams(p)
	ParseParams   func(raw []byte) (any, error)      // JSON -> rusal.Params
}

var registry = map[string]Binding{
	"RUAL": {
		DefaultParams: func() any { return rusal.DefaultParams() },
		Build:         func(params any) strategy.Strategy { return rusal.NewWithParams(params.(rusal.Params)) },
		ParseParams: func(raw []byte) (any, error) {
			p := rusal.DefaultParams() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse params: %w", err)
			}
			return p, nil
		},
	},
}

// Lookup returns the binding registered for a ticker.
func Lookup(ticker string) (Binding, bool) {
	b, ok := registry[ticker]
	return b, ok
}

// ParamRows reflects a params struct into report rows (field name -> value).
func ParamRows(params any) []backtest.ParamLine {
	v := reflect.ValueOf(params)
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	rows := make([]backtest.ParamLine, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		rows = append(rows, backtest.ParamLine{
			Name:  t.Field(i).Name,
			Value: fmt.Sprintf("%v", v.Field(i).Interface()),
		})
	}
	return rows
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/backtest/ -run 'TestLookup|TestRUAL|TestParamRows' -v`
Expected: PASS (all four tests).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/registry.go internal/service/backtest/registry_test.go
git commit -m "feat(backtest): ticker->strategy registry with RUAL binding"
```

---

## Task 8: Grid calibration

**Files:**
- Create: `internal/service/backtest/calibrate.go`
- Test: `internal/service/backtest/calibrate_test.go`

`RunGrid` takes a binding's defaults, applies each grid combination via reflection (int and float fields), runs the engine, computes metrics, and ranks by the chosen metric. `RenderCalibrationMarkdown` is a pure string renderer for the summary (best run uses the domain `RenderMarkdown`).

- [ ] **Step 1: Write the failing test**

```go
package backtest

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

func tinyCandles(n int) []backtest.Candle {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]backtest.Candle, n)
	for i := 0; i < n; i++ {
		price := 100.0 + float64(i%7)
		out[i] = backtest.Candle{
			Time: base.Add(time.Duration(i) * time.Hour),
			Open: price, High: price + 1, Low: price - 1, Close: price, Volume: 100,
		}
	}
	return out
}

func TestApplyFieldIntAndFloat(t *testing.T) {
	p := rusal.DefaultParams()
	updated, err := applyField(p, "EMAPeriod", 50) // int field
	if err != nil {
		t.Fatal(err)
	}
	if updated.(rusal.Params).EMAPeriod != 50 {
		t.Fatalf("EMAPeriod = %d, want 50", updated.(rusal.Params).EMAPeriod)
	}
	updated2, err := applyField(p, "SLMult", 2.5) // float64 field
	if err != nil {
		t.Fatal(err)
	}
	if updated2.(rusal.Params).SLMult != 2.5 {
		t.Fatalf("SLMult = %f, want 2.5", updated2.(rusal.Params).SLMult)
	}
}

func TestApplyFieldUnknownErrors(t *testing.T) {
	if _, err := applyField(rusal.DefaultParams(), "Nonexistent", 1); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestRunGridCartesianProduct(t *testing.T) {
	b, _ := Lookup("RUAL")
	grid := Grid{
		"EMAPeriod": {12, 21},
		"SLMult":    {1.0, 1.5, 2.0},
	}
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	results, err := RunGrid(b, grid, tinyCandles(400), cfg, "profit_factor", 16)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 6 { // 2 * 3
		t.Fatalf("combos = %d, want 6", len(results))
	}
}

func TestRunGridRanksByMetric(t *testing.T) {
	// Hand-built results to exercise ranking only.
	in := []CalibResult{
		{Metrics: backtest.Metrics{ProfitFactor: 1.2, MaxDrawdown: 500}},
		{Metrics: backtest.Metrics{ProfitFactor: 2.5, MaxDrawdown: 900}},
		{Metrics: backtest.Metrics{ProfitFactor: 0.8, MaxDrawdown: 100}},
	}
	byPF := rankResults(append([]CalibResult(nil), in...), "profit_factor")
	if byPF[0].Metrics.ProfitFactor != 2.5 {
		t.Fatalf("top PF = %f, want 2.5", byPF[0].Metrics.ProfitFactor)
	}
	byDD := rankResults(append([]CalibResult(nil), in...), "max_drawdown")
	if byDD[0].Metrics.MaxDrawdown != 100 { // smaller is better
		t.Fatalf("top DD = %f, want 100", byDD[0].Metrics.MaxDrawdown)
	}
}

func TestRenderCalibrationMarkdown(t *testing.T) {
	results := []CalibResult{
		{Params: rusal.DefaultParams(), Metrics: backtest.Metrics{ProfitFactor: 2.0, NetPnL: 1000, TotalTrades: 5}},
	}
	out := RenderCalibrationMarkdown("profit_factor", results, 10)
	if !strings.Contains(out, "profit_factor") || !strings.Contains(out, "Калибровка") {
		t.Fatalf("calibration markdown missing headers:\n%s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'TestApplyField|TestRunGrid|TestRender' -v`
Expected: FAIL — `undefined: applyField`.

- [ ] **Step 3: Write the implementation**

```go
package backtest

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"tinvest/internal/domain/backtest"
)

// Grid maps a Params field name to the values to sweep over it. Fields not
// listed keep their default.
type Grid map[string][]float64

// CalibResult pairs a parameter combination with its run metrics.
type CalibResult struct {
	Params  any
	Metrics backtest.Metrics
}

// RunGrid runs the engine for every combination in the grid and returns the
// results ranked by metric (best first). periodDays feeds CAGR.
func RunGrid(b Binding, grid Grid, candles []backtest.Candle, cfg backtest.Config,
	metric string, periodDays float64,
) ([]CalibResult, error) {
	if err := validateMetric(metric); err != nil {
		return nil, err
	}
	combos, err := expandGrid(b.DefaultParams(), grid)
	if err != nil {
		return nil, err
	}
	results := make([]CalibResult, 0, len(combos))
	for _, params := range combos {
		res := backtest.Run(b.Build(params), candles, cfg)
		m := backtest.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)
		results = append(results, CalibResult{Params: params, Metrics: m})
	}
	return rankResults(results, metric), nil
}

// expandGrid builds the cartesian product of the grid, applying each field over
// a copy of the default params.
func expandGrid(defaults any, grid Grid) ([]any, error) {
	combos := []any{defaults}
	// Stable field order for deterministic output.
	names := make([]string, 0, len(grid))
	for name := range grid {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		values := grid[name]
		var next []any
		for _, base := range combos {
			for _, v := range values {
				updated, err := applyField(base, name, v)
				if err != nil {
					return nil, err
				}
				next = append(next, updated)
			}
		}
		combos = next
	}
	return combos, nil
}

// applyField returns a copy of params with field `name` set to `value`,
// converting to the field's int or float kind. Unknown/unsettable fields error.
func applyField(params any, name string, value float64) (any, error) {
	v := reflect.ValueOf(params)
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("backtest: params is not a struct")
	}
	out := reflect.New(v.Type()).Elem()
	out.Set(v)
	f := out.FieldByName(name)
	if !f.IsValid() {
		return nil, fmt.Errorf("backtest: unknown grid field %q", name)
	}
	if !f.CanSet() {
		return nil, fmt.Errorf("backtest: field %q is not settable", name)
	}
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f.SetInt(int64(value))
	case reflect.Float32, reflect.Float64:
		f.SetFloat(value)
	default:
		return nil, fmt.Errorf("backtest: field %q has unsupported kind %s", name, f.Kind())
	}
	return out.Interface(), nil
}

// metricValue extracts the ranking key for a metric (already validated).
func metricValue(m backtest.Metrics, metric string) float64 {
	switch metric {
	case "net_pnl":
		return m.NetPnL
	case "win_rate":
		return m.WinRate
	case "max_drawdown":
		return m.MaxDrawdown
	case "expectancy":
		return m.Expectancy
	default: // profit_factor
		return m.ProfitFactor
	}
}

// rankResults sorts best-first: ascending for max_drawdown, descending otherwise.
func rankResults(results []CalibResult, metric string) []CalibResult {
	sort.SliceStable(results, func(i, j int) bool {
		a, b := metricValue(results[i].Metrics, metric), metricValue(results[j].Metrics, metric)
		if metric == "max_drawdown" {
			return a < b
		}
		return a > b
	})
	return results
}

var supportedMetrics = map[string]struct{}{
	"profit_factor": {}, "net_pnl": {}, "win_rate": {}, "max_drawdown": {}, "expectancy": {},
}

func validateMetric(metric string) error {
	if _, ok := supportedMetrics[metric]; !ok {
		return fmt.Errorf("backtest: unknown metric %q (want profit_factor|net_pnl|win_rate|max_drawdown|expectancy)", metric)
	}
	return nil
}

// RenderCalibrationMarkdown renders the top-N combinations as a Markdown table.
func RenderCalibrationMarkdown(metric string, results []CalibResult, topN int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Калибровка (ранжирование по %s)\n\n", metric)
	fmt.Fprintf(&b, "Всего комбинаций: %d. Топ-%d:\n\n", len(results), topN)
	b.WriteString("| # | Метрика | Profit factor | Net PnL | Win rate | Max DD | Сделок |\n|---|---|---|---|---|---|---|\n")
	for i, r := range results {
		if i >= topN {
			break
		}
		m := r.Metrics
		fmt.Fprintf(&b, "| %d | %.4g | %.3f | %.2f | %.2f%% | %.2f | %d |\n",
			i+1, metricValue(m, metric), m.ProfitFactor, m.NetPnL, m.WinRate*100, m.MaxDrawdown, m.TotalTrades)
	}
	if len(results) > 0 {
		b.WriteString("\n## Лучшая комбинация — параметры\n\n| Параметр | Значение |\n|---|---|\n")
		for _, row := range ParamRows(results[0].Params) {
			fmt.Fprintf(&b, "| %s | %s |\n", row.Name, row.Value)
		}
	}
	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/backtest/ -v`
Expected: PASS (all service tests).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/calibrate.go internal/service/backtest/calibrate_test.go
git commit -m "feat(backtest): grid calibration with reflection apply and ranking"
```

---

## Task 9: CLI entry point + gitignore

**Files:**
- Create: `cmd/backtest/main.go`
- Modify: `.gitignore`

The CLI is the only I/O orchestrator: parse flags, load env + gRPC, resolve the share, load candles, run a single backtest or a calibration, and write report files. No unit tests (pure logic is already covered); it is validated by `go build` and a `go vet`.

- [ ] **Step 1: Update `.gitignore`**

Append these lines to `.gitignore` (the file currently ends at `/cache` with no trailing newline — add a newline first):

```
data/candles/
reports/
```

After editing, `.gitignore` should contain (tail):

```
qdrant_data
/cache
data/candles/
reports/
```

- [ ] **Step 2: Write the CLI**

```go
// Command backtest replays the per-share scalping strategy over historical
// candles, simulates a mock portfolio, and writes Markdown + CSV reports.
// It supports a single run (default params or -params) and grid calibration
// (-calibrate). All gRPC/file I/O is here; the engine and metrics are pure.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	domain "tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	grpcclient "tinvest/pkg/client/grpc"
)

const (
	apiAddress = "invest-public-api.tinkoff.ru:443"
	cacheDir   = "data/candles"
)

func main() {
	var (
		ticker     = flag.String("ticker", "", "ticker, e.g. RUAL (required)")
		months     = flag.Int("months", 12, "lookback period in months")
		cash       = flag.Float64("cash", 100000, "starting mock cash")
		fraction   = flag.Float64("fraction", 1.0, "fraction of cash per Buy")
		commission = flag.Float64("commission", 0.0005, "commission as a fraction of turnover")
		paramsPath = flag.String("params", "", "path to JSON Params (default: DefaultParams)")
		calibrate  = flag.String("calibrate", "", "path to grid JSON (grid-search mode)")
		metric     = flag.String("metric", "profit_factor", "ranking metric for calibration")
		outDir     = flag.String("out", "reports", "report output directory")
		refresh    = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
	)
	flag.Parse()

	if err := run(*ticker, *months, *cash, *fraction, *commission,
		*paramsPath, *calibrate, *metric, *outDir, *refresh); err != nil {
		log.Fatalf("backtest: %v", err)
	}
}

func run(ticker string, months int, cash, fraction, commission float64,
	paramsPath, calibratePath, metric, outDir string, refresh bool,
) error {
	if ticker == "" {
		return fmt.Errorf("-ticker is required")
	}
	if paramsPath != "" && calibratePath != "" {
		return fmt.Errorf("-params and -calibrate are mutually exclusive")
	}

	token, err := loadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(apiAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}

	ctx := context.Background()
	binding, ok := svc.Lookup(ticker)
	if !ok {
		return fmt.Errorf("no strategy binding registered for ticker %q", ticker)
	}

	share, err := resolveShare(ctx, client, ticker)
	if err != nil {
		return err
	}

	to := time.Now()
	from := to.AddDate(0, -months, 0)
	interval := enum.Hour1

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	candles, err := provider.Load(ctx, ticker, share.ID, interval, from, to, refresh)
	if err != nil {
		return err
	}

	cfg := domain.Config{InitialCash: cash, Fraction: fraction, Commission: commission, Lot: share.Lot}
	periodDays := to.Sub(from).Hours() / 24

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	base := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s", ticker, interval.String(), stamp))

	if calibratePath != "" {
		return runCalibration(binding, calibratePath, candles, cfg, metric, periodDays, base,
			metaCommon(ticker, interval, from, to, cfg))
	}
	return runSingle(binding, paramsPath, candles, cfg, periodDays, base,
		metaCommon(ticker, interval, from, to, cfg))
}

func runSingle(b svc.Binding, paramsPath string, candles []domain.Candle, cfg domain.Config,
	periodDays float64, base string, meta domain.Meta,
) error {
	params := b.DefaultParams()
	if paramsPath != "" {
		raw, err := os.ReadFile(paramsPath)
		if err != nil {
			return fmt.Errorf("read params: %w", err)
		}
		params, err = b.ParseParams(raw)
		if err != nil {
			return err
		}
	}

	if len(candles) < b.Build(params).Lookback() {
		fmt.Printf("⚠️ not enough candles (%d) for lookback; empty report\n", len(candles))
	}

	res := domain.Run(b.Build(params), candles, cfg)
	m := domain.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)

	meta.Params = svc.ParamRows(params)
	meta.OpenPosition = openPosition(res)

	if err := writeFile(base+".md", domain.RenderMarkdown(meta, m, res.Trades, res.Equity)); err != nil {
		return err
	}
	if err := writeFile(base+"_trades.csv", domain.RenderTradesCSV(res.Trades)); err != nil {
		return err
	}
	if err := writeFile(base+"_equity.csv", domain.RenderEquityCSV(res.Equity)); err != nil {
		return err
	}
	fmt.Printf("report: %s.md (trades=%d, net=%.2f, PF=%.3f)\n", base, m.TotalTrades, m.NetPnL, m.ProfitFactor)
	return nil
}

func runCalibration(b svc.Binding, gridPath string, candles []domain.Candle, cfg domain.Config,
	metric string, periodDays float64, base string, meta domain.Meta,
) error {
	raw, err := os.ReadFile(gridPath)
	if err != nil {
		return fmt.Errorf("read grid: %w", err)
	}
	var grid svc.Grid
	if err := jsonUnmarshal(raw, &grid); err != nil {
		return fmt.Errorf("parse grid: %w", err)
	}

	results, err := svc.RunGrid(b, grid, candles, cfg, metric, periodDays)
	if err != nil {
		return err
	}
	calibPath := base + "_calibration.md"
	if err := writeFile(calibPath, svc.RenderCalibrationMarkdown(metric, results, 20)); err != nil {
		return err
	}

	// Also emit the full single-run report for the best combination.
	if len(results) > 0 {
		best := results[0].Params
		res := domain.Run(b.Build(best), candles, cfg)
		m := domain.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)
		meta.Params = svc.ParamRows(best)
		meta.OpenPosition = openPosition(res)
		if err := writeFile(base+"_best.md", domain.RenderMarkdown(meta, m, res.Trades, res.Equity)); err != nil {
			return err
		}
	}
	fmt.Printf("calibration: %s (combos=%d)\n", calibPath, len(results))
	return nil
}

func metaCommon(ticker string, interval enum.Interval, from, to time.Time, cfg domain.Config) domain.Meta {
	return domain.Meta{
		Ticker:      ticker,
		Interval:    interval.String(),
		From:        from,
		To:          to,
		InitialCash: cfg.InitialCash,
		Fraction:    cfg.Fraction,
		Commission:  cfg.Commission,
	}
}

func openPosition(res domain.Result) bool {
	// A position is open at the end iff no trade closed on the final bar AND the
	// engine held shares; equivalently FinalEquity differs from pure cash only
	// when in market. Simplest robust check: bars in market on the last point.
	if len(res.Equity) == 0 {
		return false
	}
	// If the count of closed trades' bars is less than bars in market, a tail
	// position remained open.
	var closedBars int
	for _, t := range res.Trades {
		closedBars += t.BarsHeld
	}
	return res.BarsInMarket > closedBars
}

func resolveShare(ctx context.Context, client grpcclient.GrpcClient, ticker string) (shareInfo, error) {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return shareInfo{}, fmt.Errorf("load shares: %w", err)
	}
	for _, s := range shares {
		if s.Ticker == ticker {
			if !s.Trading {
				return shareInfo{}, fmt.Errorf("share %s is not trading", ticker)
			}
			return shareInfo{ID: s.ID, Lot: s.Lot}, nil
		}
	}
	return shareInfo{}, fmt.Errorf("ticker %q not found in Shares()", ticker)
}

type shareInfo struct {
	ID  string
	Lot int32
}

func loadToken() (string, error) {
	_ = godotenv.Load("./env/local.env")
	_ = godotenv.Load("./env/token.env")
	token := os.Getenv("T_BANK")
	if token == "" {
		return "", fmt.Errorf("T_BANK is not set (checked env + ./env/local.env, ./env/token.env)")
	}
	return token, nil
}

func writeFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
```

- [ ] **Step 3: Add the JSON import helper**

The CLI references `jsonUnmarshal`. Add `encoding/json` to the import block and replace the `jsonUnmarshal(raw, &grid)` call with `json.Unmarshal(raw, &grid)` directly (do not define a wrapper). Concretely: add `"encoding/json"` to imports and change the calibration parse line to:

```go
	if err := json.Unmarshal(raw, &grid); err != nil {
		return fmt.Errorf("parse grid: %w", err)
	}
```

- [ ] **Step 4: Verify it builds and vets**

Run: `go build ./cmd/backtest/ && go vet ./cmd/backtest/ ./internal/service/backtest/ ./internal/domain/backtest/`
Expected: no output (success). If the compiler reports an unused import or the `jsonUnmarshal` symbol, fix per Step 3.

- [ ] **Step 5: Run the full test suite for the new packages**

Run: `go test ./internal/domain/backtest/ ./internal/service/backtest/`
Expected: `ok` for both packages.

- [ ] **Step 6: Commit**

```bash
git add cmd/backtest/main.go .gitignore
git commit -m "feat(backtest): cmd/backtest CLI for single runs and calibration"
```

---

## Final verification

- [ ] **Step 1: Build everything**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 2: Vet and test everything new**

Run: `go vet ./cmd/backtest/ ./internal/service/backtest/ ./internal/domain/backtest/ && go test ./internal/domain/backtest/ ./internal/service/backtest/`
Expected: `ok` for both test packages.

- [ ] **Step 3: Smoke-run help**

Run: `go run ./cmd/backtest -h`
Expected: usage text listing all flags (`-ticker`, `-months`, `-calibrate`, etc.). No panic.

> A real backtest run (`go run ./cmd/backtest -ticker RUAL -months 12`) requires a valid `T_BANK` token and network access; it is a manual check for the user, not part of automated verification.

---

## Self-review notes (addressed)

- **Spec coverage:** types/portfolio/engine/metrics/report (domain) — Tasks 1–5; candle cache + chunked fetch — Task 6; registry — Task 7; grid calibration + ranking + metric validation — Task 8; CLI flags, env/gRPC bootstrap, share resolution, report writing, `.gitignore`, stub deletion — Task 9 + Task 1. Error handling (missing token, ticker not found/not trading, insufficient history warning, unknown grid field/metric) covered in Tasks 8–9.
- **`Result.BarsInMarket`:** deliberate, documented extension of the spec's `Result` for exact exposure.
- **`engine.go` two-call note:** the plan flags `lastSignalReason` as a deliberate teaching artifact and instructs the worker to implement the single-`Decide` form and delete the helper.
- **`openPosition` heuristic:** derived from `BarsInMarket` vs summed `BarsHeld`; correct because a still-open tail adds to `BarsInMarket` without contributing a closed trade.
- **Type consistency:** `domain.Config/Result/Metrics/Meta/ParamLine`, `svc.Binding/Grid/CalibResult`, `applyField`/`rankResults`/`metricValue`/`expandGrid` names match across Tasks 4, 7, 8, 9.
