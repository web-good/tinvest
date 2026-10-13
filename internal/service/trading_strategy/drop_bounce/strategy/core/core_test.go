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

// trendCloses is 60 zigzag bars (+2%, −0.5%) from 100, then `drops` bars of −1% each. With
// RSI(6) and EMA(30): drops=5 leaves RSI 33.52 (above the zone, no cross), drops=6 → RSI 28.34
// with the close 146.73 above EMA 144.43, drops=12 → RSI deep in the zone with the close below
// the EMA.
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

// withLast appends one more close mult times the last.
func withLast(c []float64, mult float64) []float64 {
	return append(append([]float64(nil), c...), c[len(c)-1]*mult)
}

// entryEnd is the start of the decision bar: Wednesday 12:00 MSK, inside the window.
const entryEnd = "2026-09-16 12:00"

// endingAt builds 1h bars over closes whose last bar starts at end (MSK).
func endingAt(end string, closes []float64) []bar {
	start := at(end).Add(-time.Duration(len(closes)-1) * stepH1)
	return path(start.Format("2006-01-02 15:04"), closes)
}

// entryParams is the default params with EMA(30): the fixtures are ~70 bars long.
func entryParams() Params {
	p := DefaultParams()
	p.EMAPeriod = 30
	return p
}

// decide runs a fresh strategy over closes ending at end.
func decide(p Params, end string, closes []float64) model.Signal {
	return NewWithParams("TEST", stepH1, p).Decide(fixture(endingAt(end, closes)))
}

// rsiAt / emaAt are the fixture's indicator values at the last bar, for guards.
func rsiAt(c []float64, period int) (prev, cur float64) {
	r := indicators.RSISeries(c, period)
	return r[len(r)-2], r[len(r)-1]
}

func emaAt(c []float64, period int) float64 { return ema.Compute(c, period)[len(c)-1] }

func TestDefaultParams(t *testing.T) {
	want := Params{RSIPeriod: 6, RSIZone: 30, RSICap: 50, EMAPeriod: 200, DaySpentATR: 0.7,
		StopDailyATR: 1.0, DailyATRPeriod: 14, RSIExit: 70, StartHour: 7,
		MACDFast: 12, MACDSlow: 26, MACDSignal: 9, StaleBars: 10, StaleATR: 0.3}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestLookback(t *testing.T) {
	cases := []struct {
		p    Params
		want int
	}{
		{DefaultParams(), 600},
		{Params{RSIPeriod: 6, EMAPeriod: 10}, 64},
		{Params{RSIPeriod: 20, EMAPeriod: 10}, 82},
		{Params{RSIPeriod: 6, EMAPeriod: 10, MACDSlow: 26, MACDSignal: 9}, 105},
	}
	for _, c := range cases {
		if got := NewWithParams("T", stepH1, c.p).Lookback(); got != c.want {
			t.Fatalf("Lookback(%+v) = %d, want %d", c.p, got, c.want)
		}
	}
}

func TestEnterZoneAboveEMA(t *testing.T) {
	c := trendCloses(6)
	if _, cur := rsiAt(c, 6); cur >= 30 || c[len(c)-1] <= emaAt(c, 30) {
		t.Fatalf("fixture: RSI %.2f must be < 30 and close %.2f > EMA %.2f", cur, c[len(c)-1], emaAt(c, 30))
	}
	sig := decide(entryParams(), entryEnd, c)
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "ZONE") {
		t.Fatalf("got %v %q, want Buy ZONE", sig.Kind, sig.EntryReason)
	}
	if math.Abs(sig.ATR-fixtureATR) > eps {
		t.Fatalf("ATR = %v, want %v", sig.ATR, fixtureATR)
	}
	if want := at(entryEnd).Add(stepH1); !sig.FillBy.Equal(want) {
		t.Fatalf("FillBy = %v, want %v", sig.FillBy, want)
	}
}

func TestEnterZoneBelowEMANoSignal(t *testing.T) {
	c := trendCloses(12)
	if _, cur := rsiAt(c, 6); cur >= 30 || c[len(c)-1] >= emaAt(c, 30) {
		t.Fatalf("fixture: RSI %.2f must be < 30 and close %.2f < EMA %.2f", cur, c[len(c)-1], emaAt(c, 30))
	}
	if sig := decide(entryParams(), entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("below EMA: got %v %q, want none", sig.Kind, sig.EntryReason)
	}
}

func TestTrendUpIsStrict(t *testing.T) {
	if trendUp(100, 100) || trendUp(100, 0) || !trendUp(100.01, 100) {
		t.Fatal("trendUp must need close strictly above a warmed (non-zero) EMA")
	}
}

func TestEnterCrossUnderCap(t *testing.T) {
	c := withLast(trendCloses(6), 1.01)
	if prev, cur := rsiAt(c, 6); !(prev < 30 && cur >= 30 && cur <= 50) {
		t.Fatalf("fixture: RSI %.2f → %.2f must cross 30 and stay ≤ 50", prev, cur)
	}
	sig := decide(entryParams(), entryEnd, c)
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "CROSS") {
		t.Fatalf("got %v %q, want Buy CROSS", sig.Kind, sig.EntryReason)
	}
}

func TestEnterCrossAboveCap(t *testing.T) {
	c := withLast(trendCloses(6), 1.03)
	if prev, cur := rsiAt(c, 6); !(prev < 30 && cur > 50 && cur <= 60) {
		t.Fatalf("fixture: RSI %.2f → %.2f must cross 30 and land in (50, 60]", prev, cur)
	}
	if sig := decide(entryParams(), entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("above cap 50: got %v %q, want none", sig.Kind, sig.EntryReason)
	}
	p := entryParams()
	p.RSICap = 60
	if sig := decide(p, entryEnd, c); sig.Kind != model.SignalBuy {
		t.Fatalf("cap 60: got %v, want Buy", sig.Kind)
	}
}

func TestEnterNoSignalWhenRSIAboveZoneWithoutCross(t *testing.T) {
	c := trendCloses(5)
	if prev, cur := rsiAt(c, 6); prev < 30 || cur < 30 {
		t.Fatalf("fixture: RSI %.2f → %.2f must stay at/above 30", prev, cur)
	}
	if sig := decide(entryParams(), entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("got %v %q, want none", sig.Kind, sig.EntryReason)
	}
	// A flat tape has RSI exactly 50 on every bar (no losses): never a zone, never a cross.
	if sig := decide(entryParams(), entryEnd, flatN(70, 100)); sig.Kind != model.SignalNone {
		t.Fatalf("flat: got %v, want none", sig.Kind)
	}
}

func TestEnterNeedsWarmRSI(t *testing.T) {
	p := entryParams()
	p.EMAPeriod = 2 // keep the EMA warm so only the RSI warm-up is under test
	p.RSICap = 100  // a false CROSS from the warm-up zero must not be stopped by the cap
	// Rising tape: the close is above the EMA, so only the RSI guard can block. 7 closes: RSI(6)
	// has one value and the previous one is the warm-up zero (0 -> cur would read as a CROSS).
	c := steps(100, 7, +1)
	if sig := decide(p, entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("got %v %q, want none before RSI has two values", sig.Kind, sig.EntryReason)
	}
	// Contrast: same params, one more bar (warm RSI), a fall then a bounce -> a real ZONE entry.
	warm := append(steps(100, 7, -1), 94.5)
	if prev, cur := rsiAt(warm, 6); cur <= 0 || cur >= 30 || warm[len(warm)-1] <= emaAt(warm, 2) {
		t.Fatalf("fixture: RSI %.2f -> %.2f must be warm (>0) and < 30, close above EMA", prev, cur)
	}
	if sig := decide(p, entryEnd, warm); sig.Kind != model.SignalBuy {
		t.Fatalf("warm: got %v %q, want Buy", sig.Kind, sig.EntryReason)
	}
}

func TestEnterNeedsWarmEMA(t *testing.T) {
	c := trendCloses(6) // 67 closes
	p := entryParams()
	p.EMAPeriod = len(c) + 1
	if sig := decide(p, entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("got %v, want none with an unwarmed EMA", sig.Kind)
	}
}

func TestEnterDayGate(t *testing.T) {
	c := trendCloses(6)
	p := entryParams()
	p.DaySpentATR = 0.5 // threshold 0.5 · fixtureATR 2 = 1.0
	run := func(hi, lo float64) model.Signal {
		md := fixture(endingAt(entryEnd, c))
		md.TodayHigh, md.TodayLow = hi, lo
		return NewWithParams("TEST", stepH1, p).Decide(md)
	}
	if sig := run(100.99, 100); sig.Kind != model.SignalBuy {
		t.Fatalf("used 0.99 < 1.0: got %v, want Buy", sig.Kind)
	}
	if sig := run(101, 100); sig.Kind != model.SignalNone {
		t.Fatalf("used exactly 1.0: got %v, want none (spent)", sig.Kind)
	}
	if sig := run(103, 100); sig.Kind != model.SignalNone {
		t.Fatalf("used 3.0: got %v, want none", sig.Kind)
	}
	for _, ext := range [][2]float64{{0, 0}, {101, 0}, {0, 100}, {100, 101}} {
		if sig := run(ext[0], ext[1]); sig.Kind != model.SignalNone {
			t.Fatalf("broken extent %v: got %v, want none", ext, sig.Kind)
		}
	}
	md := fixture(endingAt(entryEnd, c))
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
	if sig := NewWithParams("TEST", stepH1, p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("no daily ATR: got %v, want none", sig.Kind)
	}
}

func TestEnterZeroParamsGiveNoEntries(t *testing.T) {
	c := trendCloses(6)
	for name, mut := range map[string]func(*Params){
		"DaySpentATR": func(p *Params) { p.DaySpentATR = 0 },
		"EMAPeriod":   func(p *Params) { p.EMAPeriod = 0 },
		"RSIPeriod":   func(p *Params) { p.RSIPeriod = 0 },
	} {
		p := entryParams()
		mut(&p)
		if sig := decide(p, entryEnd, c); sig.Kind != model.SignalNone {
			t.Fatalf("%s = 0: got %v, want none", name, sig.Kind)
		}
	}
}

func TestEnterStopGuard(t *testing.T) {
	c := trendCloses(6)
	p := entryParams()
	p.StopDailyATR = c[len(c)-1]/fixtureATR + 1 // stop below zero
	if sig := decide(p, entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("stop ≤ 0: got %v, want none", sig.Kind)
	}
}

func TestEnterWindow(t *testing.T) {
	c := trendCloses(6)
	for _, end := range []string{
		"2026-09-19 12:00", // Saturday
		"2026-09-20 12:00", // Sunday
		"2026-09-16 06:00", // before StartHour 7
		"2026-09-16 23:00", // fill bar would be the next date
		"2026-09-18 12:00", // Friday from 12:00
		"2026-09-18 15:00", // Friday afternoon
	} {
		if sig := decide(entryParams(), end, c); sig.Kind != model.SignalNone {
			t.Fatalf("%s: got %v, want none", end, sig.Kind)
		}
	}
	for _, end := range []string{"2026-09-16 07:00", "2026-09-16 15:00", "2026-09-18 11:00"} {
		if sig := decide(entryParams(), end, c); sig.Kind != model.SignalBuy {
			t.Fatalf("%s: got %v, want Buy", end, sig.Kind)
		}
	}
}
