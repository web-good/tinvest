package strategy

import "tinvest/internal/service/trading_strategy/scalping/model"

// Position is an open long position in the strategy's instrument.
type Position struct {
	PurchasePrice float64
	Quantity      int64
}

// MarketData is the raw, per-instrument snapshot the runner hands to a strategy.
// All series are oldest-first and aligned to the same candles; Price is the last close.
type MarketData struct {
	Price    float64
	Highs    []float64
	Lows     []float64
	Closes   []float64
	Volumes  []int64
	Position *Position // nil when flat
}

// Strategy is the per-share trading rule. Decide must be pure: it computes its own
// indicators from md and performs no I/O.
type Strategy interface {
	Ticker() string // e.g. "RUAL"
	Lookback() int   // number of candles it needs
	Decide(md MarketData) model.Signal
}
