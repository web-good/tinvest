package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// stuckEntryIdx is the entry bar of every stuck-exit fixture: trendCloses(3)'s last bar, where
// RSI(4) crosses below 25.
const stuckEntryIdx = 402

// stuckATR is the daily ATR frozen on the stuck fixtures' position: the stop sits 20 below the
// entry, out of reach of every falling fixture.
const stuckATR = 20.0

// stuckCloses is trendCloses(3) followed by `downBars` more bars of -0.2%: RSI(4) stays below
// 25 on every bar after the entry.
func stuckCloses(downBars int) []float64 {
	out := trendCloses(3)
	p := out[len(out)-1]
	for i := 0; i < downBars; i++ {
		p *= 0.998
		out = append(out, p)
	}
	return out
}

// redipCloses leaves the zone right after the entry (one +0.5% bar lifts RSI(4) above 25), then
// falls back in with `downBars` bars of -0.8% that keep it below 25 from the second one on.
func redipCloses(downBars int) []float64 {
	out := trendCloses(3)
	p := out[len(out)-1] * 1.005
	out = append(out, p)
	for i := 0; i < downBars; i++ {
		p *= 0.992
		out = append(out, p)
	}
	return out
}

// stuckMarket is the 30-minute fixture with a long opened on stuckEntryIdx, EntryTime stamped
// with that bar's open-time, as the backtest engine does. The frozen ATR is wide (stuckATR) so
// the falling fixtures never touch the stop unless a test asks for it.
func stuckMarket(closes []float64) strategy.MarketData {
	md := fixture(closes, mondayNoon)
	pos := openPosition(stuckATR)
	pos.EntryTime = md.Times[stuckEntryIdx]
	md.Position = pos
	return md
}

func stuckParams(bars int) Params {
	p := DefaultParams()
	p.StuckExitBars = bars
	return p
}

func TestStuckFixturesHaveTheirIntendedShape(t *testing.T) {
	below := func(name string, closes []float64, from int) {
		t.Helper()
		rsi := indicators.RSISeries(closes, 4)
		for b := from; b < len(closes); b++ {
			if rsi[b] >= 25 {
				t.Errorf("%s: RSI(4)[%d] = %.2f, want < 25", name, b, rsi[b])
			}
		}
	}
	below("stuckCloses(6)", stuckCloses(6), stuckEntryIdx)
	below("redipCloses(6)", redipCloses(6), stuckEntryIdx+3)
	if rsi := indicators.RSISeries(redipCloses(6), 4); rsi[stuckEntryIdx+1] < 25 {
		t.Errorf("redipCloses: RSI(4) after the entry = %.2f, want >= 25 (the bar that leaves the zone)", rsi[stuckEntryIdx+1])
	}
}

func TestStuckExitFiresOnTheNthBarAfterEntry(t *testing.T) {
	for _, n := range []int{2, 3, 4} {
		md := stuckMarket(stuckCloses(n))
		sig := NewWithParams("TEST", stuckParams(n)).Decide(md)
		if sig.Kind != model.SignalSell || sig.Reason != "STUCK" {
			t.Fatalf("N=%d: Kind/Reason = %v/%q, want SignalSell/STUCK", n, sig.Kind, sig.Reason)
		}
		if model.IsStopReason(sig.Reason) {
			t.Errorf("STUCK fills at the bar close, it must not be a stop reason")
		}
		if sig.ExitReason == "" {
			t.Error("ExitReason must explain the exit")
		}
	}
}

// The entry bar itself is not counted: with N=3 and only two bars after the entry the position
// stays open.
func TestStuckExitDoesNotCountTheEntryBar(t *testing.T) {
	md := stuckMarket(stuckCloses(2))
	if sig := NewWithParams("TEST", stuckParams(3)).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone: only two bars after the entry", sig.Kind, sig.Reason)
	}
}

// Once RSI left the zone after the entry the filter is off for the trade: a later re-dip that
// sits in the zone for N bars does not re-arm it.
func TestStuckExitDisarmedAfterRSILeavesTheZone(t *testing.T) {
	md := stuckMarket(redipCloses(6))
	if sig := NewWithParams("TEST", stuckParams(3)).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone: RSI rose above the band after the entry", sig.Kind, sig.Reason)
	}
}

func TestStuckExitOffByDefault(t *testing.T) {
	md := stuckMarket(stuckCloses(6))
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone: StuckExitBars 0 disables the exit", sig.Kind, sig.Reason)
	}
}

// Without an entry anchor the exit cannot tell bars after the entry from bars before it, so it
// stays silent rather than guess.
func TestStuckExitDegradesWithoutEntryTimeOrTimes(t *testing.T) {
	t.Run("no EntryTime", func(t *testing.T) {
		md := stuckMarket(stuckCloses(4))
		md.Position.EntryTime = time.Time{}
		if sig := NewWithParams("TEST", stuckParams(3)).Decide(md); sig.Kind != model.SignalNone {
			t.Fatalf("Kind/Reason = %v/%q, want SignalNone", sig.Kind, sig.Reason)
		}
	})
	t.Run("no Times", func(t *testing.T) {
		md := stuckMarket(stuckCloses(4))
		md.Times = nil
		if sig := NewWithParams("TEST", stuckParams(3)).Decide(md); sig.Kind != model.SignalNone {
			t.Fatalf("Kind/Reason = %v/%q, want SignalNone", sig.Kind, sig.Reason)
		}
	})
}

// A position older than the candle window: the window (>= 120 bars) is far longer than any
// StuckExitBars, so the exit either fired long ago or was disarmed by a rise the window may no
// longer show. It stays silent rather than act on a truncated history.
func TestStuckExitSilentWhenTheEntryBarLeftTheWindow(t *testing.T) {
	md := stuckMarket(stuckCloses(4))
	md.Position.EntryTime = md.Times[0].Add(-time.Hour)
	if sig := NewWithParams("TEST", stuckParams(3)).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone", sig.Kind, sig.Reason)
	}
}

// Live stamps EntryTime between bar boundaries; the entry bar is the last bar that opened at or
// before it.
func TestStuckExitAnchorsOnTheBarContainingEntryTime(t *testing.T) {
	md := stuckMarket(stuckCloses(3))
	md.Position.EntryTime = md.Times[stuckEntryIdx].Add(10 * time.Minute)
	if sig := NewWithParams("TEST", stuckParams(3)).Decide(md); sig.Reason != "STUCK" {
		t.Fatalf("Kind/Reason = %v/%q, want SignalSell/STUCK", sig.Kind, sig.Reason)
	}
}

func TestStopWinsOverStuckExitOnTheSameBar(t *testing.T) {
	md := stuckMarket(stuckCloses(3))
	md.Lows[len(md.Lows)-1] = md.Position.PurchasePrice - stuckATR
	if sig := NewWithParams("TEST", stuckParams(3)).Decide(md); sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL: the stop wins a same-bar tie", sig.Reason)
	}
}

// With the stochastic gate an entry can fire while RSI has already been in its zone for a while
// (Stoch crossed, RSI was seen in the window). The stuck count starts at the entry bar, not at the
// start of RSI's run in the zone: here RSI sits below 25 from bar 402, the entry is on bar 404 and
// only two bars follow it, so N=3 must not fire yet.
func TestStuckExitCountsFromTheEntryBarNotFromTheZoneRun(t *testing.T) {
	md := stuckMarket(stuckCloses(4)) // RSI < 25 on bars 402..406
	md.Position.EntryTime = md.Times[stuckEntryIdx+2]
	if sig := NewWithParams("TEST", stuckParams(3)).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone: two bars after the entry, not four", sig.Kind, sig.Reason)
	}
	md = stuckMarket(stuckCloses(5))
	md.Position.EntryTime = md.Times[stuckEntryIdx+2]
	if sig := NewWithParams("TEST", stuckParams(3)).Decide(md); sig.Reason != "STUCK" {
		t.Fatalf("Kind/Reason = %v/%q, want SignalSell/STUCK on the third bar after the entry", sig.Kind, sig.Reason)
	}
}
