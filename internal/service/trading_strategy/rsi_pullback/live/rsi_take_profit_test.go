package live

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	pb "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	zonelive "tinvest/internal/service/trading_strategy/rsi_zone/live"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
)

type rsiSignal struct {
	sig   model.Signal
	calls int
}

func (d *rsiSignal) Lookback() int                           { return 2 }
func (d *rsiSignal) Decide(strategy.MarketData) model.Signal { d.calls++; return d.sig }

func rsiOrderEnv(t *testing.T, name string) (*env, *slot, string) {
	t.Helper()
	var e *env
	ticker := "GAZP"
	if name == "rsi_zone" {
		ticker = "AFKS"
		zone := zonelive.New(&config.RSIZoneConfig{Tickers: []string{ticker}, TradeEnabled: true, BuyPct: 5}, nil)
		e = newSharedEnv(t, sharedNow, nil, nil, zone)
	} else {
		e = newEnv(t, ticker, sharedNow, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
		e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	}
	return e, e.svc.slotByName(name), ticker
}

func rsiPass(e *env) *passCtx {
	return &passCtx{state: map[string]statestore.Entry{}, store: e.svc.stateStore(), now: sharedNow,
		stopByID: map[string]stoporders.ActiveStop{}, stopByInstrument: map[string]stoporders.ActiveStop{}}
}

func expectTakeProfit(t *testing.T, e *env, lots int64, target float64) {
	t.Helper()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.PostStopOrderRequest) bool {
		return r.GetStopOrderType() == pb.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT && r.GetQuantity() == lots &&
			(target == 0 || math.Abs(utils.CombinePrice(r.GetStopPrice().GetUnits(), r.GetStopPrice().GetNano())-target) < 1e-8)
	})).Return(&pb.PostStopOrderResponse{StopOrderId: "tp"}, nil).Once()
}

func TestRSIEntryPlacesBrokerTargetFromFill(t *testing.T) {
	for _, name := range []string{"rsi_pullback", "rsi_zone"} {
		t.Run(name, func(t *testing.T) {
			e, sl, ticker := rsiOrderEnv(t, name)
			pc := rsiPass(e)
			distance := 15.0
			if name == "rsi_pullback" {
				p, _ := ParamsFor(ticker)
				distance = p.TPDailyATR * 10
			}
			d := &rsiSignal{sig: model.Signal{Kind: model.SignalBuy, Price: 100, ATR: 10, TakeProfit: 100 + distance, TargetDistance: distance}}
			e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
			e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
			e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(filledOrder(10, 101), nil).Once()
			e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.PostStopOrderRequest) bool {
				return r.GetStopOrderType() == pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS
			})).Return(&pb.PostStopOrderResponse{StopOrderId: "sl"}, nil).Once()
			e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.PostStopOrderRequest) bool {
				return r.GetStopOrderType() == pb.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT && r.GetQuantity() == 10 &&
					r.GetExchangeOrderType() == pb.ExchangeOrderType_EXCHANGE_ORDER_TYPE_MARKET &&
					math.Abs(utils.CombinePrice(r.GetStopPrice().GetUnits(), r.GetStopPrice().GetNano())-(101+distance)) < 1e-8
			})).Return(&pb.PostStopOrderResponse{StopOrderId: "tp"}, nil).Once()
			if _, err := e.svc.buy(context.Background(), pc, sl, ticker, shareFor(ticker)[0], d, strategy.MarketData{Price: 100}); err != nil {
				t.Fatal(err)
			}
			entry := e.state(t)[ticker]
			if entry.StopOrderID != "sl" || entry.TakeProfitOrderID != "tp" || entry.TakeProfit != 101+distance {
				t.Fatalf("unprotected fill: %+v", entry)
			}
		})
	}
}

func TestRSIIndicatorExitCancelsBothOrders(t *testing.T) {
	for _, name := range []string{"rsi_pullback", "rsi_zone"} {
		t.Run(name, func(t *testing.T) {
			e, sl, ticker := rsiOrderEnv(t, name)
			pc := rsiPass(e)
			entry := statestore.Entry{Ticker: ticker, Quantity: 100, StopOrderID: "sl", TakeProfitOrderID: "tp"}
			pc.state[ticker] = entry
			cancelled := map[string]bool{}
			e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, r *pb.CancelStopOrderRequest, _ ...grpc.CallOption) (*pb.CancelStopOrderResponse, error) {
				cancelled[r.GetStopOrderId()] = true
				return &pb.CancelStopOrderResponse{}, nil
			}).Twice()
			e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ *pb.PostOrderRequest, _ ...grpc.CallOption) (*pb.PostOrderResponse, error) {
				if !cancelled["sl"] || !cancelled["tp"] {
					t.Fatal("sale before cancelling both orders")
				}
				return &pb.PostOrderResponse{OrderId: "sale", LotsExecuted: 10}, nil
			}).Once()
			if err := e.svc.sell(context.Background(), pc, sl, ticker, shareFor(ticker)[0], entry, held(ticker, 100), model.Signal{Kind: model.SignalSell, Reason: "RSI"}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRSITriggeredTargetBlocksAnotherSale(t *testing.T) {
	for _, name := range []string{"rsi_pullback", "rsi_zone"} {
		t.Run(name, func(t *testing.T) {
			e, sl, ticker := rsiOrderEnv(t, name)
			pc := rsiPass(e)
			pc.state[ticker] = statestore.Entry{Ticker: ticker, Quantity: 100, EntryPrice: 100, EntryATR: 10, TakeProfit: 115, StopOrderID: "sl", TakeProfitOrderID: "tp"}
			pc.stopByID["sl"] = stoporders.ActiveStop{InstrumentUID: "uid-" + ticker, StopOrderID: "sl", Type: pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS, Lots: 10}
			child := "tp-child"
			e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{StopOrders: []*pb.StopOrder{{StopOrderId: "tp", ExchangeOrderId: &child}}}, nil).Once()
			e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.CancelStopOrderRequest) bool { return r.GetStopOrderId() == "sl" })).Return(&pb.CancelStopOrderResponse{}, nil).Once()
			e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{OrderId: child, LotsExecuted: 10, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL}, nil).Once()
			d := &rsiSignal{sig: model.Signal{Kind: model.SignalSell, Reason: "RSI"}}
			if err := e.svc.manage(context.Background(), pc, sl, ticker, &imodel.Share{ID: "uid-" + ticker, Lot: 10}, d, strategy.MarketData{}, held(ticker, 100), true); err != nil {
				t.Fatal(err)
			}
			if d.calls != 0 || pc.state[ticker].FVGExitOrderID != child {
				t.Fatalf("triggered target was not retained: %+v", pc.state[ticker])
			}
		})
	}
}

func TestRSIOldPositionGainsFrozenTarget(t *testing.T) {
	for _, name := range []string{"rsi_pullback", "rsi_zone"} {
		t.Run(name, func(t *testing.T) {
			e, sl, ticker := rsiOrderEnv(t, name)
			pc := rsiPass(e)
			pc.state[ticker] = statestore.Entry{Ticker: ticker, EntryPrice: 100, EntryATR: 10, MaxFav: 100, Quantity: 100, StopOrderID: "sl"}
			pc.stopByID["sl"] = stoporders.ActiveStop{InstrumentUID: "uid-" + ticker, StopOrderID: "sl", Type: pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS, Lots: 10, StopPrice: 99}
			expectTakeProfit(t, e, 10, 0)
			md := strategy.MarketData{Price: 100, Times: []time.Time{sharedLastBar}}
			if err := e.svc.manage(context.Background(), pc, sl, ticker, shareFor(ticker)[0], &rsiSignal{}, md, held(ticker, 100), true); err != nil {
				t.Fatal(err)
			}
			entry := e.state(t)[ticker]
			want := 115.0
			if name == "rsi_pullback" {
				p, _ := ParamsFor(ticker)
				want = 100 + p.TPDailyATR*10
			}
			if entry.TakeProfit != want || entry.TakeProfitOrderID != "tp" {
				t.Fatalf("old entry not upgraded: %+v", entry)
			}
			// The next pass reuses the saved target/ID and posts no second order.
			pc.stopByID["tp"] = stoporders.ActiveStop{InstrumentUID: "uid-" + ticker, StopOrderID: "tp", Type: pb.StopOrderType_STOP_ORDER_TYPE_TAKE_PROFIT, Lots: 10}
			if err := e.svc.manage(context.Background(), pc, sl, ticker, shareFor(ticker)[0], &rsiSignal{}, md, held(ticker, 100), true); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRSITargetCancelFailureBlocksIndicatorSale(t *testing.T) {
	for _, name := range []string{"rsi_pullback", "rsi_zone"} {
		t.Run(name, func(t *testing.T) {
			e, sl, ticker := rsiOrderEnv(t, name)
			pc := rsiPass(e)
			entry := statestore.Entry{Ticker: ticker, Quantity: 100, StopOrderID: "sl", TakeProfitOrderID: "tp"}
			pc.state[ticker] = entry
			e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.CancelStopOrderRequest) bool { return r.GetStopOrderId() == "tp" })).Return(nil, errors.New("unavailable")).Once()
			if err := e.svc.sell(context.Background(), pc, sl, ticker, shareFor(ticker)[0], entry, held(ticker, 100), model.Signal{Kind: model.SignalSell, Reason: "RSI"}); err == nil {
				t.Fatal("missing cancellation error")
			}
			got := e.state(t)[ticker]
			if got.StopOrderID != "sl" || got.TakeProfitOrderID != "tp" || got.PendingExit != "RSI" {
				t.Fatalf("pending exit lost: %+v", got)
			}
		})
	}
}

func TestRSITargetTimeoutRetainsIntentAndDoesNotRetry(t *testing.T) {
	for _, name := range []string{"rsi_pullback", "rsi_zone"} {
		t.Run(name, func(t *testing.T) {
			e, sl, ticker := rsiOrderEnv(t, name)
			pc := rsiPass(e)
			pc.state[ticker] = statestore.Entry{Ticker: ticker, Quantity: 100, EntryPrice: 100, EntryATR: 10, StopOrderID: "sl", TakeProfit: 115}
			e.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).Return(nil, errors.New("timeout after acceptance")).Once()
			sh := shareFor(ticker)[0]
			if err := e.svc.placeTarget(context.Background(), pc, sl, ticker, sh); err == nil {
				t.Fatal("missing submission error")
			}
			if !e.state(t)[ticker].FVGTargetPending {
				t.Fatal("lost submission intent")
			}
			if err := e.svc.placeTarget(context.Background(), pc, sl, ticker, sh); err == nil {
				t.Fatal("unknown submission was retried")
			}
		})
	}
}

func TestRSIPendingEntryTargetFollowsConfirmedAverageFill(t *testing.T) {
	for _, name := range []string{"rsi_pullback", "rsi_zone"} {
		t.Run(name, func(t *testing.T) {
			e, sl, ticker := rsiOrderEnv(t, name)
			pc := rsiPass(e)
			pc.state[ticker] = statestore.Entry{Ticker: ticker, EntryRequestID: "buy-key", EntryOrderID: "buy", EntryRequestedLots: 10, EntryPrice: 100, EntryATR: 10, MaxFav: 100, TakeProfit: 115}
			e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&pb.OrderState{OrderId: "buy", InstrumentUid: "uid-" + ticker, Direction: pb.OrderDirection_ORDER_DIRECTION_BUY, LotsRequested: 10, LotsExecuted: 3, AveragePositionPrice: &pb.MoneyValue{Units: 102}, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED}, nil).Once()
			e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.PostStopOrderRequest) bool {
				return r.GetStopOrderType() == pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS && r.GetQuantity() == 3
			})).Return(&pb.PostStopOrderResponse{StopOrderId: "sl"}, nil).Once()
			done, err := e.svc.reconcilePendingEntry(context.Background(), pc, sl, ticker, shareFor(ticker)[0])
			if err != nil || !done {
				t.Fatal(done, err)
			}
			want := 117.0
			if name == "rsi_pullback" {
				p, _ := ParamsFor(ticker)
				want = 102 + p.TPDailyATR*10
			}
			got := e.state(t)[ticker]
			if got.EntryPrice != 102 || got.TakeProfit != want || got.Quantity != 30 {
				t.Fatalf("target frozen from signal instead of fill: %+v", got)
			}
		})
	}
}

func TestRSIPartialBuyDefersTargetUntilBuyIsTerminal(t *testing.T) {
	for _, name := range []string{"rsi_pullback", "rsi_zone"} {
		t.Run(name, func(t *testing.T) {
			e, sl, ticker := rsiOrderEnv(t, name)
			pc := rsiPass(e) // no density/shared account mode
			e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
			e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
			e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&pb.PostOrderResponse{OrderId: "buy", LotsExecuted: 3, ExecutedOrderPrice: &pb.MoneyValue{Units: 100}, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL}, nil).Once()
			partial := &pb.OrderState{OrderId: "buy", InstrumentUid: "uid-" + ticker, Direction: pb.OrderDirection_ORDER_DIRECTION_BUY, LotsRequested: 50, LotsExecuted: 3, AveragePositionPrice: &pb.MoneyValue{Units: 100}, ExecutionReportStatus: pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL}
			e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(partial, nil).Once()
			e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.PostStopOrderRequest) bool {
				return r.GetStopOrderType() == pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS && r.GetQuantity() == 3
			})).Return(&pb.PostStopOrderResponse{StopOrderId: "sl"}, nil).Once()
			e.orders.EXPECT().CancelOrder(mock.Anything, mock.Anything).Return(nil, errors.New("cancel response lost")).Once()
			distance := 15.0
			if name == "rsi_pullback" {
				p, _ := ParamsFor(ticker)
				distance = p.TPDailyATR * 10
			}
			d := &rsiSignal{sig: model.Signal{Kind: model.SignalBuy, Price: 100, ATR: 10, TakeProfit: 100 + distance, TargetDistance: distance}}
			sh := shareFor(ticker)[0]
			if _, err := e.svc.buy(context.Background(), pc, sl, ticker, sh, d, strategy.MarketData{Price: 100}); err == nil {
				t.Fatal("pending BUY must retain its uncertain cancellation")
			}
			got := e.state(t)[ticker]
			if got.EntryRequestID == "" || got.TakeProfitOrderID != "" || got.Quantity != 30 {
				t.Fatalf("TP placed before BUY settled: %+v", got)
			}
			pc.stopByID["sl"] = stoporders.ActiveStop{InstrumentUID: sh.ID, StopOrderID: "sl", Type: pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS, Lots: 3}
			final := proto.Clone(partial).(*pb.OrderState)
			final.LotsExecuted = 5
			final.AveragePositionPrice = &pb.MoneyValue{Units: 102}
			final.ExecutionReportStatus = pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED
			e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(final, nil).Twice()
			e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).Return(&pb.CancelStopOrderResponse{}, nil).Once()
			e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
			e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(r *pb.PostStopOrderRequest) bool {
				return r.GetStopOrderType() == pb.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS && r.GetQuantity() == 5
			})).Return(&pb.PostStopOrderResponse{StopOrderId: "sl-final"}, nil).Once()
			done, err := e.svc.reconcilePendingEntry(context.Background(), pc, sl, ticker, sh)
			if err != nil || !done {
				t.Fatal(done, err)
			}
			expectTakeProfit(t, e, 5, 102+distance)
			if err := e.svc.placeTarget(context.Background(), pc, sl, ticker, sh); err != nil {
				t.Fatal(err)
			}
			got = e.state(t)[ticker]
			if got.EntryPrice != 102 || got.TakeProfit != 102+distance || got.Quantity != 50 || got.EntryRequestID != "" || got.TakeProfitOrderID != "tp" {
				t.Fatalf("final BUY not reflected in TP: %+v", got)
			}
		})
	}
}
