package bounce

import (
	"testing"
	"time"
)

func TestWatchRequiresTwoConsecutiveClosedCandlesAndFreshEntryMinute(t *testing.T) {
	start := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name       string
		low, close int64
		lost, skip bool
		want       bool
	}{
		{"touch", 100, 101, false, false, true},
		{"five tick breach", 95, 101, false, false, true},
		{"deep breach", 94, 101, false, false, false},
		{"close equal", 100, 100, false, false, false},
		{"lost wall", 100, 101, true, false, false},
		{"missed candle", 100, 101, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var w Watch[string]
			w.Book(start.Add(-time.Second), func(string) bool { return true })
			w.Track(100, "signal")
			closeAt := start.Add(time.Minute)
			w.Candle(start, tc.low, 101, closeAt)
			if _, _, ok := w.Ready(closeAt); ok {
				t.Fatal("entry before confirmation")
			}
			if tc.lost {
				w.Book(closeAt, func(string) bool { return false })
				w.Track(100, "replacement")
			}
			second := closeAt
			if tc.skip {
				second = second.Add(time.Minute)
			}
			entry := second.Add(time.Minute)
			w.Candle(second, 101, tc.close, entry)
			if _, _, ok := w.Ready(entry); ok {
				t.Fatal("entry without a book from the entry minute")
			}
			w.Book(entry, func(string) bool { return true })
			if value, until, ok := w.Ready(entry); ok != tc.want || ok && (value != "signal" || !until.Equal(entry.Add(time.Minute))) {
				t.Fatalf("ready=%v value=%q until=%v", ok, value, until)
			}
			w.Consume(entry)
			if _, _, ok := w.Ready(entry); ok {
				t.Fatal("entry replayed")
			}
		})
	}
}

func TestRetainsThirtyPercentAtOriginalPrice(t *testing.T) {
	for _, tc := range []struct {
		initial, current float64
		want             bool
	}{
		{1000, 300, true},
		{1000, 299, false},
		{0.07, 0.021, true},
		{0.07, 0.0209999, false},
		{1000, 0, false},
	} {
		if got := Retains(tc.initial, tc.current); got != tc.want {
			t.Errorf("retains %v of %v = %t, want %t", tc.current, tc.initial, got, tc.want)
		}
	}
}
