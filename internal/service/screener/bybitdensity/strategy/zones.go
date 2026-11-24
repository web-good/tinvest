package strategy

import (
	"slices"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	searchDistance   = .003
	approachDistance = .0005
	zoneMultiple     = 3
	zoneAge          = 8 * time.Second
	retainedFraction = .6
)

// ZoneFilter compares a bid zone's notional with the larger of its two neighbors.
type ZoneFilter struct {
	MinNotional      float64
	NeighborMultiple float64
}

// Enabled reports whether to use the notional and neighbor filter instead of the M5 threshold.
func (f ZoneFilter) Enabled() bool {
	return f.MinNotional > 0 && f.NeighborMultiple > 0
}

func (f ZoneFilter) qualifies(zones []bidZone, i int) (multiple float64, ok bool) {
	if i == 0 || i+1 == len(zones) {
		return 0, false
	}
	neighbor := max(zones[i-1].value, zones[i+1].value)
	z := zones[i]
	return z.value / neighbor, z.value >= f.MinNotional && z.value >= f.NeighborMultiple*neighbor
}

type bidZone struct {
	low        int64
	qty, value float64
}

// nextBidZone groups adjacent bid levels in one band of a descending book.
func nextBidZone(bids []model.Level, width int64, instrument model.Instrument) (bidZone, []model.Level) {
	best := bids[0].Ticks
	zone := bidZone{low: best - best%width}
	for i, level := range bids {
		if level.Ticks < zone.low {
			return zone, bids[i:]
		}
		zone.qty += level.Qty
		zone.value += level.Qty * instrument.Price(level.Ticks)
	}
	return zone, nil
}

func bidZoneQuantity(bids []model.Level, low, width int64) float64 {
	var quantity float64
	for _, level := range bids {
		if level.Ticks < low {
			break
		}
		if level.Ticks-low < width {
			quantity += level.Qty
		}
	}
	return quantity
}

func completeBidZones(zones []bidZone, bids []model.Level, width int64, instrument model.Instrument) []bidZone {
	deepest := bids[len(bids)-1].Ticks
	for len(bids) > 0 {
		zone, rest := nextBidZone(bids, width, instrument)
		if deepest > zone.low {
			break
		}
		zones = append(zones, zone)
		bids = rest
	}
	return zones
}

func (s *Strategy) findZoneCandidates(b *model.Book, now time.Time) {
	best := b.Bids[0].Ticks
	var buffer [200]bidZone
	zones := completeBidZones(buffer[:0], b.Bids, s.zoneWidth, s.instrument)
	for i, zone := range zones {
		high := zone.low + s.zoneWidth - 1
		distance := float64(max(0, best-high)) / float64(best)
		if distance > searchDistance {
			break
		}
		var qualifies bool
		if s.cfg.SpotZoneFilter.Enabled() {
			_, qualifies = s.cfg.SpotZoneFilter.qualifies(zones, i)
		} else {
			qualifies = s.spotVolumeAllows(zone.qty, now)
		}
		if qualifies {
			s.candidates = append(s.candidates, model.Signal{
				Key: s.instrument.Key, Side: model.Bid, Ticks: zone.low, Price: s.instrument.Price(zone.low),
				Qty: zone.qty, Notional: zone.value, Distance: distance,
				AverageVolume: s.averageVolume.Quantity, HasAverageVolume: s.hasSpotAverage(now),
			})
		}
	}
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

func (s *Strategy) hasSpotAverage(now time.Time) bool {
	return s.usesSpotZones() && s.averageVolume.Quantity > 0 && now.Before(s.averageVolume.Until)
}

func (s *Strategy) observeZones(b *model.Book, now time.Time) []model.Signal {
	best := b.Bids[0].Ticks
	var buffer [200]bidZone
	zones := completeBidZones(buffer[:0], b.Bids, s.zoneWidth, s.instrument)
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
		median := 0.0
		if !s.cfg.SpotZoneFilter.Enabled() {
			median = zoneMedian(zones, i)
		}
		multiple := 0.0
		if median > 0 {
			multiple = z.value / median
		} else if !s.cfg.SpotZoneFilter.Enabled() && s.instrument.Key.Market != model.Spot {
			continue
		}
		k := priceLevel{model.Bid, z.low}
		ep := s.alertEpisodes[k]
		qualifies := z.value >= s.cfg.MinNotional && (s.instrument.Key.Market == model.Spot || multiple >= zoneMultiple)
		if s.cfg.SpotZoneFilter.Enabled() {
			multiple, qualifies = s.cfg.SpotZoneFilter.qualifies(zones, i)
		} else if s.cfg.SpotVolumeMultiplier > 0 {
			qualifies = s.spotVolumeAllows(z.qty, now)
		}
		if ep == nil {
			if !qualifies {
				continue
			}
			ep = &episode{initialQty: z.qty, firstReceipt: b.Received, signal: model.Signal{Since: b.At}}
			s.alertEpisodes[k] = ep
		}
		retained := z.qty / ep.initialQty
		if retained < retainedFraction || (ep.confirmed.IsZero() || s.usesSpotZones()) && !qualifies {
			continue
		}
		ep.seen = true
		elapsed := min(b.At.Sub(ep.signal.Since), b.Received.Sub(ep.firstReceipt))
		ep.signal = model.Signal{
			Key: s.instrument.Key, Side: model.Bid, Ticks: z.low,
			Price: s.instrument.Price(z.low), ZoneHigh: s.instrument.Price(high),
			Qty: z.qty, Notional: z.value, Multiple: multiple,
			Distance: distance, Retained: retained, Observed: elapsed,
			Since: ep.signal.Since, TradeQty: ep.signal.TradeQty,
			HeldAfterSales: !ep.lastSell.IsZero() && b.At.After(ep.lastSell),
			AverageVolume:  s.averageVolume.Quantity, HasAverageVolume: s.hasSpotAverage(now),
			HasNeighborMultiple: s.cfg.SpotZoneFilter.Enabled(),
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
