package live

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"tinvest/internal/config"
	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
	candlemocks "tinvest/internal/service/trading_strategy/livecore/candles/mocks"
	execmocks "tinvest/internal/service/trading_strategy/livecore/executor/mocks"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	stopmocks "tinvest/internal/service/trading_strategy/livecore/stoporders/mocks"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/marketdata"
	livemocks "tinvest/internal/service/trading_strategy/rsi_pullback/live/mocks"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
	grpcmodel "tinvest/pkg/client/grpc/model"
	tgmocks "tinvest/pkg/client/telegram/mocks"
	"tinvest/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

var msk = func() *time.Location {
	l, _ := time.LoadLocation("Europe/Moscow")
	return l
}()

func q(v int64) imodel.Quotation { return imodel.Quotation{Units: v} }

// qf строит Quotation с дробной частью (q() несёт только Units).
func qf(v float64) imodel.Quotation {
	u, n := utils.SplitPrice(v)
	return imodel.Quotation{Units: u, Nano: n}
}

func gq(v float64) grpcmodel.Quotation { return grpcmodel.Quotation{Units: int64(v)} }

func isM30(interval int32) bool  { return interval == enum.Minutes30.ToNumberInvestAPI() }
func isDay1(interval int32) bool { return interval == enum.Day1.ToNumberInvestAPI() }

// emptyStopList — ответ GetStopOrders без активных заявок: на бирже по счёту ничего не
// висит.
func emptyStopList() *investapi.GetStopOrdersResponse { return &investapi.GetStopOrdersResponse{} }

// activeList — один активный SELL-стоп по инструменту. Lots попадают в ActiveStop.Lots и
// участвуют в реконсиляции размера заявки.
func activeList(instrumentUID, id string, price float64, lots int64) *investapi.GetStopOrdersResponse {
	return &investapi.GetStopOrdersResponse{StopOrders: []*investapi.StopOrder{{
		StopOrderId: id, InstrumentUid: instrumentUID,
		Direction:     investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL,
		StopPrice:     &investapi.MoneyValue{Units: int64(price)},
		LotsRequested: lots,
	}}}
}

// executedList — ответ GetStopOrders(EXECUTED) с одной исполненной заявкой.
func executedList(id string) *investapi.GetStopOrdersResponse {
	return &investapi.GetStopOrdersResponse{StopOrders: []*investapi.StopOrder{{
		StopOrderId: id,
		Direction:   investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL,
	}}}
}

// statusIs матчит запрос GetStopOrders по статусу — разводит ACTIVE-снапшот пасса и
// EXECUTED-проверку исчезнувшей заявки на одном моке.
func statusIs(st investapi.StopOrderStatusOption) func(*investapi.GetStopOrdersRequest) bool {
	return func(in *investapi.GetStopOrdersRequest) bool { return in.GetStatus() == st }
}

// filledOrder — ответ на рыночный ордер: биржа отчиталась об исполнении lots лотов по price.
func filledOrder(lots int64, price int64) *investapi.PostOrderResponse {
	return &investapi.PostOrderResponse{
		LotsExecuted:       lots,
		ExecutedOrderPrice: &investapi.MoneyValue{Units: price},
	}
}

// cfgFor — вселенная из одного тикера. Тесты входа гоняются на GAZP: его Lookback (160)
// самый маленький из зарегистрированных и UseVolume=0, поэтому серии короче и читаемее.
// Тесты трейла обязаны гоняться на UGLD — у GAZP UseTrail=0, и там трейл вообще
// не считается, то есть тест был бы зелёным при любой реализации.
func cfgFor(ticker string) *config.RSIPullbackConfig {
	return &config.RSIPullbackConfig{
		AccountID: "acc", Tickers: []string{ticker}, BuyPct: 5,
		TradeEnabled: false, NotifyEnabled: true,
	}
}

func shareFor(ticker string) []*imodel.Share {
	return []*imodel.Share{{ID: "uid-" + ticker, Ticker: ticker,
		Lot: 10, Trading: true, MinPriceIncrement: 0.01}}
}

// tape30m строит завершённые 30-минутные бары по закрытиям (oldest-first), последний —
// в end. High/Low раздвигаются на 0.1% вокруг закрытия; тест, которому нужен конкретный
// экстремум последнего бара, переопределяет его после вызова.
func tape30m(end time.Time, closes []float64) []*imodel.CandleItemTechAnalyse {
	n := len(closes)
	out := make([]*imodel.CandleItemTechAnalyse, 0, n)
	for i, c := range closes {
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time:       end.Add(-time.Duration(n-1-i) * 30 * time.Minute),
			Open:       qf(c),
			High:       qf(c * 1.001),
			Low:        qf(c * 0.999),
			Close:      qf(c),
			Volume:     1000,
			IsComplete: true,
		})
	}
	return out
}

// flat30m — ровная лента без сигнала: RSI никуда не пересекает, вход невозможен.
func flat30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	closes := make([]float64, n)
	for i := range closes {
		closes[i] = 100
	}
	return tape30m(end, closes)
}

// Параметры ленты входа. Подъём достаточно крутой, чтобы EMA(10) осталась выше EMA(70)
// даже после провала последнего бара: разрыв EMA растёт со скоростью тренда, а провал
// на L съедает у него 0.154*L (EMA(10) теряет 2/11 от провала, EMA(70) — 2/71).
const (
	pullbackRiseFrom = 60.0
	pullbackRiseTo   = 120.0
	// pullbackClose — закрытие последнего (провального) бара. Ровно 100, чтобы арифметика
	// сайзинга в тестах читалась глазами: 5% от 1 000 000 при лоте 10 — это ровно 50 лотов.
	pullbackClose = 100.0
)

// pullback30m — лента, на последнем баре которой ядро GAZP открывает лонг: длинный
// подъём поднимает EMA(10) над EMA(70), затем резкий провал последнего бара загоняет
// RSI(4) вниз через 25. Провал в один бар (а не в пять, как в ядровом тесте) — потому
// что RSI(4) после серии падений уходит под порог раньше последнего бара, и креста
// ИМЕННО на нём уже не будет. На монотонном подъёме RSISeries отдаёт 50, а не 100
// (при нулевом avgLoss индикатор оставляет rs=1) — креста вниз через 25 это не меняет:
// предыдущий бар всё равно выше порога, текущий (2.2) — ниже.
func pullback30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	closes := make([]float64, n)
	for i := 0; i < n-1; i++ {
		closes[i] = pullbackRiseFrom + (pullbackRiseTo-pullbackRiseFrom)*float64(i)/float64(n-2)
	}
	closes[n-1] = pullbackClose
	bars := tape30m(end, closes)
	last := bars[n-1]
	last.Open, last.High, last.Low = qf(closes[n-2]), qf(closes[n-2]), qf(pullbackClose)
	return bars
}

// dailies — дневные свечи с ненулевым истинным диапазоном: без них dailyATR() = 0
// и enter() выходит на четвёртом гейте, не дойдя до проверяемого. Истинный диапазон
// каждого бара ровно 10, поэтому дневной ATR любой длины равен 10.
func dailies(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	day := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, msk)
	out := make([]*imodel.CandleItemTechAnalyse, 0, n)
	for i := n - 1; i >= 0; i-- {
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time: day.AddDate(0, 0, -i),
			Open: q(100), High: q(105), Low: q(95), Close: q(100),
			Volume: 100000, IsComplete: true,
		})
	}
	return out
}

// env — общая обвязка пасса: моки, конфиг и сервис с подменёнными часами и путём стейта.
// Ожидания каждый тест выставляет сам — вакуумно зелёный тест тут дороже пары повторов.
type env struct {
	svc         *service
	statePath   string
	instruments *livemocks.MockinstrumentsClient
	market      *candlemocks.MockCandleClient
	ops         *livemocks.MockoperationsClient
	orders      *execmocks.MockOrdersClient
	stops       *stopmocks.MockClient
	tg          *tgmocks.MockClient
}

// newEnv собирает сервис на одном тикере с часами, прибитыми к now: гейт свежести бара
// (maxBarAge) меряет возраст от часов сервиса, а фикстуры датированы фиксированной датой.
func newEnv(t *testing.T, ticker string, now time.Time, tweak ...func(*config.RSIPullbackConfig)) *env {
	t.Helper()
	c := cfgFor(ticker)
	for _, f := range tweak {
		f(c)
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
	e.svc = NewService(e.instruments, e.market, e.ops, e.orders, e.stops, e.tg, c)
	e.svc.statePath = e.statePath
	e.svc.now = func() time.Time { return now }
	return e
}

// expectCandles навешивает обе ленты на market-мок (30m и дневные).
func (e *env) expectCandles(m30, day []*imodel.CandleItemTechAnalyse) {
	e.market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isM30),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(m30, nil)
	e.market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(day, nil)
}

func (e *env) state(t *testing.T) map[string]statestore.Entry {
	t.Helper()
	st, err := statestore.New(e.statePath).Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	return st
}

func (e *env) seed(t *testing.T, entry statestore.Entry) {
	t.Helper()
	if err := statestore.New(e.statePath).Save(map[string]statestore.Entry{entry.Ticker: entry}); err != nil {
		t.Fatalf("seed state: %v", err)
	}
}

// Вход: BUY-сигнал обязан привести к ордеру, записи стейта с замороженной целью и
// немедленной постановке защитного стопа. Позиция не должна прожить без биржевой
// защиты ни одного тика: следующая проверка только через полчаса, а стоп внутрибарный.
func TestBuySignalPlacesOrderStateAndProtectiveStop(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now)
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	entry, ok := e.state(t)["GAZP"]
	if !ok {
		t.Fatal("после BUY-сигнала в стейте нет записи по GAZP")
	}
	if entry.EntryATR <= 0 {
		t.Fatalf("EntryATR = %v, want > 0 (дневной ATR замораживается на входе)", entry.EntryATR)
	}
	if entry.TakeProfit <= entry.EntryPrice {
		t.Fatalf("TakeProfit = %v при входе %v — цель обязана быть заморожена выше входа",
			entry.TakeProfit, entry.EntryPrice)
	}
	if entry.StopReason == "" {
		t.Fatal("защитный стоп не выставлен сразу после входа")
	}
	if entry.StopPrice <= 0 || entry.StopPrice >= entry.EntryPrice {
		t.Fatalf("StopPrice = %v при входе %v, want 0 < stop < entry", entry.StopPrice, entry.EntryPrice)
	}
}

// Боевой вход (TradeEnabled=true): в dry-run и executor, и stoporders коротят до gRPC, то
// есть главная гарантия задачи — «после реального фила сразу уходит реальная стоп-заявка» —
// проверяется только здесь. Заодно пинуется цепочка «ответ брокера → стейт → размер
// заявки»: фил пришёл на 40 лотов по 101 (а не запрошенные 50 по сигнальным 100), значит
// в стейте обязаны быть 400 штук и вход 101, а на бирже — стоп на 40 лотов по 101−0.5×10=96.
func TestLiveBuyPlacesRealOrderAndStopFromTheFill(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.orders.EXPECT().PostOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostOrderRequest) bool {
		return in.GetQuantity() == 50 && // 5% от 1 000 000 при цене 100 и лоте 10
			in.GetDirection() == investapi.OrderDirection_ORDER_DIRECTION_BUY
	})).Return(filledOrder(40, 101), nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return in.GetQuantity() == 40 &&
			utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano()) == 96
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-1"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok {
		t.Fatal("после боевого входа в стейте нет записи по GAZP")
	}
	if got.EntryPrice != 101 || got.MaxFav != 101 || got.Quantity != 400 {
		t.Fatalf("стейт после фила: %+v, want вход 101, MaxFav 101, 400 штук", got)
	}
	if got.StopOrderID != "so-1" || got.StopPrice != 96 || got.StopReason != "SL" {
		t.Fatalf("биржевая защита после входа: %+v, want so-1 на 96 по причине SL", got)
	}
}

// Сайзинг идёт от BuyPct конфига, а не от захардкоженного числа: тот же прогон с
// BuyPct=5 против BuyPct=50 обязан дать разное Quantity в стейте (при цене 100 и лоте 10
// это 500 против 5000 штук при счёте 1 000 000).
func TestLotsAreSizedFromConfiguredBuyPct(t *testing.T) {
	buyWith := func(t *testing.T, pct float64) statestore.Entry {
		t.Helper()
		now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
		lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

		e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.BuyPct = pct })
		e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
		e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
		e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
		e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
		e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
		e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

		if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		entry, ok := e.state(t)["GAZP"]
		if !ok {
			t.Fatal("после BUY-сигнала в стейте нет записи по GAZP")
		}
		return entry
	}

	small := buyWith(t, 5)
	big := buyWith(t, 50)
	if small.EntryPrice != pullbackClose {
		t.Fatalf("EntryPrice = %v, want %v (фикстура закрывает последний бар ровно на нём)",
			small.EntryPrice, pullbackClose)
	}
	if small.Quantity != 500 {
		t.Fatalf("Quantity при BuyPct=5 = %d, want 500 (5%% от 1 000 000 при цене 100 и лоте 10)",
			small.Quantity)
	}
	if big.Quantity != 5000 {
		t.Fatalf("Quantity при BuyPct=50 = %d, want 5000", big.Quantity)
	}
}

// Уже открытая позиция у брокера не должна порождать второй вход: GetPortfolio отдаёт
// позицию по uid-GAZP, PostOrder не должен быть вызван ни разу. Лента — та же, что даёт
// BUY в тесте входа: проверяется гейт, а не отсутствие сигнала. Стейт засеян с целью 200 —
// заведомо выше high этой ленты: иначе позиция вышла бы по TP и PostOrder ушёл бы на
// продажу, то есть тест перестал бы проверять именно повторный вход.
func TestNoSecondEntryWhenBrokerAlreadyHolds(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	held := openEntry("so-1")
	held.TakeProfit = 200
	e.seed(t, held)
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(
		[]*grpcmodel.Position{{ShareID: "uid-GAZP", InstrumentType: "share", Quantity: 100,
			PurchasePrice: gq(100)}}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).
		Return(activeList("uid-GAZP", "so-1", 95, 10), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// orders без ожиданий: любой PostOrder провалил бы мок как неожиданный вызов.
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
}

// Позиция у брокера есть, локального стейта нет (переезд контейнера, потерянный том):
// раннер обязан восстановить вход по API и НЕМЕДЛЕННО вернуть биржевую защиту. Проверяются
// именно те величины, без которых ядро не работает: дневной ATR входа (10 — истинный
// диапазон каждой дневной свечи фикстуры), замороженная цель (100 + 0.7×10) и количество
// от брокера. Стоп 95 = 100 − 0.5×10: восстановленный уровень обязан совпасть с тем, что
// стратегия задала бы на входе, иначе реконструкция «защищает» позицию не там.
func TestHeldPositionWithoutStateIsReconstructed(t *testing.T) {
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
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return in.GetQuantity() == 50 && // 500 штук при лоте 10
			utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano()) == 95
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-rec"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok {
		t.Fatal("стейт по GAZP не восстановлен")
	}
	if !got.EntryTime.Equal(fill) {
		t.Fatalf("EntryTime = %v, want %v (последняя BUY-сделка у брокера)", got.EntryTime, fill)
	}
	if got.EntryPrice != 100 || got.Quantity != 500 {
		t.Fatalf("EntryPrice/Quantity = %v/%d, want 100/500 (средняя цена и объём от брокера)",
			got.EntryPrice, got.Quantity)
	}
	if got.EntryATR < 9.5 || got.EntryATR > 10.5 {
		t.Fatalf("EntryATR = %v, want ~10 (дневной ATR фикстуры)", got.EntryATR)
	}
	if got.TakeProfit < 106.5 || got.TakeProfit > 107.5 {
		t.Fatalf("TakeProfit = %v, want ~107 (вход + 0.7×ATR)", got.TakeProfit)
	}
	if got.StopOrderID != "so-rec" || got.StopPrice != 95 || got.StopReason != "SL" {
		t.Fatalf("стоп после реконструкции = %q/%v/%q, want so-rec/95/SL",
			got.StopOrderID, got.StopPrice, got.StopReason)
	}
}

// Реконструкция не удалась (у брокера нет ни одной BUY-сделки по инструменту) — позицию
// НЕ трогаем: ни ордера, ни стоп-заявки по угаданным величинам, стейт остаётся пустым.
// Стоп, поставленный от выдуманного ATR, продал бы позицию не там, где задумала стратегия.
func TestHeldPositionWithFailedReconstructIsLeftAlone(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(flat30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(
		[]*grpcmodel.Position{{ShareID: "uid-GAZP", InstrumentType: "share", Quantity: 500,
			PurchasePrice: gq(100)}}, nil)
	e.ops.EXPECT().GetInstrumentTrades(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil) // сделок нет
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "реконструкция не удалась")
	})).Return(nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Ни ордеров, ни стоп-заявок: моки без ожиданий провалили бы любой вызов.
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
	e.stops.AssertNotCalled(t, "PostStopOrder", mock.Anything, mock.Anything)
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("стейт = %+v, want пустой (реконструировать нечего)", st)
	}
}

// Свежий вход, который портфель брокера ещё не показывает, обязан пережить пасс целиком:
// расчёты отстают от исполненного ордера, и трактовать это как «продали вне раннера»
// значит снять ЖИВОЙ защитный стоп и стереть стейт — после чего следующий пасс вошёл бы
// в позицию второй раз. Заодно пинует гейт повторного входа: лента та же, что даёт BUY
// в тесте входа, а PostOrder не зовётся ни разу.
func TestFreshEntrySurvivesLaggingPortfolio(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	fresh := openEntry("so-1")
	fresh.EntryTime = lastBar // вход был на предыдущем баре, полчаса назад
	e.seed(t, fresh)
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil) // портфель отстаёт
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", 95, 10), nil)
	// stops без ожидания на CancelStopOrder: снятие живой защиты провалит мок.
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "портфель")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok {
		t.Fatal("свежая запись стёрта из стейта: отставание портфеля принято за продажу вне раннера")
	}
	if got.StopOrderID != "so-1" {
		t.Fatalf("StopOrderID = %q, want so-1 (защитный стоп обязан остаться на бирже)", got.StopOrderID)
	}
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
}

// Отклонённый ордер не оставляет записи в стейте: следующий тик повторит попытку.
func TestRejectedBuyLeavesStateUntouched(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("rejected")).Once()
	// Строгое (не Maybe()) ожидание: тест не должен проходить вакуумно, если сигнала не было.
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "ордер на покупку отклонён")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("после отклонённого ордера стейт обязан остаться пустым, got %+v", st)
	}
}

// Тикер из конфига, которого нет в реестре, даёт алерт и пропускается, а не паникует.
// Реестр регистрозависим, поэтому "ugld" из env — такой же неизвестный тикер, как и
// заведомая опечатка: свечи по нему не запрашиваются вовсе.
func TestUnknownTickerAlertsAndSkips(t *testing.T) {
	for _, ticker := range []string{"ugld", "НЕТ-ТАКОГО"} {
		t.Run(ticker, func(t *testing.T) {
			now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
			e := newEnv(t, ticker, now)
			e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
			e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
			// market без ожиданий: любой GetCandles провалит тест — решений по тикеру нет.
			e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
				return strings.Contains(s, "не зарегистрирован") && strings.Contains(s, ticker)
			})).Return(nil).Once()

			if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			if st := e.state(t); len(st) != 0 {
				t.Fatalf("незарегистрированный тикер не должен попадать в стейт, got %+v", st)
			}
		})
	}
}

// Протухшие данные: последний завершённый бар старше maxBarAge — решений не принимаем.
// Лента заканчивается за 3 часа до now, и это та же лента, что в тесте входа даёт BUY,
// то есть пропуск — заслуга гейта свежести, а не отсутствия сигнала.
func TestStaleBarSkipsTicker(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 9, 0, 0, 0, msk)

	e := newEnv(t, "GAZP", now)
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("по протухшему бару решений быть не должно, got %+v", st)
	}
}

// assembleFromFixture гоняет фикстуры через настоящий marketdata.Assemble — тот же путь,
// которым их видит пасс.
func assembleFromFixture(t *testing.T, ticker string, m30, day []*imodel.CandleItemTechAnalyse) strategy.MarketData {
	t.Helper()
	market := candlemocks.NewMockCandleClient(t)
	market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isM30),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(m30, nil)
	market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(day, nil)

	st, ok := StrategyFor(ticker)
	if !ok {
		t.Fatalf("тикер %s не зарегистрирован", ticker)
	}
	md, err := marketdata.Assemble(context.Background(), market, "uid-"+ticker, st.Lookback(),
		m30[len(m30)-1].Time.Add(time.Minute))
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return md
}

// Фикстура обязана доходить до самого входа, а не останавливаться на раннем гейте.
func TestPullbackFixtureActuallyProducesABuySignal(t *testing.T) {
	st, _ := StrategyFor("GAZP")
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	md := assembleFromFixture(t, "GAZP", pullback30m(lastBar, 400), dailies(lastBar, 60))
	if sig := st.Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("фикстура не даёт BUY: Kind=%v\n%s", sig.Kind, st.Explain(md))
	}
}

// --- сопровождение позиции: выходы и биржевая стоп-заявка ---

// heldGAZP — открытая позиция GAZP у брокера: 100 штук по 100 (10 лотов по 10 штук).
func heldGAZP(qty int64) []*grpcmodel.Position {
	return []*grpcmodel.Position{{ShareID: "uid-GAZP", InstrumentType: "share",
		Quantity: qty, PurchasePrice: gq(100)}}
}

// openEntry — стейт открытой позиции GAZP: вход 100, дневной ATR 10 (уровень стопа
// 100 − 0.5×10 = 95), цель 107 (+0.7 ATR) — ровно те величины, которыми ядро мерит выходы.
// Пустой stopID означает, что биржевой заявки нет вовсе, поэтому и штампов от неё
// (уровень/причина) в стейте тоже нет.
func openEntry(stopID string) statestore.Entry {
	e := statestore.Entry{Ticker: "GAZP", EntryPrice: 100, EntryATR: 10, MaxFav: 100,
		TakeProfit: 107, Quantity: 100, EntryTime: time.Date(2026, 3, 9, 11, 30, 0, 0, msk)}
	if stopID != "" {
		e.StopOrderID, e.StopPrice, e.StopReason = stopID, 95, "SL"
	}
	return e
}

// tpHit30m — ровная лента, у последнего бара которой high достаёт цель из стейта (108 ≥ 107),
// а low остаётся выше уровня стопа (104 > 95): сработать может только TP.
func tpHit30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	closes := make([]float64, n)
	for i := range closes {
		closes[i] = 100
	}
	closes[n-1] = 105
	bars := tape30m(end, closes)
	last := bars[n-1]
	last.Open, last.High, last.Low = qf(100), qf(108), qf(104)
	return bars
}

// rsiExit30m — лента, на последнем баре которой ядро GAZP закрывает лонг по RSI. Пологий
// спад с мелкой пилой держит RSI(4) в середине диапазона (пила обязательна: на монотонной
// серии Уайлдер оставляет один из средних ровно нулём, RSI вырождается в 0 или 50, и
// crossedUp отбрасывает такой бар guard'ом series[i-1] > 0), затем один мощный up-бар
// перебрасывает RSI вверх через 70. Стоп (95) ниже low этого бара, цель (107) выше его
// high — сработать может только выход по RSI.
func rsiExit30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	closes := make([]float64, n)
	for i := 0; i < n-1; i++ {
		closes[i] = 106 - 5*float64(i)/float64(n-2)
		if i%2 == 1 {
			closes[i] += 0.2 // пила поверх дрейфа: и подъёмы, и падения ненулевые
		}
	}
	closes[n-1] = 106.5
	bars := tape30m(end, closes)
	last := bars[n-1]
	last.Open, last.High, last.Low = qf(101), qf(106.6), qf(101)
	return bars
}

// risingWithDeepLow30m — лента, где последний бар закрывается ВЫШЕ прежнего максимума
// (закрытие 112 при MaxFav 100), но его low проваливается между двумя кандидатами на
// уровень трейла: ниже уровня, посчитанного от нового закрытия, но выше уровня,
// посчитанного от прежнего максимума. Именно этот зазор разводит честное решение и
// завышенное; без него тест зелёный при обеих реализациях. n обязано быть >= 403:
// столько баров требует Lookback UGLD, и на более коротком окне Assemble вернёт ошибку.
func risingWithDeepLow30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	closes := make([]float64, n)
	for i := range closes {
		closes[i] = 100 + 12*float64(i)/float64(n-1)
	}
	bars := tape30m(end, closes)
	last := bars[n-1]
	last.Open, last.High, last.Low, last.Close = qf(112), qf(112), qf(100), qf(112)
	return bars
}

// RSI-выход: перед рыночной продажей биржевой стоп обязан быть снят, иначе он позже
// продаст новую позицию по этому тикеру.
func TestRSIExitCancelsStopBeforeSelling(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, openEntry("so-1"))
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(rsiExit30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", 95, 10), nil)

	var seq []string
	e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-1"
	})).Run(func(_ context.Context, _ *investapi.CancelStopOrderRequest, _ ...grpc.CallOption) {
		seq = append(seq, "cancel")
	}).Return(&investapi.CancelStopOrderResponse{}, nil).Once()
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything).
		Run(func(_ context.Context, _ *investapi.PostOrderRequest, _ ...grpc.CallOption) {
			seq = append(seq, "sell")
		}).Return(filledOrder(10, 106), nil).Once()
	// Строгое ожидание: тест не должен проходить вакуумно, если ядро перестанет давать SELL.
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "Выход") && strings.Contains(s, "RSI")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(seq) != 2 || seq[0] != "cancel" || seq[1] != "sell" {
		t.Fatalf("порядок вызовов %v, want [cancel sell]", seq)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("после выхода стейт обязан очиститься, got %+v", st)
	}
}

// TP по close-модели: бар, чей high достал цели из стейта, закрывает позицию по рынку.
func TestTakeProfitExitFiresOnBarHigh(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, openEntry("so-1"))
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(tpHit30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", 95, 10), nil)
	e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).
		Return(&investapi.CancelStopOrderResponse{}, nil).Once()
	e.orders.EXPECT().PostOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostOrderRequest) bool {
		return in.GetQuantity() == 10 &&
			in.GetDirection() == investapi.OrderDirection_ORDER_DIRECTION_SELL
	})).Return(filledOrder(10, 107), nil).Once()
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "Выход") && strings.Contains(s, "TP")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("после TP стейт обязан очиститься, got %+v", st)
	}
}

// Отклонённая продажа обязана вернуть биржевую защиту: стоп уже снят, позиция голая.
func TestRejectedSellRestoresTheStop(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, openEntry("so-1"))
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(tpHit30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", 95, 10), nil)
	e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).
		Return(&investapi.CancelStopOrderResponse{}, nil).Once()
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("rejected")).Once()
	// Защита возвращается на прежнем уровне и в прежнем объёме.
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return in.GetQuantity() == 10 &&
			utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano()) == 95
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-2"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok {
		t.Fatal("отклонённая продажа не должна стирать позицию из стейта")
	}
	if got.StopOrderID != "so-2" || got.StopPrice != 95 || got.StopReason != "SL" {
		t.Fatalf("стоп обязан вернуться после отклонённой продажи, got %+v", got)
	}
}

// Трейл: уровень НОВОЙ заявки считается от обновлённого maxFav (она будет работать на
// следующем баре), а решение о срабатывании — от PrevMaxFavorablePrice (заявка,
// работавшая на ЭТОМ баре, выставлена после закрытия предыдущего и не могла знать его
// закрытия, а проверяется против low этого же бара). MaxFav >= PrevMaxFav всегда, поэтому
// чтение первого вместо второго завышает результат систематически и в обе стороны:
// уровень срабатывает не реже и заливается не хуже.
func TestTrailDecisionUsesPrevMaxFavorable(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	// UGLD, а не GAZP: у GAZP UseTrail=0, трейл не считается вовсе и тест был бы зелёным
	// при любой реализации. У UGLD UseTrail=1, TrailDailyATR=0.5 — как только максимум
	// закрытий уходит выше входа, связывает именно трейл, а не фиксированный SL.
	// Позиция открыта по 100, максимум закрытий пока 100, дневной ATR 10. Уровень от
	// PrevMaxFav=100 лежит НИЖЕ low последнего бара, а от MaxFav=112 (закрытие этого
	// бара) — уже ВЫШЕ. Стоп обязан не сработать.
	e := newEnv(t, "UGLD", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, statestore.Entry{Ticker: "UGLD", EntryPrice: 100, EntryATR: 10, MaxFav: 100,
		TakeProfit: 200, Quantity: 100})
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("UGLD"), nil)
	e.expectCandles(risingWithDeepLow30m(lastBar, 500), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(
		[]*grpcmodel.Position{{ShareID: "uid-UGLD", InstrumentType: "share", Quantity: 100,
			PurchasePrice: gq(100)}}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(emptyStopList(), nil)
	// Новая заявка — от ОБНОВЛЁННОГО максимума: 112 − 0.5×10 = 107.
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano()) == 107
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-new"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Продажи быть не должно: от честного (Prev) уровня стоп не задет.
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
	st := e.state(t)
	if _, ok := st["UGLD"]; !ok {
		t.Fatal("позиция закрыта: уровень посчитан от MaxFav вместо PrevMaxFavorablePrice")
	}
	// А вот НОВАЯ заявка обязана подтянуться к обновлённому максимуму.
	if st["UGLD"].MaxFav <= 100 {
		t.Fatalf("MaxFav = %v, want > 100 (максимум закрытий обязан подняться)", st["UGLD"].MaxFav)
	}
	if st["UGLD"].StopPrice != 107 || st["UGLD"].StopReason != "TRAIL" {
		t.Fatalf("новая заявка: %+v, want уровень 107 по причине TRAIL", st["UGLD"])
	}
}

// Перенос трейла — самая частая операция раннера в проде (у UGLD UseTrail=1, стоп
// подтягивается каждые полчаса): старая заявка снимается, новая ставится на подтянутый
// уровень. Порядок обязателен — постановка второй заявки до снятия первой оставила бы на
// бирже два живых SELL-стопа, и второй продал бы уже не имеющиеся бумаги.
func TestTrailRepostCancelsBeforePlacing(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "UGLD", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, statestore.Entry{Ticker: "UGLD", EntryPrice: 100, EntryATR: 10, MaxFav: 100,
		TakeProfit: 200, Quantity: 100, EntryTime: time.Date(2026, 3, 9, 11, 30, 0, 0, msk),
		StopOrderID: "so-1", StopPrice: 95, StopReason: "SL"})
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("UGLD"), nil)
	e.expectCandles(risingWithDeepLow30m(lastBar, 500), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(
		[]*grpcmodel.Position{{ShareID: "uid-UGLD", InstrumentType: "share", Quantity: 100,
			PurchasePrice: gq(100)}}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-UGLD", "so-1", 95, 10), nil)

	var seq []string
	e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-1"
	})).Run(func(_ context.Context, _ *investapi.CancelStopOrderRequest, _ ...grpc.CallOption) {
		seq = append(seq, "cancel")
	}).Return(&investapi.CancelStopOrderResponse{}, nil).Once()
	// Новый уровень — трейл от обновлённого максимума: 112 − 0.5×10 = 107 > прежних 95.
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano()) == 107
	})).Run(func(_ context.Context, _ *investapi.PostStopOrderRequest, _ ...grpc.CallOption) {
		seq = append(seq, "post")
	}).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-2"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(seq) != 2 || seq[0] != "cancel" || seq[1] != "post" {
		t.Fatalf("порядок вызовов %v, want [cancel post]", seq)
	}
	got := e.state(t)["UGLD"]
	if got.StopOrderID != "so-2" || got.StopPrice != 107 || got.StopReason != "TRAIL" {
		t.Fatalf("стейт после переноса трейла: %+v, want so-2 на 107 по причине TRAIL", got)
	}
}

// Размер живой заявки важнее неизменного уровня: после частичного исполнения на бирже
// висит SELL-стоп на 10 лотов при позиции в 6, и он продал бы бумаг больше, чем есть.
// Уровень при этом тот же (95), то есть переставить заявку заставляет ровно size-mismatch.
func TestOversizedStopIsRepostedAtTheSameLevel(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, openEntry("so-old")) // Quantity 100 = 10 лотов, уровень 95
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(flat30m(lastBar, 400), dailies(now, 60)) // ровная лента: уровень остаётся 95
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(60), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-old", 95, 10), nil) // биржа всё ещё держит 10 лотов
	e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-old"
	})).Return(&investapi.CancelStopOrderResponse{}, nil).Once() // ровно один cancel, не два
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return in.GetQuantity() == 6 &&
			utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano()) == 95
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-new"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := e.state(t)["GAZP"]
	if got.Quantity != 60 || got.StopOrderID != "so-new" || got.StopPrice != 95 {
		t.Fatalf("стейт после реконсиляции размера заявки: %+v", got)
	}
}

// Заявка исчезла из ACTIVE и найдена в EXECUTED — это сработавший стоп: уведомление
// о выходе и чистка стейта, без репоста.
func TestFiredStopClosesTheTradeInState(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, openEntry("so-1"))
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(flat30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil) // позиции больше нет
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(emptyStopList(), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED))).
		Return(executedList("so-1"), nil)
	// Строгое ожидание: именно Выход по причине из стейта, а не алерт про внешнюю отмену.
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "Выход") && strings.Contains(s, "SL")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("после сработавшего стопа стейт обязан очиститься, got %+v", st)
	}
}

// stopHit30m — ровная лента, у последнего бара которой low проваливает уровень стопа из
// стейта (94 < 95), а high не достаёт цели (100 < 107): единственный возможный выход —
// стоповый. Именно так выглядит бар, на котором сработала биржевая стоп-заявка.
func stopHit30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	closes := make([]float64, n)
	for i := range closes {
		closes[i] = 100
	}
	bars := tape30m(end, closes)
	last := bars[n-1]
	last.Open, last.High, last.Low, last.Close = qf(100), qf(100), qf(94), qf(96)
	return bars
}

// Биржевой стоп сработал внутри бара, а портфель брокера ЕЩЁ показывает позицию (расчёты
// отстают от исполнения — на этом же лаге построен freshEntryGrace). Уровень заявки и
// уровень, по которому решает ядро, совпадают по построению: заявка выставлена от того же
// prevMaxFav, а округление вниз делает биржевой уровень не выше ядрового. Значит всякий
// раз, когда срабатывает стоп, Decide на этом же баре тоже возвращает SELL — и если
// исполнение заявки разбирается ПОСЛЕ Decide, раннер идёт продавать рыночным ордером
// бумаги, которых уже нет (отказ брокера или шорт на марже). Здесь PostOrder не ожидается
// ни разу: любой вызов провалит мок.
func TestFiredStopIsSettledBeforeDecideWhenPortfolioLags(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, openEntry("so-1")) // стоп so-1 на 95, цель 107
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(stopHit30m(lastBar, 400), dailies(now, 60))
	// Портфель ОТСТАЁТ: позиция всё ещё числится, хотя стоп уже исполнен.
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(
		[]*grpcmodel.Position{{ShareID: "uid-GAZP", InstrumentType: "share", Quantity: 100,
			PurchasePrice: gq(100)}}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(emptyStopList(), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED))).
		Return(executedList("so-1"), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "Выход") && strings.Contains(s, "SL")
	})).Return(nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Ни рыночной продажи, ни отмены уже исполненной заявки.
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
	e.stops.AssertNotCalled(t, "CancelStopOrder", mock.Anything, mock.Anything)
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("после сработавшего стопа стейт обязан очиститься, got %+v", st)
	}
}

// Заявка исчезла, а EXECUTED недоступен — репост вслепую запрещён: он продал бы уже
// проданную позицию при касании уровня.
func TestVanishedStopWithUnavailableExecutedDefersRepost(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, openEntry("so-1"))
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(flat30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(emptyStopList(), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED))).
		Return(nil, fmt.Errorf("boom"))
	// stops без ожидания на PostStopOrder: любой репост провалит мок.
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "репост отложен")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, ok := e.state(t)["GAZP"]
	if !ok || got.StopOrderID != "so-1" {
		t.Fatalf("стейт обязан пережить недоступный EXECUTED без изменений, got %+v", got)
	}
}

// Позиция усохла (частичный стоп или ручная продажа) — размер в стейте
// реконсилируется, иначе следующая заявка окажется переразмеренной.
func TestShrunkPositionReconcilesQuantity(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, openEntry("")) // заявки нет: её нужно поставить уже на усохший размер
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(flat30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(60), nil) // было 100
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(emptyStopList(), nil)
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return in.GetQuantity() == 6 // сайзинг от реконсилированного количества, не от 10 лотов
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-new"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "частично")
	})).Return(nil).Once()
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "стоп-заявка SL")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := e.state(t)["GAZP"]
	if got.Quantity != 60 || got.StopOrderID != "so-new" {
		t.Fatalf("стейт после реконсиляции размера: %+v", got)
	}
}
