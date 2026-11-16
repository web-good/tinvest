package live

import (
	"context"
	"fmt"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
)

var _ adapter.BrokerTakeProfit = (*pullbackStrategy)(nil)

func (p *pullbackStrategy) TakeProfit(ticker string, entry statestore.Entry) float64 {
	if entry.TakeProfit > 0 {
		return entry.TakeProfit
	}
	params := mustParams(ticker)
	if params.TPDailyATR <= 0 || entry.EntryPrice <= 0 || entry.EntryATR <= 0 {
		return 0
	}
	return entry.EntryPrice + params.TPDailyATR*entry.EntryATR
}

// PlaceTakeProfit is Pullback's broker order path, independent of Zone and FVG.
func (p *pullbackStrategy) PlaceTakeProfit(ctx context.Context, stops *stoporders.Executor, share *imodel.Share, entry statestore.Entry) (stoporders.Result, error) {
	if !p.cfg.TradeEnabled || entry.TakeProfit <= 0 {
		return stoporders.Result{}, nil
	}
	if share.Lot <= 0 || entry.Quantity < int64(share.Lot) {
		return stoporders.Result{}, fmt.Errorf("rsi_pullback: invalid take-profit quantity")
	}
	return stops.PlaceTakeProfit(ctx, share.ID, entry.Quantity/int64(share.Lot), entry.TakeProfit, share.MinPriceIncrement)
}

// pairedOrders extends account reconciliation without coupling strategy placement code.
func pairedOrders(sl *slot) bool {
	_, ok := sl.strat.(adapter.BrokerTakeProfit)
	return sl.strat.Name() == "fvg_imbalance" || ok && sl.strat.TradeEnabled()
}

// Old stop-only entries keep their existing exit path until a TP is submitted.
func pairedPosition(sl *slot, entry statestore.Entry) bool {
	return sl.strat.Name() == "fvg_imbalance" || entry.TakeProfitOrderID != "" || entry.FVGTargetPending ||
		entry.FVGStopPending || entry.FVGRecovering || entry.FVGExitStopID != "" || entry.FVGExitOrderID != ""
}

func (s *service) placeTarget(ctx context.Context, pc *passCtx, sl *slot, ticker string, share *imodel.Share) error {
	if sl.strat.Name() == "fvg_imbalance" {
		return s.placeFVGTarget(ctx, pc, sl, ticker, share)
	}
	owner, ok := sl.strat.(adapter.BrokerTakeProfit)
	if !ok || !sl.strat.TradeEnabled() {
		return nil
	}
	entry := pc.state[ticker]
	if entry.TakeProfit <= 0 || entry.TakeProfitOrderID != "" {
		return nil
	}
	if pc.listErr != nil || share.Lot <= 0 {
		// Stop management already reports unavailable snapshots and invalid lots.
		return nil
	}
	if _, reason := sl.strat.DesiredStop(ticker, entry); reason != "" && entry.StopOrderID == "" && !entry.FVGStopPending {
		// Do not add a second SELL when an unknown/uncancelled stop blocks protection.
		return nil
	}
	if entry.FVGStopPending || entry.FVGTargetPending || entry.FVGExitStopID != "" || entry.FVGExitOrderID != "" {
		return fmt.Errorf("take profit waits for unresolved broker operation")
	}
	entry.FVGTargetPending, entry.FVGTargetPendingAt = true, s.now()
	pc.state[ticker] = entry
	if err := pc.store.Save(pc.state); err != nil {
		return err
	}
	res, err := owner.PlaceTakeProfit(ctx, sl.stops, share, entry)
	if err != nil {
		return err
	}
	if res.Placed {
		entry.TakeProfitOrderID, entry.FVGTargetPending = res.OrderID, false
		pc.state[ticker] = entry
		return pc.store.Save(pc.state)
	}
	return nil
}
