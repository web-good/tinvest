package live

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/config"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	pdlrsilive "tinvest/internal/service/trading_strategy/pdl_rsi/live"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
	tgmocks "tinvest/pkg/client/telegram/mocks"
)

// forcedBuy — настоящий pdl_rsi-гость, у которого Decider всегда отдаёт BUY с ATR 10:
// синтетическая лента не умеет честно собрать сигнал ядра (RSI, стохастик, EMA, дневки), а
// проверяется здесь проводка раннера — имя, BarInterval, стоп из DesiredStop, LastBar.
type forcedBuy struct{ *pdlrsilive.Strategy }

type forcedDecider struct{ adapter.Decider }

func (d forcedDecider) Decide(md strategy.MarketData) model.Signal {
	return model.Signal{Kind: model.SignalBuy, Reason: "FORCED", ATR: 10, Price: md.Price}
}

func (f forcedBuy) Decider(ticker string) (adapter.Decider, bool) {
	d, ok := f.Strategy.Decider(ticker)
	if !ok {
		return nil, false
	}
	return forcedDecider{d}, true
}

func pdlGuest(trade bool, tg *tgmocks.MockClient) forcedBuy {
	cfg := &config.PDLRSIConfig{Tickers: []string{"GAZP"}, BuyPct: 5, TradeEnabled: trade, NotifyEnabled: tg != nil}
	if tg == nil {
		return forcedBuy{pdlrsilive.New(cfg, nil)}
	}
	return forcedBuy{pdlrsilive.New(cfg, tg)}
}

// Бумажный режим: сигнал есть, TradeEnabled=false — ни PostOrder, ни PostStopOrder (моки без
// ожиданий падают на любом вызове).
func TestPDLRSIGuestPaperDoesNotTrade(t *testing.T) {
	e := newSharedEnv(t, hourPass1, nil, nil, pdlGuest(false, nil))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectHourCandles(flatAt1h(hourBar9, 600, 100), dailies(hourPass1, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// rsi_pullback по GAZP молчит (ровная лента) — тикер берёт pdl_rsi: запись с его именем и
// LastBar, биржевой стоп на уровне core.StopLevel от цены фила, округлённом так, как округляет раннер.
func TestPDLRSIGuestTakesTickerOnlyWhenPullbackSilent(t *testing.T) {
	guestTG := tgmocks.NewMockClient(t)
	guestTG.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	g := pdlGuest(true, guestTG)
	p, ok := pdlrsilive.ParamsFor("GAZP")
	if !ok {
		t.Fatal("GAZP не зарегистрирован")
	}
	const fill, atr = 100.0, 10.0
	wantStop := stoporders.RoundDownToIncrement(core.StopLevel(p, fill, atr), 0.01)

	e := newSharedEnv(t, hourPass1, []string{"GAZP"}, nil, g)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	// Свечи и получасовые для rsi_pullback, и часовые для гостя; дневки общие.
	e.market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isM30),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(flatAt30m(hourPass1.Add(-31*time.Minute), 400, 100), nil)
	e.market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isH1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(flatAt1h(hourBar9, 600, 100), nil)
	e.market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(dailies(hourPass1, 60), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.orders.EXPECT().PostOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostOrderRequest) bool {
		return in.GetDirection() == investapi.OrderDirection_ORDER_DIRECTION_BUY
	}), mock.Anything, mock.Anything).Return(filledOrder(50, int64(fill)), nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return math.Abs(utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano())-wantStop) < 1e-9
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-pdl"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok || got.Strategy != "pdl_rsi" {
		t.Fatalf("нет входа pdl_rsi: %+v", e.state(t))
	}
	if !got.LastBar.Equal(hourBar9) {
		t.Fatalf("LastBar = %v, want %v", got.LastBar, hourBar9)
	}
	if got.StopOrderID != "so-pdl" || math.Abs(got.StopPrice-wantStop) > 1e-9 || got.StopReason != "SL" {
		t.Fatalf("стоп = %q %v/%q, want so-pdl %v/SL", got.StopOrderID, got.StopPrice, got.StopReason, wantStop)
	}
}

// Стартовое объявление pdl_rsi уходит в тему гостя.
func TestAnnounceIncludesPDLRSI(t *testing.T) {
	guestTG := tgmocks.NewMockClient(t)
	guestTG.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return containsAll(s, "PDL RSI", "GAZP")
	})).Return(nil).Once()
	e := newSharedEnv(t, hourPass1, []string{"GAZP"}, nil, pdlGuest(false, guestTG))
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Once()

	e.svc.Announce()
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
