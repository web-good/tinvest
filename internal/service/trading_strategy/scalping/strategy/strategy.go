package strategy

import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

// Position is an open long position in the strategy's instrument.
type Position struct {
	PurchasePrice float64
	Quantity      int64
	// StopLoss is the hard stop frozen at entry. Zero means "not set" (e.g. live
	// trading, which does not yet persist entry state — see the levels
	// entry-locked-stops spec). The backtest engine always populates it.
	StopLoss float64
	// EntryATR is the ATR captured at entry, used as the arm threshold unit.
	EntryATR float64
	// MaxFavorablePrice is the highest close seen since entry (monotonic
	// non-decreasing); it makes the trail's arming latch monotonic.
	MaxFavorablePrice float64
	// PrevMaxFavorablePrice is MaxFavorablePrice as of the PREVIOUS bar (before the
	// current bar's close was marked). The reversion intrabar stop computes its trail
	// level from it: the exchange stop order working during bar i was placed after bar
	// i-1 closed, so its level knows nothing about bar i. Seeded to the entry price.
	PrevMaxFavorablePrice float64
	// EntryTime is the open-time of the entry bar. Zero means "not set" (live
	// trading does not persist it); the backtest engine always populates it.
	// Time-based exits must degrade to a no-op when zero.
	EntryTime time.Time
}

// MarketData is the raw, per-instrument snapshot the runner hands to a strategy.
// All series are oldest-first and aligned to the same candles; Price is the last close.
type MarketData struct {
	Price   float64
	Highs   []float64
	Lows    []float64
	Closes  []float64
	Volumes []int64
	// Times are oldest-first bar open-times, index-aligned to Closes/Volumes. Empty when
	// the runner does not supply them (e.g. live trading); consumers must degrade
	// gracefully (the reversion volume filter skips weekend exclusion when Times is empty).
	Times []time.Time
	// DailyCloses are oldest-first closes of COMPLETED daily candles, aligned so the
	// last element is the most recent day closed at/before the current bar. Empty if
	// no higher-timeframe data is supplied or the filter is disabled.
	DailyCloses []float64
	// DailyHighs / DailyLows are oldest-first highs/lows of the same COMPLETED daily
	// candles as DailyCloses (aligned index-for-index). Empty when no daily data.
	DailyHighs []float64
	DailyLows  []float64
	// HTFCloses are oldest-first closes of COMPLETED higher-timeframe (4H) candles,
	// aligned so the last element is the most recent 4H bar fully closed at/before the
	// current bar. Empty if no HTF data is supplied or the filter is disabled.
	HTFCloses []float64
	// HTFHighs / HTFLows are oldest-first highs/lows of the same COMPLETED 4H candles
	// as HTFCloses (aligned index-for-index). Empty when no HTF data.
	HTFHighs []float64
	HTFLows  []float64
	// TodayHigh / TodayLow are the high/low across all bars of the current MSK
	// calendar day up to and including the current bar (no lookahead). Zero when n/a.
	TodayHigh float64
	TodayLow  float64
	Position  *Position // nil when flat
}

// Strategy is the per-share trading rule. Decide must be pure: it computes its own
// indicators from md and performs no I/O.
type Strategy interface {
	Ticker() string // e.g. "RUAL"
	Lookback() int  // number of candles it needs
	Decide(md MarketData) model.Signal
}
