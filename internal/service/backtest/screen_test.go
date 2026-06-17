package backtest

import (
	"strings"
	"testing"
)

func TestRenderScreenMarkdown(t *testing.T) {
	rows := []ScreenRow{
		{Ticker: "NVTK", VR2: 0.80, VR4: 0.75, VR8: 0.70, Autocorr1: -0.20, Verdict: "mean-reverting"},
		{Ticker: "AFKS", Note: "нет свечей"},
	}
	out := RenderScreenMarkdown(rows)
	if !strings.Contains(out, "NVTK") || !strings.Contains(out, "mean-reverting") {
		t.Fatalf("missing NVTK row:\n%s", out)
	}
	if !strings.Contains(out, "AFKS") || !strings.Contains(out, "нет свечей") {
		t.Fatalf("skipped row should surface its note:\n%s", out)
	}
}
