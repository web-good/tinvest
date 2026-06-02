package universe

import "testing"

func TestTopN(t *testing.T) {
	in := []Scored{
		{InstrumentID: "a", ATRPercent: 1.0},
		{InstrumentID: "b", ATRPercent: 3.0},
		{InstrumentID: "c", ATRPercent: 2.0},
		{InstrumentID: "d", ATRPercent: 3.0},
	}

	got := TopN(in, 2)

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	// Highest ATR% first; ties broken by InstrumentID ascending ("b" < "d").
	if got[0].InstrumentID != "b" || got[1].InstrumentID != "d" {
		t.Fatalf("order = [%s %s], want [b d]", got[0].InstrumentID, got[1].InstrumentID)
	}
}

func TestTopN_FewerThanN(t *testing.T) {
	in := []Scored{{InstrumentID: "a", ATRPercent: 1.0}}
	got := TopN(in, 5)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestTopN_DoesNotMutateInput(t *testing.T) {
	in := []Scored{
		{InstrumentID: "a", ATRPercent: 1.0},
		{InstrumentID: "b", ATRPercent: 3.0},
	}
	_ = TopN(in, 2)
	if in[0].InstrumentID != "a" {
		t.Fatalf("input was mutated: in[0] = %s", in[0].InstrumentID)
	}
}
