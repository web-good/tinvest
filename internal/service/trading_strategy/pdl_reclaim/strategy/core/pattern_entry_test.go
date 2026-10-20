package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

// hammerEntryDay: 09:45 breaks PDL 99 (red close 98.7), 10:00 is a red hammer under PDL (tail to
// 98.0), 10:15 is red, 10:30 is the first green bar after the hammer (still under PDL), 10:45 is
// green again.
func hammerEntryDay() []bar {
	return []bar{
		{"2026-09-14 09:30", 100, 100, 99.5, 99.5},
		{"2026-09-14 09:45", 99.5, 99.5, 98.7, 98.7},
		{"2026-09-14 10:00", 98.7, 98.75, 98.0, 98.6},
		{"2026-09-14 10:15", 98.6, 98.6, 98.5, 98.5},
		{"2026-09-14 10:30", 98.5, 98.8, 98.5, 98.8},
		{"2026-09-14 10:45", 98.8, 98.9, 98.8, 98.9},
	}
}

// engulfEntryDay: 09:45 breaks PDL, 10:00 is red, 10:15 engulfs it (low 98.5 under 98.6, close
// 98.8 over the red high 98.7), 10:30 is the first green bar after the engulfing.
func engulfEntryDay() []bar {
	return []bar{
		{"2026-09-14 09:30", 100, 100, 99.5, 99.5},
		{"2026-09-14 09:45", 99.5, 99.5, 98.7, 98.7},
		{"2026-09-14 10:00", 98.7, 98.7, 98.6, 98.6},
		{"2026-09-14 10:15", 98.6, 98.8, 98.5, 98.8},
		{"2026-09-14 10:30", 98.8, 98.9, 98.8, 98.9},
	}
}

func patternEntryP(engulf, hammer int) Params {
	p := DefaultParams()
	p.PDLPatternEntry, p.PDLEngulf, p.PDLHammer = 1, engulf, hammer
	return p
}

func TestPatternEntryHammerBuysFirstGreenAfter(t *testing.T) {
	d := hammerEntryDay()
	for i, want := range []model.SignalKind{model.SignalNone, model.SignalNone, model.SignalNone, model.SignalNone, model.SignalBuy, model.SignalNone} {
		sig := decide(patternEntryP(0, 1), d[:i+1])
		if sig.Kind != want {
			t.Fatalf("%s: Kind = %v, want %v (reason %q)", d[i].t, sig.Kind, want, sig.EntryReason)
		}
		if want == model.SignalBuy && !strings.Contains(sig.EntryReason, "молот у PDL на свече 10:00, первая зелёная после паттерна 10:30") {
			t.Fatalf("reason %q", sig.EntryReason)
		}
	}
}

// A green hammer is the structure itself: the next green bar is the trigger.
func TestPatternEntryGreenHammer(t *testing.T) {
	d := hammerEntryDay()[:3]
	d[2] = bar{"2026-09-14 10:00", 98.6, 98.75, 98.0, 98.7}
	if sig := decide(patternEntryP(0, 1), d); sig.Kind != model.SignalNone {
		t.Fatalf("hammer bar: Kind = %v, want None", sig.Kind)
	}
	d = append(d, bar{"2026-09-14 10:15", 98.7, 98.8, 98.7, 98.8})
	if sig := decide(patternEntryP(0, 1), d); sig.Kind != model.SignalBuy {
		t.Fatalf("green after hammer: Kind = %v, want Buy", sig.Kind)
	}
}

func TestPatternEntryEngulfBuysFirstGreenAfter(t *testing.T) {
	d := engulfEntryDay()
	if sig := decide(patternEntryP(1, 0), d[:4]); sig.Kind != model.SignalNone {
		t.Fatalf("engulfing bar: Kind = %v, want None", sig.Kind)
	}
	sig := decide(patternEntryP(1, 0), d)
	if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "поглощение у PDL на свече 10:15, первая зелёная после паттерна 10:30") {
		t.Fatalf("Kind = %v, reason %q", sig.Kind, sig.EntryReason)
	}
}

func TestPatternEntryNeedsEnabledKind(t *testing.T) {
	if sig := decide(patternEntryP(1, 0), hammerEntryDay()[:5]); sig.Kind != model.SignalNone {
		t.Fatalf("hammer day, engulf only: Kind = %v, want None", sig.Kind)
	}
	if sig := decide(patternEntryP(0, 1), engulfEntryDay()); sig.Kind != model.SignalNone {
		t.Fatalf("engulf day, hammer only: Kind = %v, want None", sig.Kind)
	}
	if sig := decide(patternEntryP(0, 0), hammerEntryDay()[:5]); sig.Kind != model.SignalNone {
		t.Fatalf("no kinds: Kind = %v, want None", sig.Kind)
	}
}

// With the switch off the same day has no reclaim of PDL: no buy.
func TestPatternEntryOffNeedsReclaim(t *testing.T) {
	p := patternEntryP(0, 1)
	p.PDLPatternEntry = 0
	if sig := decide(p, hammerEntryDay()[:5]); sig.Kind != model.SignalNone {
		t.Fatalf("switch off: Kind = %v, want None", sig.Kind)
	}
}

// A hammer that formed before any close below PDL is not a pattern at PDL.
func TestPatternEntryHammerAbovePDLIgnored(t *testing.T) {
	d := []bar{
		{"2026-09-14 09:30", 99.9, 100, 99.3, 99.95},
		{"2026-09-14 09:45", 99.95, 99.95, 99.5, 99.5},
		{"2026-09-14 10:00", 99.5, 99.6, 99.5, 99.6},
	}
	if sig := decide(patternEntryP(0, 1), d); sig.Kind != model.SignalNone {
		t.Fatalf("hammer above PDL: Kind = %v, want None", sig.Kind)
	}
}

func TestExplainPatternEntry(t *testing.T) {
	out := newS(patternEntryP(0, 1)).Explain(fixture(warm(hammerEntryDay()[:5])))
	for _, want := range []string{"✓ вход по паттерну у PDL (поглощение 0, молот 1): молот 10:00, первая зелёная после паттерна 10:30", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	out = newS(patternEntryP(0, 1)).Explain(fixture(warm(hammerEntryDay())))
	for _, want := range []string{"✗ вход по паттерну у PDL (поглощение 0, молот 1): нет", "ИТОГ: входа нет"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestPatternEntryStopAndTarget(t *testing.T) {
	green := hammerEntryDay()[:3]
	green[2] = bar{"2026-09-14 10:00", 98.6, 98.75, 98.0, 98.7}
	green = append(green, bar{"2026-09-14 10:15", 98.7, 98.8, 98.7, 98.8})
	for name, tc := range map[string]struct {
		p            Params
		bars         []bar
		stop, target float64
		why          string
	}{
		// red hammer 10:00: its own low; entry close 98.8, risk 0.8
		"red hammer": {patternEntryP(0, 1), hammerEntryDay()[:5], 98.0, 100.4, "стоп по лою свечи паттерна 10:00 98.0000, цель 2.00R"},
		// green hammer 10:00: the red 09:45 before it, low 98.7; risk 0.1
		"green hammer": {patternEntryP(0, 1), green, 98.7, 99.0, "стоп по лою красной перед паттерном 09:45 98.7000, цель 2.00R"},
		// engulfing 10:15 is green: the red 10:00 it engulfed, low 98.6; entry 98.9, risk 0.3
		"engulfing": {patternEntryP(1, 0), engulfEntryDay(), 98.6, 99.5, "стоп по лою красной перед паттерном 10:00 98.6000, цель 2.00R"},
	} {
		sig := decide(tc.p, tc.bars)
		if sig.Kind != model.SignalBuy {
			t.Fatalf("%s: Kind = %v, want Buy", name, sig.Kind)
		}
		if math.Abs(sig.StopLoss-tc.stop) > eps || math.Abs(sig.TakeProfit-tc.target) > eps {
			t.Errorf("%s: stop %.4f target %.4f, want %.4f %.4f", name, sig.StopLoss, sig.TakeProfit, tc.stop, tc.target)
		}
		if !strings.Contains(sig.EntryReason, tc.why) {
			t.Errorf("%s: reason %q, want %q", name, sig.EntryReason, tc.why)
		}
	}
}

func TestPatternEntryTargetR(t *testing.T) {
	p := patternEntryP(0, 1)
	p.PDLPatternTargetR = 3
	if sig := decide(p, hammerEntryDay()[:5]); math.Abs(sig.TakeProfit-101.2) > eps {
		t.Fatalf("R 3: target %.4f, want 101.2", sig.TakeProfit)
	}
	if got := DefaultParams().PDLPatternTargetR; got != 2 {
		t.Fatalf("default PDLPatternTargetR = %v, want 2", got)
	}
}

// A pattern-entry position stops at the structural stop it carries (98.3 on pos0), not at the
// fill − StopDailyATR × ATR (99.4 − 0.5·2 = 98.4).
func TestPatternEntryExitAtCarriedStop(t *testing.T) {
	p := patternEntryP(0, 1)
	if sig := held(p, []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.35, 98.6}}, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("low 98.35: got %v %q, want None", sig.Kind, sig.Reason)
	}
	sig := held(p, []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.3, 98.6}}, pos0())
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-98.3) > eps {
		t.Fatalf("low 98.3: got %q at %v, want SL at 98.3", sig.Reason, sig.StopLoss)
	}
}
