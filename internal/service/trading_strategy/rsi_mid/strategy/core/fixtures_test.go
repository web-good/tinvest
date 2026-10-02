package core

import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// mondayNoon is a weekday bar time (2026-06-01 is a Monday).
var mondayNoon = time.Date(2026, 6, 1, 12, 0, 0, 0, mskLoc)

// saturdayNoon is a weekend bar time (2026-05-30 is a Saturday).
var saturdayNoon = time.Date(2026, 5, 30, 12, 0, 0, 0, mskLoc)

// dailyWidth is the weekday daily range of the fixture; with a flat close the daily ATR over
// weekday bars is exactly this value.
const dailyWidth = 2.0

// dailyBars builds `days` consecutive calendar days ending the day BEFORE `before` (MSK),
// oldest-first. Every bar closes at 100; a weekday bar spans `w`, a weekend bar spans `we`, so a
// test can prove weekend sessions never reach the ATR.
func dailyBars(before time.Time, days int, w, we float64) (highs, lows, closes []float64, times []time.Time) {
	b := before.In(mskLoc)
	start := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, mskLoc).AddDate(0, 0, -days)
	const price = 100.0
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		width := w
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			width = we
		}
		highs = append(highs, price+width/2)
		lows = append(lows, price-width/2)
		closes = append(closes, price)
		times = append(times, d.Add(10*time.Hour))
	}
	return highs, lows, closes, times
}

// fixtureWithDaily builds 30-minute MarketData from closes, the LAST bar stamped at `last`,
// with a 0.3% high/low envelope and 40 calendar days of completed dailies whose weekday ATR is
// exactly `width` (weekend dailies are ten times narrower).
func fixtureWithDaily(closes []float64, last time.Time, width float64) strategy.MarketData {
	n := len(closes)
	start := last.Add(-time.Duration(n-1) * 30 * time.Minute)
	md := strategy.MarketData{
		Highs:  make([]float64, n),
		Lows:   make([]float64, n),
		Closes: append([]float64(nil), closes...),
		Times:  make([]time.Time, n),
	}
	for i, c := range closes {
		md.Highs[i] = c * 1.003
		md.Lows[i] = c * 0.997
		md.Times[i] = start.Add(time.Duration(i) * 30 * time.Minute)
	}
	if n > 0 {
		md.Price = closes[n-1]
	}
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = dailyBars(last, 40, width, width/10)
	return md
}

func fixture(closes []float64, last time.Time) strategy.MarketData {
	return fixtureWithDaily(closes, last, dailyWidth)
}

// rsiAt returns the RSI(period) reading of the last bar, measured over the trailing `window`
// closes (the whole series when window <= 0 or longer than the series). The backtest engine
// hands Decide exactly the last Lookback() bars, so a fixture meant for the engine measures over
// that same window to land the cross on the designed bar.
func rsiAt(closes []float64, period, window int) float64 {
	src := closes
	if window > 0 && window < len(src) {
		src = src[len(src)-window:]
	}
	r := indicators.RSISeries(src, period)
	return r[len(r)-1]
}

// crossCloses builds a history whose LAST bar is an RSI(period) cross UP through `level`:
// `lead` bars drifting by `drift` per bar (every 4th bar a counter-move of the opposite sign),
// then -0.3% bars until RSI reads below `level`, then
// +0.3% bars until it reads above `level`. Every bar is measured as it is appended, so the shape
// holds for any period. The lead carries counter-move bars because indicators.RSISeries returns
// exactly 50 while avgLoss == 0: a pure monotonic lead would read 50 on every engine window, and
// the first dip bar (tiny positive avgLoss, large avgGain) would jump RSI to ~79 and fire a false
// cross up through 50 there. Real data always has losing bars, so avgLoss > 0 from the seed on.
func crossCloses(lead int, drift float64, period int, level float64, window int) []float64 {
	out := make([]float64, 0, lead+64)
	p := 100.0
	for i := 0; i < lead; i++ {
		if i%4 == 3 {
			p *= 1 - drift
		} else {
			p *= 1 + drift
		}
		out = append(out, p)
	}
	for guard := 0; rsiAt(out, period, window) >= level; guard++ {
		if guard > 500 {
			panic("crossCloses: RSI never dropped below the level")
		}
		p *= 0.997
		out = append(out, p)
	}
	for guard := 0; rsiAt(out, period, window) <= level; guard++ {
		if guard > 500 {
			panic("crossCloses: RSI never rose above the level")
		}
		p *= 1.003
		out = append(out, p)
	}
	return out
}

// upCross is the canonical entry fixture: a 400-bar uptrend (+0.08% per bar), a dip and a
// bounce whose last bar is an RSI(14) cross up through 50 with the close above EMA(100).
func upCross() []float64 { return crossCloses(400, 0.0008, 14, midLine, 0) }

// downCross is the same cross after a 400-bar DOWNtrend: the close sits below EMA(100).
func downCross() []float64 { return crossCloses(400, -0.0008, 14, midLine, 0) }
