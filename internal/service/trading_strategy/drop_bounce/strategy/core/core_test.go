package core

import (
	"math"
	"strings"
	"testing"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
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

// stepM30 is the timeframe drop_bounce runs on.
const stepM30 = 30 * time.Minute

// path builds consecutive 30m bars from start (MSK): each bar opens at the previous close (the
// first at its own close) and its high/low span the open and the close.
func path(start string, closes []float64) []bar {
	t0 := at(start)
	out := make([]bar, len(closes))
	prev := closes[0]
	for i, c := range closes {
		out[i] = bar{t0.Add(time.Duration(i) * stepM30), prev, max(prev, c), min(prev, c), c}
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

// halfHours expands hourly closes into 30m closes: the first half-hour goes halfway from the
// previous hourly close, the second ends on the hourly close.
func halfHours(hourly []float64) []float64 {
	out := make([]float64, 0, 2*len(hourly))
	prev := hourly[0]
	for _, c := range hourly {
		out = append(out, (prev+c)/2, c)
		prev = c
	}
	return out
}

// fixtureATR is the daily ATR every fixture carries: its dailies range exactly this much.
const fixtureATR = 2.0

// fixture builds the market data over bars with 20 weekday dailies of range fixtureATR before the
// first bar.
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
	return NewWithParams("TEST", stepM30, p).Decide(md)
}

// trendCloses is 60 zigzag hours (+2%, −0.5%) from 100, then `drops` hours of −1% each. With
// hourly RSI(6) and EMA(30): drops=6 → RSI 28.34 with the close above the EMA, drops=12 → RSI deep
// in the zone with the close below the EMA.
func trendCloses(drops int) []float64 {
	c := []float64{100}
	for i := range 60 {
		k := 1.02
		if i%2 == 1 {
			k = 0.995
		}
		c = append(c, c[len(c)-1]*k)
	}
	for range drops {
		c = append(c, c[len(c)-1]*0.99)
	}
	return c
}

// entryParams is the default params with an hourly EMA(30): the fixtures are ~70 hours long.
func entryParams() Params {
	p := DefaultParams()
	p.EMAPeriod = 30
	return p
}

// signalStart is the first bar of the entry fixture: Monday 00:00 MSK, so the 67 hours end on
// Wednesday 18:30 and the confirmation bars stay on Wednesday.
const signalStart = "2026-09-14 00:00"

// entryFixture is the hourly trendCloses(drops) as 30m bars from start, plus `after` extra 30m
// closes. The signal hour (the last of trendCloses) completes on bar armBar.
func entryFixture(start string, drops int, after ...float64) (bars []bar, armBar int) {
	c := halfHours(trendCloses(drops))
	armBar = len(c) - 1
	return path(start, append(c, after...)), armBar
}

// greenAfterSignal is the entry fixture with one +15% 30m bar after the signal hour: the histogram
// is red on the arming bar a and green on a+1.
func greenAfterSignal() (bars []bar, a int) {
	base, a := entryFixture(signalStart, 6)
	bars, _ = entryFixture(signalStart, 6, base[a].c*1.15)
	return bars, a
}

// histOf is the 30m MACD(12,26,9) histogram over the bars, as the strategy computes it.
func histOf(bars []bar) []float64 {
	cl := make([]float64, len(bars))
	for i, b := range bars {
		cl[i] = b.c
	}
	return macdHistogram(cl, DefaultParams())
}

// decideAt runs a fresh strategy over bars[:i+1].
func decideAt(p Params, bars []bar, i int) model.Signal {
	return NewWithParams("TEST", stepM30, p).Decide(fixture(bars[:i+1]))
}

func TestDefaultParams(t *testing.T) {
	want := Params{RSIPeriod: 6, RSIZone: 30, RSICap: 50, ZoneEntry: 1, EMAPeriod: 200,
		StopDailyATR: 1.0, BreakevenATR: 0.5, TrailATR: 0, DailyATRPeriod: 14, RSIExitPeriod: 10, RSIExit: 70,
		MACDFast: 12, MACDSlow: 26, MACDSignal: 9, ConfirmBars: 3, FridayTo: 11, StaleBars: 0, StaleATR: 0.3, PrevCloseGate: 0, PrevCloseATR: 0}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestLookback(t *testing.T) {
	cases := []struct {
		p    Params
		want int
	}{
		{DefaultParams(), 1202},                    // 2 × (600 hours + 1)
		{Params{RSIPeriod: 6, EMAPeriod: 10}, 130}, // 2 × (64 + 1)
		{Params{RSIPeriod: 6, EMAPeriod: 10, RSIExitPeriod: 20}, 166},
		{Params{RSIPeriod: 6, EMAPeriod: 10, MACDSlow: 100, MACDSignal: 9, ConfirmBars: 4}, 332},
	}
	for _, c := range cases {
		if got := NewWithParams("T", stepM30, c.p).Lookback(); got != c.want {
			t.Fatalf("Lookback(%+v) = %d, want %d", c.p, got, c.want)
		}
	}
}

func TestHourBars(t *testing.T) {
	bars := path("2026-09-16 10:00", []float64{1, 2, 3, 4, 5})
	// The 11:30 bar is missing: 11:00 is completed by the 12:00 bar.
	bars = append(bars[:3], bar{at("2026-09-16 12:00"), 4, 4, 4, 4}, bar{at("2026-09-16 12:30"), 5, 5, 5, 5}, bar{at("2026-09-16 13:00"), 6, 6, 6, 6})
	hs := NewWithParams("T", stepM30, DefaultParams()).hourBars(fixture(bars))
	if len(hs) != 3 {
		t.Fatalf("hours = %+v, want 3 completed (13:00 still forming)", hs)
	}
	want := []struct {
		start      string
		close      float64
		last, done int
	}{{"2026-09-16 10:00", 2, 1, 1}, {"2026-09-16 11:00", 3, 2, 3}, {"2026-09-16 12:00", 5, 4, 4}}
	for k, w := range want {
		if !hs[k].start.Equal(at(w.start)) || hs[k].close != w.close || hs[k].last != w.last || hs[k].done != w.done {
			t.Fatalf("hour %d = %+v, want %+v", k, hs[k], w)
		}
	}
}

// The armed signal waits for the 30m histogram: red on the arming bar holds, the jump that turns it
// green buys at the next bar's open.
func TestEnterWaitsForGreenHistogram(t *testing.T) {
	bars, a := greenAfterSignal()
	h := histOf(bars)
	if h == nil || !isRed(h[a]) || isRed(h[a+1]) {
		t.Fatalf("fixture: histogram must be red on the arming bar and green after the jump: %v / %v", h[a], h[a+1])
	}
	cl := trendCloses(6)
	if r := indicators.RSISeries(cl, 6); r[len(r)-1] >= 30 || cl[len(cl)-1] <= ema.Compute(cl, 30)[len(cl)-1] {
		t.Fatal("fixture: the last hour must have RSI(6) < 30 above EMA(30)")
	}
	p := entryParams()
	if sig := decideAt(p, bars, a); sig.Kind != model.SignalNone {
		t.Fatalf("arming bar with a red histogram: got %v %q, want none", sig.Kind, sig.EntryReason)
	}
	sig := decideAt(p, bars, a+1)
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "ZONE: час ") || !strings.Contains(sig.EntryReason, "MACD 30m") {
		t.Fatalf("green bar: got %v %q, want Buy ZONE", sig.Kind, sig.EntryReason)
	}
	if want := bars[a+1].t.Add(stepM30); !sig.FillBy.Equal(want) || math.Abs(sig.ATR-fixtureATR) > eps {
		t.Fatalf("FillBy/ATR = %v/%v, want %v/%v", sig.FillBy, sig.ATR, want, fixtureATR)
	}
	// Outside the confirmation window the same green bar no longer buys.
	p.ConfirmBars = 1
	if sig := decideAt(p, bars, a+1); sig.Kind != model.SignalNone {
		t.Fatalf("ConfirmBars 1, green on the second bar: got %v, want none", sig.Kind)
	}
}

func TestConfirmed(t *testing.T) {
	h := []float64{-1, 0.5, 0.6, -0.1, 0}
	cases := []struct {
		a, i int
		want bool
	}{
		{1, 1, true},  // green on the arming bar: buy at once
		{0, 1, true},  // red → green flip
		{0, 2, false}, // green, but already green before: the flip was missed
		{0, 3, false}, // red
		{0, 4, true},  // exactly 0 is green, after red
	}
	for _, c := range cases {
		if got := confirmed(h, c.a, c.i); got != c.want {
			t.Fatalf("confirmed(a=%d, i=%d) = %v, want %v", c.a, c.i, got, c.want)
		}
	}
	if confirmed(nil, 0, 0) {
		t.Fatal("no histogram must never confirm")
	}
}

func TestEnterBelowEMANoSignal(t *testing.T) {
	bars, a := entryFixture(signalStart, 12)
	bars = append(bars, path(bars[a].t.Add(stepM30).Format("2006-01-02 15:04"), []float64{bars[a].c * 1.15})...)
	if isRed(histOf(bars)[a+1]) {
		t.Fatal("fixture: histogram must be green after the jump")
	}
	for i := a; i <= a+1; i++ {
		if sig := decideAt(entryParams(), bars, i); sig.Kind != model.SignalNone {
			t.Fatalf("below EMA, bar %d: got %v %q, want none", i, sig.Kind, sig.EntryReason)
		}
	}
}

// Only weekdays: the same signal shifted so the green bar falls on Saturday does not buy.
func TestEnterWeekdaysOnly(t *testing.T) {
	bars, a := greenAfterSignal()
	shift := at("2026-09-19 12:00").Sub(bars[a+1].t)
	for i := range bars {
		bars[i].t = bars[i].t.Add(shift)
	}
	if sig := decideAt(entryParams(), bars, a+1); sig.Kind != model.SignalNone {
		t.Fatalf("Saturday: got %v, want none", sig.Kind)
	}
}

// On Friday the fill bar must start before FridayTo: a 10:30 fill buys, an 11:00 fill does not;
// FridayTo 24 lifts the limit, 0 closes Friday entirely.
func TestEnterFridayBeforeCutoff(t *testing.T) {
	cases := []struct {
		green    string
		fridayTo int
		want     model.SignalKind
	}{
		{"2026-09-18 10:00", 11, model.SignalBuy},
		{"2026-09-18 10:30", 11, model.SignalNone},
		{"2026-09-18 17:00", 11, model.SignalNone},
		{"2026-09-18 17:00", 24, model.SignalBuy},
		{"2026-09-18 07:00", 0, model.SignalNone},
		{"2026-09-17 17:00", 11, model.SignalBuy}, // Thursday evening is unaffected
	}
	for _, c := range cases {
		bars, a := greenAfterSignal()
		shift := at(c.green).Sub(bars[a+1].t)
		for i := range bars {
			bars[i].t = bars[i].t.Add(shift)
		}
		p := entryParams()
		p.FridayTo = c.fridayTo
		if sig := decideAt(p, bars, a+1); sig.Kind != c.want {
			t.Fatalf("green bar %s, FridayTo %d: got %v, want %v", c.green, c.fridayTo, sig.Kind, c.want)
		}
	}
}

// ZoneEntry 0 leaves only CROSS: the ZONE signal of greenAfterSignal no longer buys.
func TestZoneEntryOff(t *testing.T) {
	bars, a := greenAfterSignal()
	p := entryParams()
	p.ZoneEntry = 0
	if sig := decideAt(p, bars, a+1); sig.Kind != model.SignalNone {
		t.Fatalf("ZoneEntry 0, ZONE signal: got %v %q, want none", sig.Kind, sig.EntryReason)
	}
}

func TestEnterZeroParamsGiveNoEntries(t *testing.T) {
	bars, a := greenAfterSignal()
	for name, mut := range map[string]func(*Params){
		"EMAPeriod":   func(p *Params) { p.EMAPeriod = 0 },
		"RSIPeriod":   func(p *Params) { p.RSIPeriod = 0 },
		"ConfirmBars": func(p *Params) { p.ConfirmBars = 0 },
		"MACD":        func(p *Params) { p.MACDFast = p.MACDSlow },
	} {
		p := entryParams()
		mut(&p)
		if sig := decideAt(p, bars, a+1); sig.Kind != model.SignalNone {
			t.Fatalf("%s off: got %v, want none", name, sig.Kind)
		}
	}
}

func TestEnterNeedsDailyATR(t *testing.T) {
	bars, a := greenAfterSignal()
	md := fixture(bars[:a+2])
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
	if sig := NewWithParams("TEST", stepM30, entryParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("no daily ATR: got %v, want none", sig.Kind)
	}
}

func TestTrendUpIsStrict(t *testing.T) {
	if trendUp(100, 100) || trendUp(100, 0) || !trendUp(100.01, 100) {
		t.Fatal("trendUp must need close strictly above a warmed (non-zero) EMA")
	}
}

// The buy carries the stop off the signal close (StopDailyATR daily ATRs below it) for risk sizing.
func TestEnterCarriesStopForSizing(t *testing.T) {
	bars, a := greenAfterSignal()
	p := entryParams()
	sig := decideAt(p, bars, a+1)
	if want := bars[a+1].c - p.StopDailyATR*fixtureATR; sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-want) > eps {
		t.Fatalf("got %v stop %v, want Buy with stop %v", sig.Kind, sig.StopLoss, want)
	}
	p.StopDailyATR = 0
	if sig := decideAt(p, bars, a+1); sig.Kind != model.SignalBuy || sig.StopLoss != 0 {
		t.Fatalf("stop off: got %v stop %v, want Buy without a stop", sig.Kind, sig.StopLoss)
	}
}

// dailiesAt is dailies(before, n, fixtureATR) moved to close at `close`, so the ATR stays fixtureATR.
func dailiesAt(before time.Time, n int, close float64) []bar {
	ds := dailies(before, n, fixtureATR)
	for k := range ds {
		ds[k].o, ds[k].c, ds[k].h, ds[k].l = close, close, close+fixtureATR/2, close-fixtureATR/2
	}
	return ds
}

// PrevCloseGate buys only at or below the previous weekday close + PrevCloseATR daily ATRs; a
// weekend session after it does not count.
func TestEnterPrevCloseGate(t *testing.T) {
	bars, a := greenAfterSignal()
	c := bars[a+1].c
	cases := []struct {
		gate      int
		prevClose float64
		allowance float64
		want      model.SignalKind
	}{
		{0, c - 5, 0, model.SignalBuy},       // gate off
		{1, c - 0.5, 0, model.SignalNone},    // above yesterday's close
		{1, c, 0, model.SignalBuy},           // exactly at it
		{1, c + 0.1, 0, model.SignalBuy},     // below it
		{1, c - 0.5, 0.5, model.SignalBuy},   // within +0.5 ATR (1.0)
		{1, c + 1.5, -0.5, model.SignalBuy},  // dropped more than the demanded 0.5 ATR
		{1, c + 0.5, -0.5, model.SignalNone}, // dropped less
	}
	for _, k := range cases {
		md := fixture(bars[:a+2])
		withDailies(&md, dailiesAt(bars[0].t, 20, k.prevClose))
		p := entryParams()
		p.PrevCloseGate, p.PrevCloseATR = k.gate, k.allowance
		if sig := NewWithParams("TEST", stepM30, p).Decide(md); sig.Kind != k.want {
			t.Fatalf("gate %d, prev close %v, allowance %v, close %v: got %v, want %v", k.gate, k.prevClose, k.allowance, c, sig.Kind, k.want)
		}
	}
	// The fixture starts on Monday: a Saturday session closing below the price does not replace
	// Friday's close above it.
	md := fixture(bars[:a+2])
	ds := dailiesAt(bars[0].t, 20, c-1)
	sat := ds[len(ds)-1].t.AddDate(0, 0, 1)
	if sat.Weekday() != time.Saturday {
		t.Fatalf("fixture: %v must be Saturday", sat)
	}
	withDailies(&md, append(ds, bar{sat, c + 1, c + 1.1, c + 0.9, c + 1}))
	p := entryParams()
	p.PrevCloseGate = 1
	if sig := NewWithParams("TEST", stepM30, p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Saturday close above, Friday close below the price: got %v, want none", sig.Kind)
	}
}
