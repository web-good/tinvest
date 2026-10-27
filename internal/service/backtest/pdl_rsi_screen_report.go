package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"tinvest/internal/enum"
)

// PDLRSIShortlistKey names the shortlist line the report and the console print: the calibration
// queue for the pdl-rsi-calibrate skill.
const PDLRSIShortlistKey = "PDL_RSI_CANDIDATES"

// PDLRSIGates are the pdl_rsi screener's hard gates; same meaning as DropGates.
type PDLRSIGates = DropGates

// PDLRSIRejectReason names every turnover/trades gate the row fails, "" when it passes both.
func PDLRSIRejectReason(r PDLRSIRow, g PDLRSIGates) string {
	return DropRejectReason(DropRow{Pregated: r.Pregated, TurnoverM: r.TurnoverM, TradesMed: r.TradesMed}, g)
}

// FilterAndRankPDLRSI buckets and ranks rows exactly as FilterAndRankDrop does.
func FilterAndRankPDLRSI(rows []PDLRSIRow, g PDLRSIGates) (ranked, noSignals, tickRejected, rejected []PDLRSIRow) {
	for _, r := range rows {
		switch {
		case r.Pregated:
			rejected = append(rejected, r)
		case r.NoSignals:
			noSignals = append(noSignals, r)
		case g.MaxTickPct > 0 && !r.TickUnknown && r.TickPct > g.MaxTickPct:
			tickRejected = append(tickRejected, r)
		case PDLRSIRejectReason(r, g) != "":
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
	byTicker := func(rs []PDLRSIRow) {
		sort.SliceStable(rs, func(i, j int) bool { return rs[i].Ticker < rs[j].Ticker })
	}
	byTicker(noSignals)
	byTicker(tickRejected)
	byTicker(rejected)
	return ranked, noSignals, tickRejected, rejected
}

// PDLRSIShortlist returns the first n tickers of the ranking; n <= 0 or beyond the ranking returns
// all of it.
func PDLRSIShortlist(ranked []PDLRSIRow, n int) []string {
	if n <= 0 || n > len(ranked) {
		n = len(ranked)
	}
	out := make([]string, 0, n)
	for _, r := range ranked[:n] {
		out = append(out, r.Ticker)
	}
	return out
}

// PDLRSIScreenMeta carries the run context shown in the report header.
type PDLRSIScreenMeta struct {
	Interval      enum.Interval
	Months        int
	HoldoutMonths int
	TopN          int
	Shortlist     int
	Split         time.Time
	Gates         PDLRSIGates
	PFCap         float64
	Commission    float64 // per side, fraction of turnover; the stress adds this again per side
	Cash          float64 // mock starting cash; half-year results are shown as a percentage of it
	Scanned       int
	Skipped       int
}

// RenderPDLRSIScreenMarkdown renders the pdl_rsi screening report; the last line is the shortlist.
func RenderPDLRSIScreenMarkdown(ranked, noSignals, tickRejected, rejected []PDLRSIRow, meta PDLRSIScreenMeta) string {
	var b strings.Builder
	grid, _ := PDLRSIGrid(meta.Interval)
	gridSize := len(grid)
	pfs := make([]float64, 0, len(ranked))
	for _, r := range ranked {
		pfs = append(pfs, r.PFMedStress)
	}
	d := distributionOf(pfs)

	b.WriteString("# pdl_rsi screener\n\n")
	fmt.Fprintf(&b, "Таймфрейм: %s. Окно: %d мес., holdout: последние %d мес. (срез %s).\n",
		meta.Interval, meta.Months, meta.HoldoutMonths, meta.Split.Format("2006-01-02"))
	fmt.Fprintf(&b, "Сетка: %d конфигураций — режим входа {без стохастика, %%K(14,1) < 20} × HourRSIPeriod {4, 6} × StopATR {0.5, 1.0}; "+
		"прибито: RSI 30 (без стохастика — крест вниз, со стохастиком — RSI ниже 30 на том же часе), RSIFloor 22, EMA(200) часа, цель 2R, выход RSI 70, дневной ATR(14), без потолка стопа, фильтры объёма и запаса хода выключены.\n",
		gridSize)
	fmt.Fprintf(&b, "Гейты: оборот >= %.0f млн ₽/день, медиана сделок на train >= %.0f, два шага цены <= %.2f%% цены (0 — гейт выключен). PF зажат сверху на %.1f.\n",
		meta.Gates.MinTurnoverM, meta.Gates.MinTrades, meta.Gates.MaxTickPct, meta.PFCap)
	fmt.Fprintf(&b, "Стресс издержек (`PFmed×2`, ключ рейтинга): к комиссии %.4f за сторону добавлена ещё одна такая же и два шага цены на круг.\n",
		meta.Commission)
	var pregated int
	for _, r := range rejected {
		if r.Pregated {
			pregated++
		}
	}
	fmt.Fprintf(&b, "Вселенная: scanned=%d ranked=%d no-signal=%d tick-rejected=%d rejected=%d pregated=%d skipped=%d.\n\n",
		meta.Scanned, len(ranked), len(noSignals), len(tickRejected), len(rejected), pregated, meta.Skipped)

	b.WriteString("## Распределение PFmed×2 по ранжированной вселенной\n\n")
	fmt.Fprintf(&b, "min %.2f · Q1 %.2f · медиана %.2f · Q3 %.2f · max %.2f · доля PFmed×2 >= 1.5: %.0f%% (n=%d)\n\n",
		d.Min, d.Q1, d.Median, d.Q3, d.Max, d.ShareAbove15*100, d.N)

	top := ranked
	if meta.TopN > 0 && meta.TopN < len(top) {
		top = top[:meta.TopN]
	}

	b.WriteString("## Рейтинг\n\n")
	b.WriteString("| # | Ticker | Name | PFmed×2 | PFmed | TradesMed | Без стох. | Уровень | Plateau | Capped | SilentCfg | PFmed×2 HO | Trades HO | Halves+ | Оборот, млн | ATR% дн | Tick% |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range top {
		tick := fmt.Sprintf("%.2f", r.TickPct)
		if r.TickUnknown {
			tick = "?"
		}
		mode := func(m PDLRSIStochMode) string {
			return fmt.Sprintf("%.2f/%.0f", r.Modes[m].PFMedStress, r.Modes[m].TradesMed)
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %.2f | %.2f | %.0f | %s | %s | %s | %d/%d | %d/%d | %.2f | %.0f | %s | %.0f | %.2f | %s |\n",
			i+1, r.Ticker, r.Name, r.PFMedStress, r.PFMed, r.TradesMed,
			mode(PDLRSIStochOff), mode(PDLRSIStochLevel), pct0(r.Plateau),
			r.Capped, gridSize, r.SilentCfg, gridSize, r.PFMedStressHO, r.TradesMedHO, pct0(r.HalvesPos),
			r.TurnoverM, r.DailyATRPct, tick)
	}
	b.WriteString("\nСтолбцы режимов — PFmed×2/медиана сделок на train по четырём конфигурациям режима. ")
	b.WriteString("`Tick%` `?` — API не отдал шаг цены: стресс учитывает только комиссию. ")
	b.WriteString("`PFmed×2 HO` в сортировке не участвует: это красный флаг («работало и развалилось»), а не критерий отбора.\n\n")

	b.WriteString("## Детали топа\n\n")
	b.WriteString("| Ticker | SL% | TP% | RSI% | HoldD | TopHalf | Лучшая конфигурация | Полугодия лучшей конфигурации |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, r := range top {
		topHalf := "—"
		if r.TopHalf > 0 {
			topHalf = pct0(r.TopHalf)
		}
		best := fmt.Sprintf("%s, RSI %d, стоп %g", PDLRSIModeOf(r.Best), r.Best.HourRSIPeriod, r.Best.StopATR)
		halves := make([]string, 0, len(r.BestHalves))
		for _, h := range r.BestHalves {
			share := 0.0
			if meta.Cash > 0 {
				share = h.PnL / meta.Cash * 100
			}
			halves = append(halves, fmt.Sprintf("%s %+.1f%%", h.Label, share))
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %.1f | %s | %s | %s |\n",
			r.Ticker, pct0(r.SLShare), pct0(r.TPShare), pct0(r.RSIShare), r.HoldDays, topHalf, best, strings.Join(halves, " · "))
	}
	b.WriteString("\n`SL%`/`TP%`/`RSI%` — доли выходов: стоп, цель и крест RSI 70 вверх (остаток — закрытие в конце окна); `HoldD` — медианная длительность сделки, дней; ")
	b.WriteString("`TopHalf` — доля лучшего полугодия в сумме прибыльных (100% — весь профит из одного отрезка, «—» — прибыльных нет). ")
	b.WriteString("Полугодия лучшей конфигурации — train-сделки в процентах от стартового кэша.\n\n")

	if len(noSignals) > 0 {
		b.WriteString("## Нет сигналов\n\n")
		b.WriteString("Ни одной сделки ни в одной конфигурации ни в одном окне:\n\n")
		fmt.Fprintf(&b, "%s\n\n", strings.Join(PDLRSIShortlist(noSignals, 0), ", "))
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
			pf := fmt.Sprintf("%.2f", r.PFMedStress)
			if r.Pregated {
				pf = "—"
			}
			fmt.Fprintf(&b, "| %s | %s | %s |\n", r.Ticker, PDLRSIRejectReason(r, meta.Gates), pf)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Как это читать\n\n")
	b.WriteString("Шортлист — это **очередь на калибровку** (скилл pdl-rsi-calibrate), а не доказательство края. ")
	b.WriteString("Верх рейтинга по построению содержит везунчиков: испытаний столько же, сколько тикеров умножить на конфигурации. ")
	b.WriteString("Судья не меняется — pooled OOS скользящего walk-forward калибровки.\n\n")
	b.WriteString("## Шортлист\n\n")
	fmt.Fprintf(&b, "%s=%s\n", PDLRSIShortlistKey, strings.Join(PDLRSIShortlist(ranked, meta.Shortlist), ","))
	return b.String()
}
