package backtest

import (
	"fmt"
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

func TestFilterAndRankBreaksTiesByTicker(t *testing.T) {
	// The worker pool in cmd/pullscreen fills rows in completion order, which varies
	// run to run. Rows tied on PFMed (common once the zero-trade-config PF=0 rows pile
	// up — see Aggregate's SilentCfg) must not depend on that arrival order: two runs
	// against the same cache must render the same report.
	rows := []PullbackRow{
		{Ticker: "ZZZZ", TurnoverM: 100, DailyATRPct: 2.0, PFMed: 0},
		{Ticker: "AAAA", TurnoverM: 100, DailyATRPct: 2.0, PFMed: 0},
		{Ticker: "MMMM", TurnoverM: 100, DailyATRPct: 2.0, PFMed: 1.5},
		{Ticker: "BBBB", TurnoverM: 100, DailyATRPct: 2.0, PFMed: 1.5},
	}
	ranked, _, _ := FilterAndRank(rows, 50, 1.5)
	got := make([]string, len(ranked))
	for i, r := range ranked {
		got[i] = r.Ticker
	}
	want := []string{"BBBB", "MMMM", "AAAA", "ZZZZ"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ranked order = %v, want %v (PFMed desc, Ticker asc within a tie)", got, want)
		}
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
	// Odd-sized sample: N=5
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
	// Q1 and Q3 must be computed; for N=5 sorted [0.5, 1.0, 1.5, 2.0, 3.0]:
	// Q1 = median of [0.5, 1.0] = 0.75, Q3 = median of [2.0, 3.0] = 2.5
	if math.Abs(d.Q1-0.75) > 1e-9 {
		t.Fatalf("Q1 = %v, want 0.75", d.Q1)
	}
	if math.Abs(d.Q3-2.5) > 1e-9 {
		t.Fatalf("Q3 = %v, want 2.5", d.Q3)
	}
}

func TestDistributionEvenSized(t *testing.T) {
	// Even-sized sample: N=4
	rows := []PullbackRow{
		{PFMed: 1.0}, {PFMed: 2.0}, {PFMed: 3.0}, {PFMed: 4.0},
	}
	d := Distribution(rows)
	if d.N != 4 {
		t.Fatalf("N = %d, want 4", d.N)
	}
	if math.Abs(d.Median-2.5) > 1e-9 {
		t.Fatalf("median = %v, want 2.5 (midpoint of 2.0 and 3.0)", d.Median)
	}
	// Q1 = median of [1.0, 2.0] = 1.5, Q3 = median of [3.0, 4.0] = 3.5
	if math.Abs(d.Q1-1.5) > 1e-9 {
		t.Fatalf("Q1 = %v, want 1.5", d.Q1)
	}
	if math.Abs(d.Q3-3.5) > 1e-9 {
		t.Fatalf("Q3 = %v, want 3.5", d.Q3)
	}
}

func TestDistributionSingleValue(t *testing.T) {
	// Single-row universe (N=1): Q1 and Q3 must equal the median to avoid
	// misleading 0.00 values in the report
	rows := []PullbackRow{{PFMed: 2.5}}
	d := Distribution(rows)
	if d.N != 1 {
		t.Fatalf("N = %d, want 1", d.N)
	}
	if math.Abs(d.Min-2.5) > 1e-9 || math.Abs(d.Max-2.5) > 1e-9 {
		t.Fatalf("min/max = %v/%v, want both 2.5", d.Min, d.Max)
	}
	if math.Abs(d.Median-2.5) > 1e-9 {
		t.Fatalf("median = %v, want 2.5", d.Median)
	}
	if math.Abs(d.Q1-2.5) > 1e-9 {
		t.Fatalf("Q1 = %v, want 2.5 (must equal median for N=1)", d.Q1)
	}
	if math.Abs(d.Q3-2.5) > 1e-9 {
		t.Fatalf("Q3 = %v, want 2.5 (must equal median for N=1)", d.Q3)
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

func TestRenderPullbackScreenMarkdownShowsCappedFraction(t *testing.T) {
	// Spec §4.1: the report must surface HOW MANY of the 24 grid configurations hit the
	// PF cap, not just the cap value in the header — a row whose "best" config is itself
	// capped is a thin/no-loss sample dressed up as a strong PFmed, and the reader needs
	// that visible next to the row, not just knowable from the raw PF header note.
	meta := ScreenMeta{Months: 36, HoldoutMonths: 6, TopN: 50, Split: time.Now(), PFCap: 10}
	ranked := []PullbackRow{
		{Ticker: "AFKS", PFMed: 2.19, Capped: 2},
		{Ticker: "CALM", PFMed: 1.1, Capped: 0},
	}
	md := RenderPullbackScreenMarkdown(ranked, nil, meta)

	denom := fmt.Sprintf("/%d", len(PullbackGrid()))
	if !strings.Contains(md, "2"+denom) {
		t.Fatalf("report is missing the capped fraction \"2%s\" for AFKS\n---\n%s", denom, md)
	}
	if !strings.Contains(md, "0"+denom) {
		t.Fatalf("report is missing the capped fraction \"0%s\" for CALM\n---\n%s", denom, md)
	}
	if !strings.Contains(md, "Capped") {
		t.Fatal("report is missing a Capped column header")
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

func TestRenderPullbackScreenMarkdownRendersAllWhenTopNIsZero(t *testing.T) {
	meta := ScreenMeta{Months: 36, HoldoutMonths: 6, TopN: 0, Split: time.Now(), PFCap: 10} // TopN=0 means all
	ranked := []PullbackRow{{Ticker: "FIRST", PFMed: 2}, {Ticker: "SECOND", PFMed: 1}}
	md := RenderPullbackScreenMarkdown(ranked, nil, meta)
	if !strings.Contains(md, "FIRST") || !strings.Contains(md, "SECOND") {
		t.Fatal("TopN=0 must render all ranking rows")
	}
}
