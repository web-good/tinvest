package executor

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
)

type fakeOrders struct {
	last *investapi.PostOrderRequest
	resp *investapi.PostOrderResponse
}

func (f *fakeOrders) PostOrder(_ context.Context, in *investapi.PostOrderRequest, _ ...grpc.CallOption) (*investapi.PostOrderResponse, error) {
	f.last = in
	return f.resp, nil
}

func TestBuy_PlacesMarketOrder(t *testing.T) {
	f := &fakeOrders{resp: &investapi.PostOrderResponse{
		LotsExecuted:       7,
		ExecutedOrderPrice: &investapi.MoneyValue{Units: 101, Nano: 500000000},
	}}
	e := New(f, "acc-1", true)

	res, err := e.Buy(context.Background(), "uid-1", 7)
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if !res.Placed || res.FilledLots != 7 || res.FillPrice != 101.5 {
		t.Fatalf("Result = %+v, want Placed lots=7 price=101.5", res)
	}
	if f.last.Direction != investapi.OrderDirection_ORDER_DIRECTION_BUY {
		t.Fatalf("direction = %v, want BUY", f.last.Direction)
	}
	if f.last.OrderType != investapi.OrderType_ORDER_TYPE_MARKET {
		t.Fatalf("order type = %v, want MARKET", f.last.OrderType)
	}
	if f.last.Quantity != 7 || f.last.InstrumentId != "uid-1" || f.last.AccountId != "acc-1" {
		t.Fatalf("request fields wrong: %+v", f.last)
	}
	if len(f.last.OrderId) == 0 || len(f.last.OrderId) > 36 {
		t.Fatalf("OrderId must be a non-empty UID <=36 chars, got %q", f.last.OrderId)
	}
}

func TestSell_Direction(t *testing.T) {
	f := &fakeOrders{resp: &investapi.PostOrderResponse{LotsExecuted: 3}}
	e := New(f, "acc-1", true)
	if _, err := e.Sell(context.Background(), "uid-1", 3); err != nil {
		t.Fatalf("Sell: %v", err)
	}
	if f.last.Direction != investapi.OrderDirection_ORDER_DIRECTION_SELL {
		t.Fatalf("direction = %v, want SELL", f.last.Direction)
	}
}

func TestBuy_DryRunPlacesNoOrder(t *testing.T) {
	f := &fakeOrders{}
	e := New(f, "acc-1", false) // trade disabled
	res, err := e.Buy(context.Background(), "uid-1", 5)
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if res.Placed {
		t.Fatal("dry-run must not place an order")
	}
	if f.last != nil {
		t.Fatal("PostOrder must not be called in dry-run")
	}
}
