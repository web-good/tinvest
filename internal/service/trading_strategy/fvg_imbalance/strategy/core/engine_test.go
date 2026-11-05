package core

import (
	"strings"
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
	bars = append(bars, bar{103, 104, 101, 101})
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
	// Signal close 101 implies a displayed target 102.5. Actual fill 104 and stop
	// 100 imply 104 + 1.5*(104-100) = 110; crossing 102.5 must not exit.
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
	if tr.Reason != "SL_EMERGENCY" || tr.EntryPrice != 99 || tr.ExitPrice != 99 || !tr.EntryTime.Equal(fillTime) || !tr.ExitTime.Equal(fillTime) {
		t.Fatalf("want same-fill-bar SL at open: %+v", tr)
	}
}
func TestEngineStopWinsOnFillBar(t *testing.T) {
	candles := engineCandles(bar{104, 120, 99, 110})
	result := runEngine(candles)
	if len(result.Trades) != 1 || result.Trades[0].Reason != "SL_EMERGENCY" || result.Trades[0].ExitPrice != 100 {
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
		{"gap through buffered stop", []bar{{98, 104, 97, 103}}, 98, 97, "SL_EMERGENCY"},
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

// Only H4 candles complete by the M30 decision close may influence the retest.
func TestFVGEngineHTFDecisionClose(t *testing.T) {
	p := testParams()
	p.HTFTrendEMA = 2
	candles := engineCandles(bar{104, 111, 101, 110})
	decisionStart := candles[len(candles)-2].Time
	cfg := bt.Config{InitialCash: 100000, Fraction: 1, Lot: 1, EntryAtNextOpen: true, HTFInterval: 4 * time.Hour}
	for _, tc := range []struct {
		name      string
		lastClose time.Duration
		wantTrade bool
	}{
		{"still forming at decision close", 31 * time.Minute, false},
		{"exactly at decision close", 30 * time.Minute, true},
		{"before decision close", 29 * time.Minute, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			htf := []bt.Candle{
				{Time: decisionStart.Add(-12 * time.Hour), Close: 110},
				{Time: decisionStart.Add(-8 * time.Hour), Close: 100},
				{Time: decisionStart.Add(-4*time.Hour + tc.lastClose), Close: 120},
			}
			result := bt.Run(NewWithParams("TEST", step, p), candles, nil, htf, cfg)
			if tc.wantTrade {
				if len(result.Trades) != 1 || result.Trades[0].Reason != "TP" {
					t.Fatalf("completed H4 should allow entry: %+v", result)
				}
			} else if len(result.Trades) != 0 || result.BarsInMarket != 0 {
				t.Fatalf("forming H4 allowed entry: %+v", result)
			}
		})
	}
}

// Primary is 99, formation ATR is 2, so the emergency stop is 97.
func TestEngineDualStopFills(t *testing.T) {
	for _, tc := range []struct {
		name    string
		exitBar bar
		reason  string
		exit    float64
	}{
		{"emergency at level", bar{104, 105, 97, 102}, "SL_EMERGENCY", 97},
		{"emergency gap", bar{96, 105, 95, 102}, "SL_EMERGENCY", 96},
		{"primary close at level", bar{104, 105, 98, 99}, "SL_CLOSE", 99},
		{"primary gap recovered above emergency", bar{98, 105, 97.5, 98.5}, "SL_CLOSE", 98.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.StopBufferATR = .5
			candles := engineCandles(bar{104, 105, 102, 104}, tc.exitBar)
			result := bt.Run(NewWithParams("TEST", step, p), candles, nil, nil, bt.Config{InitialCash: 100000, Fraction: 1, Lot: 1, EntryAtNextOpen: true})
			if len(result.Trades) != 1 {
				t.Fatalf("trades=%+v", result.Trades)
			}
			tr := result.Trades[0]
			if tr.EntryPrice != 104 || tr.ExitPrice != tc.exit || tr.Reason != tc.reason || !tr.ExitTime.Equal(candles[len(candles)-1].Time) {
				t.Fatalf("trade=%+v", tr)
			}
		})
	}
}

// H4 rejection must leave a depth-confirmed zone eligible. Only a confirmation
// that also passed H4 can spend its exact next-open execution opportunity.
func TestEngineStochasticWaitsForH4(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		gap, weekend, h4PassesFirst bool
		wantTrade                   bool
	}{
		{"H4 rejects then passes", false, false, false, true},
		{"missing slot before H4 passes", true, false, false, true},
		{"missing slot after H4 passes", true, false, true, false},
		{"weekend before H4 passes", true, true, false, true},
		{"weekend after H4 passes", true, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.UseStochFilter, p.HTFTrendEMA = 1, 2
			bars := make([]bar, 12)
			for i := range bars {
				bars[i] = bar{110, 120, 108, 110}
			}
			bars = append(bars, stochasticFormation()...)
			first := len(bars)
			bars = append(bars, bar{103, 104, 100.5, 101}, bar{104, 105, 103, 103}, bar{104, 105, 103, 104}, bar{104, 111, 103, 110})
			md := snapshot(bars...)
			if tc.weekend {
				friday := time.Date(2026, 11, 6, 23, 30, 0, 0, time.UTC)
				for i := range md.Times {
					md.Times[i] = friday.Add(time.Duration(i-first) * step)
				}
			}
			if tc.gap {
				next := md.Times[first].Add(2 * step)
				if tc.weekend {
					next = time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
				}
				for i := first + 1; i < len(bars); i++ {
					md.Times[i] = next.Add(time.Duration(i-first-1) * step)
				}
			}
			candles := make([]bt.Candle, len(bars))
			for i, b := range bars {
				candles[i] = bt.Candle{Time: md.Times[i], Open: b.o, High: b.h, Low: b.l, Close: b.c}
			}
			passesAt := md.Times[first+1].Add(step)
			if tc.h4PassesFirst {
				passesAt = md.Times[first].Add(step)
			}
			htf := []bt.Candle{
				{Time: md.Times[first].Add(-12 * time.Hour), Close: 110},
				{Time: md.Times[first].Add(-8 * time.Hour), Close: 100},
				{Time: passesAt.Add(-4 * time.Hour), Close: 120},
			}
			cfg := bt.Config{InitialCash: 100000, Fraction: 1, Lot: 1, EntryAtNextOpen: true, HTFInterval: 4 * time.Hour}
			s := NewWithParams("TEST", step, p)
			result := bt.Run(s, candles, nil, htf, cfg)
			if tc.wantTrade {
				if len(result.Trades) != 1 || !result.Trades[0].EntryTime.Equal(md.Times[first+2]) || result.Trades[0].Reason != "TP" {
					t.Errorf("zone must wait for H4 then fill next open: %+v", result.Trades)
				}
			} else if len(result.Trades) != 0 || result.BarsInMarket != 0 {
				t.Errorf("rejected execution slot carried forward: %+v", result.Trades)
			}
			trace := bt.Trace(s, candles, nil, htf, cfg, md.Times[first+2])
			if got := strings.Contains(trace, "В ПОЗИЦИИ"); got != tc.wantTrade {
				t.Errorf("Trace position=%v, want %v: %s", got, tc.wantTrade, trace)
			}
		})
	}
}
