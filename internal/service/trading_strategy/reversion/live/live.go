package live

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/reversion/live/dto"
	"tinvest/internal/service/trading_strategy/reversion/live/executor"
	"tinvest/internal/service/trading_strategy/reversion/live/marketdata"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/client/telegram"
)

type instrumentsClient interface {
	Shares(ctx context.Context) ([]*imodel.Share, error)
}

type operationsClient interface {
	GetPortfolio(ctx context.Context, accountID string) ([]*grpcmodel.Position, error)
	GetPortfolioTotal(ctx context.Context, accountID string) (float64, error)
	GetAvailableCash(ctx context.Context, accountID string) (float64, error)
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)
}

// Service runs one scheduled reversion pass.
type Service interface {
	Run(ctx context.Context, in dto.Run) error
}

type service struct {
	// mu serializes the buy-pass and manage-pass: both are scheduled at minute 0 of
	// overlapping hours and share the same *service instance (via GetReversionLiveService
	// memoization in service_provider). Without the lock, concurrent passes can interleave
	// their Load→mutate→Save cycles and silently drop each other's state writes.
	mu          sync.Mutex
	instruments instrumentsClient
	market      marketdata.CandleClient
	ops         operationsClient
	exec        *executor.Executor
	tg          telegram.Client
	cfg         *config.ReversionConfig
	statePath   string
}

// NewService wires the live reversion service. The orders client may be nil only when
// TradeEnabled is false and no order will ever be placed (tests/dry-run).
func NewService(
	instruments instrumentsClient,
	market marketdata.CandleClient,
	ops operationsClient,
	orders executor.OrdersClient,
	tg telegram.Client,
	cfg *config.ReversionConfig,
) *service {
	return &service{
		instruments: instruments,
		market:      market,
		ops:         ops,
		exec:        executor.New(orders, cfg.AccountID, cfg.TradeEnabled),
		tg:          tg,
		cfg:         cfg,
		statePath:   filepath.Join("data", "state", "reversion_"+cfg.AccountID+".json"),
	}
}

// Run dispatches to the buy or manage pass.
// The mutex is held for the entire pass so that two cron workers sharing this
// service instance (buy-pass and manage-pass) cannot interleave their
// Load→mutate→Save cycles and overwrite each other's state.
func (s *service) Run(ctx context.Context, in dto.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch in.Mode {
	case dto.ModeBuy:
		return s.buyPass(ctx)
	case dto.ModeManage:
		return s.managePass(ctx)
	default:
		return fmt.Errorf("reversion: unknown run mode %d", in.Mode)
	}
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
		return nil, fmt.Errorf("reversion: load shares: %w", err)
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
		return nil, fmt.Errorf("reversion: load portfolio: %w", err)
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

// stateStore returns a FileStore for the configured state path.
func (s *service) stateStore() *statestore.FileStore {
	return statestore.New(s.statePath)
}
