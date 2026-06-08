package backtest

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMarkdownShowsEntryReason(t *testing.T) {
	trades := []Trade{{
		EntryTime:   time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
		EntryPrice:  36.2,
		ExitTime:    time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC),
		ExitPrice:   38.0,
		Reason:      "TP",
		EntryReason: "Тренд↑; MACD кросс; объём 1.8×",
	}}
	out := RenderMarkdown(Meta{Ticker: "RUAL", Interval: "Hour1"}, Metrics{}, trades, nil)
	if !strings.Contains(out, "Причина входа") {
		t.Fatal("markdown header missing 'Причина входа' column")
	}
	if !strings.Contains(out, "MACD кросс") {
		t.Fatal("markdown missing the trade EntryReason text")
	}
}

func TestRenderTradesCSVShowsEntryReason(t *testing.T) {
	trades := []Trade{{Reason: "SL", EntryReason: `пробой " key" уровня; объём 2×`}}
	out := RenderTradesCSV(trades)
	if !strings.Contains(out, "entry_reason") {
		t.Fatal("csv header missing entry_reason")
	}
	// A value with an embedded double-quote must be RFC 4180 quoted: wrapped in
	// double-quotes with inner quotes DOUBLED (not backslash-escaped).
	if !strings.Contains(out, `"пробой "" key"" уровня; объём 2×"`) {
		t.Fatalf("csv entry_reason not RFC4180-quoted; got:\n%s", out)
	}
}
