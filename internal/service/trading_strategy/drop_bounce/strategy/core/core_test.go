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

// path builds consecutive 15m bars from start (MSK): each bar opens at the previous close (the
// first at its own close) and its high/low span the open and the close.
func path(start string, closes []float64) []bar {
	t0 := at(start)
	out := make([]bar, len(closes))
	prev := closes[0]
	for i, c := range closes {
		out[i] = bar{t0.Add(time.Duration(i) * BarStep), prev, max(prev, c), min(prev, c), c}
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
	return md
}

func decide(p Params, bars []bar) model.Signal {
	return NewWithParams("TEST", p).Decide(fixture(bars))
}

// held decides with a position bought at entry on the bar starting at entryTime.
func held(p Params, bars []bar, entry float64, entryTime time.Time) model.Signal {
	md := fixture(bars)
	md.Position = &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryTime: entryTime}
	return NewWithParams("TEST", p).Decide(md)
}

// dropAfter appends to closes one bar falling pct percent from the last close.
func dropAfter(closes []float64, pct float64) []float64 {
	return append(closes, closes[len(closes)-1]*(1-pct/100))
}

// downtrend is 20 closes sliding 0.1% per bar: RSI(6) sits near 0.
func downtrend() []float64 { return steps(100, 20, -0.1) }

func TestDefaultParams(t *testing.T) {
	want := Params{MinDropPct: 0.6, RSIPeriod: 6, RSIEntry: 30, RSIExit: 70, MACDExit: 1, MACDFast: 9, MACDSlow: 14,
		MACDSignal: 3, StochPeriod: 14, StochSmooth: 3, StochExit: 80, ProfitPct: 1, StopPct: 1, StartHour: 8}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestEnterBuysOversoldDrop(t *testing.T) {
	bars := path("2026-09-14 08:00", dropAfter(downtrend(), 0.6))
	sig := decide(DefaultParams(), bars)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
	if want := bars[len(bars)-1].t.Add(BarStep); !sig.FillBy.Equal(want) {
		t.Fatalf("FillBy = %v, want %v: the fill must be the very next 15m slot", sig.FillBy, want)
	}
	if sig.StopLoss != 0 || sig.TakeProfit != 0 {
		t.Fatalf("levels = %v/%v, want none: they are measured from the fill", sig.StopLoss, sig.TakeProfit)
	}
	if !strings.HasPrefix(sig.EntryReason, "DROP:") {
		t.Fatalf("reason = %q", sig.EntryReason)
	}
}

func TestEnterRejectsShallowDrop(t *testing.T) {
	if sig := decide(DefaultParams(), path("2026-09-14 08:00", dropAfter(downtrend(), 0.59))); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None for 0.59%%", sig.Kind)
	}
}

func TestEnterRejectsRSIAtOrAboveEntry(t *testing.T) {
	closes := dropAfter(steps(100, 20, 1.0), 0.6) // strong uptrend: RSI(6) stays high after the drop
	if sig := decide(DefaultParams(), path("2026-09-14 08:00", closes)); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None: RSI is not oversold", sig.Kind)
	}
	p := DefaultParams()
	p.RSIEntry = 100
	if sig := decide(p, path("2026-09-14 08:00", closes)); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy with RSIEntry 100 (the drop itself qualifies)", sig.Kind)
	}
}

func TestEnterNeedsRSIHistory(t *testing.T) {
	closes := dropAfter(steps(100, 5, -0.1), 0.6) // 6 closes: RSI(6) has no value yet
	if sig := decide(DefaultParams(), path("2026-09-14 08:00", closes)); sig.Kind != model.SignalNone {
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
		if sig := decide(DefaultParams(), path(c.start, closes)); sig.Kind != c.want {
			t.Errorf("%s: kind = %v, want %v", c.name, sig.Kind, c.want)
		}
	}
}

func TestEnterRejectsFillOnNextDate(t *testing.T) {
	closes := dropAfter(downtrend(), 0.6)
	if sig := decide(DefaultParams(), path("2026-09-14 18:45", closes)); sig.Kind != model.SignalNone { // drop at 23:45
		t.Fatalf("kind = %v, want None: the fill would be the next morning", sig.Kind)
	}
}

func TestEnterInvalidParams(t *testing.T) {
	bars := path("2026-09-14 08:00", dropAfter(downtrend(), 0.6))
	for name, mut := range map[string]func(*Params){
		"MinDropPct 0": func(p *Params) { p.MinDropPct = 0 },
		"RSIPeriod 0":  func(p *Params) { p.RSIPeriod = 0 },
		"RSIEntry 0":   func(p *Params) { p.RSIEntry = 0 },
	} {
		p := DefaultParams()
		mut(&p)
		if sig := decide(p, bars); sig.Kind != model.SignalNone {
			t.Errorf("%s: kind = %v, want None", name, sig.Kind)
		}
	}
}

// exitsOff leaves only the stop on.
func exitsOff() Params {
	p := DefaultParams()
	p.RSIExit, p.StochExit, p.MACDExit, p.ProfitPct = 0, 0, 0, 0
	return p
}

// flatWithLast is 30 flat bars at 100 and a last bar with the given open/high/low/close.
func flatWithLast(o, h, l, c float64) []bar {
	bars := path("2026-09-14 08:00", flatN(30, 100))
	last := bars[len(bars)-1]
	return append(bars, bar{last.t.Add(BarStep), o, h, l, c})
}

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
	p.StopPct = 0
	bars = flatWithLast(100, 100, 50, 60)
	if sig := held(p, bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("StopPct 0: kind = %v, want None", sig.Kind)
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

func TestEnterEMAFilter(t *testing.T) {
	// 40 rising closes put EMA(20) far below a short dip; a downtrend puts it above the dip.
	above := dropAfter(append(steps(100, 40, 0.5), steps(100*math.Pow(1.005, 40), 6, -0.3)...), 0.6)
	below := dropAfter(steps(100, 40, -0.1), 0.6)
	p := DefaultParams()
	p.RSIEntry = 100 // isolate the trend filter from RSI
	p.EMAPeriod = 20
	if sig := decide(p, path("2026-09-14 08:00", above)); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy: close is above EMA(20)", sig.Kind)
	} else if !strings.Contains(sig.EntryReason, "EMA(20)") {
		t.Fatalf("reason = %q, want the EMA value", sig.EntryReason)
	}
	if sig := decide(p, path("2026-09-14 08:00", below)); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None: close is below EMA(20)", sig.Kind)
	}
	if sig := decide(p, path("2026-09-14 08:00", below[len(below)-10:])); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None on EMA warm-up", sig.Kind)
	}
	p.EMAPeriod = 0
	if sig := decide(p, path("2026-09-14 08:00", below)); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy with the filter off", sig.Kind)
	}
}

func TestLookbackCoversEMA(t *testing.T) {
	p := DefaultParams()
	p.EMAPeriod = 200
	if got := NewWithParams("TEST", p).Lookback(); got < 2*200 {
		t.Fatalf("Lookback = %d, want ≥ 400 for EMA(200)", got)
	}
}
