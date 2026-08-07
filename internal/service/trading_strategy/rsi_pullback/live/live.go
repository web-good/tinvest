// Package live runs the rsi_pullback strategy against a real account: one pass per
// 30-minute bar that either opens a position (core signal + sizing + market BUY +
// protective exchange stop) or manages the open one (SL/TRAIL/TP/RSI exits and the
// exchange stop order that mirrors the level). The trading core, the market-data
// assembly and the state file are shared with the backtest, so live and backtest take
// the same decision on the same bar.
package live

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/livecore/executor"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/client/telegram"
)

// alertLabel — заголовок операционных уведомлений; пакет notifier общий, и по сообщению
// должно быть видно, какой раннер его прислал.
const alertLabel = "RSI Pullback"

type instrumentsClient interface {
	Shares(ctx context.Context) ([]*imodel.Share, error)
}

type operationsClient interface {
	GetPortfolio(ctx context.Context, accountID string) ([]*grpcmodel.Position, error)
	GetPortfolioTotal(ctx context.Context, accountID string) (float64, error)
	GetAvailableCash(ctx context.Context, accountID string) (float64, error)
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)
}

// Service runs one scheduled rsi_pullback pass.
type Service interface {
	Run(ctx context.Context, in dto.Run) error
}

type service struct {
	// mu serializes passes: the cron fires every half hour and shares one *service
	// instance (memoized in service_provider). Without the lock a delayed pass could
	// interleave its Load→mutate→Save cycle with the next one and silently drop its
	// state writes.
	mu          sync.Mutex
	instruments instrumentsClient
	market      candles.CandleClient
	ops         operationsClient
	exec        *executor.Executor
	stops       *stoporders.Executor
	tg          telegram.Client
	cfg         *config.RSIPullbackConfig
	statePath   string
	// now — источник времени пасса. Подменяется в тестах так же, как statePath: гейт
	// свежести бара (maxBarAge) сравнивает время последнего бара именно с ним, поэтому
	// на фиксированных датах фикстур настенные часы дали бы «протухшие» данные всегда.
	now func() time.Time
	// store — подменяемое хранилище стейта. Нужен ровно затем, что сбой записи файлом не
	// воспроизвести переносимо: право на запись отбирается правами каталога, а под root
	// они не действуют. nil означает обычный FileStore по statePath.
	store statestore.Store
}

// NewService wires the live rsi_pullback service. The orders and stops clients may be nil
// only when TradeEnabled is false and no order will ever be placed (tests/dry-run).
func NewService(
	instruments instrumentsClient,
	market candles.CandleClient,
	ops operationsClient,
	orders executor.OrdersClient,
	stops stoporders.Client,
	tg telegram.Client,
	cfg *config.RSIPullbackConfig,
) *service {
	return &service{
		instruments: instruments,
		market:      market,
		ops:         ops,
		exec:        executor.New(orders, cfg.AccountID, cfg.TradeEnabled),
		stops:       stoporders.New(stops, cfg.AccountID, cfg.TradeEnabled),
		tg:          tg,
		cfg:         cfg,
		statePath:   filepath.Join("data", "state", "rsi_pullback_"+cfg.AccountID+".json"),
		now:         nowMSK,
	}
}

// Run makes the single pass. The mutex is held for the whole pass so that two overlapping
// cron invocations cannot interleave their Load→mutate→Save cycles.
func (s *service) Run(ctx context.Context, _ dto.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.pass(ctx)
}

// notify sends a Telegram message only when NotifyEnabled.
func (s *service) notify(msg string) {
	if s.cfg.NotifyEnabled {
		_ = s.tg.SendMessage(msg)
	}
}

// sharesByTicker indexes tradable shares for the configured universe.
func (s *service) sharesByTicker(ctx context.Context) (map[string]*imodel.Share, error) {
	all, err := s.instruments.Shares(ctx)
	if err != nil {
		return nil, fmt.Errorf("rsi_pullback: load shares: %w", err)
	}
	out := make(map[string]*imodel.Share, len(all))
	for _, sh := range all {
		out[sh.Ticker] = sh
	}
	return out, nil
}

// heldByShareID indexes the account's share positions with qty > 0.
func (s *service) heldByShareID(ctx context.Context) (map[string]*grpcmodel.Position, error) {
	positions, err := s.ops.GetPortfolio(ctx, s.cfg.AccountID)
	if err != nil {
		return nil, fmt.Errorf("rsi_pullback: load portfolio: %w", err)
	}
	out := make(map[string]*grpcmodel.Position, len(positions))
	for _, p := range positions {
		if p.InstrumentType == "share" && p.Quantity > 0 {
			out[p.ShareID] = p
		}
	}
	return out, nil
}

func nowMSK() time.Time {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(loc)
}

// stateStore returns the injected store, or a FileStore for the configured state path.
func (s *service) stateStore() statestore.Store {
	if s.store != nil {
		return s.store
	}
	return statestore.New(s.statePath)
}
