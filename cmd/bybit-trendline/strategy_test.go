package main

import (
	"testing"
	"time"
)

func TestBreakoutOnlyAfterConfirmedPeaksAndClose(t *testing.T) {
	d := detector{width: 1}
	var bars []bar
	highs := []float64{8, 10, 8, 9, 8, 9, 9, 9}
	closes := []float64{7, 9, 7, 8, 7, 7, 8, 8}
	for i := range highs {
		bars = append(bars, bar{Open: closes[i] - 2, Low: closes[i] - 2, High: highs[i], Close: closes[i]})
		if got := d.step(bars); got != (i == 6) {
			t.Fatalf("bar %d breakout %v", i, got)
		}
	}
}
func TestAmbiguousBarAndGap(t *testing.T) {
	for _, tc := range []struct{ open, exit float64 }{{100, 95}, {90, 90}, {112, 112}} {
		e := newEngine(0, 0, 2)
		e.position = &trade{entry: 100, stop: 95, target: 110}
		e.step(bar{Open: tc.open, Low: 89, High: 115, Close: 100}, time.Minute, true, true)
		if len(e.done) != 1 || e.done[0].exit != tc.exit {
			t.Fatalf("open %v: %+v", tc.open, e.done)
		}
	}
}

func TestEntryAtPreviousDayHigh(t *testing.T) {
	for _, open := range []float64{99, 100, 101} {
		e := newEngine(0, 0, 2)
		e.entryFloor = 100
		e.pending = &trade{stop: open - 0.1, target: 110}
		e.step(bar{Open: open, High: open, Low: open, Close: open}, time.Minute, true, true)
		if (e.position != nil) != (open >= 100) || e.pending != nil {
			t.Fatalf("unexpected entry state at open %v", open)
		}
	}
}

func TestBreakoutStopAndTarget(t *testing.T) {
	for _, tc := range []struct {
		fast, slow bool
		multiple   float64
	}{{true, false, 1.5}, {false, true, 2}, {true, true, 2.5}} {
		e := newEngine(0, 0.0002, tc.multiple)
		e.bars = []bar{{High: 120, Low: 115}, {High: 112, Low: 110}, {Low: 98}, {Low: 99.9, High: 100, Close: 100}}
		l := line{anchor: peak{3, 100}, slope: -0.01}
		if tc.fast {
			e.fast.active = &l
		}
		if tc.slow {
			e.slow.active = &l
		}
		e.step(bar{Open: 99.95, Low: 99.95, High: 100.02, Close: 100.01}, time.Minute, true, true)
		stop := 99.9 * (1 - stopBuffer)
		if e.pending == nil || e.pending.stop != stop {
			t.Fatalf("unexpected stop: %+v", e.pending)
		}
		e.step(bar{Open: 100, High: 100, Low: 100, Close: 100}, time.Minute, true, true)
		if e.position == nil || e.position.target != e.position.entry+tc.multiple*(e.position.entry-stop) {
			t.Fatalf("unexpected target: %+v", e.position)
		}
	}
}

func TestEntryRiskLimit(t *testing.T) {
	for _, tc := range []struct {
		stop, slip float64
		allowed    bool
	}{{99.8, 0, true}, {99.79, 0, false}, {99.8, 0.0002, false}} {
		e := newEngine(0, tc.slip, 2)
		e.pending = &trade{stop: tc.stop, target: 110}
		e.step(bar{Open: 100, Low: 100, High: 100, Close: 100}, time.Minute, true, true)
		if (e.position != nil) != tc.allowed || e.pending != nil {
			t.Fatalf("unexpected entry for %+v", tc)
		}
	}
}

func TestBreakoutCandleShape(t *testing.T) {
	for _, tc := range []struct {
		name        string
		open, close float64
		want        bool
	}{
		{"65 percent", 95, 108, true}, {"below threshold", 96, 108, false},
		{"opens on line", 100, 119, false}, {"closes on line", 90, 100, false},
		{"no wicks", 90, 110, true}, {"red", 109, 101, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := detector{width: 3, active: &line{anchor: peak{0, 101}, slope: -1}}
			bars := []bar{{Close: 99}, {Open: tc.open, Low: 90, High: max(110, tc.close), Close: tc.close}}
			if got := d.step(bars); got != tc.want {
				t.Fatalf("signal %v, want %v", got, tc.want)
			}
			if tc.close > 100 && d.active != nil {
				t.Fatal("crossed line still active")
			}
		})
	}
}
