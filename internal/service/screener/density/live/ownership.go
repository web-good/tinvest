package live

import (
	"slices"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
)

// rememberOwnership is part of the same durable transition as broker status.
// Closed IDs remain recognizable even to recovery scanning executed stop history.
func rememberOwnership(state *State) {
	for ticker, old := range state.Ownership {
		old.Quantity = 0
		state.Ownership[ticker] = old
	}
	for _, ep := range state.Episodes {
		owned := state.Ownership[ep.Instrument.Ticker]
		if previous := state.OwnershipKeys[ep.Instrument.Ticker]; previous != ep.Entry.Key {
			owned = accountcoord.DensityPosition{}
			state.OwnershipKeys[ep.Instrument.Ticker] = ep.Entry.Key
		}
		owned.InstrumentID = ep.Instrument.ID
		owned.Quantity = ep.Remaining() * ep.Instrument.Lot
		for _, stop := range []*Stop{&ep.Stop, &ep.Take} {
			if stop.ID != "" && !slices.Contains(owned.StopIDs, stop.ID) {
				owned.StopIDs = append(owned.StopIDs, stop.ID)
			}
		}
		for _, order := range []*Order{&ep.Entry, ep.Exit, ep.Stop.Child, ep.Take.Child} {
			if order != nil && order.ID != "" && !slices.Contains(owned.OrderIDs, order.ID) {
				owned.OrderIDs = append(owned.OrderIDs, order.ID)
			}
		}
		state.Ownership[ep.Instrument.Ticker] = owned
	}
}
