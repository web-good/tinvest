package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/reversion/live/executor/mocks"
)

func TestBuy_PlacesMarketOrder(t *testing.T) {
	var last *investapi.PostOrderRequest
	m := mocks.NewMockOrdersClient(t)
	m.EXPECT().PostOrder(mock.Anything, mock.Anything).
		Run(func(_ context.Context, in *investapi.PostOrderRequest, _ ...grpc.CallOption) {
			last = in
		}).
		Return(&investapi.PostOrderResponse{
			LotsExecuted:       7,
			ExecutedOrderPrice: &investapi.MoneyValue{Units: 101, Nano: 500000000},
		}, nil)
	e := New(m, "acc-1", true)

	res, err := e.Buy(context.Background(), "uid-1", 7)
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if !res.Placed || res.FilledLots != 7 || res.FillPrice != 101.5 {
		t.Fatalf("Result = %+v, want Placed lots=7 price=101.5", res)
	}
	if last.Direction != investapi.OrderDirection_ORDER_DIRECTION_BUY {
		t.Fatalf("direction = %v, want BUY", last.Direction)
	}
	if last.OrderType != investapi.OrderType_ORDER_TYPE_MARKET {
		t.Fatalf("order type = %v, want MARKET", last.OrderType)
	}
	if last.Quantity != 7 || last.InstrumentId != "uid-1" || last.AccountId != "acc-1" {
		t.Fatalf("request fields wrong: %+v", last)
	}
	if len(last.OrderId) == 0 || len(last.OrderId) > 36 {
		t.Fatalf("OrderId must be a non-empty UID <=36 chars, got %q", last.OrderId)
	}
}

func TestSell_Direction(t *testing.T) {
	var last *investapi.PostOrderRequest
	m := mocks.NewMockOrdersClient(t)
	m.EXPECT().PostOrder(mock.Anything, mock.Anything).
		Run(func(_ context.Context, in *investapi.PostOrderRequest, _ ...grpc.CallOption) {
			last = in
		}).
		Return(&investapi.PostOrderResponse{LotsExecuted: 3}, nil)
	e := New(m, "acc-1", true)
	if _, err := e.Sell(context.Background(), "uid-1", 3); err != nil {
		t.Fatalf("Sell: %v", err)
	}
	if last.Direction != investapi.OrderDirection_ORDER_DIRECTION_SELL {
		t.Fatalf("direction = %v, want SELL", last.Direction)
	}
}

func TestBuy_DryRunPlacesNoOrder(t *testing.T) {
	m := mocks.NewMockOrdersClient(t)
	e := New(m, "acc-1", false) // trade disabled
	res, err := e.Buy(context.Background(), "uid-1", 5)
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if res.Placed {
		t.Fatal("dry-run must not place an order")
	}
	// m has no expectations set; NewMockOrdersClient(t) asserts on cleanup that
	// PostOrder was never called, preserving "must not be called in dry-run".
}
