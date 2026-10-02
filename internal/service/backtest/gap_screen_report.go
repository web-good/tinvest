package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// GapScreenMeta is the run context printed in the screener report header.
type GapScreenMeta struct {
	Generated time.Time
	Months    int
	Scanned   int
	Opts      GapScreenOptions
	NoData    []string // "TICKER: reason" for tickers without usable candles
}

// GapScreenPassed returns the rows that passed every gate, by median 07:00 turnover descending
// (ticker ascending on ties) — the most executable entries first.
func GapScreenPassed(rows []GapScreenRow) []GapScreenRow {
	var out []GapScreenRow
	for _, r := range rows {
		if r.Pass {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MorningTurnoverM != out[j].MorningTurnoverM {
			return out[i].MorningTurnoverM > out[j].MorningTurnoverM
		}
		return out[i].Ticker < out[j].Ticker
	})
	return out
}

// GapScreenTickersLine is the env line for env/prod.env: passed tickers in report order.
func GapScreenTickersLine(rows []GapScreenRow) string {
	passed := GapScreenPassed(rows)
	t := make([]string, len(passed))
	for i, r := range passed {
		t[i] = r.Ticker
	}
	return "GAP_FADE_TICKERS=" + strings.Join(t, ",")
}

// RenderGapScreenMarkdown renders the screener report.
func RenderGapScreenMarkdown(rows []GapScreenRow, meta GapScreenMeta) string {
	passed := GapScreenPassed(rows)
	var rejected []GapScreenRow
	for _, r := range rows {
		if !r.Pass {
			rejected = append(rejected, r)
		}
	}
	sort.Slice(rejected, func(i, j int) bool { return rejected[i].Ticker < rejected[j].Ticker })

	tick := fmt.Sprintf("два шага ≤ %.2f%%", meta.Opts.MaxTickPct)
	if meta.Opts.MaxTickPct == 0 {
		tick = "два шага — выкл."
	}
	var b strings.Builder
	b.WriteString("# gap_fade — скринер тикеров\n\n")
	fmt.Fprintf(&b, "- Сформирован: %s\n", meta.Generated.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "- Окно: %d мес., просмотрено тикеров: %d, прошли: %d, отсеяны: %d, нет данных: %d\n",
		meta.Months, meta.Scanned, len(passed), len(rejected), len(meta.NoData))
	fmt.Fprintf(&b, "- Гейты: бар 07:00 ≥ %.0f%% будних дней; утренний оборот ≥ %.1f млн ₽; %s; дневной оборот ≥ %.0f млн ₽\n",
		meta.Opts.MinMorningDays*100, meta.Opts.MinMorningTurnoverM, tick, meta.Opts.MinTurnoverM)
	b.WriteString("- Сигналов в год — число сделок gap_fade с общими параметрами, справочно: дни дивидендных отсечек не выброшены, прибыль не считается.\n\n")

	b.WriteString("## Итог\n\n```\n")
	b.WriteString(GapScreenTickersLine(rows))
	b.WriteString("\n```\n\n")

	b.WriteString("## Прошли\n\n")
	if len(passed) == 0 {
		b.WriteString("Нет.\n\n")
	} else {
		b.WriteString("| Тикер | Название | Бар 07:00, дней | Утренний оборот, млн ₽ | Дневной оборот, млн ₽ | Два шага | Сигналов в год |\n")
		b.WriteString("|---|---|---|---|---|---|---|\n")
		for _, r := range passed {
			t := "—"
			if r.TickPct > 0 {
				t = fmt.Sprintf("%.2f%%", r.TickPct)
			}
			fmt.Fprintf(&b, "| %s | %s | %.0f%% | %.1f | %.0f | %s | %.1f |\n",
				r.Ticker, r.Name, r.MorningDays*100, r.MorningTurnoverM, r.TurnoverM, t, r.SignalsPerYear)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Отсеяны\n\n")
	if len(rejected) == 0 {
		b.WriteString("Нет.\n\n")
	} else {
		b.WriteString("| Тикер | Название | Причины |\n|---|---|---|\n")
		for _, r := range rejected {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", r.Ticker, r.Name, strings.Join(r.Reasons, "; "))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Нет данных\n\n")
	if len(meta.NoData) == 0 {
		b.WriteString("Нет.\n")
	} else {
		for _, s := range meta.NoData {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}
	return b.String()
}
