package backtest

import "math"

// Compute derives metrics from a Result. barsInMarket/totalBars/periodDays are
// supplied by the caller (typically r.BarsInMarket, len(r.Equity), and the
// candle span in days) so Compute stays pure.
func Compute(r Result, barsInMarket, totalBars int, periodDays float64) Metrics {
	var m Metrics
	m.TotalTrades = len(r.Trades)
	for i, t := range r.Trades {
		if i == 0 {
			m.BestTrade, m.WorstTrade = t.PnL, t.PnL
		} else {
			if t.PnL > m.BestTrade {
				m.BestTrade = t.PnL
			}
			if t.PnL < m.WorstTrade {
				m.WorstTrade = t.PnL
			}
		}
		if t.PnL >= 0 {
			m.Wins++
			m.GrossProfit += t.PnL
		} else {
			m.Losses++
			m.GrossLoss += -t.PnL
		}
	}
	if m.TotalTrades > 0 {
		m.WinRate = float64(m.Wins) / float64(m.TotalTrades)
		m.LossRate = float64(m.Losses) / float64(m.TotalTrades)
		m.Expectancy = (m.GrossProfit - m.GrossLoss) / float64(m.TotalTrades)
		m.Sortino = sortino(r.Trades, m.Expectancy)
	}
	if m.Wins > 0 {
		m.AvgWin = m.GrossProfit / float64(m.Wins)
	}
	if m.Losses > 0 {
		m.AvgLoss = m.GrossLoss / float64(m.Losses)
	}
	switch {
	case m.GrossLoss > 0:
		m.ProfitFactor = m.GrossProfit / m.GrossLoss
	case m.GrossProfit > 0:
		m.ProfitFactor = m.GrossProfit
	default:
		m.ProfitFactor = 0
	}
	m.NetPnL = r.FinalEquity - r.InitialCash
	if r.InitialCash > 0 {
		m.NetPnLPct = m.NetPnL / r.InitialCash
	}
	m.MaxDrawdown, m.MaxDrawdownPct = maxDrawdown(r.Equity)
	if totalBars > 0 {
		m.ExposurePct = float64(barsInMarket) / float64(totalBars)
	}
	if periodDays > 0 && r.InitialCash > 0 && r.FinalEquity > 0 {
		m.CAGR = math.Pow(r.FinalEquity/r.InitialCash, 365.0/periodDays) - 1
	}
	return m
}

// maxDrawdown returns the largest peak-to-trough drop on the equity curve, both
// absolute and as a fraction of the running peak.
func maxDrawdown(eq []EquityPoint) (abs, pct float64) {
	var peak float64
	for i, p := range eq {
		if i == 0 || p.Equity > peak {
			peak = p.Equity
		}
		dd := peak - p.Equity
		if dd > abs {
			abs = dd
			if peak > 0 {
				pct = dd / peak
			}
		}
	}
	return abs, pct
}

// sortino returns mean trade PnL divided by the downside deviation of trade PnL
// (the root-mean-square of the negative PnLs). With no losing trades the downside
// deviation is zero; a positive mean then returns the mean itself (mirrors the
// ProfitFactor convention), otherwise 0. The squared losses are divided by n
// (total trades), not by the loser count, so the metric also penalises a scarcity
// of winners.
func sortino(trades []Trade, mean float64) float64 {
	n := len(trades)
	if n == 0 {
		return 0
	}
	var sqSum float64
	for _, t := range trades {
		if t.PnL < 0 {
			sqSum += t.PnL * t.PnL
		}
	}
	dd := math.Sqrt(sqSum / float64(n))
	switch {
	case dd > 0:
		return mean / dd
	case mean > 0:
		return mean
	default:
		return 0
	}
}
