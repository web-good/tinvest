package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

var msk = time.FixedZone("MSK", 3*60*60)

// series builds MarketData for bars laid out at a fixed 30-minute span starting at the given
// MSK day/hour. Volumes are uniform so VWAP is the plain mean of typical prices.
type barSpec struct {
	h, l, c float64
	v       int64
}

// buildMD lays bars out at 30-minute steps from 2026-03-day hour:min MSK and returns the
// MarketData the strategy sees on the LAST bar.
func buildMD(day, hour, min int, bars []barSpec, dailyCloses []float64) strategy.MarketData {
	md := strategy.MarketData{DailyCloses: dailyCloses}
	start := time.Date(2026, 3, day, hour, min, 0, 0, msk)
	for i, b := range bars {
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, b.v)
		md.Times = append(md.Times, start.Add(time.Duration(i)*30*time.Minute))
	}
	md.Price = md.Closes[len(md.Closes)-1]
	return md
}

// flatDay returns n bars whose typical price is exactly p, each with volume 1000. The bars
// carry a small high/low range on purpose: a zero-range series would drive ATR to 0, and the
// entry rejects a non-positive ATR whenever the stop is armed.
func flatDay(n int, p float64) []barSpec {
	out := make([]barSpec, n)
	for i := range out {
		out[i] = barSpec{h: p + 0.15, l: p - 0.15, c: p, v: 1000}
	}
	return out
}

// setupEntry builds a window whose LAST bar is a textbook entry: a previous session, then a
// session trading at 100 that finally dips to 99.5 with the close in the upper half of the
// bar. Daily closes trend up so the daily gate passes.
//
// The resulting numbers on the last bar: VWAP ≈ 99.947, σ ≈ 0.16, deviation ≈ 0.447
// (≈ 2.8×σ, ≈ 0.449% of price) — clear of every default threshold with margin, so a single
// tweak in the table-driven test isolates exactly one gate.
func setupEntry() strategy.MarketData {
	bars := flatDay(10, 100) // previous session (window's first -> barsFromOpen -1)
	// Current session starts the next calendar day.
	cur := flatDay(9, 100)
	cur = append(cur, barSpec{h: 99.7, l: 99.2, c: 99.5, v: 1000}) // the dip bar
	all := append(bars, cur...)
	daily := make([]float64, 60)
	for i := range daily {
		daily[i] = 50 + float64(i) // strongly rising -> EMA far below 99.5
	}
	md := buildMD(2, 10, 0, all, daily)
	// Re-stamp the current session onto the next day starting at 07:00 MSK.
	base := time.Date(2026, 3, 3, 7, 0, 0, 0, msk)
	for i := range cur {
		md.Times[len(bars)+i] = base.Add(time.Duration(i) * 30 * time.Minute)
	}
	return md
}

func decide(t *testing.T, p Params, md strategy.MarketData) model.Signal {
	t.Helper()
	return NewWithParams("TEST", p).Decide(md)
}

func TestEnterFiresOnTextbookSetup(t *testing.T) {
	sig := decide(t, DefaultParams(), setupEntry())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v want SignalBuy", sig.Kind)
	}
	if sig.StopLoss <= 0 || sig.StopLoss >= md0Close {
		t.Fatalf("StopLoss = %v must sit below the entry price", sig.StopLoss)
	}
}

// md0Close is the close of the dip bar built by setupEntry.
const md0Close = 99.5

func TestEnterGates(t *testing.T) {
	tests := []struct {
		name  string
		tweak func(p *Params)
		want  model.SignalKind
	}{
		{"baseline fires", func(p *Params) {}, model.SignalBuy},
		{"deviation too shallow", func(p *Params) { p.EntryK = 50 }, model.SignalNone},
		{"deviation deeper than MaxDevK", func(p *Params) { p.MaxDevK = 0.01 }, model.SignalNone},
		{"MaxDevK<=0 disables the cap", func(p *Params) { p.MaxDevK = 0 }, model.SignalBuy},
		{"MinEdgePct not met", func(p *Params) { p.MinEdgePct = 5 }, model.SignalNone},
		{"MinEdgePct=0 lets it through", func(p *Params) { p.MinEdgePct = 0 }, model.SignalBuy},
		{"too early in the session", func(p *Params) { p.MinBarsFromOpen = 50 }, model.SignalNone},
		{"close in the lower third", func(p *Params) { p.MinClosePos = 0.99 }, model.SignalNone},
		{"MinClosePos<=0 disables the filter", func(p *Params) { p.MinClosePos = 0 }, model.SignalBuy},
		{"daily trend gate off", func(p *Params) { p.UseDailyTrend = 0 }, model.SignalBuy},
		{"entry cutoff already passed", func(p *Params) { p.SessionEndMin = 421 }, model.SignalNone},
		{"stop disabled is still a valid entry", func(p *Params) { p.StopATR = 0 }, model.SignalBuy},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			tc.tweak(&p)
			if got := decide(t, p, setupEntry()).Kind; got != tc.want {
				t.Fatalf("Kind = %v want %v", got, tc.want)
			}
		})
	}
}

func TestEnterRejectedWhenPriceBelowDailyEMA(t *testing.T) {
	md := setupEntry()
	// Falling daily series: EMA ends far ABOVE the intraday price.
	for i := range md.DailyCloses {
		md.DailyCloses[i] = 500 - float64(i)
	}
	if got := decide(t, DefaultParams(), md).Kind; got != model.SignalNone {
		t.Fatalf("Kind = %v want SignalNone below the daily EMA", got)
	}
}

// The daily trend gate is the one gate in this strategy that FAILS CLOSED: without a usable
// daily series there is no way to tell a pullback from a collapse.
func TestEnterRejectedWhenDailySeriesMissing(t *testing.T) {
	for _, name := range []string{"nil", "too short"} {
		t.Run(name, func(t *testing.T) {
			md := setupEntry()
			if name == "nil" {
				md.DailyCloses = nil
			} else {
				md.DailyCloses = md.DailyCloses[:3]
			}
			if got := decide(t, DefaultParams(), md).Kind; got != model.SignalNone {
				t.Fatalf("Kind = %v want SignalNone", got)
			}
		})
	}
}

func TestEnterRejectedOnFirstSessionOfWindow(t *testing.T) {
	// A window holding a single session: its bars carry barsFromOpen = -1 by the indicator's
	// rule, so no entry may fire however good the setup looks.
	cur := flatDay(9, 100)
	cur = append(cur, barSpec{h: 99.7, l: 99.2, c: 99.5, v: 1000})
	daily := make([]float64, 60)
	for i := range daily {
		daily[i] = 50 + float64(i)
	}
	md := buildMD(3, 7, 0, cur, daily)
	p := DefaultParams()
	p.MinBarsFromOpen = 0
	if got := decide(t, p, md).Kind; got != model.SignalNone {
		t.Fatalf("Kind = %v want SignalNone on the window's first session", got)
	}
}

func TestLookbackCoversAFullSession(t *testing.T) {
	if got := NewWithParams("T", DefaultParams()).Lookback(); got < 300 {
		t.Fatalf("Lookback = %d want >= 300", got)
	}
}
