package core

import (
	"math"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

const eps = 1e-9

// step is the bar length the strategy trades on.
const step = 15 * time.Minute

// at parses "2006-01-02 15:04" as MSK.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, mskLoc)
	if err != nil {
		panic(err)
	}
	return t
}

// ts formats t like at parses it.
func ts(t time.Time) string { return t.In(mskLoc).Format("2006-01-02 15:04") }

type bar struct {
	t          string
	o, h, l, c float64
}

// flat returns flat bars (o=h=l=c=p) every 15 minutes over [from, to], both MSK.
func flat(from, to string, p float64) []bar {
	var out []bar
	for t := at(from); !t.After(at(to)); t = t.Add(step) {
		out = append(out, bar{ts(t), p, p, p, p})
	}
	return out
}

// path chains bars every 15 minutes from MSK `from`: each opens at the previous close (the first
// at o0) and closes at the next value; highs and lows hug the bodies.
func path(from string, o0 float64, closes ...float64) []bar {
	var out []bar
	o := o0
	for k, c := range closes {
		out = append(out, bar{ts(at(from).Add(time.Duration(k) * step)), o, max(o, c), min(o, c), c})
		o = c
	}
	return out
}

// dailyBars returns `days` calendar days of completed dailies ending the day before `before`, all
// closing at 100. Weekdays span 99..101 (PDL 99, PDH 101, Wilder ATR exactly 2); weekend sessions
// span 99.9..100.1, so a weekend daily leaking into the levels is visible.
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

// fixture builds MarketData from bars plus 40 days of dailies before the last bar.
func fixture(bars []bar) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, at(b.t))
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = dailyBars(md.Times[len(md.Times)-1], 40)
	return md
}

func newS(p Params) *Strategy { return NewWithParams("TEST", step, p) }

// warmBars is how many flat bars warm prepends: history before the day under test.
const warmBars = 150

// warm prepends warmBars flat bars at 100 ending one bar before bars[0].
func warm(bars []bar) []bar {
	var out []bar
	for k := warmBars; k >= 1; k-- {
		t := ts(at(bars[0].t).Add(-time.Duration(k) * step))
		out = append(out, bar{t, 100, 100, 100, 100})
	}
	return append(out, bars...)
}

// withVolumes sets the volume of every bar to base and of bar i to spike.
func withVolumes(md strategy.MarketData, base int64, i int, spike int64) strategy.MarketData {
	md.Volumes = make([]int64, len(md.Times))
	for j := range md.Volumes {
		md.Volumes[j] = base
	}
	md.Volumes[i] = spike
	return md
}

func decide(p Params, bars []bar) model.Signal { return newS(p).Decide(fixture(warm(bars))) }

// Monday 2026-09-14. PDL 99, PDH 101 (Friday's daily), daily ATR 2.
//
// pdlDay: 09:30 drifts to 99.5, 09:45 and 10:00 are red closes below PDL, 10:15 is the green
// reclaim (99.2) — the buy trigger — then the price climbs.
func pdlDay() []bar {
	return path("2026-09-14 09:30", 100, 99.5, 98.7, 98.6, 99.2, 99.3, 99.4, 99.5, 99.6, 99.7)
}

const reclaimAt = 3 // index of the reclaim bar in pdlDay

// upTo is pdlDay up to and including bar i.
func upTo(i int) []bar { return pdlDay()[:i+1] }

func TestReclaimEntry(t *testing.T) {
	sig := decide(DefaultParams(), upTo(reclaimAt))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.StopLoss-98.2) > eps || math.Abs(sig.TakeProfit-100) > eps || math.Abs(sig.ATR-2) > eps || math.Abs(sig.Level-99) > eps {
		t.Fatalf("SL/TP/ATR/Level = %v/%v/%v/%v, want 98.2/100/2/99", sig.StopLoss, sig.TakeProfit, sig.ATR, sig.Level)
	}
	if !sig.FillBy.Equal(at("2026-09-14 10:30")) {
		t.Fatalf("FillBy = %v, want 10:30", sig.FillBy)
	}
	if !strings.HasPrefix(sig.EntryReason, "PDL 15m: возврат над PDL на свече 10:15, покупка сразу") {
		t.Fatalf("EntryReason = %q", sig.EntryReason)
	}
}

// Only the reclaim bar buys: not the bars before it, not the climb after it.
func TestReclaimBuysOnReclaimBarOnly(t *testing.T) {
	for i := 0; i < len(pdlDay()); i++ {
		if got := decide(DefaultParams(), upTo(i)).Kind == model.SignalBuy; got != (i == reclaimAt) {
			t.Errorf("bar %d: buy = %v, want %v", i, got, i == reclaimAt)
		}
	}
}

// series builds a series from bars without warm-up.
func seriesOf(bars []bar) series { return barsOf(fixture(bars)) }

func TestIsReclaim(t *testing.T) {
	for name, tc := range map[string]struct {
		bars []bar
		want bool
	}{
		"red below, then green above":      {path("2026-09-14 10:00", 99.5, 98.7, 99.2), true},
		"red below, green below, green up": {path("2026-09-14 10:00", 99.5, 98.7, 98.9, 99.2), true},
		"dip only, no red close below":     {[]bar{{"2026-09-14 10:00", 99.5, 99.5, 98.5, 99.4}, {"2026-09-14 10:15", 99.4, 99.6, 99.3, 99.5}}, false},
		"green close below, no red":        {path("2026-09-14 10:00", 98.5, 98.7, 99.2), false},
		"close above PDL in between":       {path("2026-09-14 10:00", 99.5, 98.7, 99.1, 99.0, 99.2), false},
		"close at PDL":                     {path("2026-09-14 10:00", 99.5, 98.7, 99.0), false},
		"reclaim bar is red":               {[]bar{{"2026-09-14 10:00", 99.5, 99.5, 98.7, 98.7}, {"2026-09-14 10:15", 99.4, 99.4, 99.1, 99.2}}, false},
		"red below on the previous date":   {[]bar{{"2026-09-11 23:45", 99.5, 99.5, 98.7, 98.7}, {"2026-09-14 07:00", 98.8, 99.3, 98.8, 99.2}}, false},
	} {
		ser := seriesOf(tc.bars)
		if got := isReclaim(ser, len(tc.bars)-1, 99); got != tc.want {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
}

func TestIsBreakout(t *testing.T) {
	for name, tc := range map[string]struct {
		bars []bar
		want bool
	}{
		"first green close above":   {[]bar{preC, barC}, true},
		"previous closed above PDH": {[]bar{{"2026-09-14 09:45", 101, 101.2, 100.9, 101.1}, barC}, false},
		"red bar":                   {[]bar{preC, {"2026-09-14 10:00", 101.5, 101.6, 101.2, 101.3}}, false},
		"close at PDH":              {[]bar{preC, {"2026-09-14 10:00", 100.6, 101.2, 100.5, 101}}, false},
		"previous of another date":  {[]bar{{"2026-09-11 23:45", 100.5, 100.5, 100.5, 100.5}, {"2026-09-14 07:00", 100.6, 101.5, 100.5, 101.4}}, false},
	} {
		if got := isBreakout(seriesOf(tc.bars), 1, 101); got != tc.want {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
}

// PDH 101, daily ATR 2: preC closes under PDH, barC is green and closes 0.4 above it, holdC is
// the next bar and closes above PDH too.
var (
	preC  = bar{"2026-09-14 09:45", 100.4, 100.6, 100.3, 100.5}
	barC  = bar{"2026-09-14 10:00", 100.6, 101.5, 100.5, 101.4}
	holdC = bar{"2026-09-14 10:15", 101.4, 101.6, 101.2, 101.5}
)

// pdhMD is preC, the breakout barC and the bars after it, with the volume of barC at spike against
// 100 on every other bar. The warm-up reaches back only to Saturday, so Friday's main session goes
// first: the volume average needs main-session bars.
func pdhMD(spike int64, after ...bar) strategy.MarketData {
	friday := flat("2026-09-11 10:00", "2026-09-11 18:30", 100)
	md := fixture(append(friday, warm(append([]bar{preC, barC}, after...))...))
	return withVolumes(md, 100, len(friday)+warmBars+1, spike)
}

func TestBreakoutEntry(t *testing.T) {
	sig := newS(DefaultParams()).Decide(pdhMD(151, holdC))
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "PDH 15m: пробой PDH на свече 10:00, держится над PDH 1 св. (мин. close 101.4000)") {
		t.Fatalf("got %v %q, want Buy PDH", sig.Kind, sig.EntryReason)
	}
	if math.Abs(sig.TakeProfit-102) > eps || math.Abs(sig.StopLoss-100.5) > eps || math.Abs(sig.Level-101) > eps {
		t.Fatalf("TP/SL/Level = %v/%v/%v, want 102/100.5/101", sig.TakeProfit, sig.StopLoss, sig.Level)
	}
	if !sig.FillBy.Equal(at("2026-09-14 10:30")) {
		t.Fatalf("FillBy = %v, want 10:30", sig.FillBy)
	}
	if !strings.Contains(sig.EntryReason, "объём пробоя 151 > 1.50 × среднее 20 дневных свечей 100") {
		t.Fatalf("EntryReason misses the volume: %q", sig.EntryReason)
	}
}

// The breakout buys only on the bar that completes the hold: not on the breakout bar, not after.
func TestBreakoutBuysAfterHold(t *testing.T) {
	later := bar{"2026-09-14 10:30", 101.5, 101.7, 101.4, 101.6}
	s := newS(DefaultParams())
	if sig := s.Decide(pdhMD(151)); sig.Kind != model.SignalNone {
		t.Errorf("breakout bar, hold 1: Kind = %v, want None", sig.Kind)
	}
	if sig := s.Decide(pdhMD(151, holdC, later)); sig.Kind != model.SignalNone {
		t.Errorf("bar after the hold: Kind = %v, want None", sig.Kind)
	}
	p := DefaultParams()
	p.HoldBars = 0
	if sig := newS(p).Decide(pdhMD(151)); sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "держится над PDH 0 св.") {
		t.Errorf("hold 0, breakout bar: got %v %q, want Buy", sig.Kind, sig.EntryReason)
	}
	p.HoldBars = 2
	if sig := newS(p).Decide(pdhMD(151, holdC)); sig.Kind != model.SignalNone {
		t.Errorf("hold 2, one bar held: Kind = %v, want None", sig.Kind)
	}
	if sig := newS(p).Decide(pdhMD(151, holdC, later)); sig.Kind != model.SignalBuy {
		t.Errorf("hold 2, two bars held: Kind = %v, want Buy", sig.Kind)
	}
	p.HoldBars = -1
	if sig := newS(p).Decide(pdhMD(151)); sig.Kind != model.SignalNone {
		t.Errorf("hold -1: Kind = %v, want None", sig.Kind)
	}
}

// A close below PDH during the hold cancels the breakout; a close exactly at PDH does not.
func TestBreakoutHoldCancelledBelowPDH(t *testing.T) {
	below := bar{"2026-09-14 10:15", 101.4, 101.4, 100.8, 100.9}
	back := bar{"2026-09-14 10:30", 100.9, 101.4, 100.9, 101.3}
	p := DefaultParams()
	p.HoldBars = 2
	if sig := newS(p).Decide(pdhMD(151, below, back)); sig.Kind != model.SignalNone {
		t.Errorf("hold 2, close below PDH: Kind = %v, want None", sig.Kind)
	}
	if sig := newS(DefaultParams()).Decide(pdhMD(151, below)); sig.Kind != model.SignalNone {
		t.Errorf("hold 1, close below PDH: Kind = %v, want None", sig.Kind)
	}
	atPDH := bar{"2026-09-14 10:15", 101.4, 101.4, 100.9, 101}
	if sig := newS(DefaultParams()).Decide(pdhMD(151, atPDH)); sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "мин. close 101.0000") {
		t.Errorf("hold 1, close at PDH: got %v %q, want Buy", sig.Kind, sig.EntryReason)
	}
}

// The hold does not cross midnight: a breakout on the last bar of Friday is not held on Monday.
func TestBreakoutHoldSameDate(t *testing.T) {
	bars := []bar{
		{"2026-09-11 23:30", 100.4, 100.6, 100.3, 100.5},
		{"2026-09-11 23:45", 100.6, 101.5, 100.5, 101.4},
		{"2026-09-14 07:00", 101.4, 101.6, 101.2, 101.5},
	}
	md := withVolumes(fixture(warm(bars)), 100, warmBars+1, 151)
	if sig := newS(DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
}

func TestBreakoutNeedsVolume(t *testing.T) {
	if sig := newS(DefaultParams()).Decide(pdhMD(150, holdC)); sig.Kind != model.SignalNone {
		t.Fatalf("volume 150 = 1.5 × 100: Kind = %v, want None", sig.Kind)
	}
	if sig := decide(DefaultParams(), []bar{preC, barC, holdC}); sig.Kind != model.SignalNone {
		t.Fatalf("no volumes: Kind = %v, want None", sig.Kind)
	}
	p := DefaultParams()
	p.VolumeMult = 0
	if sig := decide(p, []bar{preC, barC, holdC}); sig.Kind != model.SignalBuy {
		t.Fatalf("volume filter off: Kind = %v, want Buy", sig.Kind)
	}
}

func TestEntrySwitches(t *testing.T) {
	p := DefaultParams()
	p.EntryPDL = 0
	if sig := decide(p, upTo(reclaimAt)); sig.Kind != model.SignalNone {
		t.Errorf("EntryPDL off: Kind = %v, want None", sig.Kind)
	}
	p = DefaultParams()
	p.EntryPDH = 0
	if sig := newS(p).Decide(pdhMD(151, holdC)); sig.Kind != model.SignalNone {
		t.Errorf("EntryPDH off: Kind = %v, want None", sig.Kind)
	}
}

// The window gates the trigger bar: the reclaim shifted so it lands at the given time.
func TestEntryWindow(t *testing.T) {
	for trig, want := range map[string]bool{
		"2026-09-14 06:45": false,
		"2026-09-14 07:00": true,
		"2026-09-14 17:30": true,  // buy at 17:45
		"2026-09-14 17:45": false, // buy at 18:00 is too late
		"2026-09-12 10:00": false, // Saturday
	} {
		from := ts(at(trig).Add(-reclaimAt * step))
		bars := path(from, 100, 99.5, 98.7, 98.6, 99.2)
		if got := decide(DefaultParams(), bars).Kind == model.SignalBuy; got != want {
			t.Errorf("%s: buy = %v, want %v", trig, got, want)
		}
	}
}

// With the window widened to midnight the fill bar of a 23:45 trigger is the next date — no entry.
func TestEntryFillMustBeSameDate(t *testing.T) {
	p := DefaultParams()
	p.EntryToHour = 24
	bars := path("2026-09-14 23:00", 100, 99.5, 98.7, 98.6, 99.2)
	if sig := decide(p, bars); sig.Kind != model.SignalNone {
		t.Fatalf("23:45: Kind = %v, want None", sig.Kind)
	}
	bars = path("2026-09-14 22:45", 100, 99.5, 98.7, 98.6, 99.2)
	if sig := decide(p, bars); sig.Kind != model.SignalBuy {
		t.Fatalf("23:30: Kind = %v, want Buy", sig.Kind)
	}
}

func TestTargetOf(t *testing.T) {
	if got := targetOf(setup{name: "PDL", level: 99}, 99, 101); math.Abs(got-100) > eps {
		t.Errorf("PDL target = %v, want 100", got)
	}
	if got := targetOf(setup{name: "PDH", level: 101}, 99, 101); math.Abs(got-102) > eps {
		t.Errorf("PDH target = %v, want 102", got)
	}
}

// A trigger closing at/above the target does not buy: the reclaim shoots straight to 100.
func TestEntryCloseBelowTarget(t *testing.T) {
	bars := path("2026-09-14 09:30", 100, 99.5, 98.7, 98.6, 100.2)
	if sig := decide(DefaultParams(), bars); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
}

// Sunday's daily (99.9..100.1) is not the reference day; Friday's (99..101) is.
func TestLevelsIgnoreWeekendDaily(t *testing.T) {
	pdl, pdh, ok := newS(DefaultParams()).levels(fixture(pdlDay()))
	if !ok || pdl != 99 || pdh != 101 {
		t.Fatalf("levels = %v %v %v, want 99 101 true", pdl, pdh, ok)
	}
}

func TestEntryNeedsDailiesAndStop(t *testing.T) {
	md := fixture(warm(upTo(reclaimAt)))
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
	if sig := newS(DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Errorf("no dailies: Kind = %v, want None", sig.Kind)
	}
	p := DefaultParams()
	p.StopDailyATR = 0
	if sig := decide(p, upTo(reclaimAt)); sig.Kind != model.SignalNone {
		t.Errorf("no stop: Kind = %v, want None", sig.Kind)
	}
}

func TestLookbackCoversDayAndVolume(t *testing.T) {
	p := DefaultParams()
	p.VolumeMult = 0
	if got := newS(p).Lookback(); got != dayBars {
		t.Fatalf("Lookback = %d, want %d", got, dayBars)
	}
	// VolBars 20 fit one main session; a weekend may lie between: three days of bars.
	if got := newS(DefaultParams()).Lookback(); got != 3*dayBars+dayBars {
		t.Fatalf("Lookback = %d, want %d", got, 4*dayBars)
	}
	p = DefaultParams()
	p.VolBars = 500
	if l, need := newS(p).Lookback(), (500/sessionBars+1)*dayBars; l < need {
		t.Fatalf("Lookback %d with VolBars 500, want ≥ %d", l, need)
	}
}
