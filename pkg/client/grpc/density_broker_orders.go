package grpc

import (
	"context"
	"fmt"
	"math"
	"time"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/density/live"
)

func (b *DensityBroker) PostOrder(parent context.Context, r live.OrderRequest) (live.Order, error) {
	if !b.trade {
		return live.Order{}, fmt.Errorf("check-only broker cannot send orders")
	}
	if r.Kind != "buy" && r.Kind != "exit" {
		return live.Order{}, fmt.Errorf("invalid density order kind")
	}
	if err := b.validate(r.InstrumentID, r.Key, r.Lots, r.Price, r.Kind == "buy"); err != nil {
		return live.Order{}, err
	}
	req := &investapi.PostOrderRequest{AccountId: b.account, InstrumentId: r.InstrumentID, OrderId: r.Key, Quantity: 1, Direction: investapi.OrderDirection_ORDER_DIRECTION_BUY, OrderType: investapi.OrderType_ORDER_TYPE_LIMIT, Price: quote(r.Price)}
	if r.Kind == "exit" {
		req.Direction = investapi.OrderDirection_ORDER_DIRECTION_SELL
		req.OrderType = investapi.OrderType_ORDER_TYPE_MARKET
		req.Price = nil
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	resp, err := b.orders.PostOrder(ctx, req, NewRPCCredential(b.auth))
	if err != nil {
		return live.Order{}, brokerError(err)
	}
	if resp.GetInstrumentUid() != "" && resp.GetInstrumentUid() != r.InstrumentID || resp.GetDirection() != req.Direction || resp.GetOrderRequestId() != "" && resp.GetOrderRequestId() != r.Key {
		return live.Order{}, fmt.Errorf("broker placement identifiers mismatch; result unknown")
	}
	o, err := normalizeOrder(resp.GetOrderId(), resp.GetOrderRequestId(), r.InstrumentID, resp.GetDirection(), resp.GetExecutionReportStatus(), resp.GetLotsRequested(), resp.GetLotsExecuted(), resp.GetExecutedCommission())
	if err != nil {
		return o, err
	}
	avg, err := moneyNano(resp.GetExecutedOrderPrice())
	if err != nil {
		return o, nil // accounting is unknown; the confirmed lot still needs protection
	}
	lot := b.instruments[r.InstrumentID].Lot
	if o.Filled > 0 && avg > 0 && avg <= math.MaxInt64/lot/o.Filled {
		o.Amount = avg * lot * o.Filled
	}
	return o, nil
}

func normalizeOrder(id, key, uid string, direction investapi.OrderDirection, s investapi.OrderExecutionReportStatus, lots, filled int64, commission *investapi.MoneyValue) (live.Order, error) {
	o := live.Order{ID: id, Key: key, InstrumentID: uid, Lots: lots, Filled: filled, Sent: true}
	switch direction {
	case investapi.OrderDirection_ORDER_DIRECTION_BUY:
		o.Kind = "buy"
	case investapi.OrderDirection_ORDER_DIRECTION_SELL:
		o.Kind = "sell"
	default:
		return o, fmt.Errorf("unknown broker order direction")
	}
	switch s {
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL:
		o.Status = "filled"
		o.Terminal = true
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED:
		o.Status = "rejected"
		o.Terminal = true
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED:
		o.Status = "cancelled"
		o.Terminal = true
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_NEW:
		o.Status = "new"
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL:
		o.Status = "partial"
	default:
		return o, fmt.Errorf("unknown broker execution status")
	}
	if id == "" || lots != 1 || filled < 0 || filled > lots || o.Status == "filled" && filled != lots {
		return o, fmt.Errorf("invalid broker lot quantities")
	}
	fees, err := moneyNano(commission)
	if err != nil || fees < 0 {
		return o, nil // do not hide confirmed inventory because commission is unavailable
	}
	o.Fees = fees
	o.FeesKnown = commission != nil
	return o, nil
}

func (b *DensityBroker) Order(parent context.Context, id string, request bool) (live.Order, error) {
	typ := investapi.OrderIdType_ORDER_ID_TYPE_EXCHANGE
	if request {
		typ = investapi.OrderIdType_ORDER_ID_TYPE_REQUEST
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	resp, err := b.orders.GetOrderState(ctx, &investapi.GetOrderStateRequest{AccountId: b.account, OrderId: id, OrderIdType: &typ}, NewRPCCredential(b.auth))
	if err != nil {
		return live.Order{}, brokerError(err)
	}
	i, ok := b.instruments[resp.GetInstrumentUid()]
	if !ok {
		return live.Order{}, fmt.Errorf("broker order belongs to an unselected instrument")
	}
	o, err := normalizeOrder(resp.GetOrderId(), resp.GetOrderRequestId(), i.ID, resp.GetDirection(), resp.GetExecutionReportStatus(), resp.GetLotsRequested(), resp.GetLotsExecuted(), resp.GetExecutedCommission())
	if err != nil {
		return o, err
	}
	if request && o.Key != "" && o.Key != id || !request && o.ID != id {
		return o, fmt.Errorf("broker order lookup identifiers mismatch")
	}
	var stageLots, amount int64
	for _, s := range resp.GetStages() {
		p, err := moneyNano(s.GetPrice())
		qty := s.GetQuantity()
		if err != nil || p <= 0 || qty < 0 || qty > 1 || i.Lot <= 0 || p > math.MaxInt64/i.Lot || qty > 0 && p*i.Lot > math.MaxInt64/qty {
			return o, nil // retain execution status, report unknown notional
		}
		value := p * i.Lot * qty
		if amount > math.MaxInt64-value {
			return o, nil
		}
		amount += value
		stageLots += qty
	}
	if stageLots == o.Filled {
		o.Amount = amount
	}
	return o, nil
}

func (b *DensityBroker) CancelOrder(parent context.Context, id string, request bool) error {
	if !b.trade {
		return fmt.Errorf("check-only broker cannot cancel orders")
	}
	typ := investapi.OrderIdType_ORDER_ID_TYPE_EXCHANGE
	if request {
		typ = investapi.OrderIdType_ORDER_ID_TYPE_REQUEST
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	_, err := b.orders.CancelOrder(ctx, &investapi.CancelOrderRequest{AccountId: b.account, OrderId: id, OrderIdType: &typ}, NewRPCCredential(b.auth))
	return brokerError(err)
}
