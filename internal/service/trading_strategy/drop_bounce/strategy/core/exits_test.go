package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/pkg/indicators"
)

func TestProfitOnlyOnFillBar(t *testing.T) {
	p := exitsOff()
	p.ProfitPct = 1
	bars := flatWithLast(100, 101, 100, 101)
	fill := bars[len(bars)-1].t
	if sig := held(p, bars, 100, fill); sig.Kind != model.SignalSell || sig.Reason != "PROFIT" {
		t.Fatalf("fill bar +1%%: got %v %q, want PROFIT", sig.Kind, sig.Reason)
	}
	bars = flatWithLast(100, 100.99, 100, 100.99)
	if sig := held(p, bars, 100, bars[len(bars)-1].t); sig.Kind != model.SignalNone {
		t.Fatalf("fill bar +0.99%%: kind = %v, want None", sig.Kind)
	}
	bars = flatWithLast(100, 101, 100, 101)
	if sig := held(p, bars, 100, bars[len(bars)-2].t); sig.Kind != model.SignalNone {
		t.Fatalf("second bar +1%%: kind = %v, want None (only the fill bar counts)", sig.Kind)
	}
}

func TestStopBeatsProfit(t *testing.T) {
	p := exitsOff()
	p.ProfitPct = 1
	bars := flatWithLast(100, 101.5, 98.9, 101.2)
	if sig := held(p, bars, 100, bars[len(bars)-1].t); sig.Reason != "SL" {
		t.Fatalf("reason = %q, want SL first", sig.Reason)
	}
}

func TestCrossUp(t *testing.T) {
	if !crossUp(69.9, 70, 70) || !crossUp(10, 90, 70) {
		t.Fatal("want cross")
	}
	if crossUp(70, 75, 70) || crossUp(60, 69.9, 70) {
		t.Fatal("want no cross")
	}
}

// firstSell walks the path bar by bar holding from bar entryIdx and returns the index and signal
// of the first sell, or −1.
func firstSell(p Params, bars []bar, entryIdx int) (int, model.Signal) {
	for k := entryIdx; k < len(bars); k++ {
		if sig := held(p, bars[:k+1], bars[entryIdx].o, bars[entryIdx].t); sig.Kind == model.SignalSell {
			return k, sig
		}
	}
	return -1, model.Signal{}
}

func TestDecideRSICross(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	bars := path("2026-09-14 08:00", closes)
	p := exitsOff()
	p.StopDailyATR, p.RSIExit = 0, 70
	k, sig := firstSell(p, bars, 36)
	if k < 0 || sig.Reason != "RSI" {
		t.Fatalf("first sell at %d %q, want RSI", k, sig.Reason)
	}
	r := indicators.RSISeries(closes[:k+1], p.RSIPeriod)
	if !(r[k-1] < 70 && r[k] >= 70) {
		t.Fatalf("exit at %d: RSI %v → %v, want a cross of 70", k, r[k-1], r[k])
	}
}

func TestDecideStochCross(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	bars := path("2026-09-14 08:00", closes)
	p := exitsOff()
	p.StopDailyATR, p.StochExit = 0, 80
	k, sig := firstSell(p, bars, 36)
	if k < 0 || sig.Reason != "STOCH" {
		t.Fatalf("first sell at %d %q, want STOCH", k, sig.Reason)
	}
	md := fixture(bars[:k+1])
	_, ds := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, p.StochPeriod, p.StochSmooth)
	if !(ds[len(ds)-2] < 80 && ds[len(ds)-1] >= 80) {
		t.Fatalf("exit at %d: %%K %v → %v, want a cross of 80", k, ds[len(ds)-2], ds[len(ds)-1])
	}
}

func TestDecideAllExitsOffHolds(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	p := exitsOff()
	p.StopDailyATR = 0
	if k, sig := firstSell(p, path("2026-09-14 08:00", closes), 36); k >= 0 {
		t.Fatalf("sell at %d %q, want hold with every exit off", k, sig.Reason)
	}
}

func TestDecideInvalidStochPeriodDoesNotPanic(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	bars := path("2026-09-14 08:00", closes)
	p := DefaultParams()
	p.StopDailyATR, p.ProfitPct = 0, 0
	p.StochPeriod = 0
	p.RSIExit = 0
	if k, sig := firstSell(p, bars, 36); k >= 0 {
		t.Fatalf("sell at %d %q, want hold: an invalid stochastic period disables that exit", k, sig.Reason)
	}
}

// underwaterCross is a path whose RSI(6) crosses 70 upward while the price is still below an entry
// at 100 on bar 30, then climbs back above it: flat at 100, an eight-bar slide, a steady recovery.
func underwaterCross() []bar {
	closes := append(flatN(30, 100), steps(100, 8, -0.5)...)
	closes = append(closes, steps(closes[len(closes)-1], 20, 0.3)...)
	return path("2026-09-14 08:00", closes)
}

// rsiOnly leaves only the RSI exit on, with the stop off so the slide cannot stop the trade.
func rsiOnly(inProfit int) Params {
	p := exitsOff()
	p.StopDailyATR, p.RSIExit, p.ExitInProfit = 0, 70, inProfit
	return p
}

// stochOnly leaves only the stochastic exit on, with the stop off.
func stochOnly(inProfit int) Params {
	p := exitsOff()
	p.StopDailyATR, p.StochExit, p.ExitInProfit = 0, 80, inProfit
	return p
}

// firstStochCross returns the first bar at or after from whose smoothed %K crosses level upward,
// or −1.
func firstStochCross(bars []bar, level float64, from int) int {
	md := fixture(bars)
	p := DefaultParams()
	_, k := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, p.StochPeriod, p.StochSmooth)
	off := len(bars) - len(k)
	for j := max(from-off, 1); j < len(k); j++ {
		if crossUp(k[j-1], k[j], level) {
			return off + j
		}
	}
	return -1
}

func TestStochExitUnderwaterWaitsForProfit(t *testing.T) {
	bars := underwaterCross()
	cross := firstStochCross(bars, 80, 31)
	if cross < 0 || bars[cross].c >= 100 {
		t.Fatalf("fixture: %%K cross at %d, want one below the entry 100", cross)
	}
	if k, sig := firstSell(stochOnly(0), bars, 30); k != cross || sig.Reason != "STOCH" {
		t.Fatalf("ExitInProfit 0: sell at %d %q, want STOCH at the cross %d", k, sig.Reason, cross)
	}
	want := -1
	for k := cross + 1; k < len(bars); k++ {
		if bars[k].c > 100 {
			want = k
			break
		}
	}
	k, sig := firstSell(stochOnly(1), bars, 30)
	if want < 0 || k != want || sig.Reason != "STOCH" || !strings.Contains(sig.ExitReason, "в плюс") {
		t.Fatalf("ExitInProfit 1: sell at %d %q %q, want a deferred STOCH at %d", k, sig.Reason, sig.ExitReason, want)
	}
}

// firstCross returns the first bar at or after from whose RSI crosses level upward, or −1.
func firstCross(bars []bar, period int, level float64, from int) int {
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.c
	}
	r := indicators.RSISeries(closes, period)
	for k := max(from, 1); k < len(r); k++ {
		if crossUp(r[k-1], r[k], level) {
			return k
		}
	}
	return -1
}

func TestRSIExitUnderwaterWaitsForProfit(t *testing.T) {
	bars := underwaterCross()
	cross := firstCross(bars, 6, 70, 30)
	if cross < 0 || bars[cross].c >= 100 {
		t.Fatalf("fixture: cross at %d close %v, want an RSI cross below the entry 100", cross, bars[cross].c)
	}
	if k, sig := firstSell(rsiOnly(0), bars, 30); k != cross || sig.Reason != "RSI" {
		t.Fatalf("ExitInProfit 0: sell at %d %q, want RSI at the cross %d", k, sig.Reason, cross)
	}
	want := -1
	for k := cross + 1; k < len(bars); k++ {
		if bars[k].c > 100 {
			want = k
			break
		}
	}
	if want < 0 {
		t.Fatal("fixture: the price never recovers above the entry")
	}
	k, sig := firstSell(rsiOnly(1), bars, 30)
	if k != want || sig.Reason != "RSI" {
		t.Fatalf("ExitInProfit 1: sell at %d %q, want RSI at the first close above the entry %d", k, sig.Reason, want)
	}
	if !strings.Contains(sig.ExitReason, "в плюс") {
		t.Fatalf("exit reason = %q, want the deferred-exit note", sig.ExitReason)
	}
}

func TestExitInProfitSellsAtTheCross(t *testing.T) {
	// TestDecideRSICross's path crosses above the entry: the rule does not delay that exit.
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	bars := path("2026-09-14 08:00", closes)
	cross := firstCross(bars, 6, 70, 36)
	if k, sig := firstSell(rsiOnly(1), bars, 36); k != cross || sig.Reason != "RSI" {
		t.Fatalf("sell at %d %q, want RSI at the cross %d", k, sig.Reason, cross)
	}
}

func TestRSIExitIgnoresCrossBeforeEntry(t *testing.T) {
	bars := underwaterCross()
	cross := firstCross(bars, 6, 70, 30)
	// Bought on the bar after the cross: RSI stays above 70 through the recovery, no new cross.
	if k, sig := firstSell(rsiOnly(1), bars, cross+1); k >= 0 {
		t.Fatalf("sell at %d %q, want hold: the only cross came before the entry", k, sig.Reason)
	}
}

func TestStopBeatsDeferredRSIExit(t *testing.T) {
	bars := underwaterCross()
	cross := firstCross(bars, 6, 70, 30)
	armed := bars[:cross+1]
	last := armed[len(armed)-1]
	armed = append(armed, bar{last.t.Add(step15), last.c, 101, 90, 100.5}) // above the entry, through the stop
	p := rsiOnly(1)
	p.StopDailyATR = 0.5
	if sig := held(p, armed, 100, bars[30].t); sig.Reason != "SL" {
		t.Fatalf("reason = %q, want SL first", sig.Reason)
	}
}

func TestLookbackCoversDeferredRSIExit(t *testing.T) {
	p := DefaultParams()
	if got := NewWithParams("TEST", step15, p).Lookback(); got < 200 {
		t.Fatalf("Lookback = %d, want ≥ 200 with ExitInProfit on", got)
	}
	p.ExitInProfit = 0
	if got := NewWithParams("TEST", step15, p).Lookback(); got != minLookback {
		t.Fatalf("Lookback = %d, want %d with the rule off", got, minLookback)
	}
}
