package core

import (
	"math"
	"strings"
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

// openAt returns MarketData for a held position entered `heldBars` bars before the last bar.
func withPosition(md strategy.MarketData, entryPrice, stop float64, heldBars int) strategy.MarketData {
	last := md.Times[len(md.Times)-1]
	md.Position = &strategy.Position{
		PurchasePrice: entryPrice,
		Quantity:      10,
		StopLoss:      stop,
		EntryTime:     last.Add(-time.Duration(heldBars) * 30 * time.Minute),
	}
	return md
}

func TestExitStopWinsOverTargetOnTheSameBar(t *testing.T) {
	md := setupEntry()
	i := len(md.Closes) - 1
	// The bar sweeps both the stop below and the VWAP above.
	md.Lows[i] = 90
	md.Highs[i] = 120
	md = withPosition(md, 99.5, 95, 1)
	sig := decide(t, DefaultParams(), md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("Kind/Reason = %v/%q want Sell/SL", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 95 {
		t.Fatalf("StopLoss = %v want 95 (frozen at entry)", sig.StopLoss)
	}
}

func TestExitTargetIsPreviousBarVWAP(t *testing.T) {
	md := setupEntry()
	i := len(md.Closes) - 1
	md.Highs[i] = 1000 // certainly reaches any target
	md.Lows[i] = 99
	md = withPosition(md, 99.5, 1, 1)

	vwap, _, _, ok := NewWithParams("T", DefaultParams()).sessionVWAP(md)
	if !ok {
		t.Fatalf("sessionVWAP not usable in the fixture")
	}
	sig := decide(t, DefaultParams(), md)
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("Kind/Reason = %v/%q want Sell/TP", sig.Kind, sig.Reason)
	}
	if sig.TakeProfit != vwap[i-1] {
		t.Fatalf("TakeProfit = %v want previous-bar VWAP %v (current bar's is %v)",
			sig.TakeProfit, vwap[i-1], vwap[i])
	}
}

func TestExitTimeStop(t *testing.T) {
	base := setupEntry()
	i := len(base.Closes) - 1
	base.Highs[i] = 99.7 // never reaches the VWAP
	base.Lows[i] = 99.4

	tests := []struct {
		name     string
		held     int
		maxHold  int
		wantKind model.SignalKind
	}{
		{"not held long enough", 2, 8, model.SignalNone},
		{"held exactly the limit", 8, 8, model.SignalSell},
		{"MaxHoldBars<=0 disables", 50, 0, model.SignalNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			p.MaxHoldBars = tc.maxHold
			md := withPosition(base, 99.5, 1, tc.held)
			sig := decide(t, p, md)
			if sig.Kind != tc.wantKind {
				t.Fatalf("Kind = %v want %v", sig.Kind, tc.wantKind)
			}
			if tc.wantKind == model.SignalSell && sig.Reason != "TIME" {
				t.Fatalf("Reason = %q want TIME", sig.Reason)
			}
		})
	}
}

// A zero EntryTime must NOT be read as "held forever" — the time exit degrades to off.
func TestExitTimeStopSilentOnUnknownEntryTime(t *testing.T) {
	md := setupEntry()
	i := len(md.Closes) - 1
	md.Highs[i] = 99.7
	md.Lows[i] = 99.4
	md = withPosition(md, 99.5, 1, 1)
	md.Position.EntryTime = time.Time{}
	if got := decide(t, DefaultParams(), md).Kind; got != model.SignalNone {
		t.Fatalf("Kind = %v want SignalNone when EntryTime is unknown", got)
	}
}

func TestExitEndOfDay(t *testing.T) {
	md := setupEntry()
	i := len(md.Closes) - 1
	md.Highs[i] = 99.7
	md.Lows[i] = 99.4
	// Move the last bar to 22:30 MSK: it is the last one before DayEndMin (23:00).
	md.Times[i] = time.Date(2026, 3, 3, 22, 30, 0, 0, msk)
	md = withPosition(md, 99.5, 1, 1)
	p := DefaultParams()
	p.MaxHoldBars = 0 // isolate the EOD rule
	sig := decide(t, p, md)
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("Kind/Reason = %v/%q want Sell/EOD", sig.Kind, sig.Reason)
	}
}

func TestExplainMentionsEveryGate(t *testing.T) {
	out := NewWithParams("T", DefaultParams()).Explain(setupEntry())
	for _, want := range []string{"сессия", "VWAP", "σ", "MinEdgePct", "дневной тренд", "стоп"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Explain missing %q; got:\n%s", want, out)
		}
	}
}

// multiDay builds `days` sessions of `perDay` 30-minute bars starting at 07:00 MSK, with a
// deterministic zig-zag so deviations from the VWAP actually occur.
func multiDay(days, perDay int) strategy.MarketData {
	md := strategy.MarketData{}
	for d := 0; d < days; d++ {
		day := time.Date(2026, 3, 2+d, 7, 0, 0, 0, msk)
		for b := 0; b < perDay; b++ {
			p := 100 + float64(d) + math.Sin(float64(b)/3)*0.6
			md.Highs = append(md.Highs, p+0.2)
			md.Lows = append(md.Lows, p-0.25)
			md.Closes = append(md.Closes, p)
			md.Volumes = append(md.Volumes, 1000+int64(b))
			md.Times = append(md.Times, day.Add(time.Duration(b)*30*time.Minute))
		}
	}
	md.Price = md.Closes[len(md.Closes)-1]
	md.DailyCloses = make([]float64, 60)
	for i := range md.DailyCloses {
		md.DailyCloses[i] = 50 + float64(i)
	}
	return md
}

// slice returns md restricted to bars [from:to).
func sliceMD(md strategy.MarketData, from, to int) strategy.MarketData {
	out := md
	out.Highs = md.Highs[from:to]
	out.Lows = md.Lows[from:to]
	out.Closes = md.Closes[from:to]
	out.Volumes = md.Volumes[from:to]
	out.Times = md.Times[from:to]
	out.Price = out.Closes[len(out.Closes)-1]
	return out
}

// positionAt attaches an open long position to md, entered at absolute index entryIdx of md's
// OWN series. The fields mirror what the backtest engine populates on a real fill (see
// internal/domain/backtest/engine.go): PurchasePrice, StopLoss, TakeProfit, EntryATR, EntryTime.
// EntryTime is read from md.Times[entryIdx] — an absolute instant from the series itself — so
// barsHeld() stays consistent no matter how the window is later cut with sliceMD (sliceMD copies
// the struct and only overwrites the OHLCV/Times slices, so the Position pointer, and the
// absolute values it carries, survive the cut unchanged).
func positionAt(md strategy.MarketData, entryIdx int) strategy.MarketData {
	out := md
	entry := md.Closes[entryIdx]
	out.Position = &strategy.Position{
		PurchasePrice: entry,
		Quantity:      10,
		StopLoss:      entry - 0.5, // close enough that the fixture's zig-zag trips it sometimes
		TakeProfit:    entry + 1.5, // engine bookkeeping only; manage() derives its own TP from VWAP
		EntryATR:      0.4,
		EntryTime:     md.Times[entryIdx],
	}
	return out
}

// TestNoLookaheadAcrossWindowCuts is the load-bearing safety net: the decision on bar i must
// not depend on how much history precedes it, as long as one full session still precedes the
// session bar i belongs to. Any use of future bars, or any dependence on the window length,
// breaks this. It exercises BOTH sides of Decide: flat (routes to enter()) and with an open
// position (routes to manage()), so an exit gate cannot silently depend on the window either.
func TestNoLookaheadAcrossWindowCuts(t *testing.T) {
	const perDay = 20
	full := multiDay(4, perDay)
	s := NewWithParams("T", DefaultParams())

	checked := 0
	exitsSeen := 0
	// Bars of the last two sessions: each has at least one whole session before its own.
	for i := 2 * perDay; i < len(full.Closes); i++ {
		want := s.Decide(sliceMD(full, 0, i+1))

		// Cut so that exactly one full session precedes bar i's session.
		sessionStart := (i / perDay) * perDay

		// Managed sub-case: the same bar i, but with a position opened `held` bars earlier
		// (held cycles 3..9 across i so both a TIME exit, MaxHoldBars default 8, and a
		// not-yet-due case are covered by the loop). entryIdx indexes into `full` itself, so
		// the position's absolute entry price/time are identical for every window cut below.
		// Guarded to entryIdx >= sessionStart: manage()'s TP branch reads vwap[i-1], and if i
		// were the session's own opening bar, i-1 would belong to the PRECEDING session —
		// whose own completeness-in-window is exactly the thing TestFirstSessionInWindowNever-
		// Trades already covers, not what this loop guarantees for i. A real position could
		// never observe that seam anyway: the EOD rule always force-closes it before a new
		// session's first bar (see barsHeld's doc), so keeping the synthetic entry inside i's
		// own (already-guaranteed-complete) session matches how positions actually behave.
		held := 3 + i%7
		entryIdx := i - held
		havePos := entryIdx >= sessionStart
		var posFull strategy.MarketData
		var wantPos model.Signal
		if havePos {
			posFull = positionAt(full, entryIdx)
			wantPos = s.Decide(sliceMD(posFull, 0, i+1))
			if wantPos.Kind != model.SignalNone {
				exitsSeen++
			}
		}
		for _, from := range []int{sessionStart - perDay, sessionStart - perDay + 3} {
			if from < 0 {
				continue
			}
			got := s.Decide(sliceMD(full, from, i+1))
			if got.Kind != want.Kind || got.Reason != want.Reason {
				t.Fatalf("bar %d: cut at %d gave %v/%q, full window gave %v/%q",
					i, from, got.Kind, got.Reason, want.Kind, want.Reason)
			}
			// TakeProfit is the VWAP target: it must match exactly regardless of window cut,
			// since the session VWAP is scoped to the current session and ignores anything
			// before it. StopLoss is ATR-derived, and indicators.ATR is Wilder's recursive
			// (unbounded-memory) smoothing: its value at bar i depends on how much warm-up
			// history precedes it, not on any future bar, so a longer window legitimately
			// converges to a slightly different value than a shorter one. That is not
			// look-ahead — it is why StopLoss gets a relative tolerance here while
			// Kind/Reason/TakeProfit stay exact. In production this never matters: the
			// backtest engine (internal/domain/backtest/engine.go) always feeds Decide a
			// FIXED-length window (candles[i-l+1:i+1], l = Strategy.Lookback()), so ATR's
			// warm-up length — and thus the stop — is deterministic bar to bar.
			if math.Abs(got.TakeProfit-want.TakeProfit) > 1e-9 {
				t.Fatalf("bar %d: cut at %d gave target %v, full window %v",
					i, from, got.TakeProfit, want.TakeProfit)
			}
			if diff := math.Abs(got.StopLoss - want.StopLoss); diff > 1e-3*math.Abs(want.StopLoss) {
				t.Fatalf("bar %d: cut at %d gave stop %v, full window %v (relative diff %.6f exceeds 0.1%%)",
					i, from, got.StopLoss, want.StopLoss, diff/math.Abs(want.StopLoss))
			}
			checked++

			if havePos {
				gotPos := s.Decide(sliceMD(posFull, from, i+1))
				if gotPos.Kind != wantPos.Kind || gotPos.Reason != wantPos.Reason {
					t.Fatalf("bar %d (managed): cut at %d gave %v/%q, full window gave %v/%q",
						i, from, gotPos.Kind, gotPos.Reason, wantPos.Kind, wantPos.Reason)
				}
				if math.Abs(gotPos.TakeProfit-wantPos.TakeProfit) > 1e-9 {
					t.Fatalf("bar %d (managed): cut at %d gave exit target %v, full window %v",
						i, from, gotPos.TakeProfit, wantPos.TakeProfit)
				}
				if diff := math.Abs(gotPos.StopLoss - wantPos.StopLoss); diff > 1e-3*math.Abs(wantPos.StopLoss) {
					t.Fatalf("bar %d (managed): cut at %d gave exit stop %v, full window %v",
						i, from, gotPos.StopLoss, wantPos.StopLoss)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatalf("fixture produced no comparisons")
	}
	if exitsSeen == 0 {
		t.Fatalf("managed sub-case never produced a non-None signal — the exit branch was never exercised")
	}
}

// The mirror case: a window that leaves bar i's session FIRST must refuse to trade, because
// the anchor of that session is unknown.
func TestFirstSessionInWindowNeverTrades(t *testing.T) {
	const perDay = 20
	full := multiDay(4, perDay)
	s := NewWithParams("T", DefaultParams())
	for i := 3*perDay + 1; i < len(full.Closes); i++ {
		got := s.Decide(sliceMD(full, 3*perDay, i+1))
		if got.Kind != model.SignalNone {
			t.Fatalf("bar %d traded (%v) while its session was the window's first", i, got.Kind)
		}
	}
}
