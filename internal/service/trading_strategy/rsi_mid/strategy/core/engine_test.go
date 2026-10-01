// engine_test.go exercises rsi_mid through the real backtest engine (backtest.Run) to guard two
// handoffs a hand-built strategy.Position never tests: the entry signal's ATR reaching
// Position.EntryATR (the stop is built from it for the whole trade), and Position.EntryTime
// anchoring the MID run on the entry bar.
package core

import (
	"math"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

// engineCandles turns closes into 30-minute bars ending at `last` (the LAST close's bar time),
// with Open == Close and a 0.3% high/low envelope.
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

// engineDailyCandles converts dailyBars into daily candles for backtest.Run.
func engineDailyCandles(before time.Time, days int, w, we float64) []bt.Candle {
	h, l, c, times := dailyBars(before, days, w, we)
	out := make([]bt.Candle, len(c))
	for i := range c {
		out[i] = bt.Candle{Time: times[i], Open: c[i], High: h[i], Low: l[i], Close: c[i]}
	}
	return out
}

var engineCfg = bt.Config{InitialCash: 1_000_000, Fraction: 1.0, Commission: 0, Lot: 1}

// The Buy's ATR must reach Position.EntryATR unchanged, and the stop built from it must fill at
// min(level, open) on a gap through it.
func TestEngineStopUsesTheSignalATRAndFillsAtTheGap(t *testing.T) {
	p := DefaultParams()
	s := NewWithParams("TEST", p)
	l := s.Lookback()
	closes := crossCloses(l, 0.0008, p.RSIPeriod, midLine, l)
	entryPrice := closes[len(closes)-1]
	entryBarTime := mondayNoon.Add(-30 * time.Minute)
	candles := engineCandles(closes, entryBarTime)

	level := entryPrice - p.StopDailyATR*dailyWidth
	gapOpen := entryPrice - 3*dailyWidth // well below the level: a real gap
	candles = append(candles, bt.Candle{Time: mondayNoon, Open: gapOpen, High: gapOpen, Low: gapOpen - 1, Close: gapOpen - 0.5})

	res := bt.Run(s, candles, engineDailyCandles(mondayNoon, 40, dailyWidth, dailyWidth/10), nil, engineCfg)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if !tr.EntryTime.Equal(entryBarTime) || math.Abs(tr.EntryPrice-entryPrice) > 1e-9 {
		t.Fatalf("entry at %v by %v, want %v by %v: the cross did not fire on the designed bar",
			tr.EntryTime, tr.EntryPrice, entryBarTime, entryPrice)
	}
	if tr.Reason != "SL" || math.Abs(tr.ATR-dailyWidth) > 1e-9 {
		t.Fatalf("exit %q with ATR %v, want SL with ATR %v", tr.Reason, tr.ATR, dailyWidth)
	}
	if want := math.Min(level, gapOpen); math.Abs(tr.ExitPrice-want) > 1e-9 {
		t.Fatalf("ExitPrice = %v, want %v = min(level %v, open %v)", tr.ExitPrice, want, level, gapOpen)
	}
}

// MID anchors on the EntryTime the engine stamps: with BelowMidBars 3 the trade closes at the
// close of the third bar after the entry bar.
func TestEngineMIDClosesOnTheNthBarAfterEntry(t *testing.T) {
	p := DefaultParams()
	p.BelowMidBars = 3
	p.StopDailyATR = 0 // isolate MID: -1% bars would otherwise hit a 0.5 ATR stop first
	s := NewWithParams("TEST", p)
	l := s.Lookback()
	closes := crossCloses(l, 0.0008, p.RSIPeriod, midLine, l)
	q := closes[len(closes)-1]
	for i := 0; i < 3; i++ {
		q *= 0.99
		closes = append(closes, q)
	}
	candles := engineCandles(closes, mondayNoon)

	res := bt.Run(s, candles, engineDailyCandles(mondayNoon, 40, dailyWidth, dailyWidth/10), nil, engineCfg)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	entryBar, last := candles[len(candles)-4], candles[len(candles)-1]
	if !tr.EntryTime.Equal(entryBar.Time) {
		t.Fatalf("EntryTime = %v, want %v", tr.EntryTime, entryBar.Time)
	}
	if tr.Reason != "MID" || !tr.ExitTime.Equal(last.Time) || math.Abs(tr.ExitPrice-last.Close) > 1e-9 {
		t.Fatalf("exit %q at %v by %v, want MID at %v by close %v", tr.Reason, tr.ExitTime, tr.ExitPrice, last.Time, last.Close)
	}
}
