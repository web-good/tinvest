package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func pdlExitP(buffer float64) Params {
	p := DefaultParams()
	p.PDLExitBelowPDL = 1
	p.PDLExitBufferATR = buffer
	return p
}

// pos0 entered Monday with PDL 99 and ATR 2: a buffer of 0.2 ATR exits on a close below 98.6, at
// the close; the stop 98.4 is not touched.
func TestPDLExitBuffer(t *testing.T) {
	if sig := held(pdlExitP(0.2), []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.5, 98.7}}, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("close 98.7: got %v %q, want None", sig.Kind, sig.Reason)
	}
	sig := held(pdlExitP(0.2), []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.5, 98.55}}, pos0())
	if sig.Kind != model.SignalSell || sig.Reason != "PDL" ||
		!strings.Contains(sig.ExitReason, "close 98.5500 < PDL дня входа 99.0000 − 0.20 ATR = 98.6000") {
		t.Fatalf("close 98.55: got %v %q %q, want Sell PDL", sig.Kind, sig.Reason, sig.ExitReason)
	}
	if sig := held(pdlExitP(0), []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.8, 98.9}}, pos0()); sig.Reason != "PDL" {
		t.Fatalf("buffer 0, close 98.9: got %q, want PDL", sig.Reason)
	}
	p := pdlExitP(0.2)
	p.PDLExitBelowPDL = 0
	if sig := held(p, []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.5, 98.55}}, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("off: got %v %q, want None", sig.Kind, sig.Reason)
	}
}

// The stop touched inside the bar wins over the close below PDL.
func TestPDLExitStopFirst(t *testing.T) {
	if sig := held(pdlExitP(0), []bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.4, 98.5}}, pos0()); sig.Reason != "SL" {
		t.Fatalf("got %q, want SL", sig.Reason)
	}
}

// Carried to Tuesday, the exit keeps the PDL of the entry date (Friday's low 99), not Monday's 97.
func TestPDLExitKeepsEntryDatePDL(t *testing.T) {
	md := fixture([]bar{fillBar, {"2026-09-15 10:30", 99.4, 99.5, 98.5, 98.55}})
	for j, tm := range md.DailyTimes {
		if dayKey(tm) == dayKey(at("2026-09-14 10:00")) {
			md.DailyLows[j] = 97
		}
	}
	pos := pos0()
	md.Position = &pos
	if sig := newS(pdlExitP(0.2)).Decide(md); sig.Reason != "PDL" {
		t.Fatalf("got %v %q, want PDL", sig.Kind, sig.Reason)
	}
}

func TestExplainPDLExit(t *testing.T) {
	md := fixture([]bar{fillBar, {"2026-09-14 10:30", 99.4, 99.5, 98.5, 98.55}})
	pos := pos0()
	md.Position = &pos
	out := newS(pdlExitP(0.2)).Explain(md)
	if !strings.Contains(out, "✓ выход ниже PDL дня входа: close 98.5500 < PDL 99.0000 − 0.20 ATR = 98.6000") {
		t.Fatalf("missing the PDL exit in:\n%s", out)
	}
}
