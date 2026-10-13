package live

import "testing"

// TestNoLiveLiteralUsesTheCloseStop: the live runner always places an exchange stop at
// core.StopLevel and knows nothing about StopOnClose/CatStopMult. A literal with the close-based
// stop would trade with an intrabar stop in prod while its backtest assumed a close-based one.
// Lift this guard only together with live support.
func TestNoLiveLiteralUsesTheCloseStop(t *testing.T) {
	for ticker, p := range paramsByTicker {
		if p.StopOnClose != 0 || p.CatStopMult != 0 {
			t.Errorf("%s: StopOnClose=%d CatStopMult=%v — live has no close-based stop yet", ticker, p.StopOnClose, p.CatStopMult)
		}
	}
}
