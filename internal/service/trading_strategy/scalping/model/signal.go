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
	Reason         string  // exit reason code, e.g. "TP", "SL", "TRAIL", "ATRSL", "OB", "RSI50"; drives backtest fill pricing (see IsStopReason); ignored for entries
	EntryReason    string  // human-readable entry rationale (set on Buy); empty for sells
	ExitReason     string  // human-readable exit rationale (set on Sell); empty for buys
}

// IsStopReason reports whether reason is a stop-style exit. The list is
// load-bearing for the backtest engine (internal/domain/backtest/engine.go,
// Run): stop reasons fill intrabar at min(StopLoss, open) — the stop level,
// adjusted for a gap-down open; "TP" fills at max(TakeProfit, open); any other
// Sell reason fills at the bar's close. A new stop-style exit reason MUST be
// added here, otherwise the backtest will silently fill it at close and
// understate gap/stop slippage.
func IsStopReason(reason string) bool {
	switch reason {
	case "SL", "TRAIL", "ATRSL":
		return true
	}
	return false
}
