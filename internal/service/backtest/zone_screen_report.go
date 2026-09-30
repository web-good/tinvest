package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ZoneGates are the zone screener's hard gates. MaxTickPct 0 disables the tick gate.
type ZoneGates struct {
	MinTurnoverM float64 // mean daily turnover, millions of RUB
	MinATRPct    float64 // mean weekday daily ATR, percent of close
	MaxTickPct   float64 // two price ticks as a percentage of price
}

// FilterAndRankZone applies the hard gates in order (turnover, daily ATR, price tick),
// sends signal-less survivors to their own bucket and ranks the rest by the median
// STRESSED profit factor, descending, ticker ascending on ties. A row whose tick is
// unknown passes the tick gate: the report flags it instead of guessing a cost.
func FilterAndRankZone(rows []ZoneRow, g ZoneGates) (ranked, noSignals, tickRejected, rejected []ZoneRow) {
	for _, r := range rows {
		switch {
		case r.TurnoverM < g.MinTurnoverM || r.DailyATRPct < g.MinATRPct:
			rejected = append(rejected, r)
		case g.MaxTickPct > 0 && !r.TickUnknown && r.TickPct > g.MaxTickPct:
			tickRejected = append(tickRejected, r)
		case r.NoSignals:
			noSignals = append(noSignals, r)
		default:
			ranked = append(ranked, r)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].PFMedStress != ranked[j].PFMedStress {
			return ranked[i].PFMedStress > ranked[j].PFMedStress
		}
		return ranked[i].Ticker < ranked[j].Ticker
	})
	byTicker := func(rs []ZoneRow) {
		sort.SliceStable(rs, func(i, j int) bool { return rs[i].Ticker < rs[j].Ticker })
	}
	byTicker(noSignals)
	byTicker(tickRejected)
	return ranked, noSignals, tickRejected, rejected
}

// ZoneDistribution summarizes the ranking key (stressed median PF) across the ranking.
func ZoneDistribution(ranked []ZoneRow) PFDist {
	vals := make([]float64, 0, len(ranked))
	for _, r := range ranked {
		vals = append(vals, r.PFMedStress)
	}
	return distributionOf(vals)
}

// ZoneScreenMeta carries the run context shown in the report header.
type ZoneScreenMeta struct {
	Months        int
	HoldoutMonths int
	TopN          int
	Split         time.Time
	Gates         ZoneGates
	PFCap         float64
	Commission    float64 // per side, fraction of turnover; the stress adds this again per side
	Cash          float64 // mock starting cash; half-year results are shown as a percentage of it
	Scanned       int
	Skipped       int
	Rejected      int // rows that failed the turnover or daily-ATR gate
}

func pct0(v float64) string { return fmt.Sprintf("%.0f%%", v*100) }

// RenderZoneScreenMarkdown renders the zone screening report.
func RenderZoneScreenMarkdown(ranked, noSignals, tickRejected []ZoneRow, meta ZoneScreenMeta) string {
	var b strings.Builder
	d := ZoneDistribution(ranked)
	gridSize := len(ZoneGrid())

	b.WriteString("# RSI zone screener\n\n")
	fmt.Fprintf(&b, "Окно: %d мес., holdout: последние %d мес. (срез %s).\n",
		meta.Months, meta.HoldoutMonths, meta.Split.Format("2006-01-02"))
	fmt.Fprintf(&b, "Сетка: %d конфигураций (RSIPeriod x RSILower x EMAPeriod x RSIUpper), стоп 1.0 дневного ATR.\n", gridSize)
	fmt.Fprintf(&b, "Гейты: оборот >= %.0f млн ₽/день, дневной ATR >= %.2f%%, два шага цены <= %.2f%% цены (0 — гейт выключен). PF зажат сверху на %.1f.\n",
		meta.Gates.MinTurnoverM, meta.Gates.MinATRPct, meta.Gates.MaxTickPct, meta.PFCap)
	fmt.Fprintf(&b, "Стресс издержек (`PFmed×2`): к комиссии %.4f за сторону добавлена ещё одна такая же и два шага цены на круг.\n",
		meta.Commission)
	fmt.Fprintf(&b, "Вселенная: scanned=%d ranked=%d no-signal=%d tick-rejected=%d rejected=%d skipped=%d.\n\n",
		meta.Scanned, len(ranked), len(noSignals), len(tickRejected), meta.Rejected, meta.Skipped)

	b.WriteString("## Распределение PFmed×2 по ранжированной вселенной\n\n")
	fmt.Fprintf(&b, "min %.2f · Q1 %.2f · медиана %.2f · Q3 %.2f · max %.2f · доля PFmed×2 >= 1.5: %.0f%% (n=%d)\n\n",
		d.Min, d.Q1, d.Median, d.Q3, d.Max, d.ShareAbove15*100, d.N)
	b.WriteString("Читать эту строку раньше первой строки топа: если планку проходит половина вселенной, планка ничего не значит.\n\n")

	top := ranked
	if meta.TopN > 0 && meta.TopN < len(top) {
		top = top[:meta.TopN]
	}

	b.WriteString("## Рейтинг\n\n")
	b.WriteString("| # | Ticker | Name | Оборот, млн | ATR% дн | Tick% | Бары | TradesMed | PFmed | PFmed×2 | Capped | SilentCfg | Plateau | PFmed×2 HO | Trades HO |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range top {
		tick := fmt.Sprintf("%.2f", r.TickPct)
		if r.TickUnknown {
			tick = "?"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %.0f | %.2f | %s | %d | %.0f | %.2f | %.2f | %d/%d | %d/%d | %s | %.2f | %.0f |\n",
			i+1, r.Ticker, r.Name, r.TurnoverM, r.DailyATRPct, tick, r.Bars, r.TradesMed,
			r.PFMed, r.PFMedStress, r.Capped, gridSize, r.SilentCfg, gridSize,
			pct0(r.Plateau), r.PFMedStressHO, r.TradesMedHO)
	}
	b.WriteString("\n`Tick%` `?` — API не отдал шаг цены: издержки шага не измерены, стресс учитывает только комиссию.\n")
	b.WriteString("`PFmed×2 HO` в сортировке не участвует: это красный флаг («работало и развалилось»), а не критерий отбора.\n\n")

	b.WriteString("## Детали топа\n\n")
	b.WriteString("| Ticker | Halves+ | TopHalf | SL% | HoldD | >EMA50 | >EMA200 | Лучшая конфигурация | Полугодия лучшей конфигурации |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|\n")
	for _, r := range top {
		topHalf := "—"
		if r.TopHalf > 0 {
			topHalf = pct0(r.TopHalf)
		}
		best := fmt.Sprintf("RSI %d %.0f/%.0f, EMA %d", r.Best.RSIPeriod, r.Best.RSILower, r.Best.RSIUpper, r.Best.EMAPeriod)
		halves := make([]string, 0, len(r.BestHalves))
		for _, h := range r.BestHalves {
			share := 0.0
			if meta.Cash > 0 {
				share = h.PnL / meta.Cash * 100
			}
			halves = append(halves, fmt.Sprintf("%s %+.1f%%", h.Label, share))
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %.1f | %s | %s | %s | %s |\n",
			r.Ticker, pct0(r.HalvesPos), topHalf, pct0(r.SLShare), r.HoldDays,
			pct0(r.AboveEMA50), pct0(r.AboveEMA200), best, strings.Join(halves, " · "))
	}
	b.WriteString("\n`Halves+` — доля прибыльных полугодий; `TopHalf` — доля лучшего полугодия в сумме прибыльных (100% — весь профит из одного отрезка, «—» — прибыльных нет); ")
	b.WriteString("`SL%` — доля выходов по стопу; `HoldD` — медианная длительность сделки, дней; `>EMA50`/`>EMA200` — доля train-баров выше EMA. ")
	b.WriteString("Полугодия лучшей конфигурации — результат train-сделок в процентах от стартового кэша.\n\n")

	if len(noSignals) > 0 {
		b.WriteString("## Нет сигналов\n\n")
		b.WriteString("Тикеры, прошедшие гейты, но не давшие ни одной сделки ни в одной из конфигураций:\n\n")
		names := make([]string, 0, len(noSignals))
		for _, r := range noSignals {
			names = append(names, r.Ticker)
		}
		fmt.Fprintf(&b, "%s\n\n", strings.Join(names, ", "))
	}

	if len(tickRejected) > 0 {
		b.WriteString("## Отсеяно гейтом шага цены\n\n")
		b.WriteString("| Ticker | Tick% | PFmed | PFmed×2 |\n|---|---|---|---|\n")
		for _, r := range tickRejected {
			fmt.Fprintf(&b, "| %s | %.2f | %.2f | %.2f |\n", r.Ticker, r.TickPct, r.PFMed, r.PFMedStress)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Как это читать\n\n")
	b.WriteString("Шортлист — это **кандидаты на калибровку**, а не доказательство edge. ")
	b.WriteString("Верх такого рейтинга по построению содержит везунчиков: испытаний столько же, сколько тикеров умножить на конфигурации. ")
	b.WriteString("Планка приёмки не меняется — pooled OOS profit factor >= 1.5 в персональном walk-forward (docs/rsi_zone/strategy.md).\n")
	return b.String()
}
