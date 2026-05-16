package golden_x

import "sync"

type alertTier int

const (
	tierNone alertTier = iota
	tierYellow
	tierGreen
)

// alertState tracks the last tier emitted per shareID and decides whether
// a new alert should be sent. An alert fires only when the tier changes
// AND the new tier is not tierNone (RSI above p15 means "silent reset").
type alertState struct {
	mu   sync.Mutex
	last map[string]alertTier
}

func newAlertState() *alertState {
	return &alertState{last: make(map[string]alertTier)}
}

// ShouldAlert returns true if a fresh alert with `tier` should be emitted
// for `shareID`. On a tierNone input it resets the stored state silently.
func (s *alertState) ShouldAlert(shareID string, tier alertTier) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.last[shareID]
	if tier == tierNone {
		s.last[shareID] = tier
		return false
	}
	if prev == tier {
		return false
	}
	s.last[shareID] = tier
	return true
}
