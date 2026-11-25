package main

import (
	"strings"
	"testing"
	"time"
)

func TestReportInPlayAcrossBatches(t *testing.T) {
	start := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	f := inPlayFilter{status: map[time.Time]bool{
		start: true, start.Add(checkInterval): true, start.Add(90 * time.Minute): true,
	}}
	split := start.Add(17 * time.Minute)
	periods := appendPeriods(nil, &f, start, split)
	end := start.Add(2 * time.Hour)
	periods = appendPeriods(periods, &f, split, end)
	if len(periods) != 2 || periods[0] != (inPlayPeriod{start, start.Add(time.Hour)}) || periods[1] != (inPlayPeriod{start.Add(90 * time.Minute), end}) {
		t.Fatalf("unexpected periods: %+v", periods)
	}
}

func TestReportMetricsCompounded(t *testing.T) {
	start := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)
	e := engine{done: []trade{
		{opened: start, closed: start.Add(time.Hour), net: .1},
		{opened: end.Add(-time.Hour), closed: end, net: -.1},
	}}
	var b strings.Builder
	writeMetrics(&b, start, end, &e)
	for _, want := range []string{"Win rate | 50.00%", "Чистый PnL | -100.00 (-1.00%)", "Profit factor | 0.909", "просадка по закрытиям | 10.00%", "Время в позиции | 4.17%"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	b.Reset()
	writeMetrics(&b, start, end, &engine{})
	if strings.Contains(b.String(), "NaN") || !strings.Contains(b.String(), "Win rate | —") {
		t.Fatal(b.String())
	}
}
