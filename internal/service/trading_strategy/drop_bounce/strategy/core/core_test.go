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

// step15 is the timeframe the fixtures are built on unless a test names another.
const step15 = 15 * time.Minute

// path builds consecutive 15m bars from start (MSK): each bar opens at the previous close (the
// first at its own close) and its high/low span the open and the close.
func path(start string, closes []float64) []bar { return pathStep(start, step15, closes) }

// pathStep is path on bars of the given step.
func pathStep(start string, step time.Duration, closes []float64) []bar {
	t0 := at(start)
	out := make([]bar, len(closes))
	prev := closes[0]
	for i, c := range closes {
		out[i] = bar{t0.Add(time.Duration(i) * step), prev, max(prev, c), min(prev, c), c}
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

// fixture builds the market data over bars with fixtureATR dailies before the first bar and the
// engine's own extent of the last bar's MSK day.
func fixture(bars []bar) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, b.t)
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	withDailies(&md, dailies(bars[0].t, 20, fixtureATR))
	last := len(bars) - 1
	md.TodayHigh, md.TodayLow = bars[last].h, bars[last].l
	for j := last - 1; j >= 0 && dayKey(bars[j].t) == dayKey(bars[last].t); j-- {
		md.TodayHigh, md.TodayLow = max(md.TodayHigh, bars[j].h), min(md.TodayLow, bars[j].l)
	}
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

func decide(p Params, bars []bar) model.Signal {
	return NewWithParams("TEST", step15, p).Decide(fixture(bars))
}

// held decides with a position bought at entry on the bar starting at entryTime, with the daily
// ATR frozen at entry equal to fixtureATR.
func held(p Params, bars []bar, entry float64, entryTime time.Time) model.Signal {
	md := fixture(bars)
	md.Position = &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryTime: entryTime, EntryATR: fixtureATR}
	return NewWithParams("TEST", step15, p).Decide(md)
}

// dropAfter appends to closes one bar falling pct percent from the last close.
func dropAfter(closes []float64, pct float64) []float64 {
	return append(closes, closes[len(closes)-1]*(1-pct/100))
}

// downtrend is 20 closes sliding 0.1% per bar: RSI(6) sits near 0.
func downtrend() []float64 { return steps(100, 20, -0.1) }

func TestDefaultParams(t *testing.T) {
	want := Params{MinDropPct: 0.6, RSIPeriod: 6, RSIEntry: 30, RSIExit: 70, ExitInProfit: 1, StochPeriod: 14, StochSmooth: 3, StochExit: 80, ProfitPct: 1, StopDailyATR: 0.5, DailyATRPeriod: 14,
		VolBars: 20, StartHour: 8, GreenWaitBars: 3}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

// immediate is the defaults with the green confirmation off: the fill follows the signal bar.
func immediate() Params {
	p := DefaultParams()
	p.GreenWaitBars = 0
	return p
}

// greenBars is the oversold drop (the signal bar starts Monday 13:00) followed by bars moving the
// given percents from the previous close: a positive one is green, a negative one red, zero a doji.
func greenBars(after ...float64) []bar {
	closes := dropAfter(downtrend(), 0.6)
	for _, pct := range after {
		closes = append(closes, closes[len(closes)-1]*(1+pct/100))
	}
	return path("2026-09-14 08:00", closes)
}

func TestEnterWaitsForFirstGreen(t *testing.T) {
	cases := []struct {
		name  string
		after []float64
		want  model.SignalKind
	}{
		{"signal bar itself", nil, model.SignalNone},
		{"green right after", []float64{0.1}, model.SignalBuy},
		{"two reds then green", []float64{-0.1, -0.1, 0.1}, model.SignalBuy},
		{"doji then green", []float64{0, 0.1}, model.SignalBuy},
		{"three reds then green, past the wait", []float64{-0.1, -0.1, -0.1, 0.1}, model.SignalNone},
		{"second green", []float64{0.1, 0.1}, model.SignalNone},
		{"red after the signal", []float64{-0.1}, model.SignalNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bars := greenBars(c.after...)
			sig := decide(DefaultParams(), bars)
			if sig.Kind != c.want {
				t.Fatalf("kind = %v, want %v", sig.Kind, c.want)
			}
			if c.want != model.SignalBuy {
				return
			}
			if want := bars[len(bars)-1].t.Add(step15); !sig.FillBy.Equal(want) {
				t.Fatalf("FillBy = %v, want %v: the fill follows the green bar", sig.FillBy, want)
			}
			if !strings.HasPrefix(sig.EntryReason, "DROP:") || !strings.Contains(sig.EntryReason, "зелёная") {
				t.Fatalf("reason = %q, want the drop and the green bar", sig.EntryReason)
			}
		})
	}
}

// TestEnterGreenSameDate: the green bar must come on the signal's MSK date and the signal bar
// itself at or after StartHour.
func TestEnterGreenSameDate(t *testing.T) {
	bars := path("2026-09-14 18:30", dropAfter(downtrend(), 0.6)) // signal at 23:30
	c := bars[len(bars)-1].c
	next := append(bars, bar{at("2026-09-15 07:00"), c, c * 1.002, c, c * 1.002}, bar{at("2026-09-15 08:00"), c * 1.002, c * 1.004, c * 1.002, c * 1.004})
	if sig := decide(DefaultParams(), next); sig.Kind != model.SignalNone {
		t.Fatalf("green next morning: kind = %v, want None", sig.Kind)
	}
	bars = path("2026-09-14 02:45", dropAfter(downtrend(), 0.6)) // signal at 07:45
	c = bars[len(bars)-1].c
	bars = append(bars, bar{at("2026-09-14 08:00"), c, c * 1.001, c, c * 1.001})
	if sig := decide(DefaultParams(), bars); sig.Kind != model.SignalNone {
		t.Fatalf("signal before StartHour: kind = %v, want None", sig.Kind)
	}
}

// TestEnterGreenVolumeOnSignalBar: the volume gate measures the signal bar, not the green one.
func TestEnterGreenVolumeOnSignalBar(t *testing.T) {
	p := DefaultParams()
	p.VolMult, p.VolBars = 2, 20
	run := func(signal, green int64) model.SignalKind {
		md := fixture(greenBars(0.1))
		md.Volumes = append(volumes(21, 100, signal), green)
		return NewWithParams("TEST", step15, p).Decide(md).Kind
	}
	if got := run(200, 1); got != model.SignalBuy {
		t.Fatalf("loud signal, quiet green: kind = %v, want Buy", got)
	}
	if got := run(1, 1000); got != model.SignalNone {
		t.Fatalf("quiet signal, loud green: kind = %v, want None", got)
	}
}

func TestEnterBuysOversoldDrop(t *testing.T) {
	bars := path("2026-09-14 08:00", dropAfter(downtrend(), 0.6))
	sig := decide(immediate(), bars)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
	if want := bars[len(bars)-1].t.Add(step15); !sig.FillBy.Equal(want) {
		t.Fatalf("FillBy = %v, want %v: the fill must be the very next 15m slot", sig.FillBy, want)
	}
	if sig.StopLoss != 0 || sig.TakeProfit != 0 {
		t.Fatalf("levels = %v/%v, want none: they are measured from the fill", sig.StopLoss, sig.TakeProfit)
	}
	if math.Abs(sig.ATR-fixtureATR) > eps {
		t.Fatalf("ATR = %v, want the daily ATR %v for the engine to freeze", sig.ATR, fixtureATR)
	}
	if !strings.HasPrefix(sig.EntryReason, "DROP:") {
		t.Fatalf("reason = %q", sig.EntryReason)
	}
}

func TestEnterRejectsShallowDrop(t *testing.T) {
	if sig := decide(immediate(), path("2026-09-14 08:00", dropAfter(downtrend(), 0.59))); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None for 0.59%%", sig.Kind)
	}
}

func TestEnterRejectsRSIAtOrAboveEntry(t *testing.T) {
	closes := dropAfter(steps(100, 20, 1.0), 0.6) // strong uptrend: RSI(6) stays high after the drop
	if sig := decide(immediate(), path("2026-09-14 08:00", closes)); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None: RSI is not oversold", sig.Kind)
	}
	p := immediate()
	p.RSIEntry = 100
	if sig := decide(p, path("2026-09-14 08:00", closes)); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy with RSIEntry 100 (the drop itself qualifies)", sig.Kind)
	}
}

func TestEnterNeedsRSIHistory(t *testing.T) {
	closes := dropAfter(steps(100, 5, -0.1), 0.6) // 6 closes: RSI(6) has no value yet
	if sig := decide(immediate(), path("2026-09-14 08:00", closes)); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None on RSI warm-up", sig.Kind)
	}
}

func TestEnterWindow(t *testing.T) {
	closes := dropAfter(downtrend(), 0.6) // 21 bars: the drop is the bar 300 minutes after start
	cases := []struct {
		name, start string
		want        model.SignalKind
	}{
		{"monday 13:00", "2026-09-14 08:00", model.SignalBuy},
		{"drop 07:45, before StartHour", "2026-09-14 02:45", model.SignalNone},
		{"drop 08:00, at StartHour", "2026-09-14 03:00", model.SignalBuy},
		{"saturday", "2026-09-19 08:00", model.SignalNone},
		{"drop 23:30, fill 23:45 same date", "2026-09-14 18:30", model.SignalBuy},
	}
	for _, c := range cases {
		if sig := decide(immediate(), path(c.start, closes)); sig.Kind != c.want {
			t.Errorf("%s: kind = %v, want %v", c.name, sig.Kind, c.want)
		}
	}
}

func TestEnterRejectsFillOnNextDate(t *testing.T) {
	closes := dropAfter(downtrend(), 0.6)
	if sig := decide(immediate(), path("2026-09-14 18:45", closes)); sig.Kind != model.SignalNone { // drop at 23:45
		t.Fatalf("kind = %v, want None: the fill would be the next morning", sig.Kind)
	}
}

// TestEnterOnOtherSteps checks the 5m, 30m and 1h timeframes: the fill is one step after the signal
// bar, and a signal on the day's last bar (its fill would start the next date) is rejected.
func TestEnterOnOtherSteps(t *testing.T) {
	closes := dropAfter(downtrend(), 0.6) // 21 bars: the drop is the last one
	cases := []struct {
		name  string
		step  time.Duration
		start string // the drop bar starts 20 steps later
		want  model.SignalKind
	}{
		{"5m drop 13:00", 5 * time.Minute, "2026-09-14 11:20", model.SignalBuy},
		{"5m drop 23:55, fill on next date", 5 * time.Minute, "2026-09-14 22:15", model.SignalNone},
		{"30m drop 13:00", 30 * time.Minute, "2026-09-14 03:00", model.SignalBuy},
		{"30m drop 23:30, fill on next date", 30 * time.Minute, "2026-09-14 13:30", model.SignalNone},
		{"1h drop 13:00", time.Hour, "2026-09-13 17:00", model.SignalBuy},
		{"1h drop 22:00, fill 23:00 same date", time.Hour, "2026-09-14 02:00", model.SignalBuy},
		{"1h drop 23:00, fill on next date", time.Hour, "2026-09-14 03:00", model.SignalNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bars := pathStep(c.start, c.step, closes)
			sig := NewWithParams("TEST", c.step, immediate()).Decide(fixture(bars))
			if sig.Kind != c.want {
				t.Fatalf("kind = %v, want %v (drop at %v)", sig.Kind, c.want, bars[len(bars)-1].t)
			}
			if c.want != model.SignalBuy {
				return
			}
			if want := bars[len(bars)-1].t.Add(c.step); !sig.FillBy.Equal(want) {
				t.Fatalf("FillBy = %v, want %v: the fill must be the very next slot", sig.FillBy, want)
			}
		})
	}
}

func TestEnterInvalidParams(t *testing.T) {
	bars := path("2026-09-14 08:00", dropAfter(downtrend(), 0.6))
	for name, mut := range map[string]func(*Params){
		"MinDropPct 0": func(p *Params) { p.MinDropPct = 0 },
		"RSIPeriod 0":  func(p *Params) { p.RSIPeriod = 0 },
		"RSIEntry 0":   func(p *Params) { p.RSIEntry = 0 },
	} {
		p := immediate()
		mut(&p)
		if sig := decide(p, bars); sig.Kind != model.SignalNone {
			t.Errorf("%s: kind = %v, want None", name, sig.Kind)
		}
	}
}

// exitsOff leaves only the stop on.
func exitsOff() Params {
	p := immediate()
	p.RSIExit, p.StochExit, p.ProfitPct = 0, 0, 0
	return p
}

// flatWithLast is 30 flat bars at 100 and a last bar with the given open/high/low/close.
func flatWithLast(o, h, l, c float64) []bar {
	bars := path("2026-09-14 08:00", flatN(30, 100))
	last := bars[len(bars)-1]
	return append(bars, bar{last.t.Add(step15), o, h, l, c})
}

// The default stop is 0.5 daily ATR (fixtureATR 2) below the fill: 99 for a fill at 100.
func TestStopFromFillPrice(t *testing.T) {
	bars := flatWithLast(100, 100, 99.0, 99.5)
	sig := held(exitsOff(), bars, 100, bars[20].t)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || math.Abs(sig.StopLoss-99.0) > eps {
		t.Fatalf("got %v %q stop %v, want SL at 99.0", sig.Kind, sig.Reason, sig.StopLoss)
	}
	bars = flatWithLast(100, 100, 99.01, 99.5)
	if sig := held(exitsOff(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("low 99.01: kind = %v, want None", sig.Kind)
	}
	p := exitsOff()
	p.StopDailyATR = 0
	bars = flatWithLast(100, 100, 50, 60)
	if sig := held(p, bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("StopDailyATR 0: kind = %v, want None", sig.Kind)
	}
}

// TestStopUsesATRFrozenAtEntry: dailies that closed after the entry must not move the stop.
func TestStopUsesATRFrozenAtEntry(t *testing.T) {
	bars := flatWithLast(100, 100, 98.5, 99.5)
	md := fixture(bars)
	withDailies(&md, dailies(bars[0].t, 20, 4)) // today's ATR is 4: a live stop would sit at 98
	md.Position = &strategy.Position{PurchasePrice: 100, Quantity: 1, EntryTime: bars[20].t, EntryATR: fixtureATR}
	sig := NewWithParams("TEST", step15, exitsOff()).Decide(md)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-99) > eps {
		t.Fatalf("got %q stop %v, want SL at 99 from the entry ATR", sig.Reason, sig.StopLoss)
	}
	if !strings.Contains(sig.ExitReason, "ATR") {
		t.Fatalf("exit reason = %q, want the ATR distance", sig.ExitReason)
	}
}

// TestStopFallsBackToCurrentATR: a position without a frozen ATR keeps a stop from today's ATR
// rather than running naked.
func TestStopFallsBackToCurrentATR(t *testing.T) {
	bars := flatWithLast(100, 100, 99.0, 99.5)
	md := fixture(bars)
	md.Position = &strategy.Position{PurchasePrice: 100, Quantity: 1, EntryTime: bars[20].t}
	sig := NewWithParams("TEST", step15, exitsOff()).Decide(md)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-99) > eps {
		t.Fatalf("got %q stop %v, want SL at 99 from the current ATR", sig.Reason, sig.StopLoss)
	}
}

// TestEnterNeedsDailyATR: the ATR sizes the stop and the day-range gate, so without dailies
// neither can arm and the entry is refused; with both off the entry needs no ATR.
func TestEnterNeedsDailyATR(t *testing.T) {
	bars := path("2026-09-14 08:00", dropAfter(downtrend(), 0.6))
	noDaily := func(p Params) model.Signal {
		md := fixture(bars)
		withDailies(&md, nil)
		return NewWithParams("TEST", step15, p).Decide(md)
	}
	if sig := noDaily(immediate()); sig.Kind != model.SignalNone {
		t.Fatalf("stop on, no dailies: kind = %v, want None", sig.Kind)
	}
	p := immediate()
	p.StopDailyATR, p.MaxDayRangeATR = 0, 1
	if sig := noDaily(p); sig.Kind != model.SignalNone {
		t.Fatalf("gate on, no dailies: kind = %v, want None", sig.Kind)
	}
	p.MaxDayRangeATR = 0
	if sig := noDaily(p); sig.Kind != model.SignalBuy {
		t.Fatalf("stop and gate off, no dailies: kind = %v, want Buy", sig.Kind)
	}
}

// TestEnterIgnoresWeekendDailies: MOEX weekend sessions do not count towards the daily ATR.
func TestEnterIgnoresWeekendDailies(t *testing.T) {
	bars := path("2026-09-14 08:00", dropAfter(downtrend(), 0.6))
	md := fixture(bars)
	ds := dailies(bars[0].t, 20, fixtureATR)
	ds = append(ds, bar{at("2026-09-12 00:00"), 100, 120, 80, 100}, bar{at("2026-09-13 00:00"), 100, 120, 80, 100})
	withDailies(&md, ds)
	sig := NewWithParams("TEST", step15, immediate()).Decide(md)
	if sig.Kind != model.SignalBuy || math.Abs(sig.ATR-fixtureATR) > eps {
		t.Fatalf("got %v ATR %v, want Buy with ATR %v", sig.Kind, sig.ATR, fixtureATR)
	}
}

// TestEnterDayRangeGate: the day's high−low up to the signal bar must stay within
// MaxDayRangeATR daily ATRs.
func TestEnterDayRangeGate(t *testing.T) {
	bars := path("2026-09-14 08:00", dropAfter(downtrend(), 0.6))
	md := fixture(bars)
	used := (md.TodayHigh - md.TodayLow) / fixtureATR
	p := immediate()
	p.MaxDayRangeATR = used
	sig := decide(p, bars)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("range at the cap (%.4f ATR): kind = %v, want Buy", used, sig.Kind)
	}
	if !strings.Contains(sig.EntryReason, "размах дня") {
		t.Fatalf("reason = %q, want the day range", sig.EntryReason)
	}
	p.MaxDayRangeATR = used - 0.01
	if sig := decide(p, bars); sig.Kind != model.SignalNone {
		t.Fatalf("range above the cap: kind = %v, want None", sig.Kind)
	}
	md.TodayHigh, md.TodayLow = 0, 0
	if sig := NewWithParams("TEST", step15, p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("no day extent: kind = %v, want Buy (the gate never blocks on missing data)", sig.Kind)
	}
}

func TestNoEODExit(t *testing.T) {
	// Overnight: entry Monday 23:45, the bar is Tuesday 07:00 with nothing triggered.
	bars := path("2026-09-14 16:15", flatN(31, 100)) // last bar 23:45
	bars = append(bars, bar{at("2026-09-15 07:00"), 100, 100.2, 99.8, 100})
	if sig := held(exitsOff(), bars, 100, at("2026-09-14 23:45")); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v %q, want None: positions carry overnight", sig.Kind, sig.Reason)
	}
}

// volumeDecide decides on the oversold drop with the given volumes (the last is the signal bar's).
func volumeDecide(p Params, vols []int64) model.Signal {
	bars := path("2026-09-14 08:00", dropAfter(downtrend(), 0.6))
	md := fixture(bars)
	md.Volumes = vols
	return NewWithParams("TEST", step15, p).Decide(md)
}

// volumes returns n−1 prior bars of `prior` volume and a signal bar of `last`.
func volumes(n int, prior, last int64) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = prior
	}
	out[n-1] = last
	return out
}

// TestEnterVolumeFilter: the signal bar's volume must reach VolMult times the mean volume of the
// VolBars bars before it.
func TestEnterVolumeFilter(t *testing.T) {
	p := immediate()
	p.VolMult, p.VolBars = 2, 20
	sig := volumeDecide(p, volumes(21, 100, 200))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("volume at the cap: kind = %v, want Buy", sig.Kind)
	}
	if !strings.Contains(sig.EntryReason, "объём") {
		t.Fatalf("reason = %q, want the volume", sig.EntryReason)
	}
	p.VolMult = 2.01
	if sig := volumeDecide(p, volumes(21, 100, 200)); sig.Kind != model.SignalNone {
		t.Fatalf("volume below the cap: kind = %v, want None", sig.Kind)
	}
	p.VolMult = 0
	if sig := volumeDecide(p, volumes(21, 100, 1)); sig.Kind != model.SignalBuy {
		t.Fatalf("VolMult 0: kind = %v, want Buy", sig.Kind)
	}
}

// TestEnterVolumeFilterIgnoresZeroVolumes: zero-volume bars do not dilute the mean.
func TestEnterVolumeFilterIgnoresZeroVolumes(t *testing.T) {
	p := immediate()
	p.VolMult, p.VolBars = 2.01, 20
	vols := volumes(21, 100, 200)
	for i := range 10 {
		vols[i] = 0
	}
	if sig := volumeDecide(p, vols); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None: the mean over non-zero bars is 100", sig.Kind)
	}
}

// TestEnterVolumeFilterNeverBlocksOnMissingData: no volumes, too short a history or a mean of
// nothing let the signal through.
func TestEnterVolumeFilterNeverBlocksOnMissingData(t *testing.T) {
	p := immediate()
	p.VolMult, p.VolBars = 2, 20
	for name, vols := range map[string][]int64{
		"no volumes": nil,
		"misaligned": volumes(21, 100, 1)[1:],
		"all zero":   volumes(21, 0, 1),
	} {
		if sig := volumeDecide(p, vols); sig.Kind != model.SignalBuy {
			t.Errorf("%s: kind = %v, want Buy", name, sig.Kind)
		}
	}
	p.VolBars = 21 // the 21-bar fixture has only 20 bars before the signal
	if sig := volumeDecide(p, volumes(21, 100, 1)); sig.Kind != model.SignalBuy {
		t.Errorf("short history: kind = %v, want Buy", sig.Kind)
	}
}

func TestLookbackCoversVolumeBars(t *testing.T) {
	p := immediate()
	p.VolMult, p.VolBars = 2, 100
	if got := NewWithParams("TEST", step15, p).Lookback(); got < 101 {
		t.Fatalf("Lookback = %d, want ≥ 101 for VolBars 100", got)
	}
}
