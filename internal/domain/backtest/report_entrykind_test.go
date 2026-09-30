package backtest

import (
	"strings"
	"testing"
	"time"
)

func kindTrade(kind, reason string, pnl float64) Trade {
	return Trade{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 100 + pnl, Quantity: 1, Reason: reason, PnL: pnl, BarsHeld: 1, EntryKind: kind,
	}
}

func TestKindBreakdownOrdersAndSkipsEmpty(t *testing.T) {
	trades := []Trade{
		kindTrade("zone", "SL", -50),
		kindTrade("", "TP", 10),
		kindTrade("pullback", "TP", 100),
		kindTrade("alpha", "TP", 5),
		kindTrade("zone", "TP", 150),
		kindTrade("pullback", "SL", -40),
	}
	got := KindBreakdown(trades)
	var kinds []string
	for _, k := range got {
		kinds = append(kinds, k.Kind)
	}
	if strings.Join(kinds, ",") != "pullback,zone,alpha" {
		t.Fatalf("kinds = %v, want pullback,zone,alpha", kinds)
	}
	zone := got[1]
	if zone.Metrics.TotalTrades != 2 || zone.Metrics.NetPnL != 100 {
		t.Fatalf("zone: сделок %d, Net PnL %v, want 2 и 100 (−50+150)", zone.Metrics.TotalTrades, zone.Metrics.NetPnL)
	}
	if got[0].Metrics.NetPnL != 60 || got[2].Metrics.NetPnL != 5 {
		t.Fatalf("Net PnL pullback/alpha = %v/%v, want 60/5", got[0].Metrics.NetPnL, got[2].Metrics.NetPnL)
	}
	var tbl strings.Builder
	RenderKindTable(&tbl, got)
	if !strings.Contains(tbl.String(), "| zone | 2 |") || !strings.Contains(tbl.String(), "| 100.00 |") {
		t.Fatalf("строка zone не показывает Net PnL 100.00: %q", tbl.String())
	}
	if zone.SLExits != 1 || got[0].SLExits != 1 || got[2].SLExits != 0 {
		t.Fatalf("SLExits = %d/%d/%d, want 1/1/0", got[0].SLExits, zone.SLExits, got[2].SLExits)
	}
}

func TestKindBreakdownNilWithoutKinds(t *testing.T) {
	if got := KindBreakdown([]Trade{kindTrade("", "TP", 10)}); got != nil {
		t.Fatalf("KindBreakdown = %v, want nil", got)
	}
}

func TestRenderMarkdownEntryKindColumnAndSection(t *testing.T) {
	trades := []Trade{kindTrade("pullback", "TP", 100), kindTrade("zone", "SL", -50), kindTrade("", "TP", 10)}
	out := RenderMarkdown(sampleMeta(), Metrics{}, trades, nil)
	const header = "| № | Вход | Цена входа | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода | Тип входа |"
	if !strings.Contains(out, header) {
		t.Fatalf("заголовок журнала: старые 14 колонок на местах, «Тип входа» последней; got %q", out)
	}
	rows := map[string]string{"| 1 |": "| pullback |", "| 2 |": "| zone |", "| 3 |": "| — |"}
	for _, line := range strings.Split(out, "\n") {
		for prefix, suffix := range rows {
			if strings.HasPrefix(line, prefix) && !strings.HasSuffix(line, suffix) {
				t.Fatalf("строка %q не оканчивается типом входа %q", line, suffix)
			}
		}
	}
	sec := strings.Index(out, "## По типу входа")
	journal := strings.Index(out, "## Журнал сделок")
	if sec < 0 || sec > journal {
		t.Fatalf("раздел «По типу входа» отсутствует или стоит после журнала")
	}
	if !strings.Contains(out, "| pullback | 1 |") || !strings.Contains(out, "| zone | 1 |") {
		t.Fatalf("строки раздела по типам не найдены: %q", out[sec:journal])
	}
}

func TestRenderMarkdownNoKindSectionWithoutKinds(t *testing.T) {
	out := RenderMarkdown(sampleMeta(), Metrics{}, []Trade{kindTrade("", "TP", 10)}, nil)
	if strings.Contains(out, "## По типу входа") {
		t.Fatal("раздел по типам выведен для стратегии без типа входа")
	}
	if !strings.Contains(out, "| — |\n") {
		t.Fatalf("пустой тип должен печататься «—» последней колонкой: %q", out)
	}
}

func TestRenderTradesCSVEntryKindIsLastColumn(t *testing.T) {
	out := RenderTradesCSV([]Trade{kindTrade("zone", "SL", -50)})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if lines[0] != "idx,entry_time,entry_price,exit_time,exit_price,qty,reason,pnl,pnl_pct,bars_held,support_level,resistance_level,atr,entry_reason,exit_reason,entry_kind" {
		t.Fatalf("header = %q: старые колонки обязаны остаться на местах, entry_kind — последней", lines[0])
	}
	if !strings.HasSuffix(lines[1], ",zone") {
		t.Fatalf("row = %q, want suffix ,zone", lines[1])
	}
}
