package backtest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/backtest/mocks"
	"tinvest/pkg/logger"
)

// TestMain initializes the package-level logger: Load's head top-up warns via
// logger.Warn on a legitimately short history, and that call panics on a nil
// logger if Init is never called.
func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

func qt(v float64) model.Quotation { return model.Quotation{Units: int64(v), Nano: 0} }

func bar(tm time.Time, c float64, complete bool) *model.CandleItemTechAnalyse {
	return &model.CandleItemTechAnalyse{
		Time: tm, Open: qt(c), High: qt(c), Low: qt(c), Close: qt(c), Volume: 1, IsComplete: complete,
	}
}

func TestConvertCandlesDropsIncomplete(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	in := []*model.CandleItemTechAnalyse{
		bar(base, 10, true),
		bar(base.Add(time.Hour), 11, false), // dropped
	}
	out := convertCandles(in)
	if len(out) != 1 || out[0].Close != 10 {
		t.Fatalf("convert = %+v, want 1 complete bar @10", out)
	}
}

func TestMergeCandlesDedupAndSort(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := []backtest.Candle{{Time: base.Add(time.Hour), Close: 2}, {Time: base, Close: 1}}
	b := []backtest.Candle{{Time: base.Add(time.Hour), Close: 99}, {Time: base.Add(2 * time.Hour), Close: 3}}
	out := mergeCandles(a, b)
	if len(out) != 3 {
		t.Fatalf("merged len = %d, want 3", len(out))
	}
	if !out[0].Time.Equal(base) || !out[1].Time.Equal(base.Add(time.Hour)) || !out[2].Time.Equal(base.Add(2*time.Hour)) {
		t.Fatalf("not sorted: %+v", out)
	}
	if out[1].Close != 2 { // first occurrence wins on dup Time
		t.Fatalf("dedup kept wrong value: %f, want 2", out[1].Close)
	}
}

func TestSliceWindow(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var c []backtest.Candle
	for i := 0; i < 5; i++ {
		c = append(c, backtest.Candle{Time: base.Add(time.Duration(i) * time.Hour)})
	}
	got := sliceWindow(c, base.Add(time.Hour), base.Add(3*time.Hour))
	if len(got) != 3 { // inclusive [1h, 3h]
		t.Fatalf("window len = %d, want 3", len(got))
	}
}

func TestLoadNoFileFetchesAndCaches(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []*model.CandleItemTechAnalyse{
		bar(base, 10, true),
		bar(base.Add(time.Hour), 11, true),
		bar(base.Add(2*time.Hour), 12, true),
	}
	var calls int
	m := mocks.NewMockcandleFetcher(t)
	m.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *string, int32, *timestamppb.Timestamp, *timestamppb.Timestamp, *int32, bool) ([]*model.CandleItemTechAnalyse, error) {
			calls++
			return candles, nil
		})

	p := NewCandleProvider(m, t.TempDir())
	got, err := p.Load(context.Background(), "RUAL", "id-1", enum.Hour1, base, base.Add(2*time.Hour), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("loaded %d candles, want 3", len(got))
	}
	if calls == 0 {
		t.Fatal("expected at least one fetch on cold cache")
	}
	// Second load (no refresh): cache file exists; last cached == to, so no new
	// tail fetch is required.
	callsAfterFirst := calls
	if _, err := p.Load(context.Background(), "RUAL", "id-1", enum.Hour1, base, base.Add(2*time.Hour), false); err != nil {
		t.Fatal(err)
	}
	if calls != callsAfterFirst {
		t.Fatalf("warm cache refetched: calls %d -> %d", callsAfterFirst, calls)
	}
}

// TestLoadFetchesMissingHead pins that a warm but SHORT cache does not silently truncate the
// requested window. Load used to top up only the tail, so a cache starting later than `from`
// returned a shorter series with no error at all — a walk-forward would then be built on part
// of the history while the report still claimed the full range.
func TestLoadFetchesMissingHead(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// Cache holds hours 2..4 only; the caller asks for hours 0..4.
	cached := []backtest.Candle{
		{Time: base.Add(2 * time.Hour), Close: 12},
		{Time: base.Add(3 * time.Hour), Close: 13},
		{Time: base.Add(4 * time.Hour), Close: 14},
	}
	dir := t.TempDir()
	raw, err := json.Marshal(cached)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "T_Hour1.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	var reqFrom, reqTo time.Time
	var calls int
	m := mocks.NewMockcandleFetcher(t)
	m.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ *string, _ int32, from, to *timestamppb.Timestamp, _ *int32, _ bool) ([]*model.CandleItemTechAnalyse, error) {
			calls++
			reqFrom, reqTo = from.AsTime().UTC(), to.AsTime().UTC()
			return []*model.CandleItemTechAnalyse{
				bar(base, 10, true),
				bar(base.Add(time.Hour), 11, true),
			}, nil
		})

	p := NewCandleProvider(m, dir)
	got, err := p.Load(context.Background(), "T", "id-T", enum.Hour1, base, base.Add(4*time.Hour), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("loaded %d candles, want 5 — the head of the window was not fetched", len(got))
	}
	if !got[0].Time.Equal(base) {
		t.Fatalf("first candle = %s, want the requested from %s", got[0].Time, base)
	}
	if calls != 1 {
		t.Fatalf("fetcher called %d times, want exactly 1 (head only; the tail is already cached)", calls)
	}
	if !reqFrom.Equal(base) || !reqTo.Equal(base.Add(2*time.Hour)) {
		t.Fatalf("head fetch asked for [%s, %s], want [%s, %s] — only the missing head",
			reqFrom, reqTo, base, base.Add(2*time.Hour))
	}
}

// TestLoadShortHistoryDoesNotFail pins the legitimate case behind the head top-up: an
// instrument that simply did not trade as far back as `from`. The head fetch returns nothing,
// and Load must still return the cached window instead of failing the whole run.
func TestLoadShortHistoryDoesNotFail(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cached := []backtest.Candle{
		{Time: base.Add(2 * time.Hour), Close: 12},
		{Time: base.Add(3 * time.Hour), Close: 13},
	}
	dir := t.TempDir()
	raw, err := json.Marshal(cached)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "T_Hour1.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	m := mocks.NewMockcandleFetcher(t)
	m.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil) // no candles that far back

	p := NewCandleProvider(m, dir)
	got, err := p.Load(context.Background(), "T", "id-T", enum.Hour1, base, base.Add(3*time.Hour), false)
	if err != nil {
		t.Fatalf("Load failed on an instrument with a short history: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("loaded %d candles, want the 2 cached ones", len(got))
	}
}
