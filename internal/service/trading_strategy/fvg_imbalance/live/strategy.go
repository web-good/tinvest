// Package live adapts fvg_imbalance to the shared account runner.
package live

import (
	"context"
	"fmt"
	"time"

	"tinvest/internal/config"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/afks"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/alrs"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/bspb"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/fixr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/gazp"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/gmkn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/irao"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/ivat"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/lsrg"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/magn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mgnt"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/moex"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mtlr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/phor"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/posi"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/reni"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/rosn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sber"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sberp"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sngs"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/spbe"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/vkco"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/vtbr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/x5"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/ydex"
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
	case "AFKS":
		return afks.Params(), true
	case "ALRS":
		return alrs.Params(), true
	case "BSPB":
		return bspb.Params(), true
	case "FIXR":
		return fixr.Params(), true
	case "GAZP":
		return gazp.Params(), true
	case "GMKN":
		return gmkn.Params(), true
	case "IRAO":
		return irao.Params(), true
	case "IVAT":
		return ivat.Params(), true
	case "LSRG":
		return lsrg.Params(), true
	case "MAGN":
		return magn.Params(), true
	case "MGNT":
		return mgnt.Params(), true
	case "MTLR":
		return mtlr.Params(), true
	case "MOEX":
		return moex.Params(), true
	case "PHOR":
		return phor.Params(), true
	case "POSI":
		return posi.Params(), true
	case "RENI":
		return reni.Params(), true
	case "ROSN":
		return rosn.Params(), true
	case "SBER":
		return sber.Params(), true
	case "SBERP":
		return sberp.Params(), true
	case "SNGS":
		return sngs.Params(), true
	case "SPBE":
		return spbe.Params(), true
	case "YDEX":
		return ydex.Params(), true
	case "VTBR":
		return vtbr.Params(), true
	case "VKCO":
		return vkco.Params(), true
	case "X5":
		return x5.Params(), true
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
