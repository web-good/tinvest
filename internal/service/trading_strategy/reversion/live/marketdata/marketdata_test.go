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

// fakeHTFCandleClient returns a fixed hourly series and a fixed 4H series.
type fakeHTFCandleClient struct {
	hourly []*imodel.CandleItemTechAnalyse
	htf    []*imodel.CandleItemTechAnalyse
}

func (f *fakeHTFCandleClient) GetCandles(_ context.Context, _ *string, interval int32,
	_, _ *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	// Hour1 == 4; Hour4 == 11 per enum.ToNumberInvestApi mapping.
	if interval == 4 {
		return f.hourly, nil
	}
	return f.htf, nil
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

// TestAssemble_ParityWithBacktest_HTF verifies that Assemble anchors `cur` to the last
// completed hourly bar's open-time rather than to `now`. The discriminating design:
//
//   - The 4H series contains a "discriminating bar" D at time T such that
//     T.Add(4h) > window[last].Time  (NOT completed at correct anchor → excluded)
//     T.Add(4h) == now               (WOULD be completed at wrong anchor now → included)
//
// If the implementation uses `cur = now` instead of `cur = window[last].Time`, the
// snapshot will contain an extra 4H bar in HTFCloses/HTFHighs/HTFLows and the diff
// against the reference (built with the correct anchor) will be non-empty.
func TestAssemble_ParityWithBacktest_HTF(t *testing.T) {
	// Base timeline:
	//   base+0h … base+9h : 10 completed hourly bars (lookback=10)
	//   base+10h           : still-forming hourly bar (IsComplete=false, dropped)
	//   window[last].Time  = base+9h  (the correct `cur` anchor)
	//   now                = base+12h (3h after last completed hourly bar)
	//
	// 4H bars (all complete, 8 bars total; htfEMAPeriod=3 → fetchCompleted requests
	// 3+20=23 bars, but we only supply 8 so nothing is trimmed):
	//   bar0: T=base+0h   → T+4h=base+4h  ≤ base+9h → included at correct anchor ✓
	//   bar1: T=base+4h   → T+4h=base+8h  ≤ base+9h → included at correct anchor ✓
	//   bar2: T=base+8h   → T+4h=base+12h > base+9h → NOT completed at correct anchor
	//                      T+4h=base+12h = now       → WOULD be included at wrong anchor (DISCRIMINATING BAR)
	//
	// Reference: backtest.AssembleMarketData(hourlyWindow, nil, htfDomainCompleted, window[last].Time)
	// where htfDomainCompleted excludes bar2 (only bar0, bar1 are completed at base+9h).
	base := time.Date(2026, 2, 10, 8, 0, 0, 0, time.UTC)

	// Build hourly API series: 10 completed bars + 1 still-forming bar.
	const lookback = 10
	var apiHourly []*imodel.CandleItemTechAnalyse
	var domHourly []backtest.Candle
	for i := 0; i < lookback; i++ {
		ts := base.Add(time.Duration(i) * time.Hour)
		o, h, l, c, v := 200.0+float64(i), 201.0+float64(i), 199.0+float64(i), 200.5+float64(i), int64(500+i)
		apiHourly = append(apiHourly, apiCandle(ts, o, h, l, c, v, true))
		domHourly = append(domHourly, backtest.Candle{Time: ts, Open: o, High: h, Low: l, Close: c, Volume: v})
	}
	// Append still-forming bar (must be dropped by Assemble).
	apiHourly = append(apiHourly, apiCandle(base.Add(lookback*time.Hour), 999, 999, 999, 999, 0, false))

	lastCompletedTime := base.Add(time.Duration(lookback-1) * time.Hour) // base+9h
	now := lastCompletedTime.Add(3 * time.Hour)                          // base+12h

	// Build 4H API series: bar0, bar1, bar2 (all marked complete by the exchange).
	// bar2 is the discriminating bar: T+4h == now > lastCompletedTime.
	type htfSpec struct {
		ts         time.Time
		o, h, l, c float64
		v          int64
	}
	htfSpecs := []htfSpec{
		{base.Add(0 * time.Hour), 50.0, 52.0, 49.0, 51.0, 1000}, // bar0: T+4h=base+4h  ≤ base+9h
		{base.Add(4 * time.Hour), 51.0, 53.0, 50.0, 52.0, 1100}, // bar1: T+4h=base+8h  ≤ base+9h
		{base.Add(8 * time.Hour), 52.0, 54.0, 51.0, 53.0, 1200}, // bar2 (DISC): T+4h=base+12h=now
	}
	var apiHTF []*imodel.CandleItemTechAnalyse
	for _, s := range htfSpecs {
		apiHTF = append(apiHTF, apiCandle(s.ts, s.o, s.h, s.l, s.c, s.v, true))
	}

	// Reference: only bar0 and bar1 are completed at the correct anchor (lastCompletedTime).
	// bar2 is NOT completed because bar2.Time.Add(4h) = base+12h > base+9h = lastCompletedTime.
	var htfDomainCompleted []backtest.Candle
	for _, s := range htfSpecs[:2] { // bar0 and bar1 only
		htfDomainCompleted = append(htfDomainCompleted, backtest.Candle{
			Time: s.ts, Open: s.o, High: s.h, Low: s.l, Close: s.c, Volume: s.v,
		})
	}

	const htfEMAPeriod = 3
	client := &fakeHTFCandleClient{hourly: apiHourly, htf: apiHTF}
	live, err := Assemble(context.Background(), client, "uid", lookback, htfEMAPeriod, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Reference snapshot uses the correct anchor = window[last].Time = lastCompletedTime.
	want := backtest.AssembleMarketData(domHourly, nil, htfDomainCompleted, lastCompletedTime)

	if diff := cmp.Diff(want, live); diff != "" {
		t.Fatalf("HTF parity mismatch (-backtest +live):\n%s", diff)
	}
}
