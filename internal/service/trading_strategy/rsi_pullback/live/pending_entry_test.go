package live

import (
	"context"
	"errors"
	"testing"
	imodel "tinvest/internal/model"
	pb "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

func TestPendingEntryProtectsFillBeforeUncertainBuyCancellation(t *testing.T) {
	guest := zoneFake(0, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	pc := &passCtx{shared: true, state: map[string]statestore.Entry{"GAZP": {Ticker: "GAZP", Strategy: "rsi_zone", EntryRequestID: "buy-key", EntryOrderID: "buy", Quantity: 3, EntryATR: 10, EntryPrice: 100, MaxFav: 100}}, store: e.svc.stateStore(), stopByID: map[string]stoporders.ActiveStop{}}
	e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{OrderId: "buy", InstrumentUid: "uid-GAZP", Direction: pb.OrderDirection_ORDER_DIRECTION_BUY, LotsRequested: 10, LotsExecuted: 3, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL}, nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).Return(&pb.PostStopOrderResponse{StopOrderId: "protect"}, nil).Once()
	e.orders.EXPECT().CancelOrder(mock.Anything, mock.Anything).Run(func(context.Context, *pb.CancelOrderRequest, ...grpc.CallOption) {
		if pc.state["GAZP"].StopOrderID != "protect" {
			t.Error("BUY cancelled before confirmed fill protected")
		}
	}).Return(nil, errors.New("cancel timeout")).Once()
	if _, err := e.svc.reconcilePendingEntry(context.Background(), pc, e.svc.slotByName("rsi_zone"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1, MinPriceIncrement: 0.01}); err == nil {
		t.Fatal("lost uncertain cancellation error")
	}
	if got := pc.state["GAZP"]; got.StopOrderID != "protect" || got.EntryRequestID != "buy-key" || got.Quantity != 3 {
		t.Fatalf("lost protected own partial: %+v", got)
	}
}

func TestPendingEntrySubtractsItsExecutedProtectionBeforeProtectingLaterFills(t *testing.T) {
	guest := zoneFake(0, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	pc := &passCtx{shared: true, state: map[string]statestore.Entry{"GAZP": {Ticker: "GAZP", Strategy: "rsi_zone", EntryRequestID: "buy-key", EntryOrderID: "buy", EntryFilledQuantity: 3, Quantity: 3, StopOrderID: "old-protect", EntryATR: 10, EntryPrice: 100, MaxFav: 100}}, store: e.svc.stateStore(), stopByID: map[string]stoporders.ActiveStop{}}
	e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{OrderId: "buy", InstrumentUid: "uid-GAZP", Direction: pb.OrderDirection_ORDER_DIRECTION_BUY, LotsRequested: 10, LotsExecuted: 5, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED}, nil).Once()
	child := "old-child"
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{StopOrders: []*pb.StopOrder{{StopOrderId: "old-protect", ExchangeOrderId: &child}}}, nil).Once()
	e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{OrderId: child, InstrumentUid: "uid-GAZP", Direction: pb.OrderDirection_ORDER_DIRECTION_SELL, LotsRequested: 3, LotsExecuted: 3, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL}, nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.PostStopOrderRequest) bool { return r.GetQuantity() == 2 })).Return(&pb.PostStopOrderResponse{StopOrderId: "new-protect"}, nil).Once()
	done, err := e.svc.reconcilePendingEntry(context.Background(), pc, e.svc.slotByName("rsi_zone"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1, MinPriceIncrement: 0.01})
	if err != nil || !done {
		t.Fatal(done, err)
	}
	if got := pc.state["GAZP"]; got.Quantity != 2 || got.EntrySoldQuantity != 3 || got.EntryRequestID != "" || got.StopOrderID != "new-protect" {
		t.Fatalf("resurrected sold shares: %+v", got)
	}
}

func TestPendingEntryAdditionalFillsAtCancellationAreProtectedImmediately(t *testing.T) {
	guest := zoneFake(0, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	pc := &passCtx{shared: true, state: map[string]statestore.Entry{"GAZP": {Ticker: "GAZP", Strategy: "rsi_zone", EntryRequestID: "key", EntryOrderID: "buy", Quantity: 3, EntryATR: 10, EntryPrice: 100, MaxFav: 100}}, store: e.svc.stateStore(), stopByID: map[string]stoporders.ActiveStop{}}
	partial := &pb.OrderState{OrderId: "buy", InstrumentUid: "uid-GAZP", Direction: pb.OrderDirection_ORDER_DIRECTION_BUY, LotsRequested: 10, LotsExecuted: 3, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL}
	final := &pb.OrderState{OrderId: "buy", InstrumentUid: "uid-GAZP", Direction: pb.OrderDirection_ORDER_DIRECTION_BUY, LotsRequested: 10, LotsExecuted: 5, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED}
	e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(partial, nil).Once()
	e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(final, nil).Times(3)
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.PostStopOrderRequest) bool { return r.GetQuantity() == 3 })).Return(&pb.PostStopOrderResponse{StopOrderId: "protect3"}, nil).Once()
	e.orders.EXPECT().CancelOrder(mock.Anything, mock.Anything).Return(&pb.CancelOrderResponse{}, nil).Once()
	e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).Return(&pb.CancelStopOrderResponse{}, nil).Once()
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{}, nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.PostStopOrderRequest) bool { return r.GetQuantity() == 5 })).Return(&pb.PostStopOrderResponse{StopOrderId: "protect5"}, nil).Once()
	done, err := e.svc.reconcilePendingEntry(context.Background(), pc, e.svc.slotByName("rsi_zone"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1, MinPriceIncrement: 0.01})
	if err != nil || !done {
		t.Fatal(done, err)
	}
	if got := pc.state["GAZP"]; got.Quantity != 5 || got.StopOrderID != "protect5" || got.EntryRequestID != "" {
		t.Fatalf("new fills left unprotected: %+v", got)
	}
}

func TestPendingEntryRejectsMismatchedBrokerOrder(t *testing.T) {
	for _, name := range []string{"instrument", "direction", "lots", "identity"} {
		t.Run(name, func(t *testing.T) {
			guest := zoneFake(0, "GAZP")
			guest.trade = true
			e := newSharedEnv(t, sharedNow, nil, nil, guest)
			pc := &passCtx{shared: true, state: map[string]statestore.Entry{"GAZP": {EntryRequestID: "key", EntryOrderID: "buy", EntryRequestedLots: 10, Quantity: 3}}, store: e.svc.stateStore()}
			order := &pb.OrderState{OrderId: "buy", InstrumentUid: "uid-GAZP", Direction: pb.OrderDirection_ORDER_DIRECTION_BUY, LotsRequested: 10, LotsExecuted: 3}
			switch name {
			case "instrument":
				order.InstrumentUid = "density-uid"
			case "direction":
				order.Direction = pb.OrderDirection_ORDER_DIRECTION_SELL
			case "lots":
				order.LotsExecuted = 11
			case "identity":
				order.OrderId = "other-order"
			}
			e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(order, nil).Once()
			if _, err := e.svc.reconcilePendingEntry(context.Background(), pc, e.svc.slotByName("rsi_zone"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1}); err == nil {
				t.Fatal("mismatched order adopted")
			}
			if pc.state["GAZP"].Quantity != 3 {
				t.Fatal("mismatch changed owned quantity")
			}
		})
	}
}
