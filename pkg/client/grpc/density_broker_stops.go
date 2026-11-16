package grpc

import (
	"context"
	"fmt"
	"time"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/density/live"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func (b *DensityBroker) PostStop(parent context.Context, r live.StopRequest) (string, error) {
	if !b.trade {
		return "", fmt.Errorf("check-only broker cannot send stop orders")
	}
	if err := b.validate(r.InstrumentID, r.Key, r.Lots, r.Price, true); err != nil {
		return "", err
	}
	req := &investapi.PostStopOrderRequest{AccountId: b.account, InstrumentId: r.InstrumentID, OrderId: r.Key, Quantity: 1, StopPrice: quote(r.Price), Direction: investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL, ExpirationType: investapi.StopOrderExpirationType_STOP_ORDER_EXPIRATION_TYPE_GOOD_TILL_CANCEL, StopOrderType: investapi.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS, ExchangeOrderType: investapi.ExchangeOrderType_EXCHANGE_ORDER_TYPE_MARKET}
	if r.Kind == "take" {
		req.StopOrderType = investapi.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT
		req.TakeProfitType = investapi.TakeProfitType_TAKE_PROFIT_TYPE_REGULAR
		req.ExchangeOrderType = investapi.ExchangeOrderType_EXCHANGE_ORDER_TYPE_LIMIT
		req.Price = quote(r.Price)
	} else if r.Kind != "stop" {
		return "", fmt.Errorf("invalid stop kind")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	resp, err := b.stops.PostStopOrder(ctx, req, NewRPCCredential(b.auth))
	if err != nil {
		return "", brokerError(err)
	}
	if resp.GetStopOrderId() == "" || resp.GetOrderRequestId() != "" && resp.GetOrderRequestId() != r.Key {
		return "", fmt.Errorf("stop placement identifiers missing or mismatched; result unknown")
	}
	return resp.GetStopOrderId(), nil
}

func (b *DensityBroker) Stops(parent context.Context, from time.Time) ([]live.BrokerStop, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	resp, err := b.stops.GetStopOrders(ctx, &investapi.GetStopOrdersRequest{AccountId: b.account, Status: investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ALL, From: timestamppb.New(from), To: timestamppb.Now()}, NewRPCCredential(b.auth))
	if err != nil {
		return nil, brokerError(err)
	}
	var result []live.BrokerStop
	for _, s := range resp.GetStopOrders() {
		kind := ""
		switch s.GetStatus() {
		case investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE:
			kind = "active"
		case investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED:
			kind = "executed"
		case investapi.StopOrderStatusOption_STOP_ORDER_STATUS_CANCELED:
			kind = "cancelled"
		case investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXPIRED:
			kind = "expired"
		default:
			return nil, fmt.Errorf("unrecognized broker stop status")
		}
		result = append(result, live.BrokerStop{ID: s.GetStopOrderId(), InstrumentID: s.GetInstrumentUid(), Status: kind, ChildID: s.GetExchangeOrderId(), Activated: s.GetActivationDateTime() != nil})
	}
	return result, nil
}

func (b *DensityBroker) CancelStop(parent context.Context, id string) error {
	if !b.trade {
		return fmt.Errorf("check-only broker cannot cancel stop orders")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	_, err := b.stops.CancelStopOrder(ctx, &investapi.CancelStopOrderRequest{AccountId: b.account, StopOrderId: id}, NewRPCCredential(b.auth))
	return brokerError(err)
}
