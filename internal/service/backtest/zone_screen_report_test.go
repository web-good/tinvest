package backtest

import (
	"strings"
	"testing"
	"time"
)

func zoneRow(ticker string, stress float64) ZoneRow {
	return ZoneRow{Ticker: ticker, Name: ticker + " name", TurnoverM: 100, DailyATRPct: 2, TickPct: 0.05,
		PFMed: stress + 0.5, PFMedStress: stress, TradesMed: 30, Best: ZoneGrid()[0]}
}

var testZoneGates = ZoneGates{MinTurnoverM: 50, MinATRPct: 1.5, MaxTickPct: 0.3}

func TestFilterAndRankZoneGates(t *testing.T) {
	lowTurnover := zoneRow("LOWT", 3)
	lowTurnover.TurnoverM = 10
	lowATR := zoneRow("LOWA", 3)
	lowATR.DailyATRPct = 1
	wideTick := zoneRow("KMAZ", 3)
	wideTick.TickPct = 0.4
	silent := zoneRow("SILN", 0)
	silent.NoSignals = true
	good := zoneRow("GOOD", 1.2)

	ranked, noSignals, tickRejected, rejected := FilterAndRankZone(
		[]ZoneRow{lowTurnover, lowATR, wideTick, silent, good}, testZoneGates)

	if len(ranked) != 1 || ranked[0].Ticker != "GOOD" {
		t.Fatalf("ranked = %+v, want only GOOD", ranked)
	}
	if len(noSignals) != 1 || noSignals[0].Ticker != "SILN" {
		t.Fatalf("noSignals = %+v, want only SILN", noSignals)
	}
	if len(tickRejected) != 1 || tickRejected[0].Ticker != "KMAZ" {
		t.Fatalf("tickRejected = %+v, want only KMAZ", tickRejected)
	}
	if len(rejected) != 2 {
		t.Fatalf("rejected = %d rows, want 2 (turnover, ATR)", len(rejected))
	}
}

func TestFilterAndRankZoneUnknownTickPasses(t *testing.T) {
	r := zoneRow("NOTK", 2)
	r.TickPct, r.TickUnknown = 0, true
	ranked, _, tickRejected, _ := FilterAndRankZone([]ZoneRow{r}, testZoneGates)
	if len(ranked) != 1 || len(tickRejected) != 0 {
		t.Fatalf("ranked/tickRejected = %d/%d, want 1/0 — an unmeasured tick is flagged, not rejected", len(ranked), len(tickRejected))
	}
}

func TestFilterAndRankZoneTickGateDisabledAtZero(t *testing.T) {
	r := zoneRow("KMAZ", 2)
	r.TickPct = 5
	g := testZoneGates
	g.MaxTickPct = 0
	ranked, _, tickRejected, _ := FilterAndRankZone([]ZoneRow{r}, g)
	if len(ranked) != 1 || len(tickRejected) != 0 {
		t.Fatalf("ranked/tickRejected = %d/%d, want 1/0 — -max-tick-pct 0 disables the gate", len(ranked), len(tickRejected))
	}
}

func TestFilterAndRankZoneSortsByStressedPFThenTicker(t *testing.T) {
	a, b, c := zoneRow("BBBB", 1.5), zoneRow("AAAA", 1.5), zoneRow("CCCC", 2.0)
	c.PFMed = 0.1 // raw PF must not drive the order
	ranked, _, _, _ := FilterAndRankZone([]ZoneRow{a, b, c}, testZoneGates)
	got := []string{ranked[0].Ticker, ranked[1].Ticker, ranked[2].Ticker}
	if strings.Join(got, ",") != "CCCC,AAAA,BBBB" {
		t.Fatalf("order = %v, want CCCC,AAAA,BBBB", got)
	}
}

func TestZoneDistributionUsesStressedPF(t *testing.T) {
	d := ZoneDistribution([]ZoneRow{zoneRow("A", 1), zoneRow("B", 2), zoneRow("C", 3)})
	if d.Median != 2 || d.N != 3 || d.Min != 1 || d.Max != 3 {
		t.Fatalf("distribution = %+v, want median 2 over stressed PF 1..3", d)
	}
}

func testZoneMeta() ZoneScreenMeta {
	return ZoneScreenMeta{Months: 36, HoldoutMonths: 6, TopN: 50, Split: time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC),
		Gates: testZoneGates, PFCap: 10, Commission: 0.0005, Cash: 100000, Scanned: 5}
}

func TestRenderZoneScreenMarkdownSections(t *testing.T) {
	r := zoneRow("GOOD", 1.8)
	r.HalvesPos, r.TopHalf, r.SLShare, r.HoldDays, r.AboveEMA50, r.AboveEMA200 = 0.75, 0.4, 0.2, 3.5, 0.6, 0.55
	r.BestHalves = []HalfResult{{Label: "2025H1", PnL: 3100}, {Label: "2025H2", PnL: -1200}}
	tick := zoneRow("KMAZ", 1)
	tick.TickPct = 0.4
	silent := zoneRow("SILN", 0)

	md := RenderZoneScreenMarkdown([]ZoneRow{r}, []ZoneRow{silent}, []ZoneRow{tick}, testZoneMeta())

	for _, want := range []string{
		"# RSI zone screener",
		"## Распределение PFmed×2",
		"## Рейтинг",
		"| # | Ticker | Name | Оборот, млн | ATR% дн | Tick% | Бары | TradesMed | PFmed | PFmed×2 |",
		"## Детали топа",
		"| Ticker | Halves+ | TopHalf | SL% | HoldD | >EMA50 | >EMA200 | Лучшая конфигурация | Полугодия лучшей конфигурации |",
		"| GOOD | 75% | 40% | 20% | 3.5 | 60% | 55% |",
		"RSI 4 15/65, EMA 50",
		"2025H1 +3.1% · 2025H2 -1.2%",
		"## Нет сигналов",
		"SILN",
		"## Отсеяно гейтом шага цены",
		"| KMAZ | 0.40 |",
		"## Как это читать",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("report missing %q:\n%s", want, md)
		}
	}
}

func TestRenderZoneDashesMissingTopHalf(t *testing.T) {
	r := zoneRow("NOPF", 0.5)
	r.TopHalf = 0
	md := RenderZoneScreenMarkdown([]ZoneRow{r}, nil, nil, testZoneMeta())
	if !strings.Contains(md, "| NOPF | 0% | — |") {
		t.Fatalf("TopHalf 0 must render as a dash:\n%s", md)
	}
}

func TestRenderZoneMarksUnknownTick(t *testing.T) {
	r := zoneRow("NOTK", 1.1)
	r.TickPct, r.TickUnknown = 0, true
	md := RenderZoneScreenMarkdown([]ZoneRow{r}, nil, nil, testZoneMeta())
	if !strings.Contains(md, "| 2.00 | ? |") {
		t.Fatalf("unknown tick must render as ?:\n%s", md)
	}
}

func TestRenderZoneRespectsTopNAndOmitsEmptySections(t *testing.T) {
	meta := testZoneMeta()
	meta.TopN = 1
	md := RenderZoneScreenMarkdown([]ZoneRow{zoneRow("AAAA", 2), zoneRow("BBBB", 1)}, nil, nil, meta)
	if strings.Contains(md, "BBBB") {
		t.Fatalf("TopN=1 rendered the second row:\n%s", md)
	}
	if strings.Contains(md, "## Нет сигналов") || strings.Contains(md, "## Отсеяно гейтом шага цены") {
		t.Fatalf("empty sections must be omitted:\n%s", md)
	}
}
