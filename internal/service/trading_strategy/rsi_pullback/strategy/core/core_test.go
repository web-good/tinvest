package core

import (
	"math"
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
