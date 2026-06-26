package live

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/reversion/live/dto"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// --- fakes ---

type fakeInstruments struct{ shares []*imodel.Share }

func (f *fakeInstruments) Shares(context.Context) ([]*imodel.Share, error) { return f.shares, nil }

type fakeMarket struct {
	hourly []*imodel.CandleItemTechAnalyse
}

func (f *fakeMarket) GetCandles(_ context.Context, _ *string, interval int32, _, _ *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	if interval == 4 {
		return f.hourly, nil
	}
	return nil, nil
}

type fakeOps struct {
	positions []*grpcmodel.Position
	total     float64
	cash      float64
}

func (f *fakeOps) GetPortfolio(context.Context, string) ([]*grpcmodel.Position, error) {
	return f.positions, nil
}
func (f *fakeOps) GetPortfolioTotal(context.Context, string) (float64, error) { return f.total, nil }
func (f *fakeOps) GetAvailableCash(context.Context, string) (float64, error)  { return f.cash, nil }
func (f *fakeOps) GetInstrumentTrades(context.Context, string, string, time.Time, time.Time) ([]grpcmodel.Trade, error) {
	return nil, nil
}

type fakeTg struct{ sent []string }

func (f *fakeTg) SendMessage(m string) error            { f.sent = append(f.sent, m); return nil }
func (f *fakeTg) SendMessageToChat(int64, string) error { return nil }

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
	svc := NewService(
		&fakeInstruments{shares: []*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1, Trading: true}}},
		&fakeMarket{hourly: flatHourly(400)},
		&fakeOps{total: 100000, cash: 100000},
		nil, // ordersClient unused in dry-run no-signal path
		&fakeTg{},
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

	svc := NewService(
		&fakeInstruments{shares: []*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1, Trading: true}}},
		&fakeMarket{hourly: hourly},
		&fakeOps{positions: []*grpcmodel.Position{{ShareID: "uid-ugld", InstrumentType: "share", Quantity: 10,
			PurchasePrice: gq(100)}}},
		nil,
		&fakeTg{},
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
