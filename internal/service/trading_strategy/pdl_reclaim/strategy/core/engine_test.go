// engine_test.go runs pdl_reclaim through the real backtest engine to pin the fills the strategy
// relies on: entry at the next bar's open after the reclaim bar, TP at the target, SL at
// min(stop from the fill, open), a position carried overnight, and a second trade on the same day
// after an exit.
package core

import (
	"math"
	"testing"

	bt "tinvest/internal/domain/backtest"
)

func toCandles(bars []bar) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: at(b.t), Open: b.o, High: b.h, Low: b.l, Close: b.c}
	}
	return out
}

func engineDaily() []bt.Candle {
	h, l, c, times := dailyBars(at("2026-09-14 00:00"), 40)
	out := make([]bt.Candle, len(c))
	for i := range c {
		out[i] = bt.Candle{Time: times[i], Open: c[i], High: h[i], Low: l[i], Close: c[i]}
	}
	return out
}

// history is Monday to Friday 07:00–23:45 flat at 100: 340 bars, more than the 272-bar lookback.
func history() []bar {
	var out []bar
	for _, d := range []string{"2026-09-07", "2026-09-08", "2026-09-09", "2026-09-10", "2026-09-11"} {
		out = append(out, flat(d+" 07:00", d+" 23:45", 100)...)
	}
	return out
}

func run(t *testing.T, monday []bar) []bt.Trade {
	t.Helper()
	bars := append(history(), monday...)
	res := bt.Run(NewWithParams("TEST", step, DefaultParams()), toCandles(bars), engineDaily(), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	return res.Trades
}

// mondayToTrigger is Monday flat at 100 until 09:15, then pdlDay up to its reclaim at 10:15.
func mondayToTrigger() []bar {
	return append(flat("2026-09-14 07:00", "2026-09-14 09:15", 100), upTo(reclaimAt)...)
}

func TestEngineReclaimAndTarget(t *testing.T) {
	mon := append(mondayToTrigger(),
		bar{"2026-09-14 10:30", 99.25, 99.4, 99.2, 99.3},
		bar{"2026-09-14 10:45", 99.3, 100.2, 99.3, 100.1})
	mon = append(mon, flat("2026-09-14 11:00", "2026-09-14 11:30", 100.1)...)
	trades := run(t, mon)
	if len(trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(trades), trades)
	}
	tr := trades[0]
	if math.Abs(tr.EntryPrice-99.25) > eps || !tr.EntryTime.Equal(at("2026-09-14 10:30")) {
		t.Fatalf("entry %v @ %v, want 99.25 @ 10:30", tr.EntryPrice, tr.EntryTime)
	}
	if tr.Reason != "TP" || math.Abs(tr.ExitPrice-100) > eps {
		t.Fatalf("exit %q @ %v, want TP @ 100 (mid of 99..101)", tr.Reason, tr.ExitPrice)
	}
}

// The fill at 99.25 sets the stop at 98.25; the 10:45 bar gaps to 98.0 — SL at the open.
func TestEngineStopGapFillsAtOpen(t *testing.T) {
	mon := append(mondayToTrigger(),
		bar{"2026-09-14 10:30", 99.25, 99.3, 99.2, 99.25},
		bar{"2026-09-14 10:45", 98.0, 98.1, 97.9, 98.0})
	mon = append(mon, flat("2026-09-14 11:00", "2026-09-14 11:30", 98)...)
	trades := run(t, mon)
	if len(trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(trades), trades)
	}
	if tr := trades[0]; tr.Reason != "SL" || math.Abs(tr.ExitPrice-98.0) > eps {
		t.Fatalf("exit %q @ %v, want SL @ 98.0", tr.Reason, tr.ExitPrice)
	}
}

// After the target the price settles at 100 and the same reclaim repeats from 15:00 (reclaim at
// 15:45): a second trade fills at 16:00, holds past 23:45 and overnight, and takes the target (100)
// on Tuesday morning.
func TestEngineReentersSameDayAndCarriesOvernight(t *testing.T) {
	mon := append(mondayToTrigger(),
		bar{"2026-09-14 10:30", 99.25, 99.4, 99.2, 99.3},
		bar{"2026-09-14 10:45", 99.3, 100.2, 99.3, 100})
	mon = append(mon, flat("2026-09-14 11:00", "2026-09-14 14:45", 100)...)
	mon = append(mon, path("2026-09-14 15:00", 100, 99.5, 98.7, 98.6, 99.2, 99.3, 99.4, 99.5, 99.6, 99.7)...)
	mon = append(mon, flat("2026-09-14 17:15", "2026-09-14 23:45", 99.7)...)
	mon = append(mon, flat("2026-09-15 07:00", "2026-09-15 09:45", 99.7)...)
	mon = append(mon, bar{"2026-09-15 10:00", 99.7, 100.1, 99.7, 100}, bar{"2026-09-15 10:15", 100, 100, 100, 100})
	trades := run(t, mon)
	if len(trades) != 2 {
		t.Fatalf("trades = %d, want 2: %+v", len(trades), trades)
	}
	tr := trades[1]
	if !tr.EntryTime.Equal(at("2026-09-14 16:00")) || math.Abs(tr.EntryPrice-99.2) > eps {
		t.Fatalf("second entry %v @ %v, want 99.2 @ 16:00", tr.EntryPrice, tr.EntryTime)
	}
	if tr.Reason != "TP" || !tr.ExitTime.Equal(at("2026-09-15 10:00")) || math.Abs(tr.ExitPrice-100) > eps {
		t.Fatalf("second exit %q %v @ %v, want TP 100 @ Tuesday 10:00", tr.Reason, tr.ExitPrice, tr.ExitTime)
	}
}
