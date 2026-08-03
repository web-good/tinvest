package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ScreenMeta carries the run context shown in the report header.
type ScreenMeta struct {
	Months        int
	HoldoutMonths int
	TopN          int
	Split         time.Time // train/holdout boundary
	MinTurnoverM  float64
	MinATRPct     float64
	PFCap         float64
	Scanned       int // universe size after the currency/trading filter
	Passed        int // rows that cleared both gates
	Skipped       int // tickers whose candles failed to load
}

// FilterAndRank applies the two hard gates and splits the survivors into the ranking
// (sorted by median profit factor, descending) and the no-signal bucket. Minimum
// history and minimum trade count are deliberately NOT gates — they are columns the
// reader judges; only liquidity and daily ATR% exclude a ticker.
func FilterAndRank(rows []PullbackRow, minTurnoverM, minATRPct float64) (ranked, noSignals, rejected []PullbackRow) {
	for _, r := range rows {
		if r.TurnoverM < minTurnoverM || r.DailyATRPct < minATRPct {
			rejected = append(rejected, r)
			continue
		}
		if r.NoSignals {
			noSignals = append(noSignals, r)
			continue
		}
		ranked = append(ranked, r)
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].PFMed > ranked[j].PFMed })
	sort.SliceStable(noSignals, func(i, j int) bool { return noSignals[i].Ticker < noSignals[j].Ticker })
	return ranked, noSignals, rejected
}

// PFDist is the spread of median profit factor across the ranked universe. Without
// it the top of the table is unreadable: 271 tickers times 24 configurations is 6504
// trials, so some rows are lucky by construction. If half the universe clears 1.5,
// the bar means nothing — and that must be visible in the report, not discovered a
// month of calibrations later.
type PFDist struct {
	Min, Q1, Median, Q3, Max float64
	ShareAbove15             float64 // share of ranked tickers with PFMed >= 1.5
	N                        int
}

// Distribution summarizes PFMed across the ranked rows.
func Distribution(ranked []PullbackRow) PFDist {
	if len(ranked) == 0 {
		return PFDist{}
	}
	vals := make([]float64, 0, len(ranked))
	var above int
	for _, r := range ranked {
		vals = append(vals, r.PFMed)
		if r.PFMed >= 1.5 {
			above++
		}
	}
	sort.Float64s(vals)
	return PFDist{
		Min:          vals[0],
		Q1:           medianF(vals[:len(vals)/2]),
		Median:       medianF(vals),
		Q3:           medianF(vals[(len(vals)+1)/2:]),
		Max:          vals[len(vals)-1],
		ShareAbove15: float64(above) / float64(len(ranked)),
		N:            len(ranked),
	}
}

// RenderPullbackScreenMarkdown renders the screening report.
func RenderPullbackScreenMarkdown(ranked, noSignals []PullbackRow, meta ScreenMeta) string {
	var b strings.Builder
	d := Distribution(ranked)

	b.WriteString("# RSI pullback screener\n\n")
	fmt.Fprintf(&b, "Окно: %d мес., holdout: последние %d мес. (срез %s).\n",
		meta.Months, meta.HoldoutMonths, meta.Split.Format("2006-01-02"))
	fmt.Fprintf(&b, "Сетка: %d конфигураций (RSIPeriod x RSILower x EMASlow x TPDailyATR), объёмный гейт и трейл выключены.\n",
		len(PullbackGrid()))
	fmt.Fprintf(&b, "Гейты: оборот >= %.0f млн ₽/день, дневной ATR >= %.2f%%. PF зажат сверху на %.1f.\n",
		meta.MinTurnoverM, meta.MinATRPct, meta.PFCap)
	fmt.Fprintf(&b, "Вселенная: scanned=%d passed=%d no-signal=%d skipped=%d.\n\n",
		meta.Scanned, meta.Passed, len(noSignals), meta.Skipped)

	b.WriteString("## Распределение PFmed по прошедшей вселенной\n\n")
	fmt.Fprintf(&b, "min %.2f · Q1 %.2f · медиана %.2f · Q3 %.2f · max %.2f · доля PFmed >= 1.5: %.0f%% (n=%d)\n\n",
		d.Min, d.Q1, d.Median, d.Q3, d.Max, d.ShareAbove15*100, d.N)
	b.WriteString("Читать эту строку раньше первой строки топа: если планку проходит половина вселенной, планка ничего не значит.\n\n")

	b.WriteString("## Рейтинг\n\n")
	b.WriteString("| # | Ticker | Name | Оборот, млн | ATR% дн | Бары | TradesMed | PFmed | Plateau | PFmed HO | Trades HO | Лучшая конфигурация |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	limit := len(ranked)
	if meta.TopN > 0 && meta.TopN < limit {
		limit = meta.TopN
	}
	for i, r := range ranked[:limit] {
		best := fmt.Sprintf("RSI %d/%.0f, EMA %d/%d, TP %.1f",
			r.Best.RSIPeriod, r.Best.RSILower, r.Best.EMAFast, r.Best.EMASlow, r.Best.TPDailyATR)
		fmt.Fprintf(&b, "| %d | %s | %s | %.0f | %.2f | %d | %.0f | %.2f | %.0f%% | %.2f | %.0f | %s |\n",
			i+1, r.Ticker, r.Name, r.TurnoverM, r.DailyATRPct, r.Bars,
			r.TradesMed, r.PFMed, r.Plateau*100, r.PFMedHO, r.TradesMedHO, best)
	}
	b.WriteString("\nКолонка «лучшая конфигурация» — справочная стартовая точка для ручной калибровки, а не рекомендация.\n")
	b.WriteString("`PFmed HO` в сортировке не участвует: это красный флаг («работало и развалилось»), а не критерий отбора.\n\n")

	if len(noSignals) > 0 {
		b.WriteString("## Нет сигналов\n\n")
		b.WriteString("Тикеры, прошедшие гейты, но не давшие ни одной сделки ни в одной из конфигураций — profit factor у них не существует:\n\n")
		names := make([]string, 0, len(noSignals))
		for _, r := range noSignals {
			names = append(names, r.Ticker)
		}
		fmt.Fprintf(&b, "%s\n\n", strings.Join(names, ", "))
	}

	b.WriteString("## Как это читать\n\n")
	b.WriteString("Шортлист — это **кандидаты на калибровку**, а не доказательство edge. ")
	b.WriteString("Верх такого рейтинга по построению содержит везунчиков: испытаний столько же, сколько тикеров умножить на конфигурации. ")
	b.WriteString("Планка приёмки не меняется — pooled OOS profit factor >= 1.5 в персональном walk-forward (docs/rsi_pullback/strategy.md, §8).\n")
	return b.String()
}
