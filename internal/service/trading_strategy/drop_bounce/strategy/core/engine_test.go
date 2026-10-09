// engine_test.go runs drop_bounce through the real backtest engine to pin the fills it relies on:
// entry at the next bar's open, the stop from the fill price filled at the gap open overnight, and
// the fill-bar profit at the close.
package core

import (
	"math"
	"testing"

	bt "tinvest/internal/domain/backtest"
)

func toCandles(bars []bar) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: b.t, Open: b.o, High: b.h, Low: b.l, Close: b.c}
	}
	return out
}

// runAfterDrop runs the engine over the oversold drop at Monday 13:00 followed by after and
// returns the single trade. 50 flat bars lead the downtrend: the engine decides only from bar
// Lookback()−1 (63) on, so the drop must sit past it (71 bars, the drop at index 70).
func runAfterDrop(t *testing.T, after func(entry float64) []bar) (bt.Trade, float64) {
	t.Helper()
	bars := path("2026-09-13 19:30", dropAfter(append(flatN(50, 100), downtrend()...), 0.6))
	entry := bars[len(bars)-1].c
	bars = append(bars, after(entry)...)
	res := bt.Run(NewWithParams("TEST", DefaultParams()), toCandles(bars), nil, nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-entry) > eps || !tr.EntryTime.Equal(at("2026-09-14 13:15")) {
		t.Fatalf("entry %v @ %v, want %v @ 13:15 (next bar's open)", tr.EntryPrice, tr.EntryTime, entry)
	}
	return tr, entry
}

func TestEngineOvernightGapStop(t *testing.T) {
	tr, entry := runAfterDrop(t, func(e float64) []bar {
		gap := e * 0.97
		return []bar{
			{at("2026-09-14 13:15"), e, e, e, e},
			{at("2026-09-15 07:00"), gap, gap, gap * 0.999, gap},
		}
	})
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-entry*0.97) > 1e-6 || !tr.ExitTime.Equal(at("2026-09-15 07:00")) {
		t.Fatalf("exit %q @ %v %v, want SL at the gap open %v next morning", tr.Reason, tr.ExitPrice, tr.ExitTime, entry*0.97)
	}
}

func TestEngineProfitOnFillBar(t *testing.T) {
	tr, entry := runAfterDrop(t, func(e float64) []bar {
		c := e * 1.012
		return []bar{{at("2026-09-14 13:15"), e, c, e, c}}
	})
	if tr.Reason != "PROFIT" || math.Abs(tr.ExitPrice-entry*1.012) > 1e-6 {
		t.Fatalf("exit %q @ %v, want PROFIT at the fill bar's close %v", tr.Reason, tr.ExitPrice, entry*1.012)
	}
}
