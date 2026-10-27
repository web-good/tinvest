package backtest

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/enum"
)

func dropRow(ticker string, pfStress, trades, turnover, tick float64) DropRow {
	return DropRow{Ticker: ticker, PFMedStress: pfStress, PFMed: pfStress + 0.2, TradesMed: trades, TurnoverM: turnover, TickPct: tick}
}

func tickers(rows []DropRow) string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Ticker)
	}
	return strings.Join(out, ",")
}

func TestFilterAndRankDrop(t *testing.T) {
	g := DropGates{MinTurnoverM: 50, MaxTickPct: 0.3, MinTrades: 20}
	unknownTick := dropRow("G", 1.1, 30, 100, 0)
	unknownTick.TickUnknown = true
	silent := dropRow("H", 0, 0, 100, 0.1)
	silent.NoSignals = true
	rows := []DropRow{
		dropRow("A", 1.5, 30, 100, 0.1),
		dropRow("B", 1.5, 40, 100, 0.1), // same key as A, more trades: ranks above A
		dropRow("C", 2.0, 25, 100, 0.1),
		dropRow("D", 3.0, 30, 49.9, 0.1), // turnover below the gate
		dropRow("E", 3.0, 19, 100, 0.1),  // too few trades
		dropRow("F", 3.0, 30, 100, 0.31), // ticks too expensive
		unknownTick,                      // unknown tick passes
		silent,                           // no signals: its own bucket, not "too few trades"
		dropRow("I", 1.2, 20, 50, 0.3),   // every gate exactly at its threshold passes
	}
	ranked, noSignals, tickRejected, rejected := FilterAndRankDrop(rows, g)
	if got := tickers(ranked); got != "C,B,A,I,G" {
		t.Fatalf("ranked = %s, want C,B,A,I,G", got)
	}
	if got := tickers(noSignals); got != "H" {
		t.Fatalf("noSignals = %s, want H", got)
	}
	if got := tickers(tickRejected); got != "F" {
		t.Fatalf("tickRejected = %s, want F", got)
	}
	if got := tickers(rejected); got != "D,E" {
		t.Fatalf("rejected = %s, want D,E", got)
	}
}

func TestFilterAndRankDropTickGateOff(t *testing.T) {
	ranked, _, tickRejected, _ := FilterAndRankDrop([]DropRow{dropRow("F", 3.0, 30, 100, 5)}, DropGates{MinTurnoverM: 50, MinTrades: 20})
	if len(tickRejected) != 0 || len(ranked) != 1 {
		t.Fatalf("MaxTickPct 0 must disable the tick gate: ranked=%d tickRejected=%d", len(ranked), len(tickRejected))
	}
}

func TestDropRejectReason(t *testing.T) {
	g := DropGates{MinTurnoverM: 50, MinTrades: 20}
	if got := DropRejectReason(dropRow("A", 1, 30, 100, 0), g); got != "" {
		t.Fatalf("passing row reason = %q, want empty", got)
	}
	got := DropRejectReason(dropRow("D", 1, 19, 49.9, 0), g)
	if !strings.Contains(got, "оборот 49.9 < 50") || !strings.Contains(got, "сделок 19 < 20") {
		t.Fatalf("reason = %q, want both the turnover and the trades failure", got)
	}
}

func TestDropShortlist(t *testing.T) {
	ranked := []DropRow{{Ticker: "A"}, {Ticker: "B"}, {Ticker: "C"}}
	if got := strings.Join(DropShortlist(ranked, 2), ","); got != "A,B" {
		t.Fatalf("n=2: %s, want A,B", got)
	}
	if got := strings.Join(DropShortlist(ranked, 10), ","); got != "A,B,C" {
		t.Fatalf("n beyond the ranking: %s, want A,B,C", got)
	}
	if got := strings.Join(DropShortlist(ranked, 0), ","); got != "A,B,C" {
		t.Fatalf("n=0 means the whole ranking: %s, want A,B,C", got)
	}
	if got := DropShortlist(nil, 10); len(got) != 0 {
		t.Fatalf("empty ranking: %v, want none", got)
	}
}

func dropMeta() DropScreenMeta {
	return DropScreenMeta{
		Interval: enum.Minutes15, Months: 36, HoldoutMonths: 6, TopN: 50, Shortlist: 2,
		Split: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC), Gates: DropGates{MinTurnoverM: 50, MaxTickPct: 0.3, MinTrades: 20},
		PFCap: 10, Commission: 0.0005, Cash: 100000, Scanned: 5, Skipped: 1,
	}
}

func TestRenderDropScreenMarkdown(t *testing.T) {
	ranked := []DropRow{dropRow("C", 2.0, 25, 100, 0.1), dropRow("B", 1.5, 40, 100, 0.1), dropRow("A", 1.5, 30, 100, 0.1)}
	ranked[0].BestHalves = []HalfResult{{Label: "2024H1", PnL: 1500, Trades: 3}}
	ranked[0].RSIShare = 0.4
	md := RenderDropScreenMarkdown(ranked, []DropRow{{Ticker: "H", NoSignals: true}},
		[]DropRow{dropRow("F", 3, 30, 100, 0.31)}, []DropRow{dropRow("D", 3, 30, 49.9, 0.1)}, dropMeta())
	for _, want := range []string{
		"# drop_bounce screener",
		"Minutes15",
		"RSIPeriod {4, 6} × RSIExitPeriod {8, 12} × StopDailyATR {0.5, 1.0}",
		"## Рейтинг",
		"## Детали топа",
		"2024H1 +1.5%",
		"| Ticker | SL% | RSI% | HoldD |",
		"`SL%`/`RSI%`",
		" 40% |",
		"## Нет сигналов",
		"## Отсеяно гейтом шага цены",
		"## Отсеяно гейтами",
		"оборот 49.9 < 50",
		"DROP_BOUNCE_CANDIDATES=C,B\n",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("report lacks %q:\n%s", want, md)
		}
	}
}

func TestRenderDropScreenMarkdownEmptyRanking(t *testing.T) {
	md := RenderDropScreenMarkdown(nil, nil, nil, nil, dropMeta())
	if !strings.Contains(md, "DROP_BOUNCE_CANDIDATES=\n") {
		t.Fatalf("empty ranking must still print the key with nothing after it:\n%s", md)
	}
}

func pregatedRow() DropRow {
	return DropRow{Ticker: "P", Pregated: true, TurnoverM: 3, TickPct: 5}
}

func TestFilterAndRankDropPregated(t *testing.T) {
	g := DropGates{MinTurnoverM: 50, MaxTickPct: 0.3, MinTrades: 20}
	p := pregatedRow()
	p.NoSignals = true // zero trades and an expensive tick: still neither bucket
	ranked, noSignals, tickRejected, rejected := FilterAndRankDrop([]DropRow{p}, g)
	if len(ranked) != 0 || len(noSignals) != 0 || len(tickRejected) != 0 || tickers(rejected) != "P" {
		t.Fatalf("ranked=%v noSignals=%v tickRejected=%v rejected=%v", ranked, noSignals, tickRejected, rejected)
	}
}

func TestDropRejectReasonPregated(t *testing.T) {
	got := DropRejectReason(pregatedRow(), DropGates{MinTurnoverM: 50, MinTrades: 20})
	if !strings.Contains(got, "по дневным") || strings.Contains(got, "сделок") || !strings.Contains(got, "оборот 3.0 < 50") {
		t.Fatalf("reason = %q", got)
	}
}

func TestRenderDropScreenMarkdownPregated(t *testing.T) {
	md := RenderDropScreenMarkdown(nil, nil, nil, []DropRow{pregatedRow()}, dropMeta())
	if !strings.Contains(md, "| P | оборот 3.0 < 50 млн ₽ (по дневным свечам, сетка не гонялась) | — |") {
		t.Fatalf("pregated row must show a dash in PFmed×2:\n%s", md)
	}
	if !strings.Contains(md, "pregated=1") {
		t.Fatalf("header lacks pregated=1:\n%s", md)
	}
}
