package dividend

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func Render(ranked []RankedShare, stats Stats) string {
	var b strings.Builder
	b.WriteString("<b>🏆 Дивидендный скринер (Мосбиржа)</b>\n\n")
	fmt.Fprintf(&b, "<i>Вселенная: %d · в рейтинге: %d · отсеяно: %d</i>\n\n", stats.Universe, stats.Ranked, stats.Gated)

	for i, rs := range ranked {
		f := rs.Scored
		trap := ""
		if f.YieldTrap {
			trap = " ⚠️"
		}
		fmt.Fprintf(&b, "<b>%d. %s</b> (%s)%s\n", i+1, htmlEscape(rs.Share.Name), rs.Share.Ticker, trap)
		fmt.Fprintf(&b, "   Композит: <b>%.0f</b>/100\n", f.Composite)
		fmt.Fprintf(&b, "   Устойчивость %.2f · Долг %.2f · Рост %.2f · Качество %.2f · Оценка %.2f\n\n",
			f.Sustainability, f.Safety, f.DivGrowth, f.Quality, f.Valuation)
	}

	if len(stats.ByReason) > 0 {
		b.WriteString("<i>Отсеяно по причинам:</i>\n")
		reasons := make([]string, 0, len(stats.ByReason))
		for r := range stats.ByReason {
			reasons = append(reasons, r)
		}
		sort.Strings(reasons)
		for _, r := range reasons {
			fmt.Fprintf(&b, "· %s — %d\n", r, stats.ByReason[r])
		}
	}

	fmt.Fprintf(&b, "\n<i>Данные на %s</i>", time.Now().Format("02.01.2006 15:04"))
	return b.String()
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
