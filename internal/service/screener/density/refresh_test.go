package density

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUniverseSource struct {
	inputs []UniverseInput
	calls  int
	err    error
}

func (f *fakeUniverseSource) Instruments(context.Context) ([]UniverseInput, error) {
	return f.inputs, nil
}
func (f *fakeUniverseSource) Candles(ctx context.Context, id string) ([]DailyCandle, error) {
	f.calls++
	return nil, f.err
}
func TestRefreshPreservesFailureAndCancellation(t *testing.T) {
	f := &fakeUniverseSource{inputs: []UniverseInput{{Instrument: Instrument{ID: "x", Ticker: "X", Lot: 1}, Currency: "rub", Exchange: "MOEX", BuyAvailable: true}}, err: status.Error(codes.Unavailable, "unavailable")}
	if _, err := RefreshUniverse(context.Background(), f, DefaultConfig(), time.Millisecond); err == nil || f.calls != 3 {
		t.Fatalf("err=%v calls=%d", err, f.calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RefreshUniverse(ctx, f, DefaultConfig(), time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

type partialSource struct{}

func (partialSource) Instruments(context.Context) ([]UniverseInput, error) {
	return []UniverseInput{{Instrument: Instrument{ID: "missing", Ticker: "M", Lot: 10}, Currency: "rub", Exchange: "MOEX", BuyAvailable: true}, {Instrument: Instrument{ID: "good", Ticker: "G", Lot: 10}, Currency: "rub", Exchange: "MOEX", BuyAvailable: true}}, nil
}
func (partialSource) Candles(_ context.Context, id string) ([]DailyCandle, error) {
	if id == "missing" {
		return nil, status.Error(codes.NotFound, "no history")
	}
	a := make([]DailyCandle, 10)
	for i := range a {
		a[i] = DailyCandle{Time: time.Now().Add(-time.Duration(i+1) * 24 * time.Hour), Close: Price{Units: 100}, Volume: 100000, Complete: true}
	}
	return a, nil
}
func TestRefreshSkipsMissingHistory(t *testing.T) {
	got, err := RefreshUniverse(context.Background(), partialSource{}, DefaultConfig(), time.Millisecond)
	if err != nil || len(got) != 1 || got[0].ID != "good" {
		t.Fatalf("got %+v error %v", got, err)
	}
}
