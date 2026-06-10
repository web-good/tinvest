package backtest

import (
	"testing"

	"tinvest/internal/domain/backtest"
)

func TestPooledMetrics(t *testing.T) {
	trades := []backtest.Trade{
		{PnL: 100}, {PnL: 50}, {PnL: -40}, {PnL: -10},
	}
	m := PooledMetrics(trades)
	if m.TotalTrades != 4 {
		t.Fatalf("TotalTrades=%d want 4", m.TotalTrades)
	}
	if m.Wins != 2 || m.Losses != 2 {
		t.Fatalf("Wins/Losses=%d/%d want 2/2", m.Wins, m.Losses)
	}
	if m.GrossProfit != 150 || m.GrossLoss != 50 {
		t.Fatalf("Gross profit/loss=%.0f/%.0f want 150/50", m.GrossProfit, m.GrossLoss)
	}
	if m.ProfitFactor != 3.0 {
		t.Fatalf("ProfitFactor=%.3f want 3.0", m.ProfitFactor)
	}
	if m.Expectancy != 25 {
		t.Fatalf("Expectancy=%.3f want 25", m.Expectancy)
	}
	if m.BestTrade != 100 || m.WorstTrade != -40 {
		t.Fatalf("Best/Worst=%.0f/%.0f want 100/-40", m.BestTrade, m.WorstTrade)
	}
	if m.WinRate != 0.5 || m.LossRate != 0.5 {
		t.Fatalf("WinRate/LossRate=%.3f/%.3f want 0.5/0.5", m.WinRate, m.LossRate)
	}
	if m.AvgWin != 75 || m.AvgLoss != 25 {
		t.Fatalf("AvgWin/AvgLoss=%.3f/%.3f want 75/25", m.AvgWin, m.AvgLoss)
	}
	// Equity-based fields have no meaning across a pool and must stay zero — this guards
	// the PooledMetrics contract against future changes to Compute (see basket.go).
	if m.MaxDrawdown != 0 || m.MaxDrawdownPct != 0 || m.CAGR != 0 || m.NetPnL != 0 || m.NetPnLPct != 0 || m.ExposurePct != 0 {
		t.Fatalf("equity fields must be zero: DD=%.3f DD%%=%.3f CAGR=%.3f Net=%.3f Net%%=%.3f Exp=%.3f",
			m.MaxDrawdown, m.MaxDrawdownPct, m.CAGR, m.NetPnL, m.NetPnLPct, m.ExposurePct)
	}
}

func TestPooledMetricsEmpty(t *testing.T) {
	m := PooledMetrics(nil)
	if m.TotalTrades != 0 || m.ProfitFactor != 0 || m.Expectancy != 0 {
		t.Fatalf("empty pool must be zero metrics: %+v", m)
	}
}

func TestPooledMetricsAllWins(t *testing.T) {
	m := PooledMetrics([]backtest.Trade{{PnL: 10}, {PnL: 20}})
	if m.ProfitFactor != 30 {
		t.Fatalf("ProfitFactor=%.3f want 30 (GrossProfit when no losses)", m.ProfitFactor)
	}
}
