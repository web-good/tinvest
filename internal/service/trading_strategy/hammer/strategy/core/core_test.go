package core

import (
	"math"
	"strings"
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
	var warm []float64
	for i := 0; i < warmBars; i++ {
		c := 100.0
		if i%2 == 1 {
			c = 100.2
		}
		warm = append(warm, c)
	}
	return chain(append(warm, closes...), last)
}

// chain turns closes into bars opening at the previous close (the first at 100), highs and lows
// hugging the bodies, then appends last.
func chain(closes []float64, last bar) []bar {
	var out []bar
	prev := 100.0
	for _, c := range closes {
		out = append(out, bar{prev, max(prev, c), min(prev, c), c})
		prev = c
	}
	return append(out, last)
}

// noTrend is the owner's rules with the EMA trend filter off, for the tests of the other gates
// (their short fixtures cannot warm EMA 200).
func noTrend() Params {
	p := DefaultParams()
	p.TrendEMA = 0
	return p
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
// `days` days of dailies before it. Every bar trades 100 lots, the last one 200: it passes the
// default volume filter (above 1.5 × 100).
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
	md.Volumes[n-1] = 200
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
		{"doji: zero body with a long lower tail", bar{99.9, 100, 97, 99.9}, false},
		{"body bottom under the middle", bar{98.4, 100, 97, 98.6}, false},
		{"tail under two bodies", bar{98.6, 100, 97, 99.9}, false},
		{"GAZP 07-16 12:00: tail exactly two bodies, body under the upper third", bar{85.27, 85.34, 83.98, 84.84}, true},
		{"body just above the middle", bar{98.6, 98.9, 97, 98.8}, true},
		{"NVTK 10-19 12:00: upper tail 1.5 bodies", bar{1141.9, 1145.4, 1137.4, 1143.3}, false},
		{"upper tail exactly one body", bar{99.7, 100, 97, 99.85}, true},
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
	sig := decide(noTrend(), md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want buy", sig.Kind)
	}
	c := hammerAfterFall.c
	if !sig.FillBy.Equal(last.Add(step)) {
		t.Errorf("FillBy = %v, want %v", sig.FillBy, last.Add(step))
	}
	// stop = low 97 − 0.05 × ATR 2 = 96.9; target = close 98.15 + 2 × (98.15 − 96.9) = 100.65
	if math.Abs(sig.StopLoss-96.9) > eps || math.Abs(sig.TakeProfit-(c+2*(c-96.9))) > eps || math.Abs(sig.ATR-2) > eps {
		t.Errorf("stop %.4f target %.4f atr %.4f, want 96.9 %.4f 2", sig.StopLoss, sig.TakeProfit, sig.ATR, c+2*(c-96.9))
	}
}

func TestNoEntryWhenRSIAboveZone(t *testing.T) {
	h := bar{o: 102.1, h: 102.3, l: 100.8, c: 102.25}
	md := fixture(bars(rise, h), at(wednesday), 40)
	if !isHammer(h.o, h.h, h.l, h.c) {
		t.Fatal("fixture: not a hammer")
	}
	if sig := decide(noTrend(), md); sig.Kind != model.SignalNone {
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
	if sig := decide(noTrend(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
}

func TestNoEntryWhenBuyBarIsNextDate(t *testing.T) {
	md := fixture(bars(fall, hammerAfterFall), at("2026-09-16 23:00"), 40)
	if sig := decide(noTrend(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
}

func TestNoEntryOnWeekend(t *testing.T) {
	md := fixture(bars(fall, hammerAfterFall), at("2026-09-19 12:00"), 40)
	if sig := decide(noTrend(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
}

func TestMondayATRSkipsWeekendDailies(t *testing.T) {
	md := fixture(bars(fall, hammerAfterFall), at("2026-09-21 12:00"), 40)
	// 66 bars back from Monday noon hold only two main-session bars: the volume filter is off
	p := noTrend()
	p.VolumeMult = 0
	p.StopBufferATR = 1
	sig := decide(p, md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want buy", sig.Kind)
	}
	if math.Abs(sig.ATR-2) > eps || math.Abs(sig.StopLoss-95) > eps {
		t.Fatalf("ATR %.4f stop %.4f, want 2 and 95 (weekday ranges 2, the weekend's 0.2 left out)", sig.ATR, sig.StopLoss)
	}
}

func TestNoEntryWithoutDailies(t *testing.T) {
	last := at(wednesday)
	if sig := decide(noTrend(), fixture(bars(fall, hammerAfterFall), last, 10)); sig.Kind != model.SignalNone {
		t.Fatalf("10 days of dailies: kind = %v, want none", sig.Kind)
	}
	p := noTrend()
	p.StopBufferATR = 60 // the stop would fall under zero
	if sig := decide(p, fixture(bars(fall, hammerAfterFall), last, 40)); sig.Kind != model.SignalNone {
		t.Fatalf("stop under zero: kind = %v, want none", sig.Kind)
	}
}

// held is MarketData over bs with an open position filled at fill whose signal froze stop and
// target (EntryATR 2).
func held(bs []bar, fill, stop, target float64) strategy.MarketData {
	md := fixture(bs, at(wednesday), 40)
	md.Position = &strategy.Position{PurchasePrice: fill, Quantity: 1, StopLoss: stop, TakeProfit: target, EntryATR: 2,
		EntryTime: at(wednesday).Add(-3 * step)}
	return md
}

func TestStopBeforeTarget(t *testing.T) {
	md := held(bars(fall, bar{98.2, 101.5, 98.9, 99}), 100, 99, 101)
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("kind %v reason %q, want sell SL", sig.Kind, sig.Reason)
	}
}

func TestTargetExit(t *testing.T) {
	// stop 99, target 100 + 2 × (100 − 99) = 102
	md := held(bars(fall, bar{99.6, 102.2, 99.5, 99.7}), 100, 99, 101)
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("kind %v reason %q, want sell TP", sig.Kind, sig.Reason)
	}
}

// lastHist is the MACD histogram of the default periods over md's closes.
func lastHist(t *testing.T, md strategy.MarketData) []float64 {
	t.Helper()
	h := NewWithParams("TEST", step, DefaultParams()).macdHist(md.Closes)
	if h == nil {
		t.Fatal("fixture: MACD not computed")
	}
	return h
}

// fadedBounce is the rise with a drop to 99 on the last bar: the histogram is green through the
// rise (the hammer and the entry bar included) and red on the last bar.
func fadedBounce() []bar { return bars(rise, bar{102.2, 102.2, 99, 99}) }

func TestMACDExitAfterGreen(t *testing.T) {
	md := held(fadedBounce(), 101, 50, 200)
	h := lastHist(t, md)
	if n := len(h); h[n-1] >= 0 || h[n-3] <= 0 {
		t.Fatalf("fixture: histogram entry %.4f last %.4f, want green then red", h[n-3], h[n-1])
	}
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalSell || sig.Reason != "MACD" {
		t.Fatalf("kind %v reason %q, want sell MACD", sig.Kind, sig.Reason)
	}
}

func TestNoMACDExitWhileRedSinceHammer(t *testing.T) {
	md := held(bars(fall, bar{98.2, 98.2, 97.8, 97.8}), 98.6, 50, 200)
	h := lastHist(t, md)
	for j := len(h) - 4; j < len(h); j++ {
		if h[j] >= 0 {
			t.Fatalf("fixture: histogram %.4f at bar %d from the hammer on, want red throughout", h[j], j)
		}
	}
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind %v reason %q, want none", sig.Kind, sig.Reason)
	}
}

func TestRSIHoldDropsTarget(t *testing.T) {
	// fill 101, stop 100: target 103; the rise closed RSI(5) above 60 since the entry
	md := held(bars(rise, bar{102.2, 105, 102.2, 102.6}), 101, 100, 102)
	if h := lastHist(t, md); h[len(h)-1] <= 0 {
		t.Fatalf("fixture: histogram %.4f, want green", h[len(h)-1])
	}
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind %v reason %q, want none (target off after RSI > 60)", sig.Kind, sig.Reason)
	}
	p := DefaultParams()
	p.RSIHold = 0
	if sig := decide(p, md); sig.Reason != "TP" {
		t.Fatalf("RSIHold 0: reason %q, want TP", sig.Reason)
	}
}

func TestRSIHoldCrossOnCurrentBarKeepsTarget(t *testing.T) {
	// fill 98.6, stop 98: target 99.8; only the last close jumps RSI(5) above 60
	md := held(bars(fall, bar{98.2, 101, 98.2, 100.8}), 98.6, 98, 99)
	if prev, cur := lastRSI(md, 5); prev >= 60 || cur <= 60 {
		t.Fatalf("fixture: RSI prev %.1f cur %.1f, want the cross on the last close", prev, cur)
	}
	if sig := decide(DefaultParams(), md); sig.Reason != "TP" {
		t.Fatalf("reason %q, want TP", sig.Reason)
	}
}

func TestRSIHoldExitReason(t *testing.T) {
	sig := decide(DefaultParams(), held(fadedBounce(), 101, 50, 200))
	if sig.Reason != "MACD" || !strings.Contains(sig.ExitReason, "RSI(5) > 60") {
		t.Fatalf("reason %q %q, want MACD after RSI > 60", sig.Reason, sig.ExitReason)
	}
}

func TestExitsSwitchOff(t *testing.T) {
	p := DefaultParams()
	p.TargetR = 0
	if sig := decide(p, held(bars(fall, bar{98.2, 150, 98.1, 98.3}), 98, 97, 101)); sig.Kind != model.SignalNone {
		t.Fatalf("no target: kind %v reason %q, want none", sig.Kind, sig.Reason)
	}
	if sig := decide(p, held(bars(fall, bar{98.2, 150, 96.5, 98.3}), 98, 97, 101)); sig.Reason != "SL" {
		t.Fatalf("no target: reason %q, want SL", sig.Reason)
	}
	p = DefaultParams()
	p.MACDFast = 0
	if sig := decide(p, held(fadedBounce(), 101, 50, 200)); sig.Kind != model.SignalNone {
		t.Fatalf("MACDFast 0: kind %v reason %q, want none", sig.Kind, sig.Reason)
	}
}

// trendBars is 300 alternating closes around level (EMA 200 settles there), the five fall
// drops shifted onto it, then hammerAfterFall shifted the same way.
func trendBars(level float64) []bar {
	var closes []float64
	for i := 0; i < 300; i++ {
		c := level
		if i%2 == 1 {
			c = level + 0.2
		}
		closes = append(closes, c)
	}
	d := level - 100
	for _, c := range fall {
		closes = append(closes, c+d)
	}
	h := hammerAfterFall
	return chain(closes, bar{h.o + d, h.h + d, h.l + d, h.c + d})
}

func TestTrendFilterBlocksBelowEMA(t *testing.T) {
	// a flat 100 then a dip to 98.15: the hammer closes under EMA 200 (≈ 100)
	md := fixture(trendBars(100), at(wednesday), 40)
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
	if sig := decide(noTrend(), md); sig.Kind != model.SignalBuy {
		t.Fatalf("filter off: kind = %v, want buy", sig.Kind)
	}
}

func TestTrendFilterPassesAboveEMA(t *testing.T) {
	// 300 bars rising from 90 to 100, then the dip: EMA 200 lags far under the hammer's close
	var closes []float64
	for i := 0; i < 300; i++ {
		closes = append(closes, 90+10*float64(i)/299)
	}
	closes = append(closes, fall...)
	md := fixture(chain(closes, hammerAfterFall), at(wednesday), 40)
	if _, cur := lastRSI(md, 5); cur >= 30 {
		t.Fatalf("fixture: hammer RSI(5) %.1f, want < 30", cur)
	}
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want buy", sig.Kind)
	}
}

func TestTrendFilterNeedsWarmEMA(t *testing.T) {
	// 66 bars cannot warm EMA 200: no entry rather than "above a zero EMA"
	md := fixture(bars(fall, hammerAfterFall), at(wednesday), 40)
	if sig := decide(DefaultParams(), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
}

// halfHour lays the same bars on 30-minute steps.
func halfHour(bs []bar, last time.Time, days int) strategy.MarketData {
	md := fixture(bs, last, days)
	n := len(md.Times)
	for i := range md.Times {
		md.Times[i] = last.Add(-time.Duration(n-1-i) * 30 * time.Minute)
	}
	return md
}

func TestMinutes30EntryAndLabel(t *testing.T) {
	last := at(wednesday)
	sig := NewWithParams("TEST", 30*time.Minute, noTrend()).Decide(halfHour(bars(fall, hammerAfterFall), last, 40))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want buy", sig.Kind)
	}
	if !sig.FillBy.Equal(last.Add(30 * time.Minute)) {
		t.Errorf("FillBy = %v, want %v", sig.FillBy, last.Add(30*time.Minute))
	}
	if !strings.HasPrefix(sig.EntryReason, "молот 30m ") {
		t.Errorf("reason %q, want the 30m label", sig.EntryReason)
	}
}

func TestMinutes30LastBarOfDay(t *testing.T) {
	s := NewWithParams("TEST", 30*time.Minute, noTrend())
	if sig := s.Decide(halfHour(bars(fall, hammerAfterFall), at("2026-09-16 23:00"), 40)); sig.Kind != model.SignalBuy {
		t.Fatalf("23:00 on 30m: kind = %v, want buy (the 23:30 bar is the same date)", sig.Kind)
	}
	if sig := s.Decide(halfHour(bars(fall, hammerAfterFall), at("2026-09-16 23:30"), 40)); sig.Kind != model.SignalNone {
		t.Fatalf("23:30 on 30m: kind = %v, want none", sig.Kind)
	}
}

func TestVolumeFilter(t *testing.T) {
	md := fixture(bars(fall, hammerAfterFall), at(wednesday), 40)
	n := len(md.Volumes)
	cases := []struct {
		name string
		vol  int64
		want model.SignalKind
	}{
		{"at 1.5 × the average", 150, model.SignalNone},
		{"above 1.5 × the average", 151, model.SignalBuy},
	}
	for _, tc := range cases {
		md.Volumes[n-1] = tc.vol
		if sig := decide(noTrend(), md); sig.Kind != tc.want {
			t.Errorf("%s: kind = %v, want %v", tc.name, sig.Kind, tc.want)
		}
	}
	p := noTrend()
	p.VolumeMult = 0
	md.Volumes[n-1] = 100
	if sig := decide(p, md); sig.Kind != model.SignalBuy {
		t.Fatalf("filter off: kind = %v, want buy", sig.Kind)
	}
}

func TestVolumeAverageSkipsOffSessionBars(t *testing.T) {
	// the evening, night and morning bars trade 1000 lots: counted in, they would lift the average
	// over 151 / 1.5
	md := fixture(bars(fall, hammerAfterFall), at(wednesday), 40)
	n := len(md.Volumes)
	for i, tm := range md.Times[:n-1] {
		if h := tm.In(mskLoc).Hour(); h < 10 || h >= 19 {
			md.Volumes[i] = 1000
		}
	}
	md.Volumes[n-1] = 151
	if sig := decide(noTrend(), md); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want buy", sig.Kind)
	}
}

func TestVolumeFilterNeedsHistory(t *testing.T) {
	p := noTrend()
	p.VolBars = 500
	if sig := decide(p, fixture(bars(fall, hammerAfterFall), at(wednesday), 40)); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none", sig.Kind)
	}
}

func TestLevelsFromFill(t *testing.T) {
	s := NewWithParams("TEST", step, DefaultParams())
	// the stop stays at the hammer level; the target is 2 R off the fill
	stop, target, ok := s.exitLevels(&strategy.Position{PurchasePrice: 98.5, StopLoss: 96.9, TakeProfit: 100.65, EntryATR: 2})
	if !ok || math.Abs(stop-96.9) > eps || math.Abs(target-101.7) > eps {
		t.Fatalf("exitLevels = %.4f %.4f %v, want 96.9 101.7 true", stop, target, ok)
	}
	// filled under the stop: no target, the stop exits at once
	stop, target, ok = s.exitLevels(&strategy.Position{PurchasePrice: 96.5, StopLoss: 96.9, TakeProfit: 100.65, EntryATR: 2})
	if !ok || stop != 96.9 || target != 0 {
		t.Fatalf("gap under the stop: exitLevels = %.4f %.4f %v, want 96.9 0 true", stop, target, ok)
	}
	// a signal without a target (TargetR 0) keeps none
	if _, target, _ = s.exitLevels(&strategy.Position{PurchasePrice: 98.5, StopLoss: 96.9, EntryATR: 2}); target != 0 {
		t.Fatalf("no signal target: target %.4f, want 0", target)
	}
}

func TestTargetROffEntry(t *testing.T) {
	p := noTrend()
	p.TargetR = 0
	if sig := decide(p, fixture(bars(fall, hammerAfterFall), at(wednesday), 40)); sig.Kind != model.SignalBuy || sig.TakeProfit != 0 {
		t.Fatalf("TargetR 0: kind %v target %.4f, want buy, no target", sig.Kind, sig.TakeProfit)
	}
}

// confirmed is the fall, hammerAfterFall (at the wednesday hour, as in the other tests), the given
// bars after it and last, with the hammer trading 200 lots so the volume filter passes on it.
func confirmed(between []bar, last bar) strategy.MarketData {
	bs := bars(fall, hammerAfterFall)
	bs = append(append(bs, between...), last)
	md := fixture(bs, at(wednesday).Add(time.Duration(1+len(between))*step), 40)
	md.Volumes[len(bs)-2-len(between)] = 200
	return md
}

// confirmBar closes green at 98.5, above hammerAfterFall's high 98.2.
var confirmBar = bar{98.15, 98.6, 98.1, 98.5}

func confirmParams(n int) Params {
	p := noTrend()
	p.ConfirmBars = n
	return p
}

func TestConfirmBuysAfterGreenAboveHigh(t *testing.T) {
	sig := decide(confirmParams(3), confirmed(nil, confirmBar))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want buy", sig.Kind)
	}
	// stop under the hammer: 97 − 0.05 × 2 = 96.9; target off the confirmation close 98.5
	if math.Abs(sig.StopLoss-96.9) > eps || math.Abs(sig.TakeProfit-(98.5+2*(98.5-96.9))) > eps {
		t.Errorf("stop %.4f target %.4f, want 96.9 %.4f", sig.StopLoss, sig.TakeProfit, 98.5+2*(98.5-96.9))
	}
	if !strings.Contains(sig.EntryReason, "подтверждение") {
		t.Errorf("reason %q, want the confirmation", sig.EntryReason)
	}
}

func TestConfirmSkipsTheHammerBar(t *testing.T) {
	md := fixture(bars(fall, hammerAfterFall), at(wednesday), 40)
	if sig := decide(confirmParams(3), md); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none (waiting for the confirmation)", sig.Kind)
	}
}

func TestConfirmNeedsCloseAboveHigh(t *testing.T) {
	if sig := decide(confirmParams(3), confirmed(nil, bar{98.15, 98.3, 98.1, 98.2})); sig.Kind != model.SignalNone {
		t.Fatalf("close at the high: kind = %v, want none", sig.Kind)
	}
	if sig := decide(confirmParams(3), confirmed(nil, bar{98.7, 98.8, 98.3, 98.5})); sig.Kind != model.SignalNone {
		t.Fatalf("red bar above the high: kind = %v, want none", sig.Kind)
	}
}

func TestConfirmBurnedUnderHammerLow(t *testing.T) {
	broke := bar{98.15, 98.2, 96.8, 98.0}
	if sig := decide(confirmParams(3), confirmed([]bar{broke}, confirmBar)); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none (a low under the hammer's burned it)", sig.Kind)
	}
}

func TestConfirmWindow(t *testing.T) {
	wait := bar{98.15, 98.2, 97.9, 98.0}
	if sig := decide(confirmParams(1), confirmed([]bar{wait}, confirmBar)); sig.Kind != model.SignalNone {
		t.Fatalf("ConfirmBars 1, confirmation 2 bars later: kind = %v, want none", sig.Kind)
	}
	if sig := decide(confirmParams(2), confirmed([]bar{wait}, confirmBar)); sig.Kind != model.SignalBuy {
		t.Fatalf("ConfirmBars 2, confirmation 2 bars later: kind = %v, want buy", sig.Kind)
	}
}

func TestConfirmOnlyTheFirstGreen(t *testing.T) {
	if sig := decide(confirmParams(3), confirmed([]bar{confirmBar}, bar{98.5, 98.9, 98.4, 98.8})); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want none (the hammer was confirmed a bar earlier)", sig.Kind)
	}
}
