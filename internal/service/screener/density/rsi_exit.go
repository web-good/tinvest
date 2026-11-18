package density

import (
	"context"
	"time"
	"tinvest/pkg/indicators"
)

// ExitRSIProfit identifies the early RSI exit in journals and alerts.
const ExitRSIProfit = "rsi_profit"

const rsiPeriod = 6

type minuteCloseSource interface {
	// MinuteCloses returns closed prices oldest first, ending in the minute before end.
	MinuteCloses(ctx context.Context, id string, end time.Time) ([]float64, error)
}

type rsiSnapshot struct {
	end   time.Time
	value float64
}

type rsiHistory struct {
	snapshot  rsiSnapshot
	inFlight  chan rsiSnapshot
	nextRetry time.Time
}

// NewRSIExit creates a check owned by a single execution loop. Candle requests
// run separately so a slow history response cannot delay existing exits.
func NewRSIExit(ctx context.Context, market any) func(id string, entry, bid int64, now time.Time) bool {
	source, ok := market.(minuteCloseSource)
	if !ok {
		return nil
	}
	histories := make(map[string]*rsiHistory)
	requests := make(chan struct{}, 4)
	return func(id string, entry, bid int64, now time.Time) bool {
		h := histories[id]
		if h == nil {
			h = &rsiHistory{}
			histories[id] = h
		}
		end := now.Truncate(time.Minute)
		select {
		case result := <-h.inFlight:
			h.snapshot = result
			h.inFlight = nil
			if !result.end.IsZero() {
				h.nextRetry = time.Time{}
			}
		default:
		}
		if h.snapshot.end.Equal(end) {
			// Round the minimum price gain up to a whole nanoruble.
			return h.snapshot.value > 70 && entry > 0 && bid > entry && bid-entry >= (entry-1)/500+1
		}
		if h.inFlight != nil || now.Before(h.nextRetry) || ctx.Err() != nil {
			return false
		}
		select {
		case requests <- struct{}{}:
		default:
			return false
		}
		result := make(chan rsiSnapshot, 1)
		h.inFlight = result
		h.nextRetry = now.Add(15 * time.Second)
		go func() {
			defer func() { <-requests }()
			closes, err := source.MinuteCloses(ctx, id, end)
			if err != nil || len(closes) <= rsiPeriod {
				result <- rsiSnapshot{}
				return
			}
			values := indicators.WilderRSISeries(closes, rsiPeriod)
			result <- rsiSnapshot{end: end, value: values[len(values)-1]}
		}()
		return false
	}
}
