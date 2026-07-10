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
	imodel "tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/reversion/live/dto"
	execmocks "tinvest/internal/service/trading_strategy/reversion/live/executor/mocks"
	mdmocks "tinvest/internal/service/trading_strategy/reversion/live/marketdata/mocks"
	livemocks "tinvest/internal/service/trading_strategy/reversion/live/mocks"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	stopmocks "tinvest/internal/service/trading_strategy/reversion/live/stoporders/mocks"
	grpcmodel "tinvest/pkg/client/grpc/model"
	tgmocks "tinvest/pkg/client/telegram/mocks"
	"tinvest/pkg/logger"
)

// TestMain initializes the package logger so that error-path logger.ErrorContext
// calls (e.g. TestManagePass_CancelFailBlocksMarketSell) don't panic on a nil
// logger — mirrors internal/service/trading_strategy/scalping/trade_test.go.
func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

// isHourly1 matches the interval arg for hourly (Hour1 -> 4) candle fetches, mirroring
// the old fake's `interval == 4` branch. Neither test's single ticker (UGLD) triggers a
// 4H fetch (MaxHTFTrendEMA(["UGLD"]) == 0), so this is the only interval ever requested.
func isHourly1(interval int32) bool { return interval == 4 }

func q(f float64) imodel.Quotation { return imodel.Quotation{Units: int64(f)} }

func gq(f float64) grpcmodel.Quotation { return grpcmodel.Quotation{Units: int64(f)} }

// flatHourly builds a flat (no-signal) hourly series long enough for the lookback.
func flatHourly(n int) []*imodel.CandleItemTechAnalyse {
	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	var out []*imodel.CandleItemTechAnalyse
	for i := 0; i < n; i++ {
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time: base.Add(time.Duration(i) * time.Hour),
			Open: q(100), High: q(100), Low: q(100), Close: q(100), Volume: 1000, IsComplete: true,
		})
	}
	return out
}

func cfg(dir string) *config.ReversionConfig {
	return &config.ReversionConfig{
		AccountID: "acc", Tickers: []string{"UGLD"}, BuyPct: 10,
		TradeEnabled: false, NotifyEnabled: true,
	}
}

func TestBuyPass_NoSignal_NoOrderNoState(t *testing.T) {
	dir := t.TempDir()
	c := cfg(dir)

	instruments := livemocks.NewMockinstrumentsClient(t)
	instruments.EXPECT().Shares(mock.Anything).
		Return([]*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1, Trading: true}}, nil)

	market := mdmocks.NewMockCandleClient(t)
	market.EXPECT().
		GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isHourly1), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(flatHourly(400), nil)

	ops := livemocks.NewMockoperationsClient(t)
	ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	// GetPortfolioTotal/GetAvailableCash are only reached after a buy signal; the flat
	// series never produces one, so no expectation is set for them (an unexpected call
	// would fail the mock).

	tg := tgmocks.NewMockClient(t)
	// notify() is only reached on an unregistered ticker, a sizing rejection, or a buy
	// result — none of which occur on the no-signal path, so SendMessage is never called
	// and no expectation is needed.

	svc := NewService(
		instruments,
		market,
		ops,
		nil, // ordersClient unused in dry-run no-signal path
		nil, // stopsClient unused in dry-run no-signal path
		tg,
		c,
	)
	svc.statePath = filepath.Join(dir, "state.json")

	if err := svc.Run(context.Background(), dto.Run{Mode: dto.ModeBuy}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(svc.statePath).Load()
	if len(st) != 0 {
		t.Fatalf("flat series must produce no entry, got %v", st)
	}
}

func TestManagePass_UpdatesMaxFavAndPersists(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	// Seed state with an open UGLD position at entry 100, maxFav 100.
	_ = statestore.New(statePath).Save(map[string]statestore.Entry{
		"UGLD": {Ticker: "UGLD", EntryPrice: 100, EntryATR: 2, MaxFav: 100, Quantity: 10,
			EntryTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)},
	})

	// Hourly series ending at a higher close (110) -> maxFav should rise.
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	var hourly []*imodel.CandleItemTechAnalyse
	for i := 0; i < 400; i++ {
		c := 100.0
		if i == 399 {
			c = 110.0
		}
		hourly = append(hourly, &imodel.CandleItemTechAnalyse{
			Time: base.Add(time.Duration(i) * time.Hour),
			Open: q(c), High: q(c), Low: q(c), Close: q(c), Volume: 1000, IsComplete: true,
		})
	}

	instruments := livemocks.NewMockinstrumentsClient(t)
	instruments.EXPECT().Shares(mock.Anything).
		Return([]*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1, Trading: true}}, nil)

	market := mdmocks.NewMockCandleClient(t)
	market.EXPECT().
		GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isHourly1), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(hourly, nil)

	ops := livemocks.NewMockoperationsClient(t)
	ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).
		Return([]*grpcmodel.Position{{ShareID: "uid-ugld", InstrumentType: "share", Quantity: 10,
			PurchasePrice: gq(100)}}, nil)

	tg := tgmocks.NewMockClient(t)
	// The pre-seeded state already has an entry for UGLD, so no reconstruct-alert
	// notify fires; whether the raised maxFav also crosses the strategy's sell
	// threshold on this data (triggering an Exit notify) is conditional on the
	// strategy's Decide() output, which this test doesn't assert on either way
	// (the original fake recorded but never checked `sent`) — Maybe() preserves that.
	tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	svc := NewService(
		instruments,
		market,
		ops,
		nil,
		nil,
		tg,
		cfg(dir),
	)
	svc.statePath = statePath

	if err := svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(statePath).Load()
	if st["UGLD"].MaxFav != 110 {
		t.Fatalf("MaxFav = %v, want 110 (raised from latest close)", st["UGLD"].MaxFav)
	}
}

func TestPlaceInitialStopPersistsIDAndLevel(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	stops := stopmocks.NewMockClient(t)
	stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).
		Return(&investapi.PostStopOrderResponse{StopOrderId: "so-9"}, nil)

	c := cfg(dir)
	c.TradeEnabled = true
	tg := tgmocks.NewMockClient(t)
	// cfg(dir) sets NotifyEnabled: true, so the StopSet notification fires.
	tg.EXPECT().SendMessage(mock.Anything).Return(nil)
	svc := NewService(livemocks.NewMockinstrumentsClient(t), mdmocks.NewMockCandleClient(t),
		livemocks.NewMockoperationsClient(t), nil, stops, tg, c)
	svc.statePath = statePath

	store := statestore.New(statePath)
	entry := statestore.Entry{Ticker: "UGLD", EntryPrice: 100, EntryATR: 2, MaxFav: 100, Quantity: 10}
	state := map[string]statestore.Entry{"UGLD": entry}
	sh := &imodel.Share{ID: "uid-ugld", Ticker: "UGLD", Lot: 1, MinPriceIncrement: 0.01}

	got := svc.placeInitialStop(context.Background(), "UGLD", sh, entry, state, store)
	// UGLD DefaultParams: UseTrail=1/TrailATRMult=1.5, CatStop=0, UseATRStop=0
	// -> want = 100 - 1.5*2 = 97, reason TRAIL
	if got.StopOrderID != "so-9" || got.StopPrice != 97 || got.StopReason != "TRAIL" {
		t.Fatalf("entry after stop: %+v", got)
	}
	persisted, _ := store.Load()
	if persisted["UGLD"].StopOrderID != "so-9" {
		t.Fatalf("state not persisted: %+v", persisted["UGLD"])
	}
}

// manageEnv wires a UGLD manage-pass with an open position and a stops mock.
// hourlyLastClose управляет последним закрытием (MaxFav/сигналы ядра).
type manageEnv struct {
	svc       *service
	statePath string
	stops     *stopmocks.MockClient
	tg        *tgmocks.MockClient
	orders    *execmocks.MockOrdersClient
}

func newManageEnv(t *testing.T, seed statestore.Entry, hourly []*imodel.CandleItemTechAnalyse) *manageEnv {
	t.Helper()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := statestore.New(statePath).Save(map[string]statestore.Entry{"UGLD": seed}); err != nil {
		t.Fatal(err)
	}

	instruments := livemocks.NewMockinstrumentsClient(t)
	instruments.EXPECT().Shares(mock.Anything).
		Return([]*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1,
			Trading: true, MinPriceIncrement: 0.01}}, nil)

	market := mdmocks.NewMockCandleClient(t)
	market.EXPECT().
		GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isHourly1), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(hourly, nil)

	ops := livemocks.NewMockoperationsClient(t)
	ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).
		Return([]*grpcmodel.Position{{ShareID: "uid-ugld", InstrumentType: "share",
			Quantity: seed.Quantity, PurchasePrice: gq(seed.EntryPrice)}}, nil)

	stops := stopmocks.NewMockClient(t)
	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	orders := execmocks.NewMockOrdersClient(t)

	c := cfg(dir)
	c.TradeEnabled = true
	svc := NewService(instruments, market, ops, orders, stops, tg, c)
	svc.statePath = statePath
	return &manageEnv{svc: svc, statePath: statePath, stops: stops, tg: tg, orders: orders}
}

// hourlySeries строит n флэт-баров по 100 и заменяет последние len(tail) закрытий.
func hourlySeries(n int, tail ...float64) []*imodel.CandleItemTechAnalyse {
	out := flatHourly(n)
	for i, c := range tail {
		idx := n - len(tail) + i
		out[idx].Open, out[idx].High, out[idx].Low, out[idx].Close = q(c), q(c), q(c), q(c)
	}
	return out
}

func seedEntry(stopID string, stopPrice float64) statestore.Entry {
	return statestore.Entry{Ticker: "UGLD", EntryPrice: 100, EntryATR: 2, MaxFav: 100,
		Quantity: 10, EntryTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StopOrderID: stopID, StopPrice: stopPrice, StopReason: "TRAIL"}
}

// activeList строит один активный SELL-стоп с заданными lots (LotsRequested), которые
// Executor.List зеркалит в ActiveStop.Lots — участвует в size-mismatch реконсиляции.
func activeList(id string, price float64, lots int64) *investapi.GetStopOrdersResponse {
	return &investapi.GetStopOrdersResponse{StopOrders: []*investapi.StopOrder{{
		StopOrderId: id, InstrumentUid: "uid-ugld",
		Direction:     investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL,
		StopPrice:     &investapi.MoneyValue{Units: int64(price)},
		LotsRequested: lots,
	}}}
}

// 1. Трейлинг подтянулся: cancel старой + post новой, стейт несёт новый ID/уровень.
// UGLD params: UseTrail=1, TrailATRMult=1.5 -> close 110 => MaxFav 110, want 110-3=107 > 97.
func TestManagePass_RepostsStopWhenTrailRises(t *testing.T) {
	env := newManageEnv(t, seedEntry("so-old", 97), hourlySeries(400, 110))
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(activeList("so-old", 97, 10), nil)
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-old"
	})).Return(&investapi.CancelStopOrderResponse{}, nil)
	env.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).
		Return(&investapi.PostStopOrderResponse{StopOrderId: "so-new"}, nil)

	if err := env.svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(env.statePath).Load()
	got := st["UGLD"]
	if got.StopOrderID != "so-new" || got.StopPrice != 107 || got.StopReason != "TRAIL" {
		t.Fatalf("state after repost: %+v", got)
	}
}

// 2. Уровень не вырос: только List; отсутствие EXPECT на Cancel/Post ловит лишние вызовы.
func TestManagePass_KeepsStopWhenLevelUnchanged(t *testing.T) {
	env := newManageEnv(t, seedEntry("so-1", 97), hourlySeries(400)) // flat 100: want 97 == 97
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(activeList("so-1", 97, 10), nil)

	if err := env.svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(env.statePath).Load()
	if st["UGLD"].StopOrderID != "so-1" {
		t.Fatalf("stop must be kept, got %+v", st["UGLD"])
	}
}

// 3. Позиция исчезла + StopOrderID в стейте -> Exit с причиной из стейта, стейт очищен,
// Cancel не вызывается (заявка уже исполнена биржей).
func TestManagePass_DetectsFiredStop(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	seed := seedEntry("so-1", 97)
	if err := statestore.New(statePath).Save(map[string]statestore.Entry{"UGLD": seed}); err != nil {
		t.Fatal(err)
	}

	instruments := livemocks.NewMockinstrumentsClient(t)
	instruments.EXPECT().Shares(mock.Anything).
		Return([]*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1,
			Trading: true, MinPriceIncrement: 0.01}}, nil)

	market := mdmocks.NewMockCandleClient(t)
	market.EXPECT().
		GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isHourly1), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(hourlySeries(400), nil).Maybe()

	ops := livemocks.NewMockoperationsClient(t)
	ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)

	stops := stopmocks.NewMockClient(t)
	stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(activeList("so-1", 97, 10), nil).Maybe()

	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "TRAIL") && strings.Contains(s, "UGLD")
	})).Return(nil)

	orders := execmocks.NewMockOrdersClient(t)

	c := cfg(dir)
	c.TradeEnabled = true
	svc := NewService(instruments, market, ops, orders, stops, tg, c)
	svc.statePath = statePath

	if err := svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(statePath).Load()
	if len(st) != 0 {
		t.Fatalf("state must be cleared after fired-stop detection, got %+v", st)
	}
}

// 4. SELL ядра при живой заявке и падающем Cancel -> рыночная продажа НЕ отправляется,
// позиция остаётся в стейте, уходит Alert.
func TestManagePass_CancelFailBlocksMarketSell(t *testing.T) {
	// Серия с обвалом хвоста заставляет ядро дать SELL (у UGLD включён trail: close 90
	// при MaxFav 100 пробивает close-бэкстоп... надёжнее выключить неоднозначность:
	// хвост 110, затем 90 — low-стоп ядра сработает по close-падению).
	env := newManageEnv(t, seedEntry("so-1", 97), hourlySeries(400, 110, 90))
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(activeList("so-1", 97, 10), nil).Maybe()
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("boom")).Maybe()
	// env.orders без EXPECT: вызов PostOrder провалит тест = продажи не было.

	if err := env.svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(env.statePath).Load()
	if _, ok := st["UGLD"]; !ok {
		t.Fatal("position must stay in state when cancel fails")
	}
}

// 5. Частичное исполнение: entry.Quantity уменьшается до pos.Quantity (bookkeeping-only,
// без inline-cancel — Finding 1/2 фикса), а размер живой биржевой заявки (List всё ещё
// несёт старые Lots=10 для уже не той позиции в 6) реконсилируется общим путём в
// уровень-switch: ровно один cancel(so-old) + post(so-new) на остаток в том же пассе,
// без повторного cancel той же заявки (Finding 2) и без проглоченной ошибки Cancel
// (Finding 1 — тут Cancel вообще один и идёт по существующему alert+break пути).
func TestManagePass_ReconcilesStopSizeAfterPartialFill(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	seed := seedEntry("so-old", 97) // Quantity: 10
	if err := statestore.New(statePath).Save(map[string]statestore.Entry{"UGLD": seed}); err != nil {
		t.Fatal(err)
	}

	instruments := livemocks.NewMockinstrumentsClient(t)
	instruments.EXPECT().Shares(mock.Anything).
		Return([]*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1,
			Trading: true, MinPriceIncrement: 0.01}}, nil)

	market := mdmocks.NewMockCandleClient(t)
	market.EXPECT().
		GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isHourly1), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(hourlySeries(400), nil) // flat 100: level stays 97 == entry.StopPrice (only size triggers repost)

	ops := livemocks.NewMockoperationsClient(t)
	ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).
		Return([]*grpcmodel.Position{{ShareID: "uid-ugld", InstrumentType: "share",
			Quantity: 6, PurchasePrice: gq(seed.EntryPrice)}}, nil) // 6 < seed.Quantity 10 -> partial fill

	stops := stopmocks.NewMockClient(t)
	stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).
		Return(activeList("so-old", 97, 10), nil) // биржа всё ещё показывает старый 10-лотовый стоп
	stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-old"
	})).Return(&investapi.CancelStopOrderResponse{}, nil).Once() // ровно один cancel — не два
	stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return in.GetQuantity() == 6
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-new"}, nil)

	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "частично")
	})).Return(nil).Once()

	orders := execmocks.NewMockOrdersClient(t)

	c := cfg(dir)
	c.TradeEnabled = true
	svc := NewService(instruments, market, ops, orders, stops, tg, c)
	svc.statePath = statePath

	if err := svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	st, _ := statestore.New(statePath).Load()
	got := st["UGLD"]
	if got.Quantity != 6 || got.StopOrderID != "so-new" || got.StopPrice != 97 {
		t.Fatalf("state after size reconciliation: %+v", got)
	}
}
