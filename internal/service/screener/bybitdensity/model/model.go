// Package model defines the normalized Bybit data shared by the scanner and simulator.
package model

import (
	"math"
	"time"
)

type Market string

const (
	Spot   Market = "spot"
	Linear Market = "linear"
)

type Key struct {
	Market Market
	Symbol string
}

type Instrument struct {
	Key          Key
	TickSize     float64
	TickSizeText string // Exact decimal price step from Bybit.
	MinPrice     float64
	MaxPrice     float64
	QtyStep      float64
	MinQty       float64
	MaxQty       float64
	MinNotional  float64
	Turnover     float64
}

func (i Instrument) Price(ticks int64) float64 {
	return float64(ticks) * i.TickSize
}

func (i Instrument) Ticks(price float64) int64 {
	return int64(math.Round(price / i.TickSize))
}

type Side string

const (
	Bid Side = "bid"
	Ask Side = "ask"
)

type Level struct {
	Ticks int64
	Qty   float64
}

type Book struct {
	Bids     []Level
	Asks     []Level
	At       time.Time
	Received time.Time
	Snapshot bool
}

type Trade struct {
	ID    string
	Ticks int64
	Qty   float64
	At    time.Time
}

type Candle struct {
	Start time.Time
	End   time.Time // Exclusive end of the exchange minute.
	Open  int64
	High  int64
	Low   int64
	Close int64
}

type Event struct {
	Key        Key
	Book       *Book
	Trades     []Trade
	Candle     *Candle
	TradeReady bool
	Reset      bool
}

type Signal struct {
	Key           Key
	Side          Side
	Ticks         int64
	Price         float64
	Qty           float64
	Notional      float64
	Multiple      float64
	Distance      float64
	Since         time.Time
	Observed      time.Duration
	TradeQty      float64
	TradeCoverage time.Duration
}

// TradeSummary describes the whole position and is attached to its final fill.
type TradeSummary struct {
	EntryAt      time.Time
	EntryPrice   float64
	ExitPrice    float64 // Quantity-weighted average across all exits.
	DensityPrice float64
	DensityQty   float64
	StopPrice    float64
	TakePrice    float64
	NetPnL       float64
	NetReturn    float64
	ExitReasons  []string
}

type Fill struct {
	Key       Key
	Side      Side // Direction of the position, Bid for long and Ask for short.
	Entry     bool
	Partial   bool
	Price     float64
	Qty       float64
	Reason    string
	Fee       float64
	NetPnL    float64
	NetReturn float64
	At        time.Time
	Closed    *TradeSummary `json:"closed,omitempty"`
}
