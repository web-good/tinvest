package density

type pendingAck struct {
	revision  uint64
	subscribe bool
}
type ackTracker struct {
	revisions map[string]uint64
	pending   map[string][]pendingAck
}

func newAckTracker() *ackTracker {
	return &ackTracker{revisions: make(map[string]uint64), pending: make(map[string][]pendingAck)}
}
func (a *ackTracker) changed(id string, subscribe bool) {
	a.revisions[id]++
	a.pending[id] = append(a.pending[id], pendingAck{a.revisions[id], subscribe})
}
func (a *ackTracker) ack(id string) (bool, bool) {
	q := a.pending[id]
	if len(q) == 0 {
		return false, false
	}
	item := q[0]
	if len(q) == 1 {
		delete(a.pending, id)
	} else {
		a.pending[id] = q[1:]
	}
	return item.subscribe, item.revision == a.revisions[id]
}
