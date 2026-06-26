package marketdata

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	imodel "tinvest/internal/model"
)

// fakeCandleClient returns a fixed hourly series and an empty 4H series.
type fakeCandleClient struct {
	hourly []*imodel.CandleItemTechAnalyse
}

func (f *fakeCandleClient) GetCandles(_ context.Context, _ *string, interval int32,
	_, _ *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	// Hour1 == 4 per enum.ToNumberInvestApi mapping; anything else is the 4H request.
	if interval == 4 {
		return f.hourly, nil
	}
	return nil, nil
}

func apiCandle(ts time.Time, o, h, l, c float64, v int64, complete bool) *imodel.CandleItemTechAnalyse {
	q := func(f float64) imodel.Quotation {
		units := int64(f)
		nano := int32((f - float64(units)) * 1e9)
		return imodel.Quotation{Units: units, Nano: nano}
	}
	return &imodel.CandleItemTechAnalyse{
		Time: ts, Open: q(o), High: q(h), Low: q(l), Close: q(c), Volume: v, IsComplete: complete,
	}
}

func TestAssemble_ParityWithBacktest(t *testing.T) {
	base := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	var api []*imodel.CandleItemTechAnalyse
	var dom []backtest.Candle
	for i := 0; i < 60; i++ {
		ts := base.Add(time.Duration(i) * time.Hour)
		o, h, l, c, v := 100.0+float64(i), 101.0+float64(i), 99.0+float64(i), 100.5+float64(i), int64(1000+i)
		api = append(api, apiCandle(ts, o, h, l, c, v, true))
		dom = append(dom, backtest.Candle{Time: ts, Open: o, High: h, Low: l, Close: c, Volume: v})
	}
	// Append a still-forming bar that the live path must drop.
	ts := base.Add(60 * time.Hour)
	api = append(api, apiCandle(ts, 999, 999, 999, 999, 9, false))

	const lookback = 50
	c := &fakeCandleClient{hourly: api}
	live, err := Assemble(context.Background(), c, "uid", lookback, 0, ts.Add(time.Hour))
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Backtest reference: last `lookback` completed domain candles, cur = last bar open.
	window := dom[len(dom)-lookback:]
	want := backtest.AssembleMarketData(window, nil, nil, window[len(window)-1].Time)

	if diff := cmp.Diff(want, live); diff != "" {
		t.Fatalf("snapshot parity mismatch (-backtest +live):\n%s", diff)
	}
}

func TestAssemble_ErrorsOnInsufficientCandles(t *testing.T) {
	c := &fakeCandleClient{hourly: []*imodel.CandleItemTechAnalyse{
		apiCandle(time.Now(), 1, 1, 1, 1, 1, true),
	}}
	if _, err := Assemble(context.Background(), c, "uid", 50, 0, time.Now()); err == nil {
		t.Fatal("expected error when completed candles < lookback")
	}
}
