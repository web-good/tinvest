package live

import (
	"context"
	"errors"
	"fmt"
	imodel "tinvest/internal/model"
	pb "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	"tinvest/internal/utils"
)

func validateOwnedOrder(o *pb.OrderState, uid string, direction pb.OrderDirection, maxLots int64) error {
	if o == nil || o.GetOrderId() == "" || o.GetInstrumentUid() != uid || o.GetDirection() != direction || o.GetLotsRequested() <= 0 || o.GetLotsExecuted() < 0 || o.GetLotsExecuted() > o.GetLotsRequested() || maxLots > 0 && o.GetLotsRequested() > maxLots {
		return fmt.Errorf("broker order identity or quantity mismatch; reconciliation required")
	}
	return nil
}
func terminalOrder(o *pb.OrderState) bool {
	switch o.GetExecutionReportStatus() {
	case pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL, pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED, pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED:
		return true
	}
	return false
}

// reconcilePendingEntry protects confirmed fills before cancelling the unfinished
// BUY. Its cumulative fill counters survive stops executing during cancellation.
func (s *service) reconcilePendingEntry(ctx context.Context, pc *passCtx, sl *slot, ticker string, sh *imodel.Share) (bool, error) {
	return s.pendingEntryAttempt(ctx, pc, sl, ticker, sh, 3)
}
func (s *service) pendingEntryAttempt(ctx context.Context, pc *passCtx, sl *slot, ticker string, sh *imodel.Share, attempts int) (bool, error) {
	if attempts <= 0 {
		return false, fmt.Errorf("entry fills changed during bounded reconciliation; retry required")
	}
	e := pc.state[ticker]
	buy, err := sl.exec.RequestState(ctx, e.EntryRequestID)
	if err != nil {
		return false, err
	}
	if sh.Lot <= 0 {
		return false, fmt.Errorf("invalid lot size")
	}
	if err := validateOwnedOrder(buy, sh.ID, pb.OrderDirection_ORDER_DIRECTION_BUY, e.EntryRequestedLots); err != nil {
		return false, err
	}
	if e.EntryRequestedLots > 0 && e.EntryRequestedLots != buy.GetLotsRequested() {
		return false, fmt.Errorf("entry requested lots mismatch")
	}
	if e.EntryOrderID != "" && e.EntryOrderID != buy.GetOrderId() {
		return false, fmt.Errorf("entry order ID mismatch")
	}
	filled := buy.GetLotsExecuted() * int64(sh.Lot)
	if filled < e.EntryFilledQuantity {
		return false, fmt.Errorf("entry cumulative executions decreased")
	}
	if owner, ok := sl.strat.(adapter.BrokerTakeProfit); ok && filled > 0 {
		if average := buy.GetAveragePositionPrice(); average != nil {
			if price := utils.CombinePrice(average.GetUnits(), average.GetNano()); price > 0 {
				// A pending BUY's average can change as more lots fill. No TP is
				// submitted until that BUY is terminal; freeze from its confirmed average.
				e.EntryPrice = price
				if e.EntryFilledQuantity == 0 {
					e.MaxFav = price
				} else {
					e.MaxFav = max(e.MaxFav, price)
				}
				e.TakeProfit = 0
				e.TakeProfit = owner.TakeProfit(ticker, e)
			}
		}
	}
	e.EntryOrderID = buy.GetOrderId()
	e.EntryFilledQuantity = filled
	save := func() error { pc.state[ticker] = e; return pc.store.Save(pc.state) }
	// Resolve the old protection before replacing it; a triggered stop can still
	// have an unfinished exchange child even after the BUY itself is cancelled.
	if e.StopOrderID != "" {
		if pc.listErr != nil {
			return false, fmt.Errorf("cannot reconcile existing entry protection: %w", pc.listErr)
		}
		if _, alive := pc.stopByID[e.StopOrderID]; !alive {
			child, found, err := sl.stops.Triggered(ctx, e.StopOrderID)
			if err != nil {
				return false, err
			}
			if found {
				sold, err := sl.exec.OrderState(ctx, child)
				if err != nil {
					return false, err
				}
				if err := validateOwnedOrder(sold, sh.ID, pb.OrderDirection_ORDER_DIRECTION_SELL, filled/int64(sh.Lot)); err != nil {
					return false, err
				}
				if sold.GetOrderId() != child {
					return false, fmt.Errorf("entry protection child ID mismatch")
				}
				if !terminalOrder(sold) {
					return false, fmt.Errorf("entry protection child still executing")
				}
				e.EntrySoldQuantity += sold.GetLotsExecuted() * int64(sh.Lot)
			}
			e.StopOrderID = ""
		}
	}
	e.Quantity = filled - e.EntrySoldQuantity
	if e.Quantity < 0 {
		return false, fmt.Errorf("entry protection executions exceed own entry fills")
	}
	if err := save(); err != nil {
		return false, err
	}
	if e.StopOrderID != "" {
		active := pc.stopByID[e.StopOrderID]
		if active.Lots*int64(sh.Lot) != e.Quantity {
			if attempts <= 1 {
				return false, fmt.Errorf("entry fills still changing; existing protection retained for retry")
			}
			if err := sl.stops.Cancel(ctx, e.StopOrderID); err != nil {
				return false, err
			}
			// Re-read executions before replacement: cancellation may race activation.
			// The old ID remains the durable reconciliation key until resolved.
			delete(pc.stopByID, e.StopOrderID)
			return s.pendingEntryAttempt(ctx, pc, sl, ticker, sh, attempts-1)
		}
	} else if e.Quantity > 0 {
		level, reason := sl.strat.DesiredStop(ticker, e)
		if reason == "" {
			return false, fmt.Errorf("confirmed partial entry has no protective stop level")
		}
		// All strategies persist unknown stop submissions on the shared account.
		e = s.replaceStopSafely(ctx, pc, sl, ticker, sh, e, level, reason)
		if err := save(); err != nil {
			return false, err
		}
		if e.StopOrderID == "" {
			return false, fmt.Errorf("partial entry protection placement unconfirmed")
		}
		pc.stopByID[e.StopOrderID] = stoporders.ActiveStop{StopOrderID: e.StopOrderID, InstrumentUID: sh.ID, Type: pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS, Lots: e.Quantity / int64(sh.Lot)}
	}
	if terminalOrder(buy) {
		e.EntryRequestID = ""
		if err := save(); err != nil {
			return false, err
		}
		return true, nil
	}
	// Protection exists before cancellation: a lost cancel response retains both IDs.
	if err := sl.exec.Cancel(ctx, e.EntryOrderID); err != nil {
		return false, fmt.Errorf("cancel unfinished entry: %w", err)
	}
	final, err := sl.exec.RequestState(ctx, e.EntryRequestID)
	if err != nil {
		return false, err
	}
	if err := validateOwnedOrder(final, sh.ID, pb.OrderDirection_ORDER_DIRECTION_BUY, e.EntryRequestedLots); err != nil {
		return false, err
	}
	if final.GetOrderId() != e.EntryOrderID {
		return false, errors.New("entry order ID mismatch after cancellation")
	}
	if final.GetLotsExecuted() != buy.GetLotsExecuted() || !terminalOrder(final) {
		return s.pendingEntryAttempt(ctx, pc, sl, ticker, sh, attempts-1)
	}
	e.EntryRequestID = ""
	return true, save()
}
