package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/backtest/mocks"
)

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
