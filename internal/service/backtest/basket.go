package backtest

import (
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
