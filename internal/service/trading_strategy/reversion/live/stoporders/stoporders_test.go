package stoporders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/reversion/live/stoporders/mocks"
	"tinvest/internal/utils"
)

func TestPlaceBuildsStopMarketSellRoundedDown(t *testing.T) {
	c := mocks.NewMockClient(t)
	var got *investapi.PostStopOrderRequest
	c.EXPECT().PostStopOrder(mock.Anything, mock.Anything).
		Run(func(_ context.Context, in *investapi.PostStopOrderRequest, _ ...grpc.CallOption) { got = in }).
		Return(&investapi.PostStopOrderResponse{StopOrderId: "so-1"}, nil)

	e := New(c, "acc", true)
	res, err := e.Place(context.Background(), "uid-1", 3, 107.037, 0.05) // 107.037 -> 107.00
	if err != nil || !res.Placed || res.OrderID != "so-1" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if got.GetInstrumentId() != "uid-1" || got.GetQuantity() != 3 || got.GetAccountId() != "acc" {
		t.Fatalf("req fields: %+v", got)
	}
	if got.GetDirection() != investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL ||
		got.GetStopOrderType() != investapi.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS ||
		got.GetExchangeOrderType() != investapi.ExchangeOrderType_EXCHANGE_ORDER_TYPE_MARKET ||
		got.GetExpirationType() != investapi.StopOrderExpirationType_STOP_ORDER_EXPIRATION_TYPE_GOOD_TILL_CANCEL {
		t.Fatalf("order type fields: %+v", got)
	}
	if p := utils.CombinePrice(got.GetStopPrice().GetUnits(), got.GetStopPrice().GetNano()); p != 107.00 {
		t.Fatalf("stop price = %v, want 107.00 (rounded down to 0.05)", p)
	}
	if got.GetOrderId() == "" {
		t.Fatal("idempotency order_id must be set")
	}
}

func TestDryRunTouchesNoAPI(t *testing.T) {
	c := mocks.NewMockClient(t) // без EXPECT: любой вызов = провал теста
	e := New(c, "acc", false)
	res, err := e.Place(context.Background(), "uid-1", 1, 100, 0.01)
	if err != nil || res.Placed {
		t.Fatalf("dry-run Place: res=%+v err=%v", res, err)
	}
	if err := e.Cancel(context.Background(), "so-1"); err != nil {
		t.Fatalf("dry-run Cancel: %v", err)
	}
	if list, err := e.List(context.Background()); err != nil || list != nil {
		t.Fatalf("dry-run List: %v %v", list, err)
	}
	if fired, err := e.Executed(context.Background(), "so-1"); err != nil || fired {
		t.Fatalf("dry-run Executed: %v %v", fired, err)
	}
}

// Executed отличает сработавшую заявку от отменённой: ID ищется в списке
// EXECUTED-заявок счёта (fired-check для исчезнувшего из ACTIVE стопа).
func TestExecutedLooksUpFiredStop(t *testing.T) {
	c := mocks.NewMockClient(t)
	c.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(func(in *investapi.GetStopOrdersRequest) bool {
		return in.GetAccountId() == "acc" && in.GetStatus() == investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED
	})).Return(&investapi.GetStopOrdersResponse{StopOrders: []*investapi.StopOrder{
		{StopOrderId: "so-1"},
	}}, nil).Twice()

	e := New(c, "acc", true)
	fired, err := e.Executed(context.Background(), "so-1")
	if err != nil || !fired {
		t.Fatalf("fired=%v err=%v, want fired for listed id", fired, err)
	}
	fired, err = e.Executed(context.Background(), "so-2")
	if err != nil || fired {
		t.Fatalf("fired=%v err=%v, want not-fired for missing id", fired, err)
	}
}

func TestListReturnsOnlyActiveSellStops(t *testing.T) {
	c := mocks.NewMockClient(t)
	c.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(func(in *investapi.GetStopOrdersRequest) bool {
		return in.GetAccountId() == "acc" && in.GetStatus() == investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE
	})).Return(&investapi.GetStopOrdersResponse{StopOrders: []*investapi.StopOrder{
		{StopOrderId: "so-1", InstrumentUid: "uid-1",
			Direction:     investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL,
			StopPrice:     &investapi.MoneyValue{Units: 107},
			LotsRequested: 6},
		{StopOrderId: "so-2", InstrumentUid: "uid-2",
			Direction: investapi.StopOrderDirection_STOP_ORDER_DIRECTION_BUY,
			StopPrice: &investapi.MoneyValue{Units: 50}},
	}}, nil)

	e := New(c, "acc", true)
	list, err := e.List(context.Background())
	if err != nil || len(list) != 1 || list[0].StopOrderID != "so-1" || list[0].StopPrice != 107 || list[0].Lots != 6 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}
