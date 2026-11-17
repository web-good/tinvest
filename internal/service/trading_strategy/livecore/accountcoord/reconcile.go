package accountcoord

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/livecore/adapter"
)

// ConfirmClosedCandle records an operator-confirmed external close after verify
// checks the broker for inventory and live orders. It holds the same lease as
// trading passes so a stale confirmation cannot remove a subsequent entry.
func (c *Coordinator) ConfirmClosedCandle(ctx context.Context, ticker, instrumentID string, entryTime time.Time, verify func(context.Context) error) (retErr error) {
	if c.legacy == nil || ticker == "" || instrumentID == "" || entryTime.IsZero() || verify == nil {
		return fmt.Errorf("reconciliation requires a candle store, instrument, entry time and broker verification")
	}
	lease, err := c.Lock(ctx)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lease.Close()) }()

	state, err := c.legacy.Load()
	if err != nil {
		return err
	}
	entry, ok := state[ticker]
	if !ok || !entry.EntryTime.Equal(entryTime) || entry.Quantity <= 0 {
		return fmt.Errorf("%s: no open candle entry with the specified time", ticker)
	}
	if entry.EntryRequestID != "" || entry.ExitRequestID != "" || entry.FVGExitStopID != "" ||
		entry.FVGExitOrderID != "" || entry.FVGStopPending || entry.FVGTargetPending || entry.FVGRecovering {
		return fmt.Errorf("%s: resolve pending candle order or recovery before confirming an external close", ticker)
	}

	claims, err := c.Claims(ctx)
	if err != nil {
		return err
	}
	var candle Claim
	for _, claim := range claims {
		if claim.Ticker != ticker && claim.InstrumentID != instrumentID {
			continue
		}
		if claim.Owner == "density" || claim.Owner != adapter.Owner(entry) ||
			claim.InstrumentID != instrumentID || claim.Ticker != ticker {
			return fmt.Errorf("%s: conflicting or density reservation requires reconciliation", ticker)
		}
		candle = claim
	}
	density, err := c.DensitySnapshot(ctx)
	if err != nil {
		return err
	}
	for symbol, owned := range density {
		if (symbol == ticker || owned.InstrumentID == instrumentID) && owned.Quantity != 0 {
			return fmt.Errorf("%s: density still has inventory", ticker)
		}
	}

	if err := verify(ctx); err != nil {
		return fmt.Errorf("%s: broker verification: %w", ticker, err)
	}
	if candle.Key != "" {
		// The entry still blocks competing candle strategies if saving below fails.
		if err := c.Remove(lease, candle); err != nil {
			return err
		}
	}
	delete(state, ticker)
	return c.legacy.Save(state)
}
