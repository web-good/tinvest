package golden_x

import (
	"errors"
	"testing"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

// ohlcCandle builds a CandleItemTechAnalyse with full OHLCV data.
func ohlcCandle(t time.Time, o, h, l, c int64, vol int64) *model.CandleItemTechAnalyse {
	return &model.CandleItemTechAnalyse{
		Time:   t,
		Open:   model.Quotation{Units: o},
		High:   model.Quotation{Units: h},
		Low:    model.Quotation{Units: l},
		Close:  model.Quotation{Units: c},
		Volume: vol,
	}
}

// buildWeeklyCandles builds n weekly candles ending at baseTime (ascending order).
// closeFn(i) gives close for candle i (i=0 oldest).
// High is close+2, Low is close-2, Open is close, Volume is 1000.
func buildWeeklyCandles(baseTime time.Time, n int, closeFn func(i int) int64) []*model.CandleItemTechAnalyse {
	out := make([]*model.CandleItemTechAnalyse, n)
	for i := 0; i < n; i++ {
		t := baseTime.AddDate(0, 0, -7*(n-1-i))
		c := closeFn(i)
		out[i] = ohlcCandle(t, c, c+2, c-2, c, 1000)
	}
	return out
}

// buildWeeklyCandlesOHLCV is like buildWeeklyCandles but lets caller supply
// OHLCV functions for more realistic ATR and volume data.
func buildWeeklyCandlesOHLCV(
	baseTime time.Time,
	n int,
	closeFn func(i int) int64,
	highFn func(i int, c int64) int64,
	lowFn func(i int, c int64) int64,
	volFn func(i int) int64,
) []*model.CandleItemTechAnalyse {
	out := make([]*model.CandleItemTechAnalyse, n)
	for i := 0; i < n; i++ {
		t := baseTime.AddDate(0, 0, -7*(n-1-i))
		c := closeFn(i)
		out[i] = ohlcCandle(t, c, highFn(i, c), lowFn(i, c), c, volFn(i))
	}
	return out
}

func TestDetect(t *testing.T) {
	// Use a fixed "now" so candle timestamps are always in the past (closed).
	// All candles will be timestamped well before this date.
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("insufficient_history", func(t *testing.T) {
		// 50 candles with rsiPeriod=7: 50-7=43 RSI values < adaptiveWindowMin(100)
		candles := buildWeeklyCandles(base, 50, func(i int) int64 { return int64(100 + i) })
		_, err := Detect(candles, 7, dto.StrategyKindDividend, false)
		if !errors.Is(err, ErrAdaptiveInsufficientHistory) {
			t.Fatalf("expected ErrAdaptiveInsufficientHistory, got %v", err)
		}
	})

	t.Run("no_buy", func(t *testing.T) {
		// Strictly rising series → RSI is persistently high → above P15 → no buy.
		// 120 candles with rsiPeriod=7: 113 RSI values, well above adaptiveWindowMin.
		candles := buildWeeklyCandles(base, 120, func(i int) int64 { return int64(100 + i*5) })
		sig, err := Detect(candles, 7, dto.StrategyKindDividend, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sig.GreenBuy {
			t.Error("GreenBuy should be false for rising series")
		}
		if sig.YellowBuy {
			t.Error("YellowBuy should be false for rising series")
		}
		// No buy means Stop should be zero value.
		if sig.Stop != (dto.Stop{}) {
			t.Errorf("Stop should be zero when no buy, got %+v", sig.Stop)
		}
		if sig.LastClose <= 0 {
			t.Error("LastClose should be positive even on no-buy")
		}
	})

	t.Run("mutual_exclusivity_and_threshold_ordering", func(t *testing.T) {
		// Property-based: GreenBuy and YellowBuy must be mutually exclusive.
		// Threshold monotonicity: P5 <= P15, P80 <= P90 <= P95.
		// Use a rising series (produces high, dispersed RSI history) so that
		// P5 < P15 holds (strictly for dispersed distributions).
		n := 120
		candles := buildWeeklyCandles(base, n, func(i int) int64 {
			return int64(100 + i*5)
		})
		sig, err := Detect(candles, 7, dto.StrategyKindDividend, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sig.GreenBuy && sig.YellowBuy {
			t.Error("GreenBuy and YellowBuy must not both be true")
		}
		// For a rising series, RSI history has spread → P5 < P15.
		if sig.Thresholds.P5 > sig.Thresholds.P15 {
			t.Errorf("P5 (%v) must be <= P15 (%v)", sig.Thresholds.P5, sig.Thresholds.P15)
		}
		if sig.SellThresholds.P80 > sig.SellThresholds.P90 {
			t.Errorf("P80 (%v) must be <= P90 (%v)", sig.SellThresholds.P80, sig.SellThresholds.P90)
		}
		if sig.SellThresholds.P90 > sig.SellThresholds.P95 {
			t.Errorf("P90 (%v) must be <= P95 (%v)", sig.SellThresholds.P90, sig.SellThresholds.P95)
		}
	})

	t.Run("green_buy_no_trend_filter", func(t *testing.T) {
		// Build a series that ensures the last RSI is in the bottom 5th percentile.
		// Strategy: 180 candles slowly rising (RSI ~50-60), then 20 candles sharply
		// dropping to force RSI very low. The drop produces a very low RSI relative
		// to the history.
		n := 200
		candles := buildWeeklyCandles(base, n, func(i int) int64 {
			if i < 180 {
				// Slow uptrend: produces moderate-high RSI history.
				return int64(1000 + i*3)
			}
			// Sharp drop: RSI plummets.
			dropDepth := int64(i - 180)
			return int64(1000+180*3) - dropDepth*80
		})
		sig, err := Detect(candles, 7, dto.StrategyKindDividend, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Either green or yellow buy should fire (last RSI very low vs history).
		if !sig.GreenBuy && !sig.YellowBuy {
			t.Logf("sig: RSI=%.2f P5=%.2f P15=%.2f", sig.RSI, sig.Thresholds.P5, sig.Thresholds.P15)
			t.Error("expected GreenBuy or YellowBuy to be true for sharp-drop series")
		}
		if sig.GreenBuy && sig.YellowBuy {
			t.Error("GreenBuy and YellowBuy must not both be true")
		}
		if sig.LastClose <= 0 {
			t.Error("LastClose must be positive")
		}
	})

	t.Run("trend_filter_off_leaves_trend_unknown", func(t *testing.T) {
		// With useTrendFilter=false, TrendStatus must be TrendUnknown.
		candles := buildWeeklyCandles(base, 120, func(i int) int64 { return int64(100 + i*5) })
		sig, err := Detect(candles, 7, dto.StrategyKindDividend, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sig.TrendStatus != dto.TrendUnknown {
			t.Errorf("TrendStatus = %v, want TrendUnknown when useTrendFilter=false", sig.TrendStatus)
		}
	})

	t.Run("trend_filter_on_with_strong_uptrend", func(t *testing.T) {
		// 220 candles, strictly rising — last close >> EMA200.
		// useTrendFilter=true should produce TrendWith.
		n := 220
		candles := buildWeeklyCandles(base, n, func(i int) int64 {
			return int64(1000 + i*10)
		})
		sig, err := Detect(candles, 7, dto.StrategyKindGrowth, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sig.TrendStatus != dto.TrendWith {
			t.Errorf("TrendStatus = %v, want TrendWith for strong uptrend", sig.TrendStatus)
		}
	})

	t.Run("trend_filter_on_insufficient_history", func(t *testing.T) {
		// 110 candles: enough RSI history (103 values) but fewer than trendEMAPeriod(200)
		// → ErrInsufficientHistory from trend filter.
		candles := buildWeeklyCandles(base, 110, func(i int) int64 { return int64(100 + i*5) })
		_, err := Detect(candles, 7, dto.StrategyKindGrowth, true)
		if !errors.Is(err, ErrInsufficientHistory) {
			t.Fatalf("expected ErrInsufficientHistory with trend filter and short history, got %v", err)
		}
	})

	t.Run("stop_computed_when_buy_and_atr_positive", func(t *testing.T) {
		// Build a series designed to produce a buy signal with non-zero ATR
		// and prices high enough that stop = lastClose - k*ATR > 0.
		// - 180 candles slowly rising from 5000 (giving RSI history in uptrend)
		// - 20 candles moderately dropping to force RSI below P15
		// - High/Low spreads +50/-50 give ATR ≈ 50, k=2.0 (Dividend)
		//   → stop = lastClose - 2*50 = lastClose - 100; keeping lastClose > 200.
		const base5000Rise = 5000
		n := 200
		candles := buildWeeklyCandlesOHLCV(
			base,
			n,
			func(i int) int64 {
				if i < 180 {
					return int64(base5000Rise + i*5)
				}
				// Moderate drop: 5000+180*5=5900 base, drop 15 per week.
				dropDepth := int64(i - 180)
				return int64(base5000Rise+180*5) - dropDepth*15
			},
			func(i int, c int64) int64 { return c + 50 },
			func(i int, c int64) int64 { return c - 50 },
			func(i int) int64 { return 1000 },
		)
		sig, err := Detect(candles, 7, dto.StrategyKindDividend, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !sig.GreenBuy && !sig.YellowBuy {
			t.Fatalf("series did not produce a buy signal; cannot test Stop. RSI=%.2f P5=%.2f P15=%.2f", sig.RSI, sig.Thresholds.P5, sig.Thresholds.P15)
		}
		if sig.Stop.Price <= 0 {
			t.Errorf("Stop.Price = %v, want > 0 for buy signal with non-flat candles", sig.Stop.Price)
		}
		if sig.Stop.DistancePct <= 0 {
			t.Errorf("Stop.DistancePct = %v, want > 0", sig.Stop.DistancePct)
		}
		if sig.LastClose <= 0 {
			t.Errorf("LastClose = %v, want > 0", sig.LastClose)
		}
	})

	t.Run("yellow_buy_property", func(t *testing.T) {
		// Property test: YellowBuy implies GreenBuy is false and vice versa.
		// We build many series variants and check mutual exclusivity always holds.
		seriesVariants := []struct {
			name    string
			closeFn func(i int) int64
		}{
			{"rising", func(i int) int64 { return int64(100 + i*5) }},
			{"falling", func(i int) int64 { return int64(2000 - i*5) }},
			{"flat", func(i int) int64 { return 500 }},
			{"alternating", func(i int) int64 {
				if i%2 == 0 {
					return 100
				}
				return 110
			}},
		}
		for _, v := range seriesVariants {
			t.Run(v.name, func(t *testing.T) {
				candles := buildWeeklyCandles(base, 120, v.closeFn)
				sig, err := Detect(candles, 7, dto.StrategyKindDividend, false)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if sig.GreenBuy && sig.YellowBuy {
					t.Error("GreenBuy and YellowBuy must not both be true")
				}
			})
		}
	})
}
