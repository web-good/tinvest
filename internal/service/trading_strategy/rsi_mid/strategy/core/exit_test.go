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

// afterEntry appends `bars` closes to an entry history, each moving by `step` (e.g. -0.01 is
// -1% per bar), and returns the extended closes together with the entry bar's index.
func afterEntry(entry []float64, step float64, bars int) (closes []float64, entryIdx int) {
	closes = append([]float64(nil), entry...)
	p := closes[len(closes)-1]
	for i := 0; i < bars; i++ {
		p *= 1 + step
		closes = append(closes, p)
	}
	return closes, len(entry) - 1
}

// held attaches an open long to md: bought at the close of entryIdx, opened at that bar's time,
// with entryATR frozen on the position (0 = no stop).
func held(md strategy.MarketData, entryIdx int, entryATR float64) strategy.MarketData {
	md.Position = &strategy.Position{
		PurchasePrice: md.Closes[entryIdx],
		Quantity:      1,
		EntryATR:      entryATR,
		EntryTime:     md.Times[entryIdx],
	}
	return md
}

// requireBelowMid fails the test unless RSI(14) reads below 50 on every bar in [from, to].
func requireBelowMid(t *testing.T, closes []float64, from, to int) {
	t.Helper()
	rsi := indicators.RSISeries(closes, 14)
	for b := from; b <= to; b++ {
		if rsi[b] >= midLine {
			t.Fatalf("fixture: RSI[%d] = %v, want < 50", b, rsi[b])
		}
	}
}

func midParams(bars int) *Strategy {
	p := DefaultParams()
	p.BelowMidBars = bars
	return NewWithParams("T", p)
}

func TestMIDExitsOnTheNthBarBelow50(t *testing.T) {
	closes, e := afterEntry(upCross(), -0.01, 3)
	requireBelowMid(t, closes, e+1, e+3)
	md := held(fixture(closes, mondayNoon), e, 0) // no stop: isolate MID
	sig := midParams(3).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "MID" {
		t.Fatalf("got %v %q, want Sell MID", sig.Kind, sig.Reason)
	}
	if !strings.Contains(sig.ExitReason, "3 бар") {
		t.Fatalf("ExitReason = %q, want the bar count", sig.ExitReason)
	}
}

func TestMIDHoldsOneBarShort(t *testing.T) {
	closes, e := afterEntry(upCross(), -0.01, 2)
	requireBelowMid(t, closes, e+1, e+2)
	md := held(fixture(closes, mondayNoon), e, 0)
	if sig := midParams(3).Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("sold after 2 of 3 bars: %q", sig.Reason)
	}
}

func TestMIDDisabledWithZeroBars(t *testing.T) {
	closes, e := afterEntry(upCross(), -0.01, 6)
	requireBelowMid(t, closes, e+1, e+6)
	md := held(fixture(closes, mondayNoon), e, 0)
	if sig := midParams(0).Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("sold with BelowMidBars 0: %q", sig.Reason)
	}
}

// A bar at or above 50 resets the run; a later dip counts from zero again.
func TestMIDRunResetsOnABarAbove50(t *testing.T) {
	closes, e := afterEntry(upCross(), -0.01, 2) // two bars below 50
	p := closes[len(closes)-1]
	rsi := indicators.RSISeries
	for guard := 0; ; guard++ { // climb until RSI is back at or above 50
		if guard > 50 {
			t.Fatal("fixture: RSI never recovered to 50")
		}
		p *= 1.01
		closes = append(closes, p)
		if r := rsi(closes, 14); r[len(r)-1] >= midLine {
			break
		}
	}
	reset := len(closes) - 1
	for i := 0; i < 2; i++ { // two fresh bars below 50: run is 2 < 3
		p *= 0.99
		closes = append(closes, p)
	}
	requireBelowMid(t, closes, reset+1, reset+2)
	if sig := midParams(3).Decide(held(fixture(closes, mondayNoon), e, 0)); sig.Kind == model.SignalSell {
		t.Fatalf("sold on a run of 2 after the reset: %q", sig.Reason)
	}
	p *= 0.99
	closes = append(closes, p) // third bar of the fresh run
	requireBelowMid(t, closes, reset+1, reset+3)
	if sig := midParams(3).Decide(held(fixture(closes, mondayNoon), e, 0)); sig.Reason != "MID" {
		t.Fatalf("got %q on the 3rd bar of the fresh run, want MID", sig.Reason)
	}
}

// Bars at or before Position.EntryTime never count, even when RSI reads below 50 on them.
func TestMIDIgnoresBarsUpToTheEntryBar(t *testing.T) {
	closes, e := afterEntry(upCross(), -0.01, 3)
	md := fixture(closes, mondayNoon)
	md = held(md, e, 0)
	md.Position.EntryTime = md.Times[e+1] // the entry bar is e+1: only e+2 and e+3 are after it
	if sig := midParams(3).Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("counted the entry bar: %q", sig.Reason)
	}
	md.Position.EntryTime = md.Times[e+1].Add(-time.Minute) // still bar e: 3 bars after it
	if sig := midParams(3).Decide(md); sig.Reason != "MID" {
		t.Fatalf("got %q, want MID: EntryTime inside bar e anchors on bar e", sig.Reason)
	}
}

// A multi-day position on a short timeframe outlives the candle window: every bar of the window
// is then after the entry, and the run is counted over the whole window instead of going silent.
func TestMIDCountsTheWholeWindowWhenTheEntryBarLeftIt(t *testing.T) {
	closes, e := afterEntry(upCross(), -0.01, 3)
	md := held(fixture(closes, mondayNoon), e, 0)
	md.Position.EntryTime = md.Times[0].Add(-24 * time.Hour)
	if sig := midParams(3).Decide(md); sig.Reason != "MID" {
		t.Fatalf("got %q, want MID with the entry bar outside the window", sig.Reason)
	}
}

func TestMIDSilentWithoutAnAnchor(t *testing.T) {
	closes, e := afterEntry(upCross(), -0.01, 3)
	t.Run("zero EntryTime", func(t *testing.T) {
		md := held(fixture(closes, mondayNoon), e, 0)
		md.Position.EntryTime = time.Time{}
		if sig := midParams(3).Decide(md); sig.Kind == model.SignalSell {
			t.Fatalf("sold without EntryTime: %q", sig.Reason)
		}
	})
	t.Run("misaligned Times", func(t *testing.T) {
		requireBelowMid(t, closes, e+1, e+3)
		md := held(fixture(closes, mondayNoon), e, 0)
		md.Times = md.Times[1:]
		if sig := midParams(3).Decide(md); sig.Kind == model.SignalSell {
			t.Fatalf("sold with misaligned Times: %q", sig.Reason)
		}
	})
}

func TestSLFiresIntrabarAtTheFrozenLevel(t *testing.T) {
	closes := upCross()
	e := len(closes) - 1
	md := held(fixture(closes, mondayNoon), e, dailyWidth)
	level := closes[e] - 0.5*dailyWidth
	md.Lows[e] = level // touch
	sig := NewWithParams("T", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("got %v %q, want Sell SL", sig.Kind, sig.Reason)
	}
	if math.Abs(sig.StopLoss-level) > 1e-9 {
		t.Fatalf("StopLoss = %v, want %v", sig.StopLoss, level)
	}
	if !model.IsStopReason(sig.Reason) {
		t.Fatal("SL must be a stop reason so the engine fills it at min(level, open)")
	}
	md.Lows[e] = level + 0.01 // a hair above: no stop
	if sig := NewWithParams("T", DefaultParams()).Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("sold above the stop: %q", sig.Reason)
	}
}

// The level is built from Position.EntryATR, never from today's daily ATR.
func TestStopUsesTheATRFrozenAtEntry(t *testing.T) {
	closes := upCross()
	e := len(closes) - 1
	md := held(fixtureWithDaily(closes, mondayNoon, 10*dailyWidth), e, dailyWidth) // ATR now 10x wider
	frozen := closes[e] - 0.5*dailyWidth
	md.Lows[e] = frozen
	sig := NewWithParams("T", DefaultParams()).Decide(md)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-frozen) > 1e-9 {
		t.Fatalf("got %q at %v, want SL at the frozen level %v", sig.Reason, sig.StopLoss, frozen)
	}
}

func TestNoStopWithoutEntryATR(t *testing.T) {
	closes := upCross()
	e := len(closes) - 1
	md := held(fixture(closes, mondayNoon), e, 0)
	md.Lows[e] = 0.01
	if sig := NewWithParams("T", DefaultParams()).Decide(md); sig.Reason == "SL" {
		t.Fatal("stopped out without an EntryATR")
	}
}

func TestRSIExitOnUpperCross(t *testing.T) {
	entry := upCross()
	e := len(entry) - 1
	// Extend the entry history with +1% bars until RSI(14) crosses 70.
	closes := append([]float64(nil), entry...)
	p := closes[e]
	for guard := 0; ; guard++ {
		if guard > 100 {
			t.Fatal("fixture: RSI never crossed 70")
		}
		p *= 1.01
		closes = append(closes, p)
		r := indicators.RSISeries(closes, 14)
		if n := len(r); r[n-2] <= 70 && r[n-1] > 70 {
			break
		}
	}
	md := held(fixture(closes, mondayNoon), e, 0)
	sig := NewWithParams("T", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("got %v %q, want Sell RSI", sig.Kind, sig.Reason)
	}
	if model.IsStopReason(sig.Reason) {
		t.Fatal("RSI exit must fill at close, not as a stop")
	}
}

// Entered with RSI already above RSIUpper: no RSI exit while it stays above — only a fresh
// upward cross counts. The position is left to the stop and MID.
func TestRSIExitNeedsAFreshCrossNotAReading(t *testing.T) {
	closes, e := afterEntry(upCross(), 0.01, 6) // keeps rising: RSI well above 60 on the last two bars
	r := indicators.RSISeries(closes, 14)
	n := len(r)
	if r[n-2] <= 60 || r[n-1] <= 60 {
		t.Fatalf("fixture: RSI %v, %v, want both above 60", r[n-2], r[n-1])
	}
	p := DefaultParams()
	p.RSIUpper = 60
	if sig := NewWithParams("T", p).Decide(held(fixture(closes, mondayNoon), e, 0)); sig.Kind == model.SignalSell {
		t.Fatalf("sold on a reading above the band, not a cross: %q", sig.Reason)
	}
}

// Same bar: the stop is touched AND the MID run completes. The stop wins.
func TestSLBeatsMIDOnTheSameBar(t *testing.T) {
	closes, e := afterEntry(upCross(), -0.01, 3)
	md := held(fixture(closes, mondayNoon), e, dailyWidth)
	md.Lows[len(closes)-1] = closes[e] - 0.5*dailyWidth - 1
	if sig := midParams(3).Decide(md); sig.Reason != "SL" {
		t.Fatalf("got %q, want SL to win the tie", sig.Reason)
	}
}

// Same bar: the stop is touched AND RSI crosses up through RSIUpper. The stop wins and fills
// as a stop, not at the close.
func TestSLBeatsRSIOnTheSameBar(t *testing.T) {
	entry := upCross()
	e := len(entry) - 1
	closes := append([]float64(nil), entry...)
	p := closes[e]
	for guard := 0; ; guard++ {
		if guard > 100 {
			t.Fatal("fixture: RSI never crossed 70")
		}
		p *= 1.01
		closes = append(closes, p)
		r := indicators.RSISeries(closes, 14)
		if n := len(r); r[n-2] <= 70 && r[n-1] > 70 {
			break
		}
	}
	md := held(fixture(closes, mondayNoon), e, dailyWidth)
	level := closes[e] - 0.5*dailyWidth
	last := len(closes) - 1
	md.Lows[last] = level - 1
	// Precondition: without the stop this bar is an RSI exit, so the test is not vacuous.
	if sig := NewWithParams("T", DefaultParams()).Decide(held(fixture(closes, mondayNoon), e, 0)); sig.Reason != "RSI" {
		t.Fatalf("fixture: got %q without a stop, want RSI on this bar", sig.Reason)
	}
	sig := NewWithParams("T", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("got %v %q, want Sell SL to win the tie", sig.Kind, sig.Reason)
	}
	if math.Abs(sig.StopLoss-level) > 1e-9 {
		t.Fatalf("StopLoss = %v, want %v", sig.StopLoss, level)
	}
	if !model.IsStopReason(sig.Reason) {
		t.Fatal("SL must be a stop reason so the engine fills it at min(level, open)")
	}
}

func TestExitsFireOnWeekendBars(t *testing.T) {
	closes, e := afterEntry(upCross(), -0.01, 3)
	md := held(fixture(closes, saturdayNoon), e, 0)
	if sig := midParams(3).Decide(md); sig.Reason != "MID" {
		t.Fatalf("got %q on a Saturday, want MID: exits have no calendar gate", sig.Reason)
	}
}

func TestManageNoPanicOnDegenerateInput(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	pos := &strategy.Position{PurchasePrice: 100, EntryATR: 2}
	for name, md := range map[string]strategy.MarketData{
		"empty":      {Position: pos},
		"one bar":    {Closes: []float64{100}, Lows: []float64{99}, Position: pos},
		"short lows": {Closes: []float64{100, 101}, Lows: []float64{99}, Position: pos},
	} {
		if sig := s.Decide(md); sig.Kind == model.SignalSell {
			t.Fatalf("%s: sold on degenerate input", name)
		}
	}
}

// A bar exactly at the midline resets the run: only strictly-below bars count.
func TestBelowMidRunResetsOnExactly50(t *testing.T) {
	p := DefaultParams()
	p.RSIPeriod = 1 // no warm-up cut inside the handmade series
	s := NewWithParams("T", p)
	rsi := []float64{49, 50, 49, 49}
	times := make([]time.Time, len(rsi))
	for i := range times {
		times[i] = mondayNoon.Add(time.Duration(i) * 30 * time.Minute)
	}
	md := strategy.MarketData{
		Closes:   make([]float64, len(rsi)),
		Times:    times,
		Position: &strategy.Position{EntryTime: times[0].Add(-time.Hour)}, // every bar is after the entry
	}
	if got := s.belowMidRun(md, rsi); got != 2 {
		t.Fatalf("run = %d, want 2: the bar at exactly 50 must reset it", got)
	}
}
