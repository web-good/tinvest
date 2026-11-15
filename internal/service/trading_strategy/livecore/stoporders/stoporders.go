// Package stoporders places (or dry-runs) the single protective stop-market SELL
// order a live runner keeps on the exchange per open position. Each order carries a
// fresh UUID order_id for idempotency.
package stoporders

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/utils"
)

// Client is the stop-orders client surface the executor needs — today the whole
// generated service (all three RPCs), so it embeds it instead of restating the methods.
type Client interface {
	investapi.StopOrdersServiceClient
}

// ActiveStop describes one active SELL stop-order returned by List.
type ActiveStop struct {
	Type          investapi.StopOrderType
	InstrumentUID string
	StopOrderID   string
	StopPrice     float64
	Lots          int64
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

// RoundDownToIncrement snaps price DOWN to the instrument's min price increment —
// conservative for a sell stop (never above the strategy level). incr<=0 keeps price.
// Exported so the runner can compare desired levels at exchange granularity and skip
// cancel+repost cycles that would land on the same rounded price.
func RoundDownToIncrement(price, incr float64) float64 {
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
	rounded := RoundDownToIncrement(stopPrice, minPriceIncrement)
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

// PlaceTakeProfit posts a market sell triggered at the target, rounded up to a valid tick.
func (e *Executor) PlaceTakeProfit(ctx context.Context, instrumentID string, lots int64, target, increment float64) (Result, error) {
	if !e.tradeEnabled {
		return Result{}, nil
	}
	if increment > 0 {
		target = math.Ceil(target/increment-1e-9) * increment
	}
	units, nano := utils.SplitPrice(target)
	resp, err := e.client.PostStopOrder(ctx, &investapi.PostStopOrderRequest{
		InstrumentId: instrumentID, Quantity: lots, StopPrice: &investapi.Quotation{Units: units, Nano: nano},
		Direction: investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL, AccountId: e.accountID,
		ExpirationType:    investapi.StopOrderExpirationType_STOP_ORDER_EXPIRATION_TYPE_GOOD_TILL_CANCEL,
		StopOrderType:     investapi.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT,
		ExchangeOrderType: investapi.ExchangeOrderType_EXCHANGE_ORDER_TYPE_MARKET, OrderId: uuid.NewString(),
	})
	if err != nil {
		return Result{}, fmt.Errorf("post take profit: %w", err)
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

// Executed reports whether stopOrderID is among the account's EXECUTED stop orders —
// the fired-check for a stop that vanished from the ACTIVE list (fired vs cancelled
// externally). Dry-run always reports false.
func (e *Executor) Executed(ctx context.Context, stopOrderID string) (bool, error) {
	_, found, err := e.Triggered(ctx, stopOrderID)
	return found, err
}

// Triggered returns the exchange order ID produced by a triggered stop. A found
// stop with an empty exchange ID must be treated as unresolved, not as a fill.
func (e *Executor) Triggered(ctx context.Context, stopOrderID string) (string, bool, error) {
	if !e.tradeEnabled {
		return "", false, nil
	}
	resp, err := e.client.GetStopOrders(ctx, &investapi.GetStopOrdersRequest{
		AccountId: e.accountID, Status: investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED,
	})
	if err != nil {
		return "", false, fmt.Errorf("get executed stop orders: %w", err)
	}
	for _, so := range resp.GetStopOrders() {
		if so.GetStopOrderId() == stopOrderID {
			return so.GetExchangeOrderId(), true, nil
		}
	}
	return "", false, nil
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
			Type:          so.GetOrderType(),
			StopOrderID:   so.GetStopOrderId(),
			StopPrice:     utils.CombinePrice(so.GetStopPrice().GetUnits(), so.GetStopPrice().GetNano()),
			Lots:          so.GetLotsRequested(),
		})
	}
	return out, nil
}

// TriggeredSince returns executed sell stops for an instrument since its entry.
func (e *Executor) TriggeredSince(ctx context.Context, instrumentID string, since time.Time) ([]*investapi.StopOrder, error) {
	if !e.tradeEnabled {
		return nil, nil
	}
	resp, err := e.client.GetStopOrders(ctx, &investapi.GetStopOrdersRequest{AccountId: e.accountID, Status: investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED})
	if err != nil {
		return nil, fmt.Errorf("get executed stop orders: %w", err)
	}
	var out []*investapi.StopOrder
	for _, order := range resp.GetStopOrders() {
		if order.GetInstrumentUid() != instrumentID || order.GetDirection() != investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL {
			continue
		}
		if order.GetActivationDateTime() == nil {
			return nil, fmt.Errorf("executed stop %s has no activation time", order.GetStopOrderId())
		}
		if !order.GetActivationDateTime().AsTime().Before(since) {
			out = append(out, order)
		}
	}
	return out, nil
}
