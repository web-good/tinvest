// engine_test.go runs drop_bounce through the real backtest engine on 1h bars to pin what it
// relies on: the day extent (the day gate) and dailies reaching the strategy, entry at the next bar's open, the
// daily-ATR stop filled at the gap open overnight, the RSI exit on a red MACD histogram, the armed MACD exit, and a signal dropped when the
// next slot is missing.
package core

import (
	"math"
	"strings"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

// engineParams opens the day gate wide: the fixture's trend runs far beyond one daily ATR within
// a day, and the gate itself is unit-tested in core_test.go. TestEngineDayGateUsesEngineExtent
// narrows it to prove the engine-supplied extent decides.
func engineParams() Params {
	p := entryParams()
	p.DaySpentATR = 100
	return p
}

// toCandles converts bars to candles.
func toCandles(bars []bar) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: b.t, Open: b.o, High: b.h, Low: b.l, Close: b.c, Volume: 10}
	}
	return out
}

// signalBars is Lookback flat bars at 100, then trendCloses(6) scaled from 100, with the last
// (signal) bar starting at end. The engine decides only from bar Lookback()−1 on.
func signalBars(end string) []bar {
	lead := NewWithParams("TEST", stepH1, engineParams()).Lookback()
	closes := append(flatN(lead, 100), trendCloses(6)[1:]...)
	return endingAt(end, closes)
}

func runEngine(bars []bar) bt.Result {
	return bt.Run(NewWithParams("TEST", stepH1, engineParams()), toCandles(bars),
		toCandles(dailies(bars[0].t, 20, fixtureATR)), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
}

func TestEngineEntryAndRSIExit(t *testing.T) {
	bars := signalBars("2026-09-16 12:00")
	last := bars[len(bars)-1].c
	f := last * 1.001
	bars = append(bars, bar{at("2026-09-16 13:00"), f, f, f, f})
	for i, c := range steps(f, 6, 2) {
		bars = append(bars, bar{at("2026-09-16 14:00").Add(time.Duration(i) * stepH1), c / 1.02, c, c / 1.02, c})
	}
	res := runEngine(bars)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-f) > eps || !tr.EntryTime.Equal(at("2026-09-16 13:00")) {
		t.Fatalf("entry %v @ %v, want %v @ 13:00 (next bar's open)", tr.EntryPrice, tr.EntryTime, f)
	}
	if tr.Reason != "RSI" {
		t.Fatalf("exit reason %q, want RSI", tr.Reason)
	}
	// The steady +2% tail follows a drop: the histogram is still red at the cross, so it sells at once.
	if !strings.Contains(tr.ExitReason, "гистограмма MACD") {
		t.Fatalf("exit reason text %q lacks the histogram", tr.ExitReason)
	}
}

// TestEngineArmedMACDExit buys the same dip, then climbs a zigzag (+0.8%, −0.4%: RSI(6) stays
// below 70) until the histogram is green, spikes through RSI 70 (arming the exit, no sale) and
// falls: the first green→red flip sells with reason MACD.
func TestEngineArmedMACDExit(t *testing.T) {
	bars := signalBars("2026-09-16 12:00")
	last := bars[len(bars)-1].c
	f := last * 1.001
	bars = append(bars, bar{at("2026-09-16 13:00"), f, f, f, f})
	entryIdx := len(bars) - 1
	p := f
	var tail []float64
	for i := 0; i < 20; i++ {
		if i%2 == 0 {
			p *= 1.008
		} else {
			p *= 0.996
		}
		tail = append(tail, p)
	}
	tail = append(tail, steps(p, 3, 1.5)...)
	tail = append(tail, steps(tail[len(tail)-1], 10, -0.7)...)
	prev := f
	for i, c := range tail {
		bars = append(bars, bar{at("2026-09-16 14:00").Add(time.Duration(i) * stepH1), prev, max(prev, c), min(prev, c), c})
		prev = c
	}
	// Premise: the first RSI cross of 70 after the fill has a green histogram, and a flip follows.
	h := histOf(bars)
	cross := firstCross(bars, 6, 70, entryIdx)
	if cross < 0 || h[cross] < 0.05 {
		t.Fatalf("fixture: first cross %d must carry a clearly green histogram, hist=%v", cross, h)
	}
	flip := -1
	for k := cross + 1; k < len(bars); k++ {
		if h[k-1] >= 0 && h[k] < 0 {
			flip = k
			break
		}
	}
	if flip < 0 {
		t.Fatal("fixture: no green→red flip after the cross")
	}
	res := runEngine(bars)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if !tr.EntryTime.Equal(at("2026-09-16 13:00")) || math.Abs(tr.EntryPrice-f) > eps {
		t.Fatalf("entry %v @ %v, want %v @ 13:00", tr.EntryPrice, tr.EntryTime, f)
	}
	if tr.Reason != "MACD" || !tr.ExitTime.Equal(bars[flip].t) {
		t.Fatalf("exit %q at %v, want MACD at the flip bar %v", tr.Reason, tr.ExitTime, bars[flip].t)
	}
}

// TestEngineDayGateUsesEngineExtent runs the TestEngineEntryAndRSIExit fixture twice: with the
// gate wide it trades (control), with a tight gate the extent the engine builds from the day's
// bars up to the signal bar already exceeds it, so no trade enters on 2026-09-16.
func TestEngineDayGateUsesEngineExtent(t *testing.T) {
	bars := signalBars("2026-09-16 12:00")
	last := bars[len(bars)-1].c
	f := last * 1.001
	bars = append(bars, bar{at("2026-09-16 13:00"), f, f, f, f})
	for i, c := range steps(f, 6, 2) {
		bars = append(bars, bar{at("2026-09-16 14:00").Add(time.Duration(i) * stepH1), c / 1.02, c, c / 1.02, c})
	}
	run := func(spent float64) bt.Result {
		p := engineParams()
		p.DaySpentATR = spent
		return bt.Run(NewWithParams("TEST", stepH1, p), toCandles(bars),
			toCandles(dailies(bars[0].t, 20, fixtureATR)), nil,
			bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	}
	if n := len(run(100).Trades); n != 1 {
		t.Fatalf("control (DaySpentATR=100) trades = %d, want 1", n)
	}
	res := run(0.05)
	for _, tr := range res.Trades {
		if tr.EntryTime.In(mskLoc).Format("2006-01-02") == "2026-09-16" {
			t.Fatalf("tight day gate still entered on 2026-09-16: %+v", tr)
		}
	}
}

func TestEngineOvernightGapStop(t *testing.T) {
	bars := signalBars("2026-09-16 21:00")
	f := bars[len(bars)-1].c
	gap := f - 3*fixtureATR
	bars = append(bars,
		bar{at("2026-09-16 22:00"), f, f, f, f},
		bar{at("2026-09-17 07:00"), gap, gap, gap * 0.999, gap},
	)
	res := runEngine(bars)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-gap) > eps {
		t.Fatalf("exit %q @ %v, want SL at the gap open %v (stop %v)", tr.Reason, tr.ExitPrice, gap, f-fixtureATR)
	}
}

func TestEngineDropsSignalWhenNextSlotMissing(t *testing.T) {
	bars := signalBars("2026-09-16 12:00")
	f := bars[len(bars)-1].c
	// 13:00 is missing: the next bar is 15:00.
	bars = append(bars, bar{at("2026-09-16 15:00"), f, f, f, f})
	if res := runEngine(bars); len(res.Trades) != 0 {
		t.Fatalf("trades = %d, want 0 (fill slot missing): %+v", len(res.Trades), res.Trades)
	}
}
