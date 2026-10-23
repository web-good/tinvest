package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func hammerP() Params {
	p := DefaultParams()
	p.PDLHammer = 1
	return p
}

// slowHammerDay: 09:45 breaks PDL 99, 10:00 is a red hammer (low 98.0), 10:15 is red under PDL,
// 10:30 reclaims PDL (close 99.2) two bars after the hammer.
func slowHammerDay() []bar {
	return []bar{
		{"2026-09-14 09:30", 100, 100, 99.5, 99.5},
		{"2026-09-14 09:45", 99.5, 99.5, 98.7, 98.7},
		{"2026-09-14 10:00", 98.7, 98.75, 98.0, 98.6},
		{"2026-09-14 10:15", 98.6, 98.6, 98.4, 98.5},
		{"2026-09-14 10:30", 98.5, 99.2, 98.5, 99.2},
	}
}

// The hammer 10:00 (low 98.0) puts the stop under it; the reclaim close 99.2 risks 1.2, the 2R
// target is 101.6.
func TestHammerReclaimStopAndTarget(t *testing.T) {
	sig := decide(hammerP(), hammerDay())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.StopLoss-98.0) > eps || math.Abs(sig.TakeProfit-101.6) > eps {
		t.Fatalf("stop %.4f target %.4f, want 98.0 101.6", sig.StopLoss, sig.TakeProfit)
	}
	for _, want := range []string{"возврат над PDL на свече 10:15, покупка сразу, молот 10:00", "стоп по лою молота 10:00 98.0000, цель 2.00R"} {
		if !strings.Contains(sig.EntryReason, want) {
			t.Errorf("reason %q, want %q", sig.EntryReason, want)
		}
	}
}

func TestHammerReclaimTargetR(t *testing.T) {
	p := hammerP()
	p.PDLPatternTargetR = 3
	if sig := decide(p, hammerDay()); math.Abs(sig.TakeProfit-102.8) > eps {
		t.Fatalf("R 3: target %.4f, want 102.8", sig.TakeProfit)
	}
	// R 0: the middle of yesterday's range, the stop stays under the hammer
	p.PDLPatternTargetR = 0
	if sig := decide(p, hammerDay()); math.Abs(sig.TakeProfit-100) > eps || math.Abs(sig.StopLoss-98.0) > eps {
		t.Fatalf("R 0: stop %.4f target %.4f, want 98.0 100", sig.StopLoss, sig.TakeProfit)
	}
	if got := DefaultParams().PDLPatternTargetR; got != 2 {
		t.Fatalf("default PDLPatternTargetR = %v, want 2", got)
	}
}

// The reclaim two bars after the hammer: PDLReclaimBars 1 burns the setup, 2 and 0 (no limit) buy.
func TestHammerReclaimBars(t *testing.T) {
	for limit, want := range map[int]model.SignalKind{0: model.SignalBuy, 1: model.SignalNone, 2: model.SignalBuy} {
		p := hammerP()
		p.PDLReclaimBars = limit
		if sig := decide(p, slowHammerDay()); sig.Kind != want {
			t.Errorf("PDLReclaimBars %d: Kind = %v, want %v", limit, sig.Kind, want)
		}
	}
}

// A green hammer closing back above PDL is the reclaim itself: zero bars of waiting, stop at its low.
func TestHammerIsReclaimBar(t *testing.T) {
	d := []bar{
		{"2026-09-14 09:30", 100, 100, 99.5, 99.5},
		{"2026-09-14 09:45", 99.5, 99.5, 98.7, 98.7},
		{"2026-09-14 10:00", 99.05, 99.15, 98.0, 99.1},
	}
	p := hammerP()
	p.PDLReclaimBars = 1
	sig := decide(p, d)
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-98.0) > eps || math.Abs(sig.TakeProfit-101.3) > eps {
		t.Fatalf("got %v stop %.4f target %.4f, want Buy 98.0 101.3", sig.Kind, sig.StopLoss, sig.TakeProfit)
	}
}

// With two hammers under PDL the latest one carries the stop.
func TestHammerLatestCarriesStop(t *testing.T) {
	d := slowHammerDay()
	d[3] = bar{"2026-09-14 10:15", 98.6, 98.65, 98.2, 98.6}
	if sig := decide(hammerP(), d); sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-98.2) > eps {
		t.Fatalf("got %v stop %.4f, want Buy 98.2", sig.Kind, sig.StopLoss)
	}
}

// The engulfing keeps the reclaim's ATR stop from the close and the middle-of-range target.
func TestEngulfReclaimKeepsATRStop(t *testing.T) {
	p := DefaultParams()
	p.PDLEngulf = 1
	sig := decide(p, engulfDay())
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-98.2) > eps || math.Abs(sig.TakeProfit-100) > eps {
		t.Fatalf("got %v stop %.4f target %.4f, want Buy 98.2 100", sig.Kind, sig.StopLoss, sig.TakeProfit)
	}
}

// A hammer-reclaim position stops at the stop it carries (98.3 on pos0), not at the fill −
// StopDailyATR × ATR (99.4 − 0.5·2 = 98.4).
func TestHammerExitAtCarriedStop(t *testing.T) {
	if sig := held(hammerP(), []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.35, 98.6}}, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("low 98.35: got %v %q, want None", sig.Kind, sig.Reason)
	}
	sig := held(hammerP(), []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.3, 98.6}}, pos0())
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-98.3) > eps {
		t.Fatalf("low 98.3: got %q at %v, want SL at 98.3", sig.Reason, sig.StopLoss)
	}
}

func TestExplainHammerReclaim(t *testing.T) {
	p := hammerP()
	p.PDLReclaimBars = 1
	out := newS(p).Explain(fixture(warm(hammerDay())))
	for _, want := range []string{"✓ свечной паттерн между пробоем PDL и возвратом (поглощение 0, молот 1 не дальше 1 св.): молот 10:00",
		"✓ стоп по лою молота 98.0000 ниже close 99.2000, цель 2.00R = 101.6000", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	out = newS(p).Explain(fixture(warm(slowHammerDay())))
	if !strings.Contains(out, "✗ свечной паттерн между пробоем PDL и возвратом (поглощение 0, молот 1 не дальше 1 св.): нет") {
		t.Errorf("missing the burnt setup in:\n%s", out)
	}
}

// The hammer stop 98.0 is 1.2 = 0.6 ATR under the close 99.2: a cap of 0.5 ATR lifts it to 98.2
// and the 2R target to 101.2; a cap of 0.7 ATR leaves it.
func TestHammerMaxStop(t *testing.T) {
	p := hammerP()
	p.PDLMaxStopATR = 0.5
	sig := decide(p, hammerDay())
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-98.2) > eps || math.Abs(sig.TakeProfit-101.2) > eps {
		t.Fatalf("cap 0.5: got %v stop %.4f target %.4f, want Buy 98.2 101.2", sig.Kind, sig.StopLoss, sig.TakeProfit)
	}
	if !strings.Contains(sig.EntryReason, "стоп по лою молота 10:00 98.0000, поднят до 0.50 ATR: 98.2000, цель 2.00R") {
		t.Fatalf("reason %q", sig.EntryReason)
	}
	out := newS(p).Explain(fixture(warm(hammerDay())))
	if !strings.Contains(out, "✓ стоп по лою молота 98.0000 поднят до 0.50 ATR: 98.2000 ниже close 99.2000, цель 2.00R = 101.2000") {
		t.Errorf("explain missing the cap:\n%s", out)
	}
	p.PDLMaxStopATR = 0.7
	if sig := decide(p, hammerDay()); math.Abs(sig.StopLoss-98.0) > eps || math.Abs(sig.TakeProfit-101.6) > eps {
		t.Fatalf("cap 0.7: stop %.4f target %.4f, want 98.0 101.6", sig.StopLoss, sig.TakeProfit)
	}
}

// The hammer stop 98.0 is 0.6 ATR under the close 99.2: a floor of 0.8 ATR lowers it to 97.6 and
// the 2R target to 102.4; a floor of 0.5 ATR leaves it.
func TestHammerMinStop(t *testing.T) {
	p := hammerP()
	p.PDLMinStopATR = 0.8
	sig := decide(p, hammerDay())
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-97.6) > eps || math.Abs(sig.TakeProfit-102.4) > eps {
		t.Fatalf("floor 0.8: got %v stop %.4f target %.4f, want Buy 97.6 102.4", sig.Kind, sig.StopLoss, sig.TakeProfit)
	}
	if !strings.Contains(sig.EntryReason, "стоп по лою молота 10:00 98.0000, опущен до 0.80 ATR: 97.6000, цель 2.00R") {
		t.Fatalf("reason %q", sig.EntryReason)
	}
	out := newS(p).Explain(fixture(warm(hammerDay())))
	if !strings.Contains(out, "✓ стоп по лою молота 98.0000 опущен до 0.80 ATR: 97.6000 ниже close 99.2000, цель 2.00R = 102.4000") {
		t.Errorf("explain missing the floor:\n%s", out)
	}
	p.PDLMinStopATR = 0.5
	if sig := decide(p, hammerDay()); math.Abs(sig.StopLoss-98.0) > eps || math.Abs(sig.TakeProfit-101.6) > eps {
		t.Fatalf("floor 0.5: stop %.4f target %.4f, want 98.0 101.6", sig.StopLoss, sig.TakeProfit)
	}
}

// PDLHammerEntry: the hammer 10:00 (low 98.0) and the green 10:15 (close 98.9, risk 0.9) buy the
// next open: stop 98.0, 2R target 100.7. The hammer bar alone does not buy.
func TestHammerEntry(t *testing.T) {
	p := hammerP()
	p.PDLHammerEntry = 1
	d := append(hammerDay()[:3], bar{"2026-09-14 10:15", 98.6, 98.95, 98.5, 98.9})
	sig := decide(p, d)
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-98.0) > eps || math.Abs(sig.TakeProfit-100.7) > eps {
		t.Fatalf("got %v stop %.4f target %.4f, want Buy 98.0 100.7", sig.Kind, sig.StopLoss, sig.TakeProfit)
	}
	if !strings.Contains(sig.EntryReason, "молот после пробоя PDL на свече 10:00, следующая свеча 10:15 зелёная") {
		t.Fatalf("reason %q", sig.EntryReason)
	}
	if sig := decide(p, d[:3]); sig.Kind != model.SignalNone {
		t.Fatalf("hammer bar: Kind = %v, want None", sig.Kind)
	}
	// A red bar after the hammer burns it; a green one later does not buy.
	red := append(hammerDay()[:3], bar{"2026-09-14 10:15", 98.6, 98.65, 98.3, 98.4}, bar{"2026-09-14 10:30", 98.4, 98.8, 98.4, 98.7})
	for n := 4; n <= 5; n++ {
		if sig := decide(p, red[:n]); sig.Kind != model.SignalNone {
			t.Fatalf("red after hammer, %d bars: Kind = %v, want None", n, sig.Kind)
		}
	}
	p.PDLHammerEntry = 0
	if sig := decide(p, d); sig.Kind != model.SignalNone {
		t.Fatalf("off: Kind = %v, want None", sig.Kind)
	}
}

// The hammer counts only after a red close below PDL with no close above PDL since.
func TestHammerEntryNeedsBreak(t *testing.T) {
	p := hammerP()
	p.PDLHammerEntry = 1
	d := append(hammerDay()[:3], bar{"2026-09-14 10:15", 98.6, 98.95, 98.5, 98.9})
	d[2] = bar{"2026-09-14 10:00", 98.55, 98.65, 98.0, 98.6}
	if sig := decide(p, d); sig.Kind != model.SignalBuy {
		t.Fatalf("after the break: Kind = %v, want Buy", sig.Kind)
	}
	d[1] = bar{"2026-09-14 09:45", 99.0, 99.2, 98.9, 99.1}
	if sig := decide(p, d); sig.Kind != model.SignalNone {
		t.Fatalf("no break: Kind = %v, want None", sig.Kind)
	}
}

func TestExplainHammerEntry(t *testing.T) {
	p := hammerP()
	p.PDLHammerEntry = 1
	d := append(hammerDay()[:3], bar{"2026-09-14 10:15", 98.6, 98.95, 98.5, 98.9})
	out := newS(p).Explain(fixture(warm(d)))
	for _, want := range []string{"✓ эта свеча зелёная, предыдущая — молот после пробоя PDL без закрытий выше PDL (вход после молота 1)",
		"✓ стоп по лою молота 98.0000 ниже close 98.9000, цель 2.00R = 100.7000", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// tuesdayHammer is tuesdayDip with the 10:00 bar a hammer (low 98.0) and the reclaim 10:15; both
// sit in the main session after Monday's warm-up of volume 100.
func tuesdayHammer() []bar {
	d := tuesdayDip()
	d[2] = bar{"2026-09-15 10:00", 98.7, 98.75, 98.0, 98.6}
	return d
}

// Each of the hammer and the reclaim must trade more than 1.5 × the normal 100.
func TestHammerSignalVolume(t *testing.T) {
	p := hammerP()
	p.PDLSignalVolMult = 1.5
	sig := newS(p).Decide(dipMD(tuesdayHammer(), 151))
	if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "объём молота и сигнала от 151 > 1.50 × обычный 100") {
		t.Fatalf("151: got %v %q", sig.Kind, sig.EntryReason)
	}
	if sig := newS(p).Decide(dipMD(tuesdayHammer(), 150)); sig.Kind != model.SignalNone {
		t.Fatalf("150: Kind = %v, want None", sig.Kind)
	}
	md := dipMD(tuesdayHammer(), 151)
	md.Volumes[len(md.Volumes)-1] = 120
	if sig := newS(p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("quiet reclaim: Kind = %v, want None", sig.Kind)
	}
	if out := newS(p).Explain(md); !strings.Contains(out, "✗ объём молота и сигнала (меньший) 120 > 1.50 × обычный 100") {
		t.Errorf("explain missing the volume:\n%s", out)
	}
	md.Volumes = nil
	if sig := newS(p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("no volumes: Kind = %v, want None", sig.Kind)
	}
}
