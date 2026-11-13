package density

import "testing"

func TestSubscriptionAckIgnoresPreviousSelection(t *testing.T) {
	a := newAckTracker()
	a.changed("x", true)
	a.changed("x", false)
	a.changed("x", true)
	if _, current := a.ack("x"); current {
		t.Fatal("old subscribe acknowledgement accepted")
	}
	if _, current := a.ack("x"); current {
		t.Fatal("old unsubscribe acknowledgement accepted")
	}
	subscribe, current := a.ack("x")
	if !current || !subscribe {
		t.Fatal("current subscribe not acknowledged")
	}
}
