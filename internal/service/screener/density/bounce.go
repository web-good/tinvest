package density

import (
	"time"

	"tinvest/internal/service/screener/bounce"
)

type Candle struct {
	Time                   time.Time
	Open, High, Low, Close Price
}

func priceUnits(p Price) int64 {
	if !valid(p) || p.Units > 100_000_000 {
		return 0
	}
	return p.Units*1e9 + int64(p.Nano)
}

// BounceWatch adapts MOEX book levels and closed candles to the entry sequence.
type BounceWatch struct {
	watch bounce.Watch[Signal]
	step  int64
}

func (w *BounceWatch) Book(b Book, signals []Signal, step Price) {
	w.step = priceUnits(step)
	w.watch.Book(b.Time, func(s Signal) bool {
		for _, level := range b.Bids {
			if level.Price == s.Price {
				return bounce.Retains(float64(s.Quantity), float64(level.Quantity))
			}
		}
		return false
	})
	if w.step <= 0 {
		return
	}
	for _, s := range signals {
		price := priceUnits(s.Price)
		if price%w.step == 0 {
			w.watch.Track(price/w.step, s)
		}
	}
}

func (w *BounceWatch) Candle(c Candle, now time.Time) {
	if w.step > 0 {
		low, closePrice := priceUnits(c.Low), priceUnits(c.Close)
		if low%w.step == 0 && closePrice%w.step == 0 {
			w.watch.Candle(c.Time, low/w.step, closePrice/w.step, now)
		}
	}
}

func (w *BounceWatch) Ready(now time.Time) (Signal, time.Time, bool) {
	return w.watch.Ready(now)
}

func (w *BounceWatch) Consume(now time.Time) { w.watch.Consume(now) }
