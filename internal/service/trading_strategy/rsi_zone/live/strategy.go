package live

import (
	"context"
	"errors"
	"fmt"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/rebuild"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// Name — имя стратегии в стейте счёта.
const Name = "rsi_zone"

const alertLabel = "RSI Zone"

// Strategy — rsi_zone как гость раннера счёта.
type Strategy struct {
	cfg *config.RSIZoneConfig
	tg  telegram.Client
}

var _ adapter.Strategy = (*Strategy)(nil)

func New(cfg *config.RSIZoneConfig, tg telegram.Client) *Strategy {
	return &Strategy{cfg: cfg, tg: tg}
}

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

// DesiredStop — стоп, замороженный на входе: у rsi_zone нет трейла, и уровень не меняется
// за всю жизнь позиции. Раннер переставляет заявку только при расхождении размера или после
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
// ATR по будним дневкам до дня входа. Цели и трейла у стратегии нет — TakeProfit 0,
// MaxFav равен цене входа и ядром не читается.
func (s *Strategy) Reconstruct(ctx context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	p, ok := ParamsFor(in.Ticker)
	if !ok {
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: %s не зарегистрирован", in.Ticker)
	}
	entryTime, err := rebuild.LastBuyTime(ctx, in.Trades, in.AccountID, in.InstrumentID, in.Now)
	switch {
	case errors.Is(err, rebuild.ErrNoBuyFill):
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: no BUY fill found for %s", in.Ticker)
	case err != nil:
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: %w", err)
	}
	atr, err := rebuild.WeekdayDailyATRBefore(ctx, in.Candles, in.InstrumentID, entryTime, p.DailyATRPeriod)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: %w", err)
	}
	if atr <= 0 {
		// Нулевой ATR — стоп вплотную к цене входа; лучше отказ и решение человека.
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: %s: cannot rebuild daily ATR as of %s", in.Ticker, entryTime.Format("2006-01-02T15:04:05Z07:00"))
	}
	return statestore.Entry{
		Ticker:     in.Ticker,
		EntryTime:  entryTime,
		EntryPrice: in.PurchasePrice,
		EntryATR:   atr,
		MaxFav:     in.PurchasePrice,
	}, nil
}

// Notify шлёт в тему rsi_zone с меткой стратегии: сообщения входа/выхода/стопа из общего
// notifier метки не несут. Сбой доставки — ERROR-лог, как у rsi_pullback.
func (s *Strategy) Notify(msg string) {
	if !s.cfg.NotifyEnabled || s.tg == nil {
		return
	}
	if err := s.tg.SendMessage("[" + alertLabel + "] " + msg); err != nil {
		logger.ErrorContext(context.Background(), fmt.Sprintf("rsi_zone: уведомление не доставлено: %v", err))
	}
}
