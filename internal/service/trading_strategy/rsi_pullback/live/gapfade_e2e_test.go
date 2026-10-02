package live

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	gaplive "tinvest/internal/service/trading_strategy/gap_fade/live"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// gapTape — лента gap_fade: 120 баров по 100 до 23:30 9 марта, затем бары 10 марта с 06:30 по
// last. Первый бар дня открывается гэпом 85 и закрывается 88 (дневной ATR фикстуры dailies = 10,
// гэп −1.5 ATR), остальные стоят на 90 с диапазоном ±0.5 — ни цели 97, ни стопа 78.
func gapTape(last time.Time) []*imodel.CandleItemTechAnalyse {
	closes := make([]float64, 120)
	for i := range closes {
		closes[i] = 100
	}
	out := tape30m(time.Date(2026, 3, 9, 23, 30, 0, 0, msk), closes)
	for t := time.Date(2026, 3, 10, 6, 30, 0, 0, msk); !t.After(last); t = t.Add(30 * time.Minute) {
		o, c := 90.0, 90.0
		if t.Hour() == 6 {
			o, c = 85, 88
		}
		out = append(out, &imodel.CandleItemTechAnalyse{Time: t, Open: qf(o), High: qf(max(o, c) + 0.5),
			Low: qf(min(o, c) - 0.5), Close: qf(c), Volume: 1000, IsComplete: true})
	}
	return out
}

type noDivs struct{}

func (noDivs) GetDividends(context.Context, string, time.Time, time.Time) ([]*grpcmodel.Dividend, error) {
	return nil, nil
}

func gapGuest() *gaplive.Strategy {
	return gaplive.New(&config.GapFadeConfig{Tickers: []string{"GAZP"}, BuyPct: 5}, nil, noDivs{})
}

// Гэп-день: пасс 07:01 входит (dry-run, боевых стратегий нет), стоп и цель заморожены от close
// сигнала 88: стоп 88 − 1.0·10 = 78, цель 88 + 0.75·(100 − 88) = 97.
func TestGapFadeEntersOnGapMorning(t *testing.T) {
	now := time.Date(2026, 3, 10, 7, 1, 0, 0, msk)
	e := newSharedEnv(t, now, nil, nil, gapGuest())
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(gapTape(time.Date(2026, 3, 10, 6, 30, 0, 0, msk)), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	en, ok := e.state(t)["GAZP"]
	if !ok || en.Strategy != "gap_fade" {
		t.Fatalf("нет входа gap_fade: %+v", e.state(t))
	}
	if en.StopLoss != 78 || en.TakeProfit != 97 || en.StopPrice != 78 || en.StopReason != "SL" {
		t.Fatalf("уровни = stopLoss %v, tp %v, stop %v/%q; want 78/97/78/SL", en.StopLoss, en.TakeProfit, en.StopPrice, en.StopReason)
	}
}

func seededGap(now time.Time) statestore.Entry {
	return statestore.Entry{Ticker: "GAZP", Strategy: "gap_fade", EntryTime: time.Date(2026, 3, 10, 7, 1, 0, 0, msk),
		EntryPrice: 88, EntryATR: 10, TakeProfit: 97, StopLoss: 78, MaxFav: 88, Quantity: 100}
}

// Цель: бар с high ≥ 97 — SELL на следующем пассе, запись удалена.
func TestGapFadeTakeProfitExit(t *testing.T) {
	now := time.Date(2026, 3, 10, 8, 1, 0, 0, msk)
	e := newSharedEnv(t, now, nil, nil, gapGuest())
	e.seed(t, seededGap(now))
	tape := gapTape(time.Date(2026, 3, 10, 7, 30, 0, 0, msk))
	tape[len(tape)-1].High = qf(98)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(tape, dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("TP не закрыл позицию")
	}
}

// EOD: пасс 19:01 после бара 18:30 закрывает позицию — вне окна входа сопровождение работает.
func TestGapFadeEODExitOutsideEntryWindow(t *testing.T) {
	now := time.Date(2026, 3, 10, 19, 1, 0, 0, msk)
	e := newSharedEnv(t, now, nil, nil, gapGuest())
	e.seed(t, seededGap(now))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(gapTape(time.Date(2026, 3, 10, 18, 30, 0, 0, msk)), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("EOD не закрыл позицию")
	}
}

// Тот же гэп в 12:01 — вне окна: свечи не запрашиваются (market-мок без ожиданий), входа нет.
func TestGapFadeDoesNotFetchOutsideEntryWindow(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	e := newSharedEnv(t, now, nil, nil, gapGuest())
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(e.state(t)) != 0 {
		t.Fatal("вход вне окна")
	}
}
