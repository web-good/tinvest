package core

import (
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
