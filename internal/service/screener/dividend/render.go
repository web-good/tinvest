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

	writeSectorGroups(&b, ranked)

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

// sectorGroup — сектор-бакет топ-N: хранит индексы карточек в исходном
// срезе ranked, чтобы нумерация карточек оставалась сквозной (глобальной)
// после группировки по секторам.
type sectorGroup struct {
	sector string
	items  []int // индексы в исходном срезе ranked
}

// writeSectorGroups группирует ranked по Share.Sector (порядок внутри бакета
// сохраняется — вход уже отсортирован по composite desc), упорядочивает
// бакеты по максимальному composite (бакет с #1 идёт первым) и печатает
// подзаголовок-агрегат перед карточками каждого сектора.
func writeSectorGroups(b *strings.Builder, ranked []RankedShare) {
	if len(ranked) == 0 {
		return
	}

	order := make([]string, 0)
	groups := make(map[string]*sectorGroup)
	for i, rs := range ranked {
		sector := rs.Share.Sector
		g, ok := groups[sector]
		if !ok {
			g = &sectorGroup{sector: sector}
			groups[sector] = g
			order = append(order, sector)
		}
		g.items = append(g.items, i)
	}

	sort.SliceStable(order, func(i, j int) bool {
		return maxComposite(ranked, groups[order[i]]) > maxComposite(ranked, groups[order[j]])
	})

	for _, sector := range order {
		g := groups[sector]
		var sum float64
		for _, idx := range g.items {
			sum += ranked[idx].Scored.Composite
		}
		avg := sum / float64(len(g.items))
		fmt.Fprintf(b, "<b>%s</b> · %d %s · ср. %.0f\n\n",
			sectorLabel(sector), len(g.items), plural(len(g.items), "имя", "имени", "имён"), avg)

		for _, idx := range g.items {
			writeCard(b, idx, ranked[idx])
		}
	}
}

func maxComposite(ranked []RankedShare, g *sectorGroup) float64 {
	m := ranked[g.items[0]].Scored.Composite
	for _, idx := range g.items[1:] {
		if c := ranked[idx].Scored.Composite; c > m {
			m = c
		}
	}
	return m
}

func writeCard(b *strings.Builder, pos int, rs RankedShare) {
	f := rs.Scored
	trap := ""
	if f.YieldTrap {
		trap = " ⚠️"
	}
	fmt.Fprintf(b, "<b>%d. %s</b> (%s)%s\n", pos+1, htmlEscape(rs.Share.Name), rs.Share.Ticker, trap)
	fmt.Fprintf(b, "   Композит: <b>%.0f</b>/100\n", f.Composite)
	fmt.Fprintf(b, "   Устойчивость %.2f · Долг %.2f · Рост %.2f · Качество %.2f · Оценка %.2f\n\n",
		f.Sustainability, f.Safety, f.DivGrowth, f.Quality, f.Valuation)
}

func sectorLabel(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "financial":
		return "🏦 Финансы"
	case "energy":
		return "🛢 Нефтегаз"
	case "utilities":
		return "⚡ Энергетика"
	case "materials":
		return "⛏ Материалы"
	case "telecom":
		return "📡 Телеком"
	case "consumer":
		return "🛒 Потребительский"
	case "it":
		return "💻 IT"
	case "health_care":
		return "💊 Здравоохранение"
	case "industrials":
		return "🏭 Промышленность"
	case "real_estate":
		return "🏢 Недвижимость"
	default:
		return "📊 Прочее"
	}
}

func plural(n int, one, few, many string) string {
	nn := n % 100
	if nn >= 11 && nn <= 14 {
		return many
	}
	switch n % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	default:
		return many
	}
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
