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

// stopOnly is the default params with the RSI and time exits off.
func stopOnly() Params {
	p := DefaultParams()
	p.RSIExit, p.StaleBars = 0, 0
	return p
}

// stopOnlyOff is the default params without the stop, so only the RSI exit can fire.
func stopOnlyOff() Params {
	p := DefaultParams()
	p.StopDailyATR = 0
	return p
}

// flatWithLast is 30 flat bars at 100 and a last bar with the given open/high/low/close.
func flatWithLast(o, h, l, c float64) []bar {
	bars := path("2026-09-14 08:00", flatN(30, 100))
	last := bars[len(bars)-1]
	return append(bars, bar{last.t.Add(stepM30), o, h, l, c})
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
	sig := NewWithParams("TEST", stepM30, stopOnly()).Decide(md)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-98) > eps {
		t.Fatalf("got %q stop %v, want SL at 98 from the entry ATR", sig.Reason, sig.StopLoss)
	}
}

// A position without a frozen ATR keeps a stop from today's ATR rather than running naked.
func TestStopFallsBackToCurrentATR(t *testing.T) {
	bars := flatWithLast(100, 100, 98, 99)
	md := fixture(bars)
	md.Position = position(100, bars[20].t, 0)
	sig := NewWithParams("TEST", stepM30, stopOnly()).Decide(md)
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

// slideAndRise is 30 flat hours at 100, 10 hours sliding 0.5% each, then 15 hours rising 0.3%
// each, as 30m bars from Monday 00:00: the hourly RSI(10) crosses 70 on the rise.
func slideAndRise() []bar {
	hourly := append(flatN(30, 100), steps(100, 10, -0.5)...)
	hourly = append(hourly, steps(hourly[len(hourly)-1], 15, 0.3)...)
	return path("2026-09-14 00:00", halfHours(hourly))
}

// exitHour returns the first hour ≥ 40 whose hourly RSI(10) crosses 70 in slideAndRise and the
// 30m bar that completes it, guarded against a fixture that no longer crosses.
func exitHour(t *testing.T) (hour, doneBar int) {
	t.Helper()
	hourly := append(flatN(30, 100), steps(100, 10, -0.5)...)
	hourly = append(hourly, steps(hourly[len(hourly)-1], 15, 0.3)...)
	rsi := indicators.RSISeries(hourly, 10)
	for k := 40; k < len(hourly); k++ {
		if crossUp(rsi[k-1], rsi[k], 70) {
			return k, 2*k + 1
		}
	}
	t.Fatal("fixture: hourly RSI(10) never crosses 70 on the rise")
	return 0, 0
}

// The RSI exit sells on the bar that completes the crossing hour whatever the price, not on the
// hour's first half.
func TestRSIExitSellsWhenTheHourCompletes(t *testing.T) {
	bars := slideAndRise()
	_, d := exitHour(t)
	p := stopOnlyOff()
	if sig := held(p, bars[:d], 200, bars[60].t); sig.Kind != model.SignalNone {
		t.Fatalf("first half of the crossing hour: got %v %q, want None", sig.Kind, sig.Reason)
	}
	sig := held(p, bars[:d+1], 200, bars[60].t)
	if sig.Kind != model.SignalSell || sig.Reason != "RSI" || !strings.Contains(sig.ExitReason, "RSI(10)") {
		t.Fatalf("hour completed: got %v %q %q, want Sell RSI(10)", sig.Kind, sig.Reason, sig.ExitReason)
	}
	if sig := held(p, bars[:d+2], 200, bars[60].t); sig.Kind != model.SignalNone {
		t.Fatalf("next bar: got %v %q, want None (the cross is on the previous hour)", sig.Kind, sig.Reason)
	}
	for name, mut := range map[string]func(*Params){
		"RSIExit":       func(p *Params) { p.RSIExit = 0 },
		"RSIExitPeriod": func(p *Params) { p.RSIExitPeriod = 0 },
	} {
		q := p
		mut(&q)
		if sig := held(q, bars[:d+1], 200, bars[60].t); sig.Kind != model.SignalNone {
			t.Fatalf("%s 0: kind = %v, want None", name, sig.Kind)
		}
	}
}

// A crossing hour that ended before the entry does not sell; one the position held part of does.
func TestRSIExitNeedsTheHourAfterEntry(t *testing.T) {
	bars := slideAndRise()
	_, d := exitHour(t)
	if sig := held(stopOnlyOff(), bars[:d+1], 200, bars[d].t.Add(stepM30)); sig.Kind != model.SignalNone {
		t.Fatalf("entry after the hour: got %v %q, want None", sig.Kind, sig.Reason)
	}
	if sig := held(stopOnlyOff(), bars[:d+1], 200, bars[d].t); sig.Reason != "RSI" {
		t.Fatalf("entry on the hour's last bar: reason %q, want RSI", sig.Reason)
	}
}

// On the bar that both completes the crossing hour and touches the stop, the stop wins.
func TestStopBeatsRSIExit(t *testing.T) {
	bars := slideAndRise()
	_, d := exitHour(t)
	bars = bars[:d+1]
	entry := bars[d].c
	bars[d].l = entry - 2.5 // stop = entry − 1.0 × fixtureATR
	sig := held(DefaultParams(), bars, entry, bars[60].t)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-(entry-2)) > eps {
		t.Fatalf("got %q stop %v, want SL at %v", sig.Reason, sig.StopLoss, entry-2)
	}
}

// Positions carry overnight: nothing sells the last evening bar on its own.
func TestNoEODExit(t *testing.T) {
	bars := path("2026-09-14 08:30", flatN(31, 100)) // last bar 23:30
	if sig := held(DefaultParams(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("23:30 flat: kind = %v %q, want None", sig.Kind, sig.Reason)
	}
}

// staleOn is stopOnly with the time exit after 5 bars below a 0.3 ATR gain (0.6 on fixtureATR 2).
func staleOn() Params {
	p := stopOnly()
	p.StaleBars, p.StaleATR = 5, 0.3
	return p
}

// Flat bars at 100 never reach 100.6: the fifth held bar (the fill bar counts) sells at its close,
// the fourth holds.
func TestStaleExitAfterStaleBars(t *testing.T) {
	bars := path("2026-09-14 08:00", flatN(30, 100))
	n := len(bars)
	sig := held(staleOn(), bars, 100, bars[n-5].t)
	if sig.Kind != model.SignalSell || sig.Reason != "STALE" || !strings.Contains(sig.ExitReason, "5 баров") {
		t.Fatalf("got %v %q %q, want STALE after 5 bars", sig.Kind, sig.Reason, sig.ExitReason)
	}
	if sig := held(staleOn(), bars, 100, bars[n-4].t); sig.Kind != model.SignalNone {
		t.Fatalf("4 bars held: kind = %v, want None", sig.Kind)
	}
}

// A high at entry + StaleATR·ATR on any bar since the fill keeps the position; one before the fill
// does not.
func TestStaleExitKeptByFavourableHigh(t *testing.T) {
	bars := path("2026-09-14 08:00", flatN(30, 100))
	n := len(bars)
	bars[n-3].h = 100.6
	if sig := held(staleOn(), bars, 100, bars[n-5].t); sig.Kind != model.SignalNone {
		t.Fatalf("high 100.6 after the fill: kind = %v, want None", sig.Kind)
	}
	bars[n-3].h = 100
	bars[n-6].h = 105
	if sig := held(staleOn(), bars, 100, bars[n-5].t); sig.Reason != "STALE" {
		t.Fatalf("high before the fill: reason = %q, want STALE", sig.Reason)
	}
}

// The time exit is off at StaleBars 0, and silent without a fill bar in the window or entry time.
func TestStaleExitOffOrUnknownFill(t *testing.T) {
	bars := path("2026-09-14 08:00", flatN(30, 100))
	p := staleOn()
	p.StaleBars = 0
	if sig := held(p, bars, 100, bars[0].t); sig.Kind != model.SignalNone {
		t.Fatalf("StaleBars 0: kind = %v, want None", sig.Kind)
	}
	if sig := held(staleOn(), bars, 100, bars[0].t.Add(-stepM30)); sig.Kind != model.SignalNone {
		t.Fatalf("fill before the window: kind = %v, want None", sig.Kind)
	}
	if sig := held(staleOn(), bars, 100, time.Time{}); sig.Kind != model.SignalNone {
		t.Fatalf("zero entry time: kind = %v, want None", sig.Kind)
	}
}

// The stop has priority over the time exit on the same bar.
func TestStopBeatsStaleExit(t *testing.T) {
	bars := flatWithLast(100, 100, 97, 99)
	if sig := held(staleOn(), bars, 100, bars[20].t); sig.Reason != "SL" {
		t.Fatalf("reason = %q, want SL", sig.Reason)
	}
}

// breakevenOn is the stop-only params with the breakeven armed at +0.5 daily ATR (101 for a fill at
// 100 with fixtureATR 2).
func breakevenOn() Params {
	p := stopOnly()
	p.BreakevenATR = 0.5
	return p
}

// A completed bar since the fill with its high at entry + BreakevenATR·ATR moves the stop to the
// entry price: a later low at the entry sells BE, a low above it holds.
func TestBreakevenAfterFavourableHigh(t *testing.T) {
	bars := flatWithLast(100, 100.2, 99.9, 100)
	bars[25].h = 101.2
	sig := held(breakevenOn(), bars, 100, bars[20].t)
	if sig.Kind != model.SignalSell || sig.Reason != "BE" || math.Abs(sig.StopLoss-100) > eps {
		t.Fatalf("got %v %q stop %v, want BE at 100", sig.Kind, sig.Reason, sig.StopLoss)
	}
	bars[len(bars)-1].l = 100.1
	if sig := held(breakevenOn(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("low above the entry: kind = %v, want None", sig.Kind)
	}
	bars[len(bars)-1].l = 99.9
	bars[25].h = 100.9
	if sig := held(breakevenOn(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("high 100.9 < 101: kind = %v, want None", sig.Kind)
	}
}

// The current bar's own high does not arm its stop, a high before the fill does not count, and
// BreakevenATR 0 is off.
func TestBreakevenNeedsACompletedBarSinceTheFill(t *testing.T) {
	bars := flatWithLast(100, 102, 99.9, 100)
	if sig := held(breakevenOn(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("own high: kind = %v, want None", sig.Kind)
	}
	bars = flatWithLast(100, 100, 99.9, 100)
	bars[18].h = 105
	if sig := held(breakevenOn(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("high before the fill: kind = %v, want None", sig.Kind)
	}
	bars[25].h = 101.2
	p := breakevenOn()
	p.BreakevenATR = 0
	if sig := held(p, bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("BreakevenATR 0: kind = %v, want None", sig.Kind)
	}
}

// With TrailATR the stop trails the best high when that is above the entry, else stays at it.
func TestTrailAboveEntry(t *testing.T) {
	bars := flatWithLast(101, 101, 100.6, 100.8)
	bars[25].h = 101.2
	p := breakevenOn()
	p.TrailATR = 0.25 // 101.2 − 0.5 = 100.7
	sig := held(p, bars, 100, bars[20].t)
	if sig.Reason != "TRAIL" || math.Abs(sig.StopLoss-100.7) > eps {
		t.Fatalf("got %q stop %v, want TRAIL at 100.7", sig.Reason, sig.StopLoss)
	}
	bars[len(bars)-1].l = 100.8
	if sig := held(p, bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("low above the trail: kind = %v, want None", sig.Kind)
	}
	bars[len(bars)-1].l = 99.95
	p.TrailATR = 1 // 101.2 − 2 = 99.2 < entry: the stop stays at 100
	if sig := held(p, bars, 100, bars[20].t); sig.Reason != "BE" || math.Abs(sig.StopLoss-100) > eps {
		t.Fatalf("trail below the entry: got %q stop %v, want BE at 100", sig.Reason, sig.StopLoss)
	}
}
