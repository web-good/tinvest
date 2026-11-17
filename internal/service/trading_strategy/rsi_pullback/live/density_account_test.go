package live

import (
	"context"
	"errors"
	"testing"
	"time"
	"tinvest/internal/config"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	grpcmodel "tinvest/pkg/client/grpc/model"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func sharedCoordinator(t *testing.T, e *env) *accountcoord.Coordinator {
	t.Helper()
	c := accountcoord.New(t.TempDir(), e.svc.cfg.AccountID, statestore.New(e.statePath))
	if err := e.svc.SetAccountCoordinator(c, true); err != nil {
		t.Fatal(err)
	}
	return c
}
func reserveTicker(t *testing.T, c *accountcoord.Coordinator, owner string) accountcoord.Claim {
	t.Helper()
	lease, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	claim := accountcoord.Claim{Ticker: "GAZP", InstrumentID: shareFor("GAZP")[0].ID, Owner: owner, Key: "reserved"}
	if err = c.Reserve(lease, claim); err != nil {
		t.Fatal(err)
	}
	return claim
}
func TestCandleRunnerNeverReconstructsOrTouchesDensityPosition(t *testing.T) {
	guest := zoneFake(model.SignalSell, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, guest)
	c := sharedCoordinator(t, e)
	reserveTicker(t, c, "density")
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(10), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatal(err)
	}
	if guest.decideCalls != 0 || guest.rebuildCalls != 0 {
		t.Fatal("density position adopted by candle guest")
	}
}
func TestClosedDensityMarkerStillExcludesCurrentCandleSnapshot(t *testing.T) {
	guest := zoneFake(model.SignalBuy, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	c := sharedCoordinator(t, e)
	claim := reserveTicker(t, c, "density")
	if err := c.MarkClosed(context.Background(), claim); err != nil {
		t.Fatal(err)
	}
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatal(err)
	}
	if guest.decideCalls != 0 {
		t.Fatal("cleanup admitted ticker in same snapshot")
	}
	claims, err := c.Claims(context.Background())
	if err != nil || len(claims) != 0 {
		t.Fatal(claims, err)
	}
}
func TestDensityCloseDuringPortfolioReadCannotBeAdopted(t *testing.T) {
	guest := zoneFake(model.SignalSell, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	c := sharedCoordinator(t, e)
	claim := reserveTicker(t, c, "density")
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).RunAndReturn(func(context.Context, string) ([]*grpcmodel.Position, error) {
		if err := c.MarkClosed(context.Background(), claim); err != nil {
			t.Error(err)
		}
		return heldGAZP(10), nil
	})
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatal(err)
	}
	if guest.rebuildCalls != 0 || guest.decideCalls != 0 {
		t.Fatal("adopted old portfolio after density closed")
	}
}
func TestUncertainLegacyBuyReservationBlocksSecondBuy(t *testing.T) {
	guest := zoneFake(model.SignalBuy, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	c := sharedCoordinator(t, e)
	reserveTicker(t, c, "rsi_zone")
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Once()
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatal(err)
	}
	if guest.decideCalls != 0 {
		t.Fatal("pending legacy order treated as free ticker")
	}
}
func TestLegacyBuyReservesBeforeOrderAndHandsOffToState(t *testing.T) {
	guest := zoneFake(model.SignalBuy, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, func(c *config.RSIPullbackConfig) { c.TradeEnabled = false }, guest)
	c := sharedCoordinator(t, e)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1000000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1000000.0, nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(_ context.Context, _ *investapi.PostOrderRequest, _ ...grpc.CallOption) {
		claims, err := c.Claims(context.Background())
		if err != nil || claims["GAZP"].Owner != "rsi_zone" {
			t.Error("legacy BUY missing ownership", claims, err)
		}
	}).Return(filledOrder(50, 100), nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).Return(&investapi.PostStopOrderResponse{StopOrderId: "protect"}, nil).Once()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatal(err)
	}
	if e.state(t)["GAZP"].Strategy != "rsi_zone" {
		t.Fatal("owner not handed off to shared state")
	}
	claims, err := c.Claims(context.Background())
	if err != nil || len(claims) != 0 {
		t.Fatal(claims, err)
	}
}

func TestFailedLegacyStateSaveDoesNotDropOwnership(t *testing.T) {
	guest := zoneFake(model.SignalBuy, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	c := sharedCoordinator(t, e)
	e.svc.store = &failingStore{failOn: 1, data: map[string]statestore.Entry{}}
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1000000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1000000.0, nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(filledOrder(50, 100), nil).Once()
	if err := e.svc.Run(context.Background(), dto.Run{}); err == nil {
		t.Fatal("save failure ignored")
	}
	claims, err := c.Claims(context.Background())
	if err != nil || claims["GAZP"].Owner != "rsi_zone" {
		t.Fatal("failed Save released real inventory", claims, err)
	}
}

func TestKnownLegacyRejectionReleasesReservationButTimeoutKeepsIt(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		keep bool
	}{{"rejected", status.Error(codes.InvalidArgument, "30078"), false}, {"timeout", status.Error(codes.DeadlineExceeded, "response lost"), true}} {
		t.Run(tc.name, func(t *testing.T) {
			guest := zoneFake(model.SignalBuy, "GAZP")
			guest.trade = true
			e := newSharedEnv(t, sharedNow, nil, nil, guest)
			c := sharedCoordinator(t, e)
			e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
			e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
			e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1000000.0, nil)
			e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1000000.0, nil)
			e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
			e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
			e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, tc.err).Once()
			if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
				t.Fatal(err)
			}
			claims, err := c.Claims(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if (claims["GAZP"].Key != "") != tc.keep {
				t.Fatal("wrong reservation after request failure", claims)
			}
			if errors.Is(tc.err, context.Canceled) {
				t.Fatal(tc.err)
			}
		})
	}
}

type visibleFailureStore struct{ statestore.Store }

func (s visibleFailureStore) Save(m map[string]statestore.Entry) error {
	if err := s.Store.Save(m); err != nil {
		return err
	}
	return errors.New("directory sync failed after rename")
}
func TestVisibleButFailedSaveStillKeepsLegacyClaim(t *testing.T) {
	guest := zoneFake(model.SignalBuy, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	c := sharedCoordinator(t, e)
	e.svc.store = visibleFailureStore{statestore.New(e.statePath)}
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1000000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1000000.0, nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(filledOrder(50, 100), nil).Once()
	if err := e.svc.Run(context.Background(), dto.Run{}); err == nil {
		t.Fatal("save error ignored")
	}
	claims, err := c.Claims(context.Background())
	if err != nil || claims["GAZP"].Key == "" {
		t.Fatal("failed Save released ownership just because data was visible", claims, err)
	}
}

func TestPendingClaimOutsideConfiguredUniverseIsStillVisited(t *testing.T) {
	guest := zoneFake(model.SignalNone)
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	c := sharedCoordinator(t, e)
	reserveTicker(t, c, "rsi_zone")
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Once()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatal(err)
	}
}

func TestCandleBuysSameTickerWhileDensityOwnsInventoryAndStop(t *testing.T) {
	guest := zoneFake(model.SignalBuy, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, func(c *config.RSIPullbackConfig) { c.TradeEnabled = false }, guest)
	c := sharedCoordinator(t, e)
	reserveTicker(t, c, "density")
	e.orders.EXPECT().GetOrders(mock.Anything, mock.Anything).Return(&investapi.GetOrdersResponse{}, nil).Once()
	c.SetDensitySnapshot(func(context.Context) (map[string]accountcoord.DensityPosition, error) {
		return map[string]accountcoord.DensityPosition{"GAZP": {InstrumentID: "uid-GAZP", Quantity: 10, StopIDs: []string{"density-stop"}}}, nil
	})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(10), nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1000000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1000000.0, nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Once()
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(_ context.Context, _ *investapi.PostOrderRequest, _ ...grpc.CallOption) {
		claims, err := c.Claims(context.Background())
		if err != nil || claims["GAZP"].Owner != "rsi_zone" {
			t.Error("legacy BUY missing ownership", claims, err)
		}
	}).Return(&investapi.PostOrderResponse{OrderId: "own-buy", LotsExecuted: 50, ExecutionReportStatus: investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL}, nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).Return(&investapi.PostStopOrderResponse{StopOrderId: "protect"}, nil).Once()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatal(err)
	}
	if e.state(t)["GAZP"].Strategy != "rsi_zone" {
		t.Fatal("owner not handed off to shared state")
	}
	claims, err := c.CandleClaims(context.Background())
	if err != nil || len(claims) != 0 {
		t.Fatal(claims, err)
	}
}

func TestSharedCandleKeepsCorePriceExitLevels(t *testing.T) {
	guest := zoneFake(model.SignalNone, "GAZP")
	guest.trade = true
	guest.stopLevel = 0
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	c := sharedCoordinator(t, e)
	e.orders.EXPECT().GetOrders(mock.Anything, mock.Anything).Return(&investapi.GetOrdersResponse{}, nil).Once()
	c.SetDensitySnapshot(func(context.Context) (map[string]accountcoord.DensityPosition, error) {
		return map[string]accountcoord.DensityPosition{}, nil
	})
	e.seed(t, statestore.Entry{Ticker: "GAZP", Strategy: "rsi_zone", Quantity: 100, EntryPrice: 100, EntryATR: 10, MaxFav: 100, StopLoss: 90, TakeProfit: 120, EntryTime: sharedNow.Add(-48 * time.Hour)})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatal(err)
	}
	if guest.lastPosition == nil || guest.lastPosition.StopLoss != 90 || guest.lastPosition.TakeProfit != 120 {
		t.Fatalf("core price exits suppressed: %+v", guest.lastPosition)
	}
}

func TestSharedUnknownOrdersAndStopsBlockCandleEntry(t *testing.T) {
	for _, kind := range []string{"buy", "sell", "stop"} {
		t.Run(kind, func(t *testing.T) {
			guest := zoneFake(model.SignalBuy, "GAZP")
			guest.trade = true
			e := newSharedEnv(t, sharedNow, nil, nil, guest)
			c := sharedCoordinator(t, e)
			c.SetDensitySnapshot(func(context.Context) (map[string]accountcoord.DensityPosition, error) {
				return map[string]accountcoord.DensityPosition{}, nil
			})
			e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
			e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
			active := &investapi.GetOrdersResponse{}
			stops := emptyStopList()
			if kind == "stop" {
				stops = &investapi.GetStopOrdersResponse{StopOrders: []*investapi.StopOrder{{StopOrderId: "unknown-stop", InstrumentUid: "uid-GAZP", Direction: investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL, OrderType: investapi.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS}}}
			} else {
				direction := investapi.OrderDirection_ORDER_DIRECTION_BUY
				if kind == "sell" {
					direction = investapi.OrderDirection_ORDER_DIRECTION_SELL
				}
				active.Orders = []*investapi.OrderState{{OrderId: "unknown", InstrumentUid: "uid-GAZP", Direction: direction, LotsRequested: 1}}
			}
			e.orders.EXPECT().GetOrders(mock.Anything, mock.Anything).Return(active, nil)
			e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(stops, nil)
			e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
			err := e.svc.Run(context.Background(), dto.Run{})
			if kind != "stop" && err == nil {
				t.Fatal("unknown active order did not surface error")
			}
			if guest.decideCalls != 0 {
				t.Fatal("candle entry evaluated beside unattributed order")
			}
		})
	}
}

func TestConfirmedCandleSaleReleasesOnlyCandleReservation(t *testing.T) {
	ctx := context.Background()
	guest := zoneFake(model.SignalNone, "GAZP")
	guest.trade = true
	e := newSharedEnv(t, sharedNow, nil, nil, guest)
	c := sharedCoordinator(t, e)
	candle := reserveTicker(t, c, "rsi_zone")
	density := reserveTicker(t, c, "density")
	lease, err := c.Lock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	state := map[string]statestore.Entry{
		"GAZP": {Strategy: candle.Owner, Quantity: 10, FVGExitStopID: "market", FVGExitOrderID: "candle-exit"},
	}
	pc := &passCtx{shared: true, state: state, store: e.svc.stateStore(), lease: lease, claims: map[string]accountcoord.Claim{"GAZP": candle}}
	e.orders.EXPECT().GetOrderState(mock.Anything, mock.Anything).Return(&investapi.OrderState{
		OrderId: "candle-exit", InstrumentUid: candle.InstrumentID, Direction: investapi.OrderDirection_ORDER_DIRECTION_SELL,
		LotsRequested: 10, LotsExecuted: 10, ExecutionReportStatus: investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL,
	}, nil).Once()
	share := shareFor("GAZP")[0]
	share.Lot = 1
	if err := e.svc.reconcileFVG(ctx, pc, e.svc.slotByName(candle.Owner), "GAZP", share, held("GAZP", 5)); err != nil {
		t.Fatal(err)
	}
	claims, err := c.CandleClaims(ctx)
	if err != nil || len(claims) != 0 {
		t.Fatalf("candle reservation survived closure: %v, %v", claims, err)
	}
	densityClaims, err := c.DensityClaims(ctx)
	if err != nil || densityClaims["GAZP"].Key != density.Key {
		t.Fatalf("density reservation changed: %v, %v", densityClaims, err)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("confirmed candle position is still open")
	}
}
