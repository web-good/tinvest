package inplay

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/feed"
	"tinvest/internal/service/screener/bybitdensity/model"
)

func bybitBars(end time.Time, spread float64) []feed.SpotBar {
	var bars []feed.SpotBar
	for i := 14; i >= 0; i-- {
		bars = append(bars, feed.SpotBar{End: end.Add(-time.Duration(i) * step), Open: 1000, Low: 1000, High: 1000 + spread, Close: 1000})
	}
	return bars
}

type bybitFixture struct {
	instruments []model.Instrument
	requests    []string
	bars        map[string][]feed.SpotBar
}

func (f *bybitFixture) SpotTurnovers(context.Context) ([]model.Instrument, error) {
	return f.instruments, nil
}
func (f *bybitFixture) AvailableSpotBars(_ context.Context, k model.Key, interval time.Duration, from, to time.Time) ([]feed.SpotBar, error) {
	f.requests = append(f.requests, k.Symbol)
	if interval != step || to.Sub(from) != 24*time.Hour {
		return nil, errors.New("unexpected window")
	}
	b, ok := f.bars[k.Symbol]
	if !ok {
		return nil, errors.New("unavailable")
	}
	return b, nil
}

func TestBybitSpotReport(t *testing.T) {
	end := time.Date(2026, 11, 25, 15, 0, 0, 0, time.UTC)
	f := &bybitFixture{bars: map[string][]feed.SpotBar{}}
	for _, tc := range []struct {
		symbol           string
		turnover, spread float64
	}{
		{"LOWER", 90e6, 4}, {"UPPER", 90e6, 17}, {"EXACT", 80e6, 10}, {"BELOW", 79e6, 10}, {"JUST", 80e6 + 1, 10}, {"HIGH", 100e6, 17.01}, {"LOW", 100e6, 3.99}, {"GAP", 100e6, 8}, {"STALE", 100e6, 8}, {"SHORT", 100e6, 8}, {"ERROR", 100e6, 8}, {"FUTURE", 100e6, 8},
	} {
		f.instruments = append(f.instruments, model.Instrument{Key: model.Key{Market: model.Spot, Symbol: tc.symbol}, Turnover: tc.turnover})
		f.bars[tc.symbol] = bybitBars(end, tc.spread)
	}
	for i := range f.bars["UPPER"] {
		b := &f.bars["UPPER"][i]
		b.Open, b.Low, b.High, b.Close = 100, 100, 101.7, 100
	}
	delete(f.bars, "ERROR")
	f.bars["GAP"][6].End = f.bars["GAP"][5].End
	f.bars["STALE"] = bybitBars(end.Add(-step), 8)
	f.bars["FUTURE"] = bybitBars(end.Add(step), 8)
	f.bars["SHORT"] = f.bars["SHORT"][1:]
	var report string
	w := BybitWorker{market: f, send: func(s string) error { report = s; return nil }}
	if err := w.scan(context.Background(), end); err == nil {
		t.Fatal("missing partial failure")
	}
	for _, symbol := range []string{"EXACT", "BELOW"} {
		if strings.Contains(strings.Join(f.requests, " "), symbol) {
			t.Fatalf("requested candles for %s", symbol)
		}
	}
	for _, symbol := range []string{"EXACT", "BELOW", "HIGH", "LOW ", "GAP", "STALE", "SHORT", "ERROR", "FUTURE"} {
		if strings.Contains(report, "\n"+symbol) {
			t.Fatal(report)
		}
	}
	a, b, c := strings.Index(report, "\nLOWER"), strings.Index(report, "\nUPPER"), strings.Index(report, "\nJUST")
	if a < 0 || b <= a || c <= b || !strings.Contains(report, "18:00 МСК") || !strings.Contains(report, "0.40%") || !strings.Contains(report, "1.70%") {
		t.Fatal(report)
	}
}

func TestBybitTrueRangeIncludesPreviousClose(t *testing.T) {
	end := time.Now().Truncate(time.Hour)
	bars := bybitBars(end, 0)
	bars[0].Close = 900
	got, ok := bybitNATR(bars, end)
	if !ok || got < 0.7142 || got > 0.7144 {
		t.Fatalf("NATR %v, valid %v", got, ok)
	}
}

func TestBybitEmptyAndFailedScan(t *testing.T) {
	end := time.Now().Truncate(time.Hour)
	var messages []string
	f := &bybitFixture{}
	w := BybitWorker{market: f, send: func(s string) error { messages = append(messages, s); return nil }}
	if err := w.scan(context.Background(), end); err != nil || len(messages) != 1 || !strings.Contains(messages[0], "нет") {
		t.Fatalf("%v %v", messages, err)
	}
	f.instruments = []model.Instrument{{Key: model.Key{Market: model.Spot, Symbol: "A"}, Turnover: 81e6}}
	if err := w.scan(context.Background(), end); err == nil || len(messages) != 1 {
		t.Fatalf("%v %v", messages, err)
	}
}
