package live

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// hourFake — гость на часовых свечах (как pdl_rsi): fakeStrategy с BarInterval.
type hourFake struct {
	*fakeStrategy
}

func (h *hourFake) BarInterval() enum.Interval { return enum.Hour1 }

func newHourFake(sig model.SignalKind, trade bool) *hourFake {
	f := guestFake("pdl_rsi", sig, "GAZP")
	f.trade = trade
	return &hourFake{fakeStrategy: f}
}

func isH1(interval int32) bool { return interval == enum.Hour1.ToNumberInvestAPI() }

// flatAt1h — ровная лента завершённых часовых баров, последний открылся в end.
func flatAt1h(end time.Time, n int, price float64) []*imodel.CandleItemTechAnalyse {
	out := make([]*imodel.CandleItemTechAnalyse, 0, n)
	for i := range n {
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time: end.Add(-time.Duration(n-1-i) * time.Hour),
			Open: qf(price), High: qf(price * 1.001), Low: qf(price * 0.999), Close: qf(price),
			Volume: 1000, IsComplete: true,
		})
	}
	return out
}

// expectHourCandles — часовая лента и дневки. Получасовых ожиданий нет: запрос Minutes30
// часовым гостем провалит мок.
func (e *env) expectHourCandles(h1, day []*imodel.CandleItemTechAnalyse) {
	e.market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isH1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(h1, nil)
	e.market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(day, nil)
}

var (
	hourBar9  = time.Date(2026, 3, 10, 9, 0, 0, 0, msk)
	hourPass1 = time.Date(2026, 3, 10, 10, 1, 0, 0, msk)  // пасс сразу после закрытия бара 09:00
	hourPass2 = time.Date(2026, 3, 10, 10, 31, 0, 0, msk) // получасовой пасс того же бара
)

func hourEntry(lastBar time.Time, stopID string) statestore.Entry {
	return statestore.Entry{Ticker: "GAZP", Strategy: "pdl_rsi", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: hourPass1.Add(-24 * time.Hour),
		StopOrderID: stopID, StopPrice: 90, StopReason: "SL", LastBar: lastBar}
}

// Вход часового гостя — только на пассе сразу после закрытия часа: на :31 тот же час
// повторно не решается, иначе вход опоздал бы на полчаса против бэктеста.
func TestHourGuestBuysOnlyOnPassAfterHourClose(t *testing.T) {
	g := newHourFake(model.SignalBuy, false)
	e := newSharedEnv(t, hourPass1, nil, nil, g)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectHourCandles(flatAt1h(hourBar9, 400, 100), dailies(hourPass1, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if g.decideCalls != 1 {
		t.Fatalf("Decide = %d, want 1", g.decideCalls)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok || got.Strategy != "pdl_rsi" {
		t.Fatalf("вход не записан: %+v", got)
	}
	if !got.LastBar.Equal(hourBar9) {
		t.Fatalf("LastBar = %v, want %v", got.LastBar, hourBar9)
	}

	// Тот же бар на пассе :31 — ни Decide, ни ордеров, ни сообщений (tg без ожиданий).
	g2 := newHourFake(model.SignalBuy, false)
	e2 := newSharedEnv(t, hourPass2, nil, nil, g2)
	e2.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e2.expectHourCandles(flatAt1h(hourBar9, 400, 100), dailies(hourPass2, 60))
	e2.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)

	if err := e2.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if g2.decideCalls != 0 {
		t.Fatalf("Decide на :31 = %d, want 0", g2.decideCalls)
	}
	if len(e2.state(t)) != 0 {
		t.Fatal("вход на :31 по уже решённому часу")
	}
	if len(g2.msgs) != 0 {
		t.Fatalf("сообщения на :31: %q", g2.msgs)
	}
}

// После покупки на 10:01 сигнальный бар 09:00 уже решён: на 10:31 ядро не спрашивается,
// стоящая на бирже заявка не трогается.
func TestHourGuestDoesNotDecideSignalBarAfterBuy(t *testing.T) {
	g := newHourFake(model.SignalSell, true)
	e := newSharedEnv(t, hourPass2, nil, nil, g)
	e.seed(t, hourEntry(hourBar9, "so-1"))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectHourCandles(flatAt1h(hourBar9, 400, 100), dailies(hourPass2, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", 90, 10), nil)
	// Ни Cancel, ни PostStopOrder, ни ордеров: мок провалит любой такой вызов.

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if g.decideCalls != 0 {
		t.Fatalf("Decide = %d, want 0: бар 09:00 уже решён", g.decideCalls)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok || got.StopOrderID != "so-1" {
		t.Fatalf("позиция или заявка потеряны: %+v", got)
	}
}

// Час 23:00 решается на утреннем пассе 06:01: пасса в 00:01 нет, а выход по нему иначе
// терялся бы каждый вечер. Торги ещё закрыты, поэтому SELL не исполняется сразу: стоп
// остаётся на бирже, выход запоминается в PendingExit и уходит на первом свежем пассе.
func TestHourGuestCatchesUpMissedHourOnNextPass(t *testing.T) {
	now := time.Date(2026, 3, 11, 6, 1, 0, 0, msk)
	last := time.Date(2026, 3, 10, 23, 0, 0, 0, msk)
	g := newHourFake(model.SignalSell, true)
	e := newSharedEnv(t, now, nil, nil, g)
	e.seed(t, hourEntry(last.Add(-time.Hour), "so-1"))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectHourCandles(flatAt1h(last, 400, 100), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", 90, 10), nil)
	// Ни CancelStopOrder, ни PostStopOrder, ни PostOrder: мок провалит любой такой вызов.
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if g.decideCalls != 1 {
		t.Fatalf("Decide = %d, want 1", g.decideCalls)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok {
		t.Fatal("запись удалена до открытия торгов")
	}
	if got.PendingExit != "FAKE" || !got.LastBar.Equal(last) || got.StopOrderID != "so-1" {
		t.Fatalf("PendingExit=%q LastBar=%v StopOrderID=%q, want FAKE/%v/so-1",
			got.PendingExit, got.LastBar, got.StopOrderID, last)
	}
	if g.said("Выход GAZP") {
		t.Fatalf("выход до открытия торгов: %q", g.msgs)
	}

	// Первый свежий пасс после открытия: ядро по новому часу молчит, продаёт PendingExit.
	now2 := time.Date(2026, 3, 11, 8, 1, 0, 0, msk)
	g2 := newHourFake(model.SignalNone, true)
	e2 := newSharedEnv(t, now2, nil, nil, g2)
	e2.seed(t, got)
	e2.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e2.expectHourCandles(flatAt1h(time.Date(2026, 3, 11, 7, 0, 0, 0, msk), 400, 100), dailies(now2, 60))
	e2.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e2.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", 90, 10), nil)
	e2.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-1"
	})).Return(&investapi.CancelStopOrderResponse{}, nil).Once()
	e2.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(filledOrder(10, 100), nil).Once()
	e2.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e2.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if g2.decideCalls != 1 {
		t.Fatalf("Decide на свежем пассе = %d, want 1", g2.decideCalls)
	}
	if !g2.said("Выход GAZP") {
		t.Fatalf("нет выхода на свежем пассе: %q", g2.msgs)
	}
	if _, ok := e2.state(t)["GAZP"]; ok {
		t.Fatal("SELL не закрыл запись")
	}
}

// Настоящее протухание часового бара (уже решённый 09:00 на пассе 11:31) по-прежнему
// даёт алерт и пропуск без решения.
func TestHourGuestStaleDecidedBarAlerts(t *testing.T) {
	now := time.Date(2026, 3, 10, 11, 31, 0, 0, msk)
	g := newHourFake(model.SignalSell, false)
	e := newSharedEnv(t, now, nil, nil, g)
	e.seed(t, hourEntry(hourBar9, ""))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectHourCandles(flatAt1h(hourBar9, 400, 100), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if g.decideCalls != 0 {
		t.Fatalf("Decide = %d, want 0", g.decideCalls)
	}
	if !g.said("протух") {
		t.Fatalf("нет алерта протухания: %q", g.msgs)
	}
	if _, ok := e.state(t)["GAZP"]; !ok {
		t.Fatal("запись потеряна")
	}
}

// Страховка стопа идёт и на пассе без решения: снятая вне раннера заявка перевыставляется.
func TestHourGuestReplacesStopOnSkippedPass(t *testing.T) {
	g := newHourFake(model.SignalSell, true)
	e := newSharedEnv(t, hourPass2, nil, nil, g)
	e.seed(t, hourEntry(hourBar9, "so-1"))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectHourCandles(flatAt1h(hourBar9, 400, 100), dailies(hourPass2, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(emptyStopList(), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED))).
		Return(emptyStopList(), nil)
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).
		Return(&investapi.PostStopOrderResponse{StopOrderId: "so-2"}, nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if g.decideCalls != 0 {
		t.Fatalf("Decide = %d, want 0", g.decideCalls)
	}
	if got := e.state(t)["GAZP"]; got.StopOrderID != "so-2" {
		t.Fatalf("StopOrderID = %q, want so-2", got.StopOrderID)
	}
}

// Получасовой пасс с часовым баром часовой давности — это не протухание.
func TestHourGuestNoStaleAlertOnHalfHourPass(t *testing.T) {
	g := newHourFake(model.SignalNone, false)
	e := newSharedEnv(t, hourPass2, nil, nil, g)
	e.seed(t, hourEntry(hourBar9, ""))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectHourCandles(flatAt1h(hourBar9, 400, 100), dailies(hourPass2, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	var sent []string
	e.tg.EXPECT().SendMessage(mock.Anything).RunAndReturn(func(s string) error {
		sent = append(sent, s)
		return nil
	}).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, m := range append(sent, g.msgs...) {
		if strings.Contains(m, "протух") {
			t.Fatalf("алерт протухания на :31: %q", m)
		}
	}
}

// Выход, принятый на прошлом пассе и не исполненный, повторяется и без нового часа.
func TestHourGuestRetriesPendingExitOnSkippedPass(t *testing.T) {
	g := newHourFake(model.SignalNone, false)
	e := newSharedEnv(t, hourPass2, nil, nil, g)
	entry := hourEntry(hourBar9, "")
	entry.PendingExit = "RSI"
	e.seed(t, entry)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectHourCandles(flatAt1h(hourBar9, 400, 100), dailies(hourPass2, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if g.decideCalls != 0 {
		t.Fatalf("Decide = %d, want 0", g.decideCalls)
	}
	if !g.said("Выход GAZP") {
		t.Fatalf("PendingExit не превратился в SELL: %q", g.msgs)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("запись не удалена после SELL")
	}
}

// Часовой гость покупает на открытии следующего часа (EntryAtNextOpen движка): бар входа —
// тот, внутри которого заявка, а не на бар раньше.
func TestHourGuestPositionEntryTimeIsFillHour(t *testing.T) {
	now := time.Date(2026, 3, 10, 11, 1, 0, 0, msk)
	fill := time.Date(2026, 3, 10, 10, 1, 7, 0, msk)
	g := newHourFake(model.SignalNone, false)
	e := newSharedEnv(t, now, nil, nil, g)
	entry := hourEntry(hourBar9, "")
	entry.EntryTime = fill
	e.seed(t, entry)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectHourCandles(flatAt1h(now.Truncate(time.Hour).Add(-time.Hour), 400, 100), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if g.lastPosition == nil {
		t.Fatal("Decide не получил позицию")
	}
	if want := time.Date(2026, 3, 10, 10, 0, 0, 0, msk); !g.lastPosition.EntryTime.Equal(want) {
		t.Fatalf("Position.EntryTime = %v, want %v", g.lastPosition.EntryTime, want)
	}
	if got := e.state(t)["GAZP"].LastBar; !got.Equal(time.Date(2026, 3, 10, 10, 0, 0, 0, msk)) {
		t.Fatalf("LastBar после Decide = %v, want 10:00", got)
	}
}

// 30m-гость поле LastBar не пишет: формат его записей прежний.
func TestHalfHourGuestStateHasNoLastBar(t *testing.T) {
	g := guestFake("rsi_zone", model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, nil, nil, g)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok {
		t.Fatal("вход не записан")
	}
	if !got.LastBar.IsZero() {
		t.Fatalf("LastBar = %v, want zero", got.LastBar)
	}
	raw, err := os.ReadFile(e.statePath)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if strings.Contains(string(raw), "lastBar") {
		t.Fatalf("в файле стейта 30m-гостя есть lastBar:\n%s", raw)
	}
}
