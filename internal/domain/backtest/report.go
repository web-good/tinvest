package backtest

import (
	"fmt"
	"strings"
	"time"
)

const tsLayout = "2006-01-02 15:04"

// RenderMarkdown renders the full single-run report as a Markdown string.
func RenderMarkdown(meta Meta, m Metrics, trades []Trade, equity []EquityPoint) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Бэктест %s (%s)\n\n", meta.Ticker, meta.Interval)
	fmt.Fprintf(&b, "- Период: %s — %s\n", meta.From.Format(tsLayout), meta.To.Format(tsLayout))
	fmt.Fprintf(&b, "- Стартовый кэш: %.2f\n", meta.InitialCash)
	fmt.Fprintf(&b, "- Fraction: %.4g; Commission: %.4g\n", meta.Fraction, meta.Commission)
	if meta.OpenPosition {
		b.WriteString("- ⚠️ На конце прогона осталась открытая позиция (оценена mark-to-market)\n")
	}
	b.WriteString("\n## Параметры стратегии\n\n| Параметр | Значение |\n|---|---|\n")
	for _, p := range meta.Params {
		fmt.Fprintf(&b, "| %s | %s |\n", p.Name, p.Value)
	}

	b.WriteString("\n## Сводка метрик\n\n| Метрика | Значение |\n|---|---|\n")
	fmt.Fprintf(&b, "| Всего сделок | %d |\n", m.TotalTrades)
	fmt.Fprintf(&b, "| Выигрышных / проигрышных | %d / %d |\n", m.Wins, m.Losses)
	fmt.Fprintf(&b, "| Win rate | %.2f%% |\n", m.WinRate*100)
	fmt.Fprintf(&b, "| Gross profit / loss | %.2f / %.2f |\n", m.GrossProfit, m.GrossLoss)
	fmt.Fprintf(&b, "| Profit factor | %.3f |\n", m.ProfitFactor)
	fmt.Fprintf(&b, "| Чистый PnL | %.2f (%.2f%%) |\n", m.NetPnL, m.NetPnLPct*100)
	fmt.Fprintf(&b, "| Стартовый / финальный капитал | %.2f / %.2f |\n", meta.InitialCash, meta.InitialCash+m.NetPnL)
	fmt.Fprintf(&b, "| Макс. просадка | %.2f (%.2f%%) |\n", m.MaxDrawdown, m.MaxDrawdownPct*100)
	fmt.Fprintf(&b, "| Средняя прибыль / убыток | %.2f / %.2f |\n", m.AvgWin, m.AvgLoss)
	fmt.Fprintf(&b, "| Expectancy | %.2f |\n", m.Expectancy)
	fmt.Fprintf(&b, "| Лучшая / худшая сделка | %.2f / %.2f |\n", m.BestTrade, m.WorstTrade)
	fmt.Fprintf(&b, "| Exposure | %.2f%% |\n", m.ExposurePct*100)
	fmt.Fprintf(&b, "| CAGR | %.2f%% |\n", m.CAGR*100)

	b.WriteString("\n## Журнал сделок\n\n| № | Вход | Цена входа | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, t := range trades {
		fmt.Fprintf(&b, "| %d | %s | %.4f | %s | %.4f | %s | %d | %.2f | %.2f%% | %.4f | %.4f | %.4f | %s |\n",
			i+1, t.EntryTime.Format(tsLayout), t.EntryPrice, t.ExitTime.Format(tsLayout),
			t.ExitPrice, t.Reason, t.BarsHeld, t.PnL, t.PnLPct*100,
			t.SupportLevel, t.ResistanceLevel, t.ATR, t.EntryReason)
	}

	b.WriteString("\n## Движение капитала\n\n")
	if len(equity) == 0 {
		b.WriteString("Нет данных equity.\n")
	} else {
		minEq, maxEq := equity[0].Equity, equity[0].Equity
		for _, p := range equity {
			if p.Equity < minEq {
				minEq = p.Equity
			}
			if p.Equity > maxEq {
				maxEq = p.Equity
			}
		}
		fmt.Fprintf(&b, "- Старт: %.2f\n- Мин: %.2f\n- Макс: %.2f\n- Финал: %.2f\n",
			equity[0].Equity, minEq, maxEq, equity[len(equity)-1].Equity)
		b.WriteString("\nПолная кривая — в `*_equity.csv`.\n")
	}
	return b.String()
}

// csvField renders s as an RFC 4180 CSV field: values containing a comma, double
// quote, or newline are wrapped in double quotes with inner quotes doubled.
// Plain values pass through unquoted.
func csvField(s string) string {
	if !strings.ContainsAny(s, ",\"\n\r") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// RenderTradesCSV renders the trade journal as CSV.
func RenderTradesCSV(trades []Trade) string {
	var b strings.Builder
	b.WriteString("idx,entry_time,entry_price,exit_time,exit_price,qty,reason,pnl,pnl_pct,bars_held,support_level,resistance_level,atr,entry_reason\n")
	for i, t := range trades {
		fmt.Fprintf(&b, "%d,%s,%.6f,%s,%.6f,%d,%s,%.6f,%.6f,%d,%.6f,%.6f,%.6f,%s\n",
			i+1, t.EntryTime.UTC().Format(time.RFC3339), t.EntryPrice,
			t.ExitTime.UTC().Format(time.RFC3339), t.ExitPrice, t.Quantity,
			t.Reason, t.PnL, t.PnLPct, t.BarsHeld,
			t.SupportLevel, t.ResistanceLevel, t.ATR, csvField(t.EntryReason))
	}
	return b.String()
}

// RenderEquityCSV renders the equity curve as CSV.
func RenderEquityCSV(equity []EquityPoint) string {
	var b strings.Builder
	b.WriteString("time,equity\n")
	for _, p := range equity {
		fmt.Fprintf(&b, "%s,%.6f\n", p.Time.UTC().Format(time.RFC3339), p.Equity)
	}
	return b.String()
}
