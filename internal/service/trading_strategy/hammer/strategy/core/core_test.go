package core

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

const eps = 1e-9

// step is the bar length the strategy trades on.
const step = time.Hour

// at parses "2006-01-02 15:04" as MSK.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, mskLoc)
	if err != nil {
		panic(err)
	}
	return t
}

type bar struct {
	o, h, l, c float64
}

// warmBars alternate 100 / 100.2 so RSI starts near 50.
const warmBars = 60

// fall is five hourly drops by 0.4 from 100.2: RSI(5) ends deep under 30.
var fall = []float64{99.8, 99.4, 99.0, 98.6, 98.2}

// rise is five hourly rises by 0.4 from 100.2: RSI(5) ends far above 70.
var rise = []float64{100.6, 101.0, 101.4, 101.8, 102.2}

// hammerAfterFall is a hammer closing a little under the last fall close: body 0.05 at the top of
// a 1.2 range.
var hammerAfterFall = bar{o: 98.1, h: 98.2, l: 97.0, c: 98.15}

// bars chains warmBars alternating bars, then bars with the given closes (each opening at the
// previous close, highs and lows hugging the bodies), then last.
func bars(closes []float64, last bar) []bar {
	var out []bar
	prev := 100.0
	for i := 0; i < warmBars; i++ {
		c := 100.0
		if i%2 == 1 {
			c = 100.2
		}
		out = append(out, bar{prev, max(prev, c), min(prev, c), c})
		prev = c
	}
	for _, c := range closes {
		out = append(out, bar{prev, max(prev, c), min(prev, c), c})
		prev = c
	}
	return append(out, last)
}

// dailyBars returns `days` calendar days of completed dailies ending the day before `before`, all
// closing at 100. Weekdays span 99..101 (range 2, Wilder ATR exactly 2); weekend sessions span
// 99.9..100.1, so a weekend daily leaking into the levels is visible.
func dailyBars(before time.Time, days int) (highs, lows, closes []float64, times []time.Time) {
	b := before.In(mskLoc)
	start := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, mskLoc).AddDate(0, 0, -days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		half := 1.0
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			half = 0.1
		}
		highs = append(highs, 100+half)
		lows = append(lows, 100-half)
		closes = append(closes, 100)
		times = append(times, d.Add(10*time.Hour))
	}
	return highs, lows, closes, times
}

// fixture builds MarketData from bars whose last bar starts at `last`, hourly back from it, plus
// `days` days of dailies before it.
func fixture(bs []bar, last time.Time, days int) strategy.MarketData {
	var md strategy.MarketData
	n := len(bs)
	for i, b := range bs {
		md.Times = append(md.Times, last.Add(-time.Duration(n-1-i)*step))
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, 100)
	}
	md.Price = md.Closes[n-1]
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = dailyBars(last, days)
	return md
}

// wednesday is a regular weekday hour: the buy bar 13:00 is on the same date.
const wednesday = "2026-09-16 12:00"

func decide(p Params, md strategy.MarketData) model.Signal {
	return NewWithParams("TEST", step, p).Decide(md)
}

// lastRSI is RSI(period) of the last two closes.
func lastRSI(md strategy.MarketData, period int) (prev, cur float64) {
	r := indicators.RSISeries(md.Closes, period)
	return r[len(r)-2], r[len(r)-1]
}

func TestIsHammer(t *testing.T) {
	cases := []struct {
		name string
		b    bar
		want bool
	}{
		{"body in the upper third", bar{99.8, 100, 97, 99.9}, true},
		{"red body in the upper third", bar{99.9, 100, 97, 99.8}, true},
		{"zero range", bar{100, 100, 100, 100}, false},
		{"body in the middle", bar{98.4, 100, 97, 98.6}, false},
		{"body over a third of the range", bar{98.5, 100, 97, 99.9}, false},
	}
	for _, tc := range cases {
		if got := isHammer(tc.b.o, tc.b.h, tc.b.l, tc.b.c); got != tc.want {
			t.Errorf("%s: isHammer = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestEntryOnHammerInZone(t *testing.T) {
	last := at(wednesday)
	md := fixture(bars(fall, hammerAfterFall), last, 40)
	if _, cur := lastRSI(md, 5); cur >= 30 {
		t.Fatalf("fixture: hammer RSI(5) %.1f, want < 30", cur)
	}
	sig := decide(DefaultParams(), md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want buy", sig.Kind)
	}
	c := hammerAfterFall.c
	if !sig.FillBy.Equal(last.Add(step)) {
		t.Errorf("FillBy = %v, want %v", sig.FillBy, last.Add(step))
	}
	if math.Abs(sig.StopLoss-(c-0.5*2)) > eps || math.Abs(sig.TakeProfit-(c+0.5*2)) > eps || math.Abs(sig.ATR-2) > eps {
		t.Errorf("stop %.4f target %.4f atr %.4f, want %.4f %.4f 2", sig.StopLoss, sig.TakeProfit, sig.ATR, c-1, c+1)
	}
}

func TestNoEntryWhenRSIAboveZone(t *testing.T) {
	h := bar{o: 102.1, h: 102.3, l: 100.8, c: 102.25}
	md := fixture(bars(rise, h), at(wednesday), 40)
	if !isHammer(h.o, h.h, h.l, h.c) {
		t.Fatal("fixture: not a hammer")
	}
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
}

func TestNoEntryWhenRSIInZoneOnlyBefore(t *testing.T) {
	h := bar{o: 99.6, h: 99.75, l: 97.0, c: 99.7}
	md := fixture(bars(fall, h), at(wednesday), 40)
	if prev, cur := lastRSI(md, 5); prev >= 30 || cur < 30 {
		t.Fatalf("fixture: RSI prev %.1f cur %.1f, want prev < 30 ≤ cur", prev, cur)
	}
	if !isHammer(h.o, h.h, h.l, h.c) {
		t.Fatal("fixture: not a hammer")
	}
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
}

func TestNoEntryWhenBuyBarIsNextDate(t *testing.T) {
	md := fixture(bars(fall, hammerAfterFall), at("2026-09-16 23:00"), 40)
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
}

func TestNoEntryOnWeekend(t *testing.T) {
	md := fixture(bars(fall, hammerAfterFall), at("2026-09-19 12:00"), 40)
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
}

func TestMondayUsesFridayRange(t *testing.T) {
	md := fixture(bars(fall, hammerAfterFall), at("2026-09-21 12:00"), 40)
	sig := decide(DefaultParams(), md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want buy", sig.Kind)
	}
	if want := hammerAfterFall.c + 0.5*2; math.Abs(sig.TakeProfit-want) > eps {
		t.Fatalf("target %.4f, want %.4f (Friday's range 2, not Sunday's 0.2)", sig.TakeProfit, want)
	}
}

func TestNoEntryWithoutDailies(t *testing.T) {
	last := at(wednesday)
	if sig := decide(DefaultParams(), fixture(bars(fall, hammerAfterFall), last, 10)); sig.Kind != model.SignalNone {
		t.Fatalf("10 days of dailies: kind = %v, want none", sig.Kind)
	}
	p := DefaultParams()
	p.StopDailyATR = 0
	if sig := decide(p, fixture(bars(fall, hammerAfterFall), last, 40)); sig.Kind != model.SignalNone {
		t.Fatalf("StopDailyATR 0: kind = %v, want none", sig.Kind)
	}
}
