// Package executor places (or dry-runs) whole-position market orders for the live
// strategy runners. Each order carries a fresh UUID order_id for idempotency.
package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/utils"
)

// OrdersClient is the slice of the orders client the executor needs.
type OrdersClient interface {
	GetOrders(context.Context, *investapi.GetOrdersRequest, ...grpc.CallOption) (*investapi.GetOrdersResponse, error)
	CancelOrder(context.Context, *investapi.CancelOrderRequest, ...grpc.CallOption) (*investapi.CancelOrderResponse, error)
	GetOrderState(ctx context.Context, in *investapi.GetOrderStateRequest, opts ...grpc.CallOption) (*investapi.OrderState, error)
	PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error)
}

// Result reports the effect of an order attempt. When Placed is false (dry-run) the
// caller falls back to the signal's price and the requested quantity.
type Result struct {
	Terminal   bool
	Placed     bool
	OrderID    string
	FilledLots int64
	FillPrice  float64
}

// Executor builds and places market orders, or dry-runs when tradeEnabled is false.
type Executor struct {
	client       OrdersClient
	accountID    string
	tradeEnabled bool
}

// New returns an Executor bound to an account.
func New(c OrdersClient, accountID string, tradeEnabled bool) *Executor {
	return &Executor{client: c, accountID: accountID, tradeEnabled: tradeEnabled}
}

// OrderState retrieves execution progress of a broker-created order.
func (e *Executor) OrderState(ctx context.Context, orderID string) (*investapi.OrderState, error) {
	if !e.tradeEnabled {
		return nil, fmt.Errorf("order state unavailable in dry run")
	}
	state, err := e.client.GetOrderState(ctx, &investapi.GetOrderStateRequest{AccountId: e.accountID, OrderId: orderID})
	if err != nil {
		return nil, fmt.Errorf("get order state: %w", err)
	}
	return state, nil
}

// Buy places a market BUY of `lots` lots (or dry-runs).
func (e *Executor) Buy(ctx context.Context, instrumentID string, lots int64) (Result, error) {
	return e.BuyWithKey(ctx, instrumentID, lots, uuid.NewString())
}

// Sell places a market SELL of `lots` lots (or dry-runs).
func (e *Executor) Sell(ctx context.Context, instrumentID string, lots int64) (Result, error) {
	return e.SellWithKey(ctx, instrumentID, lots, uuid.NewString())
}

func (e *Executor) RequestState(ctx context.Context, key string) (*investapi.OrderState, error) {
	kind := investapi.OrderIdType_ORDER_ID_TYPE_REQUEST
	return e.client.GetOrderState(ctx, &investapi.GetOrderStateRequest{AccountId: e.accountID, OrderId: key, OrderIdType: &kind})
}
func (e *Executor) BuyWithKey(ctx context.Context, id string, lots int64, key string) (Result, error) {
	return e.place(ctx, id, lots, investapi.OrderDirection_ORDER_DIRECTION_BUY, key)
}
func (e *Executor) SellWithKey(ctx context.Context, id string, lots int64, key string) (Result, error) {
	return e.place(ctx, id, lots, investapi.OrderDirection_ORDER_DIRECTION_SELL, key)
}
func (e *Executor) place(ctx context.Context, instrumentID string, lots int64, dir investapi.OrderDirection, key string) (Result, error) {
	if !e.tradeEnabled {
		return Result{Placed: false}, nil
	}
	req := &investapi.PostOrderRequest{
		InstrumentId: instrumentID,
		Quantity:     lots,
		Direction:    dir,
		AccountId:    e.accountID,
		OrderType:    investapi.OrderType_ORDER_TYPE_MARKET,
		OrderId:      key,
	}
	var header, trailer metadata.MD
	resp, err := e.client.PostOrder(ctx, req, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		return Result{}, withBrokerDetails(err, header, trailer)
	}
	status := resp.GetExecutionReportStatus()
	terminal := status == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL || status == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED || status == investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED
	res := Result{Terminal: terminal, Placed: true, OrderID: resp.GetOrderId(), FilledLots: resp.GetLotsExecuted()}
	if p := resp.GetExecutedOrderPrice(); p != nil {
		res.FillPrice = utils.CombinePrice(p.GetUnits(), p.GetNano())
	}
	return res, nil
}

// withBrokerDetails appends the broker's reject reason to err. The status description
// holds only the numeric code (e.g. "30049" = "Post order error: %s"); the text that
// fills %s arrives in the "message" metadata key, next to "x-tracking-id" for support.
// The original error stays wrapped, so status.Code still sees it.
func withBrokerDetails(err error, header, trailer metadata.MD) error {
	get := func(key string) string {
		if v := header.Get(key); len(v) > 0 {
			return strings.Join(v, "; ")
		}
		return strings.Join(trailer.Get(key), "; ")
	}
	msg, trackingID := get("message"), get("x-tracking-id")
	if msg == "" && trackingID == "" {
		return err
	}
	return &brokerError{err: err, message: msg, trackingID: trackingID}
}

type brokerError struct {
	err        error
	message    string
	trackingID string
}

func (e *brokerError) Error() string {
	return fmt.Sprintf("%v (message: %q, tracking-id: %s)", e.err, e.message, e.trackingID)
}

func (e *brokerError) Unwrap() error { return e.err }

// IsFrozenPriceRejection identifies a rejected order by its specific broker reason.
// The generic code 30049 alone does not establish what happened.
func IsFrozenPriceRejection(err error) bool {
	var broker *brokerError
	if !errors.As(err, &broker) {
		return false
	}
	rpc := status.Convert(broker.err)
	return rpc.Code() == codes.InvalidArgument && rpc.Message() == "30049" &&
		broker.message == "Post order error: Zamorozhennaya czena ne sootvetstvuet tipu zayavki"
}

func (e *Executor) Cancel(ctx context.Context, id string) error {
	_, err := e.client.CancelOrder(ctx, &investapi.CancelOrderRequest{AccountId: e.accountID, OrderId: id})
	return err
}

func (e *Executor) ActiveOrders(ctx context.Context) ([]*investapi.OrderState, error) {
	resp, err := e.client.GetOrders(ctx, &investapi.GetOrdersRequest{AccountId: e.accountID})
	if err != nil {
		return nil, err
	}
	return resp.GetOrders(), nil
}
