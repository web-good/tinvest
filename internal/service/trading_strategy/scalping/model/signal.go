package model

// SignalKind enumerates the possible decisions per instrument.
type SignalKind int

const (
	SignalNone SignalKind = iota
	SignalBuy
	SignalSell
)

// Signal is a rendered trade alert for one instrument.
type Signal struct {
	Kind           SignalKind
	InstrumentID   string
	InstrumentName string
	Ticker         string
	Price          float64
	TakeProfit     float64
	StopLoss       float64
	RSI            float64
	Reason         string // "TP" or "SL" for sells
}
