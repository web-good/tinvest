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

// position is a held long bought at entry on the bar starting at entryTime with the frozen ATR.
func position(entry float64, entryTime time.Time, atr float64) *strategy.Position {
	return &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryTime: entryTime, EntryATR: atr}
}

// stopOnly is the default params with the RSI exit off.
func stopOnly() Params {
	p := DefaultParams()
	p.RSIExit = 0
	return p
}

// stopOnlyOff is the default params without the stop, so only RSI/MACD exits can fire.
func stopOnlyOff() Params {
	p := DefaultParams()
	p.StopDailyATR = 0
	return p
}

// flatWithLast is 30 flat bars at 100 and a last bar with the given open/high/low/close.
func flatWithLast(o, h, l, c float64) []bar {
	bars := path("2026-09-14 08:00", flatN(30, 100))
	last := bars[len(bars)-1]
	return append(bars, bar{last.t.Add(stepH1), o, h, l, c})
}

// The default stop is 1.0 daily ATR (fixtureATR 2) below the fill: 98 for a fill at 100.
func TestStopFromFillPrice(t *testing.T) {
	bars := flatWithLast(100, 100, 98.0, 99)
	sig := held(stopOnly(), bars, 100, bars[20].t)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || math.Abs(sig.StopLoss-98) > eps {
		t.Fatalf("got %v %q stop %v, want SL at 98", sig.Kind, sig.Reason, sig.StopLoss)
	}
	bars = flatWithLast(100, 100, 98.01, 99)
	if sig := held(stopOnly(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("low 98.01: kind = %v, want None", sig.Kind)
	}
	p := stopOnly()
	p.StopDailyATR = 0
	bars = flatWithLast(100, 100, 50, 60)
	if sig := held(p, bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("StopDailyATR 0: kind = %v, want None", sig.Kind)
	}
}

// Dailies that closed after the entry must not move the stop.
func TestStopUsesATRFrozenAtEntry(t *testing.T) {
	bars := flatWithLast(100, 100, 97.5, 99)
	md := fixture(bars)
	withDailies(&md, dailies(bars[0].t, 20, 4)) // today's ATR is 4: a live stop would sit at 96
	md.Position = position(100, bars[20].t, fixtureATR)
	sig := NewWithParams("TEST", stepH1, stopOnly()).Decide(md)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-98) > eps {
		t.Fatalf("got %q stop %v, want SL at 98 from the entry ATR", sig.Reason, sig.StopLoss)
	}
}

// A position without a frozen ATR keeps a stop from today's ATR rather than running naked.
func TestStopFallsBackToCurrentATR(t *testing.T) {
	bars := flatWithLast(100, 100, 98, 99)
	md := fixture(bars)
	md.Position = position(100, bars[20].t, 0)
	sig := NewWithParams("TEST", stepH1, stopOnly()).Decide(md)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-98) > eps {
		t.Fatalf("got %q stop %v, want SL at 98 from the current ATR", sig.Reason, sig.StopLoss)
	}
}

func TestCrossUp(t *testing.T) {
	if !crossUp(69, 70, 70) || !crossUp(10, 90, 70) {
		t.Fatal("a move from below to at/above the level must cross")
	}
	if crossUp(70, 75, 70) || crossUp(60, 69.9, 70) || crossUp(80, 60, 70) {
		t.Fatal("no cross from at/above, below-to-below or downward")
	}
}

// slideAndRise is 30 flat bars at 100, 10 bars sliding 0.5% each (RSI(6) near 0), then 15 bars
// rising 0.3% each, which carry RSI(6) up through 70.
func slideAndRise() []bar {
	closes := append(flatN(30, 100), steps(100, 10, -0.5)...)
	closes = append(closes, steps(closes[len(closes)-1], 15, 0.3)...)
	return path("2026-09-14 08:00", closes)
}

// firstCross returns the first bar index ≥ from whose RSI(period) crosses level upward, or −1.
func firstCross(bars []bar, period int, level float64, from int) int {
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.c
	}
	rsi := indicators.RSISeries(closes, period)
	for k := max(from, period+1); k < len(bars); k++ {
		if crossUp(rsi[k-1], rsi[k], level) {
			return k
		}
	}
	return -1
}

// closesOf is the bar closes.
func closesOf(bars []bar) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = b.c
	}
	return out
}

// histOf is the MACD(12,26,9) histogram over the bars with the same function the strategy uses.
func histOf(bars []bar) []float64 {
	m, s := indicators.MACD(closesOf(bars), 12, 26, 9)
	if m == nil {
		return nil
	}
	h := make([]float64, len(m))
	for i := range m {
		h[i] = m[i] - s[i]
	}
	return h
}

// redCross is the first RSI(6) cross of 70 from bar 40 on, guarded: the histogram there must be
// red, otherwise the fixture no longer proves an immediate RSI exit.
func redCross(t *testing.T, bars []bar) int {
	t.Helper()
	k := firstCross(bars, 6, 70, 40)
	if k < 0 {
		t.Fatal("fixture: RSI(6) never crosses 70 on the rise")
	}
	if h := histOf(bars); h == nil || h[k] >= 0 {
		t.Fatalf("fixture: MACD histogram at the cross bar must be red, got %v", h)
	}
	return k
}

// The RSI exit sells at a red-histogram cross whatever the price: here far below the entry.
func TestRSIExitSellsAtTheCross(t *testing.T) {
	bars := slideAndRise()
	k := redCross(t, bars)
	p := DefaultParams()
	p.StopDailyATR = 0
	if sig := held(p, bars[:k], 200, bars[30].t); sig.Kind != model.SignalNone {
		t.Fatalf("bar before the cross: kind = %v %q, want None", sig.Kind, sig.Reason)
	}
	if sig := held(p, bars[:k+1], 200, bars[30].t); sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("cross bar: got %v %q, want Sell RSI below the entry", sig.Kind, sig.Reason)
	}
	p.RSIExit = 0
	if sig := held(p, bars[:k+1], 200, bars[30].t); sig.Kind != model.SignalNone {
		t.Fatalf("RSIExit 0: kind = %v, want None", sig.Kind)
	}
}

// On a bar that both crosses RSI 70 and touches the stop, the stop wins.
func TestStopBeatsRSIExit(t *testing.T) {
	bars := slideAndRise()
	k := redCross(t, bars)
	bars = bars[:k+1]
	entry := bars[k].c
	bars[k].l = entry - 2.5 // stop = entry − 1.0 × fixtureATR
	sig := held(DefaultParams(), bars, entry, bars[30].t)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-(entry-2)) > eps {
		t.Fatalf("got %q stop %v, want SL at %v", sig.Reason, sig.StopLoss, entry-2)
	}
}

// Positions carry overnight: nothing sells the last evening bar on its own.
func TestNoEODExit(t *testing.T) {
	bars := path("2026-09-14 16:15", flatN(31, 100)) // last bar 23:45
	if sig := held(DefaultParams(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("23:45 flat: kind = %v %q, want None", sig.Kind, sig.Reason)
	}
}

// An entry on the cross bar itself, above that bar's close, still sells: the RSI exit has no
// deferral and no in-profit condition.
func TestRSIExitOnFillBarBelowEntry(t *testing.T) {
	bars := slideAndRise()
	k := redCross(t, bars)
	p := DefaultParams()
	p.StopDailyATR = 0
	if sig := held(p, bars[:k+1], 200, bars[k].t); sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("fill bar crossing below the entry: got %v %q, want Sell RSI", sig.Kind, sig.Reason)
	}
}

// When the stop cannot be placed (no ATR anywhere, or the level would be at or below zero) there
// is no SL sell and no panic, and the RSI exit still works.
func TestStopUnplaceableLeavesRSIExit(t *testing.T) {
	bars := slideAndRise()
	k := redCross(t, bars)
	s := NewWithParams("TEST", stepH1, DefaultParams())

	// Frozen ATR and today's ATR both unknown.
	md := fixture(bars[:k])
	md.DailyHighs = nil
	md.Position = position(200, bars[30].t, 0)
	md.Lows[len(md.Lows)-1] = 1
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("ATR unknown, deep low: got %v %q, want None", sig.Kind, sig.Reason)
	}
	md = fixture(bars[:k+1])
	md.DailyHighs = nil
	md.Position = position(200, bars[30].t, 0)
	if sig := s.Decide(md); sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("ATR unknown, cross bar: got %v %q, want Sell RSI", sig.Kind, sig.Reason)
	}

	// Stop level entry − 1.0×ATR ≤ 0.
	md = fixture(bars[:k])
	md.Position = position(1.5, bars[30].t, fixtureATR)
	md.Lows[len(md.Lows)-1] = 0.1
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("stop below zero, deep low: got %v %q, want None", sig.Kind, sig.Reason)
	}
	md = fixture(bars[:k+1])
	md.Position = position(1.5, bars[30].t, fixtureATR)
	if sig := s.Decide(md); sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("stop below zero, cross bar: got %v %q, want Sell RSI", sig.Kind, sig.Reason)
	}
}

// wobble is n closes alternating ±0.1% around 100: a quiet lead whose RSI hovers near 50 and
// whose MACD histogram is ~0, so the MACD seeding bias is long gone by the interesting bars.
func wobble(n int) []float64 {
	out := make([]float64, n)
	p := 100.0
	for i := range out {
		if i%2 == 0 {
			p *= 1.001
		} else {
			p /= 1.001
		}
		out[i] = p
	}
	return out
}

// Indexes of the armedPath fixture (checked by guardArmed).
const (
	armedCross = 100 // RSI(6) crosses 70 here with a green histogram
	armedFlip  = 111 // the histogram turns red here for the first time since the cross
)

// armedPath is 100 wobble bars, 5 bars of +1% (RSI(6) jumps through 70 on the first, histogram
// green), then 12 bars of −0.4% which turn the histogram red at armedFlip.
func armedPath() []bar {
	c := wobble(armedCross)
	c = append(c, steps(c[len(c)-1], 5, 1)...)
	c = append(c, steps(c[len(c)-1], 12, -0.4)...)
	return path("2026-09-01 08:00", c)
}

// guardArmed fails loudly unless bars carry exactly one RSI cross of 70 in [7, armedFlip] at
// armedCross with a green histogram, and the histogram is green on armedFlip−1 and red on armedFlip.
func guardArmed(t *testing.T, bars []bar) {
	t.Helper()
	h := histOf(bars)
	if h == nil {
		t.Fatal("fixture: MACD histogram unavailable")
	}
	if k := firstCross(bars, 6, 70, 7); k != armedCross {
		t.Fatalf("fixture: first RSI cross of 70 at %d, want %d", k, armedCross)
	}
	if k := firstCross(bars, 6, 70, armedCross+1); k >= 0 && k <= armedFlip {
		t.Fatalf("fixture: second RSI cross of 70 at %d before the flip", k)
	}
	if h[armedCross] < 0 {
		t.Fatalf("fixture: histogram at the cross is %v, want green", h[armedCross])
	}
	for j := armedCross; j < armedFlip; j++ {
		if h[j] < 0 {
			t.Fatalf("fixture: histogram red at %d before the flip", j)
		}
	}
	if h[armedFlip] >= 0 {
		t.Fatalf("fixture: histogram at the flip is %v, want red", h[armedFlip])
	}
}

// An RSI cross on a green histogram does not sell: it only arms the exit.
func TestRSICrossOnGreenHistogramHolds(t *testing.T) {
	bars := armedPath()
	guardArmed(t, bars)
	sig := held(stopOnlyOff(), bars[:armedCross+1], 100, bars[95].t)
	if sig.Kind != model.SignalNone {
		t.Fatalf("cross on green: got %v %q, want hold", sig.Kind, sig.Reason)
	}
}

// Armed by a green-histogram cross, the first green→red flip sells with reason MACD.
func TestMACDExitAfterArmedFlip(t *testing.T) {
	bars := armedPath()
	guardArmed(t, bars)
	p := stopOnlyOff()
	if sig := held(p, bars[:armedFlip], 100, bars[95].t); sig.Kind != model.SignalNone {
		t.Fatalf("bar before the flip: got %v %q, want hold", sig.Kind, sig.Reason)
	}
	sig := held(p, bars[:armedFlip+1], 100, bars[95].t)
	if sig.Kind != model.SignalSell || sig.Reason != "MACD" {
		t.Fatalf("flip bar: got %v %q, want Sell MACD", sig.Kind, sig.Reason)
	}
	if !strings.HasPrefix(sig.ExitReason, "MACD: гистограмма ") || !strings.Contains(sig.ExitReason, "перекрасилась в красную после RSI ≥ 70") {
		t.Fatalf("exit reason %q", sig.ExitReason)
	}
}

// A cross that happened before the entry fill bar does not arm the exit.
func TestCrossBeforeEntryDoesNotArm(t *testing.T) {
	bars := armedPath()
	guardArmed(t, bars)
	sig := held(stopOnlyOff(), bars[:armedFlip+1], 100, bars[armedCross+1].t)
	if sig.Kind != model.SignalNone {
		t.Fatalf("entry after the cross: got %v %q, want hold", sig.Kind, sig.Reason)
	}
}

// The cross on the fill bar itself counts (its close is a decision bar).
func TestCrossOnFillBarArms(t *testing.T) {
	bars := armedPath()
	guardArmed(t, bars)
	sig := held(stopOnlyOff(), bars[:armedFlip+1], 100, bars[armedCross].t)
	if sig.Kind != model.SignalSell || sig.Reason != "MACD" {
		t.Fatalf("entry on the cross bar: got %v %q, want Sell MACD", sig.Kind, sig.Reason)
	}
}

// An entry older than the window (or without a time) scans from the first evaluable bar.
func TestEntryOlderThanWindowScansFromStart(t *testing.T) {
	bars := armedPath()
	guardArmed(t, bars)
	for name, et := range map[string]time.Time{"before window": bars[0].t.Add(-48 * time.Hour), "zero": {}} {
		sig := held(stopOnlyOff(), bars[:armedFlip+1], 100, et)
		if sig.Kind != model.SignalSell || sig.Reason != "MACD" {
			t.Fatalf("%s: got %v %q, want Sell MACD", name, sig.Kind, sig.Reason)
		}
	}
}

// zigPath drifts up on a ±0.3% zigzag whose RSI(6) peaks at ~66 (never reaching 70) with a green
// histogram, then falls 0.4% a bar: the histogram flips red at zigFlip without any RSI cross.
func zigPath() []bar {
	c := wobble(100)
	p := c[len(c)-1]
	for i := 0; i < 16; i++ {
		if i%2 == 0 {
			p *= 1.003
		} else {
			p *= 0.9975
		}
		c = append(c, p)
	}
	c = append(c, steps(p, 12, -0.4)...)
	return path("2026-09-01 08:00", c)
}

const zigFlip = 116

func TestRedFlipWithoutArmingHolds(t *testing.T) {
	bars := zigPath()
	h := histOf(bars)
	if h == nil || h[zigFlip-1] < 0 || h[zigFlip] >= 0 {
		t.Fatalf("fixture: histogram must flip green→red at %d: %v", zigFlip, h)
	}
	if k := firstCross(bars, 6, 70, 7); k >= 0 {
		t.Fatalf("fixture: RSI(6) crosses 70 at %d, want no cross", k)
	}
	for i := zigFlip; i < len(bars); i++ {
		if sig := held(stopOnlyOff(), bars[:i+1], 100, bars[95].t); sig.Kind != model.SignalNone {
			t.Fatalf("bar %d: got %v %q, want hold (flip without arming)", i, sig.Kind, sig.Reason)
		}
	}
}

// RSIExit = 0 switches off both the RSI and the MACD exit.
func TestRSIExitZeroDisablesMACDExit(t *testing.T) {
	bars := armedPath()
	guardArmed(t, bars)
	p := stopOnly()
	if sig := held(p, bars[:armedFlip+1], 100, bars[95].t); sig.Kind != model.SignalNone {
		t.Fatalf("RSIExit 0, armed flip: got %v %q, want hold", sig.Kind, sig.Reason)
	}
}

// Without a histogram (invalid MACD periods) it counts as red: an RSI cross sells at once, as RSI.
func TestInvalidMACDTreatsHistogramAsRed(t *testing.T) {
	bars := armedPath()
	guardArmed(t, bars)
	p := stopOnlyOff()
	p.MACDFast, p.MACDSlow = 26, 26
	if m, _ := indicators.MACD(closesOf(bars), p.MACDFast, p.MACDSlow, p.MACDSignal); m != nil {
		t.Fatal("fixture: MACD must be unavailable")
	}
	sig := held(p, bars[:armedCross+1], 100, bars[95].t)
	if sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("no histogram, cross: got %v %q, want Sell RSI", sig.Kind, sig.Reason)
	}
	if sig := held(p, bars[:armedFlip+1], 100, bars[95].t); sig.Reason == "MACD" {
		t.Fatal("MACD must never arm without a histogram")
	}
}

// The stop wins over the MACD exit on the flip bar.
func TestStopBeatsMACDExit(t *testing.T) {
	bars := armedPath()
	guardArmed(t, bars)
	bars = bars[:armedFlip+1]
	entry := bars[armedFlip].c
	bars[armedFlip].l = entry - 2.5
	sig := held(DefaultParams(), bars, entry, bars[95].t)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-(entry-2)) > eps {
		t.Fatalf("got %q stop %v, want SL at %v", sig.Reason, sig.StopLoss, entry-2)
	}
}

// RSI exit on a red histogram carries the histogram in its reason.
func TestRSIExitReasonMentionsHistogram(t *testing.T) {
	bars := slideAndRise()
	k := redCross(t, bars)
	p := DefaultParams()
	p.StopDailyATR = 0
	sig := held(p, bars[:k+1], 200, bars[30].t)
	if sig.Reason != "RSI" || !strings.Contains(sig.ExitReason, "гистограмма MACD ") {
		t.Fatalf("got %q / %q", sig.Reason, sig.ExitReason)
	}
}
