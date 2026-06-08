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
	trades := []Trade{{Reason: "SL", EntryReason: "пробой; объём 2×"}}
	out := RenderTradesCSV(trades)
	if !strings.Contains(out, "entry_reason") {
		t.Fatal("csv header missing entry_reason")
	}
	if !strings.Contains(out, "пробой; объём 2×") {
		t.Fatal("csv missing EntryReason value")
	}
}
