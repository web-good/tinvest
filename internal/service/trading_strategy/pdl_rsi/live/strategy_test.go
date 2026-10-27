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
	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
	"tinvest/internal/service/trading_strategy/pdl_rsi/strategy/svcb"
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

type fakeTrades struct {
	trades []grpcmodel.Trade
}

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

// weekdayBars — будние дневки за 60 дней до полуночи MSK дня входа, с ненулевым диапазоном.
func weekdayBars(entry time.Time) []*imodel.CandleItemTechAnalyse {
	e := entry.In(msk)
	midnight := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, msk)
	var bars []*imodel.CandleItemTechAnalyse
	for off := 60; off >= 1; off-- {
		d := midnight.AddDate(0, 0, -off)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		c := 100 + float64(off%5)
		bars = append(bars, &imodel.CandleItemTechAnalyse{
			Time: d, Open: q(c), High: q(c + 4), Low: q(c - 4), Close: q(c),
			Volume: 1000, IsComplete: true,
		})
	}
	return bars
}

func TestDesiredStopUsesStopLevel(t *testing.T) {
	s := New(&config.PDLRSIConfig{Tickers: []string{"SVCB"}}, nil)
	e := statestore.Entry{EntryPrice: 100, EntryATR: 5, MaxFav: 150}
	level, reason := s.DesiredStop("SVCB", e)
	if want := core.StopLevel(svcb.Params(), 100, 5); level != want || reason != "SL" || want <= 0 {
		t.Fatalf("DesiredStop = %v/%q, want %v/SL", level, reason, want)
	}
	// MaxFav не влияет на уровень
	e.MaxFav = 100
	if l2, _ := s.DesiredStop("SVCB", e); l2 != level {
		t.Fatalf("уровень стопа зависит от MaxFav: %v vs %v", l2, level)
	}
}

func TestDesiredStopWithoutATROrTicker(t *testing.T) {
	s := New(&config.PDLRSIConfig{}, nil)
	level, reason := s.DesiredStop("SVCB", statestore.Entry{EntryPrice: 100, EntryATR: 0})
	if level != 0 || reason != "" {
		t.Fatalf("DesiredStop = %v/%q, want 0/\"\" при EntryATR=0", level, reason)
	}
	level, reason = s.DesiredStop("НЕТ", statestore.Entry{EntryPrice: 100, EntryATR: 4})
	if level != 0 || reason != "" {
		t.Fatalf("DesiredStop незарегистрированного тикера = %v/%q, want 0/\"\"", level, reason)
	}
}

func TestStrategyIsHourly(t *testing.T) {
	if got := New(&config.PDLRSIConfig{}, nil).BarInterval(); got != enum.Hour1 {
		t.Fatalf("BarInterval = %v, want Hour1", got)
	}
}

func TestDeciderUsesTickerParams(t *testing.T) {
	s := New(&config.PDLRSIConfig{}, nil)
	d, ok := s.Decider("SVCB")
	if !ok || d == nil {
		t.Fatalf("Decider(SVCB) ok=%v d=%v", ok, d)
	}
	if want := core.NewWithParams("SVCB", time.Hour, svcb.Params()).Lookback(); d.Lookback() != want {
		t.Fatalf("Lookback = %d, want %d", d.Lookback(), want)
	}
	d, ok = s.Decider("НЕТ")
	if ok || d != nil {
		t.Fatalf("Decider(НЕТ) = %v/%v, want nil/false", d, ok)
	}
}

func TestReconstructRebuildsEntryWithoutTarget(t *testing.T) {
	s := New(&config.PDLRSIConfig{}, nil)
	now := time.Date(2026, 3, 10, 15, 0, 0, 0, msk) // вторник
	buy := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	in := adapter.ReconstructInput{
		Trades:        &fakeTrades{trades: []grpcmodel.Trade{{IsBuy: true, Date: buy}}},
		Candles:       &fakeDaily{bars: weekdayBars(buy)},
		AccountID:     "acc",
		InstrumentID:  "uid",
		Ticker:        "SVCB",
		PurchasePrice: 123.5,
		Now:           now,
	}
	e, err := s.Reconstruct(context.Background(), in)
	if err != nil {
		t.Fatalf("Reconstruct: %v", err)
	}
	if e.Strategy != "" {
		t.Errorf("Strategy = %q, want пусто (проставляет раннер)", e.Strategy)
	}
	if e.Ticker != "SVCB" || !e.EntryTime.Equal(buy) {
		t.Errorf("Ticker/EntryTime = %s/%v, want SVCB/%v", e.Ticker, e.EntryTime, buy)
	}
	if e.TakeProfit != 0 {
		t.Errorf("TakeProfit = %v, want 0", e.TakeProfit)
	}
	if e.EntryPrice != 123.5 || e.MaxFav != 123.5 {
		t.Errorf("EntryPrice/MaxFav = %v/%v, want 123.5/123.5", e.EntryPrice, e.MaxFav)
	}
	if e.EntryATR <= 0 {
		t.Errorf("EntryATR = %v, want > 0", e.EntryATR)
	}
}

func TestReconstructFailsWithoutBuy(t *testing.T) {
	s := New(&config.PDLRSIConfig{}, nil)
	now := time.Date(2026, 3, 10, 15, 0, 0, 0, msk)
	in := adapter.ReconstructInput{
		Trades:        &fakeTrades{trades: []grpcmodel.Trade{{IsBuy: false, Date: now.Add(-time.Hour)}}},
		Candles:       &fakeDaily{bars: weekdayBars(now)},
		Ticker:        "SVCB",
		PurchasePrice: 100,
		Now:           now,
	}
	_, err := s.Reconstruct(context.Background(), in)
	if err == nil || !strings.Contains(err.Error(), "SVCB") {
		t.Fatalf("err = %v, want ошибка с тикером SVCB", err)
	}
}

func TestReconstructFailsWithoutDailyHistory(t *testing.T) {
	s := New(&config.PDLRSIConfig{}, nil)
	now := time.Date(2026, 3, 10, 15, 0, 0, 0, msk)
	in := adapter.ReconstructInput{
		Trades:        &fakeTrades{trades: []grpcmodel.Trade{{IsBuy: true, Date: now.Add(-time.Hour)}}},
		Candles:       &fakeDaily{},
		Ticker:        "SVCB",
		PurchasePrice: 100,
		Now:           now,
	}
	if _, err := s.Reconstruct(context.Background(), in); err == nil {
		t.Fatal("ожидалась ошибка: ATR не восстановить без дневок")
	}
}

func TestNotifyPassesMessageAsIsAndRespectsSwitch(t *testing.T) {
	t.Run("выключено", func(t *testing.T) {
		tg := tgmocks.NewMockClient(t) // без ожиданий: любой вызов — провал
		New(&config.PDLRSIConfig{NotifyEnabled: false}, tg).Notify("вход")
	})
	t.Run("включено", func(t *testing.T) {
		tg := tgmocks.NewMockClient(t)
		tg.EXPECT().SendMessage("вход").Return(nil).Once()
		New(&config.PDLRSIConfig{NotifyEnabled: true}, tg).Notify("вход")
	})
	t.Run("сбой доставки не паникует", func(t *testing.T) {
		tg := tgmocks.NewMockClient(t)
		tg.EXPECT().SendMessage("вход").Return(errors.New("boom")).Once()
		New(&config.PDLRSIConfig{NotifyEnabled: true}, tg).Notify("вход")
	})
}
