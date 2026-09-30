package live

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	candlemocks "tinvest/internal/service/trading_strategy/livecore/candles/mocks"
	execmocks "tinvest/internal/service/trading_strategy/livecore/executor/mocks"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	stopmocks "tinvest/internal/service/trading_strategy/livecore/stoporders/mocks"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	livemocks "tinvest/internal/service/trading_strategy/rsi_pullback/live/mocks"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
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
