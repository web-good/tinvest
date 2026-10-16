package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func levelsOnly() Params {
	p := withC(m5())
	p.LevelsOnly = 1
	return p
}

// PDL 99, PDH 101, ATR 2.
var (
	// awayB is an engulfing around 100, far from both levels
	awayB = []bar{{"2026-09-14 09:55", 100.4, 100.45, 100.05, 100.1}, {"2026-09-14 10:00", 100.05, 100.5, 100.0, 100.45}}
	// pdlB is an engulfing whose pair dipped below PDL and closed above it
	pdlB = []bar{{"2026-09-14 09:55", 99.3, 99.3, 98.9, 99.2}, {"2026-09-14 10:00", 99.15, 99.4, 98.85, 99.4}}
	// retestB is an engulfing above PDH that only dipped into it: red closed above PDH, so not C
	retestB = []bar{{"2026-09-14 09:55", 101.4, 101.45, 101.15, 101.2}, {"2026-09-14 10:00", 101.1, 101.6, 100.9, 101.5}}
	// nearB bounces just above PDL: its lowest low 99.05 does not reach 99
	nearB = []bar{{"2026-09-14 09:55", 99.5, 99.5, 99.1, 99.2}, {"2026-09-14 10:00", 99.15, 99.65, 99.05, 99.6}}
)

func TestLevelsOnly(t *testing.T) {
	p := levelsOnly()
	pa := p
	pa.EntryA = 0 // pdlB's green bar is also an A
	for name, c := range map[string]struct {
		p    Params
		bars []bar
		buy  bool
	}{
		"A at PDL":             {p, []bar{pre, barA}, true},
		"engulfing at PDL":     {pa, pdlB, true},
		"C breaks PDH":         {p, []bar{preC, barC}, true},
		"engulfing away":       {p, awayB, false},
		"engulfing under PDL":  {p, []bar{redB, greenB}, false},
		"engulfing PDH retest": {p, retestB, false},
	} {
		sig := decide(c.p, c.bars)
		if got := sig.Kind == model.SignalBuy; got != c.buy {
			t.Errorf("%s: buy = %v (%q), want %v", name, got, sig.EntryReason, c.buy)
		}
	}
	// without the mode the same reversals trade
	q := withC(m5())
	for name, bars := range map[string][]bar{"engulfing away": awayB, "engulfing PDH retest": retestB} {
		if sig := decide(q, bars); sig.Kind != model.SignalBuy {
			t.Errorf("LevelsOnly 0, %s: Kind = %v, want Buy", name, sig.Kind)
		}
	}
}

// LevelTolATR 0.1 with ATR 2: a low within 0.2 above PDL counts as the level.
func TestLevelTolATR(t *testing.T) {
	p := levelsOnly()
	if sig := decide(p, nearB); sig.Kind != model.SignalNone {
		t.Fatalf("tolerance 0, low 99.05: Kind = %v, want None", sig.Kind)
	}
	p.LevelTolATR = 0.1
	sig := decide(p, nearB)
	if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "пересечение PDL") {
		t.Fatalf("tolerance 0.2, low 99.05: %v %q, want Buy at PDL", sig.Kind, sig.EntryReason)
	}
	p.LevelTolATR = 0.02 // 0.04: 99.05 is out of reach
	if sig := decide(p, nearB); sig.Kind != model.SignalNone {
		t.Fatalf("tolerance 0.04: Kind = %v, want None", sig.Kind)
	}
}

func TestExplainLevelsOnly(t *testing.T) {
	out := newS(levelsOnly()).Explain(fixture(warm(awayB)))
	if !strings.Contains(out, "✗ только уровни (LevelsOnly 1)") {
		t.Fatalf("Explain misses the levels-only gate:\n%s", out)
	}
}
