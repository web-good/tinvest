package live

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	imodel "tinvest/internal/model"
	pb "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	"tinvest/internal/service/trading_strategy/scalping/model"
)

func TestFVGTriggeredExitWaitsForPortfolioAndCancelsSibling(t *testing.T) {
	for _, tc := range []struct {
		name, fired, sibling string
		typ                  pb.StopOrderType
	}{
		{"target", "tp", "sl", pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS},
		{"stop", "sl", "tp", pb.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeStrategy{name: "fvg_imbalance", trade: true}
			env := newSharedEnv(t, sharedNow, nil, nil, f)
			entry := statestore.Entry{Ticker: "GAZP", EntryTime: sharedNow.Add(-time.Hour * 3), Quantity: 10, StopOrderID: "sl", TakeProfitOrderID: "tp"}
			pc := &passCtx{state: map[string]statestore.Entry{"GAZP": entry}, store: env.svc.stateStore(), now: sharedNow, stopByID: map[string]stoporders.ActiveStop{tc.sibling: {StopOrderID: tc.sibling, InstrumentUID: "uid-GAZP", Type: tc.typ, Lots: 10}}}
			child := "child"
			env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{StopOrders: []*pb.StopOrder{{StopOrderId: tc.fired, ExchangeOrderId: &child}}}, nil).Once()
			env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.CancelStopOrderRequest) bool { return r.GetStopOrderId() == tc.sibling })).Return(&pb.CancelStopOrderResponse{}, nil).Once()
			env.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL, LotsExecuted: 10}, nil).Twice()
			sh := &imodel.Share{ID: "uid-GAZP", Lot: 1}
			if err := env.svc.reconcileFVG(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", sh, held("GAZP", 10)); err != nil {
				t.Fatal(err)
			}
			if e := pc.state["GAZP"]; e.FVGExitOrderID != "child" || e.Quantity != 10 {
				t.Fatalf("state lost while portfolio stale: %+v", e)
			}
			pc.stopByID = map[string]stoporders.ActiveStop{}
			if err := env.svc.reconcileFVG(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", sh, nil); err != nil {
				t.Fatal(err)
			}
			if _, ok := pc.state["GAZP"]; ok {
				t.Fatal("confirmed exit still in state")
			}
		})
	}
}

func TestFVGPartialExecutionWaitsForTerminalAndUpdatedPortfolio(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": {Quantity: 10, FVGExitStopID: "sl", FVGExitOrderID: "child", StopOrderID: "sl"}}, store: env.svc.stateStore(), now: sharedNow, stopByID: map[string]stoporders.ActiveStop{}}
	env.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL, LotsExecuted: 4}, nil).Once()
	env.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED, LotsExecuted: 4}, nil).Twice()
	sh := &imodel.Share{ID: "uid-GAZP", Lot: 1}
	sl := env.svc.slotByName("fvg_imbalance")
	for _, qty := range []int64{10, 10, 6} {
		if err := env.svc.reconcileFVG(context.Background(), pc, sl, "GAZP", sh, held("GAZP", qty)); err != nil {
			t.Fatal(err)
		}
		e := pc.state["GAZP"]
		if qty == 10 && (e.Quantity != 10 || e.FVGExitOrderID == "") {
			t.Fatalf("unsafe premature release: %+v", e)
		}
		if qty == 6 && (e.Quantity != 6 || e.FVGExitOrderID != "") {
			t.Fatalf("residual not released: %+v", e)
		}
	}
}

func TestFVGCancelFailureKeepsState(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": {Quantity: 10, StopOrderID: "sl", TakeProfitOrderID: "tp", FVGExitStopID: "sl", FVGExitOrderID: "child"}}, store: env.svc.stateStore(), now: sharedNow, stopByID: map[string]stoporders.ActiveStop{"tp": {StopOrderID: "tp", InstrumentUID: "uid-GAZP", Type: pb.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT}}}
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).Return(nil, errors.New("unavailable")).Once()
	err := env.svc.reconcileFVG(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1}, held("GAZP", 10))
	if err == nil || pc.state["GAZP"].TakeProfitOrderID != "tp" {
		t.Fatalf("lost pending sibling: %v %+v", err, pc.state)
	}
}

func TestFVGStochasticExitCancelsBothOrdersBeforeSell(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	e := statestore.Entry{Quantity: 10, StopOrderID: "sl", TakeProfitOrderID: "tp"}
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": e}, store: env.svc.stateStore()}
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.CancelStopOrderRequest) bool { return r.GetStopOrderId() == "tp" })).Return(&pb.CancelStopOrderResponse{}, nil).Once()
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.CancelStopOrderRequest) bool { return r.GetStopOrderId() == "sl" })).Return(&pb.CancelStopOrderResponse{}, nil).Once()
	env.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ *pb.PostOrderRequest, _ ...grpc.CallOption) (*pb.PostOrderResponse, error) {
		if pc.state["GAZP"].StopOrderID != "" || pc.state["GAZP"].TakeProfitOrderID != "" || pc.state["GAZP"].FVGExitStopID != "market" {
			t.Fatal("orders or intent not persisted before sale")
		}
		return &pb.PostOrderResponse{OrderId: "market-child", LotsExecuted: 10}, nil
	}).Once()
	err := env.svc.sell(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1}, e, held("GAZP", 10), model.Signal{Kind: model.SignalSell, Reason: "STOCH_EXIT"})
	if err != nil || pc.state["GAZP"].FVGExitOrderID != "market-child" {
		t.Fatalf("exit state: %+v %v", pc.state, err)
	}
}

func TestFVGRecoveryFindsTriggeredOrderBeforeReplacing(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": {Quantity: 10, EntryTime: sharedNow.Add(-time.Hour), FVGRecovering: true}}, store: env.svc.stateStore(), now: sharedNow, stopByID: map[string]stoporders.ActiveStop{"sl": {StopOrderID: "sl", InstrumentUID: "uid-GAZP", Type: pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS}}}
	child := "child"
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{StopOrders: []*pb.StopOrder{{StopOrderId: "tp", InstrumentUid: "uid-GAZP", Direction: pb.StopOrderDirection_STOP_ORDER_DIRECTION_SELL, ActivationDateTime: timestamppb.New(sharedNow.Add(-time.Minute)), ExchangeOrderId: &child}}}, nil).Once()
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).Return(&pb.CancelStopOrderResponse{}, nil).Once()
	env.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL, LotsExecuted: 10}, nil).Once()
	err := env.svc.reconcileFVG(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1}, held("GAZP", 10))
	if err != nil || pc.state["GAZP"].FVGExitOrderID != "child" {
		t.Fatalf("recovery: %+v %v", pc.state, err)
	}
}

func TestFVGZeroTargetDoesNotPlaceOrder(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": {Quantity: 10}}, store: env.svc.stateStore()}
	if err := env.svc.placeFVGTarget(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestFVGUnknownMarketResultIsNotRetried(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	e := statestore.Entry{Quantity: 10}
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": e}, store: env.svc.stateStore(), stopByID: map[string]stoporders.ActiveStop{}}
	env.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("timeout")).Once()
	sh := &imodel.Share{ID: "uid-GAZP", Lot: 1}
	sl := env.svc.slotByName("fvg_imbalance")
	if err := env.svc.sell(context.Background(), pc, sl, "GAZP", sh, e, held("GAZP", 10), model.Signal{Kind: model.SignalSell, Reason: "STOCH_EXIT"}); err == nil {
		t.Fatal("expected unknown execution error")
	}
	if err := env.svc.reconcileFVG(context.Background(), pc, sl, "GAZP", sh, held("GAZP", 10)); err == nil {
		t.Fatal("unknown market outcome must block retry")
	}
	if pc.state["GAZP"].FVGExitStopID != "market" {
		t.Fatal("lost unknown outcome")
	}
}

func TestFVGUnknownTargetPlacementFindsTriggeredOrder(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	e := statestore.Entry{Quantity: 10, EntryTime: sharedNow.Add(-time.Hour), StopOrderID: "sl", TakeProfit: 110}
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": e}, store: env.svc.stateStore(), now: sharedNow, stopByID: map[string]stoporders.ActiveStop{"sl": {StopOrderID: "sl", InstrumentUID: "uid-GAZP", Type: pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS}}}
	sh := &imodel.Share{ID: "uid-GAZP", Lot: 1}
	sl := env.svc.slotByName("fvg_imbalance")
	env.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).Return(nil, errors.New("timeout after acceptance")).Once()
	if err := env.svc.placeFVGTarget(context.Background(), pc, sl, "GAZP", sh); err == nil {
		t.Fatal("expected timeout")
	}
	if !pc.state["GAZP"].FVGTargetPending {
		t.Fatal("placement intent not persisted")
	}
	child := "child"
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{StopOrders: []*pb.StopOrder{{StopOrderId: "tp", InstrumentUid: "uid-GAZP", OrderType: pb.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT, Direction: pb.StopOrderDirection_STOP_ORDER_DIRECTION_SELL, ActivationDateTime: timestamppb.New(sharedNow.Add(time.Second)), ExchangeOrderId: &child}}}, nil).Once()
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).Return(&pb.CancelStopOrderResponse{}, nil).Once()
	env.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL, LotsExecuted: 10}, nil).Once()
	if err := env.svc.reconcileFVG(context.Background(), pc, sl, "GAZP", sh, held("GAZP", 10)); err != nil {
		t.Fatal(err)
	}
	got := pc.state["GAZP"]
	if got.FVGExitOrderID != "child" || got.FVGExitStopID != "tp" {
		t.Fatalf("triggered target lost: %+v", got)
	}
}

func TestFVGLostStateAndFlatPortfolioWithOrphanBlocksEntry(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true, universe: []string{"GAZP"}, registered: map[string]bool{"GAAZP": true, "GAZP": true}, sig: model.Signal{Kind: model.SignalBuy}}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	env.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	env.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{StopOrders: []*pb.StopOrder{{StopOrderId: "sl", InstrumentUid: "uid-GAZP", Direction: pb.StopOrderDirection_STOP_ORDER_DIRECTION_SELL, OrderType: pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS}}}, nil)
	if err := env.svc.pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.decideCalls != 0 || !f.said("вход заблокирован") {
		t.Fatalf("orphan did not block entry: %+v", f)
	}
}

func TestFVGUnknownTargetWithoutBrokerEvidenceStaysBlocked(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": {Quantity: 10, EntryTime: sharedNow.Add(-time.Hour), FVGTargetPending: true}}, store: env.svc.stateStore(), now: sharedNow, stopByID: map[string]stoporders.ActiveStop{}}
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{}, nil).Once()
	err := env.svc.reconcileFVG(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1}, held("GAZP", 10))
	if err == nil || !pc.state["GAZP"].FVGTargetPending {
		t.Fatal("unknown placement must stay pending")
	}
}

func TestFVGUnknownStopPlacementFindsTriggeredOrder(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	e := statestore.Entry{Quantity: 10, EntryTime: sharedNow.Add(-time.Hour), TakeProfit: 110}
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": e}, store: env.svc.stateStore(), now: sharedNow, stopByID: map[string]stoporders.ActiveStop{}}
	sh := &imodel.Share{ID: "uid-GAZP", Lot: 1}
	sl := env.svc.slotByName("fvg_imbalance")
	env.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).Return(nil, errors.New("timeout")).Once()
	e = env.svc.replaceStopSafely(context.Background(), pc, sl, "GAZP", sh, e, 90, "SL")
	if !e.FVGStopPending {
		t.Fatal("stop submission intent not saved")
	}
	if err := env.svc.placeFVGTarget(context.Background(), pc, sl, "GAZP", sh); err == nil {
		t.Fatal("target placement must wait for unknown stop")
	}
	child := "child"
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{StopOrders: []*pb.StopOrder{{StopOrderId: "sl", InstrumentUid: "uid-GAZP", OrderType: pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS, Direction: pb.StopOrderDirection_STOP_ORDER_DIRECTION_SELL, ActivationDateTime: timestamppb.New(sharedNow.Add(time.Second)), ExchangeOrderId: &child}}}, nil).Once()
	env.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL, LotsExecuted: 10}, nil).Once()
	if err := env.svc.reconcileFVG(context.Background(), pc, sl, "GAZP", sh, held("GAZP", 10)); err != nil {
		t.Fatal(err)
	}
	if got := pc.state["GAZP"]; got.FVGExitOrderID != "child" {
		t.Fatalf("lost triggered stop: %+v", got)
	}
}

func TestFVGPendingTargetIgnoresEarlierPartialExit(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	pc := &passCtx{state: map[string]statestore.Entry{"GAZP": {Quantity: 6, EntryTime: sharedNow.Add(-time.Hour), FVGTargetPending: true, FVGTargetPendingAt: sharedNow}}, store: env.svc.stateStore(), now: sharedNow, stopByID: map[string]stoporders.ActiveStop{}}
	child := "old-child"
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{StopOrders: []*pb.StopOrder{{StopOrderId: "old", InstrumentUid: "uid-GAZP", OrderType: pb.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT, Direction: pb.StopOrderDirection_STOP_ORDER_DIRECTION_SELL, ActivationDateTime: timestamppb.New(sharedNow.Add(-time.Minute)), ExchangeOrderId: &child}}}, nil).Once()
	err := env.svc.reconcileFVG(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1}, held("GAZP", 6))
	if err == nil || !pc.state["GAZP"].FVGTargetPending || pc.state["GAZP"].FVGExitOrderID != "" {
		t.Fatalf("old partial exit adopted: %+v %v", pc.state, err)
	}
}

func TestSharedExitReconcilesOwnFillsDespiteDensityResidual(t *testing.T) {
	for _, qty := range []int64{4, 10} {
		t.Run(fmt.Sprint(qty), func(t *testing.T) {
			f := &fakeStrategy{name: "fvg_imbalance", trade: true}
			env := newSharedEnv(t, sharedNow, nil, nil, f)
			pc := &passCtx{shared: true, state: map[string]statestore.Entry{"GAZP": {Quantity: 10, FVGExitStopID: "market", FVGExitOrderID: "own-child"}}, store: env.svc.stateStore(), stopByID: map[string]stoporders.ActiveStop{}}
			env.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{OrderId: "own-child", InstrumentUid: "uid-GAZP", Direction: pb.OrderDirection_ORDER_DIRECTION_SELL, LotsRequested: 10, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED, LotsExecuted: qty}, nil).Once()
			if err := env.svc.reconcileFVG(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1}, held("GAZP", 11)); err != nil {
				t.Fatal(err)
			}
			e, exists := pc.state["GAZP"]
			if qty == 10 && exists {
				t.Fatal("density residual kept candle position alive")
			}
			if qty == 4 && (!exists || e.Quantity != 6 || e.FVGExitOrderID != "") {
				t.Fatalf("wrong residual: %+v", e)
			}
		})
	}
}

func TestSharedFlatSnapshotWithoutOwnSaleRetainsInventoryAndProtection(t *testing.T) {
	f := &fakeStrategy{name: "fvg_imbalance", trade: true}
	env := newSharedEnv(t, sharedNow, nil, nil, f)
	pc := &passCtx{shared: true, state: map[string]statestore.Entry{"GAZP": {Ticker: "GAZP", Quantity: 10, EntryTime: sharedNow.Add(-48 * time.Hour), StopOrderID: "protect"}}, store: env.svc.stateStore(), now: sharedNow, stopByID: map[string]stoporders.ActiveStop{"protect": {StopOrderID: "protect", InstrumentUID: "uid-GAZP", Lots: 10, Type: pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS}}}
	err := env.svc.reconcileFVG(context.Background(), pc, env.svc.slotByName("fvg_imbalance"), "GAZP", &imodel.Share{ID: "uid-GAZP", Lot: 1}, nil)
	if err == nil {
		t.Fatal("flat snapshot treated as confirmed own exit")
	}
	if got := pc.state["GAZP"]; got.Quantity != 10 || got.StopOrderID != "protect" {
		t.Fatalf("lost inventory/protection: %+v", got)
	}
}
