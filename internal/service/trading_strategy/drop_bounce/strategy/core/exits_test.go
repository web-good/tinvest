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
	return append(bars, bar{last.t.Add(stepBar), o, h, l, c})
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
	sig := NewWithParams("TEST", stepBar, stopOnly()).Decide(md)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-98) > eps {
		t.Fatalf("got %q stop %v, want SL at 98 from the entry ATR", sig.Reason, sig.StopLoss)
	}
}

// A position without a frozen ATR keeps a stop from today's ATR rather than running naked.
func TestStopFallsBackToCurrentATR(t *testing.T) {
	bars := flatWithLast(100, 100, 98, 99)
	md := fixture(bars)
	md.Position = position(100, bars[20].t, 0)
	sig := NewWithParams("TEST", stepBar, stopOnly()).Decide(md)
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
// each, as 15m bars from Monday 00:00: the hourly RSI(10) crosses 70 on the rise.
func slideAndRise() []bar {
	hourly := append(flatN(30, 100), steps(100, 10, -0.5)...)
	hourly = append(hourly, steps(hourly[len(hourly)-1], 15, 0.3)...)
	return path("2026-09-14 00:00", quarterHours(hourly))
}

// exitHour returns the first hour ≥ 40 whose hourly RSI(10) crosses 70 in slideAndRise and the
// 15m bar that completes it, guarded against a fixture that no longer crosses.
func exitHour(t *testing.T) (hour, doneBar int) {
	t.Helper()
	hourly := append(flatN(30, 100), steps(100, 10, -0.5)...)
	hourly = append(hourly, steps(hourly[len(hourly)-1], 15, 0.3)...)
	rsi := indicators.RSISeries(hourly, 10)
	for k := 40; k < len(hourly); k++ {
		if crossUp(rsi[k-1], rsi[k], 70) {
			return k, perHour*k + perHour - 1
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
	if sig := held(stopOnlyOff(), bars[:d+1], 200, bars[d].t.Add(stepBar)); sig.Kind != model.SignalNone {
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
