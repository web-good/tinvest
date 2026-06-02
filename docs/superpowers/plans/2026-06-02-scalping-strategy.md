# Conservative Hourly Scalping Strategy — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a new `scalping` trading strategy that picks the most volatile RUB shares on the 1h timeframe, emits conservative trend-aligned BUY/SELL Telegram alerts, and tracks open positions from a dedicated brokerage account configured via env.

**Architecture:** New self-contained package `internal/service/trading_strategy/scalping` modeled on `golden_x`. Pure decision logic (`Decide`), universe ranking (`universe.TopN`) and message rendering (`notification.Trade`) are unit-tested; the `Trade` orchestrator wires gRPC clients (instruments, market data, operations) and the EMA/RSI/ATR indicator services. Open positions are read live from the dedicated account (`GetPortfolio`), so no local state is stored. The existing `scalping_rsi` package is left untouched.

**Tech Stack:** Go 1.25, Tinkoff Invest gRPC API, Telegram Bot API, `heetch/confita` config, table-driven `testing`.

---

## Design decisions locked in

- **Universe:** dynamic top-N by ATR% (`ATR.Value / lastPriceRub * 100`), recomputed each run. ATR% (not absolute stddev) avoids price bias.
- **Entry (BUY):** `price > EMA200` AND RSI(14) crosses the reversal level (35) upward (`rsiPrev < 35 && rsiNow >= 35`), only if no open position and `openCount < MaxOpenPositions`.
- **Exit (SELL):** TP = `purchasePrice + 1.5×ATR`, SL = `purchasePrice − 1.0×ATR`, evaluated against current price.
- **Position source of truth:** dedicated account via `OperationsServiceClient.GetPortfolio(accountID)`; account ID from `SCALPING_ACCOUNT_ID` env.
- **Alerts:** one aggregated Telegram message per run.
- **Refinement vs spec:** the `specification/` subpackage is *not* created — the trend/RSI checks are trivial boolean expressions folded into the pure `Decide` function (DRY/YAGNI). All other subpackages from the spec are created.

## File structure

- Create `internal/config/scalping.go` — `ScalpingConfig{ AccountID }` + `NewScalpingConfig()`.
- Modify `internal/config/config.go` — add `Scalping *ScalpingConfig` field.
- Modify `internal/app/init_config.go:32-37` — construct `Scalping: config.NewScalpingConfig()`.
- Create `internal/service/trading_strategy/scalping/model/settings.go` — `Settings` + `DefaultSettings()`.
- Create `internal/service/trading_strategy/scalping/model/signal.go` — `SignalKind`, `Signal`.
- Create `internal/service/trading_strategy/scalping/dto/trade.go` — `Trade` DTO.
- Create `internal/service/trading_strategy/scalping/universe/universe.go` + `_test.go` — `Scored`, `TopN`.
- Create `internal/service/trading_strategy/scalping/notification/notifications.go` + `_test.go` — `Trade`.
- Create `internal/service/trading_strategy/scalping/decide.go` + `decide_test.go` — `Candidate`, `Decide`.
- Create `internal/service/trading_strategy/scalping/types.go` — interfaces, `service`, options, `NewService`.
- Create `internal/service/trading_strategy/scalping/trade.go` — `Trade` orchestrator.
- Create `internal/service/trading_strategy/scalping/scheduler/trade.go` — cron wrapper.
- Modify `internal/service_provider/service.go` — `GetScalpingTradingService()` + struct field + import.
- Modify `internal/app/app.go` — dev goroutine + prod scheduler goroutine.
- Modify `env/local.env.example` — document `SCALPING_ACCOUNT_ID`.

---

### Task 1: Config — `ScalpingConfig`

**Files:**
- Create: `internal/config/scalping.go`
- Modify: `internal/config/config.go:5-11`
- Modify: `internal/app/init_config.go:32-37`
- Modify: `env/local.env.example`

- [ ] **Step 1: Create the config type**

Create `internal/config/scalping.go`:

```go
package config

type ScalpingConfig struct {
	AccountID string `config:"SCALPING_ACCOUNT_ID,required,backend=env"`
}

func NewScalpingConfig() *ScalpingConfig {
	return &ScalpingConfig{}
}
```

- [ ] **Step 2: Add field to `Config`**

In `internal/config/config.go`, add the `Scalping` field to the `Config` struct (keep existing fields):

```go
type Config struct {
	AppEnv         string `config:"APP_ENV,backend=env"`
	AppName        string `config:"APP_NAME,required,backend=env"`
	GrpcClient     *GrpcClient
	TelegramClient *TelegramClient
	PortfolioYield *PortfolioYieldConfig
	Scalping       *ScalpingConfig
}
```

- [ ] **Step 3: Construct it during config load**

In `internal/app/init_config.go`, extend the `cfg := &config.Config{...}` literal (around line 32) to include:

```go
	cfg := &config.Config{
		AppName:        "T-invest",
		GrpcClient:     config.NewGrpcClientConfig(),
		TelegramClient: config.NewTelegramClientConfig(),
		PortfolioYield: config.NewPortfolioYieldConfig(),
		Scalping:       config.NewScalpingConfig(),
	}
```

- [ ] **Step 4: Document the env var**

Append to `env/local.env.example`:

```
# Dedicated brokerage account ID used by the scalping strategy
SCALPING_ACCOUNT_ID=
```

- [ ] **Step 5: Verify it compiles**

Run: `go build ./internal/config/... ./internal/app/...`
Expected: builds with no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/config/scalping.go internal/config/config.go internal/app/init_config.go env/local.env.example
git commit -m "feat(scalping): add SCALPING_ACCOUNT_ID config"
```

---

### Task 2: Strategy settings model

**Files:**
- Create: `internal/service/trading_strategy/scalping/model/settings.go`

- [ ] **Step 1: Create the settings model**

Create `internal/service/trading_strategy/scalping/model/settings.go`:

```go
package model

import "tinvest/internal/enum"

// Settings holds the tunable algorithm knobs for the scalping strategy.
type Settings struct {
	EmaPeriod         int           // trend filter EMA period
	RsiPeriod         int32         // RSI period
	RsiReversalLevel  float64       // RSI level that must be crossed upward to enter
	AtrTakeProfitMult float64       // take-profit = entry + mult*ATR
	AtrStopLossMult   float64       // stop-loss   = entry - mult*ATR
	UniverseSize      int           // number of most-volatile shares to scan
	MaxOpenPositions  int           // cap on simultaneously open positions
	Interval          enum.Interval // timeframe
}

// DefaultSettings returns the conservative hourly defaults.
func DefaultSettings() Settings {
	return Settings{
		EmaPeriod:         200,
		RsiPeriod:         14,
		RsiReversalLevel:  35,
		AtrTakeProfitMult: 1.5,
		AtrStopLossMult:   1.0,
		UniverseSize:      10,
		MaxOpenPositions:  5,
		Interval:          enum.Hour1,
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/service/trading_strategy/scalping/...`
Expected: builds with no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/scalping/model/settings.go
git commit -m "feat(scalping): add Settings with DefaultSettings"
```

---

### Task 3: Signal model

**Files:**
- Create: `internal/service/trading_strategy/scalping/model/signal.go`

- [ ] **Step 1: Create the signal model**

Create `internal/service/trading_strategy/scalping/model/signal.go`:

```go
package model

// SignalKind enumerates the possible decisions per instrument.
type SignalKind int

const (
	SignalNone SignalKind = iota
	SignalBuy
	SignalSell
)

// Signal is a rendered trade alert for one instrument.
type Signal struct {
	Kind           SignalKind
	InstrumentID   string
	InstrumentName string
	Ticker         string
	Price          float64
	TakeProfit     float64
	StopLoss       float64
	RSI            float64
	Reason         string // "TP" or "SL" for sells
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/service/trading_strategy/scalping/...`
Expected: builds with no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/scalping/model/signal.go
git commit -m "feat(scalping): add Signal model"
```

---

### Task 4: Decision logic (`Decide`)

**Files:**
- Create: `internal/service/trading_strategy/scalping/decide.go`
- Test: `internal/service/trading_strategy/scalping/decide_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/service/trading_strategy/scalping/decide_test.go`:

```go
package scalping

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func testSettings() model.Settings {
	return model.Settings{
		RsiReversalLevel:  35,
		AtrTakeProfitMult: 1.5,
		AtrStopLossMult:   1.0,
		MaxOpenPositions:  5,
	}
}

func TestDecide(t *testing.T) {
	s := testSettings()

	tests := []struct {
		name      string
		cand      Candidate
		openCount int
		wantKind  model.SignalKind
		wantTP    float64
		wantSL    float64
		wantReason string
	}{
		{
			name: "buy on trend + rsi reversal",
			cand: Candidate{
				Price: 100, ATR: 2, AboveEMA: true, RSIPrev: 30, RSINow: 36,
			},
			openCount: 0,
			wantKind:  model.SignalBuy,
			wantTP:    103, // 100 + 1.5*2
			wantSL:    98,  // 100 - 1.0*2
		},
		{
			name: "no buy when below ema",
			cand: Candidate{
				Price: 100, ATR: 2, AboveEMA: false, RSIPrev: 30, RSINow: 36,
			},
			openCount: 0,
			wantKind:  model.SignalNone,
		},
		{
			name: "no buy when rsi did not cross upward",
			cand: Candidate{
				Price: 100, ATR: 2, AboveEMA: true, RSIPrev: 36, RSINow: 40,
			},
			openCount: 0,
			wantKind:  model.SignalNone,
		},
		{
			name: "no buy when position cap reached",
			cand: Candidate{
				Price: 100, ATR: 2, AboveEMA: true, RSIPrev: 30, RSINow: 36,
			},
			openCount: 5,
			wantKind:  model.SignalNone,
		},
		{
			name: "sell on take profit",
			cand: Candidate{
				Price: 104, ATR: 2, HasPosition: true, PurchasePrice: 100,
			},
			openCount:  1,
			wantKind:   model.SignalSell,
			wantTP:     103,
			wantSL:     98,
			wantReason: "TP",
		},
		{
			name: "sell on stop loss",
			cand: Candidate{
				Price: 97, ATR: 2, HasPosition: true, PurchasePrice: 100,
			},
			openCount:  1,
			wantKind:   model.SignalSell,
			wantTP:     103,
			wantSL:     98,
			wantReason: "SL",
		},
		{
			name: "hold position inside the band",
			cand: Candidate{
				Price: 101, ATR: 2, HasPosition: true, PurchasePrice: 100,
			},
			openCount: 1,
			wantKind:  model.SignalNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.cand, s, tt.openCount)
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/scalping/ -run TestDecide`
Expected: FAIL — `undefined: Decide` / `undefined: Candidate`.

- [ ] **Step 3: Write the implementation**

Create `internal/service/trading_strategy/scalping/decide.go`:

```go
package scalping

import "tinvest/internal/service/trading_strategy/scalping/model"

// Candidate is the evaluated state of one instrument at decision time.
type Candidate struct {
	InstrumentID   string
	InstrumentName string
	Ticker         string
	Price          float64
	ATR            float64
	AboveEMA       bool
	RSIPrev        float64
	RSINow         float64
	HasPosition    bool
	PurchasePrice  float64
}

// Decide returns the trade signal for a candidate given the settings and the
// number of currently open positions. It is pure and side-effect free.
func Decide(c Candidate, s model.Settings, openCount int) model.Signal {
	base := model.Signal{
		InstrumentID:   c.InstrumentID,
		InstrumentName: c.InstrumentName,
		Ticker:         c.Ticker,
		Price:          c.Price,
		RSI:            c.RSINow,
	}

	if c.HasPosition {
		tp := c.PurchasePrice + s.AtrTakeProfitMult*c.ATR
		sl := c.PurchasePrice - s.AtrStopLossMult*c.ATR
		base.TakeProfit = tp
		base.StopLoss = sl
		switch {
		case c.Price >= tp:
			base.Kind = model.SignalSell
			base.Reason = "TP"
		case c.Price <= sl:
			base.Kind = model.SignalSell
			base.Reason = "SL"
		default:
			base.Kind = model.SignalNone
		}
		return base
	}

	if openCount >= s.MaxOpenPositions {
		base.Kind = model.SignalNone
		return base
	}

	if c.AboveEMA && c.RSIPrev < s.RsiReversalLevel && c.RSINow >= s.RsiReversalLevel {
		base.Kind = model.SignalBuy
		base.TakeProfit = c.Price + s.AtrTakeProfitMult*c.ATR
		base.StopLoss = c.Price - s.AtrStopLossMult*c.ATR
		return base
	}

	base.Kind = model.SignalNone
	return base
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/trading_strategy/scalping/ -run TestDecide -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/decide.go internal/service/trading_strategy/scalping/decide_test.go
git commit -m "feat(scalping): add pure Decide buy/sell logic with tests"
```

---

### Task 5: Universe ranking (`TopN`)

**Files:**
- Create: `internal/service/trading_strategy/scalping/universe/universe.go`
- Test: `internal/service/trading_strategy/scalping/universe/universe_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/service/trading_strategy/scalping/universe/universe_test.go`:

```go
package universe

import "testing"

func TestTopN(t *testing.T) {
	in := []Scored{
		{InstrumentID: "a", ATRPercent: 1.0},
		{InstrumentID: "b", ATRPercent: 3.0},
		{InstrumentID: "c", ATRPercent: 2.0},
		{InstrumentID: "d", ATRPercent: 3.0},
	}

	got := TopN(in, 2)

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	// Highest ATR% first; ties broken by InstrumentID ascending ("b" < "d").
	if got[0].InstrumentID != "b" || got[1].InstrumentID != "d" {
		t.Fatalf("order = [%s %s], want [b d]", got[0].InstrumentID, got[1].InstrumentID)
	}
}

func TestTopN_FewerThanN(t *testing.T) {
	in := []Scored{{InstrumentID: "a", ATRPercent: 1.0}}
	got := TopN(in, 5)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestTopN_DoesNotMutateInput(t *testing.T) {
	in := []Scored{
		{InstrumentID: "a", ATRPercent: 1.0},
		{InstrumentID: "b", ATRPercent: 3.0},
	}
	_ = TopN(in, 2)
	if in[0].InstrumentID != "a" {
		t.Fatalf("input was mutated: in[0] = %s", in[0].InstrumentID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/scalping/universe/`
Expected: FAIL — `undefined: TopN` / `undefined: Scored`.

- [ ] **Step 3: Write the implementation**

Create `internal/service/trading_strategy/scalping/universe/universe.go`:

```go
package universe

import "sort"

// Scored is one instrument with its volatility score (ATR%).
type Scored struct {
	InstrumentID   string
	InstrumentName string
	Ticker         string
	ATRPercent     float64
}

// TopN returns the n most volatile instruments, highest ATR% first.
// Ties are broken by InstrumentID ascending for deterministic output.
// The input slice is not mutated.
func TopN(scored []Scored, n int) []Scored {
	sorted := make([]Scored, len(scored))
	copy(sorted, scored)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ATRPercent != sorted[j].ATRPercent {
			return sorted[i].ATRPercent > sorted[j].ATRPercent
		}
		return sorted[i].InstrumentID < sorted[j].InstrumentID
	})
	if n < len(sorted) {
		sorted = sorted[:n]
	}
	return sorted
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/trading_strategy/scalping/universe/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/universe/
git commit -m "feat(scalping): add volatility universe ranking with tests"
```

---

### Task 6: Telegram notification rendering

**Files:**
- Create: `internal/service/trading_strategy/scalping/notification/notifications.go`
- Test: `internal/service/trading_strategy/scalping/notification/notifications_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/service/trading_strategy/scalping/notification/notifications_test.go`:

```go
package notification

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func TestTrade_RendersBuyAndSell(t *testing.T) {
	signals := []model.Signal{
		{Kind: model.SignalBuy, InstrumentName: "Sberbank", Ticker: "SBER", Price: 100, TakeProfit: 103, StopLoss: 98, RSI: 36},
		{Kind: model.SignalSell, InstrumentName: "Gazprom", Ticker: "GAZP", Price: 104, TakeProfit: 103, StopLoss: 98, Reason: "TP"},
	}

	got := Trade(signals)

	for _, want := range []string{"покупку", "продажу", "Sberbank", "SBER", "Gazprom", "GAZP", "TP"} {
		if !strings.Contains(got, want) {
			t.Errorf("message missing %q\n---\n%s", want, got)
		}
	}
}

func TestTrade_OnlyBuysOmitsSellSection(t *testing.T) {
	signals := []model.Signal{
		{Kind: model.SignalBuy, InstrumentName: "Sberbank", Ticker: "SBER", Price: 100, TakeProfit: 103, StopLoss: 98, RSI: 36},
	}

	got := Trade(signals)

	if strings.Contains(got, "продажу") {
		t.Errorf("sell section should be absent\n---\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/scalping/notification/`
Expected: FAIL — `undefined: Trade`.

- [ ] **Step 3: Write the implementation**

Create `internal/service/trading_strategy/scalping/notification/notifications.go`:

```go
package notification

import (
	"fmt"
	"strings"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

// Trade renders an aggregated HTML Telegram message for the given signals.
func Trade(signals []model.Signal) string {
	var buys, sells []model.Signal
	for _, s := range signals {
		switch s.Kind {
		case model.SignalBuy:
			buys = append(buys, s)
		case model.SignalSell:
			sells = append(sells, s)
		}
	}

	b := strings.Builder{}
	b.WriteString("⚡️ <b>Скальпинг (1H)</b>\n\n")

	if len(buys) > 0 {
		b.WriteString("<u><b>Сигналы на покупку:</b></u>\n")
		for _, s := range buys {
			b.WriteString(fmt.Sprintf(
				"🟢 <b>%s</b> (%s)\n  Цена: %.2f | TP: %.2f | SL: %.2f | RSI: %.0f\n",
				s.InstrumentName, s.Ticker, s.Price, s.TakeProfit, s.StopLoss, s.RSI,
			))
		}
		b.WriteString("\n")
	}

	if len(sells) > 0 {
		b.WriteString("<u><b>Сигналы на продажу:</b></u>\n")
		for _, s := range sells {
			b.WriteString(fmt.Sprintf(
				"🔴 <b>%s</b> (%s) [%s]\n  Цена: %.2f | TP: %.2f | SL: %.2f\n",
				s.InstrumentName, s.Ticker, s.Reason, s.Price, s.TakeProfit, s.StopLoss,
			))
		}
	}

	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/trading_strategy/scalping/notification/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/notification/
git commit -m "feat(scalping): add aggregated buy/sell telegram rendering with tests"
```

---

### Task 7: DTO

**Files:**
- Create: `internal/service/trading_strategy/scalping/dto/trade.go`

- [ ] **Step 1: Create the DTO**

Create `internal/service/trading_strategy/scalping/dto/trade.go`:

```go
package dto

import "tinvest/internal/enum"

// Trade carries per-run parameters for the scalping strategy.
type Trade struct {
	Interval  enum.Interval
	Scheduler string
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/service/trading_strategy/scalping/...`
Expected: builds with no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/scalping/dto/trade.go
git commit -m "feat(scalping): add Trade dto"
```

---

### Task 8: Service type, interfaces and constructor

**Files:**
- Create: `internal/service/trading_strategy/scalping/types.go`

- [ ] **Step 1: Create the service wiring**

Create `internal/service/trading_strategy/scalping/types.go`:

```go
package scalping

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain"
	domainatr "tinvest/internal/domain/atr"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/client/telegram"
)

type Scalping interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type emaInstrument interface {
	TechAnalyse(ctx context.Context, instrumentUid *string, interval int32, from time.Time, to time.Time, period int) ([]domainema.ItemTechAnalyse, error)
}

type rsiInstrument interface {
	CalculateRSI(ctx context.Context, instrumentUid string, interval enum.Interval, dateFrom *timestamppb.Timestamp, dateTo *timestamppb.Timestamp, length int32) ([]*domain.RSIItemTechAnalyse, error)
}

type atrInstrument interface {
	TechAnalyse(ctx context.Context, instrumentUid *string, interval enum.Interval, dateNow time.Time) (domainatr.ItemTechAnalyse, error)
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
	ema               emaInstrument
	rsi               rsiInstrument
	atr               atrInstrument
	instrumentsClient instrumentsClient
	marketDataClient  marketDataClient
	operationsClient  operationsClient
	tgClient          telegram.Client
	accountID         string
	settings          model.Settings
}

func NewService(
	ema emaInstrument,
	rsi rsiInstrument,
	atr atrInstrument,
	instrumentsClient instrumentsClient,
	marketDataClient marketDataClient,
	operationsClient operationsClient,
	tgClient telegram.Client,
	accountID string,
	opts ...Option,
) *service {
	svc := &service{
		ema:               ema,
		rsi:               rsi,
		atr:               atr,
		instrumentsClient: instrumentsClient,
		marketDataClient:  marketDataClient,
		operationsClient:  operationsClient,
		tgClient:          tgClient,
		accountID:         accountID,
		settings:          model.DefaultSettings(),
	}
	for _, o := range opts {
		o(svc)
	}
	return svc
}
```

> Note on imports: `imodel "tinvest/internal/model"` is the package that defines `Share` (`internal/model/share.go`) and `CandleItemTechAnalyse` (`internal/model/tech_analyse.go`). Confirm both types live in package `model` under `internal/model` before building; adjust the alias target if not.

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/service/trading_strategy/scalping/...`
Expected: builds with no errors. (Resolve any import-path/alias mismatch surfaced here.)

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/scalping/types.go
git commit -m "feat(scalping): add service type, interfaces and constructor"
```

---

### Task 9: `Trade` orchestrator

**Files:**
- Create: `internal/service/trading_strategy/scalping/trade.go`

- [ ] **Step 1: Create the orchestrator**

Create `internal/service/trading_strategy/scalping/trade.go`:

```go
package scalping

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/notification"
	"tinvest/internal/service/trading_strategy/scalping/universe"
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

	// 1. Rank the universe by ATR% (ATR / last price).
	scored := make([]universe.Scored, 0, len(shares))
	for _, sh := range shares {
		if sh.Currency != "rub" || !sh.Trading || sh.LastPriceRub == 0 {
			continue
		}
		time.Sleep(300 * time.Millisecond)
		atrItem, atrErr := s.atr.TechAnalyse(ctx, &sh.ID, interval, dateNow)
		if atrErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("scalping: atr %s: %w", sh.Ticker, atrErr).Error())
			continue
		}
		scored = append(scored, universe.Scored{
			InstrumentID:   sh.ID,
			InstrumentName: sh.Name,
			Ticker:         sh.Ticker,
			ATRPercent:     atrItem.Value / sh.LastPriceRub * 100,
		})
	}
	top := universe.TopN(scored, s.settings.UniverseSize)

	// 2. Read open positions from the dedicated account.
	positions, err := s.operationsClient.GetPortfolio(ctx, s.accountID)
	if err != nil {
		return fmt.Errorf("scalping: load portfolio: %w", err)
	}
	posByID := make(map[string]float64, len(positions))
	for _, p := range positions {
		if p.InstrumentType == "share" && p.Quantity > 0 {
			posByID[p.ShareID] = utils.CombinePrice(p.PurchasePrice.Units, p.PurchasePrice.Nano)
		}
	}
	openCount := len(posByID)

	// 3. Evaluate each candidate and collect signals.
	signals := make([]model.Signal, 0, len(top))
	for _, item := range top {
		id := item.InstrumentID
		time.Sleep(300 * time.Millisecond)

		emaItems, emaErr := s.ema.TechAnalyse(ctx, &id, interval.ToNumberInvestApi(),
			utils.TimeGenerator(dateNow, -int64(s.settings.EmaPeriod)-50, interval), dateNow, s.settings.EmaPeriod)
		if emaErr != nil || len(emaItems) == 0 {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: ema %s skipped", item.Ticker))
			continue
		}
		last := emaItems[len(emaItems)-1].SignalLine
		emaVal := utils.CombinePrice(last.Units, last.Nano)

		rsiItems, rsiErr := s.rsi.CalculateRSI(ctx, id, interval,
			utils.TimeStampPbGenerator(dateNow, -int64(s.settings.RsiPeriod)*3, interval), timestamppb.New(dateNow), s.settings.RsiPeriod)
		if rsiErr != nil || len(rsiItems) < 2 {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: rsi %s skipped", item.Ticker))
			continue
		}
		rsiNow := utils.CombinePrice(rsiItems[len(rsiItems)-1].SignalLine.Units, rsiItems[len(rsiItems)-1].SignalLine.Nano)
		rsiPrev := utils.CombinePrice(rsiItems[len(rsiItems)-2].SignalLine.Units, rsiItems[len(rsiItems)-2].SignalLine.Nano)

		limit := int32(2)
		candles, candErr := s.marketDataClient.GetCandles(ctx, &id, interval.ToNumberInvestApi(),
			utils.TimeStampPbGenerator(dateNow, -2, interval), timestamppb.New(dateNow), &limit, true)
		if candErr != nil || len(candles) == 0 {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: candles %s skipped", item.Ticker))
			continue
		}
		closeQ := candles[len(candles)-1].Close
		price := utils.CombinePrice(closeQ.Units, closeQ.Nano)

		atrItem, atrErr := s.atr.TechAnalyse(ctx, &id, interval, dateNow)
		if atrErr != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: atr %s skipped", item.Ticker))
			continue
		}

		purchase, hasPos := posByID[id]

		sig := Decide(Candidate{
			InstrumentID:   id,
			InstrumentName: item.InstrumentName,
			Ticker:         item.Ticker,
			Price:          price,
			ATR:            atrItem.Value,
			AboveEMA:       price > emaVal,
			RSIPrev:        rsiPrev,
			RSINow:         rsiNow,
			HasPosition:    hasPos,
			PurchasePrice:  purchase,
		}, s.settings, openCount)

		if sig.Kind != model.SignalNone {
			signals = append(signals, sig)
		}
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
```

- [ ] **Step 2: Verify the whole package builds and existing tests still pass**

Run: `go build ./internal/service/trading_strategy/scalping/... && go test ./internal/service/trading_strategy/scalping/...`
Expected: builds; all tests PASS.

- [ ] **Step 3: Vet the package**

Run: `go vet ./internal/service/trading_strategy/scalping/...`
Expected: no diagnostics.

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/scalping/trade.go
git commit -m "feat(scalping): add Trade orchestrator"
```

---

### Task 10: Scheduler wrapper

**Files:**
- Create: `internal/service/trading_strategy/scalping/scheduler/trade.go`

- [ ] **Step 1: Create the scheduler wrapper**

Create `internal/service/trading_strategy/scalping/scheduler/trade.go` (mirrors `golden_x/scheduler/trade.go`):

```go
package scheduler

import (
	"context"
	"time"

	"tinvest/internal/service/trading_strategy/scalping"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/pkg/logger"
	"tinvest/pkg/scheduler"
)

type schedulerService struct {
	sh      scheduler.Scheduler
	service scalping.Scalping
}

func (s *schedulerService) Trade(ctx context.Context, in dto.Trade) error {
	jobTicker := time.NewTicker(time.Hour)
	defer jobTicker.Stop()

	err := s.sh.AddJob(in.Scheduler, func() {
		logger.InfoContext(ctx, "Воркер Scalping начал работу")
		if err := s.service.Trade(ctx, in); err != nil {
			logger.ErrorContext(ctx, "Ошибка в ходе работы job", err)
		}
	})
	if err != nil {
		return err
	}

	s.sh.Start()
	defer s.sh.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-jobTicker.C:
			logger.InfoContext(ctx, "Worker Scalping is running")
		}
	}
}

func NewSchedulerService(service scalping.Scalping) scalping.Scalping {
	return &schedulerService{
		sh:      scheduler.NewScheduler(),
		service: service,
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/service/trading_strategy/scalping/...`
Expected: builds with no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/scalping/scheduler/trade.go
git commit -m "feat(scalping): add cron scheduler wrapper"
```

---

### Task 11: Service-provider wiring

**Files:**
- Modify: `internal/service_provider/service.go`

- [ ] **Step 1: Add the import**

In `internal/service_provider/service.go`, add to the import block (alongside the other `trading_strategy` imports):

```go
	"tinvest/internal/service/trading_strategy/scalping"
```

- [ ] **Step 2: Add the struct field**

In the `service` struct, add (next to `scalpingRsiTradingService`):

```go
	scalpingTradingService scalping.Scalping
```

- [ ] **Step 3: Add the getter**

Add this method (mirrors `GetScalpingRsiTradingService`, plus ATR and operations client + account ID):

```go
func (*ServiceProvider) GetScalpingTradingService() scalping.Scalping {
	if serviceProvider.service.scalpingTradingService == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.scalpingTradingService = scalping.NewService(
			serviceProvider.Ema(),
			serviceProvider.RSI(),
			serviceProvider.Atr(),
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

- [ ] **Step 4: Verify it compiles**

Run: `go build ./internal/service_provider/...`
Expected: builds with no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/service_provider/service.go
git commit -m "feat(scalping): wire scalping service into service provider"
```

---

### Task 12: App wiring (dev + prod)

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Add imports**

In `internal/app/app.go`, add to the import block:

```go
	scalpingdto "tinvest/internal/service/trading_strategy/scalping/dto"
	scalpingscheduler "tinvest/internal/service/trading_strategy/scalping/scheduler"
```

- [ ] **Step 2: Add the dev goroutine**

In `runDev`, before `wg.Wait()`, add an active worker (so you can test it locally):

```go
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetScalpingTradingService().Trade(ctx, scalpingdto.Trade{
			Interval:  enum.Hour1,
			Scheduler: "0 8-23 * * 1-5",
		})
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Scalping", err.Error())
		}
	}()
```

- [ ] **Step 3: Add the prod scheduler goroutine**

In `runProd`, change `wg.Add(5)` to `wg.Add(6)` and add:

```go
	go func() {
		defer wg.Done()
		err := scalpingscheduler.NewSchedulerService(a.sp.GetScalpingTradingService()).Trade(
			ctx,
			scalpingdto.Trade{
				Interval:  enum.Hour1,
				Scheduler: "1 8-23 * * 1-5",
			},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Scalping", err.Error())
		}
	}()
```

> The cron expression `1 8-23 * * 1-5` runs at minute 1 of each hour from 08:00 to 23:00 MSK on weekdays, covering the MOEX session. Confirm the project's `pkg/scheduler` cron field order matches the existing `golden_x` usage (`"0 */5 * * *"`) — it does (standard 5-field cron).

- [ ] **Step 4: Verify the whole project builds and vets**

Run: `go build ./... && go vet ./...`
Expected: builds and vets with no errors.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: PASS (no regressions; new scalping tests green).

- [ ] **Step 6: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(scalping): run scalping worker in dev and prod"
```

---

## Final verification

- [ ] `go build ./...` — clean
- [ ] `go vet ./...` — clean
- [ ] `go test ./...` — all green
- [ ] `gofmt -l internal/service/trading_strategy/scalping internal/config/scalping.go` — prints nothing (formatted)
- [ ] Manual smoke (optional, needs real token + `SCALPING_ACCOUNT_ID`): set env, `APP_ENV=dev go run ./cmd/main`, confirm a Telegram message arrives (or "no signals this run" in logs).

## Notes for the implementer

- The codebase mixes `int32(in.Interval)` and `interval.ToNumberInvestApi()` when calling indicator services. This plan uses `interval.ToNumberInvestApi()` everywhere for correctness; for `Hour1` both happen to equal `4`, so behavior is unchanged for the default timeframe.
- `time.Sleep(300ms)` between per-share API calls mirrors `scalping_rsi` and avoids Tinkoff rate limits; tune if needed.
- Universe ranking uses ATR% rather than the existing `volatility` service (which returns absolute price stddev and is biased toward high-priced shares). ATR is already fetched per share, so no extra service dependency.
