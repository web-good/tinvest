package core

// setupKind is what armed a setup.
type setupKind int

const (
	kindZone  setupKind = iota + 1 // a dip into the zone above PDL with a close at or above it
	kindBreak                      // a close below PDL
)

// setup is an armed setup whose signal is the scanned bar: the arming bar and the kind.
type setup struct {
	arm  int
	kind setupKind
}

// name labels the setup in reasons.
func (st setup) name() string {
	if st.kind == kindBreak {
		return "пробой PDL"
	}
	return "зона PDL"
}

// scanState is the setup machine's state while it replays the bars of a date.
type scanState int

const (
	stateIdle    scanState = iota // waiting for an arming bar
	stateArmed                    // armed, waiting for the first green histogram bar
	stateNeedRed                  // a signal burned the setup; a red/zero histogram bar resets it
)

// scan replays the bars of bar k's date up to k and reports the setup whose signal is bar k, when
// that setup's kind is enabled. On every bar, in order: a red/zero histogram bar ends the wait
// after a signal; a close below PDL arms (or re-arms) a break unless a break is already armed; an
// idle bar with low ≤ zoneTop and close ≥ PDL arms a zone (EntryZone on); an armed bar with a
// green histogram is the signal, which burns the setup whatever follows — a refused buy or a
// position already open. The arming bar may itself be the signal.
func (s *Strategy) scan(ser series, hist []float64, k int, pdl, zoneTop float64) (setup, bool) {
	start := k
	for start > 0 && dayKey(ser.t[start-1]) == dayKey(ser.t[k]) {
		start--
	}
	state, cur := stateIdle, setup{}
	for j := start; j <= k; j++ {
		if state == stateNeedRed && hist[j] <= 0 {
			state = stateIdle
		}
		switch {
		case state == stateNeedRed:
		case ser.c[j] < pdl && (state == stateIdle || cur.kind == kindZone):
			state, cur = stateArmed, setup{arm: j, kind: kindBreak}
		case state == stateIdle && s.p.EntryZone > 0 && ser.l[j] <= zoneTop && ser.c[j] >= pdl:
			state, cur = stateArmed, setup{arm: j, kind: kindZone}
		}
		if state == stateArmed && hist[j] > 0 {
			if j == k {
				return cur, s.enabled(cur.kind)
			}
			state = stateNeedRed
		}
	}
	return setup{}, false
}

// enabled reports whether setups of kind k may buy.
func (s *Strategy) enabled(k setupKind) bool {
	if k == kindBreak {
		return s.p.EntryBreak > 0
	}
	return s.p.EntryZone > 0
}
