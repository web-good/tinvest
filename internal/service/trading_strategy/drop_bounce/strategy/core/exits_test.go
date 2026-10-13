package core

import (
	"math"
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

// The RSI exit sells at the cross whatever the price: here far below the entry.
func TestRSIExitSellsAtTheCross(t *testing.T) {
	bars := slideAndRise()
	k := firstCross(bars, 6, 70, 40)
	if k < 0 {
		t.Fatal("fixture: RSI(6) never crosses 70 on the rise")
	}
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
	k := firstCross(bars, 6, 70, 40)
	if k < 0 {
		t.Fatal("fixture: RSI(6) never crosses 70 on the rise")
	}
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
	k := firstCross(bars, 6, 70, 40)
	if k < 0 {
		t.Fatal("fixture: RSI(6) never crosses 70 on the rise")
	}
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
	k := firstCross(bars, 6, 70, 40)
	if k < 0 {
		t.Fatal("fixture: RSI(6) never crosses 70 on the rise")
	}
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
