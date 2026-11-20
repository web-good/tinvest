package strategy

import (
	"math"
	"slices"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	searchDistance   = .003
	approachDistance = .0005
	zoneFraction     = .0002
	zoneMultiple     = 3
	zoneAge          = 8 * time.Second
	retainedFraction = .6
)

type bidZone struct {
	low        int64
	qty, value float64
}

// Compare observed price bands of equal width, using the nearest populated bands.
func zoneMedian(zones []bidZone, index int) float64 {
	var values [10]float64
	n := min(len(zones)-1, len(values))
	if n < 3 {
		return 0
	}
	left, right := index-1, index+1
	for i := range n {
		if right == len(zones) || left >= 0 && zones[left].low-zones[index].low <= zones[index].low-zones[right].low {
			values[i] = zones[left].value
			left--
		} else {
			values[i] = zones[right].value
			right++
		}
	}
	slices.Sort(values[:n])
	return (values[(n-1)/2] + values[n/2]) / 2
}

func (s *Strategy) observeZones(b *model.Book, now time.Time) []model.Signal {
	best := b.Bids[0].Ticks
	if s.zoneWidth == 0 {
		s.zoneWidth = max(1, int64(math.Round(float64(best)*zoneFraction)))
	}
	var zones []bidZone
	for _, level := range b.Bids {
		low := level.Ticks - level.Ticks%s.zoneWidth
		if len(zones) == 0 || zones[len(zones)-1].low != low {
			zones = append(zones, bidZone{low: low})
		}
		z := &zones[len(zones)-1]
		z.qty += level.Qty
		z.value += level.Qty * s.instrument.Price(level.Ticks)
	}
	// The deepest band may extend beyond the subscribed book.
	if last := len(zones) - 1; b.Bids[len(b.Bids)-1].Ticks > zones[last].low {
		zones = zones[:last]
	}
	for _, ep := range s.alertEpisodes {
		ep.seen = false
	}
	var signals []model.Signal
	for i, z := range zones {
		high := z.low + s.zoneWidth - 1
		distance := float64(max(0, best-high)) / float64(best)
		if distance > searchDistance {
			break
		}
		median := zoneMedian(zones, i)
		if median <= 0 {
			continue
		}
		k := priceLevel{model.Bid, z.low}
		ep := s.alertEpisodes[k]
		qualifies := z.value >= s.cfg.MinNotional && z.value >= median*zoneMultiple
		if ep == nil {
			if !qualifies {
				continue
			}
			ep = &episode{initialQty: z.qty, firstReceipt: b.Received, signal: model.Signal{Since: b.At}}
			s.alertEpisodes[k] = ep
		}
		retained := z.qty / ep.initialQty
		if retained < retainedFraction || ep.confirmed.IsZero() && !qualifies {
			continue
		}
		ep.seen = true
		elapsed := min(b.At.Sub(ep.signal.Since), b.Received.Sub(ep.firstReceipt))
		ep.signal = model.Signal{
			Key: s.instrument.Key, Side: model.Bid, Ticks: z.low,
			Price: s.instrument.Price(z.low), ZoneHigh: s.instrument.Price(high),
			Qty: z.qty, Notional: z.value, Multiple: z.value / median,
			Distance: distance, Retained: retained, Observed: elapsed,
			Since: ep.signal.Since, TradeQty: ep.signal.TradeQty,
			HeldAfterSales: !ep.lastSell.IsZero() && b.At.After(ep.lastSell),
		}
		if ep.confirmed.IsZero() && elapsed >= zoneAge {
			ep.confirmed = b.At
		}
		if !ep.confirmed.IsZero() && distance <= approachDistance && !ep.delivered && !now.Before(s.sent[k]) {
			signals = append(signals, ep.signal)
		}
	}
	for k, ep := range s.alertEpisodes {
		if !ep.seen {
			delete(s.alertEpisodes, k)
		}
	}
	return signals
}
