package live

import (
	"context"
	"fmt"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

var _ adapter.BrokerTakeProfit = (*Strategy)(nil)

// TakeProfit retains an existing frozen target, including after a parameter change.
func (s *Strategy) TakeProfit(ticker string, entry statestore.Entry) float64 {
	if entry.TakeProfit > 0 {
		return entry.TakeProfit
	}
	p, ok := ParamsFor(ticker)
	if !ok {
		return 0
	}
	return core.TargetLevel(p, entry.EntryPrice, entry.EntryATR)
}

// PlaceTakeProfit is Zone's broker order path; the account runner owns the intent/ID.
func (s *Strategy) PlaceTakeProfit(ctx context.Context, stops *stoporders.Executor, share *imodel.Share, entry statestore.Entry) (stoporders.Result, error) {
	if !s.cfg.TradeEnabled || entry.TakeProfit <= 0 {
		return stoporders.Result{}, nil
	}
	if share.Lot <= 0 || entry.Quantity < int64(share.Lot) {
		return stoporders.Result{}, fmt.Errorf("rsi_zone: invalid take-profit quantity")
	}
	return stops.PlaceTakeProfit(ctx, share.ID, entry.Quantity/int64(share.Lot), entry.TakeProfit, share.MinPriceIncrement)
}
