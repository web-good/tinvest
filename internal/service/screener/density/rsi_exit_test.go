package density

import (
	"context"
	"testing"
	"time"
)

type minuteClosesFunc func(context.Context, string, time.Time) ([]float64, error)

func (f minuteClosesFunc) MinuteCloses(ctx context.Context, id string, end time.Time) ([]float64, error) {
	return f(ctx, id, end)
}

func TestRSIExitThresholdAndCandleExpiry(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 1, 0, time.UTC)
	source := minuteClosesFunc(func(_ context.Context, id string, end time.Time) ([]float64, error) {
		if id != "uid" || !end.Equal(now.Truncate(time.Minute)) {
			return nil, nil
		}
		return []float64{100, 101, 102, 103, 104, 105, 106}, nil
	})
	exit := NewRSIExit(t.Context(), source)
	if exit("uid", 100e9, 100.2e9, now) {
		t.Fatal("exit before minute history was available")
	}
	deadline := time.Now().Add(time.Second)
	for !exit("uid", 100e9, 100.2e9, now) {
		if time.Now().After(deadline) {
			t.Fatal("RSI above 70 and exactly 0.2% gain did not trigger")
		}
		time.Sleep(time.Millisecond)
	}
	if exit("uid", 100e9, 100.2e9-1, now) {
		t.Fatal("gain below 0.2% triggered")
	}
	if exit("uid", 100e9, 100.2e9, now.Add(time.Minute)) {
		t.Fatal("RSI from a stale candle triggered")
	}
}
