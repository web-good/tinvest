package core

import (
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

// Warmup is deliberately neutral: no middle bar is green, so only the explicit
// formation can generate a trade. This exercises the engine's rolling window.
func engineCandles(later ...bar) []bt.Candle {
	bars := make([]bar, 16)
	for i := range bars {
		bars[i] = bar{99, 100, 98, 99}
	}
	bars = append(bars, formation()...)
	bars = append(bars, bar{103, 104, 101, 103})
	bars = append(bars, later...)
	md := snapshot(bars...)
	candles := make([]bt.Candle, len(bars))
	for i, b := range bars {
		candles[i] = bt.Candle{Time: md.Times[i], Open: b.o, High: b.h, Low: b.l, Close: b.c}
	}
	return candles
}
func runEngine(candles []bt.Candle) bt.Result {
	return bt.Run(NewWithParams("TEST", step, testParams()), candles, nil, nil, bt.Config{InitialCash: 100000, Fraction: 1, Lot: 1, EntryAtNextOpen: true})
}
func TestEngineMissingM30SlotExpiresSignal(t *testing.T) {
	candles := engineCandles(bar{104, 111, 101, 110})
	control := runEngine(candles)
	if len(control.Trades) != 1 || control.Trades[0].Reason != "TP" {
		t.Fatalf("consecutive-slot control: %+v", control)
	}
	candles[len(candles)-1].Time = candles[len(candles)-1].Time.Add(step)
	result := runEngine(candles)
	if len(result.Trades) != 0 || result.BarsInMarket != 0 {
		t.Fatalf("expired signal opened a position: %+v", result)
	}
}
func TestEngineTargetUsesActualEntry(t *testing.T) {
	// Signal close 103 implies a displayed target 107.5. Actual fill 104 and stop
	// 100 imply 104 + 1.5*(104-100) = 110; crossing 107.5 must not exit.
	candles := engineCandles(bar{104, 108, 101, 105}, bar{105, 111, 104, 110})
	result := runEngine(candles)
	if len(result.Trades) != 1 {
		t.Fatalf("trades=%+v", result.Trades)
	}
	tr := result.Trades[0]
	if tr.EntryPrice != 104 || tr.ExitPrice != 110 || tr.Reason != "TP" || !tr.ExitTime.Equal(candles[len(candles)-1].Time) {
		t.Fatalf("wrong actual-fill target: %+v", tr)
	}
}
func TestEngineGapThroughStopExitsFillBar(t *testing.T) {
	candles := engineCandles(bar{99, 104, 98, 103})
	result := runEngine(candles)
	if len(result.Trades) != 1 {
		t.Fatalf("trades=%+v", result.Trades)
	}
	tr := result.Trades[0]
	fillTime := candles[len(candles)-1].Time
	if tr.Reason != "SL" || tr.EntryPrice != 99 || tr.ExitPrice != 99 || !tr.EntryTime.Equal(fillTime) || !tr.ExitTime.Equal(fillTime) {
		t.Fatalf("want same-fill-bar SL at open: %+v", tr)
	}
}
func TestEngineStopWinsOnFillBar(t *testing.T) {
	candles := engineCandles(bar{104, 120, 99, 110})
	result := runEngine(candles)
	if len(result.Trades) != 1 || result.Trades[0].Reason != "SL" || result.Trades[0].ExitPrice != 100 {
		t.Fatalf("want stop at 100 before target: %+v", result.Trades)
	}
}

// Pin that a target reached within the fill bar is managed immediately.
func TestEngineTargetOnFillBar(t *testing.T) {
	candles := engineCandles(bar{104, 111, 101, 110})
	result := runEngine(candles)
	if len(result.Trades) != 1 {
		t.Fatalf("trades=%+v", result.Trades)
	}
	tr := result.Trades[0]
	if tr.Reason != "TP" || tr.ExitPrice != 110 || tr.ExitTime.Sub(tr.EntryTime) != time.Duration(0) {
		t.Fatalf("want fill-bar TP at 110: %+v", tr)
	}
}

func TestEngineBufferedTargetAndGapThroughStop(t *testing.T) {
	p := testParams()
	p.StopBufferATR = .5
	for _, tc := range []struct {
		name        string
		later       []bar
		entry, exit float64
		reason      string
	}{
		{"actual fill target", []bar{{104, 110, 100, 105}, {105, 112, 100, 110}}, 104, 111.5, "TP"},
		{"gap through buffered stop", []bar{{98, 104, 97, 103}}, 98, 98, "SL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candles := engineCandles(tc.later...)
			result := bt.Run(NewWithParams("TEST", step, p), candles, nil, nil, bt.Config{InitialCash: 100000, Fraction: 1, Lot: 1, EntryAtNextOpen: true})
			if len(result.Trades) != 1 {
				t.Fatalf("trades=%+v", result.Trades)
			}
			tr := result.Trades[0]
			if tr.EntryPrice != tc.entry || tr.ExitPrice != tc.exit || tr.Reason != tc.reason {
				t.Fatalf("trade=%+v", tr)
			}
		})
	}
}
