package live

import (
	"context"
	"errors"
	"fmt"

	"tinvest/internal/config"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/rebuild"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// Name — имя стратегии в стейте счёта.
const Name = "pdl_rsi"

const alertLabel = "PDL RSI"

// Strategy — pdl_rsi как гость раннера счёта.
type Strategy struct {
	cfg *config.PDLRSIConfig
	tg  telegram.Client
}

var (
	_ adapter.Strategy    = (*Strategy)(nil)
	_ adapter.BarInterval = (*Strategy)(nil)
)

func New(cfg *config.PDLRSIConfig, tg telegram.Client) *Strategy {
	return &Strategy{cfg: cfg, tg: tg}
}

func (s *Strategy) Name() string       { return Name }
func (s *Strategy) Label() string      { return alertLabel }
func (s *Strategy) Tickers() []string  { return s.cfg.Tickers }
func (s *Strategy) BuyPct() float64    { return s.cfg.BuyPct }
func (s *Strategy) TradeEnabled() bool { return s.cfg.TradeEnabled }

// BarInterval — стратегия часовая: раннер собирает ей Hour1-бары.
func (s *Strategy) BarInterval() enum.Interval { return enum.Hour1 }

func (s *Strategy) Decider(ticker string) (adapter.Decider, bool) {
	st, ok := StrategyFor(ticker)
	if !ok {
		return nil, false
	}
	return st, true
}

// DesiredStop — стоп, замороженный на входе: StopATR дневных ATR под ценой входа с капом
// MaxStopPct; трейла по цене нет (лесенка RSI — выход по сигналу, не стоп-заявка), уровень
// не меняется за всю жизнь позиции. Раннер переставляет заявку только при расхождении размера или после
// внешнего снятия.
func (s *Strategy) DesiredStop(ticker string, e statestore.Entry) (float64, string) {
	p, ok := ParamsFor(ticker)
	if !ok {
		return 0, ""
	}
	level := core.StopLevel(p, e.EntryPrice, e.EntryATR)
	if level <= 0 {
		return 0, ""
	}
	return level, "SL"
}

// Reconstruct поднимает вход по API: средняя цена брокера, последняя BUY-сделка, дневной
// ATR по будним дневкам до дня входа. TakeProfit остаётся 0 (цель ядро пересчитывает
// от цены входа), MaxFav равен цене входа и ядром не читается.
func (s *Strategy) Reconstruct(ctx context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	p, ok := ParamsFor(in.Ticker)
	if !ok {
		return statestore.Entry{}, fmt.Errorf("pdl_rsi reconstruct: %s не зарегистрирован", in.Ticker)
	}
	entryTime, err := rebuild.LastBuyTime(ctx, in.Trades, in.AccountID, in.InstrumentID, in.Now)
	switch {
	case errors.Is(err, rebuild.ErrNoBuyFill):
		return statestore.Entry{}, fmt.Errorf("pdl_rsi reconstruct: no BUY fill found for %s", in.Ticker)
	case err != nil:
		return statestore.Entry{}, fmt.Errorf("pdl_rsi reconstruct: %w", err)
	}
	atr, err := rebuild.WeekdayDailyATRBefore(ctx, in.Candles, in.InstrumentID, entryTime, p.DailyATRPeriod)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("pdl_rsi reconstruct: %w", err)
	}
	if atr <= 0 {
		// Нулевой ATR — стоп вплотную к цене входа; лучше отказ и решение человека.
		return statestore.Entry{}, fmt.Errorf("pdl_rsi reconstruct: %s: cannot rebuild daily ATR as of %s", in.Ticker, entryTime.Format("2006-01-02T15:04:05Z07:00"))
	}
	return statestore.Entry{
		Ticker:     in.Ticker,
		EntryTime:  entryTime,
		EntryPrice: in.PurchasePrice,
		EntryATR:   atr,
		MaxFav:     in.PurchasePrice,
	}, nil
}

// Notify шлёт в тему pdl_rsi. Метку стратегии несёт само сообщение: раннер рендерит его
// через notifier с Label(). Сбой доставки — ERROR-лог, как у rsi_pullback.
func (s *Strategy) Notify(msg string) {
	if !s.cfg.NotifyEnabled || s.tg == nil {
		return
	}
	if err := s.tg.SendMessage(msg); err != nil {
		logger.ErrorContext(context.Background(), fmt.Sprintf("pdl_rsi: уведомление не доставлено: %v", err))
	}
}
