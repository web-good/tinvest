package live

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestClosedEpisodeIsOneDurableAccountingTransition(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	x.book(499, false)
	x.step(t)
	data, err := os.ReadFile(x.s.Dir + "/events.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("episode_settled")) {
		t.Fatal("accounting saved before episode cleared; replay could double count")
	}
}

func TestUnresolvedExitRespectsPollingRate(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	x.b.unknownChild = true
	x.book(499, false)
	_ = x.e.Step(context.Background(), x.now)
	next := x.e.next
	_ = x.e.Step(context.Background(), x.now.Add(time.Millisecond))
	if !x.e.next.Equal(next) {
		t.Fatal("unresolved exit bypassed polling limit repeatedly")
	}
}
func TestAnotherJournalDirectoryCannotBypassRecovery(t *testing.T) {
	account := uuid.NewString()
	s, err := OpenStore(t.TempDir(), account)
	if err != nil {
		t.Fatal(err)
	}
	state := s.State()
	state.Episodes["uid"] = &Episode{Entry: Order{Key: "uncertain", Sent: true, Lots: 1}}
	if err = s.Save(state, "entry_intent"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	other, err := OpenStore(t.TempDir(), account)
	if err == nil {
		other.Close()
		t.Fatal("fresh state directory bypassed persisted uncertain order")
	}
}
func TestApplyOrderRejectsAnotherInstrumentOrSide(t *testing.T) {
	for _, bad := range []Order{{ID: "x", Lots: 1, InstrumentID: "other"}, {ID: "x", Lots: 1, Kind: "sell"}} {
		x := setup(t)
		x.step(t)
		bad.ID = x.e.episode.Entry.ID
		if err := x.e.applyOrder(&x.e.episode.Entry, bad); err == nil {
			t.Fatal("mismatched order accepted")
		}
	}
}
func TestFinishDoesNotMaskJoinedOperationalErrors(t *testing.T) {
	disk := errors.New("journal failed")
	if err := FinishError(errors.Join(context.Canceled, disk), nil); !errors.Is(err, disk) {
		t.Fatal("joined operational failure suppressed")
	}
}
