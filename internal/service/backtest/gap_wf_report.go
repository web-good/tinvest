package backtest

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// GapCostRow is the pooled OOS sample replayed at one per-side commission.
type GapCostRow struct {
	Commission float64
	Trades     []GapTrade
}

// GapReport is everything cmd/gapwf renders.
type GapReport struct {
	Generated, From, To, Last12From time.Time
	TrainMonths, TestMonths         int
	MinTrades                       int
	Commission                      float64
	Universe, Skipped               []string
	Runs                            []GapComboRun
	Folds                           []GapFold
	Sensitivity                     []GapCostRow
	ExDays                          map[string]int // ticker -> distinct ex-days on which a trade was dropped
	Verdict                         GapVerdict
}

func fmtGapPF(pf float64) string {
	if math.IsInf(pf, 1) {
		return "∞"
	}
	return fmt.Sprintf("%.3f", pf)
}

// GapVerdictStatus is the verdict word, flagged when the run's commission is not the one the
// gate thresholds were set for (0.05% per side).
func GapVerdictStatus(v GapVerdict, commission float64) string {
	status := "ОТКАЗ"
	if v.Pass {
		status = "ПРОЙДЕН"
	}
	if !IsGapCanonicalCommission(commission) {
		status += " (неканоничные издержки)"
	}
	return status
}

func gapDate(t time.Time) string { return t.In(gapLoc).Format("2006-01-02") }

func writeGapGroups(b *strings.Builder, title, keyHeader string, groups []GapGroup) {
	fmt.Fprintf(b, "\n## %s\n\n| %s | Сделок | PF | Сумма доходностей, %% |\n|---|---|---|---|\n", title, keyHeader)
	for _, g := range groups {
		fmt.Fprintf(b, "| %s | %d | %s | %.2f |\n", g.Key, g.Trades, fmtGapPF(g.PF), 100*g.SumPct)
	}
}

// RenderGapWFMarkdown renders the pooled walk-forward report: run settings, the gate verdict,
// the full-window ranking, the folds, pooled OOS slices, cost sensitivity, breakdowns and the
// dividend ex-day filter.
func RenderGapWFMarkdown(r GapReport) string {
	var b strings.Builder
	oos := PooledGapOOS(r.Folds)

	b.WriteString("# gap_fade — walk-forward по пулу тикеров\n\n")
	fmt.Fprintf(&b, "- Сформирован: %s\n", r.Generated.In(gapLoc).Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "- Окно: %s — %s, фолды %d/%d мес., min-trades %d, издержки %.2f%% на сторону\n",
		gapDate(r.From), gapDate(r.To), r.TrainMonths, r.TestMonths, r.MinTrades, 100*r.Commission)
	fmt.Fprintf(&b, "- Вселенная (%d): %s\n", len(r.Universe), strings.Join(r.Universe, ", "))
	if len(r.Skipped) > 0 {
		b.WriteString("- Пропущены:\n")
		for _, s := range r.Skipped {
			fmt.Fprintf(&b, "  - %s\n", s)
		}
	}

	b.WriteString("\n## Вердикт гейта\n\n")
	fmt.Fprintf(&b, "**%s**\n\n", GapVerdictStatus(r.Verdict, r.Commission))
	b.WriteString("| Условие | Значение | Порог | Итог |\n|---|---|---|---|\n")
	for _, c := range r.Verdict.Checks {
		res := "нет"
		if c.Pass {
			res = "да"
		}
		if c.Count {
			fmt.Fprintf(&b, "| %s | %d | ≥ %d | %s |\n", c.Name, int(c.Value), int(c.Threshold), res)
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | ≥ %.2f | %s |\n", c.Name, fmtGapPF(c.Value), c.Threshold, res)
	}

	b.WriteString("\n## Топ-10 комбинаций на полном окне\n\n| # | Параметры | PF | Сделок | Прибыльных тикеров |\n|---|---|---|---|---|\n")
	for i, k := range RankGapCombos(r.Runs, r.MinTrades, 10) {
		fmt.Fprintf(&b, "| %d | %+v | %s | %d | %d из %d |\n", i+1, r.Runs[k.Combo].Params, fmtGapPF(k.PF), k.Trades, k.ProfitableTickers, k.TradedTickers)
	}

	b.WriteString("\n## Фолды\n\n| # | Обучение | Тест | Параметры | Train PF | Train сделок | OOS PF | OOS сделок |\n|---|---|---|---|---|---|---|---|\n")
	for i, f := range r.Folds {
		train := fmt.Sprintf("%s — %s", gapDate(f.TrainFrom), gapDate(f.TrainTo))
		test := fmt.Sprintf("%s — %s", gapDate(f.TestFrom), gapDate(f.TestTo))
		if f.Combo < 0 {
			fmt.Fprintf(&b, "| %d | %s | %s | нет комбинации с ≥ %d сделками | — | — | — | 0 |\n", i+1, train, test, r.MinTrades)
			continue
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %+v | %s | %d | %s | %d |\n", i+1, train, test, r.Runs[f.Combo].Params,
			fmtGapPF(f.TrainPF), f.TrainTrades, fmtGapPF(GapPF(f.OOS)), len(f.OOS))
	}

	last12 := GapTradesFrom(oos, r.Last12From)
	b.WriteString("\n## Pooled OOS\n\n| Срез | Сделок | PF |\n|---|---|---|\n")
	fmt.Fprintf(&b, "| весь OOS | %d | %s |\n", len(oos), fmtGapPF(GapPF(oos)))
	fmt.Fprintf(&b, "| с %s (последние 12 месяцев) | %d | %s |\n", gapDate(r.Last12From), len(last12), fmtGapPF(GapPF(last12)))

	b.WriteString("\n## Чувствительность к издержкам\n\n| Издержки на сторону | Сделок | PF |\n|---|---|---|\n")
	for _, row := range r.Sensitivity {
		fmt.Fprintf(&b, "| %.2f%% | %d | %s |\n", 100*row.Commission, len(row.Trades), fmtGapPF(GapPF(row.Trades)))
	}

	writeGapGroups(&b, "OOS по тикерам", "Тикер", GapByTicker(oos))
	writeGapGroups(&b, "OOS по годам", "Год", GapByYear(oos))
	writeGapGroups(&b, "OOS по кварталам", "Квартал", GapByQuarter(oos))
	writeGapGroups(&b, "Выходы OOS", "Причина", GapByReason(oos))
	fmt.Fprintf(&b, "\nEOD на баре следующего дня: %d\n", GapNextDayEOD(oos))

	b.WriteString("\n## Дивидендные отсечки\n\n")
	tickers := make([]string, 0, len(r.ExDays))
	for t, n := range r.ExDays {
		if n > 0 {
			tickers = append(tickers, t)
		}
	}
	sort.Strings(tickers)
	if len(tickers) == 0 {
		b.WriteString("Сделок в дни отсечки не было.\n")
	} else {
		b.WriteString("| Тикер | Дней отсечки с выброшенными сделками |\n|---|---|\n")
		for _, t := range tickers {
			fmt.Fprintf(&b, "| %s | %d |\n", t, r.ExDays[t])
		}
	}
	return b.String()
}
