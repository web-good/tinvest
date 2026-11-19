package volume

import (
	"testing"
	"time"
)

func TestPreviousWeekdayAcrossWeekend(t *testing.T) {
	for _, day := range []int{21, 22, 23} {
		now := time.Date(2026, 11, day, 1, 0, 0, 0, time.UTC)
		from, to, until := PreviousWeekday(now)
		want := time.Date(2026, 11, 20, 6, 0, 0, 0, time.UTC)
		if !from.Equal(want) || !to.Equal(want.Add(9*time.Hour)) ||
			!until.Equal(time.Date(2026, 11, 23, 21, 0, 0, 0, time.UTC)) {
			t.Fatalf("day %d: window %v to %v, valid until %v", day, from, to, until)
		}
	}
}

func TestMeanPreservesEqualFractionalVolumes(t *testing.T) {
	from := time.Date(2026, 11, 18, 6, 0, 0, 0, time.UTC)
	samples := Samples{From: from, To: from.Add(9 * time.Hour)}
	for i := range 3 {
		samples.Add(from.Add(time.Duration(i)*CandleInterval), .1)
	}
	if got := samples.Mean(); got != .1 {
		t.Fatalf("expected volume .1, got %.17g", got)
	}
}
