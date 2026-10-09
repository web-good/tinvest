// engine_test.go runs drop_bounce through the real backtest engine to pin the fills it relies on:
// entry at the next bar's open with the levels frozen from the signal bar, TP/SL at their levels,
// EOD at the 17:00 bar's close.
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

func runOne(t *testing.T, after []bar) bt.Trade {
	t.Helper()
	bars := append(flat("2026-09-14 09:00", "2026-09-14 09:55", 100), drop("2026-09-14 10:00", 1.0))
	bars = append(bars, after...)
	res := bt.Run(NewWithParams("TEST", DefaultParams()), toCandles(bars), nil, nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-99.1) > eps || !tr.EntryTime.Equal(at("2026-09-14 10:05")) {
		t.Fatalf("entry %v @ %v, want 99.1 @ 10:05 (next bar's open)", tr.EntryPrice, tr.EntryTime)
	}
	return tr
}

func TestEngineTarget(t *testing.T) {
	tr := runOne(t, []bar{
		{"2026-09-14 10:05", 99.1, 99.3, 98.9, 99.2},
		{"2026-09-14 10:10", 99.2, 99.6, 99.1, 99.5},
	})
	if tr.Reason != "TP" || math.Abs(tr.ExitPrice-99.5) > eps {
		t.Fatalf("exit %q @ %v, want TP @ 99.5", tr.Reason, tr.ExitPrice)
	}
}

func TestEngineStop(t *testing.T) {
	tr := runOne(t, []bar{
		{"2026-09-14 10:05", 99.1, 99.2, 98.4, 98.6},
	})
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-98.5) > eps {
		t.Fatalf("exit %q @ %v, want SL @ 98.5", tr.Reason, tr.ExitPrice)
	}
}

func TestEngineEODAtFive(t *testing.T) {
	after := append([]bar{{"2026-09-14 10:05", 99.1, 99.2, 99.0, 99.1}}, flat("2026-09-14 10:10", "2026-09-14 16:55", 99.1)...)
	after = append(after, bar{"2026-09-14 17:00", 99.1, 99.2, 99.0, 99.2})
	tr := runOne(t, after)
	if tr.Reason != "EOD" || math.Abs(tr.ExitPrice-99.2) > eps || !tr.ExitTime.Equal(at("2026-09-14 17:00")) {
		t.Fatalf("exit %q @ %v %v, want EOD @ 99.2 at 17:00", tr.Reason, tr.ExitPrice, tr.ExitTime)
	}
}

func TestEngineTimeExit(t *testing.T) {
	p := DefaultParams()
	p.TimeExit = 1
	bars := append(flat("2026-09-14 09:00", "2026-09-14 09:55", 100), drop("2026-09-14 10:00", 1.0))
	bars = append(bars,
		bar{"2026-09-14 10:05", 99.1, 99.2, 99.0, 99.1},
		bar{"2026-09-14 10:10", 99.1, 99.2, 99.0, 99.1},
		bar{"2026-09-14 10:15", 99.1, 99.2, 99.0, 99.15},
		bar{"2026-09-14 10:20", 99.1, 99.2, 99.0, 99.1},
	)
	res := bt.Run(NewWithParams("TEST", p), toCandles(bars), nil, nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if tr.Reason != "TIME" || !tr.ExitTime.Equal(at("2026-09-14 10:15")) || math.Abs(tr.ExitPrice-99.15) > eps {
		t.Fatalf("exit %q @ %v %v, want TIME @ 99.15 at 10:15", tr.Reason, tr.ExitPrice, tr.ExitTime)
	}
}

// TestEngineConfirmEntersAfterConfirmation pins the confirmed fill: drop at 10:00, green bar at
// 10:05 closing above the drop close, entry at the 10:10 open with the drop's levels.
func TestEngineConfirmEntersAfterConfirmation(t *testing.T) {
	p := DefaultParams()
	p.ConfirmBar = 1
	bars := append(flat("2026-09-14 09:00", "2026-09-14 09:55", 100), drop("2026-09-14 10:00", 1.0))
	bars = append(bars,
		bar{"2026-09-14 10:05", 99.0, 99.2, 98.95, 99.1},
		bar{"2026-09-14 10:10", 99.12, 99.3, 99.1, 99.2},
		bar{"2026-09-14 10:15", 99.2, 99.6, 99.1, 99.5},
	)
	res := bt.Run(NewWithParams("TEST", p), toCandles(bars), nil, nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true, SkipFillOutsideLevels: true})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-99.12) > eps || !tr.EntryTime.Equal(at("2026-09-14 10:10")) {
		t.Fatalf("entry %v @ %v, want 99.12 @ 10:10", tr.EntryPrice, tr.EntryTime)
	}
	if tr.Reason != "TP" || math.Abs(tr.ExitPrice-99.5) > eps {
		t.Fatalf("exit %q @ %v, want TP @ 99.5", tr.Reason, tr.ExitPrice)
	}
}
