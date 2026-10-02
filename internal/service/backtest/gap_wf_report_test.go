package backtest

import (
	"strings"
	"testing"
)

func TestRenderGapWFMarkdown(t *testing.T) {
	runs := []GapComboRun{{Params: "combo-zero", Trades: []GapTrade{gt("A", "2024-08-05 07:00", 0.03), gt("B", "2024-08-06 07:00", 0.01)}}}
	folds := []GapFold{
		{TrainFrom: gapAt("2024-01-01 00:00"), TrainTo: gapAt("2024-07-01 00:00"), TestFrom: gapAt("2024-07-01 00:00"), TestTo: gapAt("2024-10-01 00:00"),
			Combo: 0, TrainPF: 2, TrainTrades: 40, OOS: runs[0].Trades},
		{TrainFrom: gapAt("2024-04-01 00:00"), TrainTo: gapAt("2024-10-01 00:00"), TestFrom: gapAt("2024-10-01 00:00"), TestTo: gapAt("2025-01-01 00:00"), Combo: -1},
	}
	oos := PooledGapOOS(folds)
	r := GapReport{
		Generated: gapAt("2026-10-02 12:00"), From: gapAt("2024-01-01 00:00"), To: gapAt("2025-01-01 00:00"), Last12From: gapAt("2024-01-01 00:00"),
		TrainMonths: 6, TestMonths: 3, MinTrades: 30, Commission: 0.0005,
		Universe: []string{"A", "B"}, Skipped: []string{"ZZZ: дивиденды: timeout"},
		Runs: runs, Folds: folds,
		Sensitivity: []GapCostRow{{Commission: 0.0005, Trades: oos}, {Commission: 0.002, Trades: oos}},
		ExDays:      map[string]int{"A": 2},
		Verdict:     GapGate(oos, oos, gapAt("2024-01-01 00:00"), 2, 2),
	}
	md := RenderGapWFMarkdown(r)
	for _, want := range []string{
		"# gap_fade — walk-forward по пулу тикеров",
		"ZZZ: дивиденды: timeout",
		"**ПРОЙДЕН**",
		"combo-zero",
		"нет комбинации с ≥ 30 сделками",
		"∞",
		"0.20%",
		"| A | 2 |",
		"EOD на баре следующего дня: 0",
		"| сделок в pooled OOS | 2 | ≥ 2 | да |",
		"| сделок в OOS за последние 12 месяцев | 2 | ≥ 2 | да |",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("report lacks %q:\n%s", want, md)
		}
	}
	if strings.Contains(md, "неканоничные издержки") {
		t.Errorf("canonical commission must not be flagged:\n%s", md)
	}
	r.Verdict = GapGate(nil, nil, gapAt("2024-01-01 00:00"), 2, 2)
	if md := RenderGapWFMarkdown(r); !strings.Contains(md, "**ОТКАЗ**") {
		t.Errorf("failed gate must render ОТКАЗ:\n%s", md)
	}
	r.Commission = 0.001
	if md := RenderGapWFMarkdown(r); !strings.Contains(md, "**ОТКАЗ (неканоничные издержки)**") {
		t.Errorf("non-canonical commission must be flagged in the verdict heading:\n%s", md)
	}
	r.Verdict = GapGate(oos, oos, gapAt("2024-01-01 00:00"), 2, 2)
	if md := RenderGapWFMarkdown(r); !strings.Contains(md, "**ПРОЙДЕН (неканоничные издержки)**") {
		t.Errorf("non-canonical commission must be flagged in the verdict heading:\n%s", md)
	}
}

func TestGapCanonicalCommission(t *testing.T) {
	if !IsGapCanonicalCommission(0.0005) || IsGapCanonicalCommission(0.001) || IsGapCanonicalCommission(0) {
		t.Fatal("only 0.0005 per side is the canonical gap_fade commission")
	}
}
