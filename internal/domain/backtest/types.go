// Package backtest provides a pure, I/O-free replay engine, mock portfolio,
// metrics and report rendering for backtesting per-share scalping strategies.
package backtest

import "time"

// Candle is one OHLCV bar (series are oldest-first).
type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// Config controls the mock portfolio and fills.
type Config struct {
	InitialCash float64 // starting mock cash
	Fraction    float64 // fraction of current cash deployed per Buy (1.0 = all-in)
	Commission  float64 // commission as a fraction of turnover (e.g. 0.0005)
	Lot         int32   // share lot size (orders are whole lots)
}

// Trade is one completed round-trip (entry -> exit).
type Trade struct {
	EntryTime       time.Time
	EntryPrice      float64
	ExitTime        time.Time
	ExitPrice       float64
	Quantity        int64   // shares (lots * Lot)
	Reason          string  // exit reason: "SL" / "TRAIL" / "TP"
	PnL             float64 // net of commission, in currency
	PnLPct          float64 // PnL relative to entry cost
	BarsHeld        int
	SupportLevel    float64 // HVN support the entry bounced off; 0 when n/a
	ResistanceLevel float64 // HVN resistance / target at entry; 0 when n/a
	ATR             float64 // ATR at entry; 0 when n/a
	EntryReason     string  // human-readable entry rationale captured at entry
	ExitReason      string  // human-readable exit rationale captured at exit; empty when n/a
}

// EquityPoint is portfolio value at one bar.
type EquityPoint struct {
	Time   time.Time
	Equity float64 // cash + position * close
}

// Result is the raw outcome of a single backtest run.
type Result struct {
	Trades       []Trade
	Equity       []EquityPoint
	InitialCash  float64
	FinalEquity  float64
	BarsInMarket int // bars with an open position (for exposure)
}

// Metrics are qualitative measures derived from a Result.
type Metrics struct {
	TotalTrades    int
	Wins, Losses   int
	WinRate        float64 // Wins/TotalTrades
	LossRate       float64
	GrossProfit    float64 // sum of positive PnL
	GrossLoss      float64 // sum of |negative PnL|
	ProfitFactor   float64 // GrossProfit/GrossLoss; if GrossLoss==0 and GrossProfit>0 -> GrossProfit
	NetPnL         float64 // FinalEquity - InitialCash
	NetPnLPct      float64
	MaxDrawdown    float64 // absolute, from the equity curve
	MaxDrawdownPct float64
	AvgWin         float64
	AvgLoss        float64
	Expectancy     float64 // average PnL per trade
	Sortino        float64 // mean trade PnL / downside deviation of trade PnL
	BestTrade      float64
	WorstTrade     float64
	ExposurePct    float64 // fraction of bars in market
	CAGR           float64 // annualized return over the run duration
}

// ParamLine is one strategy parameter rendered for the report header.
type ParamLine struct {
	Name  string
	Value string
}

// Meta is the report header context for a single run.
type Meta struct {
	Ticker       string
	Interval     string
	From         time.Time
	To           time.Time
	InitialCash  float64
	Fraction     float64
	Commission   float64
	Params       []ParamLine
	OpenPosition bool // a position was still open at the end
}
