package main

import (
	"context"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

func TestInPlayHalfHoursAndWeekBoundary(t *testing.T) {
	monday := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	start := monday.Add(-checkInterval)
	last := monday.Add(checkInterval)
	var referenceReads []time.Time
	f := inPlayFilter{minRangePct: 1, load: func(_ context.Context, _ model.Key, from, to time.Time) ([]model.M5Candle, error) {
		var bars []model.M5Candle
		if to.Sub(from) > 24*time.Hour {
			referenceReads = append(referenceReads, from)
			if from.Weekday() != time.Monday || to.Sub(from) != 5*24*time.Hour {
				t.Fatalf("unexpected reference window %s to %s", from, to)
			}
		}
		for end := from.Add(m5); !end.After(to); end = end.Add(m5) {
			volume := 20.
			switch {
			case !end.After(monday.AddDate(0, 0, -7)):
				volume = 21
			case !end.After(monday.AddDate(0, 0, -2)):
				volume = 10
			case end.After(monday):
				volume = 0
			}
			bars = append(bars, model.M5Candle{End: end, High: 101, Low: 100, Volume: volume})
		}
		return bars, nil
	}}
	if err := f.update(t.Context(), start, last); err != nil {
		t.Fatal(err)
	}
	if f.allows(monday.Add(-time.Minute)) || !f.allows(monday) || !f.allows(last.Add(-time.Minute)) || f.allows(last) {
		t.Fatal("expected Sunday inactive, Monday midnight active, and equal volume inactive at 00:30")
	}
	if len(referenceReads) != 2 || !referenceReads[0].Equal(monday.AddDate(0, 0, -14)) || !referenceReads[1].Equal(monday.AddDate(0, 0, -7)) {
		t.Fatalf("wrong reference weeks %v", referenceReads)
	}
}

func TestInPlayRequiresCompleteWindowsAndIndividualRanges(t *testing.T) {
	at := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	var bars []model.M5Candle
	for i := 0; i < 12; i++ {
		low := float64(100 + i)
		bars = append(bars, model.M5Candle{End: at.Add(time.Duration(i-11) * m5), High: low * 1.003, Low: low, Volume: 3})
	}
	if inPlay(bars, at, 2, 1) {
		t.Fatal("hour moved >1% but individual candles averaged only 0.3%")
	}
	if !inPlay(bars, at, 2, 0.3) {
		t.Fatal("configured threshold of 0.3% should pass")
	}
	for i := range bars {
		bars[i].High = bars[i].Low * 1.01
	}
	if !inPlay(bars, at, 2, 1) {
		t.Fatal("exactly 1% should pass")
	}
	bars[4].End = bars[4].End.Add(-m5)
	if inPlay(bars, at, 2, 1) {
		t.Fatal("gap should block entries")
	}
	bars[4].End = bars[4].End.Add(m5)
	f := inPlayFilter{minRangePct: 1, load: func(_ context.Context, _ model.Key, from, to time.Time) ([]model.M5Candle, error) {
		if to.Sub(from) > time.Hour {
			return nil, nil
		}
		return bars, nil
	}}
	if err := f.update(t.Context(), at, at); err != nil || f.allows(at) {
		t.Fatalf("missing reference should block entries: %v", err)
	}
}

func TestInPlayEntryAndExit(t *testing.T) {
	e := newEngine(0, 0, 2)
	e.pending = &trade{stop: 99.8, target: 101}
	c := bar{Open: 100, Low: 99.95, High: 100.05, Close: 100}
	e.step(c, time.Hour, false, true)
	if e.pending != nil || e.position != nil {
		t.Fatal("out-of-play open must cancel pending entry")
	}
	e.pending = &trade{stop: 99.8, target: 101}
	e.step(c, time.Hour, true, false)
	if e.position == nil {
		t.Fatal("entry must use status at open")
	}
	c.Low = 89
	e.step(c, time.Hour, false, false)
	if len(e.done) != 1 || e.done[0].exit != 99.8 {
		t.Fatal("stop must execute outside inplay")
	}
}
