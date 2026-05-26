package golden_x

import (
	"errors"
	"math"
	"testing"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func TestComputeEMA(t *testing.T) {
	// Period=3 EMA over the sequence [1,2,3,4,5,6]:
	// seed (pos 2) = (1+2+3)/3 = 2
	// multiplier  = 2/(3+1) = 0.5
	// pos 3: (4-2)*0.5 + 2 = 3
	// pos 4: (5-3)*0.5 + 3 = 4
	// pos 5: (6-4)*0.5 + 4 = 5
	got := ema.Compute([]float64{1, 2, 3, 4, 5, 6}, 3)
	want := []float64{0, 0, 2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("pos %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestComputeEMA_InsufficientReturnsZeroes(t *testing.T) {
	got := ema.Compute([]float64{1, 2}, 3)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	for i, v := range got {
		if v != 0 {
			t.Errorf("pos %d: got %v, want 0", i, v)
		}
	}
}

// candleAt builds a CandleItemTechAnalyse with Time t and Close c (Units only).
func candleAt(t time.Time, c int64) *model.CandleItemTechAnalyse {
	return &model.CandleItemTechAnalyse{
		Time:  t,
		Close: model.Quotation{Units: c},
	}
}

func TestTrendStatusFromCandles(t *testing.T) {
	// All weekly times are arbitrary — trendStatusFromClosed treats the slice
	// as opaque ordered closes. Index 0 is the oldest, the last index is the
	// newest candle (which in prod may be the currently forming week).
	base := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)

	build := func(n int, closeFn func(i int) int64) []*model.CandleItemTechAnalyse {
		out := make([]*model.CandleItemTechAnalyse, n)
		for i := 0; i < n; i++ {
			weekTime := base.AddDate(0, 0, 7*i)
			out[i] = candleAt(weekTime, closeFn(i))
		}
		return out
	}

	t.Run("rising series — close above EMA → dto.TrendWith", func(t *testing.T) {
		candles := build(10, func(i int) int64 { return int64(100 + i*5) })
		got, err := trendStatusFromClosed(candles, 3)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != dto.TrendWith {
			t.Fatalf("got %v, want dto.TrendWith", got)
		}
	})

	t.Run("falling series — close below EMA → dto.TrendAgainst", func(t *testing.T) {
		candles := build(10, func(i int) int64 { return int64(200 - i*5) })
		got, err := trendStatusFromClosed(candles, 3)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != dto.TrendAgainst {
			t.Fatalf("got %v, want dto.TrendAgainst", got)
		}
	})

	t.Run("close == EMA → dto.TrendAgainst (strict >)", func(t *testing.T) {
		candles := build(10, func(int) int64 { return 100 })
		got, err := trendStatusFromClosed(candles, 3)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != dto.TrendAgainst {
			t.Fatalf("got %v, want dto.TrendAgainst", got)
		}
	})

	t.Run("insufficient history — fewer candles than period", func(t *testing.T) {
		candles := build(2, func(i int) int64 { return int64(100 + i) })
		_, err := trendStatusFromClosed(candles, 3)
		if !errors.Is(err, ErrInsufficientHistory) {
			t.Fatalf("err = %v, want ErrInsufficientHistory", err)
		}
	})

	t.Run("forming week is counted toward history (no time-based filter)", func(t *testing.T) {
		// 3 candles total — 2 prior + 1 forming. Under the new behavior the
		// forming candle is part of the input; period=3 is now satisfied.
		candles := build(3, func(i int) int64 { return int64(100 + i) })
		// Mark the last one as forming — trendStatusFromClosed must not care.
		candles[2].IsComplete = false
		_, err := trendStatusFromClosed(candles, 3)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})

	t.Run("forming week biases the comparison (last close drives trend)", func(t *testing.T) {
		// 4 flat closes (=100) plus a forming week with a much higher close.
		// Under the new behavior the forming candle pulls last close above EMA.
		candles := build(5, func(i int) int64 {
			if i == 4 {
				return 10_000
			}
			return 100
		})
		candles[4].IsComplete = false
		got, err := trendStatusFromClosed(candles, 3)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != dto.TrendWith {
			t.Fatalf("got %v, want dto.TrendWith (forming week must influence trend)", got)
		}
	})
}

func TestTrendStatus_Mark(t *testing.T) {
	tests := []struct {
		status dto.TrendStatus
		want   string
	}{
		{dto.TrendWith, "✅"},
		{dto.TrendAgainst, "🚫"},
		{dto.TrendUnknown, ""},
	}
	for _, tc := range tests {
		if got := tc.status.Mark(); got != tc.want {
			t.Errorf("status %v: got %q, want %q", tc.status, got, tc.want)
		}
	}
}
