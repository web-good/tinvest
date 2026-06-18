package backtest

import (
	"strings"
	"testing"

	domainbt "tinvest/internal/domain/backtest"
)

func TestVolMetrics_BasicShape(t *testing.T) {
	// 6 bars, period 3 → series valid from index 3; ATR% positive.
	candles := []domainbt.Candle{
		{High: 10, Low: 9, Close: 9.5, Volume: 100},
		{High: 11, Low: 9.5, Close: 10.5, Volume: 100},
		{High: 12, Low: 10, Close: 11.5, Volume: 100},
		{High: 13, Low: 11, Close: 12.5, Volume: 100},
		{High: 14, Low: 12, Close: 13.5, Volume: 100},
		{High: 15, Low: 13, Close: 14.5, Volume: 100},
	}
	mean, last, turn, bars := VolMetrics(candles, 1, 3)

	if bars != 6 {
		t.Fatalf("bars = %d, want 6", bars)
	}
	if mean <= 0 || last <= 0 {
		t.Errorf("mean=%v last=%v, want both > 0", mean, last)
	}
	// turnover = mean(volume*lot*close)/1e6 ; closes avg ~12 → ~100*12/1e6
	if turn <= 0 {
		t.Errorf("turnover = %v, want > 0", turn)
	}
}

func TestVolMetrics_InsufficientHistory(t *testing.T) {
	candles := []domainbt.Candle{
		{High: 10, Low: 9, Close: 9.5, Volume: 100},
		{High: 11, Low: 9.5, Close: 10.5, Volume: 100},
	}
	mean, last, _, bars := VolMetrics(candles, 1, 14)
	if bars != 2 {
		t.Fatalf("bars = %d, want 2", bars)
	}
	if mean != 0 || last != 0 {
		t.Errorf("mean=%v last=%v, want 0 (no valid ATR)", mean, last)
	}
}

func TestRenderVolatilityMarkdown_SortsDescAndTrend(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", MeanATRpct: 1.0, LastATRpct: 1.5, TurnoverM: 200, Bars: 120}, // trend up
		{Ticker: "BBB", MeanATRpct: 3.0, LastATRpct: 2.0, TurnoverM: 50, Bars: 120},  // trend down
	}
	meta := VolMeta{Months: 6, ATRPeriod: 14, MinTurnover: 50, Scanned: 100, Passed: 2}

	out := RenderVolatilityMarkdown(rows, meta, 0)

	bbb := strings.Index(out, "BBB")
	aaa := strings.Index(out, "AAA")
	if bbb == -1 || aaa == -1 {
		t.Fatalf("both tickers must appear; out=%q", out)
	}
	if bbb > aaa {
		t.Errorf("BBB (mean 3.0) must rank before AAA (mean 1.0)")
	}
	if !strings.Contains(out, "↑") || !strings.Contains(out, "↓") {
		t.Errorf("expected both trend arrows in output")
	}
}

func TestRenderVolatilityMarkdown_TopN(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", MeanATRpct: 1.0, Bars: 120},
		{Ticker: "BBB", MeanATRpct: 3.0, Bars: 120},
		{Ticker: "CCC", MeanATRpct: 2.0, Bars: 120},
	}
	out := RenderVolatilityMarkdown(rows, VolMeta{}, 2)
	if strings.Contains(out, "AAA") {
		t.Errorf("topN=2 must drop AAA (lowest mean); out=%q", out)
	}
	if !strings.Contains(out, "BBB") || !strings.Contains(out, "CCC") {
		t.Errorf("topN=2 must keep BBB and CCC")
	}
}
