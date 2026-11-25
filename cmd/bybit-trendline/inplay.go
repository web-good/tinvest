package main

import (
	"context"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	m5            = 5 * time.Minute
	checkInterval = 30 * time.Minute
	weekdaysM5    = 5 * 24 * time.Hour / m5
)

type inPlayFilter struct {
	minRangePct       float64
	load              func(context.Context, model.Key, time.Time, time.Time) ([]model.M5Candle, error)
	key               model.Key
	status            map[time.Time]bool
	latest            time.Time
	referenceStart    time.Time
	referenceMean     float64
	referenceComplete bool
}

func previousMonday(at time.Time) time.Time {
	day := at.UTC().Truncate(24 * time.Hour)
	return day.AddDate(0, 0, -7-(int(day.Weekday())+6)%7)
}

func (f *inPlayFilter) reference(ctx context.Context, at time.Time) error {
	start := previousMonday(at)
	if start.Equal(f.referenceStart) {
		return nil
	}
	candles, err := f.load(ctx, f.key, start, start.AddDate(0, 0, 5))
	if err != nil {
		return err
	}
	var volume float64
	for _, c := range candles {
		volume += c.Volume
	}
	f.referenceStart = start
	f.referenceComplete = len(candles) == int(weekdaysM5)
	f.referenceMean = volume / float64(weekdaysM5)
	return nil
}

func inPlay(hour []model.M5Candle, at time.Time, reference, minRangePct float64) bool {
	if len(hour) < 12 {
		return false
	}
	var ranges, volume float64
	for i, c := range hour[len(hour)-12:] {
		if !c.End.Equal(at.Add(time.Duration(i-11)*m5)) || c.Low <= 0 {
			return false
		}
		ranges += 100 * (c.High - c.Low) / c.Low
		volume += c.Volume
	}
	const percentTolerance = 1e-12
	return ranges/12+percentTolerance >= minRangePct && volume/12 > reference
}

// update retains the half-hour decisions needed to replay bars in [from, to].
func (f *inPlayFilter) update(ctx context.Context, from, to time.Time) error {
	first := from.UTC().Truncate(checkInterval)
	last := to.UTC().Truncate(checkInterval)
	if f.status == nil {
		f.status = make(map[time.Time]bool)
	}
	for at := range f.status {
		if at.Before(first) {
			delete(f.status, at)
		}
	}
	if !first.After(f.latest) {
		first = f.latest.Add(checkInterval)
	}
	if first.After(last) {
		return nil
	}
	candles, err := f.load(ctx, f.key, first.Add(-time.Hour), last)
	if err != nil {
		return err
	}
	index := 0
	for at := first; !at.After(last); at = at.Add(checkInterval) {
		if err := f.reference(ctx, at); err != nil {
			return err
		}
		for index < len(candles) && !candles[index].End.After(at) {
			index++
		}
		f.status[at] = f.referenceComplete && inPlay(candles[:index], at, f.referenceMean, f.minRangePct)
		f.latest = at
	}
	return nil
}

func (f *inPlayFilter) allows(at time.Time) bool {
	return f.status[at.UTC().Truncate(checkInterval)]
}
