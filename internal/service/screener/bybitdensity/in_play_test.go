package bybitdensity

import (
	"context"
	"fmt"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/bybitdensity/strategy"
)

type inPlayCandles func(from, to time.Time) []model.M5Candle

func (f inPlayCandles) SpotM5(_ context.Context, _ model.Key, from, to time.Time) ([]model.M5Candle, error) {
	return f(from, to), nil
}

func TestBacktestInPlayUsesCandlesClosedAtEachWindow(t *testing.T) {
	start := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)
	source := inPlayCandles(func(from, to time.Time) []model.M5Candle {
		if !from.Equal(start.Add(-24*time.Hour-30*time.Minute)) || !to.Equal(start.Add(5*time.Minute)) {
			t.Fatalf("unexpected history range %v to %v", from, to)
		}
		var candles []model.M5Candle
		for at := from.Add(5 * time.Minute); !at.After(to); at = at.Add(5 * time.Minute) {
			candle := model.M5Candle{End: at, Open: 100, High: 100, Low: 100, Turnover: 100}
			if at.After(start.Add(-30*time.Minute)) && !at.After(start) {
				candle.High, candle.Turnover = 101, 200
			}
			candles = append(candles, candle)
		}
		return candles
	})
	windows, err := loadInPlayWindows(t.Context(), source, model.Key{Market: model.Spot, Symbol: "PIEVERSEUSDT"},
		strategy.InPlayFilter{MinRange: .01, MinTurnoverMultiple: 2}, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 2 || !windows[0].From.Equal(start) || !windows[0].Active ||
		!windows[1].From.Equal(start.Add(5*time.Minute)) || windows[1].Active {
		t.Fatalf("unexpected windows %+v", windows)
	}
}

func TestBacktestInPlayLeavesWindowsInactiveAcrossHistoryGap(t *testing.T) {
	start := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	end := start.Add(15 * time.Minute)
	source := inPlayCandles(func(from, to time.Time) []model.M5Candle {
		var candles []model.M5Candle
		for at := from.Add(5 * time.Minute); !at.After(to); at = at.Add(5 * time.Minute) {
			if at.Equal(start.Add(5 * time.Minute)) {
				continue
			}
			candle := model.M5Candle{End: at, Open: 100, High: 100, Low: 100, Turnover: 100}
			if at.After(start.Add(-30 * time.Minute)) {
				candle.High, candle.Turnover = 105, 300
			}
			candles = append(candles, candle)
		}
		return candles
	})
	windows, err := loadInPlayWindows(t.Context(), source, model.Key{Market: model.Spot, Symbol: "PIEVERSEUSDT"},
		strategy.InPlayFilter{MinRange: .01, MinTurnoverMultiple: 2}, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 3 || !windows[0].Active || windows[1].Active || windows[2].Active ||
		!windows[1].From.Equal(start.Add(5*time.Minute)) || !windows[2].From.Equal(start.Add(10*time.Minute)) {
		t.Fatalf("unexpected windows across a missing candle: %+v", windows)
	}
}

type missingReferenceVolume struct{ marketData }

func (missingReferenceVolume) AverageVolume(context.Context, model.Key, time.Time, time.Time) (float64, error) {
	return 0, fmt.Errorf("missing m5 candle")
}

func TestBacktestInPlayCanRunWithoutReferenceVolumeForReport(t *testing.T) {
	start := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	windows, err := loadBacktestVolumes(t.Context(), missingReferenceVolume{}, model.Key{Market: model.Spot, Symbol: "PIEVERSEUSDT"},
		defaultSpotProfiles()["PIEVERSEUSDT"], start, start.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) == 0 || windows[0].average != 0 {
		t.Fatalf("expected an unavailable report volume, got %+v", windows)
	}
}
