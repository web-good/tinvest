package density

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UniverseSource interface {
	Instruments(context.Context) ([]UniverseInput, error)
	Candles(context.Context, string) ([]DailyCandle, error)
}

// RefreshUniverse returns no partial result when a request fails; the caller can retain its previous universe.
func RefreshUniverse(ctx context.Context, source UniverseSource, cfg Config, interval time.Duration) ([]Instrument, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("history request interval must be positive")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	inputs, err := source.Instruments(ctx)
	if err != nil {
		return nil, fmt.Errorf("list instruments: %w", err)
	}
	for i := range inputs {
		if !eligible(inputs[i]) {
			continue
		}
		for attempt := 0; attempt < 3; attempt++ {
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
			inputs[i].Candles, err = source.Candles(ctx, inputs[i].Instrument.ID)
			if err == nil {
				break
			}
			code := status.Code(err)
			if code != codes.Unavailable && code != codes.ResourceExhausted && code != codes.DeadlineExceeded {
				break
			}
		}
		if err != nil {
			if status.Code(err) == codes.NotFound {
				inputs[i].Candles = nil
				continue
			}
			return nil, fmt.Errorf("history %s: %w", inputs[i].Instrument.ID, err)
		}
	}
	return SelectUniverse(inputs, cfg), nil
}
