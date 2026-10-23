package core

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// hold is Decide on the warmed bars up to and including day[i] with pos open.
func hold(p Params, day []bar, i int, pos strategy.Position) model.Signal {
	md := fixture(warm(day)[:warmBars+i+1], time.Time{})
	md.Position = &pos
	return NewWithParams("TEST", step, p).Decide(md)
}

// firstExit is the first index from `from` on whose bar Decide sells pos, and its signal; -1 when
// none.
func firstExit(p Params, day []bar, from int, pos strategy.Position) (int, model.Signal) {
	for i := from; i < len(day); i++ {
		if sig := hold(p, day, i, pos); sig.Kind == model.SignalSell {
			return i, sig
		}
	}
	return -1, model.Signal{}
}

// posAt is a position filled at the open of day[i].
func posAt(day []bar, i int, price, stop float64) strategy.Position {
	return strategy.Position{PurchasePrice: price, Quantity: 1, StopLoss: stop, EntryTime: day[i].t}
}

// riseDay is Tuesday from 10:00: the 10:00 hour slides to 99.0 (hourly RSI 0), the 11:00 hour
// climbs to 100.0 (the hourly RSI crosses 50 at its 11:45 close), the 12:00 hour climbs to 100.8
// (it crosses 70 at 12:45), then a further climb.
func riseDay() []bar {
	return path("2026-09-15 10:00", 100, 99.6, 99.4, 99.2, 99.0, 99.3, 99.6, 99.8, 100.0,
		100.2, 100.4, 100.6, 100.8, 101.0, 101.2, 101.4, 101.6, 101.8, 102.0, 102.2, 102.4)
}

const (
	rise50 = 7  // 11:45: completes the hour crossing 50
	rise70 = 11 // 12:45: completes the hour crossing 70
)

func checkRise(t *testing.T, p Params, day []bar) {
	t.Helper()
	rh := hourRSI(p, day)
	if !crossUp(rh[rise50-4], rh[rise50], 50) || rh[rise50] >= 70 {
		t.Fatalf("fixture: hour RSI %v → %v, want a cross of 50 only", rh[rise50-4], rh[rise50])
	}
	if !crossUp(rh[rise50], rh[rise70], 70) {
		t.Fatalf("fixture: hour RSI %v → %v, want a cross of 70", rh[rise50], rh[rise70])
	}
}

func TestStopHitIsSL(t *testing.T) {
	p, day := DefaultParams(), riseDay()
	pos := posAt(day, 0, 100, 99.5)
	i, sig := firstExit(p, day, 0, pos)
	if i != 1 || sig.Reason != "SL" || sig.StopLoss != pos.StopLoss {
		t.Fatalf("exit at %d %q stop %v, want 1 SL %v", i, sig.Reason, sig.StopLoss, pos.StopLoss)
	}
	if !strings.Contains(sig.ExitReason, "стоп") {
		t.Errorf("ExitReason %q", sig.ExitReason)
	}
}

func TestBreakevenAfterHourlyCross50(t *testing.T) {
	p, day := DefaultParams(), riseDay()
	checkRise(t, p, day)
	pos := posAt(day, 0, 100, 98)
	if day[rise50+1].l > pos.PurchasePrice || day[rise50+1].l <= pos.StopLoss {
		t.Fatalf("fixture: bar %d low %v", rise50+1, day[rise50+1].l)
	}
	i, sig := firstExit(p, day, 0, pos)
	if i != rise50+1 || sig.Reason != "TRAIL" || sig.StopLoss != pos.PurchasePrice {
		t.Fatalf("exit at %d %q stop %v, want %d TRAIL %v", i, sig.Reason, sig.StopLoss, rise50+1, pos.PurchasePrice)
	}
	if !strings.Contains(sig.ExitReason, "безубыт") {
		t.Errorf("ExitReason %q", sig.ExitReason)
	}
}

func TestBreakevenNotOnTheCrossBarItself(t *testing.T) {
	p, day := DefaultParams(), riseDay()
	checkRise(t, p, day)
	pos := posAt(day, 0, 100, 98)
	if day[rise50].l > pos.PurchasePrice || day[rise50].l <= pos.StopLoss {
		t.Fatalf("fixture: bar %d low %v", rise50, day[rise50].l)
	}
	if sig := hold(p, day, rise50, pos); sig.Kind == model.SignalSell {
		t.Fatalf("sold %q on the bar completing the 50-cross hour", sig.Reason)
	}
}

func TestBreakevenNeverLowersStop(t *testing.T) {
	// The stop sits above the fill: the breakeven move must not lower it.
	p, day := DefaultParams(), riseDay()
	pos := posAt(day, 0, 99.5, 100.1)
	i, sig := firstExit(p, day, rise50+1, pos)
	if i != rise50+1 || sig.StopLoss != 100.1 {
		t.Fatalf("exit at %d stop %v, want %d at 100.1", i, sig.StopLoss, rise50+1)
	}
}

func TestCross50BeforeEntryIgnored(t *testing.T) {
	p, day := DefaultParams(), riseDay()
	checkRise(t, p, day)
	// Filled at 12:00, after the 11:00 hour crossed 50; its low 100.0 touches the fill.
	pos := posAt(day, rise50+1, 100, 98)
	if sig := hold(p, day, rise50+1, pos); sig.Kind == model.SignalSell {
		t.Fatalf("sold %q: the 50 cross before the entry moved the stop", sig.Reason)
	}
	sl := withLow(day, rise50+2, 97.9)
	if sig := hold(p, sl, rise50+2, pos); sig.Reason != "SL" || sig.StopLoss != 98 {
		t.Fatalf("got %q stop %v, want SL 98", sig.Reason, sig.StopLoss)
	}
}

func TestBreakevenOff(t *testing.T) {
	p, day := DefaultParams(), riseDay()
	p.RSIBreakeven, p.RSIExit = 0, 0
	pos := posAt(day, 0, 100, 98)
	if i, sig := firstExit(p, day, 0, pos); i != -1 {
		t.Fatalf("exit at %d %q, want none", i, sig.Reason)
	}
	sl := withLow(day, rise50+2, 97.9)
	if i, sig := firstExit(p, sl, 0, pos); i != rise50+2 || sig.Reason != "SL" || sig.StopLoss != 98 {
		t.Fatalf("exit at %d %q stop %v, want %d SL 98", i, sig.Reason, sig.StopLoss, rise50+2)
	}
}

// flatHourDay is riseDay with the 12:00 hour flat at 101.0 from its first bar: the forming hour
// already sits at its crossing close on 12:00–12:30, but the hour completes only at 12:45.
func flatHourDay() []bar {
	return path("2026-09-15 10:00", 100, 99.6, 99.4, 99.2, 99.0, 99.3, 99.6, 99.8, 100.0,
		101.0, 101.0, 101.0, 101.0, 101.2, 101.4)
}

func TestRSIExitOnHourCompletion(t *testing.T) {
	p, day := DefaultParams(), flatHourDay()
	rh := hourRSI(p, day)
	if !crossUp(rh[rise50], rh[rise70], 70) {
		t.Fatalf("fixture: hour RSI %v → %v", rh[rise50], rh[rise70])
	}
	pos := posAt(day, 0, 99.5, 98) // under every low after the 50 cross: no breakeven exit
	i, sig := firstExit(p, day, 0, pos)
	if i != rise70 || sig.Reason != "RSI" {
		t.Fatalf("exit at %d %q, want %d RSI", i, sig.Reason, rise70)
	}
	if !strings.Contains(sig.ExitReason, "RSI") {
		t.Errorf("ExitReason %q", sig.ExitReason)
	}
	p.RSIExit = 0
	if i, sig := firstExit(p, day, 0, pos); i != -1 {
		t.Fatalf("RSIExit 0: exit at %d %q, want none", i, sig.Reason)
	}
}

// jumpDay slides to 99.0 in the 10:00 hour and jumps to 105 at 11:00: the 11:00 hour crosses both
// 50 and 70 at its 11:45 close.
func jumpDay() []bar {
	return path("2026-09-15 10:00", 100, 99.6, 99.4, 99.2, 99.0, 105, 105, 105, 105, 105, 105)
}

func TestStopBeforeRSI(t *testing.T) {
	p, day := DefaultParams(), jumpDay()
	const done = 7
	rh := hourRSI(p, day)
	if !crossUp(rh[3], rh[done], 70) {
		t.Fatalf("fixture: hour RSI %v → %v", rh[3], rh[done])
	}
	pos := posAt(day, 0, 100, 98)
	if i, sig := firstExit(p, day, 0, pos); i != done || sig.Reason != "RSI" {
		t.Fatalf("exit at %d %q, want %d RSI", i, sig.Reason, done)
	}
	tail := withLow(day, done, 97.5)
	if sig := hold(p, tail, done, pos); sig.Reason != "SL" || sig.StopLoss != 98 {
		t.Fatalf("got %q stop %v, want SL 98", sig.Reason, sig.StopLoss)
	}
}

func TestCarriesOvernight(t *testing.T) {
	// Filled Friday 22:00, flat over the weekend, the stop hit on Monday's 10:15 bar.
	p := DefaultParams()
	fri := path("2026-09-18 22:00", 100, 100, 100, 100, 100, 100, 100, 100, 100)
	mon := path("2026-09-21 07:00", 100, slices.Repeat([]float64{100}, 12)...)
	mon = append(mon, path("2026-09-21 10:00", 100, 100, 98.8)...)
	day := append(fri, mon...)
	pos := posAt(day, 0, 100, 99)
	i, sig := firstExit(p, day, 0, pos)
	if i != len(day)-1 || sig.Reason != "SL" {
		t.Fatalf("exit at %d %q, want %d SL", i, sig.Reason, len(day)-1)
	}
}

func TestEntryOlderThanWindow(t *testing.T) {
	p, day := DefaultParams(), riseDay()
	checkRise(t, p, day)
	pos := posAt(day, 0, 100, 98)
	pos.EntryTime = warm(day)[0].t.Add(-24 * time.Hour)
	sig := hold(p, day, rise50+1, pos)
	if sig.Reason != "TRAIL" || sig.StopLoss != 100 {
		t.Fatalf("got %q stop %v, want TRAIL at the fill 100", sig.Reason, sig.StopLoss)
	}
}

func TestWarmupZerosNeverMoveStop(t *testing.T) {
	// The window's hourly RSI starts with RSIPeriod warm-up zeros followed by 50 on flat bars: that
	// step is no cross. A bar touching the fill must not exit.
	p := DefaultParams()
	day := path("2026-09-15 10:00", 100, 100, 100, 100, 100)
	pos := posAt(day, 0, 100.5, 98)
	pos.EntryTime = warm(day)[0].t.Add(-24 * time.Hour)
	if sig := hold(p, day, len(day)-1, pos); sig.Kind == model.SignalSell {
		t.Fatalf("sold %q stop %v on the warm-up zeros", sig.Reason, sig.StopLoss)
	}
}

func TestGapBelowStop(t *testing.T) {
	// The fill bar opens at 98, below the 99 stop: the engine fills min(stop, open).
	p := DefaultParams()
	day := path("2026-09-15 10:00", 98, 98.5, 98.7)
	pos := posAt(day, 0, 98, 99)
	sig := hold(p, day, 0, pos)
	if sig.Reason != "SL" || sig.StopLoss != 99 {
		t.Fatalf("got %q stop %v, want SL 99", sig.Reason, sig.StopLoss)
	}
}

func TestDecideRoundTrip(t *testing.T) {
	p := DefaultParams()
	day := append(canonical(), path("2026-09-15 14:00", 100.3,
		100.5, 100.7, 100.9, 101.1, 101.3, 101.5, 101.7, 101.9, 102.1, 102.3, 102.5, 102.7)...)
	checkCanonical(t, p, day[:16])
	loud := loudFrom(day, canonFrom)
	buy := run(p, day, canonCross, loud)
	if buy.Kind != model.SignalBuy {
		t.Fatal("fixture: no buy at the canonical cross")
	}
	fill := canonCross + 1
	pos := strategy.Position{PurchasePrice: day[fill].o, Quantity: 1, StopLoss: buy.StopLoss, EntryTime: day[fill].t}
	const exitBar = 19 // 14:45: the 14:00 hour's RSI crosses 70
	rh := hourRSI(p, day)
	if !crossUp(rh[exitBar-4], rh[exitBar], p.RSIExit) {
		t.Fatalf("fixture: hour RSI %v → %v", rh[exitBar-4], rh[exitBar])
	}
	for i := fill; i < len(day); i++ {
		md := fixture(warm(day)[:warmBars+i+1], loud)
		md.Position = &pos
		sig := NewWithParams("TEST", step, p).Decide(md)
		if sig.Kind == model.SignalBuy {
			t.Fatalf("bought at %d while in position", i)
		}
		if sig.Kind != model.SignalSell {
			continue
		}
		if i != exitBar || sig.Reason != "RSI" {
			t.Fatalf("exit at %d %q, want %d RSI", i, sig.Reason, exitBar)
		}
		if math.Abs(sig.Price-day[i].c) > eps {
			t.Fatalf("Price %v, want the close %v", sig.Price, day[i].c)
		}
		return
	}
	t.Fatal("no exit")
}

func TestSignalWhileInPositionBurnsSetup(t *testing.T) {
	// twoCrossDay crosses in zone at 12:00 and again at 12:30 with no re-arm. A position held at
	// 12:00 swallows the first signal; it stops out at 12:15 and the 12:30 cross must not buy.
	p := DefaultParams()
	p.MinRoomATR = 0 // the control's 98.0 tail would otherwise use up the room
	const first, out, second = 8, 9, 10
	day := withLow(twoCrossDay(), out, 99.0) // a 12:15 tail under the 99.1 stop
	loud := loudFrom(day, 4)
	pos := posAt(day, 0, 100, 99.1)
	if day[first].l <= pos.StopLoss {
		t.Fatalf("fixture: bar %d low %v hits the stop", first, day[first].l)
	}
	for i := first; i < out; i++ {
		if sig := hold(p, day, i, pos); sig.Kind != model.SignalNone {
			t.Fatalf("bar %d: %v %q while in position", i, sig.Kind, sig.Reason)
		}
	}
	if sig := hold(p, day, out, pos); sig.Reason != "SL" {
		t.Fatalf("bar %d: %q, want SL", out, sig.Reason)
	}
	if sig := run(p, day, second, loud); sig.Kind == model.SignalBuy {
		t.Fatal("bought the second cross of a setup burnt while in position")
	}

	// Control: with the first cross's low below the zone the setup survives and the second buys.
	ctl := withLow(day, first, 98.0)
	if inZone(p, ctl, first) {
		t.Fatal("fixture: the control's first cross must lie outside the zone")
	}
	if sig := run(p, ctl, second, loud); sig.Kind != model.SignalBuy {
		t.Fatal("control: the second cross must buy")
	}
}
