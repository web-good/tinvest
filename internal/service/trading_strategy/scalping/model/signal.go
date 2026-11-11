package model

import "time"

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
	StopDistance   float64 // optional fill-relative stop distance
	TargetDistance float64 // optional fill-relative target distance
	TakeProfit     float64
	StopLoss       float64
	RSI            float64
	Level          float64 // entry support level (HVN); 0 when n/a
	ATR            float64 // ATR at entry; 0 when n/a
	Reason         string  // exit reason code, e.g. "TP", "SL", "TRAIL", "ATRSL", "OB", "RSI50"; drives backtest fill pricing (see IsStopReason); ignored for entries
	EntryReason    string  // human-readable entry rationale (set on Buy); empty for sells
	ExitReason     string  // human-readable exit rationale (set on Sell); empty for buys
	// FillBy is, for a Buy deferred by backtest Config.EntryAtNextOpen, the latest candle open
	// time at which it may still fill; a pending Buy whose next candle opens later is dropped.
	// Zero means no expiry.
	FillBy time.Time
	// MaxEntryPrice caps the next-open fill price for a deferred Buy. Zero means
	// no limit; an open equal to the limit may fill.
	MaxEntryPrice float64
	// MinEntryPrice is the minimum execution price; zero disables the floor.
	// An execution exactly at the floor is allowed.
	MinEntryPrice float64
}

// IsCloseConfirmedStopReason reports whether an exit requires a completed candle
// and fills at its close rather than at an intrabar stop level.
func IsCloseConfirmedStopReason(reason string) bool {
	return reason == "SL_CLOSE"
}

// IsStopReason reports whether reason is a stop-style exit. In backtest Run and
// Trace, close-confirmed stops fill at the candle close; other stop reasons fill
// at min(StopLoss, open), adjusted for gap-down risk. New stop reasons must be
// listed here and classified by IsCloseConfirmedStopReason when appropriate.
func IsStopReason(reason string) bool {
	switch reason {
	case "SL", "SL_CLOSE", "SL_EMERGENCY", "TRAIL", "ATRSL", "CATSL":
		return true
	}
	return false
}
