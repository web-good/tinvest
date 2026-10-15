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

// stepBar is the timeframe drop_bounce runs on; perHour bars make an hour.
const (
	stepBar = 15 * time.Minute
	perHour = 4
)

// path builds consecutive 15m bars from start (MSK): each bar opens at the previous close (the
// first at its own close) and its high/low span the open and the close.
func path(start string, closes []float64) []bar {
	t0 := at(start)
	out := make([]bar, len(closes))
	prev := closes[0]
	for i, c := range closes {
		out[i] = bar{t0.Add(time.Duration(i) * stepBar), prev, max(prev, c), min(prev, c), c}
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

// quarterHours expands hourly closes into 15m closes stepping evenly from the previous hourly close
// to this one, the fourth ending on the hourly close.
func quarterHours(hourly []float64) []float64 {
	out := make([]float64, 0, perHour*len(hourly))
	prev := hourly[0]
	for _, c := range hourly {
		for q := 1; q <= perHour; q++ {
			out = append(out, prev+(c-prev)*float64(q)/perHour)
		}
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
	return NewWithParams("TEST", stepBar, p).Decide(md)
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
// Wednesday 18:45 and the confirmation bars stay on Wednesday.
const signalStart = "2026-09-14 00:00"

// entryFixture is the hourly trendCloses(drops) as 15m bars from start, plus `after` extra 15m
// closes. The signal hour (the last of trendCloses) completes on bar armBar.
func entryFixture(start string, drops int, after ...float64) (bars []bar, armBar int) {
	c := quarterHours(trendCloses(drops))
	armBar = len(c) - 1
	return path(start, append(c, after...)), armBar
}

// signalThenGreenHour is the entry fixture followed by one hour of four 15m bars at +15% over the
// signal hour's close: the 15m histogram turns green on a+1, the hourly MACD(8,26,5) histogram is
// red on the signal hour (done on a) and green on the next hour (done on g = a+4).
func signalThenGreenHour() (bars []bar, a, g int) {
	base, a := entryFixture(signalStart, 6)
	j := base[a].c * 1.15
	bars, _ = entryFixture(signalStart, 6, j, j, j, j)
	return bars, a, a + perHour
}

// greenAfterSignal is signalThenGreenHour for tests that need only the buying bar g.
func greenAfterSignal() (bars []bar, g int) {
	bars, _, g = signalThenGreenHour()
	return bars, g
}

// histOf is the 15m MACD(12,26,9) histogram over the bars, as the strategy computes it.
func histOf(bars []bar) []float64 {
	cl := make([]float64, len(bars))
	for i, b := range bars {
		cl[i] = b.c
	}
	return macdHistogram(cl, DefaultParams())
}

// decideAt runs a fresh strategy over bars[:i+1].
func decideAt(p Params, bars []bar, i int) model.Signal {
	return NewWithParams("TEST", stepBar, p).Decide(fixture(bars[:i+1]))
}

func TestDefaultParams(t *testing.T) {
	want := Params{RSIPeriod: 6, RSIZone: 30, RSICap: 50, ZoneEntry: 1, EMAPeriod: 200,
		StopDailyATR: 1.0, DailyATRPeriod: 14, RSIExitPeriod: 10, RSIExit: 70,
		HourMACDFast: 8, HourMACDSlow: 26, HourMACDSignal: 5, HourMACDHours: 20,
		MACDFast: 12, MACDSlow: 26, MACDSignal: 9, ConfirmBars: 6, FridayTo: 11}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestLookback(t *testing.T) {
	cases := []struct {
		p    Params
		want int
	}{
		{DefaultParams(), 2484},                    // 4 × (600 + 20 hours + 1)
		{Params{RSIPeriod: 6, EMAPeriod: 10}, 260}, // 4 × (64 + 1)
		{Params{RSIPeriod: 6, EMAPeriod: 10, RSIExitPeriod: 20}, 332},
		{Params{RSIPeriod: 6, EMAPeriod: 10, HourMACDSlow: 26, HourMACDSignal: 5, HourMACDHours: 20}, 456}, // 4 × (93 + 20 + 1)
		{Params{RSIPeriod: 6, EMAPeriod: 10, MACDSlow: 100, MACDSignal: 9, ConfirmBars: 4}, 332},
	}
	for _, c := range cases {
		if got := NewWithParams("T", stepBar, c.p).Lookback(); got != c.want {
			t.Fatalf("Lookback(%+v) = %d, want %d", c.p, got, c.want)
		}
	}
}

func TestHourBars(t *testing.T) {
	bars := path("2026-09-16 10:00", []float64{1, 2, 3, 4, 5, 6})
	// The 11:30 and 11:45 bars are missing: 11:00 is completed by the 12:00 bar.
	bars = append(bars, bar{at("2026-09-16 12:00"), 7, 7, 7, 7}, bar{at("2026-09-16 12:15"), 8, 8, 8, 8},
		bar{at("2026-09-16 12:30"), 9, 9, 9, 9}, bar{at("2026-09-16 12:45"), 10, 10, 10, 10}, bar{at("2026-09-16 13:00"), 11, 11, 11, 11})
	hs := NewWithParams("T", stepBar, DefaultParams()).hourBars(fixture(bars))
	if len(hs) != 3 {
		t.Fatalf("hours = %+v, want 3 completed (13:00 still forming)", hs)
	}
	want := []struct {
		start      string
		close      float64
		last, done int
	}{{"2026-09-16 10:00", 4, 3, 3}, {"2026-09-16 11:00", 6, 5, 6}, {"2026-09-16 12:00", 10, 9, 9}}
	for k, w := range want {
		if !hs[k].start.Equal(at(w.start)) || hs[k].close != w.close || hs[k].last != w.last || hs[k].done != w.done {
			t.Fatalf("hour %d = %+v, want %+v", k, hs[k], w)
		}
	}
}

// The signal hour waits for the hourly MACD: the 15m histogram is already green on a+1, but nothing
// buys until the next hour completes green on g; then the 15m histogram, green on the arming bar,
// buys at the next bar's open.
func TestEnterWaitsForHourlyThen15m(t *testing.T) {
	bars, a, g := signalThenGreenHour()
	p := entryParams()
	cl := trendCloses(6)
	if r := indicators.RSISeries(cl, 6); r[len(r)-1] >= 30 || cl[len(cl)-1] <= ema.Compute(cl, 30)[len(cl)-1] {
		t.Fatal("fixture: the last hour must have RSI(6) < 30 above EMA(30)")
	}
	s := NewWithParams("TEST", stepBar, p)
	hs := s.hourBars(fixture(bars))
	hh := hourHistogram(hourCloses(hs), p)
	h := histOf(bars)
	if k := len(hs) - 1; hh == nil || hs[k].done != g || isRed(hh[k]) || !isRed(hh[k-1]) || isRed(h[a+1]) || isRed(h[g]) {
		t.Fatal("fixture: hourly histogram red on the signal hour and green on the hour done at g, 15m green from a+1")
	}
	for i := a; i < g; i++ {
		if sig := decideAt(p, bars, i); sig.Kind != model.SignalNone {
			t.Fatalf("bar %d before the hourly MACD is green: got %v %q, want none", i, sig.Kind, sig.EntryReason)
		}
	}
	sig := decideAt(p, bars, g)
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "ZONE: час ") ||
		!strings.Contains(sig.EntryReason, "MACD 1h") || !strings.Contains(sig.EntryReason, "MACD 15m") {
		t.Fatalf("green hour: got %v %q, want Buy ZONE with both MACDs", sig.Kind, sig.EntryReason)
	}
	if want := bars[g].t.Add(stepBar); !sig.FillBy.Equal(want) || math.Abs(sig.ATR-fixtureATR) > eps {
		t.Fatalf("FillBy/ATR = %v/%v, want %v/%v", sig.FillBy, sig.ATR, want, fixtureATR)
	}
	// A green hour later than HourMACDHours after the signal does not arm.
	p.HourMACDHours = 0
	if sig := decideAt(p, bars, g); sig.Kind != model.SignalNone {
		t.Fatalf("HourMACDHours 0, green one hour after the signal: got %v, want none", sig.Kind)
	}
}

// armedBy: the first green hourly histogram at most HourMACDHours after the latest signal hour arms
// the 15m window; later green hours do not re-arm, a stale signal does not arm.
func TestArmedBy(t *testing.T) {
	hours := func(n int) []hourBar {
		hs := make([]hourBar, n)
		for k := range hs {
			hs[k].done = perHour*k + perHour - 1
		}
		return hs
	}
	sigAt := func(ks ...int) func(int) bool {
		return func(k int) bool {
			for _, x := range ks {
				if x == k {
					return true
				}
			}
			return false
		}
	}
	redUntil := func(n, green int) []float64 { // red before `green`, green from it on
		hh := make([]float64, n)
		for k := range hh {
			hh[k] = -1
			if k >= green {
				hh[k] = 1
			}
		}
		return hh
	}
	p := DefaultParams()
	p.ConfirmBars, p.HourMACDHours = 6, 20
	s := NewWithParams("T", stepBar, p)
	hs := hours(40)
	cases := []struct {
		name         string
		signal       func(int) bool
		hh           []float64
		i            int
		wantH, wantK int
	}{
		{"green after the signal", sigAt(2), redUntil(40, 4), hs[4].done, 4, 2},
		{"window still open", sigAt(2), redUntil(40, 4), hs[4].done + 5, 4, 2},
		{"window over", sigAt(2), redUntil(40, 4), hs[4].done + 6, -1, -1},
		{"signal hour already green", sigAt(5), redUntil(40, 0), hs[5].done, 5, 5},
		{"green before the signal: later green hours do not arm", sigAt(5), redUntil(40, 0), hs[7].done, -1, -1},
		{"signal 22 hours before the green hour", sigAt(0), redUntil(40, 22), hs[22].done, -1, -1},
		{"a later signal restarts the wait", sigAt(0, 5), redUntil(40, 22), hs[22].done, 22, 5},
		{"no signal", sigAt(), redUntil(40, 4), hs[4].done, -1, -1},
		{"no hourly histogram", sigAt(2), nil, hs[4].done, -1, -1},
	}
	for _, c := range cases {
		if h, k := s.armedBy(hs, c.signal, c.hh, c.i); h != c.wantH || k != c.wantK {
			t.Fatalf("%s: armedBy = %d, %d, want %d, %d", c.name, h, k, c.wantH, c.wantK)
		}
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

// An hour with RSI deep in the zone but its close below the EMA is no signal.
func TestEnterBelowEMANoSignal(t *testing.T) {
	cl := trendCloses(12)
	k := len(cl) - 1
	trend := ema.Compute(cl, 30)
	rsi := indicators.RSISeries(cl, 6)
	if rsi[k] >= 30 || cl[k] >= trend[k] {
		t.Fatal("fixture: the last hour must have RSI(6) < 30 below EMA(30)")
	}
	if got := NewWithParams("T", stepBar, entryParams()).hourSignal(cl, trend, rsi, k); got != "" {
		t.Fatalf("below EMA: hourSignal = %q, want none", got)
	}
}

// Only weekdays: the same signal shifted so the buying bar falls on Saturday does not buy.
func TestEnterWeekdaysOnly(t *testing.T) {
	bars, g := greenAfterSignal()
	shift := at("2026-09-19 12:45").Sub(bars[g].t)
	for i := range bars {
		bars[i].t = bars[i].t.Add(shift)
	}
	if sig := decideAt(entryParams(), bars, g); sig.Kind != model.SignalNone {
		t.Fatalf("Saturday: got %v, want none", sig.Kind)
	}
}

// On Friday the fill bar must start before FridayTo: a 10:00 fill buys, an 11:00 fill does not;
// FridayTo 24 lifts the limit, 0 closes Friday entirely. The buying bar completes an hour (:45).
func TestEnterFridayBeforeCutoff(t *testing.T) {
	cases := []struct {
		green    string
		fridayTo int
		want     model.SignalKind
	}{
		{"2026-09-18 09:45", 11, model.SignalBuy},
		{"2026-09-18 10:45", 11, model.SignalNone},
		{"2026-09-18 17:45", 11, model.SignalNone},
		{"2026-09-18 17:45", 24, model.SignalBuy},
		{"2026-09-18 07:45", 0, model.SignalNone},
		{"2026-09-17 17:45", 11, model.SignalBuy}, // Thursday evening is unaffected
	}
	for _, c := range cases {
		bars, g := greenAfterSignal()
		shift := at(c.green).Sub(bars[g].t)
		for i := range bars {
			bars[i].t = bars[i].t.Add(shift)
		}
		p := entryParams()
		p.FridayTo = c.fridayTo
		if sig := decideAt(p, bars, g); sig.Kind != c.want {
			t.Fatalf("buying bar %s, FridayTo %d: got %v, want %v", c.green, c.fridayTo, sig.Kind, c.want)
		}
	}
}

// ZoneEntry 0 leaves only CROSS: the ZONE signal of the fixture no longer buys.
func TestZoneEntryOff(t *testing.T) {
	bars, g := greenAfterSignal()
	p := entryParams()
	p.ZoneEntry = 0
	if sig := decideAt(p, bars, g); sig.Kind != model.SignalNone {
		t.Fatalf("ZoneEntry 0, ZONE signal: got %v %q, want none", sig.Kind, sig.EntryReason)
	}
}

func TestEnterZeroParamsGiveNoEntries(t *testing.T) {
	bars, g := greenAfterSignal()
	for name, mut := range map[string]func(*Params){
		"EMAPeriod":   func(p *Params) { p.EMAPeriod = 0 },
		"RSIPeriod":   func(p *Params) { p.RSIPeriod = 0 },
		"ConfirmBars": func(p *Params) { p.ConfirmBars = 0 },
		"MACD":        func(p *Params) { p.MACDFast = p.MACDSlow },
		"HourMACD":    func(p *Params) { p.HourMACDFast = p.HourMACDSlow },
	} {
		p := entryParams()
		mut(&p)
		if sig := decideAt(p, bars, g); sig.Kind != model.SignalNone {
			t.Fatalf("%s off: got %v, want none", name, sig.Kind)
		}
	}
}

func TestEnterNeedsDailyATR(t *testing.T) {
	bars, g := greenAfterSignal()
	md := fixture(bars[:g+1])
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
	if sig := NewWithParams("TEST", stepBar, entryParams()).Decide(md); sig.Kind != model.SignalNone {
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
	bars, g := greenAfterSignal()
	p := entryParams()
	sig := decideAt(p, bars, g)
	if want := bars[g].c - p.StopDailyATR*fixtureATR; sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-want) > eps {
		t.Fatalf("got %v stop %v, want Buy with stop %v", sig.Kind, sig.StopLoss, want)
	}
	p.StopDailyATR = 0
	if sig := decideAt(p, bars, g); sig.Kind != model.SignalBuy || sig.StopLoss != 0 {
		t.Fatalf("stop off: got %v stop %v, want Buy without a stop", sig.Kind, sig.StopLoss)
	}
}
