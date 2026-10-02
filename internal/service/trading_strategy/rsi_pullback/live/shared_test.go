package live

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	candlemocks "tinvest/internal/service/trading_strategy/livecore/candles/mocks"
	execmocks "tinvest/internal/service/trading_strategy/livecore/executor/mocks"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	stopmocks "tinvest/internal/service/trading_strategy/livecore/stoporders/mocks"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	livemocks "tinvest/internal/service/trading_strategy/rsi_pullback/live/mocks"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
	grpcmodel "tinvest/pkg/client/grpc/model"
	tgmocks "tinvest/pkg/client/telegram/mocks"
)

// fakeStrategy — гость раннера с заранее заданным вердиктом. Реальное ядро rsi_zone здесь
// не нужно: проверяется маршрутизация раннера, а не решения стратегии.
type fakeStrategy struct {
	name       string
	universe   []string
	registered map[string]bool
	sig        model.Signal // Kind/Reason/ATR; Price подставляется из md
	stopLevel  float64
	trade      bool

	decideCalls  int
	rebuildCalls int
	lastPosition *strategy.Position // md.Position последнего Decide
	msgs         []string
}

func (f *fakeStrategy) Name() string       { return f.name }
func (f *fakeStrategy) Label() string      { return "FAKE" }
func (f *fakeStrategy) Tickers() []string  { return f.universe }
func (f *fakeStrategy) BuyPct() float64    { return 5 }
func (f *fakeStrategy) TradeEnabled() bool { return f.trade }
func (f *fakeStrategy) Notify(msg string)  { f.msgs = append(f.msgs, msg) }

func (f *fakeStrategy) Decider(ticker string) (adapter.Decider, bool) {
	if !f.registered[ticker] {
		return nil, false
	}
	return fakeDecider{f}, true
}

func (f *fakeStrategy) DesiredStop(string, statestore.Entry) (float64, string) {
	if f.stopLevel <= 0 {
		return 0, ""
	}
	return f.stopLevel, "SL"
}

func (f *fakeStrategy) Reconstruct(_ context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	f.rebuildCalls++
	return statestore.Entry{Ticker: in.Ticker, EntryTime: in.Now.Add(-24 * time.Hour),
		EntryPrice: in.PurchasePrice, EntryATR: 10, MaxFav: in.PurchasePrice}, nil
}

type fakeDecider struct{ f *fakeStrategy }

func (d fakeDecider) Lookback() int { return 50 }
func (d fakeDecider) Decide(md strategy.MarketData) model.Signal {
	d.f.decideCalls++
	d.f.lastPosition = md.Position
	sig := d.f.sig
	sig.Price = md.Price
	return sig
}

func (f *fakeStrategy) said(sub string) bool {
	for _, m := range f.msgs {
		if strings.Contains(m, sub) {
			return true
		}
	}
	return false
}

// newSharedEnv — newEnv с гостями: вселенная rsi_pullback задаётся явно.
func newSharedEnv(t *testing.T, now time.Time, pullTickers []string,
	tweak func(*config.RSIPullbackConfig), guests ...adapter.Strategy) *env {
	t.Helper()
	c := cfgFor("GAZP")
	c.Tickers = pullTickers
	if tweak != nil {
		tweak(c)
	}
	e := &env{
		statePath:   filepath.Join(t.TempDir(), "state.json"),
		instruments: livemocks.NewMockinstrumentsClient(t),
		market:      candlemocks.NewMockCandleClient(t),
		ops:         livemocks.NewMockoperationsClient(t),
		orders:      execmocks.NewMockOrdersClient(t),
		stops:       stopmocks.NewMockClient(t),
		tg:          tgmocks.NewMockClient(t),
	}
	e.svc = NewService(e.instruments, e.market, e.ops, e.orders, e.stops, e.tg, c, guests...)
	e.svc.statePath = e.statePath
	e.svc.now = func() time.Time { return now }
	return e
}

func sharesOf(tickers ...string) []*imodel.Share {
	var out []*imodel.Share
	for _, t := range tickers {
		out = append(out, shareFor(t)...)
	}
	return out
}

func held(ticker string, qty int64) *grpcmodel.Position {
	return &grpcmodel.Position{ShareID: "uid-" + ticker, InstrumentType: "share",
		Quantity: qty, PurchasePrice: gq(100)}
}

func zoneFake(sig model.SignalKind, universe ...string) *fakeStrategy {
	reg := map[string]bool{}
	for _, t := range universe {
		reg[t] = true
	}
	return &fakeStrategy{name: "rsi_zone", universe: universe, registered: reg,
		sig: model.Signal{Kind: sig, Reason: "FAKE", ATR: 10}, stopLevel: 90}
}

var (
	sharedNow     = time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	sharedLastBar = time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
)

// Оба дают BUY по свободному тикеру на одном баре — входит rsi_pullback, гостя не спрашивают.
func TestPullbackWinsTheSameBarBuy(t *testing.T) {
	zone := zoneFake(model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(pullback30m(sharedLastBar, 400), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"].Strategy; got != "rsi_pullback" {
		t.Fatalf("владелец = %q, want rsi_pullback", got)
	}
	if zone.decideCalls != 0 {
		t.Fatalf("гостя спросили %d раз после BUY rsi_pullback", zone.decideCalls)
	}
}

// rsi_pullback молчит — входит гость, запись помечена его именем, стоп — от его DesiredStop.
func TestGuestEntersWhenPullbackIsSilent(t *testing.T) {
	zone := zoneFake(model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	entry := e.state(t)["GAZP"]
	if entry.Strategy != "rsi_zone" {
		t.Fatalf("владелец = %q, want rsi_zone", entry.Strategy)
	}
	if entry.StopPrice != 90 || entry.StopReason != "SL" {
		t.Fatalf("стоп = %v/%q, want 90/SL от DesiredStop гостя", entry.StopPrice, entry.StopReason)
	}
}

// Позицию ведёт владелец из стейта: SELL гостя закрывает позицию, которую открыл гость.
func TestOwnerFromStateManagesThePosition(t *testing.T) {
	zone := zoneFake(model.SignalSell, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.seed(t, statestore.Entry{Ticker: "GAZP", Strategy: "rsi_zone", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: sharedNow.Add(-48 * time.Hour)})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("SELL владельца-гостя не закрыл позицию")
	}
	if zone.decideCalls != 1 {
		t.Fatalf("Decide гостя вызван %d раз, want 1", zone.decideCalls)
	}
}

// В стейте EntryTime — момент заявки, он приходится на бар ПОСЛЕ бара входа: заявка уходит,
// когда бар входа уже закрылся. Ядро привязывается к бару входа по Position.EntryTime («последний
// бар, открывшийся не позже»), поэтому раннер сдвигает время на один 30-минутный бар назад —
// иначе live считал бы бары после входа на один меньше, чем бэктест.
func TestManagedPositionCarriesEntryBarTime(t *testing.T) {
	zone := zoneFake(model.SignalNone, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	fill := time.Date(2026, 3, 9, 15, 0, 7, 0, msk) // бар входа 14:30–15:00, заявка через 7 с
	e.seed(t, statestore.Entry{Ticker: "GAZP", Strategy: "rsi_zone", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: fill})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil).Maybe()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if zone.lastPosition == nil {
		t.Fatal("Decide гостя не получил позицию")
	}
	if want := fill.Add(-30 * time.Minute); !zone.lastPosition.EntryTime.Equal(want) {
		t.Fatalf("Position.EntryTime = %v, want %v (заявка минус один бар)", zone.lastPosition.EntryTime, want)
	}
}

// Занятый тикер: позиция rsi_pullback открыта — гость по нему не входит и даже не спрашивается.
func TestOccupiedTickerIsClosedForTheOtherStrategy(t *testing.T) {
	zone := zoneFake(model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.seed(t, openEntry("")) // запись без Strategy — позиция rsi_pullback
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if zone.decideCalls != 0 {
		t.Fatalf("гостя спросили по занятому тикеру %d раз", zone.decideCalls)
	}
}

// Прод-стейт до выката: запись без поля strategy ведёт rsi_pullback — без алертов о
// неизвестном владельце и без реконструкции.
func TestLegacyEntryWithoutStrategyIsOwnedByPullback(t *testing.T) {
	zone := zoneFake(model.SignalSell, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.seed(t, openEntry(""))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	var sent []string
	e.tg.EXPECT().SendMessage(mock.Anything).RunAndReturn(func(s string) error {
		sent = append(sent, s)
		return nil
	}).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, s := range sent {
		if strings.Contains(s, "неизвестн") {
			t.Fatalf("запись без strategy дала алерт о неизвестном владельце: %s", s)
		}
	}
	if _, ok := e.state(t)["GAZP"]; !ok {
		t.Fatal("позицию закрыл SELL гостя — запись без strategy досталась не rsi_pullback")
	}
	if zone.decideCalls != 0 || zone.rebuildCalls != 0 {
		t.Fatalf("гость тронул чужую позицию: decide=%d rebuild=%d", zone.decideCalls, zone.rebuildCalls)
	}
}

// Позиция без стейта на общем тикере: владельца не определить — алерт, без реконструкции,
// стоп-заявки по инструменту не трогаются.
func TestHeldSharedTickerWithoutStateIsNotReconstructed(t *testing.T) {
	zone := zoneFake(model.SignalNone, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "владелец неизвестен") && strings.Contains(s, "GAZP")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("стейт создан без владельца: %+v", st)
	}
	if zone.rebuildCalls != 0 {
		t.Fatal("гость реконструировал позицию на общем тикере")
	}
	// market без ожиданий: свечи по такому тикеру не запрашиваются; stops без ожиданий на
	// Cancel/Place — mockery провалит тест на любом вызове.
}

// Позиция без стейта на тикере одной стратегии — реконструирует она.
func TestHeldGuestOnlyTickerIsReconstructedByGuest(t *testing.T) {
	zone := zoneFake(model.SignalNone, "AFKS")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP", "AFKS"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("AFKS", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if zone.rebuildCalls != 1 {
		t.Fatalf("rebuild гостя = %d, want 1", zone.rebuildCalls)
	}
	if got := e.state(t)["AFKS"].Strategy; got != "rsi_zone" {
		t.Fatalf("владелец восстановленной позиции = %q, want rsi_zone", got)
	}
}

// Смешанный режим: rsi_pullback боевой, гость бумажный. Бумажный вход — только уведомление:
// стейт не пишется, тикер не занимается, ордеров нет.
func TestPaperGuestEntryDoesNotClaimTickerWhenPullbackIsLive(t *testing.T) {
	zone := zoneFake(model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"},
		func(c *config.RSIPullbackConfig) { c.TradeEnabled = true }, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("бумажный вход гостя занял тикер: %+v", st)
	}
	if !zone.said("GAZP") {
		t.Fatal("бумажный вход гостя не дошёл до уведомления")
	}
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
}

// Тикер ушёл из вселенной владельца, позиция открыта — владелец ведёт её до выхода.
func TestPositionOutsideUniverseIsStillManagedByOwner(t *testing.T) {
	zone := zoneFake(model.SignalSell) // вселенная пуста
	zone.registered = map[string]bool{"AFKS": true}
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.seed(t, statestore.Entry{Ticker: "AFKS", Strategy: "rsi_zone", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: sharedNow.Add(-48 * time.Hour)})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP", "AFKS"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("AFKS", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := e.state(t)["AFKS"]; ok {
		t.Fatal("позиция вне вселенной брошена: SELL владельца не исполнен")
	}
}

// Стейт с неизвестной стратегией: алерт в тему счёта, стейт и биржа не трогаются.
func TestUnknownOwnerInStateAlertsAndKeepsState(t *testing.T) {
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil)
	ghost := openEntry("")
	ghost.Strategy = "ghost"
	e.seed(t, ghost)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "ghost") && strings.Contains(s, "GAZP")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"].Strategy; got != "ghost" {
		t.Fatalf("стейт неизвестного владельца изменён: %q", got)
	}
}

// Гость объявляет о себе при старте в своей теме.
func TestAnnounceIncludesGuests(t *testing.T) {
	zone := zoneFake(model.SignalNone, "AFKS")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Once()

	e.svc.Announce()
	if !zone.said("FAKE") || !zone.said("AFKS") {
		t.Fatalf("гость не объявил о себе: %v", zone.msgs)
	}
}

// Реальная позиция без стейта на тикере бумажного гостя при боевом rsi_pullback: бумажный
// слот не вправе её восстанавливать — он не поставил бы биржевой стоп, а бумажный SELL стёр
// бы стейт, оставив бумаги у брокера. Алерт, без реконструкции, стоп-заявки не трогаются.
func TestPaperGuestDoesNotReconstructRealPosition(t *testing.T) {
	zone := zoneFake(model.SignalSell, "AFKS")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"},
		func(c *config.RSIPullbackConfig) { c.TradeEnabled = true }, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP", "AFKS"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("AFKS", 100)}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "AFKS") && strings.Contains(s, "бумажной стратегии rsi_zone")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if zone.rebuildCalls != 0 || zone.decideCalls != 0 {
		t.Fatalf("бумажный гость тронул реальную позицию: rebuild=%d decide=%d", zone.rebuildCalls, zone.decideCalls)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("стейт создан бумажным гостем: %+v", st)
	}
	// stops без ожиданий на Cancel/PostStopOrder, orders без ожиданий — mockery провалит
	// тест на любом таком вызове.
}

// Бумажный BUY старшей стратегии не занимает бар: rsi_pullback бумажный даёт BUY, боевой
// гость на том же баре входит, rsi_pullback только уведомляет.
func TestPaperPullbackBuyDoesNotBlockLiveGuest(t *testing.T) {
	zone := zoneFake(model.SignalBuy, "GAZP")
	zone.trade = true
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(pullback30m(sharedLastBar, 400), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.orders.EXPECT().PostOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostOrderRequest) bool {
		return in.GetQuantity() == 50 && // 5% от 1 000 000 при цене 100 и лоте 10
			in.GetDirection() == investapi.OrderDirection_ORDER_DIRECTION_BUY
	}), mock.Anything, mock.Anything).Return(filledOrder(50, 100), nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return in.GetQuantity() == 50 &&
			utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano()) == 90
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-z"}, nil).Once()
	var sent []string
	e.tg.EXPECT().SendMessage(mock.Anything).RunAndReturn(func(s string) error {
		sent = append(sent, s)
		return nil
	}).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := e.state(t)["GAZP"]
	if got.Strategy != "rsi_zone" || got.StopOrderID != "so-z" {
		t.Fatalf("боевой гость не вошёл после бумажного BUY rsi_pullback: %+v", got)
	}
	if zone.decideCalls != 1 {
		t.Fatalf("Decide гостя = %d, want 1", zone.decideCalls)
	}
	notified := false
	for _, s := range sent {
		notified = notified || strings.Contains(s, "GAZP")
	}
	if !notified {
		t.Fatal("бумажный вход rsi_pullback не дошёл до уведомления")
	}
}

// Запись принадлежит гостю, а у rsi_pullback на этой ленте BUY: позицию ведёт только
// владелец, rsi_pullback по чужой позиции не входит и владельца не перехватывает.
func TestPullbackBuyDoesNotTouchGuestOwnedPosition(t *testing.T) {
	zone := zoneFake(model.SignalNone, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	seeded := statestore.Entry{Ticker: "GAZP", Strategy: "rsi_zone", EntryPrice: 100,
		EntryATR: 10, MaxFav: 120, Quantity: 100, EntryTime: sharedNow.Add(-48 * time.Hour)}
	e.seed(t, seeded)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(pullback30m(sharedLastBar, 400), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st := e.state(t)
	got, ok := st["GAZP"]
	if !ok || got.Strategy != "rsi_zone" || got.Quantity != 100 || got.EntryPrice != 100 {
		t.Fatalf("позиция гостя изменена чужим BUY: %+v", st)
	}
	if zone.decideCalls != 1 {
		t.Fatalf("Decide владельца = %d, want 1", zone.decideCalls)
	}
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
}

// Смешанный режим, общий тикер: rsi_pullback боевой, гость бумажный. Бумажная стратегия при
// боевой соседке владеть реальной позицией не может, значит кандидат-владелец один —
// rsi_pullback: он восстанавливает стейт и ставит биржевой стоп, как до появления гостя.
func TestHeldSharedTickerMixedModeIsReconstructedByLivePullback(t *testing.T) {
	fill := time.Date(2026, 3, 5, 11, 30, 0, 0, msk)
	zone := zoneFake(model.SignalSell, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"},
		func(c *config.RSIPullbackConfig) { c.TradeEnabled = true }, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(500), nil)
	e.ops.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: true, Date: fill, Price: 100, Quantity: 500}}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return in.GetQuantity() == 50
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-rec"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok || got.Strategy != "rsi_pullback" || got.StopOrderID != "so-rec" {
		t.Fatalf("боевой rsi_pullback не восстановил позицию на общем тикере: %+v", e.state(t))
	}
	if zone.rebuildCalls != 0 || zone.decideCalls != 0 {
		t.Fatalf("бумажный гость тронул реальную позицию: rebuild=%d decide=%d", zone.rebuildCalls, zone.decideCalls)
	}
}

// Обе стратегии боевые, общий тикер, позиция без стейта: владелец неизвестен — алерт, без
// реконструкции, стоп-заявки не трогаются.
func TestHeldSharedTickerBothLiveIsNotReconstructed(t *testing.T) {
	zone := zoneFake(model.SignalNone, "GAZP")
	zone.trade = true
	e := newSharedEnv(t, sharedNow, []string{"GAZP"},
		func(c *config.RSIPullbackConfig) { c.TradeEnabled = true }, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "владелец неизвестен") && strings.Contains(s, "GAZP")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("стейт создан без владельца: %+v", st)
	}
	if zone.rebuildCalls != 0 {
		t.Fatal("гость реконструировал позицию на общем тикере")
	}
}

// Запись принадлежит гостю, которого поставили на паузу (бумажный) при боевом rsi_pullback,
// а реальные бумаги ещё у брокера. Бумажные исполнители «продали» бы вхолостую и стёрли
// запись, оставив бумаги со старым стопом: алерт, стейт и биржа не трогаются, гостя не
// спрашивают.
func TestPaperOwnerOfRealPositionIsNotManaged(t *testing.T) {
	zone := zoneFake(model.SignalSell, "AFKS")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"},
		func(c *config.RSIPullbackConfig) { c.TradeEnabled = true }, zone)
	seeded := statestore.Entry{Ticker: "AFKS", Strategy: "rsi_zone", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: sharedNow.Add(-48 * time.Hour),
		StopOrderID: "so-z", StopPrice: 90, StopReason: "SL"}
	e.seed(t, seeded)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP", "AFKS"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60)) // свободный GAZP rsi_pullback
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("AFKS", 100)}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	var sent []string
	e.tg.EXPECT().SendMessage(mock.Anything).RunAndReturn(func(s string) error {
		sent = append(sent, s)
		return nil
	}).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := e.state(t)["AFKS"]
	if !ok || got.Strategy != "rsi_zone" || got.StopOrderID != "so-z" || got.StopPrice != 90 ||
		got.Quantity != 100 || got.MaxFav != 100 || got.PendingExit != "" || !got.EntryTime.Equal(seeded.EntryTime) {
		t.Fatalf("стейт бумажного владельца изменён: %+v", e.state(t))
	}
	if zone.decideCalls != 0 || zone.rebuildCalls != 0 {
		t.Fatalf("бумажный владелец повёл реальную позицию: decide=%d rebuild=%d", zone.decideCalls, zone.rebuildCalls)
	}
	alerted := false
	for _, s := range sent {
		alerted = alerted || (strings.Contains(s, "AFKS") && strings.Contains(s, "реальная позиция у бумажной стратегии rsi_zone"))
	}
	if !alerted {
		t.Fatalf("нет алерта о реальной позиции у бумажной стратегии: %v", sent)
	}
	// stops без ожиданий на Cancel/PostStopOrder, orders без ожиданий — mockery провалит
	// тест на любом таком вызове.
}

// windowFake — гость с окном входа.
type windowFake struct {
	*fakeStrategy
	open bool
}

func (w *windowFake) EntryPossible(time.Time) bool { return w.open }

// filterFake — гость с вето на вход.
type filterFake struct {
	*fakeStrategy
	reason string
	err    error
	calls  int
}

func (f *filterFake) EntryBlocked(context.Context, string, string, strategy.MarketData) (string, error) {
	f.calls++
	return f.reason, f.err
}

func guestFake(name string, sig model.SignalKind, universe ...string) *fakeStrategy {
	f := zoneFake(sig, universe...)
	f.name = name
	return f
}

// Окно входа закрыто — свечи по свободному тикеру не запрашиваются, гостя не спрашивают.
// market-мок без ожиданий падает на любом GetCandles.
func TestClosedEntryWindowSkipsAssembly(t *testing.T) {
	gap := &windowFake{fakeStrategy: guestFake("gap_fade", model.SignalBuy, "GAZP"), open: false}
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gap.decideCalls != 0 {
		t.Fatalf("гостя спросили %d раз при закрытом окне", gap.decideCalls)
	}
	if len(e.state(t)) != 0 {
		t.Fatal("вход при закрытом окне")
	}
}

// Окно открыто — обычный вход.
func TestOpenEntryWindowEnters(t *testing.T) {
	gap := &windowFake{fakeStrategy: guestFake("gap_fade", model.SignalBuy, "GAZP"), open: true}
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"].Strategy; got != "gap_fade" {
		t.Fatalf("владелец = %q, want gap_fade", got)
	}
}

// Окно входа не мешает сопровождению: позиция гостя ведётся и вне окна.
func TestClosedEntryWindowStillManagesPosition(t *testing.T) {
	gap := &windowFake{fakeStrategy: guestFake("gap_fade", model.SignalSell, "GAZP"), open: false}
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.seed(t, statestore.Entry{Ticker: "GAZP", Strategy: "gap_fade", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: sharedNow.Add(-3 * time.Hour)})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gap.decideCalls != 1 {
		t.Fatalf("Decide вызван %d раз, want 1", gap.decideCalls)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("SELL владельца вне окна входа не закрыл позицию")
	}
}

// Вето с причиной: ордера и стейта нет, причина — уведомлением; следующий гость на этом же
// баре входит.
func TestEntryFilterReasonSkipsAndAsksNextStrategy(t *testing.T) {
	first := &filterFake{fakeStrategy: guestFake("rsi_zone", model.SignalBuy, "GAZP"), reason: "день дивидендной отсечки"}
	second := guestFake("gap_fade", model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, nil, nil, first, second)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if first.calls != 1 {
		t.Fatalf("EntryBlocked вызван %d раз, want 1", first.calls)
	}
	if !first.said("день дивидендной отсечки") {
		t.Fatalf("причина вето не ушла уведомлением: %v", first.msgs)
	}
	if got := e.state(t)["GAZP"].Strategy; got != "gap_fade" {
		t.Fatalf("владелец = %q, want gap_fade (следующий по приоритету)", got)
	}
}

// Ошибка вето: вход не делается (fail-closed), алерт; бар не расходуется.
func TestEntryFilterErrorFailsClosed(t *testing.T) {
	gap := &filterFake{fakeStrategy: guestFake("gap_fade", model.SignalBuy, "GAZP"), err: errors.New("api down")}
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(e.state(t)) != 0 {
		t.Fatal("вход при ошибке вето")
	}
	if !gap.said("api down") {
		t.Fatalf("ошибка вето не ушла алертом: %v", gap.msgs)
	}
}

// Стоп, замороженный сигналом, пишется в стейт при входе и возвращается ядру в Position.
func TestSignalStopLossIsFrozenAndPassedBack(t *testing.T) {
	gap := guestFake("gap_fade", model.SignalBuy, "GAZP")
	gap.sig.StopLoss = 77
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"].StopLoss; got != 77 {
		t.Fatalf("Entry.StopLoss = %v, want 77", got)
	}

	held2 := guestFake("gap_fade", model.SignalNone, "GAZP")
	e2 := newSharedEnv(t, sharedNow, nil, nil, held2)
	e2.seed(t, statestore.Entry{Ticker: "GAZP", Strategy: "gap_fade", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, StopLoss: 77, EntryTime: sharedNow.Add(-time.Hour)})
	e2.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e2.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e2.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e2.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	if err := e2.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if held2.lastPosition == nil || held2.lastPosition.StopLoss != 77 {
		t.Fatalf("Position.StopLoss = %+v, want 77", held2.lastPosition)
	}
}
