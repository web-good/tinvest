package core

import (
	"testing"
	"time"

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

func TestGreenThenRed(t *testing.T) {
	t0 := at("2026-09-14 10:00")
	times := make([]time.Time, 6)
	for i := range times {
		times[i] = t0.Add(time.Duration(i) * BarStep)
	}
	entry := times[2] // bars 2..5 are in the trade
	cases := []struct {
		name  string
		h     []float64
		entry time.Time
		want  bool
	}{
		{"green then red in trade", []float64{-1, -1, -1, 0.5, 0.2, -0.1}, entry, true},
		{"never green in trade", []float64{-1, -1, -1, -0.5, -0.2, -0.1}, entry, false},
		{"green only before entry", []float64{0.5, 0.4, -0.1, -0.2, -0.3, -0.4}, entry, false},
		{"green on the fill bar counts", []float64{-1, -1, 0.3, -0.1, -0.2, -0.3}[:4], entry, true},
		{"still green", []float64{-1, -1, -1, 0.5, 0.6, 0.7}, entry, false},
		{"zero between green and red", []float64{-1, -1, -1, 0.5, 0, -0.1}, entry, true},
		{"unknown entry scans the window", []float64{0.5, -1, -1, -1, -1, -0.1}, time.Time{}, false},
		{"unknown entry, green then red", []float64{-1, -1, -1, -1, 0.1, -0.1}, time.Time{}, true},
	}
	for _, c := range cases {
		if got := greenThenRed(c.h, times[:len(c.h)], c.entry); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
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

// macdHist is MACD − signal over closes with the default periods.
func macdHist(closes []float64) []float64 {
	p := DefaultParams()
	m, s := indicators.MACD(closes, p.MACDFast, p.MACDSlow, p.MACDSignal)
	h := make([]float64, len(closes))
	for i := range h {
		h[i] = m[i] - s[i]
	}
	return h
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

func macdOnly() Params {
	p := exitsOff()
	p.StopPct, p.MACDExit = 0, 1
	return p
}

func TestDecideMACDExitAfterGreenInTrade(t *testing.T) {
	closes := append(flatN(60, 100), steps(100, 6, 0.5)...)
	closes = append(closes, steps(closes[len(closes)-1], 15, -0.5)...)
	bars := path("2026-09-14 08:00", closes)
	k, sig := firstSell(macdOnly(), bars, 60)
	if k < 0 || sig.Reason != "MACD" {
		t.Fatalf("first sell at %d %q, want a MACD exit", k, sig.Reason)
	}
	h := macdHist(closes[:k+1])
	if h[k] >= 0 || h[k-1] <= 0 {
		t.Fatalf("exit at %d: hist %v → %v, want the first red after green", k, h[k-1], h[k])
	}
}

func TestDecideMACDIgnoresGreenBeforeEntry(t *testing.T) {
	closes := append(flatN(40, 100), steps(100, 6, 0.5)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, -0.5)...)
	h := macdHist(closes)
	r := -1
	for i := 46; i < len(h); i++ {
		if h[i] < 0 {
			r = i
			break
		}
	}
	if r < 0 {
		t.Fatal("fixture: histogram never turned red after the rise")
	}
	for i := r; i < len(h); i++ {
		if h[i] >= 0 {
			t.Fatalf("fixture: histogram green again at %d", i)
		}
	}
	bars := path("2026-09-14 08:00", closes)
	if k, sig := firstSell(macdOnly(), bars, r); k >= 0 {
		t.Fatalf("sell at %d %q, want hold: the trade never saw a green bar", k, sig.Reason)
	}
}

func TestDecideRSICross(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	bars := path("2026-09-14 08:00", closes)
	p := exitsOff()
	p.StopPct, p.RSIExit = 0, 70
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
	p.StopPct, p.StochExit = 0, 80
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
	p.StopPct = 0
	if k, sig := firstSell(p, path("2026-09-14 08:00", closes), 36); k >= 0 {
		t.Fatalf("sell at %d %q, want hold with every exit off", k, sig.Reason)
	}
}

func TestDecideInvalidPeriodsDoNotPanic(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	bars := path("2026-09-14 08:00", closes)
	p := DefaultParams()
	p.StopPct, p.ProfitPct = 0, 0
	p.MACDFast, p.MACDSlow = 14, 9
	p.StochPeriod = 0
	p.RSIExit = 0
	if k, sig := firstSell(p, bars, 36); k >= 0 {
		t.Fatalf("sell at %d %q, want hold: invalid MACD/stochastic periods disable those exits", k, sig.Reason)
	}
}
