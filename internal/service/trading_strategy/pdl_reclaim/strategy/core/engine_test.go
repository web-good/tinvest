// engine_test.go runs pdl_reclaim through the real backtest engine to pin the fills the strategy
// relies on: entry at the next bar's open, TP at max(target, open), SL at min(stop from the fill,
// open), EOD at the 23:45 bar's close, and a second trade on the same day after an exit.
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

// history is Thursday and Friday 07:00–23:45 flat at 100: 404 bars, more than the 304-bar lookback.
func history() []bar {
	return append(flat("2026-09-10 07:00", "2026-09-10 23:45", 100), flat("2026-09-11 07:00", "2026-09-11 23:45", 100)...)
}

func run(t *testing.T, monday []bar) []bt.Trade {
	t.Helper()
	bars := append(history(), monday...)
	res := bt.Run(NewWithParams("TEST", 5*time.Minute, quiet()), toCandles(bars), engineDaily(), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	return res.Trades
}

// mondayToA is Monday flat at 100 until 09:55, then the entry-A bar at 10:00.
func mondayToA() []bar { return append(flat("2026-09-14 07:00", "2026-09-14 09:55", 100), barA) }

func TestEngineEntryAAndTarget(t *testing.T) {
	mon := append(mondayToA(),
		bar{"2026-09-14 10:05", 99.35, 99.6, 99.3, 99.5},
		bar{"2026-09-14 10:10", 99.5, 101.2, 99.5, 101.0})
	mon = append(mon, flat("2026-09-14 10:15", "2026-09-14 10:30", 101)...)
	trades := run(t, mon)
	if len(trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(trades), trades)
	}
	tr := trades[0]
	if math.Abs(tr.EntryPrice-99.35) > eps || !tr.EntryTime.Equal(at("2026-09-14 10:05")) {
		t.Fatalf("entry %v @ %v, want 99.35 @ 10:05", tr.EntryPrice, tr.EntryTime)
	}
	if tr.Reason != "TP" || math.Abs(tr.ExitPrice-101) > eps {
		t.Fatalf("exit %q @ %v, want TP @ 101", tr.Reason, tr.ExitPrice)
	}
}

// slDay: entry A fills at 99.35 (stop 98.35), the 10:10 bar gaps to 98.0 — SL at the open.
func slDay() []bar {
	mon := append(mondayToA(),
		bar{"2026-09-14 10:05", 99.35, 99.4, 99.3, 99.35},
		bar{"2026-09-14 10:10", 98.0, 98.1, 97.9, 98.0})
	return append(mon, flat("2026-09-14 10:15", "2026-09-14 10:55", 98)...)
}

func TestEngineStopGapFillsAtOpen(t *testing.T) {
	trades := run(t, slDay())
	if len(trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(trades), trades)
	}
	if tr := trades[0]; tr.Reason != "SL" || math.Abs(tr.ExitPrice-98.0) > eps {
		t.Fatalf("exit %q @ %v, want SL @ 98.0", tr.Reason, tr.ExitPrice)
	}
}

func TestEngineReentersSameDayAndExitsAtEOD(t *testing.T) {
	mon := append(slDay(),
		bar{"2026-09-14 11:00", 98.9, 98.95, 98.5, 98.6},   // red under PDL
		bar{"2026-09-14 11:05", 98.55, 98.95, 98.4, 98.92}, // engulfing — entry B
		bar{"2026-09-14 11:10", 98.9, 98.95, 98.85, 98.9})
	mon = append(mon, flat("2026-09-14 11:15", "2026-09-14 23:45", 98.9)...)
	trades := run(t, mon)
	if len(trades) != 2 {
		t.Fatalf("trades = %d, want 2: %+v", len(trades), trades)
	}
	tr := trades[1]
	if !tr.EntryTime.Equal(at("2026-09-14 11:10")) || math.Abs(tr.EntryPrice-98.9) > eps {
		t.Fatalf("second entry %v @ %v, want 98.9 @ 11:10", tr.EntryPrice, tr.EntryTime)
	}
	if tr.Reason != "EOD" || !tr.ExitTime.Equal(at("2026-09-14 23:45")) {
		t.Fatalf("second exit %q @ %v, want EOD @ 23:45", tr.Reason, tr.ExitTime)
	}
}

// longHistory is Tuesday–Friday 07:00–23:55 flat at 100: 816 bars, more than the lookback with
// half-hours on (6·102 + 202 = 814).
func longHistory() []bar {
	var out []bar
	for _, d := range []string{"2026-09-08", "2026-09-09", "2026-09-10", "2026-09-11"} {
		out = append(out, flat(d+" 07:00", d+" 23:55", 100)...)
	}
	return out
}

// Morning trade on 5m (A at 09:00, target), then a trade on half-hours (A on 10:00–10:30, bought
// at 10:30, sold at EOD).
func TestEngineMorning5mThenHalfHour(t *testing.T) {
	p := quiet()
	p.SwitchHour = 10
	p.HourRSIMax = 0 // the flat history leaves the hourly RSI at 50: the test is about the timeframes
	mon := flat("2026-09-14 07:00", "2026-09-14 08:55", 100)
	mon = append(mon,
		bar{"2026-09-14 09:00", 99.2, 99.5, 98.8, 99.3},
		bar{"2026-09-14 09:05", 99.35, 99.6, 99.3, 99.5},
		bar{"2026-09-14 09:10", 99.5, 101.2, 99.5, 101.0})
	mon = append(mon, flat("2026-09-14 09:15", "2026-09-14 09:55", 101)...)
	mon = append(mon, halfA()...)
	mon = append(mon, flat("2026-09-14 10:30", "2026-09-14 23:45", 99.3)...)
	bars := append(longHistory(), mon...)
	res := bt.Run(NewWithParams("TEST", 5*time.Minute, p), toCandles(bars), engineDaily(), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	if len(res.Trades) != 2 {
		t.Fatalf("trades = %d, want 2: %+v", len(res.Trades), res.Trades)
	}
	if tr := res.Trades[0]; !tr.EntryTime.Equal(at("2026-09-14 09:05")) || tr.Reason != "TP" {
		t.Fatalf("first trade %v %q, want 09:05 TP", tr.EntryTime, tr.Reason)
	}
	tr := res.Trades[1]
	if !tr.EntryTime.Equal(at("2026-09-14 10:30")) || math.Abs(tr.EntryPrice-99.3) > eps || tr.Reason != "EOD" {
		t.Fatalf("second trade %v @ %v %q, want 10:30 @ 99.3 EOD", tr.EntryTime, tr.EntryPrice, tr.Reason)
	}
}
