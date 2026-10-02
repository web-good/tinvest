package live

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	grpcmodel "tinvest/pkg/client/grpc/model"
	tgmocks "tinvest/pkg/client/telegram/mocks"
	"tinvest/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

type fakeTrades struct{ trades []grpcmodel.Trade }

func (f *fakeTrades) GetInstrumentTrades(_ context.Context, _, _ string, _, _ time.Time) ([]grpcmodel.Trade, error) {
	return f.trades, nil
}

type fakeDaily struct {
	bars []*imodel.CandleItemTechAnalyse
}

func (f *fakeDaily) GetCandles(_ context.Context, _ *string, _ int32,
	_, _ *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	return f.bars, nil
}

func q(f float64) imodel.Quotation { return imodel.Quotation{Units: int64(f)} }

// weekdayBars — будние дневки за 60 дней до полуночи MSK дня входа, истинный диапазон 8.
func weekdayBars(entry time.Time) []*imodel.CandleItemTechAnalyse {
	e := entry.In(msk)
	midnight := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, msk)
	var bars []*imodel.CandleItemTechAnalyse
	for off := 60; off >= 1; off-- {
		d := midnight.AddDate(0, 0, -off)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		bars = append(bars, &imodel.CandleItemTechAnalyse{
			Time: d, Open: q(100), High: q(104), Low: q(96), Close: q(100),
			Volume: 1000, IsComplete: true,
		})
	}
	return bars
}

func newGap(t *testing.T) *Strategy {
	t.Helper()
	return New(&config.GapFadeConfig{Tickers: []string{"SBER"}, BuyPct: 5}, nil, nil)
}

func TestIdentity(t *testing.T) {
	s := New(&config.GapFadeConfig{Tickers: []string{"SBER", "GAZP"}, BuyPct: 7, TradeEnabled: true}, nil, nil)
	if s.Name() != "gap_fade" || s.Label() != "Gap Fade" || s.BuyPct() != 7 || !s.TradeEnabled() ||
		strings.Join(s.Tickers(), ",") != "SBER,GAZP" {
		t.Fatalf("identity = %s/%s/%v/%v/%v", s.Name(), s.Label(), s.BuyPct(), s.TradeEnabled(), s.Tickers())
	}
	var _ adapter.Strategy = s
	var _ adapter.EntryWindow = s
}

// Стоп заморожен сигналом и хранится в стейте; без него стопа нет.
func TestDesiredStopIsFrozenStopLoss(t *testing.T) {
	s := newGap(t)
	if lvl, why := s.DesiredStop("SBER", statestore.Entry{EntryPrice: 90, StopLoss: 78}); lvl != 78 || why != "SL" {
		t.Fatalf("DesiredStop = %v/%q, want 78/SL", lvl, why)
	}
	if lvl, why := s.DesiredStop("SBER", statestore.Entry{EntryPrice: 90}); lvl != 0 || why != "" {
		t.Fatalf("DesiredStop без StopLoss = %v/%q, want 0/\"\"", lvl, why)
	}
}

// Реконструкция: стоп от цены входа (close сигнала неизвестен), цели нет.
func TestReconstructStopFromEntryNoTarget(t *testing.T) {
	s := newGap(t)
	entry := time.Date(2026, 3, 10, 7, 1, 0, 0, msk)
	e, err := s.Reconstruct(context.Background(), adapter.ReconstructInput{
		Trades:  &fakeTrades{trades: []grpcmodel.Trade{{IsBuy: true, Date: entry}}},
		Candles: &fakeDaily{bars: weekdayBars(entry)},
		Ticker:  "SBER", PurchasePrice: 90, Now: entry.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Reconstruct: %v", err)
	}
	if e.EntryATR <= 0 || e.TakeProfit != 0 || e.MaxFav != 90 || !e.EntryTime.Equal(entry) {
		t.Fatalf("entry = %+v", e)
	}
	if want := 90 - 1.0*e.EntryATR; e.StopLoss != want {
		t.Fatalf("StopLoss = %v, want %v", e.StopLoss, want)
	}
}

func TestReconstructWithoutATRFails(t *testing.T) {
	s := newGap(t)
	entry := time.Date(2026, 3, 10, 7, 1, 0, 0, msk)
	_, err := s.Reconstruct(context.Background(), adapter.ReconstructInput{
		Trades:  &fakeTrades{trades: []grpcmodel.Trade{{IsBuy: true, Date: entry}}},
		Candles: &fakeDaily{}, Ticker: "SBER", PurchasePrice: 90, Now: entry,
	})
	if err == nil {
		t.Fatal("нулевой ATR дал реконструкцию")
	}
}

func TestReconstructWithoutBuyFails(t *testing.T) {
	s := newGap(t)
	now := time.Date(2026, 3, 10, 7, 1, 0, 0, msk)
	_, err := s.Reconstruct(context.Background(), adapter.ReconstructInput{
		Trades: &fakeTrades{}, Candles: &fakeDaily{bars: weekdayBars(now)},
		Ticker: "SBER", PurchasePrice: 90, Now: now,
	})
	if err == nil || !strings.Contains(err.Error(), "no BUY fill") {
		t.Fatalf("err = %v, want no BUY fill", err)
	}
}

// Окно входа: будний день до 10:00 MSK.
func TestEntryPossible(t *testing.T) {
	s := newGap(t)
	cases := []struct {
		at   time.Time
		want bool
	}{
		{time.Date(2026, 3, 10, 7, 1, 0, 0, msk), true},   // вторник 07:01
		{time.Date(2026, 3, 10, 9, 59, 0, 0, msk), true},  // до 10:00
		{time.Date(2026, 3, 10, 10, 0, 0, 0, msk), false}, // ровно 10:00 — уже нет
		{time.Date(2026, 3, 10, 19, 1, 0, 0, msk), false},
		{time.Date(2026, 3, 14, 7, 1, 0, 0, msk), false},     // суббота
		{time.Date(2026, 3, 10, 4, 1, 0, 0, time.UTC), true}, // 07:01 MSK в UTC
	}
	for _, c := range cases {
		if got := s.EntryPossible(c.at); got != c.want {
			t.Errorf("EntryPossible(%v) = %v, want %v", c.at, got, c.want)
		}
	}
}

func TestNotifyPrefixAndSwitch(t *testing.T) {
	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage("[Gap Fade] привет").Return(nil).Once()
	New(&config.GapFadeConfig{NotifyEnabled: true}, tg, nil).Notify("привет")

	off := tgmocks.NewMockClient(t) // без ожиданий: любой вызов — падение
	New(&config.GapFadeConfig{NotifyEnabled: false}, off, nil).Notify("тишина")
	New(&config.GapFadeConfig{NotifyEnabled: true}, nil, nil).Notify("nil-клиент не паникует")
}

func TestNotifyDeliveryFailureDoesNotPanic(t *testing.T) {
	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage("[Gap Fade] x").Return(errors.New("down")).Once()
	New(&config.GapFadeConfig{NotifyEnabled: true}, tg, nil).Notify("x")
}
