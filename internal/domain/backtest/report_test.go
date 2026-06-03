package backtest

import (
	"strings"
	"testing"
	"time"
)

func sampleMeta() Meta {
	return Meta{
		Ticker:      "RUAL",
		Interval:    "Hour1",
		From:        time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		To:          time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		InitialCash: 100000,
		Fraction:    1.0,
		Commission:  0.0005,
		Params:      []ParamLine{{Name: "EMAPeriod", Value: "21"}, {Name: "ADXPeriod", Value: "14"}},
	}
}

func TestRenderMarkdownHasSections(t *testing.T) {
	m := Metrics{TotalTrades: 2, Wins: 1, Losses: 1, WinRate: 0.5, ProfitFactor: 1.5, NetPnL: 1000}
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
	}}
	eq := eqCurve([]float64{100000, 101000})
	out := RenderMarkdown(sampleMeta(), m, trades, eq)
	for _, want := range []string{"RUAL", "EMAPeriod", "Сводка метрик", "Журнал сделок", "Движение капитала", "TP"} {
		if !strings.Contains(out, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
}

func TestRenderTradesCSVHeaderAndRow(t *testing.T) {
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
	}}
	out := RenderTradesCSV(trades)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("csv lines = %d, want 2 (header + 1 row)", len(lines))
	}
	if !strings.HasPrefix(lines[0], "idx,entry_time,entry_price") {
		t.Fatalf("unexpected header: %q", lines[0])
	}
	if !strings.Contains(lines[1], "TP") {
		t.Fatalf("row missing reason: %q", lines[1])
	}
}

func TestRenderEquityCSV(t *testing.T) {
	out := RenderEquityCSV(eqCurve([]float64{100, 101, 102}))
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 { // header + 3 points
		t.Fatalf("equity csv lines = %d, want 4", len(lines))
	}
	if lines[0] != "time,equity" {
		t.Fatalf("header = %q, want time,equity", lines[0])
	}
}
