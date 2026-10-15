package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

// friday is the last bar of the previous session; gapOpen opens Monday below PDL 99.
var (
	friday  = bar{"2026-09-11 23:45", 100, 100, 100, 100}
	gapOpen = bar{"2026-09-14 07:00", 98.5, 98.5, 98.5, 98.5}
)

func TestGapOnly(t *testing.T) {
	p := m5()
	p.GapOnly = 1
	if sig := decide(p, []bar{pre, barA}); sig.Kind != model.SignalNone {
		t.Fatalf("day opened above PDL: Kind = %v, want None", sig.Kind)
	}
	md := fixture(append([]bar{friday, gapOpen}, warm([]bar{pre, barA})...))
	if sig := newS(p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("day opened below PDL: Kind = %v, want Buy", sig.Kind)
	}
	// the window starts mid-day: the day's open is unknown — no entry
	md = fixture(append([]bar{gapOpen}, warm([]bar{pre, barA})...))
	if sig := newS(p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("day start not in window: Kind = %v, want None", sig.Kind)
	}
}

func TestConfirmA(t *testing.T) {
	p := m5()
	p.Confirm = 1
	if sig := decide(p, []bar{pre, barA}); sig.Kind != model.SignalNone {
		t.Fatalf("pattern bar itself: Kind = %v, want None", sig.Kind)
	}
	held := bar{"2026-09-14 10:05", 99.3, 99.45, 99.25, 99.4}
	sig := decide(p, []bar{pre, barA, held})
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "A") {
		t.Fatalf("confirmed: %v %q, want Buy A", sig.Kind, sig.EntryReason)
	}
	if math.Abs(sig.StopLoss-98.4) > eps || !sig.FillBy.Equal(at("2026-09-14 10:10")) {
		t.Fatalf("SL %v FillBy %v, want 98.4 (from the confirming close) and 10:10", sig.StopLoss, sig.FillBy)
	}
	lost := bar{"2026-09-14 10:05", 99.3, 99.3, 98.8, 98.9}
	if sig := decide(p, []bar{pre, barA, lost}); sig.Kind != model.SignalNone {
		t.Fatalf("back under PDL: Kind = %v, want None", sig.Kind)
	}
}

func TestConfirmB(t *testing.T) {
	p := m5()
	p.Confirm = 1
	conf := bar{"2026-09-14 10:05", 98.95, 99.2, 98.9, 99.1}
	if sig := decide(p, []bar{redB, greenB, conf}); sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "B") {
		t.Fatalf("got %v %q, want Buy B", sig.Kind, sig.EntryReason)
	}
}

func TestConfirmSameDate(t *testing.T) {
	p := m5()
	p.Confirm = 1
	a := barA
	a.t = "2026-09-11 17:50" // Friday
	q := pre
	q.t = "2026-09-11 17:45"
	next := bar{"2026-09-14 07:00", 99.3, 99.45, 99.25, 99.4}
	if sig := decide(p, []bar{q, a, next}); sig.Kind != model.SignalNone {
		t.Fatalf("pattern Friday, confirm Monday: Kind = %v, want None", sig.Kind)
	}
}

func TestTargetMid(t *testing.T) {
	p := m5()
	p.TargetMid = 1
	sig := decide(p, []bar{pre, barA})
	if sig.Kind != model.SignalBuy || math.Abs(sig.TakeProfit-100) > eps {
		t.Fatalf("%v TP %v, want Buy with target 100 (mid of 99..101)", sig.Kind, sig.TakeProfit)
	}
	above := bar{"2026-09-14 10:00", 99.2, 100.3, 98.8, 100.2}
	if sig := decide(p, []bar{pre, above}); sig.Kind != model.SignalNone {
		t.Fatalf("close above mid: Kind = %v, want None", sig.Kind)
	}
}
