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
const step = 5 * time.Minute

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

// path chains bars every 5 minutes from MSK `from`: each opens at the previous close (the first at
// o0) and closes at the next value; highs and lows hug the bodies.
func path(from string, o0 float64, closes ...float64) []bar {
	var out []bar
	o := o0
	for k, c := range closes {
		out = append(out, bar{at(from).Add(time.Duration(k) * step), o, max(o, c), min(o, c), c})
		o = c
	}
	return out
}

// dailyBars returns `days` calendar days of completed dailies ending the day before `before`, all
// closing at 100. Weekdays span 99..101 (PDL 99, Wilder ATR exactly 2); weekend sessions span
// 99.9..100.1, so a weekend daily leaking into the level is visible.
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

// warmBars is how many flat bars warm prepends: more than a weekend of 5m bars, so Friday's main
// session feeds the volume baseline of a Monday.
const warmBars = 1000

// warm prepends warmBars flat bars at 100 ending one bar before bars[0].
func warm(bars []bar) []bar {
	var out []bar
	for k := warmBars; k >= 1; k-- {
		out = append(out, bar{bars[0].t.Add(-time.Duration(k) * step), 100, 100, 100, 100})
	}
	return append(out, bars...)
}

// fixture builds MarketData from bars plus 40 days of dailies before the last bar. Every bar has
// volume 100; the bars from `loud` on have 200 (loud < 0 — none).
func fixture(bars []bar, loud time.Time) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, b.t)
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		v := int64(100)
		if !loud.IsZero() && !b.t.Before(loud) {
			v = 200
		}
		md.Volumes = append(md.Volumes, v)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = dailyBars(md.Times[len(md.Times)-1], 40)
	return md
}

// day is Tuesday 2026-09-15: PDL 99 (Monday's daily), daily ATR 2, zone top 99.4 at ZoneATR 0.2.
const dayStart = "2026-09-15 11:00"

// zoneDay drifts down from 100 into the zone (the 99.4 close is the first bar with low ≤ 99.4)
// without a close below PDL, then climbs.
func zoneDay() []bar {
	return path(dayStart, 100, 99.9, 99.8, 99.7, 99.6, 99.5, 99.4, 99.3, 99.2, 99.25, 99.35, 99.5, 99.7, 99.9, 100.1, 100.3, 100.5)
}

// breakDay drops below PDL (98.8 closes), then climbs.
func breakDay() []bar {
	return path(dayStart, 100, 99.8, 99.6, 99.4, 99.2, 99.0, 98.8, 98.7, 98.75, 98.85, 99.0, 99.2, 99.4, 99.6, 99.8, 100.0)
}

// run is Decide on the warmed bars up to and including day[i] (volumes loud from `loud` on).
func run(p Params, day []bar, i int, loud time.Time) model.Signal {
	return NewWithParams("TEST", step, p).Decide(fixture(warm(day)[:warmBars+i+1], loud))
}

// buys lists the indexes in day whose bar Decide turns into a buy.
func buys(p Params, day []bar, loud time.Time) []int {
	var out []int
	for i := range day {
		if run(p, day, i, loud).Kind == model.SignalBuy {
			out = append(out, i)
		}
	}
	return out
}

// hist is the MACD histogram of the warmed day under p.
func hist(p Params, day []bar) []float64 {
	all := warm(day)
	closes := make([]float64, len(all))
	for i, b := range all {
		closes[i] = b.c
	}
	m, s := indicators.MACD(closes, p.MACDFast, p.MACDSlow, p.MACDSignal)
	h := make([]float64, len(day))
	for i := range day {
		h[i] = m[warmBars+i] - s[warmBars+i]
	}
	return h
}

// firstGreen is the first index ≥ from of day with a green histogram, -1 when none.
func firstGreen(p Params, day []bar, from int) int {
	h := hist(p, day)
	for i := from; i < len(h); i++ {
		if h[i] > 0 {
			return i
		}
	}
	return -1
}

// loudFrom is the start of day[i]: the bars from it on are loud.
func loudFrom(day []bar, i int) time.Time { return day[i].t }

func TestDefaultParams(t *testing.T) {
	want := Params{EntryZone: 1, EntryBreak: 1, ZoneATR: 0.2, MACDFast: 8, MACDSlow: 21, MACDSignal: 9,
		VolumeMult: 1.5, VolBars: 15, StopBars: 5, MinStopATR: 0.2, TargetR: 2, DailyATRPeriod: 14}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams = %+v, want %+v", got, want)
	}
}

// zoneArm is the index of the arming bar in zoneDay (the 99.4 close), breakArm in breakDay (98.8).
const (
	zoneArm  = 5
	breakArm = 5
)

func TestZoneArmEntersOnFirstGreen(t *testing.T) {
	p, day := DefaultParams(), zoneDay()
	g := firstGreen(p, day, zoneArm)
	if g <= zoneArm {
		t.Fatalf("fixture: first green %d must come after the arming bar %d", g, zoneArm)
	}
	got := buys(p, day, loudFrom(day, zoneArm))
	if len(got) != 1 || got[0] != g {
		t.Fatalf("buys = %v, want [%d]", got, g)
	}
	sig := run(p, day, g, loudFrom(day, zoneArm))
	if !sig.FillBy.Equal(day[g].t.Add(step)) || math.Abs(sig.Level-99) > eps || math.Abs(sig.ATR-2) > eps {
		t.Fatalf("FillBy/Level/ATR = %v/%v/%v", sig.FillBy, sig.Level, sig.ATR)
	}
}

func TestBreakArmEntersOnFirstGreen(t *testing.T) {
	p, day := DefaultParams(), breakDay()
	g := firstGreen(p, day, breakArm)
	got := buys(p, day, loudFrom(day, breakArm))
	if g < 0 || len(got) != 1 || got[0] != g {
		t.Fatalf("buys = %v, want [%d]", got, g)
	}
}

func TestEntryFlags(t *testing.T) {
	zoneOff := DefaultParams()
	zoneOff.EntryZone = 0
	if got := buys(zoneOff, zoneDay(), loudFrom(zoneDay(), zoneArm)); len(got) != 0 {
		t.Fatalf("EntryZone 0, zone day: buys = %v, want none", got)
	}
	if got := buys(zoneOff, breakDay(), loudFrom(breakDay(), breakArm)); len(got) != 1 {
		t.Fatalf("EntryZone 0, break day: buys = %v, want one", got)
	}
	breakOff := DefaultParams()
	breakOff.EntryBreak = 0
	// breakDay touches the zone (99.4) before the close below PDL: the setup turns into a break.
	if got := buys(breakOff, breakDay(), loudFrom(breakDay(), 0)); len(got) != 0 {
		t.Fatalf("EntryBreak 0, break day: buys = %v, want none", got)
	}
	if got := buys(breakOff, zoneDay(), loudFrom(zoneDay(), zoneArm)); len(got) != 1 {
		t.Fatalf("EntryBreak 0, zone day: buys = %v, want one", got)
	}
	both := DefaultParams()
	both.EntryZone, both.EntryBreak = 0, 0
	if got := buys(both, zoneDay(), loudFrom(zoneDay(), 0)); len(got) != 0 {
		t.Fatalf("both off: buys = %v, want none", got)
	}
}

func TestBreakTradesSameWithZoneOnOrOff(t *testing.T) {
	on, off := DefaultParams(), DefaultParams()
	off.EntryZone = 0
	day := breakDay()
	a, b := buys(on, day, loudFrom(day, breakArm)), buys(off, day, loudFrom(day, breakArm))
	if len(a) != 1 || len(b) != 1 || a[0] != b[0] {
		t.Fatalf("zone on %v, zone off %v: want the same single buy", a, b)
	}
}

func TestArmBarItselfGreenIsSignal(t *testing.T) {
	// Climb first so the histogram is green, then one bar dips into the zone with a close above PDL.
	day := path(dayStart, 100, 100.2, 100.4, 100.6, 100.8, 101.0, 101.2, 101.4, 101.6)
	day = append(day, bar{day[len(day)-1].t.Add(step), 101.6, 101.8, 99.3, 101.7})
	p, k := DefaultParams(), len(day)-1
	if hist(p, day)[k] <= 0 {
		t.Fatalf("fixture: the arming bar must be green")
	}
	sig := run(p, day, k, loudFrom(day, k))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy on the arming bar", sig.Kind)
	}
}

func TestOneSignalPerArm(t *testing.T) {
	// Into the zone, a climb (the first green burns the setup, later greens stay in the zone and
	// do not buy), then a decline that turns the histogram red and a second dip and climb.
	day := path(dayStart, 100, 99.9, 99.8, 99.7, 99.6, 99.5, 99.4, 99.3, 99.2, 99.25, 99.3, 99.35, 99.38,
		99.36, 99.3, 99.25, 99.2, 99.15, 99.1, 99.05, 99.1, 99.2, 99.4, 99.7, 100.0)
	p := DefaultParams()
	p.VolumeMult = 0 // the machine alone: a loud first setup would raise the second one's baseline
	got := buys(p, day, loudFrom(day, zoneArm))
	g1 := firstGreen(p, day, zoneArm)
	if len(got) != 2 || got[0] != g1 {
		t.Fatalf("buys = %v, want two with the first at %d", got, g1)
	}
	h := hist(p, day)
	red := false
	for j := g1 + 1; j < got[1]; j++ {
		red = red || h[j] <= 0
	}
	if !red {
		t.Fatalf("fixture: no red bar between the buys %v", got)
	}
	for j := g1 + 1; j < got[1]; j++ {
		if h[j] > 0 && h[j-1] > 0 && run(p, day, j, loudFrom(day, zoneArm)).Kind == model.SignalBuy {
			t.Fatalf("bar %d: a second green of the same setup bought", j)
		}
	}
}

func TestVolumeFilter(t *testing.T) {
	day := zoneDay()
	g := firstGreen(DefaultParams(), day, zoneArm)
	cases := []struct {
		name string
		mult float64
		loud time.Time
		want bool
	}{
		{"window 2× base passes 1.5", 1.5, loudFrom(day, zoneArm), true},
		{"window 2× base fails 2.5", 2.5, loudFrom(day, zoneArm), false},
		{"quiet window fails", 1.5, time.Time{}, false},
		{"filter off buys on a quiet window", 0, time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			p.VolumeMult = tc.mult
			if got := run(p, day, g, tc.loud).Kind == model.SignalBuy; got != tc.want {
				t.Fatalf("buy = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNoMainSessionBarsInWindowFails(t *testing.T) {
	// The same day shifted to the evening session: the window has no main-session bars.
	day := path("2026-09-15 19:30", 100, 99.9, 99.8, 99.7, 99.6, 99.5, 99.4, 99.3, 99.2, 99.25, 99.35, 99.5, 99.7, 99.9, 100.1)
	p := DefaultParams()
	if got := buys(p, day, loudFrom(day, zoneArm)); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
}

func TestNoVolumesNoEntry(t *testing.T) {
	day := zoneDay()
	g := firstGreen(DefaultParams(), day, zoneArm)
	md := fixture(warm(day)[:warmBars+g+1], loudFrom(day, zoneArm))
	md.Volumes = nil
	if sig := NewWithParams("TEST", step, DefaultParams()).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatalf("bought without volumes")
	}
}

func TestStopLowestOfFiveBars(t *testing.T) {
	p := DefaultParams()
	p.MinStopATR = 0.01
	day := zoneDay()
	g := firstGreen(p, day, zoneArm)
	low := math.Inf(1)
	for j := g - 5; j < g; j++ {
		low = min(low, day[j].l)
	}
	sig := run(p, day, g, loudFrom(day, zoneArm))
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-low) > eps {
		t.Fatalf("StopLoss = %v, want the 5-bar low %v", sig.StopLoss, low)
	}
	if want := day[g].c + 2*(day[g].c-low); math.Abs(sig.TakeProfit-want) > eps {
		t.Fatalf("TakeProfit = %v, want %v", sig.TakeProfit, want)
	}
}

func TestStopLoweredToMinStopATR(t *testing.T) {
	p := DefaultParams()
	p.MinStopATR = 1 // 2 points: deeper than any low of the day
	day := zoneDay()
	g := firstGreen(p, day, zoneArm)
	sig := run(p, day, g, loudFrom(day, zoneArm))
	if want := day[g].c - 2; sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-want) > eps {
		t.Fatalf("StopLoss = %v, want close − 1 ATR = %v", sig.StopLoss, want)
	}
}

func TestShortHistoryNoEntry(t *testing.T) {
	day := zoneDay()
	g := firstGreen(DefaultParams(), day, zoneArm)
	md := fixture(warm(day)[warmBars-50:warmBars+g+1], loudFrom(day, zoneArm))
	if sig := NewWithParams("TEST", step, DefaultParams()).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatalf("bought on %d bars", len(md.Closes))
	}
	md = fixture(warm(day)[:warmBars+g+1], loudFrom(day, zoneArm))
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = md.DailyHighs[:5], md.DailyLows[:5], md.DailyCloses[:5], md.DailyTimes[:5]
	if sig := NewWithParams("TEST", step, DefaultParams()).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatalf("bought without a daily ATR")
	}
}

func TestLastBarOfDateNoEntry(t *testing.T) {
	// The zone day placed so the first green is the 23:55 bar: the buy bar is on the next date.
	p := DefaultParams()
	p.VolumeMult = 0
	day := zoneDay()
	g := firstGreen(p, day, zoneArm)
	shift := at("2026-09-15 23:55").Sub(day[g].t)
	for i := range day {
		day[i].t = day[i].t.Add(shift)
	}
	if sig := run(p, day, g, time.Time{}); sig.Kind == model.SignalBuy {
		t.Fatalf("bought on the last bar of the date")
	}
}

func TestWeekendBarNoEntry(t *testing.T) {
	p := DefaultParams()
	p.VolumeMult = 0
	day := zoneDay()
	shift := at("2026-09-19 11:00").Sub(day[0].t) // Saturday
	for i := range day {
		day[i].t = day[i].t.Add(shift)
	}
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("buys on Saturday = %v", got)
	}
}

func TestWeekendDailyIgnored(t *testing.T) {
	md := fixture(warm(path("2026-09-14 11:00", 100, 100)), time.Time{}) // Monday: Sunday is the last daily
	pdl, ok := NewWithParams("TEST", step, DefaultParams()).pdl(md)
	if !ok || math.Abs(pdl-99) > eps {
		t.Fatalf("pdl = %v, %v; want Friday's 99", pdl, ok)
	}
}

// pos is an open position filled at f with the signal's stop and a daily ATR of 2.
func pos(f, stop float64) *strategy.Position {
	return &strategy.Position{PurchasePrice: f, StopLoss: stop, EntryATR: 2, Quantity: 1}
}

// manageOn is Decide with the position on one bar.
func manageOn(p *strategy.Position, b bar) model.Signal {
	md := strategy.MarketData{Opens: []float64{b.o}, Highs: []float64{b.h}, Lows: []float64{b.l}, Closes: []float64{b.c},
		Times: []time.Time{b.t}, Position: p, Price: b.c}
	return NewWithParams("TEST", step, DefaultParams()).Decide(md)
}

func TestExitLevelsOffTheFill(t *testing.T) {
	cases := []struct {
		name               string
		fill, sigStop      float64
		wantStop, wantTake float64
	}{
		{"low deeper than 0.2 ATR keeps the low", 100, 99, 99, 102},
		{"fill lower than the signal close lowers the stop", 99.2, 99, 98.8, 100},
		{"lowered signal stop follows a lower fill", 99.5, 99.6, 99.1, 100.3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stop, take, ok := NewWithParams("TEST", step, DefaultParams()).exitLevels(pos(tc.fill, tc.sigStop))
			if !ok || math.Abs(stop-tc.wantStop) > eps || math.Abs(take-tc.wantTake) > eps {
				t.Fatalf("stop/target = %v/%v (%v), want %v/%v", stop, take, ok, tc.wantStop, tc.wantTake)
			}
		})
	}
}

func TestExitsSLThenTP(t *testing.T) {
	when := at("2026-09-16 11:00")
	if sig := manageOn(pos(100, 99), bar{when, 100, 102.5, 98.9, 100}); sig.Kind != model.SignalSell || sig.Reason != "SL" || math.Abs(sig.StopLoss-99) > eps {
		t.Fatalf("SL first: %+v", sig)
	}
	if sig := manageOn(pos(100, 99), bar{when, 100, 102.1, 99.5, 102}); sig.Kind != model.SignalSell || sig.Reason != "TP" || math.Abs(sig.TakeProfit-102) > eps {
		t.Fatalf("TP: %+v", sig)
	}
}

func TestStopHitOnFirstBarAfterFill(t *testing.T) {
	// A gap down through the stop on the fill bar exits on that very bar.
	if sig := manageOn(pos(98.5, 99), bar{at("2026-09-16 10:00"), 98.5, 98.6, 98.0, 98.2}); sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("gap through the stop: %+v", sig)
	}
}

func TestCarriesOvernight(t *testing.T) {
	if sig := manageOn(pos(100, 99), bar{at("2026-09-16 07:00"), 100.5, 100.6, 100.4, 100.5}); sig.Kind == model.SignalSell {
		t.Fatalf("exited on a quiet bar of the next date: %+v", sig)
	}
}

func TestLookbackCoversMACDWarmup(t *testing.T) {
	s := NewWithParams("TEST", step, DefaultParams())
	if s.Lookback() < 3*(21+9) || s.Lookback() < 3*dayBars {
		t.Fatalf("Lookback = %d", s.Lookback())
	}
}
