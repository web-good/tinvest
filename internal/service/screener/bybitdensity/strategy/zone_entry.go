package strategy

import (
	"math"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	ZoneOrderRetention = .40
	ZoneStopLoss       = .001
	ZoneTakeProfit     = .008
)

// ZoneOrder describes a bid zone observed from above.
type ZoneOrder struct {
	LowTicks  int64
	HighTicks int64
	Notional  float64
	Quantity  float64
}

// ZonePolicy owns the signal and price rules shared by replay and an order adapter.
type ZonePolicy struct {
	instrument  model.Instrument
	width       int64
	minQuantity float64
	filter      ZoneFilter
}

func NewZonePolicy(instrument model.Instrument, widthUSDT, minQuantity float64) ZonePolicy {
	return ZonePolicy{instrument: instrument, width: max(1, instrument.Ticks(widthUSDT)), minQuantity: minQuantity}
}

// NewNeighborZonePolicy selects zones using a notional floor and their two neighbors.
func NewNeighborZonePolicy(instrument model.Instrument, widthUSDT float64, filter ZoneFilter) ZonePolicy {
	return ZonePolicy{instrument: instrument, width: max(1, instrument.Ticks(widthUSDT)), filter: filter}
}

// Entry selects the nearest qualifying zone below the best bid in a sorted book.
func (p ZonePolicy) Entry(book *model.Book) (ZoneOrder, bool) {
	var order ZoneOrder
	p.visitEntries(book, 0, func(candidate ZoneOrder) bool {
		order = candidate
		return false
	})
	return order, order.LowTicks != 0
}

func (p ZonePolicy) visitEntries(book *model.Book, maxDistance float64, visit func(ZoneOrder) bool) {
	if len(book.Bids) == 0 || len(book.Asks) == 0 || (!p.filter.Enabled() && !(p.minQuantity > 0)) {
		return
	}
	best := book.Bids[0].Ticks
	var buffer [200]bidZone
	zones := completeBidZones(buffer[:0], book.Bids, p.width, p.instrument)
	for i, zone := range zones {
		high := zone.low + p.width - 1
		if high >= best {
			continue
		}
		if maxDistance > 0 && zoneDistance(best, high) > maxDistance {
			break
		}
		qualifies := zone.qty >= p.minQuantity
		if p.filter.Enabled() {
			_, qualifies = p.filter.qualifies(zones, i)
		}
		if qualifies && zoneStopTicks(zone.low) > 0 &&
			!visit(ZoneOrder{LowTicks: zone.low, HighTicks: high, Notional: zone.value, Quantity: zone.qty}) {
			break
		}
	}
}

func zoneDistance(best, high int64) float64 {
	return float64(max(0, best-high)) / float64(best)
}

// Keep reports whether the zone still has enough visible support.
func (p ZonePolicy) Keep(book *model.Book, order ZoneOrder) bool {
	if len(book.Bids) == 0 || book.Bids[len(book.Bids)-1].Ticks > order.LowTicks {
		return false
	}
	return bidZoneQuantity(book.Bids, order.LowTicks, p.width) >= order.Quantity*ZoneOrderRetention
}

func zoneStopTicks(low int64) int64 {
	return int64(math.Floor(math.Nextafter(float64(low)*(1-ZoneStopLoss), math.Inf(1))))
}

// ExitTicks sets the stop below the zone and the take above the fill price.
func (p ZonePolicy) ExitTicks(order ZoneOrder, entry int64) (stop, take int64) {
	return zoneStopTicks(order.LowTicks),
		int64(math.Ceil(math.Nextafter(float64(entry)*(1+ZoneTakeProfit), math.Inf(-1))))
}
