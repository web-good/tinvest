// engine_test.go runs drop_bounce through the real backtest engine to pin the fills it relies on:
// entry at the next bar's open, the daily-ATR stop from the fill price filled at the gap open overnight, and
// the fill-bar profit at the close.
package core

import (
	"math"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

func toCandles(bars []bar) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: b.t, Open: b.o, High: b.h, Low: b.l, Close: b.c}
	}
	return out
}

// dropBars is the oversold drop (the drop bar starts at dropAt) led by flat bars and a downtrend:
// the engine decides only from bar Lookback()−1 on, so the flat lead puts the drop past it.
func dropBars(dropAt string) []bar {
	closes := dropAfter(append(flatN(NewWithParams("TEST", step15, immediate()).Lookback(), 100), downtrend()...), 0.6)
	return path(at(dropAt).Add(-time.Duration(len(closes)-1)*step15).Format("2006-01-02 15:04"), closes)
}

// runEngine runs drop_bounce through the engine with entries at the next bar's open and dailies
// of range fixtureATR before the first bar, so the stop sits fixtureATR/2 below the fill.
func runEngine(bars []bar) bt.Result {
	return bt.Run(NewWithParams("TEST", step15, immediate()), toCandles(bars), toCandles(dailies(bars[0].t, 20, fixtureATR)), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
}

// runAfterDrop runs the engine over the oversold drop at Monday 13:00 followed by after(signal
// close) and returns the single trade with its fill price. The fill bar (the first of after) must
// open away from the signal close so the entry check tells the next open from the signal close.
func runAfterDrop(t *testing.T, after func(close float64) []bar) (bt.Trade, float64) {
	t.Helper()
	bars := dropBars("2026-09-14 13:00")
	next := after(bars[len(bars)-1].c)
	fill := next[0].o
	res := runEngine(append(bars, next...))
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-fill) > eps || !tr.EntryTime.Equal(at("2026-09-14 13:15")) {
		t.Fatalf("entry %v @ %v, want %v @ 13:15 (next bar's open)", tr.EntryPrice, tr.EntryTime, fill)
	}
	return tr, fill
}

func TestEngineOvernightGapStop(t *testing.T) {
	var gap float64
	tr, fill := runAfterDrop(t, func(c float64) []bar {
		f := c * 1.001
		gap = f * 0.97
		return []bar{
			{at("2026-09-14 13:15"), f, f, f, f},
			{at("2026-09-15 07:00"), gap, gap, gap * 0.999, gap},
		}
	})
	if stop := fill - fixtureATR/2; gap >= stop {
		t.Fatalf("fixture: gap open %v must be below the stop %v", gap, stop)
	}
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-gap) > 1e-6 || !tr.ExitTime.Equal(at("2026-09-15 07:00")) {
		t.Fatalf("exit %q @ %v %v, want SL at the gap open %v next morning", tr.Reason, tr.ExitPrice, tr.ExitTime, gap)
	}
}

func TestEngineProfitOnFillBar(t *testing.T) {
	tr, fill := runAfterDrop(t, func(c float64) []bar {
		f := c * 1.001
		return []bar{{at("2026-09-14 13:15"), f, f * 1.012, f, f * 1.012}}
	})
	if tr.Reason != "PROFIT" || math.Abs(tr.ExitPrice-fill*1.012) > 1e-6 {
		t.Fatalf("exit %q @ %v, want PROFIT at the fill bar's close %v", tr.Reason, tr.ExitPrice, fill*1.012)
	}
}

// TestEngineFillsAfterFirstGreen: with the green confirmation on, the fill is the open of the bar
// after the first green bar, not after the signal bar.
func TestEngineFillsAfterFirstGreen(t *testing.T) {
	bars := dropBars("2026-09-14 13:00")
	c := bars[len(bars)-1].c
	r, g, f := c*0.999, c*1.001, c*1.002
	bars = append(bars,
		bar{at("2026-09-14 13:15"), c, c, r, r},
		bar{at("2026-09-14 13:30"), r, g, r, g},
		bar{at("2026-09-14 13:45"), f, f * 1.012, f, f * 1.012}, // the fill bar closes the trade on PROFIT
	)
	res := bt.Run(NewWithParams("TEST", step15, DefaultParams()), toCandles(bars), toCandles(dailies(bars[0].t, 20, fixtureATR)), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	if tr := res.Trades[0]; math.Abs(tr.EntryPrice-f) > eps || !tr.EntryTime.Equal(at("2026-09-14 13:45")) {
		t.Fatalf("entry %v @ %v, want %v @ 13:45 (open after the green bar)", tr.EntryPrice, tr.EntryTime, f)
	}
}

// TestEngineDropsSignalWhenNextSlotMissing pins entry rules 4 and 9 against gaps in the data: a signal
// on the 23:30 bar may fill only at 23:45; when that candle is missing the next candle is the
// next morning and the signal is dropped instead of carrying over.
func TestEngineDropsSignalWhenNextSlotMissing(t *testing.T) {
	bars := dropBars("2026-09-14 23:30")
	c := bars[len(bars)-1].c
	bars = append(bars,
		bar{at("2026-09-15 07:00"), c, c, c, c},
		bar{at("2026-09-15 07:15"), c, c, c, c},
	)
	res := runEngine(bars)
	if len(res.Trades) != 0 || res.BarsInMarket != 0 {
		t.Fatalf("trades = %d, barsInMarket = %d, want no position: %+v", len(res.Trades), res.BarsInMarket, res.Trades)
	}
}
