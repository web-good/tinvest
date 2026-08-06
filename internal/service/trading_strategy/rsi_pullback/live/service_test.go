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
			Time:   end.Add(-time.Duration(n-1-i) * 30 * time.Minute),
			Open:   qf(c),
			High:   qf(c * 1.001),
			Low:    qf(c * 0.999),
			Close:  qf(c),
			Volume: 1000,

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
// BUY в тесте входа: проверяется гейт, а не отсутствие сигнала.
func TestNoSecondEntryWhenBrokerAlreadyHolds(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(
		[]*grpcmodel.Position{{ShareID: "uid-GAZP", InstrumentType: "share", Quantity: 500,
			PurchasePrice: gq(100)}}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// orders без ожиданий: любой PostOrder провалил бы мок как неожиданный вызов.
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
}

// То же, но позиция известна только из стейта (ордер прошёл, портфель ещё не догнал).
func TestNoSecondEntryWhenStateAlreadyHasEntry(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.seed(t, statestore.Entry{Ticker: "GAZP", EntryPrice: 100, EntryATR: 10, MaxFav: 100,
		TakeProfit: 107, Quantity: 500, EntryTime: lastBar.Add(-time.Hour)})
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
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
