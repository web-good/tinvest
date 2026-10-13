// engine_test.go runs drop_bounce through the real backtest engine on 1h bars to pin what it
// relies on: the day extent and dailies reaching the strategy, entry at the next bar's open, the
// daily-ATR stop filled at the gap open overnight, the RSI exit, and a signal dropped when the
// next slot is missing.
package core

import (
	"math"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

// engineParams opens the day gate wide: the fixture's trend runs far beyond one daily ATR within
// a day, and the gate itself is unit-tested in core_test.go.
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
