package live

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	"tinvest/internal/utils"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// withRealGAZP возвращает GAZP боевой литерал (стоп по закрытию) на время теста: TestMain
// подменяет его внутрибарной копией ради механических тестов пакета.
func withRealGAZP(t *testing.T) core.Params {
	t.Helper()
	pinned := paramsByTicker[gazp.Ticker]
	real := realParams[gazp.Ticker]
	if real.StopOnClose != 1 {
		t.Fatalf("боевой GAZP без StopOnClose — тест режима потерял смысл: %+v", real)
	}
	real.EntryBlockFrom, real.EntryBlockTo, real.EntryBlockMonThu = 0, 0, 0 // фикстуры днём, см. TestMain
	paramsByTicker[gazp.Ticker] = real
	t.Cleanup(func() { paramsByTicker[gazp.Ticker] = pinned })
	return real
}

// gazpLevels — защитный и аварийный уровни боевого GAZP от входа entry при дневном ATR 10,
// округлённые вниз до шага 0.01 (как их ставит биржевая заявка).
func gazpLevels(p core.Params, entry float64) (protective, cat float64) {
	prot, _ := core.DesiredStop(p, entry, 10, entry)
	return stoporders.RoundDownToIncrement(prot, 0.01),
		stoporders.RoundDownToIncrement(core.CatastropheStop(p, entry, 10), 0.01)
}

// lastBar30m — ровная лента по 100 с заданным последним баром.
func lastBar30m(end time.Time, n int, o, h, l, c float64) []*imodel.CandleItemTechAnalyse {
	bars := flatAt30m(end, n, 100)
	last := bars[n-1]
	last.Open, last.High, last.Low, last.Close = qf(o), qf(h), qf(l), qf(c)
	return bars
}

// Вход у тикера в режиме ставит на биржу аварийную заявку (CATSL), а не защитный SL.
func TestCloseStopEntryPlacesTheCatastropheOrder(t *testing.T) {
	p := withRealGAZP(t)
	_, cat := gazpLevels(p, 101) // фил по 101
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(filledOrder(40, 101), nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return math.Abs(utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano())-cat) < 1e-9
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-cat"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := e.state(t)["GAZP"]
	if got.StopOrderID != "so-cat" || got.StopReason != "CATSL" || math.Abs(got.StopPrice-cat) > 1e-9 {
		t.Fatalf("заявка после входа: %q/%v/%q, want so-cat/%v/CATSL", got.StopOrderID, got.StopPrice, got.StopReason, cat)
	}
}

// closeStopEnv — открытая позиция GAZP (вход 100) с аварийной заявкой so-1 в стейте; боевой
// литерал GAZP включён. Возвращает аварийный уровень.
func closeStopEnv(t *testing.T, bars []*imodel.CandleItemTechAnalyse) (*env, float64) {
	t.Helper()
	p := withRealGAZP(t)
	_, cat := gazpLevels(p, 100)
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	entry := openEntry("so-1")
	entry.StopPrice, entry.StopReason = cat, "CATSL"
	e.seed(t, entry)
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(bars, dailies(now, 60))
	return e, cat
}

// Бар закрылся ниже защитного уровня (но выше аварийного) — SLC: снять аварийную заявку,
// затем продать рыночной заявкой; стейт очищен, выход с причиной SLC.
func TestCloseStopSellsOnTheCloseBelowTheLevel(t *testing.T) {
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	p := realParams[gazp.Ticker]
	prot, cat := gazpLevels(p, 100)
	e, _ := closeStopEnv(t, lastBar30m(lastBar, 400, prot+1, prot+1, cat+2, prot-1))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", cat, 10), nil)
	var seq []string
	e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-1"
	})).Run(func(_ context.Context, _ *investapi.CancelStopOrderRequest, _ ...grpc.CallOption) {
		seq = append(seq, "cancel")
	}).Return(&investapi.CancelStopOrderResponse{}, nil).Once()
	e.orders.EXPECT().PostOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostOrderRequest) bool {
		return in.GetDirection() == investapi.OrderDirection_ORDER_DIRECTION_SELL
	}), mock.Anything, mock.Anything).Run(func(_ context.Context, _ *investapi.PostOrderRequest, _ ...grpc.CallOption) {
		seq = append(seq, "sell")
	}).Return(filledOrder(10, int64(prot-1)), nil).Once()
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "Выход") && strings.Contains(s, "SLC")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(seq) != 2 || seq[0] != "cancel" || seq[1] != "sell" {
		t.Fatalf("порядок вызовов %v, want [cancel sell]", seq)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("после SLC стейт обязан очиститься, got %+v", st)
	}
}

// Тень проколола защитный уровень, бар закрылся выше — ради этого режим и сделан: ни продажи,
// ни переноса заявки.
func TestCloseStopIgnoresAWickThroughTheLevel(t *testing.T) {
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	p := realParams[gazp.Ticker]
	prot, cat := gazpLevels(p, 100)
	e, _ := closeStopEnv(t, lastBar30m(lastBar, 400, 100, 100, cat+1, prot+2))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", cat, 10), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	e.stops.AssertNotCalled(t, "CancelStopOrder", mock.Anything, mock.Anything)
	if _, ok := e.state(t)["GAZP"]; !ok {
		t.Fatal("позиция закрыта тенью — режим стопа по закрытию не работает")
	}
}

// Аварийная заявка сработала внутри бара, а портфель брокера ещё показывает позицию. Ядро на
// этом баре тоже скажет CATSL, но продавать второй раз нельзя: только уведомление о выходе.
func TestFiredCatastropheStopDoesNotSellTwice(t *testing.T) {
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	p := realParams[gazp.Ticker]
	_, cat := gazpLevels(p, 100)
	e, _ := closeStopEnv(t, lastBar30m(lastBar, 400, 100, 100, cat-2, cat+1))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(emptyStopList(), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED))).
		Return(executedList("so-1"), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "Выход") && strings.Contains(s, "CATSL")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("после сработавшей аварийной заявки стейт обязан очиститься, got %+v", st)
	}
}

// Позиция без стейта у тикера в режиме: реконструкция ставит аварийную заявку, а не SL.
func TestReconstructedCloseStopPositionGetsTheCatastropheOrder(t *testing.T) {
	p := withRealGAZP(t)
	_, cat := gazpLevels(p, 100)
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	fill := time.Date(2026, 3, 5, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(flat30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(
		[]*grpcmodel.Position{{ShareID: "uid-GAZP", InstrumentType: "share", Quantity: 500,
			PurchasePrice: gq(100)}}, nil)
	e.ops.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: true, Date: fill, Price: 100, Quantity: 500}}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).
		Return(&investapi.PostStopOrderResponse{StopOrderId: "so-rec"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := e.state(t)["GAZP"]
	// EntryATR реконструкции ≈ 10, но не обязан быть ровно 10: уровень сверяется с тем, что
	// ядро даёт от восстановленной записи.
	want := stoporders.RoundDownToIncrement(core.CatastropheStop(p, got.EntryPrice, got.EntryATR), 0.01)
	if got.StopReason != "CATSL" || math.Abs(got.StopPrice-want) > 1e-9 {
		t.Fatalf("заявка после реконструкции: %v/%q, want %v/CATSL (ориентир %v)", got.StopPrice, got.StopReason, want, cat)
	}
	if got.StopOrderID != "so-rec" {
		t.Fatalf("StopOrderID = %q, want so-rec", got.StopOrderID)
	}
}
