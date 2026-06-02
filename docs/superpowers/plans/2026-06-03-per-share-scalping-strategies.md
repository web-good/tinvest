# Per-share Scalping Strategies Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the dynamic top-N volatility universe scan in the scalping strategy with a fixed registry of per-share strategies, each owning its own signal logic, starting with RUSAL.

**Architecture:** A `strategy` package defines a pure `Strategy` interface (`Ticker`, `Lookback`, `Decide(MarketData) Signal`) plus a `MarketData` snapshot of raw candle series. The runner (`scalping.service.Trade`) does all I/O — loads shares, candles, positions, sends Telegram — and delegates the entry/exit decision to each registered strategy. Strategies compute their own indicators from raw candles via the pure helpers in `pkg/indicators` and `internal/domain/ema`. The first strategy, RUSAL, ports the existing EMA200 + RSI-reversal + ATR-stop logic.

**Tech Stack:** Go 1.25, existing pure indicator helpers (`pkg/indicators.ATR`, `pkg/indicators.RSISeries`, `internal/domain/ema.Compute`), standard `testing` table-driven tests.

**Spec:** `docs/superpowers/specs/2026-06-03-per-share-scalping-strategies-design.md`

---

## File Structure

- Create `internal/service/trading_strategy/scalping/strategy/strategy.go` — `Strategy` interface, `MarketData`, `Position`.
- Create `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go` — RUSAL strategy.
- Create `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go` — table-driven tests.
- Create `internal/service/trading_strategy/scalping/registry.go` — `defaultStrategies()` enabled list (lives in the runner package to avoid a `strategy → rusal → strategy` import cycle).
- Modify `internal/service/trading_strategy/scalping/model/settings.go` — trim `Settings` to `Interval`.
- Modify `internal/service/trading_strategy/scalping/types.go` — drop EMA/RSI/ATR deps, add `strategies` field.
- Modify `internal/service/trading_strategy/scalping/trade.go` — rewrite `Trade` to loop the registry.
- Delete `internal/service/trading_strategy/scalping/decide.go`, `decide_test.go`, and the `universe/` package.
- Modify `internal/service_provider/service.go:215` — update `NewService` call.

---

## Task 1: Strategy contract package

**Files:**
- Create: `internal/service/trading_strategy/scalping/strategy/strategy.go`

- [ ] **Step 1: Write the contract types**

```go
package strategy

import "tinvest/internal/service/trading_strategy/scalping/model"

// Position is an open long position in the strategy's instrument.
type Position struct {
	PurchasePrice float64
	Quantity      int64
}

// MarketData is the raw, per-instrument snapshot the runner hands to a strategy.
// All series are oldest-first and aligned to the same candles; Price is the last close.
type MarketData struct {
	Price    float64
	Highs    []float64
	Lows     []float64
	Closes   []float64
	Volumes  []int64
	Position *Position // nil when flat
}

// Strategy is the per-share trading rule. Decide must be pure: it computes its own
// indicators from md and performs no I/O.
type Strategy interface {
	Ticker() string                  // e.g. "RUAL"
	Lookback() int                   // number of candles it needs
	Decide(md MarketData) model.Signal
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./internal/service/trading_strategy/scalping/strategy/...`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/strategy.go
git commit -m "feat(scalping): add per-share Strategy contract"
```

---

## Task 2: RUSAL strategy (TDD)

**Files:**
- Create: `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`
- Test: `internal/service/trading_strategy/scalping/strategy/rusal/rusal_test.go`

The strategy has a pure decision core (`decide`, tested directly for exact TP/SL) and a
public `Decide` that computes EMA/RSI/ATR from candles and calls the core. Tests cover
the core for precise math and `Decide` for indicator-wiring on the position and flat paths.

- [ ] **Step 1: Write the failing tests**

```go
package rusal

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestDecideCore(t *testing.T) {
	s := New() // rsiReversalLevel 35, tpMult 1.5, slMult 1.0

	tests := []struct {
		name       string
		price, atr float64
		aboveEMA   bool
		rsiPrev    float64
		rsiNow     float64
		pos        *strategy.Position
		wantKind   model.SignalKind
		wantTP     float64
		wantSL     float64
		wantReason string
	}{
		{
			name: "buy on trend + rsi reversal",
			price: 100, atr: 2, aboveEMA: true, rsiPrev: 30, rsiNow: 36,
			wantKind: model.SignalBuy, wantTP: 103, wantSL: 98,
		},
		{
			name: "no buy when below ema",
			price: 100, atr: 2, aboveEMA: false, rsiPrev: 30, rsiNow: 36,
			wantKind: model.SignalNone,
		},
		{
			name: "no buy when rsi did not cross upward",
			price: 100, atr: 2, aboveEMA: true, rsiPrev: 36, rsiNow: 40,
			wantKind: model.SignalNone,
		},
		{
			name: "sell on take profit",
			price: 104, atr: 2, pos: &strategy.Position{PurchasePrice: 100},
			wantKind: model.SignalSell, wantTP: 103, wantSL: 98, wantReason: "TP",
		},
		{
			name: "sell on stop loss",
			price: 97, atr: 2, pos: &strategy.Position{PurchasePrice: 100},
			wantKind: model.SignalSell, wantTP: 103, wantSL: 98, wantReason: "SL",
		},
		{
			name: "hold position inside the band",
			price: 101, atr: 2, pos: &strategy.Position{PurchasePrice: 100},
			wantKind: model.SignalNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.decide(tt.price, tt.atr, tt.aboveEMA, tt.rsiPrev, tt.rsiNow, tt.pos)
			if got.Kind != tt.wantKind {
				t.Fatalf("Kind = %v, want %v", got.Kind, tt.wantKind)
			}
			if tt.wantKind == model.SignalNone {
				return
			}
			if got.TakeProfit != tt.wantTP {
				t.Errorf("TakeProfit = %v, want %v", got.TakeProfit, tt.wantTP)
			}
			if got.StopLoss != tt.wantSL {
				t.Errorf("StopLoss = %v, want %v", got.StopLoss, tt.wantSL)
			}
			if got.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
		})
	}
}

// smallStrategy uses tiny indicator periods so synthetic candle series stay short.
func smallStrategy() *Strategy {
	return &Strategy{
		emaPeriod: 3, rsiPeriod: 2, atrPeriod: 2,
		rsiReversalLevel: 35, tpMult: 1.5, slMult: 1.0,
	}
}

func TestDecide_SellOnTakeProfit(t *testing.T) {
	s := smallStrategy()
	closes := []float64{100, 101, 102, 103, 104, 105}
	md := strategy.MarketData{
		Price:   200, // far above any TP -> deterministic sell
		Highs:   []float64{100, 101, 102, 103, 104, 105},
		Lows:    []float64{99, 100, 101, 102, 103, 104},
		Closes:  closes,
		Position: &strategy.Position{PurchasePrice: 100, Quantity: 1},
	}
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "TP" {
		t.Fatalf("got Kind=%v Reason=%q, want Sell/TP", got.Kind, got.Reason)
	}
	if got.Ticker != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", got.Ticker)
	}
}

func TestDecide_FlatNoCrossIsNone(t *testing.T) {
	s := smallStrategy()
	// Monotonic uptrend keeps RSI high -> no upward cross through 35 -> no buy.
	closes := []float64{10, 11, 12, 13, 14, 15}
	md := strategy.MarketData{
		Price:  15,
		Highs:  []float64{10, 11, 12, 13, 14, 15},
		Lows:   []float64{9, 10, 11, 12, 13, 14},
		Closes: closes,
	}
	got := s.Decide(md)
	if got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", got.Kind)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/...`
Expected: FAIL — `undefined: New`, `undefined: Strategy`.

- [ ] **Step 3: Write the implementation**

```go
package rusal

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

const ticker = "RUAL"

// Strategy trades RUSAL with an EMA trend filter, an upward RSI reversal entry,
// and ATR-based take-profit / stop-loss.
type Strategy struct {
	emaPeriod        int
	rsiPeriod        int
	atrPeriod        int
	rsiReversalLevel float64
	tpMult           float64
	slMult           float64
}

// New returns the RUSAL strategy with its default knobs.
func New() *Strategy {
	return &Strategy{
		emaPeriod:        200,
		rsiPeriod:        14,
		atrPeriod:        14,
		rsiReversalLevel: 35,
		tpMult:           1.5,
		slMult:           1.0,
	}
}

func (s *Strategy) Ticker() string { return ticker }

// Lookback is the candle count needed to seed the EMA plus headroom.
func (s *Strategy) Lookback() int { return s.emaPeriod + 50 }

// Decide computes the indicators from md and applies the RUSAL rule.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	closes := md.Closes
	n := len(closes)

	emaSeries := ema.Compute(closes, s.emaPeriod)
	rsiSeries := indicators.RSISeries(closes, s.rsiPeriod)
	atr := indicators.ATR(md.Highs, md.Lows, closes, s.atrPeriod)

	aboveEMA := n > 0 && emaSeries[n-1] > 0 && md.Price > emaSeries[n-1]

	var rsiPrev, rsiNow float64
	if n >= 2 {
		rsiNow = rsiSeries[n-1]
		rsiPrev = rsiSeries[n-2]
	}

	sig := s.decide(md.Price, atr, aboveEMA, rsiPrev, rsiNow, md.Position)
	sig.Ticker = ticker
	return sig
}

// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(price, atr float64, aboveEMA bool, rsiPrev, rsiNow float64, pos *strategy.Position) model.Signal {
	sig := model.Signal{Price: price, RSI: rsiNow}

	if pos != nil {
		tp := pos.PurchasePrice + s.tpMult*atr
		sl := pos.PurchasePrice - s.slMult*atr
		sig.TakeProfit = tp
		sig.StopLoss = sl
		switch {
		case price >= tp:
			sig.Kind = model.SignalSell
			sig.Reason = "TP"
		case price <= sl:
			sig.Kind = model.SignalSell
			sig.Reason = "SL"
		}
		return sig
	}

	if aboveEMA && rsiPrev < s.rsiReversalLevel && rsiNow >= s.rsiReversalLevel {
		sig.Kind = model.SignalBuy
		sig.TakeProfit = price + s.tpMult*atr
		sig.StopLoss = price - s.slMult*atr
	}
	return sig
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/scalping/strategy/rusal/...`
Expected: PASS (`ok`).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/rusal/
git commit -m "feat(scalping): add RUSAL per-share strategy"
```

---

## Task 3: Trim Settings to Interval

**Files:**
- Modify: `internal/service/trading_strategy/scalping/model/settings.go`

- [ ] **Step 1: Replace the file contents**

```go
package model

import "tinvest/internal/enum"

// Settings holds the runner-level knobs for the scalping strategy.
// Per-share signal parameters live inside each strategy.
type Settings struct {
	Interval enum.Interval // timeframe
}

// DefaultSettings returns the hourly default.
func DefaultSettings() Settings {
	return Settings{
		Interval: enum.Hour1,
	}
}
```

- [ ] **Step 2: Verify it builds (the runner still references removed fields — expected to fail until Task 4)**

Run: `go build ./internal/service/trading_strategy/scalping/model/...`
Expected: no output (the model package alone builds).

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/scalping/model/settings.go
git commit -m "refactor(scalping): trim Settings to Interval"
```

---

## Task 4: Rewrite the runner and remove the universe path

**Files:**
- Modify: `internal/service/trading_strategy/scalping/types.go`
- Modify: `internal/service/trading_strategy/scalping/trade.go`
- Create: `internal/service/trading_strategy/scalping/registry.go`
- Delete: `internal/service/trading_strategy/scalping/decide.go`
- Delete: `internal/service/trading_strategy/scalping/decide_test.go`
- Delete: `internal/service/trading_strategy/scalping/universe/` (whole package)

- [ ] **Step 1: Delete the obsolete decision and universe code**

```bash
git rm internal/service/trading_strategy/scalping/decide.go \
       internal/service/trading_strategy/scalping/decide_test.go \
       internal/service/trading_strategy/scalping/universe/universe.go \
       internal/service/trading_strategy/scalping/universe/universe_test.go
```

- [ ] **Step 2: Add the strategy registry**

Create `internal/service/trading_strategy/scalping/registry.go`:

```go
package scalping

import (
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/service/trading_strategy/scalping/strategy/rusal"
)

// defaultStrategies is the fixed set of per-share strategies the runner evaluates.
func defaultStrategies() []strategy.Strategy {
	return []strategy.Strategy{
		rusal.New(),
	}
}
```

- [ ] **Step 3: Rewrite `types.go` — drop indicator deps, add strategies field**

Replace the whole file with:

```go
package scalping

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/client/telegram"
)

type Scalping interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type instrumentsClient interface {
	Shares(ctx context.Context) ([]*imodel.Share, error)
}

type marketDataClient interface {
	GetCandles(ctx context.Context, instrumentUid *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool) ([]*imodel.CandleItemTechAnalyse, error)
}

type operationsClient interface {
	GetPortfolio(ctx context.Context, accountID string) ([]*grpcmodel.Position, error)
}

type Option func(*service)

func WithSettings(s model.Settings) Option {
	return func(svc *service) {
		svc.settings = s
	}
}

type service struct {
	instrumentsClient instrumentsClient
	marketDataClient  marketDataClient
	operationsClient  operationsClient
	tgClient          telegram.Client
	accountID         string
	settings          model.Settings
	strategies        []strategy.Strategy
}

func NewService(
	instrumentsClient instrumentsClient,
	marketDataClient marketDataClient,
	operationsClient operationsClient,
	tgClient telegram.Client,
	accountID string,
	opts ...Option,
) *service {
	svc := &service{
		instrumentsClient: instrumentsClient,
		marketDataClient:  marketDataClient,
		operationsClient:  operationsClient,
		tgClient:          tgClient,
		accountID:         accountID,
		settings:          model.DefaultSettings(),
		strategies:        defaultStrategies(),
	}
	for _, o := range opts {
		o(svc)
	}
	return svc
}
```

- [ ] **Step 4: Rewrite `trade.go`**

Replace the whole file with:

```go
package scalping

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/notification"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

func (s *service) Trade(ctx context.Context, in dto.Trade) error {
	loc, _ := time.LoadLocation("Europe/Moscow")
	dateNow := time.Now().In(loc)
	interval := s.settings.Interval

	shares, err := s.instrumentsClient.Shares(ctx)
	if err != nil {
		return fmt.Errorf("scalping: load shares: %w", err)
	}
	byTicker := make(map[string]*imodel.Share, len(shares))
	for _, sh := range shares {
		byTicker[sh.Ticker] = sh
	}

	positions, err := s.operationsClient.GetPortfolio(ctx, s.accountID)
	if err != nil {
		return fmt.Errorf("scalping: load portfolio: %w", err)
	}
	posByID := make(map[string]strategy.Position, len(positions))
	for _, p := range positions {
		if p.InstrumentType == "share" && p.Quantity > 0 {
			posByID[p.ShareID] = strategy.Position{
				PurchasePrice: utils.CombinePrice(p.PurchasePrice.Units, p.PurchasePrice.Nano),
				Quantity:      p.Quantity,
			}
		}
	}

	signals := make([]model.Signal, 0, len(s.strategies))
	for _, st := range s.strategies {
		sh, ok := byTicker[st.Ticker()]
		if !ok || !sh.Trading {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: share %s not tradable, skipped", st.Ticker()))
			continue
		}
		id := sh.ID
		lookback := st.Lookback()
		limit := int32(lookback)

		time.Sleep(300 * time.Millisecond)
		candles, candErr := s.marketDataClient.GetCandles(ctx, &id, interval.ToNumberInvestApi(),
			utils.TimeStampPbGenerator(dateNow, -int64(lookback), interval), timestamppb.New(dateNow), &limit, true)
		if candErr != nil || len(candles) == 0 {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: candles %s skipped", st.Ticker()))
			continue
		}

		md := buildMarketData(candles)
		if pos, held := posByID[id]; held {
			md.Position = &pos
		}

		sig := st.Decide(md)
		if sig.Kind == model.SignalNone {
			continue
		}
		sig.InstrumentID = sh.ID
		sig.InstrumentName = sh.Name
		signals = append(signals, sig)
	}

	if len(signals) == 0 {
		logger.InfoContext(ctx, "scalping: no signals this run")
		return nil
	}

	if err := s.tgClient.SendMessage(notification.Trade(signals)); err != nil {
		return fmt.Errorf("scalping: send message: %w", err)
	}
	return nil
}

// buildMarketData converts an oldest-first candle series into a strategy snapshot.
func buildMarketData(candles []*imodel.CandleItemTechAnalyse) strategy.MarketData {
	md := strategy.MarketData{
		Highs:   make([]float64, len(candles)),
		Lows:    make([]float64, len(candles)),
		Closes:  make([]float64, len(candles)),
		Volumes: make([]int64, len(candles)),
	}
	for i, c := range candles {
		md.Highs[i] = utils.CombinePrice(c.High.Units, c.High.Nano)
		md.Lows[i] = utils.CombinePrice(c.Low.Units, c.Low.Nano)
		md.Closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
		md.Volumes[i] = c.Volume
	}
	if n := len(md.Closes); n > 0 {
		md.Price = md.Closes[n-1]
	}
	return md
}
```

- [ ] **Step 5: Verify the package builds**

Run: `go build ./internal/service/trading_strategy/scalping/...`
Expected: no output (success).

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/scalping/
git commit -m "refactor(scalping): runner drives a fixed per-share strategy registry"
```

---

## Task 5: Update the service-provider wiring

**Files:**
- Modify: `internal/service_provider/service.go:211-228`

- [ ] **Step 1: Update the `NewService` call**

Replace the body of `GetScalpingTradingService` (the `scalping.NewService(...)` call) with the new signature — the EMA/RSI/ATR arguments are gone:

```go
func (*ServiceProvider) GetScalpingTradingService() scalping.Scalping {
	if serviceProvider.service.scalpingTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.scalpingTradingService = scalping.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			grpcClient.OperationsServiceClient(),
			tgClient,
			serviceProvider.appConfig.Scalping.AccountID,
		)
	}

	return serviceProvider.service.scalpingTradingService
}
```

- [ ] **Step 2: Verify the whole module builds**

Run: `go build ./...`
Expected: no output (success). If `serviceProvider.Ema()`, `RSI()`, or `Atr()` are now unused anywhere else, that is fine — they remain used by other strategies; do not remove them.

- [ ] **Step 3: Commit**

```bash
git add internal/service_provider/service.go
git commit -m "refactor(scalping): drop indicator deps from service wiring"
```

---

## Task 6: Full verification

**Files:** none (verification only).

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: PASS across the module; in particular `internal/service/trading_strategy/scalping/...` is green and there is no remaining reference to the deleted `universe` package or `Decide`/`Candidate`.

- [ ] **Step 2: Vet**

Run: `go vet ./internal/service/trading_strategy/scalping/...`
Expected: no output (success).

- [ ] **Step 3: Confirm dead code is gone**

Run: `git grep -n "universe\|UniverseSize\|RsiReversalLevel\|AtrTakeProfitMult" internal/service/trading_strategy/scalping`
Expected: no matches (the only remaining tunables live inside `strategy/rusal`).

- [ ] **Step 4: Final commit (only if Steps 1-3 surfaced fixes)**

```bash
git add -A
git commit -m "test(scalping): verify per-share strategy migration"
```
```
```

---

## Self-Review Notes

- **Spec coverage:** Variant A (different logic) → per-share `Strategy` interface (Task 1) + RUSAL impl (Task 2). Fully replaces universe scan → Task 4 deletes `universe/` and the runner loops the registry. Approach B (strategy computes indicators from raw candles) → RUSAL uses `ema.Compute` / `indicators.RSISeries` / `indicators.ATR` on `MarketData` (Task 2); runner passes raw series via `buildMarketData` (Task 4). Start with RUSAL only → `defaultStrategies()` lists `rusal.New()` (Task 4). No `MaxOpenPositions` / `openCount` → absent from the contract and core (Tasks 1-2). `Settings` trimmed to `Interval` → Task 3. Runner enriches `InstrumentID`/`InstrumentName` → Task 4 Step 4. Unchanged `scheduler`/`notification`/`dto` and `Scalping.Trade` signature → preserved.
- **Type consistency:** `strategy.MarketData`/`strategy.Position`/`Strategy` used identically across Tasks 1, 2, 4. `model.Signal` fields match `model/signal.go`. `grpcmodel.Position` fields (`Quantity int64`, `PurchasePrice Quotation`, `ShareID`, `InstrumentType`) and `imodel.Share` fields (`ID`, `Ticker`, `Name`, `Trading`) verified against source. `GetCandles` signature matches the existing interface.
- **Registry placement:** kept in package `scalping` (not `strategy`) to avoid the `strategy → rusal → strategy` import cycle.
