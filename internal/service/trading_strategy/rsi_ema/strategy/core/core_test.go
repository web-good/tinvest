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

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	if p.RSIPeriod != 12 || p.EMAFast != 10 || p.EMASlow != 50 {
		t.Fatalf("indicator defaults wrong: %+v", p)
	}
	if p.RSIMid != 50 || p.RSIUpper != 70 {
		t.Fatalf("zone defaults wrong: %+v", p)
	}
	if p.EntryCooldownBars != 1 {
		t.Fatalf("EntryCooldownBars = %d want 1", p.EntryCooldownBars)
	}
	if p.StopATR != 0 || p.ATRPeriod != 14 {
		t.Fatalf("risk defaults wrong: %+v", p)
	}
	if p.SessionStartMin != 420 || p.SessionEndMin != 1080 || p.FridayEndMin != 840 {
		t.Fatalf("session defaults wrong: %+v", p)
	}
}

func TestInSession(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		// 2026-07-20 Monday, 2026-07-24 Friday, 2026-07-25 Saturday.
		{"mon before open", mskAt(2026, 7, 20, 6, 55), false},
		{"mon at open", mskAt(2026, 7, 20, 7, 0), true},
		{"mon last bar", mskAt(2026, 7, 20, 17, 55), true},
		{"mon at close", mskAt(2026, 7, 20, 18, 0), false},
		{"fri before close", mskAt(2026, 7, 24, 13, 55), true},
		{"fri at close", mskAt(2026, 7, 24, 14, 0), false},
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
		t.Fatalf("zero time must not block entry")
	}
}

func TestIsDayEnd(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		span int
		want bool
	}{
		{"mon midday", mskAt(2026, 7, 20, 12, 0), 15, false},
		{"mon 17:40", mskAt(2026, 7, 20, 17, 40), 15, false},
		{"mon 17:45 last 15m bar closes 18:00", mskAt(2026, 7, 20, 17, 45), 15, true},
		{"mon after close", mskAt(2026, 7, 20, 18, 5), 15, true},
		{"fri 13:45 last 15m bar closes 14:00", mskAt(2026, 7, 24, 13, 45), 15, true},
		{"saturday always day end", mskAt(2026, 7, 25, 10, 0), 15, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.isDayEnd(tc.at, tc.span); got != tc.want {
				t.Fatalf("isDayEnd(%v,%d) = %v want %v", tc.at, tc.span, got, tc.want)
			}
		})
	}
}

func TestBarSpanMinutesMedian(t *testing.T) {
	times := []time.Time{
		mskAt(2026, 7, 20, 10, 0),
		mskAt(2026, 7, 20, 10, 15),
		mskAt(2026, 7, 20, 10, 30),
		mskAt(2026, 7, 21, 10, 0), // overnight jump — must not skew the median
	}
	if got := barSpanMinutes(times); got != 15 {
		t.Fatalf("barSpanMinutes = %d want 15", got)
	}
	if got := barSpanMinutes(nil); got != defaultBarSpanMin {
		t.Fatalf("barSpanMinutes(nil) = %d want %d", got, defaultBarSpanMin)
	}
}

func TestBarTimeRequiresAlignedTimes(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := strategy.MarketData{Closes: []float64{1, 2, 3}, Times: []time.Time{mskAt(2026, 7, 20, 10, 0)}}
	if got := s.barTime(md); !got.IsZero() {
		t.Fatalf("misaligned Times must yield zero time, got %v", got)
	}
	md.Times = []time.Time{mskAt(2026, 7, 20, 10, 0), mskAt(2026, 7, 20, 10, 15), mskAt(2026, 7, 20, 10, 30)}
	if got := s.barTime(md); !got.Equal(mskAt(2026, 7, 20, 10, 30)) {
		t.Fatalf("barTime = %v want latest", got)
	}
}

func TestLookbackWarmsSlowEMA(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if got := s.Lookback(); got < DefaultParams().EMASlow+20 {
		t.Fatalf("Lookback = %d too small to warm EMASlow", got)
	}
}

func TestTickerRoundTrip(t *testing.T) {
	if got := NewWithParams("SBER", DefaultParams()).Ticker(); got != "SBER" {
		t.Fatalf("Ticker = %q want SBER", got)
	}
}
