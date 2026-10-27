package core

import (
	"math"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func held(p Params, bars []bar, pos strategy.Position) model.Signal {
	md := fixture(bars)
	md.Position = &pos
	return newS(p).Decide(md)
}

// pos0 filled at 99.4 on Monday 10:15: stop 99.4 − 0.5·2 = 98.4, target 101 (set on the position). Its frozen
// StopLoss (98.3, from the signal close) is deliberately different: the stop is measured from the fill.
func pos0() strategy.Position {
	return strategy.Position{PurchasePrice: 99.4, EntryATR: 2, TakeProfit: 101, StopLoss: 98.3, EntryTime: at("2026-09-14 10:15")}
}

var fillBar = bar{"2026-09-14 10:15", 99.35, 99.5, 99.3, 99.4}

func TestExitStopFromFill(t *testing.T) {
	sig := held(DefaultParams(), []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.4, 98.6}}, pos0())
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || math.Abs(sig.StopLoss-98.4) > eps {
		t.Fatalf("got %v %q SL %v, want Sell SL at 98.4", sig.Kind, sig.Reason, sig.StopLoss)
	}
}

// Review focus 5.
func TestExitStopBeatsTarget(t *testing.T) {
	sig := held(DefaultParams(), []bar{fillBar, {"2026-09-14 10:30", 99.4, 101.2, 98.3, 100}}, pos0())
	if sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL", sig.Reason)
	}
}

func TestExitTarget(t *testing.T) {
	up := bar{"2026-09-14 10:30", 99.4, 101.0, 99.3, 100.9}
	sig := held(DefaultParams(), []bar{fillBar, up}, pos0())
	if sig.Kind != model.SignalSell || sig.Reason != "TP" || math.Abs(sig.TakeProfit-101) > eps {
		t.Fatalf("got %v %q TP %v, want Sell TP at 101", sig.Kind, sig.Reason, sig.TakeProfit)
	}
	noTarget := pos0()
	noTarget.TakeProfit = 0
	if sig := held(DefaultParams(), []bar{fillBar, up}, noTarget); sig.Kind != model.SignalNone {
		t.Fatalf("no target: Kind = %v, want None", sig.Kind)
	}
}

func TestExitHoldsInsideRange(t *testing.T) {
	if sig := held(DefaultParams(), []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.6, 99.3, 99.5}}, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
}

// The position carries overnight: no exit at 23:45 nor on the next date, only the stop or the target.
func TestExitCarriesOvernight(t *testing.T) {
	bars := append([]bar{fillBar}, flat("2026-09-14 10:30", "2026-09-14 23:45", 99.5)...)
	bars = append(bars, flat("2026-09-15 07:00", "2026-09-15 07:30", 99.5)...)
	for k := range bars[1:] {
		if sig := held(DefaultParams(), bars[:k+2], pos0()); sig.Kind != model.SignalNone {
			t.Fatalf("%s: Kind = %v %q, want None", bars[k+1].t, sig.Kind, sig.Reason)
		}
	}
	bars = append(bars, bar{"2026-09-15 07:45", 99.5, 101.1, 99.5, 101})
	if sig := held(DefaultParams(), bars, pos0()); sig.Reason != "TP" {
		t.Fatalf("next day: Reason = %q, want TP", sig.Reason)
	}
}

// pos0 risks 1 (fill 99.4, stop 98.4): a close at 100.4 (+1R) arms the lock (the engine's highest
// close through the previous bar), and from the next bar on the stop is 99.65 (+0.25R).
func TestExitLockAfterR(t *testing.T) {
	p := DefaultParams()
	p.LockAfterR, p.LockAtR = 1, 0.25
	dip := []bar{fillBar, {"2026-09-14 10:45", 100, 100.1, 99.6, 99.8}}
	armed := pos0()
	armed.PrevMaxFavorablePrice = 100.4
	sig := held(p, dip, armed)
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" || math.Abs(sig.StopLoss-99.65) > eps {
		t.Fatalf("got %v %q SL %v, want Sell TRAIL at 99.65", sig.Kind, sig.Reason, sig.StopLoss)
	}
	// A high at +1R with the close short of it does not arm.
	wick := pos0()
	wick.PrevMaxFavorablePrice = 100.3
	if sig := held(p, []bar{fillBar, {"2026-09-14 10:30", 99.4, 100.6, 99.3, 100.3}, dip[1]}, wick); sig.Kind != model.SignalNone {
		t.Fatalf("wick only: Kind = %v %q, want None", sig.Kind, sig.Reason)
	}
	p.LockAfterR = 0
	if sig := held(p, dip, armed); sig.Kind != model.SignalNone {
		t.Fatalf("off: Kind = %v %q, want None", sig.Kind, sig.Reason)
	}
}
