package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/executor/mocks"
)

func TestBuy_PlacesMarketOrder(t *testing.T) {
	var last *investapi.PostOrderRequest
	m := mocks.NewMockOrdersClient(t)
	m.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
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
	m.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
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

// setMD fills the header/trailer call options the way grpc-go does after a call.
func setMD(opts []grpc.CallOption, header, trailer metadata.MD) {
	for _, o := range opts {
		switch v := o.(type) {
		case grpc.HeaderCallOption:
			*v.HeaderAddr = header
		case grpc.TrailerCallOption:
			*v.TrailerAddr = trailer
		}
	}
}

func TestBuy_RejectCarriesBrokerMessage(t *testing.T) {
	tests := []struct {
		name    string
		header  metadata.MD
		trailer metadata.MD
	}{
		{"in header", metadata.Pairs("message", "Instrument is not available for trading", "x-tracking-id", "trk-42"), nil},
		{"in trailer", nil, metadata.Pairs("message", "Instrument is not available for trading", "x-tracking-id", "trk-42")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcErr := status.Error(codes.InvalidArgument, "30049")
			m := mocks.NewMockOrdersClient(t)
			m.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Run(func(_ context.Context, _ *investapi.PostOrderRequest, opts ...grpc.CallOption) {
					setMD(opts, tt.header, tt.trailer)
				}).
				Return(nil, rpcErr)
			e := New(m, "acc-1", true)

			_, err := e.Buy(context.Background(), "uid-1", 1)
			if err == nil {
				t.Fatal("Buy: want error")
			}
			msg := err.Error()
			if !strings.Contains(msg, "30049") ||
				!strings.Contains(msg, "Instrument is not available for trading") ||
				!strings.Contains(msg, "trk-42") {
				t.Fatalf("error %q must carry code, broker message and tracking id", msg)
			}
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("status code lost: %v", status.Code(err))
			}
		})
	}
}

func TestBuy_RejectWithoutMetadataKeepsError(t *testing.T) {
	rpcErr := status.Error(codes.InvalidArgument, "30049")
	m := mocks.NewMockOrdersClient(t)
	m.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, rpcErr)
	e := New(m, "acc-1", true)

	_, err := e.Buy(context.Background(), "uid-1", 1)
	if err == nil || err.Error() != rpcErr.Error() {
		t.Fatalf("err = %v, want unchanged %v", err, rpcErr)
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
