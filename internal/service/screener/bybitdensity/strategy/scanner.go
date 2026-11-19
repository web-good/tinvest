package strategy

import (
	"math"
	"slices"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

type episode struct {
	signal                  model.Signal
	firstReceipt, confirmed time.Time
	seen, delivered         bool
	// Keep the last eligible observation before the next M1 interval begins.
	lastClose, lastBook time.Time
	atClose             model.Signal
}

func (s *Strategy) findCandidates(b *model.Book) {
	s.candidates = s.candidates[:0]
	for _, side := range []model.Side{model.Bid, model.Ask} {
		levels := levelsOn(b, side)
		for i, l := range levels {
			price := s.instrument.Price(l.Ticks)
			value := price * l.Qty
			distance := float64(absTicks(l.Ticks-levels[0].Ticks)) / float64(levels[0].Ticks)
			if distance > s.cfg.MaxDistance {
				break
			}
			if value < s.cfg.MinNotional || len(levels) < 6 {
				continue
			}
			multiple := value / s.medianNeighbors(levels, i)
			if multiple < s.cfg.MinMultiple {
				continue
			}
			s.candidates = append(s.candidates, model.Signal{Key: s.instrument.Key, Side: side, Ticks: l.Ticks, Price: price, Qty: l.Qty, Notional: value, Distance: distance, Multiple: multiple})
		}
	}
}

func (s *Strategy) observe(b *model.Book, now time.Time) []model.Signal {
	for _, ep := range s.episodes {
		ep.seen = false
	}
	var signals []model.Signal
	boundary := b.At.Truncate(time.Minute)
	for _, sig := range s.candidates {
		k := priceLevel{sig.Side, sig.Ticks}
		ep := s.episodes[k]
		if ep == nil {
			ep = &episode{firstReceipt: b.Received}
			s.episodes[k] = ep
			sig.Since = b.At
		} else {
			if ep.lastBook.Before(boundary) {
				ep.lastClose, ep.atClose = boundary, ep.signal
			}
			sig.Since, sig.TradeQty = ep.signal.Since, ep.signal.TradeQty
		}
		sig.Observed = min(b.At.Sub(sig.Since), b.Received.Sub(ep.firstReceipt))
		sig.TradeCoverage = sig.Observed
		ep.seen, ep.signal, ep.lastBook = true, sig, b.At
		if ep.confirmed.IsZero() && sig.Observed >= s.cfg.Persistence {
			ep.confirmed = b.At
		}
		if !ep.confirmed.IsZero() && !ep.delivered && !now.Before(s.sent[k]) {
			signals = append(signals, sig)
		}
	}
	for k, ep := range s.episodes {
		if !ep.seen {
			delete(s.episodes, k)
		}
	}
	return signals
}

func levelsOn(b *model.Book, side model.Side) []model.Level {
	if side == model.Bid {
		return b.Bids
	}
	return b.Asks
}

func absTicks(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// On a sorted side, the nearest remaining neighbor is adjacent to the selected interval.
func (s *Strategy) medianNeighbors(levels []model.Level, i int) float64 {
	var neighbors [10]float64
	n := min(len(levels)-1, len(neighbors))
	left, right := i-1, i+1
	for j := range n {
		var next model.Level
		if right == len(levels) || (left >= 0 && absTicks(levels[i].Ticks-levels[left].Ticks) <= absTicks(levels[right].Ticks-levels[i].Ticks)) {
			next = levels[left]
			left--
		} else {
			next = levels[right]
			right++
		}
		neighbors[j] = s.instrument.Price(next.Ticks) * next.Qty
	}
	slices.Sort(neighbors[:n])
	return (neighbors[n/2] + neighbors[(n-1)/2]) / 2
}

const tradeWindow = 8192

type tradeIDs struct {
	ids  []string
	seen map[string]struct{}
	next int
}

func (t *tradeIDs) add(id string) bool {
	if id == "" {
		return false
	}
	if _, ok := t.seen[id]; ok {
		return false
	}
	if t.seen == nil {
		t.seen = make(map[string]struct{})
	}
	if len(t.ids) == tradeWindow {
		delete(t.seen, t.ids[t.next])
		t.ids[t.next] = id
		t.next = (t.next + 1) % tradeWindow
	} else {
		t.ids = append(t.ids, id)
	}
	t.seen[id] = struct{}{}
	return true
}

func (s *Strategy) observeTrade(trade model.Trade) {
	if !s.tradeReady || trade.Qty <= 0 || math.IsNaN(trade.Qty) || math.IsInf(trade.Qty, 0) || !s.trades.add(trade.ID) {
		return
	}
	for _, side := range []model.Side{model.Bid, model.Ask} {
		if ep := s.episodes[priceLevel{side, trade.Ticks}]; ep != nil && !trade.At.Before(ep.signal.Since) {
			ep.signal.TradeQty += trade.Qty
		}
	}
}

func (s *Strategy) Current(signal model.Signal, now time.Time) (model.Signal, bool) {
	k := priceLevel{signal.Side, signal.Ticks}
	ep := s.episodes[k]
	if signal.Key != s.instrument.Key || s.book == nil || !s.fresh(s.book, now) || ep == nil || !ep.signal.Since.Equal(signal.Since) || ep.confirmed.IsZero() || ep.delivered || now.Before(s.sent[k]) {
		return model.Signal{}, false
	}
	return ep.signal, true
}

func (s *Strategy) MarkSent(signal model.Signal, now time.Time) {
	k := priceLevel{signal.Side, signal.Ticks}
	s.sent[k] = now.Add(30 * time.Minute)
	if ep := s.episodes[k]; ep != nil && ep.signal.Since.Equal(signal.Since) {
		ep.delivered = true
	}
}
