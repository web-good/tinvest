package core

import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

const eps = 1e-9

// at parses "2006-01-02 15:04" as MSK.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, mskLoc)
	if err != nil {
		panic(err)
	}
	return t
}

type bar struct {
	t          time.Time
	o, h, l, c float64
}

// stepH1 is the only timeframe drop_bounce runs on.
const stepH1 = time.Hour

// path builds consecutive 1h bars from start (MSK): each bar opens at the previous close (the
// first at its own close) and its high/low span the open and the close.
func path(start string, closes []float64) []bar {
	t0 := at(start)
	out := make([]bar, len(closes))
	prev := closes[0]
	for i, c := range closes {
		out[i] = bar{t0.Add(time.Duration(i) * stepH1), prev, max(prev, c), min(prev, c), c}
		prev = c
	}
	return out
}

// steps returns n closes, each pct percent from the previous one, starting from `from` (excluded).
func steps(from float64, n int, pct float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		from *= 1 + pct/100
		out[i] = from
	}
	return out
}

// flatN returns n closes equal to p.
func flatN(n int, p float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = p
	}
	return out
}

// fixtureATR is the daily ATR every fixture carries: its dailies range exactly this much.
const fixtureATR = 2.0

// fixtureUsed is the day's range so far every fixture reports: well inside the default gate
// (DaySpentATR 0.7 · fixtureATR 2 = 1.4).
const fixtureUsed = 0.5

// fixture builds the market data over bars with 20 weekday dailies of range fixtureATR before the
// first bar, and a day extent of fixtureUsed around the last close.
func fixture(bars []bar) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, b.t)
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, 10)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	md.TodayLow = md.Price - fixtureUsed/2
	md.TodayHigh = md.Price + fixtureUsed/2
	withDailies(&md, dailies(bars[0].t, 20, fixtureATR))
	return md
}

// dailies returns n completed weekday daily bars ending the weekday before `before`, each
// spanning `rng` around a close of 100, so their ATR is exactly rng.
func dailies(before time.Time, n int, rng float64) []bar {
	d := time.Date(before.In(mskLoc).Year(), before.In(mskLoc).Month(), before.In(mskLoc).Day(), 0, 0, 0, 0, mskLoc)
	out := make([]bar, 0, n)
	for len(out) < n {
		d = d.AddDate(0, 0, -1)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		out = append([]bar{{d, 100, 100 + rng/2, 100 - rng/2, 100}}, out...)
	}
	return out
}

// withDailies replaces the daily series of md with ds.
func withDailies(md *strategy.MarketData, ds []bar) {
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
	for _, d := range ds {
		md.DailyTimes = append(md.DailyTimes, d.t)
		md.DailyHighs = append(md.DailyHighs, d.h)
		md.DailyLows = append(md.DailyLows, d.l)
		md.DailyCloses = append(md.DailyCloses, d.c)
	}
}

// held decides with a position bought at entry on the bar starting at entryTime, with the daily
// ATR frozen at entry equal to fixtureATR.
func held(p Params, bars []bar, entry float64, entryTime time.Time) model.Signal {
	md := fixture(bars)
	md.Position = &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryTime: entryTime, EntryATR: fixtureATR}
	return NewWithParams("TEST", stepH1, p).Decide(md)
}
