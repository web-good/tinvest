package live

import (
	"context"
	"testing"
	imodel "tinvest/internal/model"
	pb "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
	"tinvest/internal/service/trading_strategy/livecore/statestore"

	"github.com/stretchr/testify/mock"
)

func TestSharedActiveOrderAttribution(t *testing.T) {
	for _, name := range []string{"unknown buy", "unknown sell", "known density", "wrong uid", "wrong direction", "invalid quantity", "own request", "own stop child"} {
		t.Run(name, func(t *testing.T) {
			f := zoneFake(0, "GAZP")
			f.trade = true
			e := newSharedEnv(t, sharedNow, nil, nil, f)
			order := &pb.OrderState{OrderId: "density-order", InstrumentUid: "uid-GAZP", Direction: pb.OrderDirection_ORDER_DIRECTION_BUY, LotsRequested: 1}
			state := map[string]statestore.Entry{"GAZP": {EntryOrderID: "own-buy", EntryRequestID: "own-key"}}
			switch name {
			case "unknown buy":
				order.OrderId = "unknown"
			case "unknown sell":
				order.OrderId = "unknown"
				order.Direction = pb.OrderDirection_ORDER_DIRECTION_SELL
			case "wrong uid":
				order.InstrumentUid = "other"
			case "wrong direction":
				order.OrderId = "own-buy"
				order.Direction = pb.OrderDirection_ORDER_DIRECTION_SELL
			case "invalid quantity":
				order.LotsExecuted = 2
			case "own stop child":
				order.OrderId = "new-child"
				order.Direction = pb.OrderDirection_ORDER_DIRECTION_SELL
				state["GAZP"] = statestore.Entry{Strategy: "rsi_zone", StopOrderID: "own-stop"}
				child := "new-child"
				e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(&pb.GetStopOrdersResponse{StopOrders: []*pb.StopOrder{{StopOrderId: "own-stop", ExchangeOrderId: &child}}}, nil).Once()
			case "own request":
				order.OrderId = "resolved-own-buy"
				order.OrderRequestId = "own-key"
			}
			e.orders.EXPECT().GetOrders(mock.Anything, mock.Anything).Return(&pb.GetOrdersResponse{Orders: []*pb.OrderState{order}}, nil).Once()
			blocks, err := e.svc.sharedOrderBlocks(context.Background(), map[string]*imodel.Share{"GAZP": {ID: "uid-GAZP"}}, state, map[string]accountcoord.DensityPosition{"GAZP": {InstrumentID: "uid-GAZP", OrderIDs: []string{"density-order"}}})
			if err != nil {
				t.Fatal(err)
			}
			allowed := name == "known density" || name == "own request" || name == "own stop child"
			if (len(blocks) == 0) != allowed {
				t.Fatalf("incorrect attribution: %+v", blocks)
			}
		})
	}
}
