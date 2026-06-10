package backtest

import (
	"fmt"
	"strings"
	"time"

	"tinvest/internal/domain/backtest"
)

// BasketEntry is one ticker's out-of-sample result inside a basket walk-forward run.
type BasketEntry struct {
	Ticker         string
	Trades         int
	ProfitFactor   float64
	NetPnL         float64
	NetPnLPct      float64
	MaxDrawdownPct float64
	WinRate        float64              // fraction 0–1 (Wins/TotalTrades)
	Params         []backtest.ParamLine // winning calibrated params for this ticker
	Skipped        bool                 // true when the ticker produced no OOS result
	Note           string               // reason when skipped or no trades
}

// BasketSummary aggregates per-ticker OOS results plus the pooled-trade metrics.
type BasketSummary struct {
	Pooled  backtest.Metrics // metrics over the pooled OOS trades (trade-based fields only)
	Entries []BasketEntry
}

// PooledMetrics computes trade-based metrics over a flat list of trades drawn from
// multiple instruments. It reuses backtest.Compute with a synthetic Result carrying
// only trades; equity-based fields (MaxDrawdown, CAGR, NetPnL, Exposure) come out zero
// because a pool spanning separate capital bases has no single equity curve.
// Mechanically: FinalEquity==InitialCash==0 zeroes NetPnL/NetPnLPct, periodDays==0
// zeroes CAGR, totalBars==0 zeroes ExposurePct, and the empty Equity slice zeroes
// drawdown. If Compute ever derives those fields from trade PnL when no equity curve
// is present, the equity-zero assertions in basket_test.go will catch the regression.
func PooledMetrics(trades []backtest.Trade) backtest.Metrics {
	return backtest.Compute(backtest.Result{Trades: trades}, 0, 0, 0)
}

// RenderBasketMarkdown renders the pooled-OOS aggregate plus a per-ticker breakdown.
// from/to bound the out-of-sample window common to every ticker.
func RenderBasketMarkdown(metric string, s BasketSummary, from, to time.Time) string {
	var b strings.Builder
	m := s.Pooled
	b.WriteString("# Корзина momentum — walk-forward OOS\n\n")
	fmt.Fprintf(&b, "- OOS-период: %s — %s\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
	fmt.Fprintf(&b, "- Калибровка ранжировалась по: %s\n\n", metric)

	b.WriteString("## Пул сделок (агрегат OOS)\n\n")
	b.WriteString("| Метрика | Значение |\n|---|---|\n")
	fmt.Fprintf(&b, "| Всего сделок | %d |\n", m.TotalTrades)
	fmt.Fprintf(&b, "| Выигрышных / проигрышных | %d / %d |\n", m.Wins, m.Losses)
	fmt.Fprintf(&b, "| Win rate | %.2f%% |\n", m.WinRate*100)
	fmt.Fprintf(&b, "| Profit factor | %.3f |\n", m.ProfitFactor)
	fmt.Fprintf(&b, "| Gross profit / loss | %.2f / %.2f |\n", m.GrossProfit, m.GrossLoss)
	fmt.Fprintf(&b, "| Expectancy | %.2f |\n", m.Expectancy)
	fmt.Fprintf(&b, "| Sortino | %.3f |\n", m.Sortino)
	fmt.Fprintf(&b, "| Лучшая / худшая сделка | %.2f / %.2f |\n\n", m.BestTrade, m.WorstTrade)

	b.WriteString("## Разбивка по бумагам (OOS)\n\n")
	b.WriteString("| Тикер | Сделок | PF | Net PnL % | Max DD % | Win rate | Параметры-победителя |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for _, e := range s.Entries {
		if e.Skipped {
			fmt.Fprintf(&b, "| %s | — | — | — | — | — | %s |\n", e.Ticker, e.Note)
			continue
		}
		note := paramSummary(e.Params)
		if e.Note != "" {
			note = e.Note
		}
		fmt.Fprintf(&b, "| %s | %d | %.3f | %.2f%% | %.2f%% | %.2f%% | %s |\n",
			e.Ticker, e.Trades, e.ProfitFactor, e.NetPnLPct*100, e.MaxDrawdownPct*100, e.WinRate*100, note)
	}
	return b.String()
}

// paramSummary renders the handful of params that matter most for a quick scan.
func paramSummary(rows []backtest.ParamLine) string {
	keys := []string{"EMAPeriod", "SLMult", "TakeProfitRR", "CooldownBars", "DailyTrendPeriod"}
	idx := make(map[string]string, len(rows))
	for _, r := range rows {
		idx[r.Name] = r.Value
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := idx[k]; ok {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return strings.Join(parts, ", ")
}
