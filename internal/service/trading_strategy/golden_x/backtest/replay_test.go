package backtest

import (
	"testing"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func mkCandle(t time.Time, low, close int64) *model.CandleItemTechAnalyse {
	return &model.CandleItemTechAnalyse{
		Time:  t,
		High:  model.Quotation{Units: close + 1},
		Low:   model.Quotation{Units: low},
		Close: model.Quotation{Units: close},
	}
}

// TestReplay_StopFiresBeforeSellTier verifies §4.2 ordering 1a → 1b.
func TestReplay_StopFiresBeforeSellTier(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []*model.CandleItemTechAnalyse{
		mkCandle(base, 100, 100),
		mkCandle(base.AddDate(0, 0, 7), 100, 100),
		mkCandle(base.AddDate(0, 0, 14), 100, 100), // entry week
		mkCandle(base.AddDate(0, 0, 21), 80, 110),  // Low 80 hits stop @ 90; close 110 also above P95
		mkCandle(base.AddDate(0, 0, 28), 110, 110),
	}
	fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool) (dto.Signal, error) {
		last := closed[len(closed)-1]
		switch last.Time {
		case candles[2].Time:
			return dto.Signal{GreenBuy: true, RSI: 5, LastClose: 100,
				Stop:           dto.Stop{Price: 90},
				SellThresholds: dto.SellThresholds{P80: 70, P90: 80, P95: 90}}, nil
		case candles[3].Time, candles[4].Time:
			return dto.Signal{RSI: 95, LastClose: 110,
				SellThresholds: dto.SellThresholds{P80: 70, P90: 80, P95: 90}}, nil
		}
		return dto.Signal{RSI: 50, SellThresholds: dto.SellThresholds{P80: 70, P90: 80, P95: 90}}, nil
	}
	cfg := ReplayConfig{Kind: dto.StrategyKindDividend, StartIdx: 2, MaxWeeks: 52, UseTrendFilter: false}
	trades := Replay("X", candles, fake, cfg)

	if len(trades) != 1 || trades[0].ExitReason != ExitReasonStop {
		t.Fatalf("expected single stop trade, got %+v", trades)
	}
}

// TestReplay_SellSequenceProducesThreePartials covers Gold p80→p90→p95
// across three different weeks (§4.3).
func TestReplay_SellSequenceProducesThreePartials(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	const N = 8
	candles := make([]*model.CandleItemTechAnalyse, N)
	for i := 0; i < N; i++ {
		candles[i] = mkCandle(base.AddDate(0, 0, 7*i), 95, 100+int64(i))
	}
	st := dto.SellThresholds{P80: 70, P90: 80, P95: 90}
	fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool) (dto.Signal, error) {
		idx := len(closed) - 1
		switch idx {
		case 2:
			return dto.Signal{GreenBuy: true, LastClose: 102, Stop: dto.Stop{Price: 1}, SellThresholds: st}, nil
		case 3:
			return dto.Signal{RSI: 72, LastClose: 103, SellThresholds: st}, nil
		case 4:
			return dto.Signal{RSI: 85, LastClose: 104, SellThresholds: st}, nil
		case 5:
			return dto.Signal{RSI: 95, LastClose: 105, SellThresholds: st}, nil
		default:
			return dto.Signal{RSI: 50, SellThresholds: st}, nil
		}
	}
	trades := Replay("X", candles, fake, ReplayConfig{Kind: dto.StrategyKindDividend, StartIdx: 2, MaxWeeks: 52})
	if len(trades) != 3 {
		t.Fatalf("expected 3 partials, got %d: %+v", len(trades), trades)
	}
	reasons := [3]ExitReason{trades[0].ExitReason, trades[1].ExitReason, trades[2].ExitReason}
	want := [3]ExitReason{ExitReasonSellP80, ExitReasonSellP90, ExitReasonSellP95}
	if reasons != want {
		t.Fatalf("reasons=%v, want %v", reasons, want)
	}
}

// TestReplay_TimeoutAfter52Weeks covers exit-ordering rule 1c.
func TestReplay_TimeoutAfter52Weeks(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	const N = 60
	candles := make([]*model.CandleItemTechAnalyse, N)
	for i := 0; i < N; i++ {
		candles[i] = mkCandle(base.AddDate(0, 0, 7*i), 90, 100)
	}
	st := dto.SellThresholds{P80: 99, P90: 99, P95: 99}
	fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool) (dto.Signal, error) {
		idx := len(closed) - 1
		if idx == 2 {
			return dto.Signal{GreenBuy: true, LastClose: 100, Stop: dto.Stop{Price: 80}, SellThresholds: st}, nil
		}
		return dto.Signal{RSI: 50, SellThresholds: st}, nil
	}
	trades := Replay("X", candles, fake, ReplayConfig{Kind: dto.StrategyKindDividend, StartIdx: 2, MaxWeeks: 52})
	if len(trades) != 1 || trades[0].ExitReason != ExitReasonTimeout {
		t.Fatalf("expected one timeout, got %+v", trades)
	}
	if trades[0].WeeksHeld < 52 {
		t.Fatalf("WeeksHeld = %d, want >=52", trades[0].WeeksHeld)
	}
}

// TestReplay_OpenAtEndOfHistory ensures positions still open at the last
// candle are recorded with ExitReasonOpen + close-as-exit-price.
func TestReplay_OpenAtEndOfHistory(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	const N = 5
	candles := make([]*model.CandleItemTechAnalyse, N)
	for i := 0; i < N; i++ {
		candles[i] = mkCandle(base.AddDate(0, 0, 7*i), 90, 100+int64(i))
	}
	st := dto.SellThresholds{P80: 99, P90: 99, P95: 99}
	fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool) (dto.Signal, error) {
		if len(closed) == 3 {
			return dto.Signal{GreenBuy: true, LastClose: 102, Stop: dto.Stop{Price: 50}, SellThresholds: st}, nil
		}
		return dto.Signal{RSI: 50, SellThresholds: st}, nil
	}
	trades := Replay("X", candles, fake, ReplayConfig{Kind: dto.StrategyKindDividend, StartIdx: 2, MaxWeeks: 52})
	if len(trades) != 1 || trades[0].ExitReason != ExitReasonOpen {
		t.Fatalf("expected one open trade, got %+v", trades)
	}
	lastClose := float64(candles[N-1].Close.Units)
	if trades[0].ExitPrice != lastClose {
		t.Fatalf("ExitPrice = %v, want %v", trades[0].ExitPrice, lastClose)
	}
}
