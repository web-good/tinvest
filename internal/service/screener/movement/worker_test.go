package movement

import (
	"testing"
	"time"
)

func TestSummarize(t *testing.T) {
	end := time.Unix(3600, 0)
	bars := []bar{}
	for i := 6; i > 0; i-- {
		bars = append(bars, bar{end.Add(-time.Duration(i) * step), 100, 100.5, 100})
	}
	got, ok := summarize(bars, end, 6, 0.5)
	if !ok || got.mean != 0.5 || got.hits != 6 {
		t.Fatal(got, ok)
	}
	bars[0].high = 103
	bars[1].high = 100
	got, ok = summarize(bars, end, 6, 0.5)
	if !ok || got.hits != 5 || got.mean != 5.0/6 {
		t.Fatal(got, ok)
	}
	bars[1].at = end
	if _, ok = summarize(bars, end, 6, 0.5); ok {
		t.Fatal("accepted incomplete window")
	}
}
