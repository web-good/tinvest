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

// at parses "2006-01-02 15:04" as MSK.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, mskLoc)
	if err != nil {
		panic(err)
	}
	return t
}

type bar struct {
	t          string
	o, h, l, c float64
}

// flat returns flat bars (o=h=l=c=p) every 5 minutes over [from, to], both MSK.
func flat(from, to string, p float64) []bar {
	var out []bar
	for t := at(from); !t.After(at(to)); t = t.Add(5 * time.Minute) {
		out = append(out, bar{t.Format("2006-01-02 15:04"), p, p, p, p})
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

// m5 is the defaults with the switch to 30m turned off: the whole day trades on 5m bars.
func m5() Params {
	p := DefaultParams()
	p.SwitchHour = 24
	return p
}

func newS(p Params) *Strategy { return NewWithParams("TEST", 5*time.Minute, p) }

// warm prepends 12 flat bars at 102 ending 5 minutes before bars[0]: RSI is warm and every
// entry fixture comes off a drop, so the RSI entry gate (≤ 50) passes unless a test wants it not to.
func warm(bars []bar) []bar {
	var out []bar
	for k := 12; k >= 1; k-- {
		ts := at(bars[0].t).Add(-time.Duration(k) * 5 * time.Minute).Format("2006-01-02 15:04")
		out = append(out, bar{ts, 102, 102, 102, 102})
	}
	return append(out, bars...)
}

func decide(p Params, bars []bar) model.Signal { return newS(p).Decide(fixture(warm(bars))) }

// Monday 2026-09-14. PDL 99, PDH 101 (Friday's daily), daily ATR 2.
var (
	pre    = bar{"2026-09-14 09:55", 99.5, 99.5, 99.5, 99.5}
	barA   = bar{"2026-09-14 10:00", 99.2, 99.5, 98.8, 99.3}    // green, low < PDL, close > PDL
	redB   = bar{"2026-09-14 09:55", 98.9, 98.95, 98.5, 98.6}   // red, close < PDL
	greenB = bar{"2026-09-14 10:00", 98.55, 98.95, 98.4, 98.92} // close > red open, low < red low, close < PDL (not A)
)

func TestEntryA(t *testing.T) {
	sig := decide(m5(), []bar{pre, barA})
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.StopLoss-98.3) > eps || math.Abs(sig.TakeProfit-101) > eps || math.Abs(sig.ATR-2) > eps || math.Abs(sig.Level-99) > eps {
		t.Fatalf("SL/TP/ATR/Level = %v/%v/%v/%v, want 98.3/101/2/99", sig.StopLoss, sig.TakeProfit, sig.ATR, sig.Level)
	}
	if !sig.FillBy.Equal(at("2026-09-14 10:05")) {
		t.Fatalf("FillBy = %v, want 10:05", sig.FillBy)
	}
	if !strings.HasPrefix(sig.EntryReason, "A") {
		t.Fatalf("EntryReason = %q, want prefix A", sig.EntryReason)
	}
}

func TestEntryANeedsEveryCondition(t *testing.T) {
	for name, b := range map[string]bar{
		"red":             {"2026-09-14 10:00", 99.4, 99.5, 98.8, 99.3},
		"low at PDL":      {"2026-09-14 10:00", 99.2, 99.5, 99.0, 99.3},
		"close at PDL":    {"2026-09-14 10:00", 98.9, 99.5, 98.8, 99.0},
		"close below PDL": {"2026-09-14 10:00", 98.85, 99.5, 98.8, 98.95},
	} {
		if sig := decide(m5(), []bar{pre, b}); sig.Kind != model.SignalNone {
			t.Errorf("%s: Kind = %v, want None", name, sig.Kind)
		}
	}
}

func TestEntryB(t *testing.T) {
	sig := decide(m5(), []bar{redB, greenB})
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.StopLoss-97.92) > eps || math.Abs(sig.TakeProfit-101) > eps {
		t.Fatalf("SL/TP = %v/%v, want 97.92/101", sig.StopLoss, sig.TakeProfit)
	}
	if !strings.HasPrefix(sig.EntryReason, "B") {
		t.Fatalf("EntryReason = %q, want prefix B", sig.EntryReason)
	}
}

func TestEntryBNeedsEveryCondition(t *testing.T) {
	for name, pair := range map[string][2]bar{
		"close equals red open": {redB, {"2026-09-14 10:00", 98.55, 98.95, 98.4, 98.9}},
		"low equals red low":    {redB, {"2026-09-14 10:00", 98.55, 98.95, 98.5, 98.92}},
		"previous is a doji":    {{"2026-09-14 09:55", 98.6, 98.95, 98.5, 98.6}, greenB},
		"current is red":        {redB, {"2026-09-14 10:00", 98.95, 98.95, 98.4, 98.92}},
		"different dates":       {{"2026-09-11 23:55", 98.9, 98.95, 98.5, 98.6}, {"2026-09-14 07:00", 98.55, 98.95, 98.4, 98.92}},
	} {
		if sig := decide(m5(), pair[:]); sig.Kind != model.SignalNone {
			t.Errorf("%s: Kind = %v, want None", name, sig.Kind)
		}
	}
}

// The engulfing is a reversal anywhere: away from the levels (here around 100) it only needs the RSI
// of the red bar below 50.
func TestEntryBAnywhere(t *testing.T) {
	away := []bar{{"2026-09-14 09:55", 100.4, 100.45, 100.05, 100.1}, {"2026-09-14 10:00", 100.05, 100.5, 100.0, 100.45}}
	sig := decide(m5(), away)
	if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "разворот вне уровней, RSI(6) 5m") {
		t.Fatalf("got %v %q, want Buy B away from the levels", sig.Kind, sig.EntryReason)
	}
	if math.Abs(sig.Level-99) > eps || math.Abs(sig.TakeProfit-101) > eps {
		t.Fatalf("Level/TP = %v/%v, want 99/101", sig.Level, sig.TakeProfit)
	}
}

func TestEntryBothPatternsReportA(t *testing.T) {
	both := []bar{{"2026-09-14 09:55", 99.2, 99.2, 98.7, 98.8}, {"2026-09-14 10:00", 98.75, 99.4, 98.6, 99.3}}
	if sig := decide(m5(), both); sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "A") {
		t.Fatalf("both: %v %q, want Buy with reason A", sig.Kind, sig.EntryReason)
	}
	p := m5()
	p.EntryA = 0
	if sig := decide(p, both); sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "B") {
		t.Fatalf("EntryA off: %v %q, want Buy with reason B", sig.Kind, sig.EntryReason)
	}
}

func TestEntrySwitches(t *testing.T) {
	p := m5()
	p.EntryA = 0
	if sig := decide(p, []bar{pre, barA}); sig.Kind != model.SignalNone {
		t.Errorf("EntryA off: Kind = %v, want None", sig.Kind)
	}
	p = m5()
	p.EntryB = 0
	if sig := decide(p, []bar{redB, greenB}); sig.Kind != model.SignalNone {
		t.Errorf("EntryB off: Kind = %v, want None", sig.Kind)
	}
}

func TestEntryWindow(t *testing.T) {
	for ts, want := range map[string]bool{
		"2026-09-14 06:55": false,
		"2026-09-14 07:00": true,
		"2026-09-14 17:50": true,  // buy at 17:55
		"2026-09-14 17:55": false, // buy at 18:00 is too late
		"2026-09-12 10:00": false, // Saturday
	} {
		b := barA
		b.t = ts
		p := pre
		p.t = at(ts).Add(-5 * time.Minute).Format("2006-01-02 15:04")
		if got := decide(m5(), []bar{p, b}).Kind == model.SignalBuy; got != want {
			t.Errorf("%s: buy = %v, want %v", ts, got, want)
		}
	}
}

// Review focus 1: with the window widened to midnight the fill bar of a 23:55 signal is the next
// date — no entry.
func TestEntryFillMustBeSameDate(t *testing.T) {
	p := m5()
	p.EntryToHour = 24
	b := barA
	b.t = "2026-09-14 23:55"
	q := pre
	q.t = "2026-09-14 23:50"
	if sig := decide(p, []bar{q, b}); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
	b.t, q.t = "2026-09-14 23:50", "2026-09-14 23:45"
	if sig := decide(p, []bar{q, b}); sig.Kind != model.SignalBuy {
		t.Fatalf("23:50: Kind = %v, want Buy", sig.Kind)
	}
}

func TestEntryCloseAtPDH(t *testing.T) {
	wide := bar{"2026-09-14 10:00", 99.2, 101.2, 98.8, 101.0}
	if sig := decide(m5(), []bar{pre, wide}); sig.Kind != model.SignalNone {
		t.Fatalf("target on: Kind = %v, want None", sig.Kind)
	}
	p := m5()
	p.TakeProfitPDH = 0
	sig := decide(p, []bar{pre, wide})
	if sig.Kind != model.SignalBuy || sig.TakeProfit != 0 {
		t.Fatalf("target off: %v TP %v, want Buy with no target", sig.Kind, sig.TakeProfit)
	}
}

// Review focus 4: Sunday's daily (99.9..100.1) is not the reference day; Friday's (99..101) is.
func TestEntryIgnoresWeekendDaily(t *testing.T) {
	b := bar{"2026-09-14 10:00", 99.6, 99.97, 99.5, 99.95} // a reclaim of 99.9, not of 99
	if sig := decide(m5(), []bar{pre, b}); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
}

func TestEntryNeedsDailiesAndStop(t *testing.T) {
	md := fixture(warm([]bar{pre, barA}))
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
	if sig := newS(m5()).Decide(md); sig.Kind != model.SignalNone {
		t.Errorf("no dailies: Kind = %v, want None", sig.Kind)
	}
	p := m5()
	p.StopDailyATR = 0
	if sig := decide(p, []bar{pre, barA}); sig.Kind != model.SignalNone {
		t.Errorf("no stop: Kind = %v, want None", sig.Kind)
	}
}

func TestLookbackCoversWarmupAndDay(t *testing.T) {
	warm := 3 * (26 + 8) // MACD 9/26/8
	if got := newS(m5()).Lookback(); got != warm+dayBars {
		t.Fatalf("5m only: Lookback = %d, want %d", got, warm+dayBars)
	}
	if got := newS(DefaultParams()).Lookback(); got != 6*warm+dayBars {
		t.Fatalf("with 30m: Lookback = %d, want %d", got, 6*warm+dayBars)
	}
}

// rallyUp is 09:20–09:55 rising with small pullbacks to 99.25.
func rallyUp() []bar {
	closes := []float64{98.6, 98.8, 98.7, 98.9, 99.1, 99.0, 99.2, 99.25}
	var out []bar
	for k, c := range closes {
		ts := at("2026-09-14 09:20").Add(time.Duration(k) * 5 * time.Minute).Format("2006-01-02 15:04")
		out = append(out, bar{ts, c, c, c, c})
	}
	return out
}

// rally is a rise into the entry-A bar: RSI(6) is well above 50.
func rally() []bar { return append(rallyUp(), barA) }

// rallyB is the rise, then a small red bar (RSI stays above 50) and an engulfing that stays above
// PDL 99 — a reversal away from the levels.
func rallyB() []bar {
	return append(rallyUp(), bar{"2026-09-14 10:00", 99.3, 99.3, 99.1, 99.2}, bar{"2026-09-14 10:05", 99.15, 99.4, 99.05, 99.35})
}

func TestEntryBlockedWhenRSIAbove(t *testing.T) {
	if sig := newS(m5()).Decide(fixture(rallyB())); sig.Kind != model.SignalNone {
		t.Fatalf("reversal, RSI above 50: Kind = %v, want None", sig.Kind)
	}
	p := m5()
	p.RSIEntryMax = 0
	if sig := newS(p).Decide(fixture(rallyB())); sig.Kind != model.SignalBuy {
		t.Fatalf("gate off: Kind = %v, want Buy", sig.Kind)
	}
	// A crosses PDL: no RSI gate even after the rally
	if sig := newS(m5()).Decide(fixture(rally())); sig.Kind != model.SignalBuy {
		t.Fatalf("A after the rally: Kind = %v, want Buy", sig.Kind)
	}
}

// The gate is strict and reads the red bar before the engulfing: RSI equal to RSIEntryMax blocks.
func TestEntryRSIGateStrict(t *testing.T) {
	md := fixture(rallyB())
	n := len(md.Closes)
	rsi := indicators.RSISeries(md.Closes[:n-1], 6)[n-2]
	p := m5()
	p.RSIEntryMax = rsi
	if sig := newS(p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("red-bar RSI %.2f = max: Kind = %v, want None", rsi, sig.Kind)
	}
	p.RSIEntryMax = rsi + 0.01
	if sig := newS(p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("red-bar RSI %.2f < max: Kind = %v, want Buy", rsi, sig.Kind)
	}
}

// An engulfing crossing PDL upward is a level formation: no RSI gate.
func TestEntryBAtLevelSkipsRSI(t *testing.T) {
	p := m5()
	p.EntryA = 0
	bars := append(rallyUp(), bar{"2026-09-14 10:00", 99.3, 99.3, 98.9, 99.2}, bar{"2026-09-14 10:05", 99.15, 99.4, 98.85, 99.4})
	sig := newS(p).Decide(fixture(bars))
	if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "пересечение PDL") {
		t.Fatalf("got %v %q, want Buy B at PDL", sig.Kind, sig.EntryReason)
	}
}

// Without enough closes to compute RSI the gate refuses a reversal.
func TestEntryRSIGateNeedsHistory(t *testing.T) {
	if sig := newS(m5()).Decide(fixture([]bar{redB, greenB})); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
}
