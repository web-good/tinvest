package movement

import (
	"testing"
	"time"
)

func TestSummarize(t *testing.T) {
	end := time.Unix(3600, 0)
	bars := []bar{}
	for i := 6; i > 0; i-- {
		bars = append(bars, bar{end.Add(-time.Duration(i) * step), 100, 100.5, 100, 10})
	}
	got, ok := summarize(bars, end, 6, 0.5)
	if !ok || got.mean != 0.5 || got.hits != 6 {
		t.Fatal(got, ok)
	}
	bars[0].high = 103
	bars[1].high = 100
	got, ok = summarize(bars, end, 6, 0.5)
	if !ok || got.hits != 5 || got.mean != 5.0/6 {
		t.Fatal(got, ok)
	}
	bars[1].at = end
	if _, ok = summarize(bars, end, 6, 0.5); ok {
		t.Fatal("accepted incomplete window")
	}
}

func TestVolumeBaselineUsesCompleteMatchingWindows(t *testing.T) {
	day := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	history := make(volumeHistory)
	fetches := 0
	fetch := func(_ string, start, _ time.Time) ([]bar, error) {
		fetches++
		if !start.Before(day) {
			t.Fatalf("requested current day: %v", start)
		}
		offset := int(day.Sub(start) / (24 * time.Hour))
		if offset > 11 {
			return nil, nil
		}
		first := 0
		if offset == 1 {
			first = 1 // The first historical hour is incomplete.
		}
		var bars []bar
		for i := first; i < 12; i++ {
			volume := 1.0
			if i >= 6 {
				volume = 3
			}
			bars = append(bars, bar{start.Add(time.Duration(i) * step), 100, 101, 100, volume})
		}
		return bars, nil
	}

	end := day.Add(time.Hour)
	for _, tc := range []struct {
		bars int
		want float64
	}{{12, 24}, {6, 18}} {
		got, err := history.baseline(fetch, "TEST", end, tc.bars)
		if err != nil || got != tc.want {
			t.Fatalf("baseline for %d bars = %v, %v, want %v", tc.bars, got, err, tc.want)
		}
	}
	if fetches != 11 {
		t.Fatalf("fetched %d days, want 11 after reusing hourly history", fetches)
	}
	if got, err := history.baseline(fetch, "TEST", day.Add(2*time.Hour), 6); got != 0 || err != nil {
		t.Fatalf("baseline for missing windows = %v, %v", got, err)
	}
}
