package live

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/reversion/live/dto"
	mdmocks "tinvest/internal/service/trading_strategy/reversion/live/marketdata/mocks"
	livemocks "tinvest/internal/service/trading_strategy/reversion/live/mocks"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	stopmocks "tinvest/internal/service/trading_strategy/reversion/live/stoporders/mocks"
	grpcmodel "tinvest/pkg/client/grpc/model"
	tgmocks "tinvest/pkg/client/telegram/mocks"
)

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
