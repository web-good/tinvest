package backtest

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	pdlrsicore "tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"
)

func TestPDLRSIGridHour1(t *testing.T) {
	grid, err := PDLRSIGrid(enum.Hour1)
	if err != nil {
		t.Fatalf("PDLRSIGrid error: %v", err)
	}
	if len(grid) != 8 {
		t.Fatalf("len = %d, want 8", len(grid))
	}
	perMode := map[PDLRSIStochMode]int{}
	for _, p := range grid {
		perMode[PDLRSIModeOf(p)]++
		pinned := pdlrsicore.DefaultParams()
		pinned.HourRSIPeriod, pinned.StopATR = p.HourRSIPeriod, p.StopATR
		pinned.StochK = p.StochK
		if p != pinned {
			t.Fatalf("pinned fields drifted: %+v", p)
		}
	}
	for _, m := range pdlRSIGridModes {
		if perMode[m] != 4 {
			t.Fatalf("mode %s has %d configurations, want 4", m, perMode[m])
		}
	}
	if first := grid[0]; PDLRSIModeOf(first) != PDLRSIStochOff || first.HourRSIPeriod != 4 || first.StopATR != 0.5 {
		t.Fatalf("grid[0] = %+v, want no stochastic, RSI 4, stop 0.5", first)
	}
	if last := grid[7]; PDLRSIModeOf(last) != PDLRSIStochLevel || last.HourRSIPeriod != 6 || last.StopATR != 1.0 {
		t.Fatalf("grid[7] = %+v, want the level, RSI 6, stop 1.0", last)
	}
}

func TestPDLRSIGridRejectsOtherTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Minutes30, enum.Day1} {
		if _, err := PDLRSIGrid(iv); err == nil {
			t.Fatalf("PDLRSIGrid(%s) = nil error, want an error", iv)
		}
	}
}

func TestAggregatePDLRSIModesAndExits(t *testing.T) {
	split := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	at := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	trade := func(pnl float64, reason string) backtest.Trade {
		return backtest.Trade{EntryTime: at, ExitTime: at.Add(time.Hour), EntryPrice: 100, ExitPrice: 100 + pnl, Quantity: 1, PnL: pnl, Reason: reason}
	}
	grid, _ := PDLRSIGrid(enum.Hour1)
	results := make([]PDLRSIConfigResult, 0, len(grid))
	for _, p := range grid {
		var trades []backtest.Trade
		if PDLRSIModeOf(p) == PDLRSIStochLevel {
			trades = []backtest.Trade{trade(3, "TP"), trade(-1, "SL")}
		}
		results = append(results, PDLRSIConfigResult{Params: p, Trades: trades})
	}
	opts := DefaultScreenOpts()
	opts.Commission = 0
	row := AggregatePDLRSI("X", "x", results, split, 0, opts)
	if row.NoSignals {
		t.Fatal("NoSignals = true, want false")
	}
	if row.SilentCfg != 4 {
		t.Fatalf("SilentCfg = %d, want 4", row.SilentCfg)
	}
	if got := row.Modes[PDLRSIStochLevel]; got.PFMedStress != 3 || got.TradesMed != 2 {
		t.Fatalf("level mode = %+v, want PF 3 on 2 trades", got)
	}
	if got := row.Modes[PDLRSIStochOff]; got.TradesMed != 0 {
		t.Fatalf("off mode = %+v, want no trades", got)
	}
	if row.TPShare != 0.5 || row.SLShare != 0.5 || row.RSIShare != 0 {
		t.Fatalf("exit shares SL %.2f TP %.2f RSI %.2f, want 0.5/0.5/0", row.SLShare, row.TPShare, row.RSIShare)
	}
	if PDLRSIModeOf(row.Best) != PDLRSIStochLevel {
		t.Fatalf("Best mode = %s, want the only trading mode", PDLRSIModeOf(row.Best))
	}
}

func TestRenderPDLRSIScreenMarkdownShortlist(t *testing.T) {
	ranked := []PDLRSIRow{{Ticker: "AAA", PFMedStress: 1.5, TradesMed: 30}, {Ticker: "BBB", PFMedStress: 1.1, TradesMed: 40}}
	md := RenderPDLRSIScreenMarkdown(ranked, nil, nil, nil, PDLRSIScreenMeta{Interval: enum.Hour1, Shortlist: 1})
	if !strings.HasSuffix(md, PDLRSIShortlistKey+"=AAA\n") {
		t.Fatalf("report does not end with the shortlist line:\n%s", md[len(md)-200:])
	}
}
