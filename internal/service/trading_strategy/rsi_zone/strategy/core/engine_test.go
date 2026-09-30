// engine_test.go exercises the strategy through the real backtest engine (backtest.Run), rather
// than calling Decide directly, to guard the handoff of the entry signal's ATR into
// Position.EntryATR: the stop the position relies on for the rest of its life is built from that
// frozen value (see internal/domain/backtest/portfolio.go's open/strategyPosition), and a broken
// handoff would only show up here, never in a unit test that constructs a strategy.Position by
// hand.
package core

import (
	"math"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

// engineUptrendCloses is trendCloses generalized with a configurable uptrend length, so the
// synthetic history can be sized around a specific Lookback(): a steady uptrend (+0.08% per bar)
// of upBars bars, followed by downBars bars of -0.2% — the same shape trendCloses(downBars) uses,
// just not pinned to 400 up bars.
func engineUptrendCloses(upBars, downBars int) []float64 {
	out := make([]float64, 0, upBars+downBars)
	p := 100.0
	for i := 0; i < upBars; i++ {
		p *= 1.0008
		out = append(out, p)
	}
	for i := 0; i < downBars; i++ {
		p *= 0.998
		out = append(out, p)
	}
	return out
}

// engineCandles turns closes into 30-minute bars ending at `last` (the LAST close's bar time),
// with Open == Close and the same 0.3% high/low envelope fixtureWithDaily uses.
func engineCandles(closes []float64, last time.Time) []bt.Candle {
	n := len(closes)
	start := last.Add(-time.Duration(n-1) * 30 * time.Minute)
	out := make([]bt.Candle, n)
	for i, c := range closes {
		out[i] = bt.Candle{
			Time:  start.Add(time.Duration(i) * 30 * time.Minute),
			Open:  c,
			High:  c * 1.003,
			Low:   c * 0.997,
			Close: c,
		}
	}
	return out
}

// engineDailyCandles converts dailyBars' aligned slices into daily bars for backtest.Run, over
// `days` calendar days ending the day before `before` — weekday bars span `w`, weekend bars
// `we`, so the strategy's own weekend filter (weekdayDaily) has something to prove against.
func engineDailyCandles(before time.Time, days int, w, we float64) []bt.Candle {
	h, l, c, times := dailyBars(before, days, w, we)
	out := make([]bt.Candle, len(c))
	for i := range c {
		out[i] = bt.Candle{Time: times[i], Open: c[i], High: h[i], Low: l[i], Close: c[i]}
	}
	return out
}

// TestEngineHandsSignalATRIntoPositionEntryATRForTheStop replays a real Buy through the backtest
// engine and confirms the stop that later closes the trade is built from the ATR the Buy signal
// carried (sig.ATR -> portfolio.entryATR -> Position.EntryATR -> StopLevel), not recomputed from
// scratch on the exit bar. It also pins the engine's stop fill rule along the way: the exit price
// is min(stop level, the exit bar's open) — a gap through the stop prices at the worse, gapped
// open, not at the level.
func TestEngineHandsSignalATRIntoPositionEntryATRForTheStop(t *testing.T) {
	p := DefaultParams() // EMAPeriod 200 -> Lookback() == 420
	s := NewWithParams("TEST", p)
	lookback := s.Lookback()
	if lookback != 420 {
		t.Fatalf("Lookback() = %d, want 420 (test is sized around the default)", lookback)
	}

	// Same shape as trendCloses(3) (see TestFixturesHaveTheirIntendedShape: a steady uptrend puts
	// the RSI(4) cross below 25 exactly on the 3rd down bar while price stays far above EMA(200)),
	// just with a configurable uptrend length so the engine's sliding Lookback() window sees
	// (almost) the same history the unit-level fixture hands Decide directly.
	closes := engineUptrendCloses(lookback, 3)
	entryPrice := closes[len(closes)-1]
	entryBarTime := mondayNoon.Add(-30 * time.Minute) // a Monday; the exit bar below lands on mondayNoon itself
	candles := engineCandles(closes, entryBarTime)

	// The exit bar: a gap DOWN whose open already sits below where the frozen stop will land, so
	// the engine's fill rule (min(level, open)) must price the exit at the open, not at the level
	// — proving the test exercises the min(), not just the level by coincidence.
	level := entryPrice - p.StopDailyATR*dailyWidth // StopLevel's formula, computed independently
	gapOpen := entryPrice - 3*dailyWidth            // well below level: a real gap, not a graze
	candles = append(candles, bt.Candle{
		Time: mondayNoon, Open: gapOpen, High: gapOpen, Low: gapOpen - 1, Close: gapOpen - 0.5,
	})

	daily := engineDailyCandles(mondayNoon, 40, dailyWidth, dailyWidth/10)
	cfg := bt.Config{InitialCash: 1_000_000, Fraction: 1.0, Commission: 0, Lot: 1}

	res := bt.Run(s, candles, daily, nil, cfg)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want exactly 1 (one Buy at the cross, one SL at the gap)", len(res.Trades))
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-entryPrice) > 1e-6 {
		t.Fatalf("EntryPrice = %v, want %v: the entry did not fire on the designed cross bar", tr.EntryPrice, entryPrice)
	}
	if tr.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL", tr.Reason)
	}
	if math.Abs(tr.ATR-dailyWidth) > 1e-9 {
		t.Fatalf("ATR recorded on the trade = %v, want %v: the signal's ATR must reach Position.EntryATR unchanged", tr.ATR, dailyWidth)
	}
	wantExit := math.Min(level, gapOpen)
	if math.Abs(tr.ExitPrice-wantExit) > 1e-6 {
		t.Fatalf("ExitPrice = %v, want %v (= min(stop level %v, gap open %v)): the engine must fill a stop at the worse of the two on a gap",
			tr.ExitPrice, wantExit, level, gapOpen)
	}
}
