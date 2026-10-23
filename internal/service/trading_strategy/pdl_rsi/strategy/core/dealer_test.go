package core

import (
	"math"
	"slices"
	"testing"
	"time"
)

// flatFrom is n flat bars at 100 from MSK `from`.
func flatFrom(from string, n int) []bar {
	return path(from, 100, slices.Repeat([]float64{100}, n)...)
}

func TestDealerDate(t *testing.T) {
	times := []time.Time{
		at("2026-09-14 10:00"), // Monday, the oldest visible date starts mid-day: exchange
		at("2026-09-14 10:15"),
		at("2026-09-15 02:00"), // Tuesday holiday: dealer bars from 02:00
		at("2026-09-15 10:00"),
		at("2026-09-16 06:45"), // Wednesday: exchange
		at("2026-09-16 10:00"),
		at("2026-09-19 02:00"), // Saturday: weekends are not dealer dates
	}
	want := []bool{false, false, true, true, false, false, false}
	for k := range times {
		if got := dealerDate(times, k); got != want[k] {
			t.Errorf("dealerDate(%s) = %v, want %v", times[k].In(mskLoc).Format("Mon 01-02 15:04"), got, want[k])
		}
	}
	if !dealerDate([]time.Time{at("2026-09-15 06:30")}, 0) {
		t.Error("an oldest visible bar before 06:45 must make a dealer date")
	}
}

func TestNoEntryOnDealerWeekday(t *testing.T) {
	p := DefaultParams()
	cases := []struct {
		name string
		from string
		n    int
		buy  bool
	}{
		{"exchange day from 06:45 buys", "2026-09-15 06:45", 13, true},
		{"dealer day from 02:00 does not buy", "2026-09-15 02:00", 32, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lead := flatFrom(tc.from, tc.n)
			if got := lead[len(lead)-1].t; !got.Equal(at("2026-09-15 09:45")) {
				t.Fatalf("fixture: lead ends at %v", got)
			}
			day := append(lead, canonical()...)
			off := len(lead)
			want := []int(nil)
			if tc.buy {
				want = []int{off + canonCross}
			}
			if got := buys(p, day, loudFrom(day, off+canonFrom)); !slices.Equal(got, want) {
				t.Fatalf("buys = %v, want %v", got, want)
			}
		})
	}
}

func TestDealerBarsOutOfVolumeBaseline(t *testing.T) {
	// Monday trades volume 100, the Tuesday holiday (dealer bars from 02:00) volume 1 through its
	// main-session hours. Wednesday's baseline must skip Tuesday and average Monday's 100.
	var bars []bar
	bars = append(bars, flatFrom("2026-09-14 06:45", 69)...)
	mon := len(bars)
	bars = append(bars, flatFrom("2026-09-15 02:00", 88)...)
	tue := len(bars)
	bars = append(bars, flatFrom("2026-09-16 06:45", 14)...) // to 10:00
	md := fixture(bars, time.Time{})
	for j := mon; j < tue; j++ {
		md.Volumes[j] = 1
	}
	ser := barsOf(md)
	for j := mon; j < tue; j++ {
		if inSession(ser, j) {
			t.Fatalf("holiday bar %s is in session", ser.t[j].In(mskLoc).Format("01-02 15:04"))
		}
	}
	k := len(bars) - 1
	if minuteOf(ser.t[k]) != sessionFrom || !inSession(ser, k) {
		t.Fatalf("fixture: last bar %v must be Wednesday's 10:00 session bar", ser.t[k])
	}
	if avg, ok := averageVolume(ser, k, 20); !ok || math.Abs(avg-100) > eps {
		t.Fatalf("baseline = %v (ok %v), want 100 from Monday", avg, ok)
	}
}
