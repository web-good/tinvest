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

// riseDay is Tuesday by hours from 10:00: a dip to 99.0 (RSI 0), then a climb.
func riseDay() []bar {
	return path("2026-09-15 10:00", 100, 99.5, 99.0, 100.0, 100.5, 101.0, 101.5, 102.0)
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
	// The RSI climbs to 100 and slides: the exit is the hour it crosses 70 down, not the climb
	// through 70.
	p, day := testParams(), peakFallDay()
	p.TargetR = 0
	at70 := firstCrossDown(p, day, 70)
	if at70 < 2 {
		t.Fatalf("fixture: cross of 70 down at %d (RSI %v)", at70, rsiOf(p, day))
	}
	pos := posAt(day, 0, 100, 99)
	i, sig := firstExit(p, day, 0, pos)
	if i != at70 || sig.Reason != "RSI" || math.Abs(sig.Price-day[at70].c) > eps || !strings.Contains(sig.ExitReason, "вниз") {
		t.Fatalf("exit at %d %q price %v, want %d RSI at the close", i, sig.Reason, sig.Price, at70)
	}
	p.RSIExit = 0
	if i, sig := firstExit(p, day, 0, pos); i != -1 {
		t.Fatalf("RSIExit 0: exit at %d %q, want none", i, sig.Reason)
	}
}

func TestStopBeforeRSI(t *testing.T) {
	// The 11:00 hour jumps to 105 (RSI over 70), the later hours slide: the first cross of 70 down
	// exits; a tail under the stop on the same hour wins.
	p := testParams()
	p.TargetR = 0
	day := path("2026-09-15 10:00", 100, 99.0, 105, 104.5, 104, 103.5, 103)
	at70 := firstCrossDown(p, day, 70)
	if at70 < 2 {
		t.Fatalf("fixture: cross of 70 down at %d (RSI %v)", at70, rsiOf(p, day))
	}
	pos := posAt(day, 0, 100, 98)
	if i, sig := firstExit(p, day, 1, pos); i != at70 || sig.Reason != "RSI" {
		t.Fatalf("exit at %d %q, want %d RSI", i, sig.Reason, at70)
	}
	if sig := hold(p, withLow(day, at70, 97.5), at70, pos); sig.Reason != "SL" || sig.StopLoss != 98 {
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
	checkCanonical(t, p, day[:canonSignal+1])
	buy := run(p, day, canonSignal, time.Time{})
	if buy.Kind != model.SignalBuy {
		t.Fatal("fixture: no buy at the canonical signal")
	}
	fill := canonSignal + 1
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

// peakFallDay is Tuesday by hours from 10:00: a climb to 102 (RSI to 100), then a steady slide whose
// RSI crosses 60, 50 and 40 down on later hours.
func peakFallDay() []bar {
	return path("2026-09-15 10:00", 100, 101.0, 102.0, 101.8, 101.6, 101.4, 101.2, 101.0, 100.8, 100.6, 100.4, 100.2, 100.0)
}

// firstCrossDown is the first index of day whose RSI crosses level down; -1 when none.
func firstCrossDown(p Params, day []bar, level float64) int {
	r := rsiOf(p, day)
	for j := 1; j < len(day); j++ {
		if crossDown(r[j-1], r[j], level) {
			return j
		}
	}
	return -1
}

func ladderParams() Params {
	p := testParams()
	p.RSIExit, p.TargetR = 0, 0
	p.RSITrailFrom, p.RSITrailTo, p.RSITrailStep = 50, 70, 10
	return p
}

func TestRSILadderExitsOnTheHighestRung(t *testing.T) {
	// The RSI peaked at 100: the top rung 70 is reached, the exit is the first cross of 60 down.
	p, day := ladderParams(), peakFallDay()
	want := firstCrossDown(p, day, 60)
	if want < 0 || want >= firstCrossDown(p, day, 50) {
		t.Fatalf("fixture: cross of 60 at %d must come before the cross of 50", want)
	}
	pos := posAt(day, 0, 100, 90)
	i, sig := firstExit(p, day, 0, pos)
	if i != want || sig.Reason != "RSITR" || math.Abs(sig.Price-day[want].c) > eps || !strings.Contains(sig.ExitReason, "дошёл до 70") {
		t.Fatalf("exit at %d %q (%q), want %d RSITR off rung 70", i, sig.Reason, sig.ExitReason, want)
	}
}

func TestRSILadderTopRungCapsTheLevel(t *testing.T) {
	// RSITrailTo 50 leaves one rung: whatever the peak, the exit is the cross of 40 down.
	p, day := ladderParams(), peakFallDay()
	p.RSITrailTo = 50
	want := firstCrossDown(p, day, 40)
	if want < 0 {
		t.Fatal("fixture: no cross of 40 down")
	}
	i, sig := firstExit(p, day, 0, posAt(day, 0, 100, 90))
	if i != want || sig.Reason != "RSITR" || !strings.Contains(sig.ExitReason, "дошёл до 50") {
		t.Fatalf("exit at %d %q (%q), want %d RSITR off rung 50", i, sig.Reason, sig.ExitReason, want)
	}
}

func TestRSILadderIgnoresThePeakBeforeTheFill(t *testing.T) {
	// Filled right after the cross of 60 down: the peak of 100 before the fill does not count. The
	// highest RSI from the fill on is in [50, 60), so the rung is 50 and the exit is the cross of 40.
	p, day := ladderParams(), peakFallDay()
	from := firstCrossDown(p, day, 60) + 1
	r := rsiOf(p, day)
	if peak := slices.Max(r[from:]); peak < 50 || peak >= 60 {
		t.Fatalf("fixture: RSI peak after the fill %v, want [50, 60)", peak)
	}
	want := firstCrossDown(p, day, 40)
	i, sig := firstExit(p, day, from, posAt(day, from, 100.5, 90))
	if i != want || sig.Reason != "RSITR" || !strings.Contains(sig.ExitReason, "дошёл до 50") {
		t.Fatalf("exit at %d %q (%q), want %d RSITR off rung 50", i, sig.Reason, sig.ExitReason, want)
	}
}

func TestRSILadderOff(t *testing.T) {
	p, day := ladderParams(), peakFallDay()
	p.RSITrailFrom = 0
	if i, sig := firstExit(p, day, 0, posAt(day, 0, 100, 90)); i != -1 {
		t.Fatalf("RSITrailFrom 0: exit at %d %q, want none", i, sig.Reason)
	}
}

// stopLevelParams — набор без обращения к литералу тикера: стоп 0.7 ATR, кап 1.5% цены.
func stopLevelParams() Params {
	p := DefaultParams()
	p.StopATR, p.MaxStopPct = 0.7, 1.5
	return p
}

func TestStopLevelMatchesExitLevels(t *testing.T) {
	p := stopLevelParams()
	// кап 1.5% (1.5) сильнее 0.7 × 5 = 3.5
	if got := StopLevel(p, 100, 5); math.Abs(got-98.5) > 1e-9 {
		t.Errorf("StopLevel(100, 5) = %v, want 98.5", got)
	}
	// кап не режет: 0.7 × 1 = 0.7
	if got := StopLevel(p, 100, 1); math.Abs(got-99.3) > 1e-9 {
		t.Errorf("StopLevel(100, 1) = %v, want 99.3", got)
	}
	// exitLevels берёт тот же уровень от цены входа и ATR позиции
	stop, _, ok := NewWithParams("TEST", step, p).exitLevels(&strategy.Position{PurchasePrice: 100, EntryATR: 5})
	if !ok || math.Abs(stop-StopLevel(p, 100, 5)) > 1e-9 {
		t.Errorf("exitLevels stop = %v ok=%v, want %v", stop, ok, StopLevel(p, 100, 5))
	}
}

func TestStopLevelZeroWithoutATR(t *testing.T) {
	p := stopLevelParams()
	if got := StopLevel(p, 100, 0); got != 0 {
		t.Errorf("StopLevel(atr 0) = %v, want 0", got)
	}
	if got := StopLevel(p, 0, 5); got != 0 {
		t.Errorf("StopLevel(fill 0) = %v, want 0", got)
	}
	p.StopATR = 0
	if got := StopLevel(p, 100, 5); got != 0 {
		t.Errorf("StopLevel(StopATR 0) = %v, want 0", got)
	}
	p.StopATR, p.MaxStopPct = 30, 0 // уровень ниже нуля
	if got := StopLevel(p, 100, 5); got != 0 {
		t.Errorf("StopLevel(level<=0) = %v, want 0", got)
	}
}
