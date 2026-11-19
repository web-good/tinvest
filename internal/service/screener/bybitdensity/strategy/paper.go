package strategy

import (
	"math"
	"sort"

	"tinvest/internal/service/screener/bybitdensity/model"
)

// paperExecutor matches an intent against displayed, unspent liquidity.
type paperExecutor struct {
	instrument model.Instrument
	spent      map[priceLevel]float64
}

func quantityAt(levels []model.Level, side model.Side, ticks int64) float64 {
	i := sort.Search(len(levels), func(i int) bool {
		if side == model.Bid {
			return levels[i].Ticks <= ticks
		}
		return levels[i].Ticks >= ticks
	})
	if i < len(levels) && levels[i].Ticks == ticks {
		return levels[i].Qty
	}
	return 0
}

func (p *paperExecutor) ObserveBook(b *model.Book) {
	if b.Snapshot {
		clear(p.spent)
		return
	}
	for k, used := range p.spent {
		qty := quantityAt(levelsOn(b, k.side), k.side, k.ticks)
		if qty == 0 {
			delete(p.spent, k)
		} else if qty < used {
			p.spent[k] = qty
		}
	}
}

func (p *paperExecutor) available(side model.Side, l model.Level) float64 {
	return max(0, l.Qty-p.spent[priceLevel{side, l.Ticks}])
}

func meetsMinimum(instrument model.Instrument, qty, price float64) bool {
	return qty >= math.Nextafter(instrument.MinQty, 0) && qty*price >= math.Nextafter(instrument.MinNotional, 0)
}

func quantitySteps(qty, step float64) int64 {
	units := qty / step
	// Decimal lots such as 0.7 - 0.4 can land just below an integral step.
	nearest := math.Round(units)
	if math.Abs(units-nearest) < 1e-9 {
		units = nearest
	}
	return int64(math.Floor(units))
}

func (p *paperExecutor) Submit(order Order, b *model.Book) Execution {
	side := order.Direction
	if order.Entry {
		side = oppositeSide(side)
	}
	result := Execution{OrderID: order.ID}
	for _, l := range levelsOn(b, side) {
		if order.LimitTicks > 0 && ((side == model.Ask && l.Ticks > order.LimitTicks) || (side == model.Bid && l.Ticks < order.LimitTicks)) {
			break
		}
		steps := min(quantitySteps(p.available(side, l), p.instrument.QtyStep), order.Steps-result.Steps)
		if steps <= 0 {
			continue
		}
		qty := float64(steps) * p.instrument.QtyStep
		if order.Entry && !meetsMinimum(p.instrument, qty, p.instrument.Price(l.Ticks)) {
			break
		}
		p.spent[priceLevel{side, l.Ticks}] += qty
		result.Steps += steps
		result.Quote += p.instrument.Price(l.Ticks) * qty
		if result.Steps == order.Steps {
			break
		}
	}
	return result
}
