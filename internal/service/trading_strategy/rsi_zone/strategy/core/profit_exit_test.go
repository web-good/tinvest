package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/pkg/indicators"
)

// profitCloses is trendCloses(3) (entry on stuckEntryIdx) followed by one close per step, each
// `step` percent above the previous one.
func profitCloses(bars int, step float64) []float64 {
	out := trendCloses(3)
	p := out[len(out)-1]
	for i := 0; i < bars; i++ {
		p *= 1 + step/100
		out = append(out, p)
	}
	return out
}

// profitParams arms the lock and parks the upper RSI band out of reach, so the RSI exit never
// steals the bar the test is about.
func profitParams(bars int, pct float64) Params {
	p := DefaultParams()
	p.RSIUpper = 100
	p.ProfitExitBars = bars
	p.ProfitExitPct = pct
	return p
}

func TestProfitExitFiresOnTheFirstBarPastN(t *testing.T) {
	for _, n := range []int{0, 1, 3, 5} {
		md := stuckMarket(profitCloses(n+1, 0.2))
		sig := NewWithParams("TEST", profitParams(n, 0.15)).Decide(md)
		if sig.Kind != model.SignalSell || sig.Reason != "PROFIT" {
			t.Fatalf("N=%d: Kind/Reason = %v/%q, want SignalSell/PROFIT", n, sig.Kind, sig.Reason)
		}
		if model.IsStopReason(sig.Reason) {
			t.Error("PROFIT fills at the bar close, it must not be a stop reason")
		}
		if sig.ExitReason == "" {
			t.Error("ExitReason must explain the exit")
		}
	}
}

// "More than N bars": on the N-th bar after the entry the lock is not armed yet, whatever the gain.
func TestProfitExitWaitsMoreThanNBars(t *testing.T) {
	md := stuckMarket(profitCloses(3, 0.5)) // +1.5% on the third bar after the entry
	if sig := NewWithParams("TEST", profitParams(3, 0.5)).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone: only three bars after the entry", sig.Kind, sig.Reason)
	}
}

func TestProfitExitNeedsTheThreshold(t *testing.T) {
	md := stuckMarket(profitCloses(4, 0.1)) // ~+0.40% on the fourth bar
	if sig := NewWithParams("TEST", profitParams(3, 0.5)).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone: gain below 0.5%%", sig.Kind, sig.Reason)
	}
	md = stuckMarket(profitCloses(5, 0.1)) // ~+0.50% on the fifth bar
	if sig := NewWithParams("TEST", profitParams(3, 0.5)).Decide(md); sig.Reason != "PROFIT" {
		t.Fatalf("Kind/Reason = %v/%q, want SignalSell/PROFIT once the close reaches +0.5%%", sig.Kind, sig.Reason)
	}
}

// A loss past N bars never triggers the lock.
func TestProfitExitSilentOnALoss(t *testing.T) {
	md := stuckMarket(stuckCloses(6))
	p := profitParams(3, 0.5)
	if sig := NewWithParams("TEST", p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone", sig.Kind, sig.Reason)
	}
}

func TestProfitExitOffByDefault(t *testing.T) {
	md := stuckMarket(profitCloses(6, 0.5))
	p := DefaultParams()
	p.RSIUpper = 100
	if sig := NewWithParams("TEST", p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone: ProfitExitPct 0 disables the exit", sig.Kind, sig.Reason)
	}
}

func TestProfitExitDegradesWithoutAnAnchor(t *testing.T) {
	t.Run("no EntryTime", func(t *testing.T) {
		md := stuckMarket(profitCloses(5, 0.5))
		md.Position.EntryTime = time.Time{}
		if sig := NewWithParams("TEST", profitParams(3, 0.5)).Decide(md); sig.Kind != model.SignalNone {
			t.Fatalf("Kind/Reason = %v/%q, want SignalNone", sig.Kind, sig.Reason)
		}
	})
	t.Run("entry bar left the window", func(t *testing.T) {
		md := stuckMarket(profitCloses(5, 0.5))
		md.Position.EntryTime = md.Times[0].Add(-time.Hour)
		if sig := NewWithParams("TEST", profitParams(3, 0.5)).Decide(md); sig.Kind != model.SignalNone {
			t.Fatalf("Kind/Reason = %v/%q, want SignalNone", sig.Kind, sig.Reason)
		}
	})
}

// Live stamps EntryTime between bar boundaries; the count still starts at the bar containing it.
func TestProfitExitAnchorsOnTheBarContainingEntryTime(t *testing.T) {
	md := stuckMarket(profitCloses(4, 0.5))
	md.Position.EntryTime = md.Times[stuckEntryIdx].Add(10 * time.Minute)
	if sig := NewWithParams("TEST", profitParams(3, 0.5)).Decide(md); sig.Reason != "PROFIT" {
		t.Fatalf("Kind/Reason = %v/%q, want SignalSell/PROFIT", sig.Kind, sig.Reason)
	}
}

func TestStopWinsOverProfitExitOnTheSameBar(t *testing.T) {
	md := stuckMarket(profitCloses(4, 0.5))
	md.Lows[len(md.Lows)-1] = md.Position.PurchasePrice - stuckATR
	if sig := NewWithParams("TEST", profitParams(3, 0.5)).Decide(md); sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL: the stop wins a same-bar tie", sig.Reason)
	}
}

// With the upper band crossed on the same bar both exits fire on one close; the RSI exit keeps
// its name. The band is placed between the last two RSI readings so the cross lands on that bar.
func TestRSIExitWinsOverProfitExitOnTheSameBar(t *testing.T) {
	closes := profitCloses(4, 0.5)
	md := stuckMarket(closes)
	rsi := indicators.RSISeries(closes, 4)
	i := len(closes) - 1
	if rsi[i] <= rsi[i-1] {
		t.Fatalf("fixture: RSI(4) must rise on the last bar (%.2f -> %.2f)", rsi[i-1], rsi[i])
	}
	p := profitParams(3, 0.5)
	p.RSIUpper = (rsi[i-1] + rsi[i]) / 2
	if sig := NewWithParams("TEST", p).Decide(md); sig.Reason != "RSI" {
		t.Fatalf("Reason = %q, want RSI", sig.Reason)
	}
}
