package live

import (
	"fmt"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
)

// migrateState retains request UUIDs and unfinished release intents from v1.
func migrateState(state State) (State, error) {
	if state.Version != 1 && state.Version != 2 {
		return state, fmt.Errorf("unsupported density journal version %d", state.Version)
	}
	if state.Episodes == nil {
		state.Episodes = make(map[string]*Episode)
	}
	if state.Releases == nil {
		state.Releases = make(map[string]accountcoord.Claim)
	}
	if state.Ownership == nil {
		state.Ownership = make(map[string]accountcoord.DensityPosition)
	}
	if state.OwnershipKeys == nil {
		state.OwnershipKeys = make(map[string]string)
	}
	if state.Episode != nil {
		id := state.Episode.Instrument.ID
		if existing := state.Episodes[id]; existing != nil {
			return state, fmt.Errorf("duplicate legacy density episode")
		}
		state.Episodes[id] = state.Episode
		state.Episode = nil
	}
	if state.ReleasePending != nil {
		state.Releases[state.ReleasePending.InstrumentID] = *state.ReleasePending
		state.ReleasePending = nil
	}
	for id, ep := range state.Episodes {
		if ep == nil || id == "" || ep.Instrument.ID != id {
			return state, fmt.Errorf("invalid density episode index")
		}
	}
	for id, claim := range state.Releases {
		if id == "" || claim.InstrumentID != id || claim.Owner != "density" || claim.Key == "" {
			return state, fmt.Errorf("invalid density release intent")
		}
	}
	state.Version = 2
	return state, nil
}
