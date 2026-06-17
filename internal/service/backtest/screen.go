package backtest

import (
	"fmt"
	"sort"
	"strings"
)

// ScreenRow is one ticker's mean-reversion screen result.
type ScreenRow struct {
	Ticker    string
	VR2       float64
	VR4       float64
	VR8       float64
	Autocorr1 float64
	Verdict   string
	Note      string // non-empty when the ticker was skipped (e.g. no candles)
}

// RenderScreenMarkdown renders the screen as a Markdown table ranked by VR(2)
// ascending (most mean-reverting first). Skipped rows (Note set) sort last.
func RenderScreenMarkdown(rows []ScreenRow) string {
	sorted := make([]ScreenRow, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		if (sorted[i].Note == "") != (sorted[j].Note == "") {
			return sorted[i].Note == "" // scored rows before skipped
		}
		return sorted[i].VR2 < sorted[j].VR2
	})

	var b strings.Builder
	b.WriteString("# Скрининг возврата к среднему (variance ratio)\n\n")
	b.WriteString("VR<1 — возврат к среднему; VR>1 — тренд. Ранжир по VR(2).\n\n")
	b.WriteString("| Тикер | VR(2) | VR(4) | VR(8) | Autocorr(1) | Вердикт |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, r := range sorted {
		if r.Note != "" {
			fmt.Fprintf(&b, "| %s | — | — | — | — | %s |\n", r.Ticker, r.Note)
			continue
		}
		fmt.Fprintf(&b, "| %s | %.3f | %.3f | %.3f | %.3f | %s |\n",
			r.Ticker, r.VR2, r.VR4, r.VR8, r.Autocorr1, r.Verdict)
	}
	return b.String()
}
