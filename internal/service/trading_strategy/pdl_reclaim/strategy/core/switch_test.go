package core

import (
	"math"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// half returns the six 5m bars of the half-hour starting at start whose aggregate is o/h/l/c:
// the first bar carries the whole range, the other five sit flat at the close.
func half(start string, o, h, l, c float64) []bar {
	out := []bar{{start, o, h, l, c}}
	for k := 1; k < 6; k++ {
		out = append(out, bar{at(start).Add(time.Duration(k) * 5 * time.Minute).Format("2006-01-02 15:04"), c, c, c, c})
	}
	return out
}

// morning is Monday 06:00–09:55 flat at 102: eight completed half-hours, enough to warm RSI(6)
// on the half-hour series, and every entry fixture after it comes off a drop (RSI gate passes).
func morning() []bar { return flat("2026-09-14 06:00", "2026-09-14 09:55", 102) }

func TestHalvesAggregateAndClose(t *testing.T) {
	bars := append(half("2026-09-14 09:30", 99, 99.8, 98.7, 99.5), half("2026-09-14 10:00", 99.5, 100, 99.4, 99.9)...)
	hs, done := newS(DefaultParams()).halves(fixture(bars))
	if len(hs.t) != 2 || done[0] != 5 || done[1] != 11 {
		t.Fatalf("halves %d done %v, want 2 closed on bars 5 and 11", len(hs.t), done)
	}
	if hs.o[0] != 99 || hs.h[0] != 99.8 || hs.l[0] != 98.7 || hs.c[0] != 99.5 || !hs.t[1].Equal(at("2026-09-14 10:00")) {
		t.Fatalf("half 0 = %v/%v/%v/%v, half 1 at %v", hs.o[0], hs.h[0], hs.l[0], hs.c[0], hs.t[1])
	}
}

// A half-hour missing its :25/:55 bar closes on the first bar of a later half-hour; the
// half-hour still forming is not returned.
func TestHalvesMissingLastSlot(t *testing.T) {
	bars := half("2026-09-14 10:00", 99.5, 100, 99.4, 99.9)[:5] // 10:00..10:20, no 10:25
	bars = append(bars, bar{"2026-09-14 10:30", 99.9, 99.9, 99.9, 99.9})
	hs, done := newS(DefaultParams()).halves(fixture(bars))
	if len(hs.t) != 1 || done[0] != 5 {
		t.Fatalf("halves %d done %v, want one closed on bar 5", len(hs.t), done)
	}
}

func TestNo5mSignalAfterSwitch(t *testing.T) {
	if sig := decide(DefaultParams(), []bar{pre, barA}); sig.Kind != model.SignalNone {
		t.Fatalf("5m A at 10:00: Kind = %v, want None", sig.Kind)
	}
	b, q := barA, pre
	b.t, q.t = "2026-09-14 09:55", "2026-09-14 09:50"
	if sig := decide(DefaultParams(), []bar{q, b}); sig.Kind != model.SignalBuy {
		t.Fatalf("5m A at 09:55: Kind = %v, want Buy", sig.Kind)
	}
}

// halfA is a green half-hour 10:00–10:30 that dipped below PDL 99 and closed back above it.
func halfA() []bar { return half("2026-09-14 10:00", 99.2, 99.5, 98.8, 99.3) }

func TestEntryA30m(t *testing.T) {
	bars := append(morning(), halfA()...)
	if sig := newS(DefaultParams()).Decide(fixture(bars[:len(bars)-1])); sig.Kind != model.SignalNone {
		t.Fatalf("half-hour still forming at 10:20: Kind = %v, want None", sig.Kind)
	}
	sig := newS(DefaultParams()).Decide(fixture(bars))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.StopLoss-98.3) > eps || !sig.FillBy.Equal(at("2026-09-14 10:30")) || !strings.HasPrefix(sig.EntryReason, "A 30m") {
		t.Fatalf("SL %v FillBy %v reason %q, want 98.3, 10:30, A 30m", sig.StopLoss, sig.FillBy, sig.EntryReason)
	}
}

func TestEntryB30m(t *testing.T) {
	bars := append(flat("2026-09-14 06:00", "2026-09-14 09:25", 102), half("2026-09-14 09:30", 98.9, 98.95, 98.5, 98.6)...)
	bars = append(bars, half("2026-09-14 10:00", 98.55, 98.95, 98.4, 98.92)...)
	sig := newS(DefaultParams()).Decide(fixture(bars))
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "B 30m") {
		t.Fatalf("got %v %q, want Buy with reason B 30m", sig.Kind, sig.EntryReason)
	}
}

// The 09:30 half-hour is before the switch: its close at 09:55 is a 5m decision, never a 30m one.
func TestNo30mSignalBeforeSwitch(t *testing.T) {
	bars := append(flat("2026-09-14 06:00", "2026-09-14 09:25", 102), half("2026-09-14 09:30", 99.2, 99.5, 98.8, 99.3)...)
	if sig := newS(DefaultParams()).Decide(fixture(bars)); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None (09:55 bar itself is flat, no 5m pattern)", sig.Kind)
	}
}

func heldAt(p Params, bars []bar, pos strategy.Position) model.Signal {
	md := fixture(bars)
	md.Position = &pos
	return newS(p).Decide(md)
}

// zigzag returns 5m bars from start rising +0.1 / falling −0.02 in turn (RSI reads high).
func zigzag(start string, c float64, count int) []bar {
	var out []bar
	for k := 0; k < count; k++ {
		if k%2 == 0 {
			c += 0.1
		} else {
			c -= 0.02
		}
		ts := at(start).Add(time.Duration(k) * 5 * time.Minute).Format("2006-01-02 15:04")
		out = append(out, bar{ts, c - 0.05, c + 0.01, c - 0.06, c})
	}
	return out
}

// A position filled on 5m (09:30) keeps 5m exits after the switch.
func TestExit5mPositionAfterSwitch(t *testing.T) {
	p := DefaultParams()
	p.MACDFast = 0
	bars := append([]bar{{"2026-09-14 09:30", 99.35, 99.5, 99.3, 99.4}}, zigzag("2026-09-14 09:35", 99.4, 10)...) // to 10:20
	pos := strategy.Position{PurchasePrice: 99.4, EntryATR: 2, EntryTime: at("2026-09-14 09:30")}
	if sig := heldAt(p, bars, pos); sig.Reason != "RSI" {
		t.Fatalf("Reason = %q, want 5m RSI at 10:20", sig.Reason)
	}
}

// A position filled at 10:30 lives on half-hours: RSI is read only when a half-hour closes.
func TestExit30mPositionOnHalfClose(t *testing.T) {
	p := DefaultParams()
	p.MACDFast = 0
	bars := flat("2026-09-14 06:00", "2026-09-14 10:25", 100)
	c := 100.0
	for k := 0; k < 6; k++ { // half-hours 10:30..13:00 zigzag up
		o := c
		if k%2 == 0 {
			c += 0.3
		} else {
			c -= 0.05
		}
		bars = append(bars, half(at("2026-09-14 10:30").Add(time.Duration(k)*30*time.Minute).Format("2006-01-02 15:04"), o, c+0.01, o-0.01, c)...)
	}
	pos := strategy.Position{PurchasePrice: 100, EntryATR: 100, EntryTime: at("2026-09-14 10:30")}
	if sig := heldAt(p, bars, pos); sig.Reason != "RSI" {
		t.Fatalf("closed half-hour: Reason = %q, want RSI", sig.Reason)
	}
	if sig := heldAt(p, bars[:len(bars)-1], pos); sig.Kind != model.SignalNone {
		t.Fatalf("half-hour still forming: Kind = %v, want None", sig.Kind)
	}
}

func TestHalfIndex(t *testing.T) {
	ts := []time.Time{at("2026-09-14 10:00"), at("2026-09-14 10:30"), at("2026-09-14 11:00")}
	for entry, want := range map[string]int{"2026-09-14 10:30": 1, "2026-09-14 10:45": 1, "2026-09-14 09:55": -1, "2026-09-14 11:35": -1} {
		if got := halfIndex(ts, at(entry)); got != want {
			t.Errorf("%s: %d, want %d", entry, got, want)
		}
	}
	if halfIndex(ts, time.Time{}) != -1 {
		t.Error("zero entry time must be -1")
	}
}

// Review: a half-hour closed hours late by a gap bar must not buy — only the half-hour that ends
// at the current bar or right before its own half-hour may signal.
func TestNoStaleHalfEntry(t *testing.T) {
	bars := append(morning(), flat("2026-09-14 10:00", "2026-09-14 10:25", 102)...)
	bars = append(bars, half("2026-09-14 10:30", 99.2, 99.5, 98.8, 99.3)[:4]...) // 10:30..10:45, then silence
	if sig := newS(DefaultParams()).Decide(fixture(append(bars, bar{"2026-09-14 11:00", 99.3, 99.3, 99.3, 99.3}))); sig.Kind != model.SignalBuy {
		t.Fatalf("closed by the next half's first bar: Kind = %v, want Buy", sig.Kind)
	}
	if sig := newS(DefaultParams()).Decide(fixture(append(bars, bar{"2026-09-14 13:10", 99.3, 99.3, 99.3, 99.3}))); sig.Kind != model.SignalNone {
		t.Fatalf("closed at 13:10: Kind = %v, want None", sig.Kind)
	}
}

// Review: one gap bar closing two half-hours reads the exits on both — RSI high on the earlier one exits.
func TestExit30mTwoHalvesOneBar(t *testing.T) {
	p := DefaultParams()
	p.MACDFast = 0
	bars := flat("2026-09-14 06:00", "2026-09-14 10:25", 100)
	c := 100.0
	for k := 0; k < 6; k++ { // half-hours 10:30..13:00 zigzag up, RSI high on the 13:00 one
		o := c
		if k%2 == 0 {
			c += 0.3
		} else {
			c -= 0.05
		}
		bars = append(bars, half(at("2026-09-14 10:30").Add(time.Duration(k)*30*time.Minute).Format("2006-01-02 15:04"), o, c+0.01, o-0.01, c)...)
	}
	bars = bars[:len(bars)-1] // 13:00 half-hour loses its 13:25 bar
	// 13:55 bar: closes the 13:00 half-hour (gap) and its own 13:30 half-hour, which drops hard
	bars = append(bars, bar{"2026-09-14 13:55", c, c, c - 3, c - 3})
	pos := strategy.Position{PurchasePrice: 100, EntryATR: 100, EntryTime: at("2026-09-14 10:30")}
	if sig := heldAt(p, bars, pos); sig.Reason != "RSI" {
		t.Fatalf("Reason = %q, want RSI from the 13:00 half-hour", sig.Reason)
	}
}
