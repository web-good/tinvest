package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func TestIsEngulf(t *testing.T) {
	red := bar{"2026-09-14 10:00", 98.7, 98.8, 98.6, 98.6}
	for name, tc := range map[string]struct {
		b    bar
		want bool
	}{
		"low below, close above red high":      {bar{"2026-09-14 10:15", 98.6, 99.2, 98.5, 99.2}, true},
		"close above red open, below red high": {bar{"2026-09-14 10:15", 98.6, 98.75, 98.5, 98.75}, false},
		"low not below red low":                {bar{"2026-09-14 10:15", 98.6, 99.2, 98.6, 99.2}, false},
		"red bar":                              {bar{"2026-09-14 10:15", 99.2, 99.2, 98.5, 98.9}, false},
	} {
		if got := isEngulf(seriesOf([]bar{red, tc.b}), 1); got != tc.want {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
	green := bar{"2026-09-14 10:00", 98.6, 98.8, 98.6, 98.7}
	if isEngulf(seriesOf([]bar{green, {"2026-09-14 10:15", 98.6, 99.2, 98.5, 99.2}}), 1) {
		t.Error("previous bar green: engulf, want none")
	}
	if isEngulf(seriesOf([]bar{{"2026-09-11 23:45", 98.7, 98.8, 98.6, 98.6}, {"2026-09-14 07:00", 98.6, 99.2, 98.5, 99.2}}), 1) {
		t.Error("red bar on the previous date: engulf, want none")
	}
}

func TestIsHammer(t *testing.T) {
	for name, tc := range map[string]struct {
		b    bar
		want bool
	}{
		"green, small body on top, long tail": {bar{"2026-09-14 10:00", 99.0, 99.2, 98.0, 99.1}, true},
		"red hammer":                          {bar{"2026-09-14 10:00", 99.1, 99.2, 98.0, 99.0}, true},
		"body over a third of the range":      {bar{"2026-09-14 10:00", 98.6, 99.2, 98.0, 99.1}, false},
		"body in the middle":                  {bar{"2026-09-14 10:00", 98.6, 99.2, 98.0, 98.7}, false},
		"flat bar":                            {bar{"2026-09-14 10:00", 99, 99, 99, 99}, false},
	} {
		if got := isHammer(seriesOf([]bar{tc.b}), 0); got != tc.want {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
}

// hammerDay is pdlDay with the 10:00 red close below PDL drawn as a hammer (tail to 98.0).
func hammerDay() []bar {
	d := upTo(reclaimAt)
	d[2] = bar{"2026-09-14 10:00", 98.7, 98.75, 98.0, 98.6}
	return d
}

// engulfDay is pdlDay with the reclaim bar engulfing the red 10:00 bar: its low 98.5 under 98.6,
// its close 99.2 over the red high 98.7.
func engulfDay() []bar {
	d := upTo(reclaimAt)
	d[3] = bar{"2026-09-14 10:15", 98.6, 99.2, 98.5, 99.2}
	return d
}

func TestReclaimPatternsOffBuyPlain(t *testing.T) {
	if sig := decide(DefaultParams(), upTo(reclaimAt)); sig.Kind != model.SignalBuy {
		t.Fatalf("patterns off: Kind = %v, want Buy", sig.Kind)
	}
}

func TestReclaimNeedsEnabledPattern(t *testing.T) {
	for name, tc := range map[string]struct {
		engulf, hammer int
		bars           []bar
		want           string // "" — no buy, else the pattern fragment of the reason
	}{
		"engulf on, plain day":  {1, 0, upTo(reclaimAt), ""},
		"hammer on, plain day":  {0, 1, upTo(reclaimAt), ""},
		"hammer on, hammer day": {0, 1, hammerDay(), "покупка сразу, молот 10:00"},
		"engulf on, hammer day": {1, 0, hammerDay(), ""},
		"engulf on, engulf day": {1, 0, engulfDay(), "покупка сразу, поглощение 10:15"},
		"hammer on, engulf day": {0, 1, engulfDay(), ""},
		"both on, hammer day":   {1, 1, hammerDay(), "покупка сразу, молот 10:00"},
		"both on, engulf day":   {1, 1, engulfDay(), "покупка сразу, поглощение 10:15"},
		"both off, hammer day":  {0, 0, hammerDay(), "покупка сразу, PDL"},
	} {
		p := DefaultParams()
		p.PDLEngulf, p.PDLHammer = tc.engulf, tc.hammer
		sig := decide(p, tc.bars)
		if tc.want == "" {
			if sig.Kind != model.SignalNone {
				t.Errorf("%s: Kind = %v, want None", name, sig.Kind)
			}
			continue
		}
		if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, tc.want) {
			t.Errorf("%s: Kind = %v, reason %q, want Buy with %q", name, sig.Kind, sig.EntryReason, tc.want)
		}
	}
}

// A hammer before the first close below PDL (09:30 closed above it) is outside the window.
func TestReclaimPatternBeforeBreakIgnored(t *testing.T) {
	d := upTo(reclaimAt)
	d[0] = bar{"2026-09-14 09:30", 99.9, 100.0, 99.3, 99.95}
	d[1].o, d[1].h = 99.95, 99.95
	p := DefaultParams()
	p.PDLHammer = 1
	if sig := decide(p, d); sig.Kind != model.SignalNone {
		t.Fatalf("hammer above PDL: Kind = %v, want None", sig.Kind)
	}
}

func TestExplainReclaimPattern(t *testing.T) {
	p := DefaultParams()
	p.PDLHammer = 1
	out := newS(p).Explain(fixture(warm(hammerDay())))
	for _, want := range []string{"✓ свечной паттерн между пробоем PDL и возвратом (поглощение 0, молот 1): молот 10:00", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	out = newS(p).Explain(fixture(warm(upTo(reclaimAt))))
	for _, want := range []string{"✗ свечной паттерн между пробоем PDL и возвратом (поглощение 0, молот 1): нет", "ИТОГ: входа нет"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
