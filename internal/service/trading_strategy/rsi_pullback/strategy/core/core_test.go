package core

import (
	"math"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// msk is the timezone every session rule is anchored to.
var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// barSeries builds MarketData from closes, stamping bars every 30 minutes starting at start.
// Highs/Lows are derived from the close with a fixed 0.3% envelope so ATR is always positive.
func barSeries(closes []float64, start time.Time) strategy.MarketData {
	n := len(closes)
	md := strategy.MarketData{
		Highs:   make([]float64, n),
		Lows:    make([]float64, n),
		Closes:  append([]float64(nil), closes...),
		Volumes: make([]int64, n),
		Times:   make([]time.Time, n),
	}
	for i, c := range closes {
		md.Highs[i] = c * 1.003
		md.Lows[i] = c * 0.997
		md.Volumes[i] = 1000
		md.Times[i] = start.Add(time.Duration(i) * 30 * time.Minute)
	}
	md.Price = closes[n-1]
	return md
}

// pullbackCloses builds a long uptrend (so the fast EMA sits above the slow one) followed by a
// sharp multi-bar drop that pushes a short RSI below the lower band on the LAST bar.
func pullbackCloses() []float64 {
	const rise = 400
	out := make([]float64, 0, rise+6)
	p := 100.0
	for i := 0; i < rise; i++ {
		p *= 1.0008
		out = append(out, p)
	}
	// Five consecutive down bars: RSI(4) crosses below the lower band (15) only on the last one.
	for i := 0; i < 5; i++ {
		p *= 0.998
		out = append(out, p)
	}
	return out
}

// entryFixture returns market data whose LAST bar is a valid entry bar at 12:00 MSK Monday.
func entryFixture() strategy.MarketData {
	closes := pullbackCloses()
	// 2026-06-01 is a Monday. Place the last bar at 12:00 MSK.
	last := time.Date(2026, 6, 1, 12, 0, 0, 0, msk)
	start := last.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	return barSeries(closes, start)
}

// downtrendCloses builds a LONG DECLINE (so the fast EMA ends up AT OR BELOW the slow one)
// followed by the same shape of multi-bar shock as pullbackCloses, calibrated so RSI(4) crosses
// below the lower band (15) only on the LAST bar. This isolates the trend gate in
// TestEnterGates's "downtrend" case: the RSI gate passes cleanly (same fresh-cross shape as the
// buy fixture), so a rejection can only come from fast[i] <= slow[i].
//
// A pure monotonic decline will NOT work here: Wilder's avgGain sits at exactly 0 when every bar
// is a loss (see pkg/indicators/rsi.go), which pins RSI(4) at its 0 floor from the very first
// down bar — there is no ">= 15" bar left to cross down FROM. The baseline below alternates a
// smaller up-tick with a larger down-tick (net down) to keep avgGain > 0 throughout, so RSI has
// room to sit above 15 until the shock. The exact multipliers were found by a small offline
// search over indicators.RSISeries/ema.Compute output (not eyeballed): with this shape,
// rsi[n-2]=15.45 (>=15) and rsi[n-1]=11.40 (<15), while fast(10)=99.88 <= slow(100)=99.96.
func downtrendCloses() []float64 {
	const baseLen = 400
	out := make([]float64, 0, baseLen+6)
	p := 100.0
	for i := 0; i < baseLen; i++ {
		if i%2 == 0 {
			p *= 0.9996
		} else {
			p *= 1.0004
		}
		out = append(out, p)
	}
	// Five consecutive down bars: same role as pullbackCloses' shock, calibrated so the RSI(4)
	// cross below 15 lands on the LAST bar (not earlier).
	for i := 0; i < 5; i++ {
		p *= 0.9995
		out = append(out, p)
	}
	return out
}

// downtrendFixture returns market data whose LAST bar clears the RSI entry gate (fresh cross
// below RSILower) but sits inside a downtrend, so fast EMA <= slow EMA — used to prove the trend
// gate alone blocks the entry.
func downtrendFixture() strategy.MarketData {
	closes := downtrendCloses()
	// 2026-06-01 is a Monday. Place the last bar at 12:00 MSK (same slot as entryFixture).
	last := time.Date(2026, 6, 1, 12, 0, 0, 0, msk)
	start := last.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	return barSeries(closes, start)
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	want := Params{
		RSIPeriod: 4, RSILower: 15, RSIUpper: 70,
		EMAFast: 10, EMASlow: 100,
		StopATR: 1.2, ATRPeriod: 14, MaxHoldBars: 8,
		SessionStartMin: 420, SessionEndMin: 1020, DayEndMin: 1380,
	}
	if p != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", p, want)
	}
}

func TestLookbackCoversSlowEMA(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	if got := s.Lookback(); got < 220 {
		t.Fatalf("Lookback() = %d, want >= 220 (2*EMASlow+20)", got)
	}
	p := DefaultParams()
	p.EMASlow = 200
	if got := NewWithParams("T", p).Lookback(); got != 420 {
		t.Fatalf("Lookback() with EMASlow=200 = %d, want 420", got)
	}
}

func TestEnterBuysThePullback(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	got := s.Decide(md)
	if got.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy (EntryReason %q)", got.Kind, got.EntryReason)
	}
	i := len(md.Closes) - 1
	wantStop := md.Closes[i] - DefaultParams().StopATR*got.ATR
	if got.ATR <= 0 {
		t.Fatalf("ATR = %v, want > 0", got.ATR)
	}
	if math.Abs(got.StopLoss-wantStop) > 1e-9 {
		t.Fatalf("StopLoss = %v, want %v", got.StopLoss, wantStop)
	}
	if got.EntryReason == "" {
		t.Fatal("EntryReason is empty")
	}
}

// TestEnterGates breaks exactly one precondition at a time and requires no entry.
func TestEnterGates(t *testing.T) {
	base := DefaultParams()
	tests := []struct {
		name  string
		tweak func(p *Params, md *strategy.MarketData)
	}{
		{"before the session opens", func(_ *Params, md *strategy.MarketData) {
			shiftTo(md, time.Date(2026, 6, 1, 6, 30, 0, 0, msk))
		}},
		{"after the entry window closes", func(_ *Params, md *strategy.MarketData) {
			shiftTo(md, time.Date(2026, 6, 1, 17, 0, 0, 0, msk))
		}},
		{"weekend", func(_ *Params, md *strategy.MarketData) {
			// 2026-06-06 is a Saturday.
			shiftTo(md, time.Date(2026, 6, 6, 12, 0, 0, 0, msk))
		}},
		{"downtrend: fast EMA below slow", func(_ *Params, md *strategy.MarketData) {
			// Swap in a fixture whose RSI gate passes cleanly (same fresh-cross shape as the
			// buy fixture) but whose price path is a genuine downtrend, so a rejection here can
			// only be explained by the trend gate — not by an earlier RSI-gate failure.
			*md = downtrendFixture()
		}},
		{"EMAFast == EMASlow: equality is not \"above\"", func(p *Params, _ *strategy.MarketData) {
			// Reuse the untouched buy fixture (its RSI gate is already proven to pass) and force
			// EMAFast and EMASlow to the SAME period. ema.Compute is a pure function of
			// (closes, period), so two calls with an identical period produce BIT-IDENTICAL
			// series — fast[i] == slow[i] exactly, no floating-point luck required. This isolates
			// the "<=" in enter()'s trend check: a gate written as "< slow" instead of "<= slow"
			// would let this exact-equality case through as a Buy.
			p.EMAFast = p.EMASlow
		}},
		{"RSI stays above the lower band", func(p *Params, _ *strategy.MarketData) {
			p.RSILower = 0.5
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			md := entryFixture()
			tc.tweak(&p, &md)
			if got := NewWithParams("T", p).Decide(md); got.Kind != model.SignalNone {
				t.Fatalf("Kind = %v, want None (reason %q)", got.Kind, got.EntryReason)
			}
		})
	}
}

// shiftTo re-stamps the series so its LAST bar opens at last.
func shiftTo(md *strategy.MarketData, last time.Time) {
	start := last.Add(-time.Duration(len(md.Times)-1) * 30 * time.Minute)
	for i := range md.Times {
		md.Times[i] = start.Add(time.Duration(i) * 30 * time.Minute)
	}
}

// TestEnterCrossIsAnEventNotAState: once RSI already sits below the band on the PREVIOUS bar,
// there is no fresh cross and therefore no entry.
func TestEnterCrossIsAnEventNotAState(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	// Append one more down bar: now bar i-1 is already below the band.
	last := md.Closes[len(md.Closes)-1] * 0.994
	md = barSeries(append(md.Closes, last), md.Times[0])
	shiftTo(&md, time.Date(2026, 6, 1, 12, 30, 0, 0, msk))
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None: RSI was already below the band on the previous bar", got.Kind)
	}
}

func TestEnterRejectsShortHistory(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := barSeries([]float64{100, 99}, time.Date(2026, 6, 1, 11, 0, 0, 0, msk))
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None on a two-bar series", got.Kind)
	}
}

// withPosition attaches an open long entered heldBars bars before the last bar of md.
func withPosition(md strategy.MarketData, entryPrice, stop float64, heldBars int) strategy.MarketData {
	last := md.Times[len(md.Times)-1]
	md.Position = &strategy.Position{
		PurchasePrice: entryPrice,
		StopLoss:      stop,
		EntryATR:      entryPrice * 0.003,
		EntryTime:     last.Add(-time.Duration(heldBars) * 30 * time.Minute),
	}
	return md
}

func TestExitStopLoss(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	i := len(md.Closes) - 1
	// Stop sits just above the bar's low: the stop must fire.
	stop := md.Lows[i] * 1.0001
	md = withPosition(md, md.Closes[i]*1.02, stop, 1)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "SL" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/SL", got.Kind, got.Reason)
	}
	if math.Abs(got.StopLoss-stop) > 1e-9 {
		t.Fatalf("StopLoss = %v, want the frozen position stop %v", got.StopLoss, stop)
	}
}

// upperCrossFixture builds an uptrend whose LAST bar pushes RSI(4) above the upper band.
//
// The 1.010 up-tick from the plan's literal fixture only reaches RSI(4)=69.95 on the last bar —
// just short of the 70 upper band, so no cross fires. Verified by an offline run of
// indicators.RSISeries over this exact shape (not eyeballed): 1.012 clears it cleanly, with
// rsi[n-2]=55.32 (<=70, not yet crossed) and rsi[n-1]=73.53 (>70, crossed).
func upperCrossFixture() strategy.MarketData {
	closes := make([]float64, 0, 410)
	p := 100.0
	for i := 0; i < 400; i++ {
		p *= 1.0008
		closes = append(closes, p)
	}
	// Three down bars pull RSI below the upper band, then two sharp up bars push it back above.
	for i := 0; i < 3; i++ {
		p *= 0.994
		closes = append(closes, p)
	}
	for i := 0; i < 2; i++ {
		p *= 1.012
		closes = append(closes, p)
	}
	last := time.Date(2026, 6, 1, 14, 0, 0, 0, msk)
	start := last.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	return barSeries(closes, start)
}

func TestExitOnRSIEnteringUpperBand(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	// Stop far below so only the RSI exit can fire.
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*0.5, 2)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/RSI", got.Kind, got.Reason)
	}
}

func TestExitStopWinsOverRSIOnTheSameBar(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	// Stop inside the bar AND the RSI cross on the same bar: SL must win.
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*1.0001, 2)
	got := s.Decide(md)
	if got.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL to take precedence over RSI", got.Reason)
	}
}

func TestExitTimeStop(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 3
	s := NewWithParams("T", p)
	md := entryFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 3)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "TIME" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/TIME after 3 bars", got.Kind, got.Reason)
	}
}

func TestExitTimeStopDisabledAtZero(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 0
	s := NewWithParams("T", p)
	md := entryFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 50)
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v (%q), want None: the time stop is disabled at MaxHoldBars=0", got.Kind, got.Reason)
	}
}

func TestExitTimeStopSilentOnUnknownEntryTime(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 1
	s := NewWithParams("T", p)
	md := entryFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 5)
	md.Position.EntryTime = time.Time{}
	if got := s.Decide(md); got.Reason == "TIME" {
		t.Fatal("TIME fired with an unknown entry time; it must stay silent")
	}
}

func TestExitEndOfDay(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	shiftTo(&md, time.Date(2026, 6, 1, 22, 30, 0, 0, msk))
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 2)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "EOD" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/EOD on the 22:30 bar", got.Kind, got.Reason)
	}
}

func TestExitHoldsThroughTheEveningSession(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 0
	s := NewWithParams("T", p)
	md := entryFixture()
	shiftTo(&md, time.Date(2026, 6, 1, 19, 0, 0, 0, msk))
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 2)
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v (%q), want None: 19:00 is past the entry window but before DayEndMin",
			got.Kind, got.Reason)
	}
}

func TestExplainMentionsEveryGate(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	out := s.Explain(entryFixture())
	for _, want := range []string{"сессия", "RSI", "EMA", "стоп", "удержание"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Explain() is missing %q; got:\n%s", want, out)
		}
	}
}

// sliceMD returns md restricted to bars [from:to).
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

// TestNoLookaheadAcrossWindowCuts is the load-bearing safety net: the decision on bar i must not
// depend on how much history precedes it. Cuts stay far enough from bar i that every indicator
// is warmed in both windows; the ATR-derived stop is compared with a relative tolerance because
// indicators.ATR is an unbounded Wilder recursion whose value depends on the warm-up length —
// but never on future bars. In production the engine always feeds a fixed-length Lookback()
// window, so this discrepancy cannot arise there.
func TestNoLookaheadAcrossWindowCuts(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	full := entryFixture()
	n := len(full.Closes)

	checked := 0
	for _, from := range []int{0, 40, 80} {
		want := s.Decide(full)
		got := s.Decide(sliceMD(full, from, n))
		if got.Kind != want.Kind || got.Reason != want.Reason {
			t.Fatalf("cut at %d gave %v/%q, full window gave %v/%q",
				from, got.Kind, got.Reason, want.Kind, want.Reason)
		}
		if math.Abs(got.RSI-want.RSI) > 1e-9 {
			t.Fatalf("cut at %d gave RSI %v, full window %v", from, got.RSI, want.RSI)
		}
		if want.StopLoss > 0 {
			if rel := math.Abs(got.StopLoss-want.StopLoss) / want.StopLoss; rel > 1e-3 {
				t.Fatalf("cut at %d gave stop %v, full window %v (relative %g)",
					from, got.StopLoss, want.StopLoss, rel)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("fixture produced no comparisons")
	}
}

// TestNoLookaheadWithOpenPosition covers the manage() path the same way.
func TestNoLookaheadWithOpenPosition(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	full := upperCrossFixture()
	n := len(full.Closes)
	i := n - 1
	full = withPosition(full, full.Closes[i]*0.97, full.Lows[i]*0.5, 2)

	want := s.Decide(full)
	if want.Kind == model.SignalNone {
		t.Fatal("fixture produced no exit; the managed sub-case would assert nothing")
	}
	for _, from := range []int{40, 80} {
		cut := sliceMD(full, from, n)
		cut.Position = full.Position
		got := s.Decide(cut)
		if got.Kind != want.Kind || got.Reason != want.Reason {
			t.Fatalf("cut at %d gave %v/%q, full window gave %v/%q",
				from, got.Kind, got.Reason, want.Kind, want.Reason)
		}
	}
}
