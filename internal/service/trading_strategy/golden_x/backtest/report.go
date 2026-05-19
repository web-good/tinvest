package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

type Stats struct {
	Count, Wins, Losses, Open int
	WinRate                   float64
	AvgReturnPct              float64
	MedianReturn              float64
	CumulativeReturn          float64
	MaxDrawdown               float64
	AvgWeeksHeld              float64
	ExitReasons               map[ExitReason]int
}

type Report struct {
	Kind     dto.StrategyKind
	From, To time.Time
	Trades   []Trade
	PerShare map[string]Stats
	Overall  Stats
}

func AggregateStats(trades []Trade) Stats {
	out := Stats{ExitReasons: map[ExitReason]int{}}
	if len(trades) == 0 {
		return out
	}
	var sumReturn, sumWeeks float64
	returns := make([]float64, 0, len(trades))
	for _, tr := range trades {
		out.Count++
		out.ExitReasons[tr.ExitReason]++
		out.CumulativeReturn += tr.ReturnPct * tr.Units
		sumReturn += tr.ReturnPct
		sumWeeks += float64(tr.WeeksHeld)
		returns = append(returns, tr.ReturnPct)
		switch {
		case tr.ExitReason == ExitReasonOpen:
			out.Open++
		case tr.ReturnPct > 0:
			out.Wins++
		case tr.ReturnPct < 0:
			out.Losses++
		}
	}
	out.AvgReturnPct = sumReturn / float64(out.Count)
	out.AvgWeeksHeld = sumWeeks / float64(out.Count)
	sort.Float64s(returns)
	out.MedianReturn = median(returns)
	if out.Wins+out.Losses > 0 {
		out.WinRate = float64(out.Wins) / float64(out.Wins+out.Losses)
	}
	out.MaxDrawdown = maxDrawdown(trades)
	return out
}

func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func maxDrawdown(trades []Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	ordered := make([]Trade, len(trades))
	copy(ordered, trades)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].EntryDate.Before(ordered[j].EntryDate)
	})
	var equity, peak, worst float64
	for _, tr := range ordered {
		equity += tr.ReturnPct * tr.Units
		if equity > peak {
			peak = equity
		}
		dd := peak - equity
		if dd > worst {
			worst = dd
		}
	}
	return worst
}

func RenderMarkdown(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Golden X backtest — %s\n\n", r.Kind.Medal())
	fmt.Fprintf(&b, "_Range: %s → %s · generated %s_\n\n",
		r.From.Format("2006-01-02"), r.To.Format("2006-01-02"), time.Now().Format(time.RFC3339))

	b.WriteString("## Overall\n\n")
	writeStatsTable(&b, r.Overall)
	b.WriteString("\n")

	b.WriteString("## Exit reasons\n\n")
	writeExitReasons(&b, r.Trades)
	b.WriteString("\n")

	b.WriteString("## Per share\n\n")
	writePerShare(&b, r.PerShare)
	b.WriteString("\n")

	b.WriteString("## Trades (chronological)\n\n")
	writeTrades(&b, r.Trades)
	return b.String()
}

func writeStatsTable(b *strings.Builder, s Stats) {
	b.WriteString("| Count | Wins | Losses | Open | WinRate | AvgReturn% | Median% | Cumulative% | MaxDD% | AvgWeeks |\n")
	b.WriteString("|------:|-----:|-------:|-----:|--------:|-----------:|--------:|------------:|-------:|---------:|\n")
	fmt.Fprintf(b, "| %d | %d | %d | %d | %.2f | %.2f | %.2f | %.2f | %.2f | %.1f |\n",
		s.Count, s.Wins, s.Losses, s.Open, s.WinRate*100, s.AvgReturnPct, s.MedianReturn,
		s.CumulativeReturn, s.MaxDrawdown, s.AvgWeeksHeld)
}

func writeExitReasons(b *strings.Builder, trades []Trade) {
	counts := map[ExitReason]int{}
	sums := map[ExitReason]float64{}
	for _, tr := range trades {
		counts[tr.ExitReason]++
		sums[tr.ExitReason] += tr.ReturnPct
	}
	reasons := make([]ExitReason, 0, len(counts))
	for r := range counts {
		reasons = append(reasons, r)
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })
	b.WriteString("| Reason | Count | AvgReturn% |\n|---|---:|---:|\n")
	for _, r := range reasons {
		n := counts[r]
		fmt.Fprintf(b, "| %s | %d | %.2f |\n", r, n, sums[r]/float64(n))
	}
}

func writePerShare(b *strings.Builder, per map[string]Stats) {
	ids := make([]string, 0, len(per))
	for id := range per {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	b.WriteString("| Share | Count | WinRate | Cumulative% | MaxDD% |\n|---|---:|---:|---:|---:|\n")
	for _, id := range ids {
		s := per[id]
		fmt.Fprintf(b, "| %s | %d | %.2f | %.2f | %.2f |\n",
			id, s.Count, s.WinRate*100, s.CumulativeReturn, s.MaxDrawdown)
	}
}

func writeTrades(b *strings.Builder, trades []Trade) {
	ordered := make([]Trade, len(trades))
	copy(ordered, trades)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].EntryDate.Before(ordered[j].EntryDate) })
	b.WriteString("| Share | Entry | Exit | EntryPx | ExitPx | Units | Reason | Return% | Weeks |\n|---|---|---|---:|---:|---:|---|---:|---:|\n")
	for _, tr := range ordered {
		fmt.Fprintf(b, "| %s | %s | %s | %.2f | %.2f | %.3f | %s | %.2f | %d |\n",
			tr.ShareID, tr.EntryDate.Format("2006-01-02"), tr.ExitDate.Format("2006-01-02"),
			tr.EntryPrice, tr.ExitPrice, tr.Units, tr.ExitReason, tr.ReturnPct, tr.WeeksHeld)
	}
}
