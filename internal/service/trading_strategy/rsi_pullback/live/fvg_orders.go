package live

import (
	"context"
	"fmt"

	imodel "tinvest/internal/model"
	pb "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/notifier"
	"tinvest/internal/utils"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// reconcileFVG treats a triggered stop as an order, not as a fill. It keeps the
// original quantity until the portfolio has caught up with a terminal execution.
func (s *service) reconcileFVG(ctx context.Context, pc *passCtx, sl *slot, ticker string, sh *imodel.Share, pos *grpcmodel.Position) error {
	e, ok := pc.state[ticker]
	if !ok {
		return nil
	}
	if pc.listErr != nil {
		return fmt.Errorf("cannot reconcile FVG orders: %w", pc.listErr)
	}
	save := func() error { pc.state[ticker] = e; return pc.store.Save(pc.state) }
	if pc.shared && (e.FVGRecovering || e.FVGStopPending || e.FVGTargetPending) {
		return fmt.Errorf("shared position has ambiguous stop ownership; reconciliation required")
	}
	if e.FVGRecovering {
		triggered, err := sl.stops.TriggeredSince(ctx, sh.ID, e.EntryTime)
		if err != nil {
			return err
		}
		if len(triggered) > 1 {
			return fmt.Errorf("multiple executed FVG stops after entry; manual reconciliation required")
		}
		if len(triggered) == 1 {
			e.FVGExitStopID = triggered[0].GetStopOrderId()
			e.FVGExitOrderID = triggered[0].GetExchangeOrderId()
			if original := triggered[0].GetLotsRequested() * int64(sh.Lot); original > e.Quantity {
				e.Quantity = original
			}
		}
		e.FVGRecovering = false
		if err := save(); err != nil {
			return err
		}
	}
	// Reattach surviving orders after loss of local IDs, but refuse ambiguous sets.
	for _, a := range pc.stopByID {
		if a.InstrumentUID != sh.ID {
			continue
		}
		if pc.shared && a.StopOrderID != e.StopOrderID && a.StopOrderID != e.TakeProfitOrderID {
			return fmt.Errorf("unknown stop on shared instrument; reconciliation required")
		}
		switch a.Type {
		case pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS:
			if e.StopOrderID == "" {
				e.StopOrderID = a.StopOrderID
			} else if e.StopOrderID != a.StopOrderID {
				return fmt.Errorf("multiple FVG stop losses")
			}
		case pb.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT:
			if e.TakeProfitOrderID == "" {
				e.TakeProfitOrderID = a.StopOrderID
			} else if e.TakeProfitOrderID != a.StopOrderID {
				return fmt.Errorf("multiple FVG take profits")
			}
		default:
			return fmt.Errorf("unknown FVG stop order type")
		}
	}
	if e.FVGStopPending {
		if e.StopOrderID == "" {
			triggered, err := sl.stops.TriggeredSince(ctx, sh.ID, e.EntryTime)
			if err != nil {
				return err
			}
			var found *pb.StopOrder
			for _, order := range triggered {
				if order.GetOrderType() != pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS || (!e.FVGStopPendingAt.IsZero() && !order.GetActivationDateTime().AsTime().After(e.FVGStopPendingAt)) {
					continue
				}
				if found != nil {
					return fmt.Errorf("ambiguous FVG stop placement; manual reconciliation required")
				}
				found = order
			}
			if found == nil {
				return fmt.Errorf("FVG stop placement outcome unknown; manual reconciliation required")
			}
			if e.FVGExitStopID != "" && e.FVGExitStopID != found.GetStopOrderId() {
				return fmt.Errorf("multiple FVG exits triggered; manual reconciliation required")
			}
			e.StopOrderID = found.GetStopOrderId()
			e.FVGExitStopID, e.FVGExitOrderID = found.GetStopOrderId(), found.GetExchangeOrderId()
		}
		e.FVGStopPending = false
	}
	if e.FVGTargetPending {
		if e.TakeProfitOrderID == "" {
			triggered, err := sl.stops.TriggeredSince(ctx, sh.ID, e.EntryTime)
			if err != nil {
				return err
			}
			var target *pb.StopOrder
			for _, order := range triggered {
				if order.GetOrderType() != pb.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT || (!e.FVGTargetPendingAt.IsZero() && !order.GetActivationDateTime().AsTime().After(e.FVGTargetPendingAt)) {
					continue
				}
				if target != nil {
					return fmt.Errorf("ambiguous FVG target placement; manual reconciliation required")
				}
				target = order
			}
			if target == nil {
				return fmt.Errorf("FVG target placement outcome unknown; manual reconciliation required")
			}
			if e.FVGExitStopID != "" && e.FVGExitStopID != target.GetStopOrderId() {
				return fmt.Errorf("multiple FVG exits triggered; manual reconciliation required")
			}
			e.TakeProfitOrderID = target.GetStopOrderId()
			e.FVGExitStopID, e.FVGExitOrderID = target.GetStopOrderId(), target.GetExchangeOrderId()
		}
		e.FVGTargetPending = false
	}
	if err := save(); err != nil {
		return err
	}
	for _, id := range []string{e.StopOrderID, e.TakeProfitOrderID} {
		if id == "" || id == e.FVGExitStopID {
			continue
		}
		if _, alive := pc.stopByID[id]; alive {
			continue
		}
		child, fired, err := sl.stops.Triggered(ctx, id)
		if err != nil {
			return err
		}
		if fired {
			if e.FVGExitStopID != "" && e.FVGExitStopID != id {
				return fmt.Errorf("both FVG orders triggered; manual reconciliation required")
			}
			e.FVGExitStopID, e.FVGExitOrderID = id, child
			if err := save(); err != nil {
				return err
			}
		} else if id == e.StopOrderID {
			e.StopOrderID = ""
		} else {
			e.TakeProfitOrderID = ""
		}
	}
	if err := save(); err != nil {
		return err
	}
	if e.FVGExitStopID != "" {
		// Cancel all siblings before considering any replacement or market sale.
		for _, id := range []string{e.StopOrderID, e.TakeProfitOrderID} {
			if id == "" || id == e.FVGExitStopID {
				continue
			}
			if _, alive := pc.stopByID[id]; alive {
				if err := sl.stops.Cancel(ctx, id); err != nil {
					return err
				}
			}
			if id == e.StopOrderID {
				e.StopOrderID = ""
			} else {
				e.TakeProfitOrderID = ""
			}
			if err := save(); err != nil {
				return err
			}
		}
		if e.FVGExitOrderID == "" && e.FVGExitStopID != "market" {
			child, fired, err := sl.stops.Triggered(ctx, e.FVGExitStopID)
			if err != nil {
				return err
			}
			if fired {
				e.FVGExitOrderID = child
				if err := save(); err != nil {
					return err
				}
			}
		}
		if e.FVGExitOrderID == "" && e.ExitRequestID != "" {
			order, err := sl.exec.RequestState(ctx, e.ExitRequestID)
			if err != nil {
				return err
			}
			e.FVGExitOrderID = order.GetOrderId()
			if err := save(); err != nil {
				return err
			}
		}
		if e.FVGExitOrderID == "" {
			return fmt.Errorf("triggered FVG order has no exchange order ID")
		}
		state, err := sl.exec.OrderState(ctx, e.FVGExitOrderID)
		if err != nil {
			return err
		}
		if sh.Lot <= 0 {
			return fmt.Errorf("invalid lot size")
		}
		if pc.shared || e.ExitRequestID != "" {
			if state.GetOrderId() != e.FVGExitOrderID {
				return fmt.Errorf("exit order ID mismatch")
			}
			if err := validateOwnedOrder(state, sh.ID, pb.OrderDirection_ORDER_DIRECTION_SELL, e.Quantity/int64(sh.Lot)); err != nil {
				return err
			}
		}
		switch state.GetExecutionReportStatus() {
		case pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL, pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED, pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED:
		default:
			return nil
		}
		if sh.Lot <= 0 {
			return fmt.Errorf("invalid lot size")
		}
		remaining := e.Quantity - state.GetLotsExecuted()*int64(sh.Lot)
		if remaining < 0 {
			return fmt.Errorf("FVG execution exceeds recorded position")
		}
		if (pos == nil || pc.shared) && remaining == 0 {
			reason, price := e.StopReason, e.StopPrice
			if e.FVGExitStopID == e.TakeProfitOrderID {
				reason, price = "TP", e.TakeProfit
			}
			if e.FVGExitStopID == "market" {
				reason = e.PendingExit
			}
			if p := state.GetAveragePositionPrice(); p != nil {
				if average := utils.CombinePrice(p.GetUnits(), p.GetNano()); average > 0 {
					price = average
				}
			}
			if claim, ok := pc.claims[ticker]; ok {
				if claim.Owner != sl.strat.Name() || claim.InstrumentID != sh.ID {
					return fmt.Errorf("candle reservation does not match the confirmed exit")
				}
				// The durable entry still excludes other candle strategies if saving the close fails.
				if err := s.coordinator.Remove(pc.lease, claim); err != nil {
					return err
				}
				delete(pc.claims, ticker)
			}
			delete(pc.state, ticker)
			if err := pc.store.Save(pc.state); err != nil {
				pc.state[ticker] = e
				return err
			}
			sl.strat.Notify(notifier.Exit(sl.strat.Label(), ticker, reason, price, e.Quantity, false))
			return nil
		}
		if pc.shared && pos != nil {
			copyPosition := *pos
			copyPosition.Quantity = remaining
			pos = &copyPosition
		}
		if pos == nil || pos.Quantity > remaining {
			return nil
		}
		// The terminal order and portfolio agree; resume protection on the residual.
		e.Quantity = pos.Quantity
		if e.StopOrderID == e.FVGExitStopID {
			e.StopOrderID = ""
		}
		if e.TakeProfitOrderID == e.FVGExitStopID {
			e.TakeProfitOrderID = ""
		}
		e.FVGExitOrderID, e.FVGExitStopID, e.ExitRequestID = "", "", ""
		return save()
	}
	if pos == nil {
		if pc.shared {
			return fmt.Errorf("broker snapshot has no owned inventory and no confirmed own sale; reconciliation required")
		}
		if pc.now.Sub(e.EntryTime) < freshEntryGrace {
			return nil
		}
		for _, id := range []string{e.StopOrderID, e.TakeProfitOrderID} {
			if id == "" {
				continue
			}
			if err := sl.stops.Cancel(ctx, id); err != nil {
				return err
			}
			if id == e.StopOrderID {
				e.StopOrderID = ""
			} else {
				e.TakeProfitOrderID = ""
			}
			if err := save(); err != nil {
				return err
			}
		}
		delete(pc.state, ticker)
		return pc.store.Save(pc.state)
	}
	// Resize the target before any new sale; the stop is resized by manage.
	if a, alive := pc.stopByID[e.TakeProfitOrderID]; alive && (e.TakeProfit <= 0 || sh.Lot <= 0 || a.Lots != pos.Quantity/int64(sh.Lot)) {
		if err := sl.stops.Cancel(ctx, e.TakeProfitOrderID); err != nil {
			return err
		}
		e.TakeProfitOrderID = ""
		if err := save(); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) placeFVGTarget(ctx context.Context, pc *passCtx, sl *slot, ticker string, sh *imodel.Share) error {
	e := pc.state[ticker]
	if e.FVGStopPending {
		return fmt.Errorf("FVG stop placement outcome unknown")
	}
	if e.TakeProfit <= 0 || e.TakeProfitOrderID != "" || !sl.strat.TradeEnabled() {
		return nil
	}
	if pc.listErr != nil {
		return fmt.Errorf("cannot place FVG target: %w", pc.listErr)
	}
	if sh.Lot <= 0 || e.Quantity < int64(sh.Lot) {
		return fmt.Errorf("invalid FVG target quantity")
	}
	if e.FVGTargetPending {
		return fmt.Errorf("FVG target placement outcome unknown")
	}
	e.FVGTargetPending = true
	e.FVGTargetPendingAt = s.now()
	pc.state[ticker] = e
	if err := pc.store.Save(pc.state); err != nil {
		return err
	}
	res, err := sl.stops.PlaceTakeProfit(ctx, sh.ID, e.Quantity/int64(sh.Lot), e.TakeProfit, sh.MinPriceIncrement)
	if err != nil {
		return err
	}
	if res.Placed {
		e.TakeProfitOrderID = res.OrderID
		e.FVGTargetPending = false
		pc.state[ticker] = e
		return pc.store.Save(pc.state)
	}
	return nil
}
