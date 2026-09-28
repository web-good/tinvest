// Package executor places (or dry-runs) whole-position market orders for the live
// strategy runners. Each order carries a fresh UUID order_id for idempotency.
package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

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
	var header, trailer metadata.MD
	resp, err := e.client.PostOrder(ctx, req, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		return Result{}, withBrokerDetails(err, header, trailer)
	}
	res := Result{Placed: true, FilledLots: resp.GetLotsExecuted()}
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
	return fmt.Errorf("%w (message: %q, tracking-id: %s)", err, msg, trackingID)
}
