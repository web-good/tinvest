package golden_x

import "sync"

type alertTier int

const (
	tierNone       alertTier = iota
	tierYellow               // buy: RSI < p15
	tierGreen                // buy: RSI < p5
	tierSellYellow           // sell (Gold only): RSI > p80
	tierSellOrange           // sell: RSI > p90 (Growth's single tier)
	tierSellRed              // sell (Gold only): RSI > p95
)

type alertSnapshot struct {
	tier       alertTier
	trendOK    bool
	divergence bool
	volumeOK   bool
}

// alertState tracks the last snapshot emitted per shareID and decides whether
// a new alert should be sent. An alert fires when the tier changes (and is not
// tierNone) OR when an enrichment improves (false→true) at the same non-None tier.
type alertState struct {
	mu   sync.Mutex
	last map[string]alertSnapshot
}

func newAlertState() *alertState {
	return &alertState{last: make(map[string]alertSnapshot)}
}

// ShouldAlert returns true if a fresh alert should be emitted for shareID.
// On a tierNone snapshot it resets the stored state silently.
func (s *alertState) ShouldAlert(shareID string, snap alertSnapshot) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.last[shareID]
	s.last[shareID] = snap
	if snap.tier == tierNone {
		return false
	}
	if prev.tier != snap.tier {
		return true
	}
	return (!prev.trendOK && snap.trendOK) ||
		(!prev.divergence && snap.divergence) ||
		(!prev.volumeOK && snap.volumeOK)
}
