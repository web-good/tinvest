package backtest

import (
	"strings"
	"testing"
	"time"
)

func reportRows() []GapScreenRow {
	return []GapScreenRow{
		{Ticker: "GAZP", Name: "Газпром", MorningDays: 1, MorningTurnoverM: 109.5, TurnoverM: 4000, TickPct: 0.01, SignalsPerYear: 2, Pass: true},
		{Ticker: "TGKA", Name: "ТГК-1", MorningDays: 0.95, MorningTurnoverM: 0.4, TurnoverM: 30, TickPct: 0, SignalsPerYear: 5,
			Reasons: []string{"утренний оборот 0.4 млн ₽ < 5.0", "дневной оборот 30 млн ₽ < 50"}},
		{Ticker: "SBER", Name: "Сбербанк", MorningDays: 1, MorningTurnoverM: 127.7, TurnoverM: 9000, TickPct: 0.003, SignalsPerYear: 1, Pass: true},
		{Ticker: "AAAA", Name: "Ничья", MorningDays: 1, MorningTurnoverM: 109.5, TurnoverM: 100, TickPct: 0, SignalsPerYear: 0, Pass: true},
	}
}

func TestGapScreenPassedOrderAndLine(t *testing.T) {
	got := GapScreenPassed(reportRows())
	var names []string
	for _, r := range got {
		names = append(names, r.Ticker)
	}
	if strings.Join(names, ",") != "SBER,AAAA,GAZP" {
		t.Fatalf("order = %v, want SBER,AAAA,GAZP (turnover desc, ticker asc on ties)", names)
	}
	if line := GapScreenTickersLine(reportRows()); line != "GAP_FADE_TICKERS=SBER,AAAA,GAZP" {
		t.Fatalf("line = %q", line)
	}
	if line := GapScreenTickersLine(nil); line != "GAP_FADE_TICKERS=" {
		t.Fatalf("empty line = %q", line)
	}
}

func TestRenderGapScreenMarkdown(t *testing.T) {
	meta := GapScreenMeta{
		Generated: time.Date(2026, 10, 2, 15, 4, 0, 0, gapLoc), Months: 12, Scanned: 5,
		Opts: DefaultGapScreenOptions(), NoData: []string{"XXXX: нет свечей"},
	}
	md := RenderGapScreenMarkdown(reportRows(), meta)
	for _, want := range []string{
		"# gap_fade — скринер тикеров",
		"- Сформирован: 2026-10-02 15:04",
		"просмотрено тикеров: 5, прошли: 3, отсеяны: 1, нет данных: 1",
		"бар 07:00 ≥ 90% будних дней; утренний оборот ≥ 5.0 млн ₽; два шага ≤ 0.20%; дневной оборот ≥ 50 млн ₽",
		"GAP_FADE_TICKERS=SBER,AAAA,GAZP",
		"| SBER | Сбербанк | 100% | 127.7 | 9000 | 0.00% | 1.0 |",
		"| AAAA | Ничья | 100% | 109.5 | 100 | — | 0.0 |",
		"| TGKA | ТГК-1 | утренний оборот 0.4 млн ₽ < 5.0; дневной оборот 30 млн ₽ < 50 |",
		"- XXXX: нет свечей",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("report lacks %q\n---\n%s", want, md)
		}
	}
	if strings.Index(md, "| SBER |") > strings.Index(md, "| GAZP |") {
		t.Error("passed table not sorted by morning turnover")
	}
}

func TestRenderGapScreenMarkdownEmpty(t *testing.T) {
	opts := DefaultGapScreenOptions()
	opts.MaxTickPct = 0
	md := RenderGapScreenMarkdown(nil, GapScreenMeta{Generated: time.Now(), Months: 12, Opts: opts})
	for _, want := range []string{"GAP_FADE_TICKERS=\n", "прошли: 0", "два шага — выкл.", "Нет."} {
		if !strings.Contains(md, want) {
			t.Errorf("empty report lacks %q\n---\n%s", want, md)
		}
	}
}
