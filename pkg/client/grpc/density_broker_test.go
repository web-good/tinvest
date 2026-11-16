package grpc

import (
	"context"
	"errors"
	"testing"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/live"

	"github.com/google/uuid"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type liveUsersFake struct {
	investapi.UsersServiceClient
	marginErr error
	access    investapi.AccessLevel
}

func (f *liveUsersFake) GetAccounts(_ context.Context, _ *investapi.GetAccountsRequest, _ ...googlegrpc.CallOption) (*investapi.GetAccountsResponse, error) {
	return &investapi.GetAccountsResponse{Accounts: []*investapi.Account{{Id: "selected", Status: investapi.AccountStatus_ACCOUNT_STATUS_OPEN, AccessLevel: f.access}}}, nil
}
func (f *liveUsersFake) GetMarginAttributes(_ context.Context, _ *investapi.GetMarginAttributesRequest, _ ...googlegrpc.CallOption) (*investapi.GetMarginAttributesResponse, error) {
	return &investapi.GetMarginAttributesResponse{}, f.marginErr
}

type liveOrdersFake struct {
	investapi.OrdersServiceClient
	post *investapi.PostOrderRequest
	get  *investapi.GetOrderStateRequest
	opts []googlegrpc.CallOption
}

func (f *liveOrdersFake) PostOrder(ctx context.Context, r *investapi.PostOrderRequest, opts ...googlegrpc.CallOption) (*investapi.PostOrderResponse, error) {
	if _, ok := ctx.Deadline(); !ok {
		panic("missing deadline")
	}
	f.post = r
	f.opts = opts
	return &investapi.PostOrderResponse{OrderId: "exchange", OrderRequestId: r.OrderId, InstrumentUid: r.InstrumentId, Direction: r.Direction, LotsRequested: r.Quantity, ExecutionReportStatus: investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_NEW}, nil
}
func (f *liveOrdersFake) GetOrderState(_ context.Context, r *investapi.GetOrderStateRequest, _ ...googlegrpc.CallOption) (*investapi.OrderState, error) {
	f.get = r
	return &investapi.OrderState{OrderId: "exchange", InstrumentUid: "uid", Direction: investapi.OrderDirection_ORDER_DIRECTION_BUY, LotsRequested: 1, LotsExecuted: 1, ExecutionReportStatus: investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL, ExecutedCommission: &investapi.MoneyValue{Currency: "rub", Nano: 500000000}, Stages: []*investapi.OrderStage{{Quantity: 1, Price: &investapi.MoneyValue{Currency: "rub", Units: 100}}}}, nil
}

type liveStopsFake struct {
	investapi.StopOrdersServiceClient
	requests []*investapi.PostStopOrderRequest
}

func (f *liveStopsFake) PostStopOrder(_ context.Context, r *investapi.PostStopOrderRequest, _ ...googlegrpc.CallOption) (*investapi.PostStopOrderResponse, error) {
	f.requests = append(f.requests, r)
	return &investapi.PostStopOrderResponse{StopOrderId: "stop-id"}, nil
}

func brokerFixture() *DensityBroker {
	return NewDensityBroker(nil, "not-a-real-token", "selected", true, []density.Instrument{{ID: "uid", Lot: 10, PriceStep: density.Price{Nano: 10000000}}})
}
func TestDensityBrokerOnlyExactMarginDisabledErrorPasses(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		ok   bool
	}{{"disabled", status.Error(codes.InvalidArgument, "30051"), true}, {"enabled", nil, false}, {"network", status.Error(codes.Unavailable, "30051"), false}, {"other", status.Error(codes.InvalidArgument, "30052"), false}, {"text", errors.New("margin disabled"), false}} {
		t.Run(tc.name, func(t *testing.T) {
			b := brokerFixture()
			b.users = &liveUsersFake{marginErr: tc.err, access: investapi.AccessLevel_ACCOUNT_ACCESS_LEVEL_FULL_ACCESS}
			if err := b.CheckAccount(context.Background()); (err == nil) != tc.ok {
				t.Fatalf("err=%v want ok=%v", err, tc.ok)
			}
		})
	}
}
func TestDensityBrokerReadOnlyModeBlocksEveryMutation(t *testing.T) {
	b := brokerFixture()
	b.trade = false
	ctx := context.Background()
	if _, err := b.PostOrder(ctx, live.OrderRequest{}); err == nil {
		t.Fatal("post enabled")
	}
	if _, err := b.PostStop(ctx, live.StopRequest{}); err == nil {
		t.Fatal("stop enabled")
	}
	if err := b.CancelOrder(ctx, "id", false); err == nil {
		t.Fatal("cancel enabled")
	}
	if err := b.CancelStop(ctx, "id"); err == nil {
		t.Fatal("stop cancel enabled")
	}
}
func TestDensityBrokerOneLotAndRealStopTakeRequests(t *testing.T) {
	b := brokerFixture()
	o := &liveOrdersFake{}
	s := &liveStopsFake{}
	b.orders = o
	b.stops = s
	ctx := context.Background()
	key := uuid.NewString()
	if _, err := b.PostOrder(ctx, live.OrderRequest{Key: key, InstrumentID: "uid", Kind: "buy", Lots: 1, Price: 100010000000}); err != nil {
		t.Fatal(err)
	}
	if o.post.Quantity != 1 || o.post.OrderId != key || o.post.AccountId != "selected" || o.post.Price.Nano != 10000000 || len(o.opts) == 0 {
		t.Fatal(o.post)
	}
	for _, kind := range []string{"stop", "take"} {
		if _, err := b.PostStop(ctx, live.StopRequest{Key: uuid.NewString(), InstrumentID: "uid", Kind: kind, Lots: 1, Price: 100e9}); err != nil {
			t.Fatal(err)
		}
	}
	if s.requests[1].TakeProfitType != investapi.TakeProfitType_TAKE_PROFIT_TYPE_REGULAR {
		t.Fatal("take must explicitly use regular, not trailing or unspecified, subtype")
	}
	if s.requests[0].StopOrderType != investapi.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS || s.requests[0].ExchangeOrderType != investapi.ExchangeOrderType_EXCHANGE_ORDER_TYPE_MARKET || s.requests[1].StopOrderType != investapi.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT || s.requests[1].ExchangeOrderType != investapi.ExchangeOrderType_EXCHANGE_ORDER_TYPE_LIMIT || s.requests[1].GetPrice().Units != 100 {
		t.Fatal(s.requests)
	}
	if _, err := b.PostOrder(ctx, live.OrderRequest{Key: uuid.NewString(), InstrumentID: "uid", Kind: "buy", Lots: 2, Price: 100e9}); err == nil {
		t.Fatal("more than one lot allowed")
	}
}
func TestDensityBrokerOrderLookupByUUIDAndLotAccounting(t *testing.T) {
	b := brokerFixture()
	o := &liveOrdersFake{}
	b.orders = o
	got, err := b.Order(context.Background(), uuid.NewString(), true)
	if err != nil {
		t.Fatal(err)
	}
	if o.get.GetOrderIdType() != investapi.OrderIdType_ORDER_ID_TYPE_REQUEST || got.Amount != 1000e9 || got.Fees != 500000000 || !got.Terminal || got.Filled != 1 {
		t.Fatalf("got=%+v req=%+v", got, o.get)
	}
}
func TestDensityBrokerTimeoutIsNotDefiniteRejection(t *testing.T) {
	if errors.Is(brokerError(status.Error(codes.DeadlineExceeded, "accepted but response lost")), live.ErrRejected) {
		t.Fatal("timeout classified rejected")
	}
	if errors.Is(brokerError(status.Error(codes.InvalidArgument, "30058")), live.ErrRejected) {
		t.Fatal("processing request classified rejected")
	}
}

func TestConfirmedFillSurvivesMalformedAccounting(t *testing.T) {
	o, err := normalizeOrder("filled", "key", "uid", investapi.OrderDirection_ORDER_DIRECTION_BUY, investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL, 1, 1, &investapi.MoneyValue{Currency: "usd", Units: 1})
	if err != nil || o.Filled != 1 || !o.Terminal || o.FeesKnown {
		t.Fatalf("accounting prevented protecting confirmed inventory: %+v %v", o, err)
	}
}
