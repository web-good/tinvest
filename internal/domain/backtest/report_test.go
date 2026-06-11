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
	for _, want := range []string{"RUAL", "EMAPeriod", "Сводка метрик", "Журнал сделок", "Движение капитала", "TP", "Support"} {
		if !strings.Contains(out, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
}

func TestRenderTradesCSVHeaderAndRow(t *testing.T) {
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
		SupportLevel: 99, ResistanceLevel: 112, ATR: 1.25,
		ExitReason: "TP: high 110.0000 ≥ цель 110.0000 (2R)",
	}}
	out := RenderTradesCSV(trades)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("csv lines = %d, want 2 (header + 1 row)", len(lines))
	}
	if !strings.HasPrefix(lines[0], "idx,entry_time,entry_price") {
		t.Fatalf("header lost leading columns: %q", lines[0])
	}
	if !strings.Contains(lines[1], "TP") {
		t.Fatalf("row missing reason: %q", lines[1])
	}
	if !strings.HasSuffix(lines[0], "support_level,resistance_level,atr,entry_reason,exit_reason") {
		t.Fatalf("header missing new columns: %q", lines[0])
	}
	if !strings.Contains(lines[1], "цель") {
		t.Fatalf("row missing exit reason: %q", lines[1])
	}
	if !strings.Contains(lines[1], "99.000000,112.000000,1.250000") {
		t.Fatalf("row missing entry-context values: %q", lines[1])
	}
}

func TestRenderMarkdownHasExitReason(t *testing.T) {
	m := Metrics{TotalTrades: 1, Wins: 1, WinRate: 1.0, ProfitFactor: 2.0, NetPnL: 100}
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
		EntryReason: "Тренд↑ вход", ExitReason: "TP: high 110.0000 ≥ цель 110.0000 (2R)",
	}}
	out := RenderMarkdown(sampleMeta(), m, trades, eqCurve([]float64{100000, 100100}))
	if !strings.Contains(out, "Причина выхода") {
		t.Fatalf("markdown missing 'Причина выхода' header: %q", out)
	}
	if !strings.Contains(out, "TP: high 110.0000 ≥ цель 110.0000 (2R)") {
		t.Fatalf("markdown missing exit reason text: %q", out)
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
