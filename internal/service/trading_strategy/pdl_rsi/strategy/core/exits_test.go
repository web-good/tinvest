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

// riseDay is Tuesday by hours from 10:00: a dip to 99.0 (RSI 0), then a climb whose RSI crosses
// 70 on a later hour.
func riseDay() []bar {
	return path("2026-09-15 10:00", 100, 99.5, 99.0, 100.0, 100.5, 101.0, 101.5, 102.0)
}

// rise70 is the first hour of riseDay whose RSI crosses 70.
func rise70(t *testing.T, p Params, day []bar) int {
	t.Helper()
	r := rsiOf(p, day)
	for j := 1; j < len(day); j++ {
		if crossUp(r[j-1], r[j], 70) {
			return j
		}
	}
	t.Fatalf("fixture: no cross of 70 (RSI %v)", r)
	return -1
}

func TestStopHitIsSL(t *testing.T) {
	p, day := testParams(), riseDay()
	pos := posAt(day, 0, 100, 99.2)
	i, sig := firstExit(p, day, 0, pos)
	if i != 1 || sig.Reason != "SL" || sig.StopLoss != pos.StopLoss || !strings.Contains(sig.ExitReason, "стоп") {
		t.Fatalf("exit at %d %q stop %v (%q), want 1 SL %v", i, sig.Reason, sig.StopLoss, sig.ExitReason, pos.StopLoss)
	}
}

// climbDay is Tuesday by hours from 10:00: from a 100 open, closes 100.5, 101.0, 101.5, 102.5.
func climbDay() []bar {
	return path("2026-09-15 10:00", 100, 100.5, 101.0, 101.5, 102.5)
}

func TestStopAndTargetFromFill(t *testing.T) {
	// Filled at 100 with an entry ATR of 2: the stop 99 (0.5 ATR under the fill, not the 97 the
	// signal carried), the target 102 (2 R above it).
	p, day := testParams(), climbDay()
	p.RSIExit = 0
	pos := posAt(day, 0, 100, 97)
	pos.EntryATR = 2
	i, sig := firstExit(p, day, 0, pos)
	if i != 3 || sig.Reason != "TP" || math.Abs(sig.TakeProfit-102) > eps || !strings.Contains(sig.ExitReason, "цели") {
		t.Fatalf("exit at %d %q target %v (%q), want 3 TP 102", i, sig.Reason, sig.TakeProfit, sig.ExitReason)
	}
	if i, sig := firstExit(p, withLow(day, 1, 98.9), 0, pos); i != 1 || sig.Reason != "SL" || math.Abs(sig.StopLoss-99) > eps {
		t.Fatalf("exit at %d %q stop %v, want 1 SL 99", i, sig.Reason, sig.StopLoss)
	}
	p.TargetR = 0
	if i, sig := firstExit(p, day, 0, pos); i != -1 {
		t.Fatalf("TargetR 0: exit at %d %q, want none", i, sig.Reason)
	}
}

func TestMaxStopPctFromFill(t *testing.T) {
	// Filled at 100 with an entry ATR of 2: 0.5 ATR is 1.0, a 0.5% cap moves the stop to 99.5 and
	// the target to 101 (2 R of the capped stop).
	p, day := testParams(), climbDay()
	p.RSIExit, p.MaxStopPct = 0, 0.5
	pos := posAt(day, 0, 100, 97)
	pos.EntryATR = 2
	if i, sig := firstExit(p, withLow(day, 1, 99.4), 0, pos); i != 1 || sig.Reason != "SL" || math.Abs(sig.StopLoss-99.5) > eps {
		t.Fatalf("exit at %d %q stop %v, want 1 SL 99.5", i, sig.Reason, sig.StopLoss)
	}
	if i, sig := firstExit(p, day, 0, pos); i != 1 || sig.Reason != "TP" || math.Abs(sig.TakeProfit-101) > eps {
		t.Fatalf("exit at %d %q target %v, want 1 TP 101", i, sig.Reason, sig.TakeProfit)
	}
}

func TestStopBeforeTarget(t *testing.T) {
	// One hour spans both the stop and the target: the stop wins.
	p, day := testParams(), climbDay()
	pos := posAt(day, 0, 100, 97)
	pos.EntryATR = 2
	if sig := hold(p, withLow(day, 3, 98.5), 3, pos); sig.Reason != "SL" {
		t.Fatalf("got %q, want SL", sig.Reason)
	}
}

func TestRSIExit(t *testing.T) {
	p, day := testParams(), riseDay()
	at70 := rise70(t, p, day)
	pos := posAt(day, 0, 99.5, 98) // under every low after the dip, the 102.5 target out of reach
	i, sig := firstExit(p, day, 0, pos)
	if i != at70 || sig.Reason != "RSI" || math.Abs(sig.Price-day[at70].c) > eps || !strings.Contains(sig.ExitReason, "RSI") {
		t.Fatalf("exit at %d %q price %v, want %d RSI at the close", i, sig.Reason, sig.Price, at70)
	}
	p.RSIExit = 0
	if i, sig := firstExit(p, day, 0, pos); i != -1 {
		t.Fatalf("RSIExit 0: exit at %d %q, want none", i, sig.Reason)
	}
}

func TestStochExit(t *testing.T) {
	// A climb lifts %K over %D high up; the 15:00 pullback turns %K under %D above 40.
	p := testParams()
	p.RSIExit, p.TargetR, p.StochExitAbove = 0, 0, 40
	day := path("2026-09-15 10:00", 100, 100.5, 101.0, 101.5, 102.0, 102.5, 102.0, 102.2)
	k, d := stochOf(p, day)
	at := -1
	for j := 1; j < len(day) && at < 0; j++ {
		if crossUnder(k, d, j) {
			at = j
		}
	}
	if at < 0 || k[at] <= 40 {
		t.Fatalf("fixture: %%K %v, %%D %v", k, d)
	}
	pos := posAt(day, 0, 100, 97)
	i, sig := firstExit(p, day, 0, pos)
	if i != at || sig.Reason != "STOCH" || math.Abs(sig.Price-day[at].c) > eps || !strings.Contains(sig.ExitReason, "вниз выше 40") {
		t.Fatalf("exit at %d %q price %v (%q), want %d STOCH at the close", i, sig.Reason, sig.Price, sig.ExitReason, at)
	}
	p.StochExitAbove = k[at]
	if i, sig := firstExit(p, day, 0, pos); i != -1 {
		t.Fatalf("StochExitAbove at the cross %%K %v: exit at %d %q, want none", k[at], i, sig.Reason)
	}
	p.StochExitAbove = 0
	if i, sig := firstExit(p, day, 0, pos); i != -1 {
		t.Fatalf("StochExitAbove 0: exit at %d %q, want none", i, sig.Reason)
	}
}

func TestStopBeforeRSI(t *testing.T) {
	// The 11:00 hour jumps to 105: its RSI crosses 70; a tail under the stop on the same hour wins.
	p := testParams()
	p.TargetR = 0
	day := path("2026-09-15 10:00", 100, 99.0, 105, 105)
	r := rsiOf(p, day)
	if !crossUp(r[0], r[1], 70) {
		t.Fatalf("fixture: RSI %v", r)
	}
	pos := posAt(day, 0, 100, 98)
	if i, sig := firstExit(p, day, 1, pos); i != 1 || sig.Reason != "RSI" {
		t.Fatalf("exit at %d %q, want 1 RSI", i, sig.Reason)
	}
	if sig := hold(p, withLow(day, 1, 97.5), 1, pos); sig.Reason != "SL" || sig.StopLoss != 98 {
		t.Fatalf("got %q stop %v, want SL 98", sig.Reason, sig.StopLoss)
	}
}

func TestCarriesOvernight(t *testing.T) {
	// Filled Friday 20:00, flat over the weekend, the stop hit on Monday's 10:00 hour.
	p := testParams()
	fri := path("2026-09-18 20:00", 100, 100, 100, 100, 100)
	mon := path("2026-09-21 06:00", 100, slices.Repeat([]float64{100}, 4)...)
	mon = append(mon, path("2026-09-21 10:00", 100, 98.8)...)
	day := append(fri, mon...)
	pos := posAt(day, 0, 100, 99)
	if i, sig := firstExit(p, day, 0, pos); i != len(day)-1 || sig.Reason != "SL" {
		t.Fatalf("exit at %d %q, want %d SL", i, sig.Reason, len(day)-1)
	}
}

func TestWarmupZerosNeverExit(t *testing.T) {
	// Without the warm-up the RSI starts with HourRSIPeriod zeros followed by 50 on flat bars: that
	// step is no cross. Flat hours over the stop must not exit.
	p := testParams()
	day := path("2026-09-15 06:00", 100, slices.Repeat([]float64{100}, 10)...)
	pos := posAt(day, 0, 100.5, 98)
	md := fixture(day, time.Time{})
	md.Position = &pos
	if sig := NewWithParams("TEST", step, p).Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("sold %q stop %v on the warm-up zeros", sig.Reason, sig.StopLoss)
	}
}

func TestDecideRoundTrip(t *testing.T) {
	p := testParams()
	day := append(canonical(), path("2026-09-15 16:00", 100.5, 101.0, 101.5, 102.0, 102.5)...)
	checkCanonical(t, p, day[:canonTrigger+1])
	buy := run(p, day, canonTrigger, time.Time{})
	if buy.Kind != model.SignalBuy {
		t.Fatal("fixture: no buy at the canonical trigger")
	}
	fill := canonTrigger + 1
	pos := strategy.Position{PurchasePrice: day[fill].o, Quantity: 1, StopLoss: buy.StopLoss,
		TakeProfit: buy.TakeProfit, EntryATR: buy.ATR, EntryTime: day[fill].t}
	for i := fill; i < len(day); i++ {
		md := fixture(warm(day)[:warmBars+i+1], time.Time{})
		md.Position = &pos
		sig := NewWithParams("TEST", step, p).Decide(md)
		if sig.Kind == model.SignalBuy {
			t.Fatalf("bought at %d while in position", i)
		}
		if sig.Kind == model.SignalSell {
			if sig.Reason != "RSI" && sig.Reason != "TP" {
				t.Fatalf("exit at %d %q, want RSI or TP", i, sig.Reason)
			}
			return
		}
	}
	t.Fatal("no exit")
}
