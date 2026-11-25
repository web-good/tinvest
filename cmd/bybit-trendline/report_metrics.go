package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"tinvest/internal/domain/backtest"
)

type inPlayPeriod struct {
	start, end time.Time
}

// from and to delimit a processed batch, which may split a half-hour window.
func appendPeriods(periods []inPlayPeriod, f *inPlayFilter, from, to time.Time) []inPlayPeriod {
	for at := from; at.Before(to); {
		next := at.Truncate(checkInterval).Add(checkInterval)
		if next.After(to) {
			next = to
		}
		if f.allows(at) {
			if n := len(periods); n > 0 && periods[n-1].end.Equal(at) {
				periods[n-1].end = next
			} else {
				periods = append(periods, inPlayPeriod{at, next})
			}
		}
		at = next
	}
	return periods
}

func writePeriods(b *strings.Builder, periods []inPlayPeriod, span time.Duration) {
	b.WriteString("## Периоды inplay\n\n")
	if len(periods) == 0 {
		b.WriteString("За период теста монета не входила в inplay.\n\n")
		return
	}
	var total time.Duration
	for _, p := range periods {
		total += p.end.Sub(p.start)
	}
	fmt.Fprintf(b, "Интервалов %d, суммарно %.1f ч (%.2f%% времени теста). Конец каждого интервала исключён.\n\n", len(periods), total.Hours(), 100*float64(total)/float64(span))
	b.WriteString("| Начало МСК | Конец МСК | Длительность |\n|---|---|---:|\n")
	for _, p := range periods {
		fmt.Fprintf(b, "| %s | %s | %d мин |\n", reportTime(p.start), reportTime(p.end), int(p.end.Sub(p.start).Minutes()))
	}
	b.WriteByte('\n')
}

func ratio(a, b float64) string {
	if b == 0 {
		if a > 0 {
			return "∞"
		}
		return "—"
	}
	return fmt.Sprintf("%.3f", a/b)
}

// Daily returns use realized equity at UTC day end, including flat days.
// An exit stamped at midnight belongs to the candle ending at midnight.
func dailyRatios(start, end time.Time, trades []trade) (string, string) {
	days := int(end.Sub(start) / (24 * time.Hour))
	if days < 2 {
		return "—", "—"
	}
	growth := make([]float64, days)
	for i := range growth {
		growth[i] = 1
	}
	for _, t := range trades {
		day := int(t.closed.Add(-time.Nanosecond).Sub(start) / (24 * time.Hour))
		growth[day] *= 1 + t.net
	}
	var mean, downside, variance float64
	for _, g := range growth {
		r := g - 1
		mean += r
		if r < 0 {
			downside += r * r
		}
	}
	mean /= float64(days)
	for _, g := range growth {
		deviation := g - 1 - mean
		variance += deviation * deviation
	}
	sharpe, sortino := "—", "—"
	if variance > 0 {
		sharpe = ratio(mean*math.Sqrt(365), math.Sqrt(variance/float64(days-1)))
	}
	if downside > 0 {
		sortino = ratio(mean*math.Sqrt(365), math.Sqrt(downside/float64(days)))
	}
	return sharpe, sortino
}

func writeMetrics(b *strings.Builder, start, end time.Time, e *engine) {
	const initial = 10000.
	result := backtest.Result{InitialCash: initial, FinalEquity: initial}
	peak, maxDD := initial, 0.
	var held time.Duration
	var wins, flat, winRun, lossRun, maxWins, maxLosses int
	for _, t := range e.done {
		pnl := result.FinalEquity * t.net
		result.FinalEquity += pnl
		result.Trades = append(result.Trades, backtest.Trade{PnL: pnl})
		held += t.closed.Sub(t.opened)
		peak = max(peak, result.FinalEquity)
		maxDD = max(maxDD, 1-result.FinalEquity/peak)
		switch {
		case pnl > 0:
			wins++
			winRun++
			lossRun = 0
		case pnl < 0:
			lossRun++
			winRun = 0
		default:
			flat++
			winRun, lossRun = 0, 0
		}
		maxWins, maxLosses = max(maxWins, winRun), max(maxLosses, lossRun)
	}
	span := end.Sub(start)
	m := backtest.Compute(result, 0, 0, span.Hours()/24)
	b.WriteString("## Метрики стратегии\n\nУсловный стартовый капитал 10 000 единиц валюты котировки. Каждый вход использует весь доступный капитал. PnL, просадка и коэффициенты рассчитаны по закрытым сделкам с учётом комиссии и проскальзывания.\n\n| Метрика | Значение |\n|---|---:|\n")
	row := func(label, format string, args ...any) {
		fmt.Fprintf(b, "| %s | %s |\n", label, fmt.Sprintf(format, args...))
	}
	row("Закрытых сделок (прибыль / убыток / ноль)", "%d (%d / %d / %d)", m.TotalTrades, wins, m.Losses, flat)
	if m.TotalTrades > 0 {
		row("Win rate", "%.2f%%", 100*float64(wins)/float64(m.TotalTrades))
	} else {
		row("Win rate", "—")
	}
	row("Чистый PnL", "%+.2f (%+.2f%%)", m.NetPnL, 100*m.NetPnLPct)
	row("Итоговый капитал", "%.2f", result.FinalEquity)
	row("Сумма прибылей / сумма убытков", "%.2f / %.2f", m.GrossProfit, -m.GrossLoss)
	row("Profit factor", "%s", ratio(m.GrossProfit, m.GrossLoss))
	if m.TotalTrades > 0 {
		row("Матожидание PnL на сделку", "%+.2f", m.Expectancy)
		row("Лучшая / худшая сделка (PnL)", "%+.2f / %+.2f", m.BestTrade, m.WorstTrade)
		row("Среднее удержание закрытой позиции", "%s", (held / time.Duration(m.TotalTrades)).Round(time.Second))
	}
	if wins > 0 {
		row("Средняя прибыль", "%.2f", m.GrossProfit/float64(wins))
	}
	if m.Losses > 0 {
		row("Средний убыток", "%.2f", -m.AvgLoss)
	}
	if wins > 0 && m.Losses > 0 {
		row("Payoff ratio", "%s", ratio(m.GrossProfit/float64(wins), m.AvgLoss))
	}
	row("Макс. серия прибылей / убытков", "%d / %d", maxWins, maxLosses)
	row("Макс. просадка по закрытиям", "%.2f%%", maxDD*100)
	row("CAGR (годовой эквивалент)", "%+.2f%%", m.CAGR*100)
	sharpe, sortino := dailyRatios(start, end, e.done)
	row("Sharpe / Sortino", "%s / %s", sharpe, sortino)
	if maxDD > 0 {
		row("Calmar (CAGR / макс. просадка)", "%s", ratio(m.CAGR, maxDD))
	} else {
		row("Calmar (CAGR / макс. просадка)", "—")
	}
	if e.position != nil {
		held += end.Sub(e.position.opened)
	}
	row("Время в позиции", "%.2f%%", 100*float64(held)/float64(span))
	b.WriteString("\nSharpe и Sortino используют дневную реализованную доходность, включая дни без закрытий, нулевую ставку и 365 дней в году. Прочерк означает, что данных для коэффициента недостаточно или его знаменатель равен нулю.\n\n")
}
