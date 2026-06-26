// Package executor places (or dry-runs) whole-position market orders for the
// reversion runner. Each order carries a fresh UUID order_id for idempotency.
package executor

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/utils"
)

// OrdersClient is the slice of the orders client the executor needs.
type OrdersClient interface {
	PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error)
}

// Result reports the effect of an order attempt. When Placed is false (dry-run) the
// caller falls back to the signal's price and the requested quantity.
type Result struct {
	Placed     bool
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

// Buy places a market BUY of `lots` lots (or dry-runs).
func (e *Executor) Buy(ctx context.Context, instrumentID string, lots int64) (Result, error) {
	return e.place(ctx, instrumentID, lots, investapi.OrderDirection_ORDER_DIRECTION_BUY)
}

// Sell places a market SELL of `lots` lots (or dry-runs).
func (e *Executor) Sell(ctx context.Context, instrumentID string, lots int64) (Result, error) {
	return e.place(ctx, instrumentID, lots, investapi.OrderDirection_ORDER_DIRECTION_SELL)
}

func (e *Executor) place(ctx context.Context, instrumentID string, lots int64, dir investapi.OrderDirection) (Result, error) {
	if !e.tradeEnabled {
		return Result{Placed: false}, nil
	}
	req := &investapi.PostOrderRequest{
		InstrumentId: instrumentID,
		Quantity:     lots,
		Direction:    dir,
		AccountId:    e.accountID,
		OrderType:    investapi.OrderType_ORDER_TYPE_MARKET,
		OrderId:      uuid.NewString(),
	}
	resp, err := e.client.PostOrder(ctx, req)
	if err != nil {
		return Result{}, err
	}
	res := Result{Placed: true, FilledLots: resp.GetLotsExecuted()}
	if p := resp.GetExecutedOrderPrice(); p != nil {
		res.FillPrice = utils.CombinePrice(p.GetUnits(), p.GetNano())
	}
	return res, nil
}
