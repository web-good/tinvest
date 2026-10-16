package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// PDH 101, daily ATR 2: preC closes under PDH, barC is green and closes 0.4 above it with a 0.8 body.
var (
	preC = bar{"2026-09-14 09:55", 100.4, 100.6, 100.3, 100.5}
	barC = bar{"2026-09-14 10:00", 100.6, 101.5, 100.5, 101.4}
)

func withC(p Params) Params {
	p.EntryC = 1
	return p
}

func TestEntryC(t *testing.T) {
	sig := decide(withC(m5()), []bar{preC, barC})
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "C 5m") {
		t.Fatalf("got %v %q, want Buy C 5m", sig.Kind, sig.EntryReason)
	}
	if sig.TakeProfit != 0 || math.Abs(sig.StopLoss-100.4) > eps || math.Abs(sig.Level-101) > eps {
		t.Fatalf("TP/SL/Level = %v/%v/%v, want 0/100.4/101", sig.TakeProfit, sig.StopLoss, sig.Level)
	}
	if sig := decide(m5(), []bar{preC, barC}); sig.Kind != model.SignalNone {
		t.Fatalf("EntryC 0: Kind = %v, want None", sig.Kind)
	}
}

func TestEntryCNeedsEveryCondition(t *testing.T) {
	for name, bars := range map[string][]bar{
		"previous closed above PDH": {{"2026-09-14 09:55", 101, 101.2, 100.9, 101.1}, barC},
		"red bar":                   {preC, {"2026-09-14 10:00", 101.5, 101.6, 101.2, 101.3}},
		"close at PDH":              {preC, {"2026-09-14 10:00", 100.6, 101.2, 100.5, 101}},
	} {
		if sig := decide(withC(m5()), bars); sig.Kind != model.SignalNone {
			t.Errorf("%s: Kind = %v, want None", name, sig.Kind)
		}
	}
	// the previous bar must be of the same date: a first bar of the day has none
	first := []bar{{"2026-09-11 23:45", 100.5, 100.5, 100.5, 100.5}, {"2026-09-14 07:00", 100.6, 101.5, 100.5, 101.4}}
	if sig := decide(withC(m5()), first); sig.Kind != model.SignalNone {
		t.Errorf("previous bar of another date: Kind = %v, want None", sig.Kind)
	}
}

// A breakout runs with RSI high by definition: the entry RSI gate does not apply to C.
func TestEntryCSkipsRSIGate(t *testing.T) {
	p := withC(m5())
	p.RSIEntryMax = 0.001
	if sig := decide(p, []bar{preC, barC}); sig.Kind != model.SignalBuy {
		t.Fatalf("RSIEntryMax tiny: Kind = %v, want Buy (no RSI gate on C)", sig.Kind)
	}
}

// A bar spanning PDL and PDH is C, not an A blocked by the target.
func TestEntryCWinsOverA(t *testing.T) {
	wide := bar{"2026-09-14 10:00", 99.5, 101.5, 98.8, 101.4}
	if sig := decide(withC(m5()), []bar{preC, wide}); sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "C") {
		t.Fatalf("got %v %q, want Buy C", sig.Kind, sig.EntryReason)
	}
}

func TestEntryC30m(t *testing.T) {
	bars := append(flat("2026-09-14 00:00", "2026-09-14 09:55", 100.5), half("2026-09-14 10:00", 100.6, 101.5, 100.5, 101.4)...)
	p := withC(DefaultParams())
	sig := newS(p).Decide(fixture(bars))
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "C 30m") {
		t.Fatalf("got %v %q, want Buy C 30m", sig.Kind, sig.EntryReason)
	}
}

func TestConfirmC(t *testing.T) {
	p := withC(m5())
	p.Confirm = 1
	pattern := bar{"2026-09-14 09:55", 100.6, 101.5, 100.5, 101.4}
	pre := bar{"2026-09-14 09:50", 100.4, 100.6, 100.3, 100.5}
	if sig := decide(p, []bar{pre, pattern, {"2026-09-14 10:00", 101.4, 101.6, 101.2, 101.5}}); sig.Kind != model.SignalBuy {
		t.Fatalf("confirming bar above PDH: Kind = %v, want Buy", sig.Kind)
	}
	if sig := decide(p, []bar{pre, pattern, {"2026-09-14 10:00", 101.4, 101.4, 100.7, 100.8}}); sig.Kind != model.SignalNone {
		t.Fatalf("confirming bar under PDH: Kind = %v, want None", sig.Kind)
	}
}

// BodyATR 0.1 with ATR 2: body and the close's distance past the level both strictly above 0.2.
func TestBodyATR(t *testing.T) {
	p := m5()
	p.BodyATR = 0.1
	if sig := decide(p, []bar{pre, barA}); sig.Kind != model.SignalNone {
		t.Fatalf("A body 0.1: Kind = %v, want None", sig.Kind)
	}
	strongA := bar{"2026-09-14 10:00", 98.95, 99.4, 98.8, 99.3} // body 0.35, 0.3 past PDL
	if sig := decide(p, []bar{pre, strongA}); sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "тело 0.35") {
		t.Fatalf("strong A: %v %q, want Buy with the body", sig.Kind, sig.EntryReason)
	}
	shallowA := bar{"2026-09-14 10:00", 98.9, 99.2, 98.8, 99.15} // body 0.25, 0.15 past PDL
	if sig := decide(p, []bar{pre, shallowA}); sig.Kind != model.SignalNone {
		t.Fatalf("A 0.15 past PDL: Kind = %v, want None", sig.Kind)
	}
	if sig := decide(p, []bar{redB, greenB}); sig.Kind != model.SignalNone {
		t.Fatalf("B closed under PDL: Kind = %v, want None", sig.Kind)
	}
	p = withC(p)
	if sig := decide(p, []bar{preC, barC}); sig.Kind != model.SignalBuy {
		t.Fatalf("strong C: Kind = %v, want Buy", sig.Kind)
	}
	if sig := decide(p, []bar{preC, {"2026-09-14 10:00", 100.6, 101.2, 100.5, 101.1}}); sig.Kind != model.SignalNone {
		t.Fatalf("C 0.1 past PDH: Kind = %v, want None", sig.Kind)
	}
}

// A breakout position (bought above PDH) has no RSI exit; MACD, stop and EOD stay.
func TestBreakoutExitSkipsRSI(t *testing.T) {
	p := withC(m5())
	p.MACDFast = 0
	bars := append([]bar{{"2026-09-14 09:30", 101.35, 101.5, 101.3, 101.4}}, zigzag("2026-09-14 09:35", 101.4, 10)...)
	pos := strategy.Position{PurchasePrice: 101.4, EntryATR: 2, EntryTime: at("2026-09-14 09:30")}
	if sig := held(p, bars, pos); sig.Kind != model.SignalNone {
		t.Fatalf("breakout position, RSI high: %v %q, want hold", sig.Kind, sig.Reason)
	}
	p.EntryC = 0
	if sig := held(p, bars, pos); sig.Reason != "RSI" {
		t.Fatalf("EntryC 0: Reason = %q, want RSI", sig.Reason)
	}
}

func TestExplainC(t *testing.T) {
	p := withC(m5())
	p.BodyATR = 0.1
	out := newS(p).Explain(fixture(warm([]bar{preC, barC})))
	for _, want := range []string{"✓ C (включён 1)", "✓ сила свечи: тело 0.80 > 0.20, отрыв от PDH 0.40 > 0.20", "RSI-фильтр входа: не для C"} {
		if !strings.Contains(out, want) {
			t.Errorf("Explain misses %q:\n%s", want, out)
		}
	}
}
