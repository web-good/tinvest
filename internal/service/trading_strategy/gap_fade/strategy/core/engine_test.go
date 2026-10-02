// engine_test.go runs gap_fade through the real backtest engine to pin the fills the strategy
// relies on: TP at max(target, open), SL at min(stop, open), EOD at the bar's close, and the
// levels frozen on the position from the entry signal.
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
	h, l, c, times := dailyBars(at("2026-09-14 00:00"), 40, 2.0, 0.2)
	out := make([]bt.Candle, len(c))
	for i := range c {
		out[i] = bt.Candle{Time: times[i], Open: c[i], High: h[i], Low: l[i], Close: c[i]}
	}
	return out
}

// history is Thursday and Friday flat at 100: 68 bars, enough for the 64-bar lookback.
func history() []bar {
	out := session("2026-09-10 07:00", "2026-09-10 23:30", 100)
	return append(out, session("2026-09-11 07:00", "2026-09-11 23:30", 100)...)
}

func runOne(t *testing.T, monday []bar) bt.Trade {
	t.Helper()
	bars := append(history(), monday...)
	res := bt.Run(NewWithParams("TEST", DefaultParams()), toCandles(bars), engineDaily(), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-98.8) > eps || !tr.EntryTime.Equal(at("2026-09-14 07:00")) {
		t.Fatalf("entry %v @ %v, want 98.8 @ Monday 07:00", tr.EntryPrice, tr.EntryTime)
	}
	if dayKey(tr.ExitTime) != dayKey(tr.EntryTime) {
		t.Fatalf("exit %v is on another date than entry %v", tr.ExitTime, tr.EntryTime)
	}
	return tr
}

func TestEngineTargetFillsAtLevelOrBetterOpen(t *testing.T) {
	tr := runOne(t, []bar{
		mondayFirst,
		{"2026-09-14 07:30", 99.0, 99.4, 98.9, 99.3},
		{"2026-09-14 08:00", 99.6, 100.3, 99.5, 100.1},
		{"2026-09-14 08:30", 100.1, 100.2, 100.0, 100.1},
	})
	if tr.Reason != "TP" || math.Abs(tr.ExitPrice-100) > eps {
		t.Fatalf("exit %q @ %v, want TP @ 100", tr.Reason, tr.ExitPrice)
	}
}

func TestEngineStopGapFillsAtWorseOpen(t *testing.T) {
	tr := runOne(t, []bar{
		mondayFirst,
		{"2026-09-14 07:30", 97.0, 97.2, 96.8, 97.1},
		{"2026-09-14 08:00", 97.1, 97.2, 97.0, 97.1},
	})
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-97.0) > eps {
		t.Fatalf("exit %q @ %v, want SL @ 97.0 (gapped open below the 97.8 stop)", tr.Reason, tr.ExitPrice)
	}
}

func TestEngineEODClosesAtMainSessionEnd(t *testing.T) {
	monday := append([]bar{mondayFirst}, session("2026-09-14 07:30", "2026-09-14 18:00", 99)...)
	monday = append(monday, bar{"2026-09-14 18:30", 99, 99.3, 98.9, 99.2})
	monday = append(monday, session("2026-09-14 19:00", "2026-09-14 19:30", 99)...)
	tr := runOne(t, monday)
	if tr.Reason != "EOD" || math.Abs(tr.ExitPrice-99.2) > eps || !tr.ExitTime.Equal(at("2026-09-14 18:30")) {
		t.Fatalf("exit %q @ %v %v, want EOD @ 99.2 at 18:30", tr.Reason, tr.ExitPrice, tr.ExitTime)
	}
}

func TestEngineNextDayGuardWhenNoEODBar(t *testing.T) {
	monday := append([]bar{mondayFirst}, session("2026-09-14 07:30", "2026-09-14 15:00", 99)...)
	monday = append(monday, bar{"2026-09-15 07:00", 99.1, 99.2, 99.0, 99.1}, bar{"2026-09-15 07:30", 99.1, 99.2, 99.0, 99.1})
	bars := append(history(), monday...)
	res := bt.Run(NewWithParams("TEST", DefaultParams()), toCandles(bars), engineDaily(), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if tr.Reason != "EOD" || !tr.ExitTime.Equal(at("2026-09-15 07:00")) || math.Abs(tr.ExitPrice-99.1) > eps {
		t.Fatalf("exit %q @ %v %v, want EOD @ 99.1 on Tuesday's first bar", tr.Reason, tr.ExitPrice, tr.ExitTime)
	}
}
