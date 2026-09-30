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
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/livecore/executor"
	"tinvest/internal/service/trading_strategy/livecore/notifier"
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
	// Announce сообщает в Telegram, что воркер поднялся. Метод интерфейса, а не
	// внутренняя деталь пасса: зовёт его приложение при старте, сразу после проверки
	// Ready, — то есть ровно в той точке, где ещё известно, поднялся раннер или нет.
	Announce()
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
	cfg         *config.RSIPullbackConfig
	// slots — стратегии счёта в порядке приоритета входа; slots[0] — rsi_pullback.
	slots []*slot
	// accountStops читает список активных стоп-заявок счёта — один раз на пасс. Боевой,
	// если боевая хоть одна стратегия: в dry-run List возвращает пустой список.
	accountStops *stoporders.Executor
	// anyLive — боевая хоть одна стратегия. Тогда бумажные входы стейт не пишут: бумажная
	// запись заняла бы тикер и заблокировала боевой вход соседки.
	anyLive   bool
	statePath string
	// now — источник времени пасса. Подменяется в тестах так же, как statePath: гейт
	// свежести бара (maxBarAge) сравнивает время последнего бара именно с ним, поэтому
	// на фиксированных датах фикстур настенные часы дали бы «протухшие» данные всегда.
	now func() time.Time
	// store — подменяемое хранилище стейта. Нужен ровно затем, что сбой записи файлом не
	// воспроизвести переносимо: право на запись отбирается правами каталога, а под root
	// они не действуют. nil означает обычный FileStore по statePath.
	store statestore.Store
}

// slot — стратегия счёта со своими исполнителями: TradeEnabled у каждой свой.
type slot struct {
	strat adapter.Strategy
	exec  *executor.Executor
	stops *stoporders.Executor
}

func (sl *slot) alert(ticker, msg string) {
	sl.strat.Notify(notifier.Alert(sl.strat.Label(), ticker, msg))
}

// NewService wires the live rsi_pullback service. The orders and stops clients may be nil
// only when TradeEnabled is false and no order will ever be placed (tests/dry-run).
// guests — стратегии, торгующие на том же счёте; встают слотами после rsi_pullback.
func NewService(
	instruments instrumentsClient,
	market candles.CandleClient,
	ops operationsClient,
	orders executor.OrdersClient,
	stops stoporders.Client,
	tg telegram.Client,
	cfg *config.RSIPullbackConfig,
	guests ...adapter.Strategy,
) *service {
	strats := append([]adapter.Strategy{&pullbackStrategy{cfg: cfg, tg: tg}}, guests...)
	s := &service{
		instruments: instruments,
		market:      market,
		ops:         ops,
		cfg:         cfg,
		statePath:   filepath.Join("data", "state", "rsi_pullback_"+cfg.AccountID+".json"),
		now:         nowMSK,
	}
	for _, st := range strats {
		s.slots = append(s.slots, &slot{
			strat: st,
			exec:  executor.New(orders, cfg.AccountID, st.TradeEnabled()),
			stops: stoporders.New(stops, cfg.AccountID, st.TradeEnabled()),
		})
		s.anyLive = s.anyLive || st.TradeEnabled()
	}
	s.accountStops = stoporders.New(stops, cfg.AccountID, s.anyLive)
	return s
}

// Run makes the single pass. The mutex is held for the whole pass so that two overlapping
// cron invocations cannot interleave their Load→mutate→Save cycles.
func (s *service) Run(ctx context.Context, _ dto.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.pass(ctx)
}

// notify шлёт счётное сообщение (не привязанное к стратегии) в тему rsi_pullback.
func (s *service) notify(msg string) { s.slots[0].strat.Notify(msg) }

// Announce объявляет о подъёме воркера. Единственное сообщение раннера, не привязанное
// к событию: все остальные шлются на входе, выходе, постановке стопа или сбое, а их может
// не быть неделями — и тогда молчание темы неотличимо от раннера, который не поднялся.
// Каждая стратегия счёта объявляет о себе в своей теме: своя вселенная и свой режим.
func (s *service) Announce() {
	for _, sl := range s.slots {
		sl.strat.Notify(notifier.Startup(sl.strat.Label(), sl.strat.Tickers(), !sl.strat.TradeEnabled()))
	}
}

// slotByName — слот стратегии по имени владельца из стейта; nil — такой стратегии на
// счёте нет.
func (s *service) slotByName(name string) *slot {
	for _, sl := range s.slots {
		if sl.strat.Name() == name {
			return sl
		}
	}
	return nil
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
