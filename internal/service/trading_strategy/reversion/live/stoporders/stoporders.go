// Package stoporders places (or dry-runs) the single protective stop-market SELL
// order the reversion runner keeps on the exchange per open position. Each order
// carries a fresh UUID order_id for idempotency.
package stoporders

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/utils"
)

// Client is the slice of the stop-orders client the executor needs.
type Client interface {
	PostStopOrder(ctx context.Context, in *investapi.PostStopOrderRequest, opts ...grpc.CallOption) (*investapi.PostStopOrderResponse, error)
	GetStopOrders(ctx context.Context, in *investapi.GetStopOrdersRequest, opts ...grpc.CallOption) (*investapi.GetStopOrdersResponse, error)
	CancelStopOrder(ctx context.Context, in *investapi.CancelStopOrderRequest, opts ...grpc.CallOption) (*investapi.CancelStopOrderResponse, error)
}

// ActiveStop describes one active SELL stop-order returned by List.
type ActiveStop struct {
	InstrumentUID string
	StopOrderID   string
	StopPrice     float64
}

// Result reports the effect of a Place attempt. When Placed is false (dry-run) the
// caller keeps relying on its own state rather than an exchange-side stop.
type Result struct {
	Placed  bool
	OrderID string
}

// Executor places, cancels and lists protective stop-market SELL orders, or
// dry-runs when tradeEnabled is false.
type Executor struct {
	client       Client
	accountID    string
	tradeEnabled bool
}

// New returns an Executor bound to an account.
func New(c Client, accountID string, tradeEnabled bool) *Executor {
	return &Executor{client: c, accountID: accountID, tradeEnabled: tradeEnabled}
}

// roundDownToIncrement snaps price DOWN to the instrument's min price increment —
// conservative for a sell stop (never above the strategy level). incr<=0 keeps price.
func roundDownToIncrement(price, incr float64) float64 {
	if incr <= 0 {
		return price
	}
	return math.Floor(price/incr+1e-9) * incr
}

// Place builds and posts a stop-market SELL order for lots at stopPrice (rounded down
// to minPriceIncrement), or dry-runs when tradeEnabled is false.
func (e *Executor) Place(ctx context.Context, instrumentID string, lots int64, stopPrice, minPriceIncrement float64) (Result, error) {
	if !e.tradeEnabled {
		return Result{Placed: false}, nil
	}
	rounded := roundDownToIncrement(stopPrice, minPriceIncrement)
	units, nano := utils.SplitPrice(rounded)
	req := &investapi.PostStopOrderRequest{
		InstrumentId:      instrumentID,
		Quantity:          lots,
		StopPrice:         &investapi.Quotation{Units: units, Nano: nano},
		Direction:         investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL,
		AccountId:         e.accountID,
		ExpirationType:    investapi.StopOrderExpirationType_STOP_ORDER_EXPIRATION_TYPE_GOOD_TILL_CANCEL,
		StopOrderType:     investapi.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS,
		ExchangeOrderType: investapi.ExchangeOrderType_EXCHANGE_ORDER_TYPE_MARKET,
		OrderId:           uuid.NewString(),
	}
	resp, err := e.client.PostStopOrder(ctx, req)
	if err != nil {
		return Result{}, fmt.Errorf("post stop order: %w", err)
	}
	return Result{Placed: true, OrderID: resp.GetStopOrderId()}, nil
}

// Cancel cancels a previously placed stop order, or no-ops when tradeEnabled is false.
func (e *Executor) Cancel(ctx context.Context, stopOrderID string) error {
	if !e.tradeEnabled {
		return nil
	}
	_, err := e.client.CancelStopOrder(ctx, &investapi.CancelStopOrderRequest{
		AccountId: e.accountID, StopOrderId: stopOrderID,
	})
	if err != nil {
		return fmt.Errorf("cancel stop order %s: %w", stopOrderID, err)
	}
	return nil
}

// List returns the account's active SELL stop-orders, or (nil, nil) when
// tradeEnabled is false.
func (e *Executor) List(ctx context.Context) ([]ActiveStop, error) {
	if !e.tradeEnabled {
		return nil, nil
	}
	resp, err := e.client.GetStopOrders(ctx, &investapi.GetStopOrdersRequest{
		AccountId: e.accountID,
		Status:    investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE,
	})
	if err != nil {
		return nil, fmt.Errorf("get stop orders: %w", err)
	}
	var out []ActiveStop
	for _, so := range resp.GetStopOrders() {
		if so.GetDirection() != investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL {
			continue
		}
		out = append(out, ActiveStop{
			InstrumentUID: so.GetInstrumentUid(),
			StopOrderID:   so.GetStopOrderId(),
			StopPrice:     utils.CombinePrice(so.GetStopPrice().GetUnits(), so.GetStopPrice().GetNano()),
		})
	}
	return out, nil
}
