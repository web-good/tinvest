// Package live connects gap_fade to the rsi_pullback account runner as a guest adapter
// (docs/gap_fade/live.md): same process, pass, state file and account; its own Telegram topic.
package live

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/rebuild"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	pkgmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// Name — имя стратегии в стейте счёта.
const Name = "gap_fade"

const alertLabel = "Gap Fade"

// entryCutoffMinute — пасс с этого времени MSK (10:00) gap_fade не спрашивает: сигнал — бар 06:30,
// запас покрывает сдвиг расписания сессии и опоздавший пасс.
const entryCutoffMinute = 10 * 60

// DividendsClient — источник дивидендов для фильтра дня отсечки.
type DividendsClient interface {
	GetDividends(ctx context.Context, instrumentID string, from, to time.Time) ([]*pkgmodel.Dividend, error)
}

// Strategy — gap_fade как гость раннера счёта.
type Strategy struct {
	cfg  *config.GapFadeConfig
	tg   telegram.Client
	divs DividendsClient
}

var (
	_ adapter.Strategy    = (*Strategy)(nil)
	_ adapter.EntryWindow = (*Strategy)(nil)
)

func New(cfg *config.GapFadeConfig, tg telegram.Client, divs DividendsClient) *Strategy {
	return &Strategy{cfg: cfg, tg: tg, divs: divs}
}

var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

func (s *Strategy) Name() string       { return Name }
func (s *Strategy) Label() string      { return alertLabel }
func (s *Strategy) Tickers() []string  { return s.cfg.Tickers }
func (s *Strategy) BuyPct() float64    { return s.cfg.BuyPct }
func (s *Strategy) TradeEnabled() bool { return s.cfg.TradeEnabled }

func (s *Strategy) Decider(ticker string) (adapter.Decider, bool) {
	st, ok := StrategyFor(ticker)
	if !ok {
		return nil, false
	}
	return st, true
}

// EntryPossible — будний день MSK до 10:00: gap_fade входит только по бару 06:30.
func (s *Strategy) EntryPossible(now time.Time) bool {
	t := now.In(mskLoc)
	if wd := t.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return t.Hour()*60+t.Minute() < entryCutoffMinute
}

// DesiredStop — стоп, замороженный на входе от close сигнального бара (Entry.StopLoss). Уровень
// не меняется за всю жизнь позиции.
func (s *Strategy) DesiredStop(_ string, e statestore.Entry) (float64, string) {
	if e.StopLoss <= 0 {
		return 0, ""
	}
	return e.StopLoss, "SL"
}

// Reconstruct поднимает вход по API. Close сигнального бара неизвестен, поэтому стоп меряется
// от цены входа; цели нет — позицию закроет EOD или первый бар новой даты.
func (s *Strategy) Reconstruct(ctx context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	p, ok := ParamsFor(in.Ticker)
	if !ok {
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: %q не зарегистрирован", in.Ticker)
	}
	entryTime, err := rebuild.LastBuyTime(ctx, in.Trades, in.AccountID, in.InstrumentID, in.Now)
	switch {
	case errors.Is(err, rebuild.ErrNoBuyFill):
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: no BUY fill found for %s", in.Ticker)
	case err != nil:
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: %w", err)
	}
	atr, err := rebuild.WeekdayDailyATRBefore(ctx, in.Candles, in.InstrumentID, entryTime, p.DailyATRPeriod)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: %w", err)
	}
	if atr <= 0 {
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: %s: cannot rebuild daily ATR as of %s", in.Ticker, entryTime.Format(time.RFC3339))
	}
	return statestore.Entry{
		Ticker:     in.Ticker,
		EntryTime:  entryTime,
		EntryPrice: in.PurchasePrice,
		EntryATR:   atr,
		StopLoss:   in.PurchasePrice - p.StopDailyATR*atr,
		MaxFav:     in.PurchasePrice,
	}, nil
}

// Notify шлёт в тему gap_fade с меткой стратегии. Сбой доставки — ERROR-лог, как у соседей.
func (s *Strategy) Notify(msg string) {
	if !s.cfg.NotifyEnabled || s.tg == nil {
		return
	}
	if err := s.tg.SendMessage("[" + alertLabel + "] " + msg); err != nil {
		logger.ErrorContext(context.Background(), fmt.Sprintf("gap_fade: уведомление не доставлено: %v", err))
	}
}
