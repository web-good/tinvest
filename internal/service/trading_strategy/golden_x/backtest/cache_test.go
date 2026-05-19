package backtest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tinvest/internal/model"
)

type fakeFetcher struct {
	calls   int
	candles []*model.CandleItemTechAnalyse
	err     error
}

func (f *fakeFetcher) Fetch(_ context.Context, _ string) ([]*model.CandleItemTechAnalyse, error) {
	f.calls++
	return f.candles, f.err
}

func sampleCandles() []*model.CandleItemTechAnalyse {
	return []*model.CandleItemTechAnalyse{
		{Time: time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			Open:   model.Quotation{Units: 273, Nano: 410000000},
			High:   model.Quotation{Units: 281, Nano: 200000000},
			Low:    model.Quotation{Units: 270, Nano: 50000000},
			Close:  model.Quotation{Units: 278, Nano: 100000000},
			Volume: 18420000},
		{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			Open:   model.Quotation{Units: 279},
			High:   model.Quotation{Units: 282},
			Low:    model.Quotation{Units: 275},
			Close:  model.Quotation{Units: 280},
			Volume: 15000000},
	}
}

func TestCache_FetchOnMiss(t *testing.T) {
	dir := t.TempDir()
	f := &fakeFetcher{candles: sampleCandles()}
	c := NewCache(dir, f, false)

	got, err := c.Get(context.Background(), "SBER")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if f.calls != 1 {
		t.Fatalf("fetcher called %d times, want 1", f.calls)
	}
	if _, err := os.Stat(filepath.Join(dir, "SBER_W.json")); err != nil {
		t.Fatalf("expected cache file: %v", err)
	}
}

func TestCache_HitNoFetch(t *testing.T) {
	dir := t.TempDir()
	f := &fakeFetcher{candles: sampleCandles()}
	c := NewCache(dir, f, false)

	if _, err := c.Get(context.Background(), "SBER"); err != nil {
		t.Fatalf("priming Get: %v", err)
	}
	f.err = errors.New("boom")
	f.candles = nil
	f.calls = 0

	got, err := c.Get(context.Background(), "SBER")
	if err != nil {
		t.Fatalf("cache hit Get: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if f.calls != 0 {
		t.Fatalf("fetcher called on hit (%d calls)", f.calls)
	}
}

func TestCache_RefreshOverwrites(t *testing.T) {
	dir := t.TempDir()
	f := &fakeFetcher{candles: sampleCandles()}
	cWarm := NewCache(dir, f, false)
	if _, err := cWarm.Get(context.Background(), "SBER"); err != nil {
		t.Fatalf("warming: %v", err)
	}

	f2 := &fakeFetcher{candles: append(sampleCandles(), &model.CandleItemTechAnalyse{
		Time:  time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC),
		Close: model.Quotation{Units: 290},
	})}
	cRefresh := NewCache(dir, f2, true)
	got, err := cRefresh.Get(context.Background(), "SBER")
	if err != nil {
		t.Fatalf("refresh Get: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("after refresh len=%d, want 3", len(got))
	}
	if f2.calls != 1 {
		t.Fatalf("refresh fetcher calls = %d, want 1", f2.calls)
	}
}

func TestCache_RoundTripPreservesFields(t *testing.T) {
	dir := t.TempDir()
	f := &fakeFetcher{candles: sampleCandles()}
	c := NewCache(dir, f, false)
	_, _ = c.Get(context.Background(), "SBER")

	c2 := NewCache(dir, &fakeFetcher{err: errors.New("must not call")}, false)
	got, err := c2.Get(context.Background(), "SBER")
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	want := sampleCandles()
	if len(got) != len(want) {
		t.Fatalf("len mismatch")
	}
	for i := range want {
		if !got[i].Time.Equal(want[i].Time) {
			t.Fatalf("Time[%d] mismatch", i)
		}
		if got[i].Close.Units != want[i].Close.Units || got[i].Close.Nano != want[i].Close.Nano {
			t.Fatalf("Close[%d] mismatch", i)
		}
		if got[i].Volume != want[i].Volume {
			t.Fatalf("Volume[%d] mismatch", i)
		}
	}
}
