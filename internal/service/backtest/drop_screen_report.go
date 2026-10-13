package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"tinvest/internal/enum"
)

// DropShortlistKey names the shortlist line the report and the console print: the calibration
// queue for the drop-bounce-calibrate skill.
const DropShortlistKey = "DROP_BOUNCE_CANDIDATES"

// DropGates are the drop screener's hard gates. MaxTickPct 0 disables the tick gate. Equality
// with a threshold passes.
type DropGates struct {
	MinTurnoverM float64 // mean daily turnover, millions of RUB
	MaxTickPct   float64 // two price ticks as a percentage of price
	MinTrades    float64 // median train trades across the grid
}

// DropRejectReason names every turnover/trades gate the row fails, "" when it passes both.
func DropRejectReason(r DropRow, g DropGates) string {
	var reasons []string
	if r.TurnoverM < g.MinTurnoverM {
		reasons = append(reasons, fmt.Sprintf("оборот %.1f < %.0f млн ₽", r.TurnoverM, g.MinTurnoverM))
	}
	if r.TradesMed < g.MinTrades {
		reasons = append(reasons, fmt.Sprintf("сделок %.0f < %.0f", r.TradesMed, g.MinTrades))
	}
	return strings.Join(reasons, "; ")
}

// FilterAndRankDrop sends signal-less rows to their own bucket first (a silent ticker is not
// "too few trades"), then applies the price-tick gate, then turnover and trades. Survivors rank by
// the median STRESSED profit factor descending, then median trades descending, then ticker. A row
// whose tick is unknown passes the tick gate: the report flags it instead of guessing a cost.
func FilterAndRankDrop(rows []DropRow, g DropGates) (ranked, noSignals, tickRejected, rejected []DropRow) {
	for _, r := range rows {
		switch {
		case r.NoSignals:
			noSignals = append(noSignals, r)
		case g.MaxTickPct > 0 && !r.TickUnknown && r.TickPct > g.MaxTickPct:
			tickRejected = append(tickRejected, r)
		case DropRejectReason(r, g) != "":
			rejected = append(rejected, r)
		default:
			ranked = append(ranked, r)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].PFMedStress != ranked[j].PFMedStress {
			return ranked[i].PFMedStress > ranked[j].PFMedStress
		}
		if ranked[i].TradesMed != ranked[j].TradesMed {
			return ranked[i].TradesMed > ranked[j].TradesMed
		}
		return ranked[i].Ticker < ranked[j].Ticker
	})
	byTicker := func(rs []DropRow) {
		sort.SliceStable(rs, func(i, j int) bool { return rs[i].Ticker < rs[j].Ticker })
	}
	byTicker(noSignals)
	byTicker(tickRejected)
	byTicker(rejected)
	return ranked, noSignals, tickRejected, rejected
}

// DropShortlist returns the first n tickers of the ranking; n <= 0 or beyond the ranking returns
// all of it.
func DropShortlist(ranked []DropRow, n int) []string {
	if n <= 0 || n > len(ranked) {
		n = len(ranked)
	}
	out := make([]string, 0, n)
	for _, r := range ranked[:n] {
		out = append(out, r.Ticker)
	}
	return out
}

// DropScreenMeta carries the run context shown in the report header.
type DropScreenMeta struct {
	Interval      enum.Interval
	Months        int
	HoldoutMonths int
	TopN          int
	Shortlist     int
	Split         time.Time
	Gates         DropGates
	PFCap         float64
	Commission    float64 // per side, fraction of turnover; the stress adds this again per side
	Cash          float64 // mock starting cash; half-year results are shown as a percentage of it
	Scanned       int
	Skipped       int
}

// RenderDropScreenMarkdown renders the drop screening report; the last line is the shortlist.
func RenderDropScreenMarkdown(ranked, noSignals, tickRejected, rejected []DropRow, meta DropScreenMeta) string {
	var b strings.Builder
	grid, _ := DropGrid(meta.Interval)
	gridSize := len(grid)
	vols, _ := dropGridVols(meta.Interval)
	volStr := make([]string, 0, len(vols))
	for _, v := range vols {
		volStr = append(volStr, fmt.Sprintf("%g", v))
	}
	pfs := make([]float64, 0, len(ranked))
	for _, r := range ranked {
		pfs = append(pfs, r.PFMedStress)
	}
	d := distributionOf(pfs)

	b.WriteString("# drop_bounce screener\n\n")
	fmt.Fprintf(&b, "Таймфрейм: %s. Окно: %d мес., holdout: последние %d мес. (срез %s).\n",
		meta.Interval, meta.Months, meta.HoldoutMonths, meta.Split.Format("2006-01-02"))
	fmt.Fprintf(&b, "Сетка: %d конфигураций — VolDailyMult {%s} × RSIPeriod {4, 6} × RSIExit {55, 75} × GreenVolExit {0, 1.5}; "+
		"прибито: стоп 1.0 дневного ATR(10), VolAvgPeriod 10, StartHour 7.\n", gridSize, strings.Join(volStr, ", "))
	fmt.Fprintf(&b, "Гейты: оборот >= %.0f млн ₽/день, медиана сделок на train >= %.0f, два шага цены <= %.2f%% цены (0 — гейт выключен). PF зажат сверху на %.1f.\n",
		meta.Gates.MinTurnoverM, meta.Gates.MinTrades, meta.Gates.MaxTickPct, meta.PFCap)
	fmt.Fprintf(&b, "Стресс издержек (`PFmed×2`, ключ рейтинга): к комиссии %.4f за сторону добавлена ещё одна такая же и два шага цены на круг.\n",
		meta.Commission)
	fmt.Fprintf(&b, "Вселенная: scanned=%d ranked=%d no-signal=%d tick-rejected=%d rejected=%d skipped=%d.\n\n",
		meta.Scanned, len(ranked), len(noSignals), len(tickRejected), len(rejected), meta.Skipped)

	b.WriteString("## Распределение PFmed×2 по ранжированной вселенной\n\n")
	fmt.Fprintf(&b, "min %.2f · Q1 %.2f · медиана %.2f · Q3 %.2f · max %.2f · доля PFmed×2 >= 1.5: %.0f%% (n=%d)\n\n",
		d.Min, d.Q1, d.Median, d.Q3, d.Max, d.ShareAbove15*100, d.N)

	top := ranked
	if meta.TopN > 0 && meta.TopN < len(top) {
		top = top[:meta.TopN]
	}

	b.WriteString("## Рейтинг\n\n")
	b.WriteString("| # | Ticker | Name | PFmed×2 | PFmed | TradesMed | Plateau | Capped | SilentCfg | PFmed×2 HO | Trades HO | Halves+ | Оборот, млн | ATR% дн | Tick% |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range top {
		tick := fmt.Sprintf("%.2f", r.TickPct)
		if r.TickUnknown {
			tick = "?"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %.2f | %.2f | %.0f | %s | %d/%d | %d/%d | %.2f | %.0f | %s | %.0f | %.2f | %s |\n",
			i+1, r.Ticker, r.Name, r.PFMedStress, r.PFMed, r.TradesMed, pct0(r.Plateau),
			r.Capped, gridSize, r.SilentCfg, gridSize, r.PFMedStressHO, r.TradesMedHO, pct0(r.HalvesPos),
			r.TurnoverM, r.DailyATRPct, tick)
	}
	b.WriteString("\n`Tick%` `?` — API не отдал шаг цены: стресс учитывает только комиссию. ")
	b.WriteString("`PFmed×2 HO` в сортировке не участвует: это красный флаг («работало и развалилось»), а не критерий отбора.\n\n")

	b.WriteString("## Детали топа\n\n")
	b.WriteString("| Ticker | SL% | RSI% | GREENVOL% | HoldD | TopHalf | Лучшая конфигурация | Полугодия лучшей конфигурации |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, r := range top {
		topHalf := "—"
		if r.TopHalf > 0 {
			topHalf = pct0(r.TopHalf)
		}
		best := fmt.Sprintf("vol %g, RSI(%d)@%.0f, GV %g", r.Best.VolDailyMult, r.Best.RSIPeriod, r.Best.RSIExit, r.Best.GreenVolExit)
		halves := make([]string, 0, len(r.BestHalves))
		for _, h := range r.BestHalves {
			share := 0.0
			if meta.Cash > 0 {
				share = h.PnL / meta.Cash * 100
			}
			halves = append(halves, fmt.Sprintf("%s %+.1f%%", h.Label, share))
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %.1f | %s | %s | %s |\n",
			r.Ticker, pct0(r.SLShare), pct0(r.RSIShare), pct0(r.GreenVolShare), r.HoldDays, topHalf, best, strings.Join(halves, " · "))
	}
	b.WriteString("\n`SL%`/`RSI%`/`GREENVOL%` — доли выходов (остаток — закрытие в конце окна); `HoldD` — медианная длительность сделки, дней; ")
	b.WriteString("`TopHalf` — доля лучшего полугодия в сумме прибыльных (100% — весь профит из одного отрезка, «—» — прибыльных нет). ")
	b.WriteString("Полугодия лучшей конфигурации — train-сделки в процентах от стартового кэша.\n\n")

	if len(noSignals) > 0 {
		b.WriteString("## Нет сигналов\n\n")
		b.WriteString("Ни одной сделки ни в одной конфигурации ни в одном окне:\n\n")
		fmt.Fprintf(&b, "%s\n\n", strings.Join(DropShortlist(noSignals, 0), ", "))
	}

	if len(tickRejected) > 0 {
		b.WriteString("## Отсеяно гейтом шага цены\n\n")
		b.WriteString("| Ticker | Tick% | PFmed | PFmed×2 |\n|---|---|---|---|\n")
		for _, r := range tickRejected {
			fmt.Fprintf(&b, "| %s | %.2f | %.2f | %.2f |\n", r.Ticker, r.TickPct, r.PFMed, r.PFMedStress)
		}
		b.WriteString("\n")
	}

	if len(rejected) > 0 {
		b.WriteString("## Отсеяно гейтами\n\n")
		b.WriteString("| Ticker | Причина | PFmed×2 |\n|---|---|---|\n")
		for _, r := range rejected {
			fmt.Fprintf(&b, "| %s | %s | %.2f |\n", r.Ticker, DropRejectReason(r, meta.Gates), r.PFMedStress)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Как это читать\n\n")
	b.WriteString("Шортлист — это **очередь на калибровку** (скилл drop-bounce-calibrate), а не доказательство края. ")
	b.WriteString("Верх рейтинга по построению содержит везунчиков: испытаний столько же, сколько тикеров умножить на конфигурации. ")
	b.WriteString("Планка приёмки не меняется — pooled OOS walk-forward калибровки (PF >= 1.2, >= 20 сделок, >= 3 живых фолда).\n\n")
	b.WriteString("## Шортлист\n\n")
	fmt.Fprintf(&b, "%s=%s\n", DropShortlistKey, strings.Join(DropShortlist(ranked, meta.Shortlist), ","))
	return b.String()
}
