package core

import (
	"math"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

// TargetStopMult 2: the target is the signal close 99.2 + 2 × the stop distance 1.0 (0.5 · ATR 2).
func TestReclaimTargetStopMult(t *testing.T) {
	p := DefaultParams()
	p.TargetStopMult = 2
	if sig := decide(p, upTo(reclaimAt)); sig.Kind != model.SignalBuy || math.Abs(sig.TakeProfit-101.2) > eps {
		t.Fatalf("got %v TP %v, want Buy TP 101.2", sig.Kind, sig.TakeProfit)
	}
}

// StopAtExtremum: the stop is the dip low 98.6 − 0.1·ATR 2 = 98.4; with TargetStopMult 2 the
// target is the close 99.2 + 2 × 0.8 = 100.8.
func TestReclaimStopAtExtremum(t *testing.T) {
	p := DefaultParams()
	p.StopAtExtremum, p.StopBufferATR, p.TargetStopMult = 1, 0.1, 2
	sig := decide(p, upTo(reclaimAt))
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-98.4) > eps || math.Abs(sig.TakeProfit-100.8) > eps {
		t.Fatalf("got %v SL %v TP %v, want Buy SL 98.4 TP 100.8", sig.Kind, sig.StopLoss, sig.TakeProfit)
	}
}

// A held position stops at its frozen StopLoss (98.3 on pos0), not at the fill − ATR (98.4).
func TestExitStopAtExtremumFrozen(t *testing.T) {
	p := DefaultParams()
	p.StopAtExtremum = 1
	if sig := held(p, []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.35, 98.6}}, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("low 98.35: got %v %q, want None", sig.Kind, sig.Reason)
	}
	sig := held(p, []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.3, 98.6}}, pos0())
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-98.3) > eps {
		t.Fatalf("low 98.3: got %q at %v, want SL at 98.3", sig.Reason, sig.StopLoss)
	}
}
