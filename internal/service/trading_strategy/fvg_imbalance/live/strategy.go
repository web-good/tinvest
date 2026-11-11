// Package live adapts fvg_imbalance to the shared account runner.
package live

import (
	"context"
	"fmt"
	"time"

	"tinvest/internal/config"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/bspb"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sngs"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/rebuild"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

type Strategy struct {
	cfg *config.FVGImbalanceConfig
	tg  telegram.Client
}

var _ adapter.Strategy = (*Strategy)(nil)
var _ adapter.BarInterval = (*Strategy)(nil)

func New(cfg *config.FVGImbalanceConfig, tg telegram.Client) *Strategy { return &Strategy{cfg, tg} }
func (*Strategy) Name() string                                         { return "fvg_imbalance" }
func (*Strategy) Label() string                                        { return "FVG Imbalance" }
func (s *Strategy) Tickers() []string                                  { return s.cfg.Tickers }
func (s *Strategy) BuyPct() float64                                    { return s.cfg.BuyPct }
func (s *Strategy) TradeEnabled() bool                                 { return s.cfg.TradeEnabled }
func (*Strategy) BarInterval() enum.Interval                           { return enum.Minutes30 }
func params(ticker string) (core.Params, bool) {
	switch ticker {
	case "BSPB":
		return bspb.Params(), true
	case "SNGS":
		return sngs.Params(), true
	}
	return core.Params{}, false
}
func (*Strategy) Decider(ticker string) (adapter.Decider, bool) {
	p, ok := params(ticker)
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, 30*time.Minute, p), true
}
func (*Strategy) DesiredStop(ticker string, e statestore.Entry) (float64, string) {
	p, ok := params(ticker)
	if !ok {
		return 0, ""
	}
	stop := e.EntryPrice - p.StopATR*e.EntryATR
	if stop <= 0 {
		return 0, ""
	}
	return stop, "SL"
}
func (*Strategy) Reconstruct(ctx context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	p, ok := params(in.Ticker)
	if !ok {
		return statestore.Entry{}, fmt.Errorf("fvg_imbalance: unknown ticker %s", in.Ticker)
	}
	entryTime, err := rebuild.LastBuyTime(ctx, in.Trades, in.AccountID, in.InstrumentID, in.Now)
	if err != nil {
		return statestore.Entry{}, err
	}
	atr, err := rebuild.WeekdayDailyATRBefore(ctx, in.Candles, in.InstrumentID, entryTime, p.ATRPeriod)
	if err != nil {
		return statestore.Entry{}, err
	}
	if atr <= 0 || in.PurchasePrice <= p.StopATR*atr {
		return statestore.Entry{}, fmt.Errorf("fvg_imbalance: cannot reconstruct stop for %s", in.Ticker)
	}
	entry := statestore.Entry{Ticker: in.Ticker, EntryTime: entryTime, EntryPrice: in.PurchasePrice, EntryATR: atr, MaxFav: in.PurchasePrice, StopLoss: in.PurchasePrice - p.StopATR*atr}
	if p.TakeProfitATR > 0 {
		entry.TakeProfit = in.PurchasePrice + p.TakeProfitATR*atr
	}
	return entry, nil
}
func (s *Strategy) Notify(msg string) {
	if s.cfg.NotifyEnabled && s.tg != nil {
		if err := s.tg.SendMessage(msg); err != nil {
			logger.ErrorContext(context.Background(), fmt.Sprintf("fvg_imbalance: notification: %v", err))
		}
	}
}
