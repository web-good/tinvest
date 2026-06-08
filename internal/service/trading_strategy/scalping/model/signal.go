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
	Level          float64 // entry support level (HVN); 0 when n/a
	ATR            float64 // ATR at entry; 0 when n/a
	Reason         string  // exit reason: "TP"/"SL"/"TRAIL"; ignored for entries
	EntryReason    string  // human-readable entry rationale (set on Buy); empty for sells
}
