package backtest

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestFilterAndRankGates(t *testing.T) {
	rows := []PullbackRow{
		{Ticker: "GOOD", TurnoverM: 100, DailyATRPct: 2.0, PFMed: 1.2},
		{Ticker: "THIN", TurnoverM: 10, DailyATRPct: 2.0, PFMed: 3.0},  // liquidity gate
		{Ticker: "CALM", TurnoverM: 100, DailyATRPct: 0.8, PFMed: 3.0}, // ATR% gate
		{Ticker: "QUIET", TurnoverM: 100, DailyATRPct: 2.0, NoSignals: true},
		{Ticker: "BEST", TurnoverM: 100, DailyATRPct: 2.0, PFMed: 2.5},
	}
	ranked, noSignals, rejected := FilterAndRank(rows, 50, 1.5)

	if len(ranked) != 2 || ranked[0].Ticker != "BEST" || ranked[1].Ticker != "GOOD" {
		t.Fatalf("ranked = %+v, want BEST then GOOD sorted by PFMed desc", ranked)
	}
	if len(noSignals) != 1 || noSignals[0].Ticker != "QUIET" {
		t.Fatalf("noSignals = %+v, want QUIET", noSignals)
	}
	if len(rejected) != 2 {
		t.Fatalf("rejected = %+v, want THIN and CALM", rejected)
	}
}

func TestFilterAndRankNoSignalsGatedToo(t *testing.T) {
	// A no-signal ticker that also fails liquidity is a rejection, not a "no signals"
	// row: the report's no-signal section is about tickers that passed the gates.
	rows := []PullbackRow{{Ticker: "THIN", TurnoverM: 1, DailyATRPct: 2.0, NoSignals: true}}
	ranked, noSignals, rejected := FilterAndRank(rows, 50, 1.5)
	if len(ranked) != 0 || len(noSignals) != 0 || len(rejected) != 1 {
		t.Fatalf("ranked/noSignals/rejected = %d/%d/%d, want 0/0/1", len(ranked), len(noSignals), len(rejected))
	}
}

func TestDistribution(t *testing.T) {
	rows := []PullbackRow{
		{PFMed: 0.5}, {PFMed: 1.0}, {PFMed: 1.5}, {PFMed: 2.0}, {PFMed: 3.0},
	}
	d := Distribution(rows)
	if d.N != 5 {
		t.Fatalf("N = %d, want 5", d.N)
	}
	if math.Abs(d.Min-0.5) > 1e-9 || math.Abs(d.Max-3.0) > 1e-9 {
		t.Fatalf("min/max = %v/%v, want 0.5/3.0", d.Min, d.Max)
	}
	if math.Abs(d.Median-1.5) > 1e-9 {
		t.Fatalf("median = %v, want 1.5", d.Median)
	}
	if math.Abs(d.ShareAbove15-0.6) > 1e-9 {
		t.Fatalf("ShareAbove15 = %v, want 0.6 (three of five at or above 1.5)", d.ShareAbove15)
	}
}

func TestDistributionEmpty(t *testing.T) {
	d := Distribution(nil)
	if d.N != 0 || d.Median != 0 || d.ShareAbove15 != 0 {
		t.Fatalf("Distribution(nil) = %+v, want the zero value", d)
	}
}

func TestRenderPullbackScreenMarkdown(t *testing.T) {
	meta := ScreenMeta{
		Months: 36, HoldoutMonths: 6, TopN: 50,
		Split:        time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC),
		MinTurnoverM: 50, MinATRPct: 1.5, PFCap: 10,
		Scanned: 271, Passed: 2, Skipped: 3,
	}
	ranked := []PullbackRow{
		{Ticker: "BEST", Name: "Best Co", TurnoverM: 120.5, DailyATRPct: 2.4, Bars: 20000,
			TradesMed: 18, PFMed: 2.5, Plateau: 0.75, PFMedHO: 1.8, TradesMedHO: 4,
			Best: PullbackGrid()[0], BestPF: 4.1, Capped: 2},
		{Ticker: "GOOD", Name: "Good Co", TurnoverM: 80, DailyATRPct: 1.9, Bars: 19000,
			TradesMed: 12, PFMed: 1.2, Plateau: 0.2, PFMedHO: 0.4, TradesMedHO: 2,
			Best: PullbackGrid()[1], BestPF: 2.0},
	}
	noSignals := []PullbackRow{{Ticker: "QUIET", Name: "Quiet Co"}}

	md := RenderPullbackScreenMarkdown(ranked, noSignals, meta)

	for _, want := range []string{
		"# RSI pullback screener",
		"2026-02-03",   // the train/holdout split date
		"scanned=271",  // universe accounting
		"BEST", "GOOD", // ranking rows
		"QUIET",               // no-signal section
		"Распределение PFmed", // the universe backdrop
		"кандидаты на калибровку", // the disclaimer must be in the report itself
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("report is missing %q\n---\n%s", want, md)
		}
	}
}

func TestRenderPullbackScreenMarkdownOmitsEmptyNoSignalSection(t *testing.T) {
	meta := ScreenMeta{Months: 36, HoldoutMonths: 6, TopN: 50, Split: time.Now(), PFCap: 10}
	md := RenderPullbackScreenMarkdown([]PullbackRow{{Ticker: "ONLY", PFMed: 1.1}}, nil, meta)
	if strings.Contains(md, "Нет сигналов") {
		t.Fatal("empty no-signal section must be omitted entirely")
	}
}

func TestRenderPullbackScreenMarkdownRespectsTopN(t *testing.T) {
	meta := ScreenMeta{Months: 36, HoldoutMonths: 6, TopN: 1, Split: time.Now(), PFCap: 10}
	ranked := []PullbackRow{{Ticker: "FIRST", PFMed: 2}, {Ticker: "SECOND", PFMed: 1}}
	md := RenderPullbackScreenMarkdown(ranked, nil, meta)
	if !strings.Contains(md, "FIRST") || strings.Contains(md, "SECOND") {
		t.Fatal("TopN=1 must render exactly one ranking row")
	}
}
