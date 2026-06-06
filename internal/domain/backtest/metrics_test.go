package backtest

import (
	"math"
	"testing"
	"time"
)

func eqCurve(vals []float64) []EquityPoint {
	out := make([]EquityPoint, len(vals))
	base := time.Unix(0, 0)
	for i, v := range vals {
		out[i] = EquityPoint{Time: base.Add(time.Duration(i) * time.Hour), Equity: v}
	}
	return out
}

func TestComputeBasicCounts(t *testing.T) {
	r := Result{
		Trades: []Trade{
			{PnL: 100}, {PnL: -40}, {PnL: 60}, {PnL: -20},
		},
		Equity:      eqCurve([]float64{1000, 1100, 1060, 1120, 1100}),
		InitialCash: 1000,
		FinalEquity: 1100,
	}
	m := Compute(r, 3, 5, 0)
	if m.TotalTrades != 4 || m.Wins != 2 || m.Losses != 2 {
		t.Fatalf("counts: total=%d wins=%d losses=%d", m.TotalTrades, m.Wins, m.Losses)
	}
	if !approx(m.WinRate, 0.5) || !approx(m.LossRate, 0.5) {
		t.Fatalf("rates: win=%f loss=%f", m.WinRate, m.LossRate)
	}
	if !approx(m.GrossProfit, 160) || !approx(m.GrossLoss, 60) {
		t.Fatalf("gross: profit=%f loss=%f", m.GrossProfit, m.GrossLoss)
	}
	if !approx(m.ProfitFactor, 160.0/60.0) {
		t.Fatalf("profit factor = %f", m.ProfitFactor)
	}
	if !approx(m.NetPnL, 100) || !approx(m.NetPnLPct, 0.1) {
		t.Fatalf("net: pnl=%f pct=%f", m.NetPnL, m.NetPnLPct)
	}
	if !approx(m.Expectancy, 25) { // (160-60)/4
		t.Fatalf("expectancy = %f, want 25", m.Expectancy)
	}
	if !approx(m.AvgWin, 80) || !approx(m.AvgLoss, 30) {
		t.Fatalf("avg: win=%f loss=%f", m.AvgWin, m.AvgLoss)
	}
	if !approx(m.BestTrade, 100) || !approx(m.WorstTrade, -40) {
		t.Fatalf("best/worst: %f / %f", m.BestTrade, m.WorstTrade)
	}
	if !approx(m.ExposurePct, 0.6) { // 3/5
		t.Fatalf("exposure = %f, want 0.6", m.ExposurePct)
	}
}

func TestComputeProfitFactorNoLosses(t *testing.T) {
	r := Result{Trades: []Trade{{PnL: 50}, {PnL: 30}}, InitialCash: 1000, FinalEquity: 1080}
	m := Compute(r, 0, 0, 0)
	if !approx(m.ProfitFactor, 80) { // GrossLoss==0, GrossProfit>0 -> GrossProfit
		t.Fatalf("profit factor = %f, want 80", m.ProfitFactor)
	}
}

func TestComputeMaxDrawdown(t *testing.T) {
	// peak 1200 at idx 2, trough 900 at idx 4 -> DD 300, pct 0.25.
	r := Result{Equity: eqCurve([]float64{1000, 1100, 1200, 1000, 900, 1050}), InitialCash: 1000, FinalEquity: 1050}
	m := Compute(r, 0, 6, 0)
	if !approx(m.MaxDrawdown, 300) || !approx(m.MaxDrawdownPct, 0.25) {
		t.Fatalf("drawdown: abs=%f pct=%f, want 300 / 0.25", m.MaxDrawdown, m.MaxDrawdownPct)
	}
}

func TestComputeCAGR(t *testing.T) {
	// double in 365 days -> CAGR = 1.0.
	r := Result{InitialCash: 1000, FinalEquity: 2000}
	m := Compute(r, 0, 0, 365)
	if !approx(m.CAGR, 1.0) {
		t.Fatalf("CAGR = %f, want 1.0", m.CAGR)
	}
	// periodDays 0 -> CAGR 0 (no divide).
	if c := Compute(r, 0, 0, 0).CAGR; c != 0 {
		t.Fatalf("CAGR with 0 days = %f, want 0", c)
	}
	_ = math.Pi
}

func TestComputeSortino(t *testing.T) {
	// Trades 100,-40,60,-20: mean=25; downside sq mean = (40^2+20^2)/4 = 500;
	// downside dev = sqrt(500) = 22.360679...; Sortino = 25 / 22.360679 = 1.118034.
	r := Result{
		Trades:      []Trade{{PnL: 100}, {PnL: -40}, {PnL: 60}, {PnL: -20}},
		InitialCash: 1000, FinalEquity: 1100,
	}
	m := Compute(r, 0, 0, 0)
	if !approx(m.Sortino, 25.0/math.Sqrt(500)) {
		t.Fatalf("Sortino = %f, want %f", m.Sortino, 25.0/math.Sqrt(500))
	}
}

func TestComputeSortinoNoDownside(t *testing.T) {
	// All winners: downside dev 0, mean > 0 -> Sortino = mean (mirrors PF convention).
	r := Result{Trades: []Trade{{PnL: 50}, {PnL: 30}}, InitialCash: 1000, FinalEquity: 1080}
	m := Compute(r, 0, 0, 0)
	if !approx(m.Sortino, 40) { // mean = (50+30)/2
		t.Fatalf("Sortino = %f, want 40", m.Sortino)
	}
}
