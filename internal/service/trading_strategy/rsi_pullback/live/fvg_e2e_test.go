package live

import (
	"context"
	"math"
	"testing"
	"time"

	"tinvest/internal/config"
	investapi "tinvest/internal/pb/v1"
	fvglive "tinvest/internal/service/trading_strategy/fvg_imbalance/live"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
	tgmocks "tinvest/pkg/client/telegram/mocks"

	"github.com/stretchr/testify/mock"
)

type fvgForced struct{ *fvglive.Strategy }
type fvgBuy struct{ adapter.Decider }

func (d fvgBuy) Decide(md strategy.MarketData) model.Signal {
	return model.Signal{Kind: model.SignalBuy, Price: md.Price, ATR: 10, StopDistance: 15, TargetDistance: 10, MinEntryPrice: 100, FillBy: md.Times[len(md.Times)-1].Add(30 * time.Minute)}
}
func (f fvgForced) Decider(ticker string) (adapter.Decider, bool) {
	d, ok := f.Strategy.Decider(ticker)
	if !ok {
		return nil, false
	}
	return fvgBuy{d}, true
}

func TestFVGFillBelowFloorKeepsPositionWithFillBasedProtection(t *testing.T) {
	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	cfg := &config.FVGImbalanceConfig{Tickers: []string{"BSPB"}, BuyPct: 5, TradeEnabled: true, NotifyEnabled: true}
	e := newSharedEnv(t, sharedNow, nil, nil, fvgForced{fvglive.New(cfg, tg)})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("BSPB"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 101), dailies(sharedNow, 300))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(filledOrder(10, 99), nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return math.Abs(utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano())-84) < 0.001
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "fvg-stop"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatal(err)
	}
	entry := e.state(t)["BSPB"]
	if entry.Strategy != "fvg_imbalance" || entry.EntryPrice != 99 || entry.StopLoss != 84 || entry.TakeProfit != 109 || entry.StopOrderID != "fvg-stop" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
}
