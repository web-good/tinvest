package strategy

import (
	"time"

	"tinvest/internal/service/screener/bounce"
	"tinvest/internal/service/screener/bybitdensity/model"
)

func (s *Strategy) observeBounceBook(b *model.Book) {
	s.bounce.Book(b.At, func(sig model.Signal) bool {
		qty := quantityAt(b.Bids, model.Bid, sig.Ticks)
		return bounce.Retains(sig.Qty, qty)
	})
	if s.position != nil || s.outstanding != nil {
		return
	}
	for _, ep := range s.episodes {
		if !ep.confirmed.IsZero() {
			s.bounce.Track(ep.signal.Ticks, ep.signal)
		}
	}
}

func (s *Strategy) prepareBounce(now time.Time) {
	sig, _, ok := s.bounce.Ready(now)
	if !ok {
		return
	}
	s.bounce.Consume(now)
	ask := s.book.Asks[0].Ticks
	stop, target := s.entryPrices(model.Bid, sig.Ticks, ask)
	if ask <= sig.Ticks || !s.validPrice(ask) || !s.validPrice(stop) || !s.validPrice(target) {
		return
	}
	s.pending = &setup{side: model.Bid, origin: sig.Ticks, watched: sig.Ticks, baseQty: sig.Qty, limit: ask}
	if sig.HasAverageVolume {
		s.pending.averageVolume = &sig.AverageVolume
	}
}
