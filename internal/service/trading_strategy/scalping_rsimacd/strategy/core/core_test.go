package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// mskAt builds an MSK bar open-time for the given date and clock.
func mskAt(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, mskLoc)
}

func TestInSession(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		// 2026-07-20 is a Monday, 2026-07-24 a Friday, 2026-07-25 a Saturday.
		{"mon before open", mskAt(2026, 7, 20, 7, 55), false},
		{"mon at open", mskAt(2026, 7, 20, 8, 0), true},
		{"mon midday", mskAt(2026, 7, 20, 12, 30), true},
		{"mon last bar", mskAt(2026, 7, 20, 16, 55), true},
		{"mon at close", mskAt(2026, 7, 20, 17, 0), false},
		{"mon after close", mskAt(2026, 7, 20, 17, 5), false},
		{"fri before friday close", mskAt(2026, 7, 24, 13, 55), true},
		{"fri at friday close", mskAt(2026, 7, 24, 14, 0), false},
		{"fri after friday close", mskAt(2026, 7, 24, 16, 0), false},
		{"saturday", mskAt(2026, 7, 25, 12, 0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.inSession(tc.at); got != tc.want {
				t.Fatalf("inSession(%v) = %v want %v", tc.at, got, tc.want)
			}
		})
	}
}

func TestInSessionZeroTimeIsPermissive(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if !s.inSession(time.Time{}) {
		t.Fatalf("zero time must not block the entry gate")
	}
}

func TestIsDayEnd(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"mon midday", mskAt(2026, 7, 20, 12, 0), false},
		{"mon 16:50", mskAt(2026, 7, 20, 16, 50), false},
		{"mon 16:55 last bar closes at 17:00", mskAt(2026, 7, 20, 16, 55), true},
		{"mon 17:05", mskAt(2026, 7, 20, 17, 5), true},
		{"fri 13:50", mskAt(2026, 7, 24, 13, 50), false},
		{"fri 13:55 last bar closes at 14:00", mskAt(2026, 7, 24, 13, 55), true},
		{"saturday always day end", mskAt(2026, 7, 25, 10, 0), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.isDayEnd(tc.at); got != tc.want {
				t.Fatalf("isDayEnd(%v) = %v want %v", tc.at, got, tc.want)
			}
		})
	}
}

func TestIsDayEndZeroTimeIsNoOp(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if s.isDayEnd(time.Time{}) {
		t.Fatalf("zero time must degrade the EOD exit to a no-op")
	}
}

func TestBarTimeRequiresAlignedTimes(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := strategy.MarketData{
		Closes: []float64{1, 2, 3},
		Times:  []time.Time{mskAt(2026, 7, 20, 10, 0)}, // misaligned on purpose
	}
	if got := s.barTime(md); !got.IsZero() {
		t.Fatalf("misaligned Times must yield zero time, got %v", got)
	}
	md.Times = []time.Time{
		mskAt(2026, 7, 20, 10, 0),
		mskAt(2026, 7, 20, 10, 5),
		mskAt(2026, 7, 20, 10, 10),
	}
	if got := s.barTime(md); !got.Equal(mskAt(2026, 7, 20, 10, 10)) {
		t.Fatalf("barTime = %v want the latest bar time", got)
	}
}

func TestLookbackCoversIndicatorWarmup(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if got := s.Lookback(); got != 120 {
		t.Fatalf("Lookback = %d want 120", got)
	}
}

func TestTickerRoundTrip(t *testing.T) {
	if got := NewWithParams("SBER", DefaultParams()).Ticker(); got != "SBER" {
		t.Fatalf("Ticker = %q want SBER", got)
	}
}
