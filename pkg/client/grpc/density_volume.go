package grpc

import (
	"context"
	"fmt"
	"time"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/volume"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AverageVolume reads completed five-minute candles in lots.
func (d *DensityMarketData) AverageVolume(ctx context.Context, id string, from, to time.Time) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	resp, err := d.market.GetCandles(ctx, &investapi.GetCandlesRequest{
		InstrumentId: &id, From: timestamppb.New(from), To: timestamppb.New(to),
		Interval: investapi.CandleInterval_CANDLE_INTERVAL_5_MIN,
	}, NewRPCCredential(d.auth))
	if status.Code(err) == codes.NotFound {
		return -1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("five-minute candles: %w", err)
	}
	samples := volume.Samples{From: from, To: to}
	for _, c := range resp.GetCandles() {
		if c.GetIsComplete() && c.GetTime() != nil && c.GetTime().CheckValid() == nil {
			samples.Add(c.GetTime().AsTime(), float64(c.GetVolume()))
		}
	}
	return samples.Mean(), nil
}
