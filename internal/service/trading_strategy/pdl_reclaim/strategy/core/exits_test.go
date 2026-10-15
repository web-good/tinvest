package core

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// quiet switches off the RSI and MACD exits so stop/target/EOD are tested alone.
func quiet() Params {
	p := m5()
	p.RSIPeriod, p.MACDFast = 0, 0
	return p
}

func held(p Params, bars []bar, pos strategy.Position) model.Signal {
	md := fixture(bars)
	md.Position = &pos
	return newS(p).Decide(md)
}

// pos0 filled at 99.4 on Monday 10:05: stop 99.4 − 0.5·2 = 98.4, target PDH 101. Its frozen
// StopLoss (98.3, from the signal close) is deliberately different: the stop is measured from the fill.
func pos0() strategy.Position {
	return strategy.Position{PurchasePrice: 99.4, EntryATR: 2, TakeProfit: 101, StopLoss: 98.3, EntryTime: at("2026-09-14 10:05")}
}

var fillBar = bar{"2026-09-14 10:05", 99.35, 99.5, 99.3, 99.4}

func TestExitStopFromFill(t *testing.T) {
	sig := held(quiet(), []bar{fillBar, {"2026-09-14 10:10", 99.4, 99.5, 98.4, 98.6}}, pos0())
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || math.Abs(sig.StopLoss-98.4) > eps {
		t.Fatalf("got %v %q SL %v, want Sell SL at 98.4", sig.Kind, sig.Reason, sig.StopLoss)
	}
}

// Review focus 5.
func TestExitStopBeatsTarget(t *testing.T) {
	sig := held(quiet(), []bar{fillBar, {"2026-09-14 10:10", 99.4, 101.2, 98.3, 100}}, pos0())
	if sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL", sig.Reason)
	}
}

func TestExitTarget(t *testing.T) {
	up := bar{"2026-09-14 10:10", 99.4, 101.0, 99.3, 100.9}
	sig := held(quiet(), []bar{fillBar, up}, pos0())
	if sig.Kind != model.SignalSell || sig.Reason != "TP" || math.Abs(sig.TakeProfit-101) > eps {
		t.Fatalf("got %v %q TP %v, want Sell TP at 101", sig.Kind, sig.Reason, sig.TakeProfit)
	}
	noTarget := pos0()
	noTarget.TakeProfit = 0
	if sig := held(quiet(), []bar{fillBar, up}, noTarget); sig.Kind != model.SignalNone {
		t.Fatalf("no target: Kind = %v, want None", sig.Kind)
	}
}

func TestExitHoldsInsideRange(t *testing.T) {
	if sig := held(quiet(), []bar{fillBar, {"2026-09-14 10:10", 99.4, 99.6, 99.3, 99.5}}, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
}

func TestExitEndOfDay(t *testing.T) {
	bars := append([]bar{fillBar}, flat("2026-09-14 10:10", "2026-09-14 18:25", 99.5)...)
	if sig := held(quiet(), bars, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("18:25: Kind = %v, want None", sig.Kind)
	}
	bars = append(bars, bar{"2026-09-14 18:30", 99.5, 99.5, 99.5, 99.5})
	if sig := held(quiet(), bars, pos0()); sig.Reason != "EOD" {
		t.Fatalf("18:30: Reason = %q, want EOD", sig.Reason)
	}
}

// Review focus 3: no 18:30 bar in the data — the first bar of the next date closes the position.
func TestExitNextDate(t *testing.T) {
	bars := []bar{fillBar, {"2026-09-14 16:00", 99.5, 99.5, 99.5, 99.5}, {"2026-09-15 07:00", 99.5, 99.5, 99.5, 99.5}}
	if sig := held(quiet(), bars, pos0()); sig.Reason != "EOD" {
		t.Fatalf("Reason = %q, want EOD", sig.Reason)
	}
}

func TestExitRSI(t *testing.T) {
	p := m5()
	p.MACDFast = 0
	// up 0.1, back 0.02: Wilder RSI with no losses at all is 50 in pkg/indicators, so the path
	// needs small pullbacks to read high.
	bars := []bar{fillBar}
	c := 99.4
	for k := 1; k <= 10; k++ {
		if k%2 == 1 {
			c += 0.1
		} else {
			c -= 0.02
		}
		bars = append(bars, bar{at("2026-09-14 10:05").Add(time.Duration(k) * 5 * time.Minute).Format("2006-01-02 15:04"), c - 0.05, c + 0.01, c - 0.06, c})
	}
	if sig := held(p, bars, pos0()); sig.Reason != "RSI" {
		t.Fatalf("Reason = %q, want RSI", sig.Reason)
	}
	p.RSIExit = 0
	if sig := held(p, bars, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("RSI off: Kind = %v, want None", sig.Kind)
	}
}

// macdPath: 60 flat bars at 100 from Monday 07:00, 12 bars rising by 0.1, 30 bars falling by 0.1.
func macdPath() []bar {
	var out []bar
	c := 100.0
	for k := 0; k < 102; k++ {
		switch {
		case k >= 72:
			c -= 0.1
		case k >= 60:
			c += 0.1
		}
		ts := at("2026-09-14 07:00").Add(time.Duration(k) * 5 * time.Minute).Format("2006-01-02 15:04")
		out = append(out, bar{ts, c, c, c, c})
	}
	return out
}

// firstRed is the first bar after the rise starts whose histogram is red right after a green one.
func firstRed(t *testing.T, bars []bar) (int, []float64) {
	t.Helper()
	closes := make([]float64, len(bars))
	for k, b := range bars {
		closes[k] = b.c
	}
	d := DefaultParams()
	m, s := indicators.MACD(closes, d.MACDFast, d.MACDSlow, d.MACDSignal)
	h := make([]float64, len(m))
	for k := range m {
		h[k] = m[k] - s[k]
	}
	for r := 61; r < len(h)-1; r++ {
		if h[r] < 0 && h[r-1] >= 0 {
			if h[r+1] >= 0 {
				t.Fatalf("histogram at %d is green again", r+1)
			}
			return r, h
		}
	}
	t.Fatal("no red-after-green bar in the path")
	return 0, nil
}

func macdPos(entry bar) strategy.Position {
	return strategy.Position{PurchasePrice: entry.c, EntryATR: 100, EntryTime: at(entry.t)} // stop far, no target
}

func TestExitMACDAfterGreen(t *testing.T) {
	p := m5()
	p.RSIPeriod = 0
	bars := macdPath()
	r, _ := firstRed(t, bars)
	if sig := held(p, bars[:r+1], macdPos(bars[60])); sig.Reason != "MACD" {
		t.Fatalf("first red: Reason = %q, want MACD", sig.Reason)
	}
	if sig := held(p, bars[:r], macdPos(bars[60])); sig.Kind != model.SignalNone {
		t.Fatalf("still green: Kind = %v, want None", sig.Kind)
	}
	// green on the entry bar itself counts
	if sig := held(p, bars[:r+1], macdPos(bars[r-1])); sig.Reason != "MACD" {
		t.Fatalf("green at entry: Reason = %q, want MACD", sig.Reason)
	}
	// green only before the entry does not
	if sig := held(p, bars[:r+2], macdPos(bars[r])); sig.Kind != model.SignalNone {
		t.Fatalf("green before entry: Kind = %v, want None", sig.Kind)
	}
}

// Review focus 2: no entry time, or an entry time outside the window — MACD exit stays off.
func TestExitMACDWithoutEntryBar(t *testing.T) {
	p := m5()
	p.RSIPeriod = 0
	bars := macdPath()
	r, _ := firstRed(t, bars)
	pos := macdPos(bars[60])
	pos.EntryTime = time.Time{}
	if sig := held(p, bars[:r+1], pos); sig.Kind != model.SignalNone {
		t.Fatalf("zero EntryTime: Kind = %v, want None", sig.Kind)
	}
	pos.EntryTime = at("2026-09-14 06:00")
	if sig := held(p, bars[:r+1], pos); sig.Kind != model.SignalNone {
		t.Fatalf("EntryTime before window: Kind = %v, want None", sig.Kind)
	}
}
