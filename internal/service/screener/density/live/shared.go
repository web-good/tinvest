package live

import (
	"context"
	"errors"
	"fmt"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
)

func episodeClaim(ep *Episode) accountcoord.Claim {
	key := ep.ClaimKey
	if key == "" {
		key = ep.Entry.Key
	}
	return accountcoord.Claim{Ticker: ep.Instrument.Ticker, InstrumentID: ep.Instrument.ID, Owner: "density", Key: key}
}

// AttachCoordinator must run before the continuous worker starts making decisions.
func (e *Engine) AttachCoordinator(ctx context.Context, c *accountcoord.Coordinator) error {
	if c == nil || c.AccountHash() != e.state.AccountHash {
		return fmt.Errorf("shared coordinator account mismatch")
	}
	if e.state.SharedRoot != "" && e.state.SharedRoot != c.Root() {
		return fmt.Errorf("shared ownership directory changed; use original coordinator")
	}
	if e.state.SharedRoot == "" && len(e.state.Episodes) != 0 {
		return fmt.Errorf("unresolved standalone episode must be settled before shared-account activation")
	}
	e.coordinator = c
	if e.state.SharedRoot == "" {
		e.state.SharedRoot = c.Root()
		if err := e.save("shared_account_attached"); err != nil {
			return err
		}
	}
	if len(e.state.Releases) != 0 {
		if err := e.releaseOwnership(ctx); err != nil {
			return err
		}
	}
	claims, err := c.DensityClaims(ctx)
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if claim.Owner == "density" && claim.Status == "active" {
			ep := e.state.Episodes[claim.InstrumentID]
			if ep == nil || claim.Ticker != ep.Instrument.Ticker || claim.InstrumentID != ep.Instrument.ID || claim.Key != episodeClaim(ep).Key {
				return fmt.Errorf("shared registry contains a density episode absent from this journal")
			}
		}
	}
	for _, ep := range e.state.Episodes {
		claim := episodeClaim(ep)
		if err := c.Verify(ctx, claim); err != nil {
			if restoreErr := restoreReservation(ctx, c, claim); restoreErr != nil {
				return restoreErr
			}
		}
	}
	return nil
}

func restoreReservation(ctx context.Context, c *accountcoord.Coordinator, claim accountcoord.Claim) (retErr error) {
	// An intent can be durable before Reserve completed; no BUY is sent before
	// Reserve. Restore only a ticker with no existing density ownership record.
	lease, err := c.Lock(ctx)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lease.Close()) }()
	claims, err := c.DensityClaims(ctx)
	if err != nil {
		return err
	}
	if _, exists := claims[claim.Ticker]; exists {
		return fmt.Errorf("existing density reservation does not match persisted episode")
	}
	return c.Reserve(lease, claim)
}
func (e *Engine) releaseOwnership(ctx context.Context) error {
	if len(e.state.Releases) == 0 {
		return nil
	}
	if e.coordinator == nil {
		return fmt.Errorf("coordinator required to finish ownership release")
	}
	var failed []error
	for id, claim := range e.state.Releases {
		if err := e.coordinator.MarkClosed(ctx, claim); err != nil {
			failed = append(failed, err)
			continue
		}
		delete(e.state.Releases, id)
		if err := e.save("shared_ownership_closed"); err != nil {
			return err
		}
	}
	return errors.Join(failed...)
}
